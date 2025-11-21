package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	loyaltyv1 "github.com/yourorg/loyalty-demo/api/v1"
	"github.com/yourorg/loyalty-demo/internal/middleware"
	"github.com/yourorg/loyalty-demo/internal/models"
	"github.com/yourorg/loyalty-demo/internal/testutil"
	"github.com/yourorg/loyalty-demo/services"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestEarnPoints(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "point_transactions", "customers", "membership_tiers", "promotional_campaigns")

	// Create base tier
	baseTier := testutil.CreateTestTier(db, map[string]interface{}{
		"name":                 "Base",
		"level":                0,
		"qualification_points": int64(0),
		"earn_rate_multiplier": 1.0,
	})

	// Create Gold tier for multiplier test
	goldTier := testutil.CreateTestTier(db, map[string]interface{}{
		"name":                 "Gold",
		"level":                2,
		"qualification_points": int64(1000),
		"earn_rate_multiplier": 1.5,
	})

	// Create test campaign (double points)
	campaign := testutil.CreateTestCampaign(db, map[string]interface{}{
		"name":             "Double Points Weekend",
		"point_multiplier": 2.0,
		"is_active":        true,
		"start_date":       time.Now().Add(-1 * time.Hour),
		"end_date":         time.Now().Add(24 * time.Hour),
	})

	// Table-driven test cases
	testCases := []struct {
		name             string
		accountID        string
		request          *loyaltyv1.EarnPointsRequest
		setupFixtures    func() string // Returns customer ID
		expectedStatus   int
		expectedError    string
		validateResponse func(t *testing.T, resp *loyaltyv1.EarnPointsResponse, customerID string)
	}{
		{
			name:      "Happy path: Valid purchase earns points (1 point per dollar)",
			accountID: "user-earn-001",
			request: &loyaltyv1.EarnPointsRequest{
				Amount:        5000, // $50.00 in cents
				ReferenceId:   "order-12345",
				ReferenceType: "ORDER",
				Description:   "Purchase at Main Street Store",
			},
			setupFixtures: func() string {
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id": "user-earn-001",
					"tier_id":    &baseTier.ID,
				})
				return customer.ID
			},
			expectedStatus: http.StatusCreated,
			validateResponse: func(t *testing.T, resp *loyaltyv1.EarnPointsResponse, customerID string) {
				if resp.Transaction == nil {
					t.Fatal("Expected transaction in response")
				}

				// Build expected from REQUEST data
				// Note: Campaign is active (2.0x), so $50 -> 50 base points * 2.0 = 100 points
				expected := &loyaltyv1.EarnPointsResponse{
					Transaction: &loyaltyv1.PointTransaction{
						Id:            resp.Transaction.Id,          // Generated
						CustomerId:    customerID,                   // From fixture
						Amount:        100,                          // $50.00 -> 50 base * 2.0x campaign = 100 points
						Type:          loyaltyv1.TransactionType_TRANSACTION_TYPE_EARN,
						ReferenceId:   "order-12345",               // From request
						ReferenceType: "ORDER",                     // From request
						Description:   "Purchase at Main Street Store", // From request
						CampaignId:    resp.Transaction.CampaignId, // Campaign applied
						CreatedAt:     resp.Transaction.CreatedAt,  // Generated
						ExpiresAt:     resp.Transaction.ExpiresAt,  // Generated
					},
					NewBalance: 100, // Initial 0 + 100 earned (with campaign)
					CampaignApplied: resp.CampaignApplied, // Campaign applied (double points)
				}

				// Use protocmp for comparison (MANDATORY)
				if diff := cmp.Diff(expected, resp, protocmp.Transform()); diff != "" {
					t.Errorf("Response mismatch (-want +got):\n%s", diff)
				}

				// Verify expiration set to 12 months from now
				if resp.Transaction.ExpiresAt != nil {
					expiresAt := resp.Transaction.ExpiresAt.AsTime()
					expectedExpiry := time.Now().Add(365 * 24 * time.Hour)
					timeDiff := expiresAt.Sub(expectedExpiry).Abs()
					if timeDiff > 1*time.Minute {
						t.Errorf("Expected expiration ~12 months from now, got %v", expiresAt)
					}
				}
			},
		},
		{
			name:      "Happy path: Points with tier multiplier (Gold 1.5x) and campaign",
			accountID: "user-earn-gold",
			request: &loyaltyv1.EarnPointsRequest{
				Amount:        5000, // $50.00
				ReferenceId:   "order-gold-001",
				ReferenceType: "ORDER",
				Description:   "Gold tier purchase",
			},
			setupFixtures: func() string {
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id": "user-earn-gold",
					"tier_id":    &goldTier.ID,
				})
				return customer.ID
			},
			expectedStatus: http.StatusCreated,
			validateResponse: func(t *testing.T, resp *loyaltyv1.EarnPointsResponse, customerID string) {
				if resp.Transaction == nil {
					t.Fatal("Expected transaction in response")
				}
				
				// Build expected from REQUEST data
				// Base: $50 = 50 points, Tier: 1.5x = 75, Campaign: 2.0x bonus = 50, Total: 125
				expected := &loyaltyv1.EarnPointsResponse{
					Transaction: &loyaltyv1.PointTransaction{
						Id:            resp.Transaction.Id,
						CustomerId:    customerID,
						Amount:        125, // 50 base * 1.5 tier + 50 campaign bonus
						Type:          loyaltyv1.TransactionType_TRANSACTION_TYPE_EARN,
						ReferenceId:   "order-gold-001",
						ReferenceType: "ORDER",
						Description:   "Gold tier purchase",
						CampaignId:    resp.Transaction.CampaignId,
						CreatedAt:     resp.Transaction.CreatedAt,
						ExpiresAt:     resp.Transaction.ExpiresAt,
					},
					NewBalance:      resp.NewBalance, // Calculated field
					CampaignApplied: resp.CampaignApplied, // Campaign info
				}
				
				// Compare using protocmp (MANDATORY)
				if diff := cmp.Diff(expected, resp, protocmp.Transform()); diff != "" {
					t.Errorf("Response mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:      "Happy path: Idempotent requests (same reference_id)",
			accountID: "user-earn-idempotent",
			request: &loyaltyv1.EarnPointsRequest{
				Amount:        3000,
				ReferenceId:   "order-idempotent-123",
				ReferenceType: "ORDER",
				Description:   "Idempotent test",
			},
			setupFixtures: func() string {
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id": "user-earn-idempotent",
					"tier_id":    &baseTier.ID,
				})
				// Create existing transaction with same reference_id
				refID := "order-idempotent-123"
				testutil.CreateTestTransaction(db, customer.ID, map[string]interface{}{
					"amount":         int64(30),
					"type":           models.TransactionTypeEarn,
					"reference_id":   &refID,
					"reference_type": stringPtr("ORDER"),
					"description":    "Idempotent test",
				})
				return customer.ID
			},
			expectedStatus: http.StatusCreated,
			validateResponse: func(t *testing.T, resp *loyaltyv1.EarnPointsResponse, customerID string) {
				if resp.Transaction == nil {
					t.Fatal("Expected transaction in response")
				}
				
				// Build expected - should return existing transaction (30 points, not recalculated)
				expected := &loyaltyv1.EarnPointsResponse{
					Transaction: &loyaltyv1.PointTransaction{
						Id:            resp.Transaction.Id, // Existing transaction ID
						CustomerId:    customerID,
						Amount:        30, // Existing amount (not recalculated from 3000 cents)
						Type:          loyaltyv1.TransactionType_TRANSACTION_TYPE_EARN,
						ReferenceId:   "order-idempotent-123", // Same reference_id
						ReferenceType: resp.Transaction.ReferenceType,
						Description:   "Idempotent test",
						CampaignId:    resp.Transaction.CampaignId,
						CreatedAt:     resp.Transaction.CreatedAt,
						ExpiresAt:     resp.Transaction.ExpiresAt,
					},
					NewBalance:      resp.NewBalance,
					CampaignApplied: resp.CampaignApplied,
				}
				
				// Compare using protocmp (MANDATORY)
				if diff := cmp.Diff(expected, resp, protocmp.Transform()); diff != "" {
					t.Errorf("Response mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:      "Edge case: Negative amount",
			accountID: "user-earn-negative",
			request: &loyaltyv1.EarnPointsRequest{
				Amount:        -100,
				ReferenceId:   "order-negative",
				ReferenceType: "ORDER",
				Description:   "Negative amount test",
			},
			setupFixtures: func() string {
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id": "user-earn-negative",
					"tier_id":    &baseTier.ID,
				})
				return customer.ID
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_AMOUNT",
		},
		{
			name:      "Edge case: Zero amount",
			accountID: "user-earn-zero",
			request: &loyaltyv1.EarnPointsRequest{
				Amount:        0,
				ReferenceId:   "order-zero",
				ReferenceType: "ORDER",
				Description:   "Zero amount test",
			},
			setupFixtures: func() string {
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id": "user-earn-zero",
					"tier_id":    &baseTier.ID,
				})
				return customer.ID
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_AMOUNT",
		},
		{
			name:      "Edge case: Missing reference_id",
			accountID: "user-earn-no-ref",
			request: &loyaltyv1.EarnPointsRequest{
				Amount:        1000,
				ReferenceId:   "",
				ReferenceType: "ORDER",
				Description:   "Missing reference",
			},
			setupFixtures: func() string {
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id": "user-earn-no-ref",
					"tier_id":    &baseTier.ID,
				})
				return customer.ID
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "MISSING_REQUIRED",
		},
		{
			name:      "Edge case: Customer not enrolled",
			accountID: "user-not-enrolled-earn",
			request: &loyaltyv1.EarnPointsRequest{
				Amount:        1000,
				ReferenceId:   "order-not-enrolled",
				ReferenceType: "ORDER",
				Description:   "Not enrolled test",
			},
			setupFixtures:  func() string { return "" },
			expectedStatus: http.StatusNotFound,
			expectedError:  "CUSTOMER_NOT_FOUND",
		},
		{
			name:      "Edge case: Missing authentication",
			accountID: "",
			request: &loyaltyv1.EarnPointsRequest{
				Amount:        1000,
				ReferenceId:   "order-no-auth",
				ReferenceType: "ORDER",
				Description:   "No auth test",
			},
			setupFixtures:  func() string { return "" },
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:      "Edge case: SQL injection in reference_id",
			accountID: "user-earn-sql",
			request: &loyaltyv1.EarnPointsRequest{
				Amount:        1000,
				ReferenceId:   "'; DROP TABLE point_transactions; --",
				ReferenceType: "ORDER",
				Description:   "SQL injection test",
			},
			setupFixtures: func() string {
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id": "user-earn-sql",
					"tier_id":    &baseTier.ID,
				})
				return customer.ID
			},
			expectedStatus: http.StatusCreated, // Should handle safely with parameterized queries
			validateResponse: func(t *testing.T, resp *loyaltyv1.EarnPointsResponse, customerID string) {
				if resp.Transaction == nil {
					t.Fatal("Expected transaction in response")
				}
				
				// Build expected - verify SQL injection string stored safely
				expected := &loyaltyv1.EarnPointsResponse{
					Transaction: &loyaltyv1.PointTransaction{
						Id:            resp.Transaction.Id,
						CustomerId:    customerID,
						Amount:        resp.Transaction.Amount, // Calculated amount
						Type:          loyaltyv1.TransactionType_TRANSACTION_TYPE_EARN,
						ReferenceId:   "'; DROP TABLE point_transactions; --", // SQL string stored safely
						ReferenceType: "ORDER",
						Description:   "SQL injection test",
						CampaignId:    resp.Transaction.CampaignId,
						CreatedAt:     resp.Transaction.CreatedAt,
						ExpiresAt:     resp.Transaction.ExpiresAt,
					},
					NewBalance:      resp.NewBalance,
					CampaignApplied: resp.CampaignApplied,
				}
				
				// Compare using protocmp (MANDATORY)
				if diff := cmp.Diff(expected, resp, protocmp.Transform()); diff != "" {
					t.Errorf("Response mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:      "Edge case: Extremely large amount (overflow protection)",
			accountID: "user-earn-large",
			request: &loyaltyv1.EarnPointsRequest{
				Amount:        9223372036854775807, // Max int64
				ReferenceId:   "order-large",
				ReferenceType: "ORDER",
				Description:   "Large amount test",
			},
			setupFixtures: func() string {
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id": "user-earn-large",
					"tier_id":    &baseTier.ID,
				})
				return customer.ID
			},
			expectedStatus: http.StatusCreated,
			validateResponse: func(t *testing.T, resp *loyaltyv1.EarnPointsResponse, customerID string) {
				if resp.Transaction == nil {
					t.Fatal("Expected transaction in response")
				}
				
				// Verify overflow handled correctly (amount should be positive and large)
				// Max int64 cents / 100 = huge points but valid calculation
				// With campaign (2.0x), could be even larger, so just verify positive and reasonable
				
				if resp.Transaction.Amount <= 0 {
					t.Errorf("Expected positive points for large amount, got %d", resp.Transaction.Amount)
				}
				
				// Verify other fields match request
				if resp.Transaction.ReferenceId != "order-large" {
					t.Errorf("Expected reference_id 'order-large', got %s", resp.Transaction.ReferenceId)
				}
				if resp.Transaction.Type != loyaltyv1.TransactionType_TRANSACTION_TYPE_EARN {
					t.Error("Expected EARN transaction type")
				}
			},
		},
	}

	// Keep track of campaign for validation
	_ = campaign

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup fixtures for this test case
			customerID := tc.setupFixtures()

			// Create request
			body, err := json.Marshal(tc.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/v1/loyalty/points/earn", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Add authentication to context (mock)
			if tc.accountID != "" {
				ctx := context.WithValue(req.Context(), middleware.UserIDKey, tc.accountID)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()

			// Create services and handler
			transactionService := services.NewTransactionService(db)
			loyaltyService := services.NewLoyaltyService(db)
			handler := NewPointsHandler(transactionService, loyaltyService)

			// Call handler
			handler.HandleEarnPoints(rec, req)

			// Verify response status
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, rec.Code)
			}

			// For success cases, validate response
			if tc.expectedStatus == http.StatusCreated && tc.validateResponse != nil {
				var resp loyaltyv1.EarnPointsResponse
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				tc.validateResponse(t, &resp, customerID)
			}

			// For error cases, validate error code
			if tc.expectedError != "" {
				var errResp map[string]interface{}
				if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
					t.Fatalf("Failed to decode error response: %v", err)
				}
				if errResp["code"] != tc.expectedError {
					t.Errorf("Expected error code %s, got %v", tc.expectedError, errResp["code"])
				}
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}

