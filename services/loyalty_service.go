package services

import (
	"context"

	loyaltyv1 "github.com/yourorg/loyalty-demo/api/v1"
)

// LoyaltyService defines the interface for loyalty program operations
// All methods accept context.Context as first parameter per constitutional requirement
type LoyaltyService interface {
	// EnrollCustomer enrolls a new customer in the loyalty program
	// Returns error if customer already enrolled or referral code invalid
	EnrollCustomer(ctx context.Context, req *loyaltyv1.EnrollCustomerRequest, accountID string) (*loyaltyv1.Customer, error)

	// GetCustomerStatus retrieves a customer's current loyalty status
	// Returns error if customer not enrolled
	GetCustomerStatus(ctx context.Context, accountID string) (*loyaltyv1.GetCustomerResponse, error)

	// ListTiers returns all available membership tiers
	ListTiers(ctx context.Context) (*loyaltyv1.ListTiersResponse, error)
	
	// EvaluateCustomerTier evaluates and updates customer's tier based on activity
	// Called after point earning transactions to check for tier upgrades
	EvaluateCustomerTier(ctx context.Context, customerID string) (*loyaltyv1.Tier, error)
}

