package services

import (
	"context"

	loyaltyv1 "github.com/yourorg/loyalty-demo/api/v1"
)

// RewardService defines the interface for reward and redemption operations
// All methods accept context.Context as first parameter per constitutional requirement
type RewardService interface {
	// ListRewards returns available rewards with optional filtering
	// No authentication required (public catalog)
	ListRewards(ctx context.Context, req *loyaltyv1.ListRewardsRequest) (*loyaltyv1.ListRewardsResponse, error)

	// RedeemReward redeems points for a reward
	// Returns error if insufficient balance, reward not found, or reward inactive
	// Creates redemption record and deducts points atomically
	RedeemReward(ctx context.Context, req *loyaltyv1.RedeemRewardRequest, accountID string) (*loyaltyv1.RedeemRewardResponse, error)

	// ListRedemptions retrieves redemption history for a customer with filtering
	// Returns error if customer not enrolled
	ListRedemptions(ctx context.Context, accountID string, req *loyaltyv1.ListRedemptionsRequest) (*loyaltyv1.ListRedemptionsResponse, error)

	// ReverseRedemption reverses a redemption and restores points (admin operation)
	// Returns error if redemption not found or already reversed
	ReverseRedemption(ctx context.Context, redemptionID string, reason string, adminUserID string) (*loyaltyv1.ReverseRedemptionResponse, error)
}

