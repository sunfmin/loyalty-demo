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
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// loyaltyService implements the LoyaltyService interface
type loyaltyService struct {
	db *gorm.DB
}

// NewLoyaltyService creates a new loyalty service instance with dependency injection
func NewLoyaltyService(db *gorm.DB) LoyaltyService {
	return &loyaltyService{
		db: db,
	}
}

// EnrollCustomer implements LoyaltyService.EnrollCustomer
func (s *loyaltyService) EnrollCustomer(ctx context.Context, req *loyaltyv1.EnrollCustomerRequest, accountID string) (*loyaltyv1.Customer, error) {
	// Start OpenTracing span
	span, ctx := opentracing.StartSpanFromContext(ctx, "LoyaltyService.EnrollCustomer")
	defer span.Finish()
	
	span.SetTag("account_id", accountID)

	// Check if customer already enrolled
	var existing models.Customer
	err := s.db.WithContext(ctx).Where("account_id = ?", accountID).First(&existing).Error
	if err == nil {
		// Customer exists
		return nil, fmt.Errorf("customer %s: %w", accountID, ErrAlreadyEnrolled)
	}
	if err != gorm.ErrRecordNotFound {
		// Database error
		return nil, fmt.Errorf("check enrollment: %w", err)
	}

	// Validate referral code if provided
	var referredBy *string
	if req.ReferralCode != "" {
		var referrer models.Customer
		err := s.db.WithContext(ctx).Where("referral_code = ?", req.ReferralCode).First(&referrer).Error
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("referral code %s: %w", req.ReferralCode, ErrInvalidReferralCode)
		}
		if err != nil {
			return nil, fmt.Errorf("validate referral: %w", err)
		}
		referredBy = &req.ReferralCode
	}

	// Generate unique membership number and referral code
	membershipNumber := generateMembershipNumber()
	referralCode := generateReferralCode()

	// Get base tier (level 0)
	var baseTier models.MembershipTier
	err = s.db.WithContext(ctx).Where("level = ?", 0).First(&baseTier).Error
	if err != nil {
		return nil, fmt.Errorf("get base tier: %w", err)
	}

	// Create customer
	customer := &models.Customer{
		AccountID:        accountID,
		MembershipNumber: membershipNumber,
		ReferralCode:     referralCode,
		ReferredBy:       referredBy,
		EnrolledAt:       time.Now(),
		CurrentBalance:   0,
		TierID:           &baseTier.ID,
	}

	if err := s.db.WithContext(ctx).Create(customer).Error; err != nil {
		return nil, fmt.Errorf("create customer: %w", err)
	}

	// Preload tier for response
	if err := s.db.WithContext(ctx).Preload("Tier").First(customer, "id = ?", customer.ID).Error; err != nil {
		return nil, fmt.Errorf("reload customer: %w", err)
	}

	// Convert to protobuf
	return customerToProto(customer), nil
}

// GetCustomerStatus implements LoyaltyService.GetCustomerStatus
func (s *loyaltyService) GetCustomerStatus(ctx context.Context, accountID string) (*loyaltyv1.GetCustomerResponse, error) {
	// Start OpenTracing span
	span, ctx := opentracing.StartSpanFromContext(ctx, "LoyaltyService.GetCustomerStatus")
	defer span.Finish()
	
	span.SetTag("account_id", accountID)

	// Get customer with tier
	var customer models.Customer
	err := s.db.WithContext(ctx).Preload("Tier").Where("account_id = ?", accountID).First(&customer).Error
	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("customer %s: %w", accountID, ErrNotEnrolled)
	}
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}

	// Calculate points to next tier
	pointsToNextTier := int64(0)
	if customer.Tier != nil {
		// Get next tier
		var nextTier models.MembershipTier
		err := s.db.WithContext(ctx).
			Where("level > ?", customer.Tier.Level).
			Order("level ASC").
			First(&nextTier).Error
		
		if err == nil {
			// Calculate points earned in evaluation period
			evaluationStart := time.Now().AddDate(0, 0, -customer.Tier.EvaluationDays)
			var pointsEarned int64
			s.db.WithContext(ctx).
				Model(&models.PointTransaction{}).
				Where("customer_id = ? AND type = ? AND created_at >= ?", 
					customer.ID, models.TransactionTypeEarn, evaluationStart).
				Select("COALESCE(SUM(amount), 0)").
				Scan(&pointsEarned)
			
			// Calculate shortfall to next tier
			if pointsEarned < nextTier.QualificationPoints {
				pointsToNextTier = nextTier.QualificationPoints - pointsEarned
			}
		}
	}

	// Check for points expiring soon (within 30 days)
	var expiringPoints int64
	expiringDate := time.Now().Add(30 * 24 * time.Hour)
	s.db.WithContext(ctx).
		Model(&models.PointTransaction{}).
		Where("customer_id = ? AND expires_at <= ? AND expires_at > ? AND expired_at IS NULL", 
			customer.ID, expiringDate, time.Now()).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&expiringPoints)

	response := &loyaltyv1.GetCustomerResponse{
		Customer:         customerToProto(&customer),
		PointsToNextTier: pointsToNextTier,
	}

	if expiringPoints > 0 {
		response.PointsExpiringSoon = &loyaltyv1.PointsExpiringSoon{
			Amount:         expiringPoints,
			ExpirationDate: timestamppb.New(expiringDate),
		}
	}

	return response, nil
}

