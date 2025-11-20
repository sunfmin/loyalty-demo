package services

import (
	"context"

	loyaltyv1 "github.com/yourorg/loyalty-demo/api/v1"
)

// TransactionService defines the interface for point transaction operations
// All methods accept context.Context as first parameter per constitutional requirement
type TransactionService interface {
	// EarnPoints credits points for a qualifying activity (purchase, referral, promotion)
	// Returns error if customer not enrolled, amount invalid, or reference_id missing
	// Idempotent: same reference_id returns existing transaction
	EarnPoints(ctx context.Context, req *loyaltyv1.EarnPointsRequest, accountID string) (*loyaltyv1.EarnPointsResponse, error)

	// ListTransactions retrieves transaction history for a customer with filtering and pagination
	// Returns error if customer not enrolled
	ListTransactions(ctx context.Context, accountID string, req *loyaltyv1.ListTransactionsRequest) (*loyaltyv1.ListTransactionsResponse, error)

	// ManualAdjustment allows administrators to manually adjust customer points with audit trail
	// Returns error if customer not found, amount is zero, or reason missing
	ManualAdjustment(ctx context.Context, customerID string, amount int64, reason string, adminUserID string) (*loyaltyv1.AdjustPointsResponse, error)
}

