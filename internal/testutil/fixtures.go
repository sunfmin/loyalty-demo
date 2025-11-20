package testutil

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/yourorg/loyalty-demo/internal/models"
	"gorm.io/gorm"
)

// CreateTestTier creates a membership tier for testing with optional overrides
// Default: Base tier (Level 0, 0 points, 1.0x multiplier)
func CreateTestTier(db *gorm.DB, overrides map[string]interface{}) *models.MembershipTier {
	tier := &models.MembershipTier{
		Name:                "Base",
		Level:               0,
		QualificationPoints: 0,
		EvaluationDays:      365,
		EarnRateMultiplier:  1.0,
		Description:         "Base tier for testing",
	}

	// Apply overrides
	if name, ok := overrides["name"].(string); ok {
		tier.Name = name
	}
	if level, ok := overrides["level"].(int); ok {
		tier.Level = level
	}
	if points, ok := overrides["qualification_points"].(int64); ok {
		tier.QualificationPoints = points
	}
	if multiplier, ok := overrides["earn_rate_multiplier"].(float64); ok {
		tier.EarnRateMultiplier = multiplier
	}

	if err := db.Create(tier).Error; err != nil {
		panic(fmt.Sprintf("Failed to create test tier: %v", err))
	}

	return tier
}

// CreateTestCustomer creates a customer for testing with optional overrides
// Default: New customer with unique account_id, membership_number, referral_code
func CreateTestCustomer(db *gorm.DB, overrides map[string]interface{}) *models.Customer {
	// Generate unique values
	timestamp := time.Now().UnixNano()
	randomSuffix := rand.Intn(9999)

	customer := &models.Customer{
		AccountID:        fmt.Sprintf("test-account-%d", timestamp),
		MembershipNumber: fmt.Sprintf("LM-%d-%04d", timestamp, randomSuffix),
		ReferralCode:     fmt.Sprintf("REF%d%04d", timestamp%1000000, randomSuffix),
		EnrolledAt:       time.Now(),
		CurrentBalance:   0,
	}

	// Apply overrides
	if accountID, ok := overrides["account_id"].(string); ok {
		customer.AccountID = accountID
	}
	if membershipNumber, ok := overrides["membership_number"].(string); ok {
		customer.MembershipNumber = membershipNumber
	}
	if referralCode, ok := overrides["referral_code"].(string); ok {
		customer.ReferralCode = referralCode
	}
	if referredBy, ok := overrides["referred_by"].(*string); ok {
		customer.ReferredBy = referredBy
	}
	if balance, ok := overrides["current_balance"].(int64); ok {
		customer.CurrentBalance = balance
	}
	if tierID, ok := overrides["tier_id"].(*string); ok {
		customer.TierID = tierID
	}

	if err := db.Create(customer).Error; err != nil {
		panic(fmt.Sprintf("Failed to create test customer: %v", err))
	}

	return customer
}

// CreateTestReward creates a reward for testing with optional overrides
// Default: Active discount reward costing 100 points
func CreateTestReward(db *gorm.DB, overrides map[string]interface{}) *models.Reward {
	reward := &models.Reward{
		Name:        "Test Reward",
		Description: "Test reward description",
		Type:        models.RewardTypeDiscount,
		PointCost:   100,
		IsActive:    true,
		Metadata:    `{"discount_amount": 500}`,
	}

	// Apply overrides
	if name, ok := overrides["name"].(string); ok {
		reward.Name = name
	}
	if rewardType, ok := overrides["type"].(models.RewardType); ok {
		reward.Type = rewardType
	}
	if pointCost, ok := overrides["point_cost"].(int64); ok {
		reward.PointCost = pointCost
	}
	if isActive, ok := overrides["is_active"].(bool); ok {
		reward.IsActive = isActive
	}
	if metadata, ok := overrides["metadata"].(string); ok {
		reward.Metadata = metadata
	}

	if err := db.Create(reward).Error; err != nil {
		panic(fmt.Sprintf("Failed to create test reward: %v", err))
	}

	return reward
}

// CreateTestCampaign creates a promotional campaign for testing with optional overrides
// Default: Active campaign with 2.0x multiplier, running for next 30 days
func CreateTestCampaign(db *gorm.DB, overrides map[string]interface{}) *models.PromotionalCampaign {
	now := time.Now()
	campaign := &models.PromotionalCampaign{
		Name:            "Test Campaign",
		Description:     "Test campaign description",
		StartDate:       now,
		EndDate:         now.Add(30 * 24 * time.Hour),
		PointMultiplier: 2.0,
		BonusPoints:     0,
		IsActive:        true,
		Conditions:      `{}`,
		Priority:        0,
	}

	// Apply overrides
	if name, ok := overrides["name"].(string); ok {
		campaign.Name = name
	}
	if startDate, ok := overrides["start_date"].(time.Time); ok {
		campaign.StartDate = startDate
	}
	if endDate, ok := overrides["end_date"].(time.Time); ok {
		campaign.EndDate = endDate
	}
	if multiplier, ok := overrides["point_multiplier"].(float64); ok {
		campaign.PointMultiplier = multiplier
	}
	if bonus, ok := overrides["bonus_points"].(int64); ok {
		campaign.BonusPoints = bonus
	}
	if isActive, ok := overrides["is_active"].(bool); ok {
		campaign.IsActive = isActive
	}
	if priority, ok := overrides["priority"].(int); ok {
		campaign.Priority = priority
	}

	if err := db.Create(campaign).Error; err != nil {
		panic(fmt.Sprintf("Failed to create test campaign: %v", err))
	}

	return campaign
}

// CreateTestTransaction creates a point transaction for testing with optional overrides
// Default: EARN transaction with 100 points
func CreateTestTransaction(db *gorm.DB, customerID string, overrides map[string]interface{}) *models.PointTransaction {
	transaction := &models.PointTransaction{
		CustomerID:  customerID,
		Amount:      100,
		Type:        models.TransactionTypeEarn,
		Description: "Test transaction",
	}

	// Apply overrides
	if amount, ok := overrides["amount"].(int64); ok {
		transaction.Amount = amount
	}
	if txType, ok := overrides["type"].(models.TransactionType); ok {
		transaction.Type = txType
	}
	if referenceID, ok := overrides["reference_id"].(*string); ok {
		transaction.ReferenceID = referenceID
	}
	if description, ok := overrides["description"].(string); ok {
		transaction.Description = description
	}
	if campaignID, ok := overrides["campaign_id"].(*string); ok {
		transaction.CampaignID = campaignID
	}

	if err := db.Create(transaction).Error; err != nil {
		panic(fmt.Sprintf("Failed to create test transaction: %v", err))
	}

	return transaction
}