// ListTiers implements LoyaltyService.ListTiers
func (s *loyaltyService) ListTiers(ctx context.Context) (*loyaltyv1.ListTiersResponse, error) {
	// Start OpenTracing span
	span, ctx := opentracing.StartSpanFromContext(ctx, "LoyaltyService.ListTiers")
	defer span.Finish()

	var tiers []models.MembershipTier
	err := s.db.WithContext(ctx).Order("level ASC").Find(&tiers).Error
	if err != nil {
		return nil, fmt.Errorf("list tiers: %w", err)
	}

	protoTiers := make([]*loyaltyv1.Tier, len(tiers))
	for i, tier := range tiers {
		protoTiers[i] = tierToProto(&tier)
	}

	return &loyaltyv1.ListTiersResponse{
		Tiers: protoTiers,
	}, nil
}

// EvaluateCustomerTier implements LoyaltyService.EvaluateCustomerTier
func (s *loyaltyService) EvaluateCustomerTier(ctx context.Context, customerID string) (*loyaltyv1.Tier, error) {
	// Start OpenTracing span
	span, ctx := opentracing.StartSpanFromContext(ctx, "LoyaltyService.EvaluateCustomerTier")
	defer span.Finish()
	
	span.SetTag("customer_id", customerID)

	// Get customer with current tier
	var customer models.Customer
	err := s.db.WithContext(ctx).Preload("Tier").First(&customer, "id = ?", customerID).Error
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}

	if customer.Tier == nil {
		return nil, fmt.Errorf("customer has no tier")
	}

	// Calculate evaluation period
	evaluationStart := time.Now().AddDate(0, 0, -customer.Tier.EvaluationDays)

	// Calculate points earned in evaluation period
	var pointsEarned int64
	err = s.db.WithContext(ctx).
		Model(&models.PointTransaction{}).
		Where("customer_id = ? AND type = ? AND created_at >= ?", 
			customerID, models.TransactionTypeEarn, evaluationStart).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&pointsEarned).Error
	if err != nil {
		return nil, fmt.Errorf("calculate points earned: %w", err)
	}

	// Get all tiers ordered by level descending
	var tiers []models.MembershipTier
	err = s.db.WithContext(ctx).Order("level DESC").Find(&tiers).Error
	if err != nil {
		return nil, fmt.Errorf("get tiers: %w", err)
	}

	// Find highest tier customer qualifies for
	var newTier *models.MembershipTier
	for i := range tiers {
		if pointsEarned >= tiers[i].QualificationPoints {
			newTier = &tiers[i]
			break
		}
	}

	// Update tier if different
	if newTier != nil && (customer.TierID == nil || newTier.ID != *customer.TierID) {
		customer.TierID = &newTier.ID
		if err := s.db.WithContext(ctx).Save(&customer).Error; err != nil {
			return nil, fmt.Errorf("update tier: %w", err)
		}
		return tierToProto(newTier), nil
	}

	return tierToProto(customer.Tier), nil
}

// Helper functions

func generateMembershipNumber() string {
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("LM-%d", timestamp)
}

func generateReferralCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("REF%s", hex.EncodeToString(b))[:8]
}

func customerToProto(c *models.Customer) *loyaltyv1.Customer {
	customer := &loyaltyv1.Customer{
		Id:               c.ID,
		AccountId:        c.AccountID,
		MembershipNumber: c.MembershipNumber,
		ReferralCode:     c.ReferralCode,
		ReferredBy:       "",
		EnrolledAt:       timestamppb.New(c.EnrolledAt),
		CurrentBalance:   c.CurrentBalance,
		CreatedAt:        timestamppb.New(c.CreatedAt),
		UpdatedAt:        timestamppb.New(c.UpdatedAt),
	}
	
	if c.ReferredBy != nil {
		customer.ReferredBy = *c.ReferredBy
	}
	
	if c.Tier != nil {
		customer.Tier = tierToProto(c.Tier)
	}
	
	return customer
}

func tierToProto(t *models.MembershipTier) *loyaltyv1.Tier {
	return &loyaltyv1.Tier{
		Id:                  t.ID,
		Name:                t.Name,
		Level:               int32(t.Level),
		QualificationPoints: t.QualificationPoints,
		EvaluationDays:      int32(t.EvaluationDays),
		EarnRateMultiplier:  t.EarnRateMultiplier,
		Description:         t.Description,
	}
}

