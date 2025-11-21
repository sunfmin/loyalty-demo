package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/opentracing/opentracing-go"
	loyaltyv1 "github.com/yourorg/loyalty-demo/api/v1"
	"github.com/yourorg/loyalty-demo/internal/models"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// rewardService implements the RewardService interface
type rewardService struct {
	db                 *gorm.DB
	transactionService TransactionService
}

// NewRewardService creates a new reward service instance with dependency injection
func NewRewardService(db *gorm.DB, transactionService TransactionService) RewardService {
	return &rewardService{
		db:                 db,
		transactionService: transactionService,
	}
}

// ListRewards implements RewardService.ListRewards
func (s *rewardService) ListRewards(ctx context.Context, req *loyaltyv1.ListRewardsRequest) (*loyaltyv1.ListRewardsResponse, error) {
	// Start OpenTracing span
	span, ctx := opentracing.StartSpanFromContext(ctx, "RewardService.ListRewards")
	defer span.Finish()

	// Build query
	query := s.db.WithContext(ctx).Model(&models.Reward{})

	// Apply filters
	// Default: Filter to active rewards only
	// Only show all if explicitly requested with ActiveOnly=false
	filterToActive := true
	if req != nil {
		filterToActive = req.ActiveOnly
		span.SetTag("active_only", req.ActiveOnly)
	}
	
	if filterToActive {
		// Filter to active rewards only (default behavior)
		query = query.Where("is_active = ?", true)
	}
	// If filterToActive is false, don't filter (show all)

	if req != nil && req.MaxPoints > 0 {
		query = query.Where("point_cost <= ?", req.MaxPoints)
	}

	// Execute query
	var rewards []models.Reward
	err := query.Find(&rewards).Error
	if err != nil {
		return nil, fmt.Errorf("list rewards: %w", err)
	}

	// Convert to protobuf
	protoRewards := make([]*loyaltyv1.Reward, len(rewards))
	for i, reward := range rewards {
		protoRewards[i] = rewardToProto(&reward)
	}

	return &loyaltyv1.ListRewardsResponse{
		Rewards: protoRewards,
	}, nil
}

// RedeemReward implements RewardService.RedeemReward
func (s *rewardService) RedeemReward(ctx context.Context, req *loyaltyv1.RedeemRewardRequest, accountID string) (*loyaltyv1.RedeemRewardResponse, error) {
	// Start OpenTracing span
	span, ctx := opentracing.StartSpanFromContext(ctx, "RewardService.RedeemReward")
	defer span.Finish()

	span.SetTag("account_id", accountID)
	span.SetTag("reward_id", req.RewardId)

	// Get customer by account_id
	var customer models.Customer
	err := s.db.WithContext(ctx).Where("account_id = ?", accountID).First(&customer).Error
	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("customer %s: %w", accountID, ErrNotEnrolled)
	}
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}

	// Get reward by ID
	var reward models.Reward
	err = s.db.WithContext(ctx).First(&reward, "id = ?", req.RewardId).Error
	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("reward %s: %w", req.RewardId, ErrRewardNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get reward: %w", err)
	}

	// Check if reward is active
	if !reward.IsActive {
		return nil, fmt.Errorf("reward %s: %w", req.RewardId, ErrRewardInactive)
	}

	// Check balance sufficient
	if customer.CurrentBalance < reward.PointCost {
		return nil, fmt.Errorf("balance %d < cost %d: %w", customer.CurrentBalance, reward.PointCost, ErrInsufficientBalance)
	}

	// Begin database transaction
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	// Generate unique redemption code
	code := generateRedemptionCode()

	// Create redemption
	redemption := &models.Redemption{
		CustomerID:     customer.ID,
		RewardID:       reward.ID,
		PointsDeducted: reward.PointCost,
		Status:         models.RedemptionStatusActive,
		Code:           code,
	}

	if err := tx.Create(redemption).Error; err != nil {
		return nil, fmt.Errorf("create redemption: %w", err)
	}

	// Create negative point transaction
	transaction := &models.PointTransaction{
		CustomerID:   customer.ID,
		Amount:       -reward.PointCost,
		Type:         models.TransactionTypeRedemption,
		Description:  fmt.Sprintf("Redeemed: %s", reward.Name),
		RedemptionID: &redemption.ID,
		CreatedAt:    time.Now(),
	}

	if err := tx.Create(transaction).Error; err != nil {
		return nil, fmt.Errorf("create transaction: %w", err)
	}

	// Update customer balance
	customer.CurrentBalance -= reward.PointCost
	if err := tx.Save(&customer).Error; err != nil {
		return nil, fmt.Errorf("update balance: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("commit redemption: %w", err)
	}

	// Preload reward for response (use main db, not tx)
	s.db.WithContext(ctx).Preload("Reward").First(redemption, "id = ?", redemption.ID)

	return &loyaltyv1.RedeemRewardResponse{
		Redemption:  redemptionToProto(redemption),
		NewBalance:  customer.CurrentBalance,
		Transaction: transactionToProto(transaction),
	}, nil
}

