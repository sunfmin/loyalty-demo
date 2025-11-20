package services

import (
	"context"
	"fmt"
	"time"

	"github.com/opentracing/opentracing-go"
	loyaltyv1 "github.com/yourorg/loyalty-demo/api/v1"
	"github.com/yourorg/loyalty-demo/internal/models"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// transactionService implements the TransactionService interface
type transactionService struct {
	db *gorm.DB
}

// NewTransactionService creates a new transaction service instance with dependency injection
func NewTransactionService(db *gorm.DB) TransactionService {
	return &transactionService{
		db: db,
	}
}

// EarnPoints implements TransactionService.EarnPoints
func (s *transactionService) EarnPoints(ctx context.Context, req *loyaltyv1.EarnPointsRequest, accountID string) (*loyaltyv1.EarnPointsResponse, error) {
	// Start OpenTracing span
	span, ctx := opentracing.StartSpanFromContext(ctx, "TransactionService.EarnPoints")
	defer span.Finish()

	span.SetTag("account_id", accountID)
	span.SetTag("amount", req.Amount)

	// Validate amount > 0
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount %d: %w", req.Amount, ErrInvalidAmount)
	}

	// Validate reference_id not empty
	if req.ReferenceId == "" {
		return nil, fmt.Errorf("reference_id: %w", ErrMissingRequired)
	}

	// Check idempotency: query existing transaction by reference_id
	var existing models.PointTransaction
	err := s.db.WithContext(ctx).Where("reference_id = ?", req.ReferenceId).First(&existing).Error
	if err == nil {
		// Transaction already exists - return it (idempotent)
		span.SetTag("idempotent", true)

		// Get customer for balance
		var customer models.Customer
		s.db.WithContext(ctx).First(&customer, "id = ?", existing.CustomerID)

		return &loyaltyv1.EarnPointsResponse{
			Transaction: transactionToProto(&existing),
			NewBalance:  customer.CurrentBalance,
		}, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("check idempotency: %w", err)
	}

	// Get customer by account_id with tier
	var customer models.Customer
	err = s.db.WithContext(ctx).Preload("Tier").Where("account_id = ?", accountID).First(&customer).Error
	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("customer %s: %w", accountID, ErrNotEnrolled)
	}
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}

	// Calculate base points: amount / 100 (cents to dollars, truncate decimals)
	basePoints := req.Amount / 100

	// Apply tier multiplier
	pointsEarned := int64(float64(basePoints) * customer.Tier.EarnRateMultiplier)

	span.SetTag("base_points", basePoints)
	span.SetTag("points_earned", pointsEarned)
	span.SetTag("tier_multiplier", customer.Tier.EarnRateMultiplier)

	// Query active campaigns (start_date <= now <= end_date AND is_active = true)
	var campaigns []models.PromotionalCampaign
	now := time.Now()
	err = s.db.WithContext(ctx).
		Where("is_active = ? AND start_date <= ? AND end_date >= ?", true, now, now).
		Order("priority DESC").
		Find(&campaigns).Error
	if err != nil {
		return nil, fmt.Errorf("query campaigns: %w", err)
	}

	// Apply first matching campaign (highest priority)
	var campaignApplied *loyaltyv1.CampaignApplied
	var appliedCampaignID *string
	if len(campaigns) > 0 {
		campaign := campaigns[0]
		bonusPoints := int64(float64(basePoints) * (campaign.PointMultiplier - 1.0))
		if campaign.BonusPoints > 0 {
			bonusPoints = campaign.BonusPoints
		}
		pointsEarned += bonusPoints
		appliedCampaignID = &campaign.ID

		campaignApplied = &loyaltyv1.CampaignApplied{
			Id:          campaign.ID,
			Name:        campaign.Name,
			BonusPoints: bonusPoints,
		}

		span.SetTag("campaign_applied", campaign.Name)
		span.SetTag("bonus_points", bonusPoints)
	}

	// Calculate expiration date (12 months from now)
	expiresAt := now.Add(365 * 24 * time.Hour)

	// Begin database transaction
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	// Create PointTransaction
	transaction := &models.PointTransaction{
		CustomerID:    customer.ID,
		Amount:        pointsEarned,
		Type:          models.TransactionTypeEarn,
		ReferenceID:   &req.ReferenceId,
		ReferenceType: &req.ReferenceType,
		Description:   req.Description,
		CampaignID:    appliedCampaignID,
		ExpiresAt:     &expiresAt,
		CreatedAt:     now,
	}

	if err := tx.Create(transaction).Error; err != nil {
		return nil, fmt.Errorf("create transaction: %w", err)
	}

	// Update customer balance
	customer.CurrentBalance += pointsEarned
	if err := tx.Save(&customer).Error; err != nil {
		return nil, fmt.Errorf("update balance: %w", err)
	}

	// Check if this is referred customer's first EARN transaction (for referral bonus)
	if customer.ReferredBy != nil {
		// Count previous EARN transactions
		var earnCount int64
		tx.Model(&models.PointTransaction{}).
			Where("customer_id = ? AND type = ? AND id != ?", customer.ID, models.TransactionTypeEarn, transaction.ID).
			Count(&earnCount)

		// If this is first EARN transaction, credit referral bonus
		if earnCount == 0 {
			// Find referrer by referral code
			var referrer models.Customer
			err := tx.Where("referral_code = ?", *customer.ReferredBy).First(&referrer).Error
			if err == nil {
				// Credit referral bonus (e.g., 100 points)
				referralBonus := int64(100)
				referralTransaction := &models.PointTransaction{
					CustomerID:  referrer.ID,
					Amount:      referralBonus,
					Type:        models.TransactionTypeReferral,
					Description: fmt.Sprintf("Referral bonus for %s", customer.MembershipNumber),
					CreatedAt:   now,
					ExpiresAt:   &expiresAt,
				}
				tx.Create(referralTransaction)

				// Update referrer balance
				referrer.CurrentBalance += referralBonus
				tx.Save(&referrer)

				span.SetTag("referral_bonus_awarded", true)
			}
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	// Return response
	return &loyaltyv1.EarnPointsResponse{
		Transaction:     transactionToProto(transaction),
		NewBalance:      customer.CurrentBalance,
		CampaignApplied: campaignApplied,
	}, nil
}

// ListTransactions implements TransactionService.ListTransactions
func (s *transactionService) ListTransactions(ctx context.Context, accountID string, req *loyaltyv1.ListTransactionsRequest) (*loyaltyv1.ListTransactionsResponse, error) {
	// Start OpenTracing span
	span, ctx := opentracing.StartSpanFromContext(ctx, "TransactionService.ListTransactions")
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
	query := s.db.WithContext(ctx).Model(&models.PointTransaction{}).Where("customer_id = ?", customer.ID)

	// Apply filters
	if req.Type != loyaltyv1.TransactionType_TRANSACTION_TYPE_UNSPECIFIED {
		typeStr := transactionTypeToModel(req.Type)
		query = query.Where("type = ?", typeStr)
	}
	if req.StartDate != nil {
		query = query.Where("created_at >= ?", req.StartDate.AsTime())
	}
	if req.EndDate != nil {
		query = query.Where("created_at <= ?", req.EndDate.AsTime())
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count transactions: %w", err)
	}

	// Apply pagination
	limit := int(req.Limit)
	if limit == 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	offset := int(req.Offset)

	// Order by created_at DESC and execute
	var transactions []models.PointTransaction
	err = query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&transactions).Error
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}

	// Convert to protobuf
	protoTransactions := make([]*loyaltyv1.PointTransaction, len(transactions))
	for i, tx := range transactions {
		protoTransactions[i] = transactionToProto(&tx)
	}

	return &loyaltyv1.ListTransactionsResponse{
		Transactions: protoTransactions,
		Total:        int32(total),
		Limit:        int32(limit),
		Offset:       int32(offset),
	}, nil
}