func TestListTransactions(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "point_transactions", "customers", "membership_tiers")

	// Create base tier
	baseTier := testutil.CreateTestTier(db, map[string]interface{}{
		"name":                 "Base",
		"level":                0,
		"qualification_points": int64(0),
		"earn_rate_multiplier": 1.0,
	})

	// Create test customer with multiple transactions
	customer := testutil.CreateTestCustomer(db, map[string]interface{}{
		"account_id": "user-list-txn",
		"tier_id":    &baseTier.ID,
	})

	// Create various transaction types
	for i := 0; i < 5; i++ {
		testutil.CreateTestTransaction(db, customer.ID, map[string]interface{}{
			"amount":      int64(100 + i*10),
			"type":        models.TransactionTypeEarn,
			"description": fmt.Sprintf("EARN transaction %d", i+1),
		})
	}
	for i := 0; i < 3; i++ {
		testutil.CreateTestTransaction(db, customer.ID, map[string]interface{}{
			"amount":      int64(-50 - i*5),
			"type":        models.TransactionTypeRedemption,
			"description": fmt.Sprintf("REDEMPTION transaction %d", i+1),
		})
	}

	// Table-driven test cases
	testCases := []struct {
		name             string
		accountID        string
		queryParams      string
		setupFixtures    func()
		expectedStatus   int
		expectedError    string
		validateResponse func(t *testing.T, resp *loyaltyv1.ListTransactionsResponse)
	}{
		{
			name:          "Happy path: List all transactions (no filters)",
			accountID:     "user-list-txn",
			queryParams:   "",
			setupFixtures: func() {},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *loyaltyv1.ListTransactionsResponse) {
				// Should return all 8 transactions (5 EARN + 3 REDEMPTION)
				if len(resp.Transactions) != 8 {
					t.Errorf("Expected 8 transactions, got %d", len(resp.Transactions))
				}
				if resp.Total != 8 {
					t.Errorf("Expected total 8, got %d", resp.Total)
				}
				// Verify ordered by created_at DESC (newest first)
				if len(resp.Transactions) > 1 {
					firstTime := resp.Transactions[0].CreatedAt.AsTime()
					lastTime := resp.Transactions[len(resp.Transactions)-1].CreatedAt.AsTime()
					if firstTime.Before(lastTime) {
						t.Error("Expected transactions ordered by created_at DESC")
					}
				}
			},
		},
		{
			name:          "Happy path: Filter by type (EARN only)",
			accountID:     "user-list-txn",
			queryParams:   "?type=EARN",
			setupFixtures: func() {},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *loyaltyv1.ListTransactionsResponse) {
				// Should return only 5 EARN transactions
				if len(resp.Transactions) != 5 {
					t.Errorf("Expected 5 EARN transactions, got %d", len(resp.Transactions))
				}
				// Verify all are EARN type
				for _, tx := range resp.Transactions {
					if tx.Type != loyaltyv1.TransactionType_TRANSACTION_TYPE_EARN {
						t.Error("Expected only EARN transactions")
					}
				}
			},
		},
		{
			name:          "Happy path: Pagination (limit 3)",
			accountID:     "user-list-txn",
			queryParams:   "?limit=3&offset=0",
			setupFixtures: func() {},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *loyaltyv1.ListTransactionsResponse) {
				if len(resp.Transactions) != 3 {
					t.Errorf("Expected 3 transactions (limit), got %d", len(resp.Transactions))
				}
				if resp.Limit != 3 {
					t.Errorf("Expected limit 3, got %d", resp.Limit)
				}
				if resp.Total != 8 {
					t.Errorf("Expected total 8, got %d", resp.Total)
				}
			},
		},
		{
			name:          "Happy path: Empty result set (new customer)",
			accountID:     "user-no-transactions",
			queryParams:   "",
			setupFixtures: func() {
				testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id": "user-no-transactions",
					"tier_id":    &baseTier.ID,
				})
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *loyaltyv1.ListTransactionsResponse) {
				if len(resp.Transactions) != 0 {
					t.Errorf("Expected 0 transactions, got %d", len(resp.Transactions))
				}
				if resp.Total != 0 {
					t.Errorf("Expected total 0, got %d", resp.Total)
				}
			},
		},
		{
			name:           "Edge case: Customer not enrolled",
			accountID:      "user-not-enrolled-list",
			queryParams:    "",
			setupFixtures:  func() {},
			expectedStatus: http.StatusNotFound,
			expectedError:  "CUSTOMER_NOT_FOUND",
		},
		{
			name:           "Edge case: Missing authentication",
			accountID:      "",
			queryParams:    "",
			setupFixtures:  func() {},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup fixtures for this test case
			tc.setupFixtures()

			// Create request
			url := "/v1/loyalty/transactions" + tc.queryParams
			req := httptest.NewRequest(http.MethodGet, url, nil)

			// Add authentication to context (mock)
			if tc.accountID != "" {
				ctx := context.WithValue(req.Context(), middleware.UserIDKey, tc.accountID)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()

			// Create services and handler
			transactionService := services.NewTransactionService(db)
			loyaltyService := services.NewLoyaltyService(db)
			handler := NewPointsHandler(transactionService, loyaltyService)

			// Call handler
			handler.HandleListTransactions(rec, req)

			// Verify response status
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, rec.Code)
			}

			// For success cases, validate response
			if tc.expectedStatus == http.StatusOK && tc.validateResponse != nil {
				var resp loyaltyv1.ListTransactionsResponse
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				tc.validateResponse(t, &resp)
			}

			// For error cases, validate error code
			if tc.expectedError != "" {
				var errResp map[string]interface{}
				if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
					t.Fatalf("Failed to decode error response: %v", err)
				}
				if errResp["code"] != tc.expectedError {
					t.Errorf("Expected error code %s, got %v", tc.expectedError, errResp["code"])
				}
			}
		})
	}
}