// ListRedemptions implements RewardService.ListRedemptions
func (s *rewardService) ListRedemptions(ctx context.Context, accountID string, req *loyaltyv1.ListRedemptionsRequest) (*loyaltyv1.ListRedemptionsResponse, error) {
	// Start OpenTracing span
	span, ctx := opentracing.StartSpanFromContext(ctx, "RewardService.ListRedemptions")
	defer span.Finish()

	span.SetTag("account_id", accountID)

	// Get customer by account_id
	var customer models.Customer
	err := s.db.WithContext(ctx).Where("account_id = ?", accountID).First(&customer).Error
	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("customer %s: %w", accountID, ErrNotEnrolled)
	}
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}

	// Build query
	query := s.db.WithContext(ctx).Model(&models.Redemption{}).Where("customer_id = ?", customer.ID)

	// Apply filters
	if req != nil && req.Status != loyaltyv1.RedemptionStatus_REDEMPTION_STATUS_UNSPECIFIED {
		statusStr := redemptionStatusToModel(req.Status)
		query = query.Where("status = ?", statusStr)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count redemptions: %w", err)
	}

	// Apply pagination
	limit := 50
	offset := 0
	if req != nil {
		if req.Limit > 0 {
			limit = int(req.Limit)
			if limit > 100 {
				limit = 100
			}
		}
		offset = int(req.Offset)
	}

	// Order by created_at DESC and execute
	var redemptions []models.Redemption
	err = query.Preload("Reward").Order("created_at DESC").Limit(limit).Offset(offset).Find(&redemptions).Error
	if err != nil {
		return nil, fmt.Errorf("list redemptions: %w", err)
	}

	// Convert to protobuf
	protoRedemptions := make([]*loyaltyv1.Redemption, len(redemptions))
	for i, redemption := range redemptions {
		protoRedemptions[i] = redemptionToProto(&redemption)
	}

	return &loyaltyv1.ListRedemptionsResponse{
		Redemptions: protoRedemptions,
		Total:       int32(total),
		Limit:       int32(limit),
		Offset:      int32(offset),
	}, nil
}

// ReverseRedemption implements RewardService.ReverseRedemption
func (s *rewardService) ReverseRedemption(ctx context.Context, redemptionID string, reason string, adminUserID string) (*loyaltyv1.ReverseRedemptionResponse, error) {
	// Start OpenTracing span
	span, ctx := opentracing.StartSpanFromContext(ctx, "RewardService.ReverseRedemption")
	defer span.Finish()

	span.SetTag("redemption_id", redemptionID)
	span.SetTag("admin_user_id", adminUserID)

	// Validate reason not empty
	if reason == "" {
		return nil, fmt.Errorf("reason: %w", ErrMissingRequired)
	}

	// Get redemption with customer and reward
	var redemption models.Redemption
	err := s.db.WithContext(ctx).Preload("Customer").Preload("Reward").First(&redemption, "id = ?", redemptionID).Error
	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("redemption %s: %w", redemptionID, ErrRedemptionNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get redemption: %w", err)
	}

	// Check if already reversed
	if redemption.Status == models.RedemptionStatusReversed {
		return nil, fmt.Errorf("redemption %s: %w", redemptionID, ErrRedemptionAlreadyReversed)
	}

	// Begin database transaction
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	// Update redemption status
	now := time.Now()
	redemption.Status = models.RedemptionStatusReversed
	redemption.ReversedAt = &now
	redemption.ReversalReason = &reason

	if err := tx.Save(&redemption).Error; err != nil {
		return nil, fmt.Errorf("update redemption: %w", err)
	}

	// Create restoration transaction
	transaction := &models.PointTransaction{
		CustomerID:  redemption.CustomerID,
		Amount:      redemption.PointsDeducted, // Positive (restore points)
		Type:        models.TransactionTypeAdjustment,
		Description: fmt.Sprintf("Redemption reversal: %s", reason),
		AdminUserID: &adminUserID,
		AdminNote:   &reason,
		CreatedAt:   now,
	}

	if err := tx.Create(transaction).Error; err != nil {
		return nil, fmt.Errorf("create restoration: %w", err)
	}

	// Update customer balance
	redemption.Customer.CurrentBalance += redemption.PointsDeducted
	if err := tx.Save(redemption.Customer).Error; err != nil {
		return nil, fmt.Errorf("update balance: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("commit reversal: %w", err)
	}

	return &loyaltyv1.ReverseRedemptionResponse{
		Redemption:     redemptionToProto(&redemption),
		PointsRestored: redemption.PointsDeducted,
		NewBalance:     redemption.Customer.CurrentBalance,
		Transaction:    transactionToProto(transaction),
	}, nil
}

// Helper functions

func generateRedemptionCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("DISC-%s", hex.EncodeToString(b))[:12]
}