// ManualAdjustment implements TransactionService.ManualAdjustment
func (s *transactionService) ManualAdjustment(ctx context.Context, customerID string, amount int64, reason string, adminUserID string) (*loyaltyv1.AdjustPointsResponse, error) {
	// Start OpenTracing span
	span, ctx := opentracing.StartSpanFromContext(ctx, "TransactionService.ManualAdjustment")
	defer span.Finish()

	span.SetTag("customer_id", customerID)
	span.SetTag("amount", amount)
	span.SetTag("admin_user_id", adminUserID)

	// Validate amount != 0
	if amount == 0 {
		return nil, fmt.Errorf("amount: %w", ErrInvalidAmount)
	}

	// Validate reason not empty
	if reason == "" {
		return nil, fmt.Errorf("reason: %w", ErrMissingRequired)
	}

	// Get customer
	var customer models.Customer
	err := s.db.WithContext(ctx).First(&customer, "id = ?", customerID).Error
	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("customer %s: %w", customerID, ErrNotEnrolled)
	}
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}

	// If negative adjustment, check balance sufficient
	if amount < 0 && customer.CurrentBalance < -amount {
		return nil, fmt.Errorf("balance %d insufficient for adjustment %d: %w", customer.CurrentBalance, amount, ErrInsufficientBalance)
	}

	// Begin database transaction
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	// Create adjustment transaction
	transaction := &models.PointTransaction{
		CustomerID:  customer.ID,
		Amount:      amount,
		Type:        models.TransactionTypeAdjustment,
		Description: fmt.Sprintf("Manual adjustment by admin: %s", reason),
		AdminUserID: &adminUserID,
		AdminNote:   &reason,
		CreatedAt:   time.Now(),
	}

	if err := tx.Create(transaction).Error; err != nil {
		return nil, fmt.Errorf("create adjustment: %w", err)
	}

	// Update customer balance
	customer.CurrentBalance += amount
	if err := tx.Save(&customer).Error; err != nil {
		return nil, fmt.Errorf("update balance: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("commit adjustment: %w", err)
	}

	return &loyaltyv1.AdjustPointsResponse{
		Transaction: transactionToProto(transaction),
		NewBalance:  customer.CurrentBalance,
	}, nil
}

// Helper functions

func transactionToProto(t *models.PointTransaction) *loyaltyv1.PointTransaction {
	transaction := &loyaltyv1.PointTransaction{
		Id:          t.ID,
		CustomerId:  t.CustomerID,
		Amount:      t.Amount,
		Type:        transactionTypeToProto(t.Type),
		Description: t.Description,
		CreatedAt:   timestamppb.New(t.CreatedAt),
	}

	if t.ReferenceID != nil {
		transaction.ReferenceId = *t.ReferenceID
	}
	if t.ReferenceType != nil {
		transaction.ReferenceType = *t.ReferenceType
	}
	if t.CampaignID != nil {
		transaction.CampaignId = *t.CampaignID
	}
	if t.RedemptionID != nil {
		transaction.RedemptionId = *t.RedemptionID
	}
	if t.ExpiredAt != nil {
		transaction.ExpiredAt = timestamppb.New(*t.ExpiredAt)
	}
	if t.ExpiresAt != nil {
		transaction.ExpiresAt = timestamppb.New(*t.ExpiresAt)
	}
	if t.AdminUserID != nil {
		transaction.AdminUserId = *t.AdminUserID
	}
	if t.AdminNote != nil {
		transaction.AdminNote = *t.AdminNote
	}

	return transaction
}

func transactionTypeToProto(t models.TransactionType) loyaltyv1.TransactionType {
	switch t {
	case models.TransactionTypeEarn:
		return loyaltyv1.TransactionType_TRANSACTION_TYPE_EARN
	case models.TransactionTypeRedemption:
		return loyaltyv1.TransactionType_TRANSACTION_TYPE_REDEMPTION
	case models.TransactionTypeAdjustment:
		return loyaltyv1.TransactionType_TRANSACTION_TYPE_ADJUSTMENT
	case models.TransactionTypeReferral:
		return loyaltyv1.TransactionType_TRANSACTION_TYPE_REFERRAL
	case models.TransactionTypeExpiration:
		return loyaltyv1.TransactionType_TRANSACTION_TYPE_EXPIRATION
	default:
		return loyaltyv1.TransactionType_TRANSACTION_TYPE_UNSPECIFIED
	}
}

func transactionTypeToModel(t loyaltyv1.TransactionType) models.TransactionType {
	switch t {
	case loyaltyv1.TransactionType_TRANSACTION_TYPE_EARN:
		return models.TransactionTypeEarn
	case loyaltyv1.TransactionType_TRANSACTION_TYPE_REDEMPTION:
		return models.TransactionTypeRedemption
	case loyaltyv1.TransactionType_TRANSACTION_TYPE_ADJUSTMENT:
		return models.TransactionTypeAdjustment
	case loyaltyv1.TransactionType_TRANSACTION_TYPE_REFERRAL:
		return models.TransactionTypeReferral
	case loyaltyv1.TransactionType_TRANSACTION_TYPE_EXPIRATION:
		return models.TransactionTypeExpiration
	default:
		return models.TransactionTypeEarn
	}
}