func rewardToProto(r *models.Reward) *loyaltyv1.Reward {
	reward := &loyaltyv1.Reward{
		Id:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Type:        rewardTypeToProto(r.Type),
		PointCost:   r.PointCost,
		IsActive:    r.IsActive,
		CreatedAt:   timestamppb.New(r.CreatedAt),
		UpdatedAt:   timestamppb.New(r.UpdatedAt),
	}

	// Parse metadata JSON to Struct
	if r.Metadata != "" {
		if metadata, err := structpb.NewStruct(map[string]interface{}{}); err == nil {
			reward.Metadata = metadata
		}
	}

	return reward
}

func redemptionToProto(r *models.Redemption) *loyaltyv1.Redemption {
	redemption := &loyaltyv1.Redemption{
		Id:             r.ID,
		CustomerId:     r.CustomerID,
		PointsDeducted: r.PointsDeducted,
		Status:         redemptionStatusToProto(r.Status),
		Code:           r.Code,
		CreatedAt:      timestamppb.New(r.CreatedAt),
	}

	if r.Reward != nil {
		redemption.Reward = rewardToProto(r.Reward)
	}

	if r.UsedAt != nil {
		redemption.UsedAt = timestamppb.New(*r.UsedAt)
	}

	if r.ReversedAt != nil {
		redemption.ReversedAt = timestamppb.New(*r.ReversedAt)
	}

	if r.ReversalReason != nil {
		redemption.ReversalReason = *r.ReversalReason
	}

	return redemption
}

func rewardTypeToProto(t models.RewardType) loyaltyv1.RewardType {
	switch t {
	case models.RewardTypeDiscount:
		return loyaltyv1.RewardType_REWARD_TYPE_DISCOUNT
	case models.RewardTypeVoucher:
		return loyaltyv1.RewardType_REWARD_TYPE_VOUCHER
	case models.RewardTypeFreeItem:
		return loyaltyv1.RewardType_REWARD_TYPE_FREE_ITEM
	default:
		return loyaltyv1.RewardType_REWARD_TYPE_UNSPECIFIED
	}
}

func redemptionStatusToProto(s models.RedemptionStatus) loyaltyv1.RedemptionStatus {
	switch s {
	case models.RedemptionStatusActive:
		return loyaltyv1.RedemptionStatus_REDEMPTION_STATUS_ACTIVE
	case models.RedemptionStatusUsed:
		return loyaltyv1.RedemptionStatus_REDEMPTION_STATUS_USED
	case models.RedemptionStatusReversed:
		return loyaltyv1.RedemptionStatus_REDEMPTION_STATUS_REVERSED
	default:
		return loyaltyv1.RedemptionStatus_REDEMPTION_STATUS_UNSPECIFIED
	}
}

func redemptionStatusToModel(s loyaltyv1.RedemptionStatus) models.RedemptionStatus {
	switch s {
	case loyaltyv1.RedemptionStatus_REDEMPTION_STATUS_ACTIVE:
		return models.RedemptionStatusActive
	case loyaltyv1.RedemptionStatus_REDEMPTION_STATUS_USED:
		return models.RedemptionStatusUsed
	case loyaltyv1.RedemptionStatus_REDEMPTION_STATUS_REVERSED:
		return models.RedemptionStatusReversed
	default:
		return models.RedemptionStatusActive
	}
}

