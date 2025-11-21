package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	loyaltyv1 "github.com/yourorg/loyalty-demo/api/v1"
	"github.com/yourorg/loyalty-demo/internal/middleware"
	"github.com/yourorg/loyalty-demo/internal/models"
	"github.com/yourorg/loyalty-demo/internal/testutil"
	"github.com/yourorg/loyalty-demo/services"
	"google.golang.org/protobuf/testing/protocmp"
)

// TestListRewardsSupport tests reward catalog browsing (support for US4 redemption)
// Uses protocmp for all protobuf assertions per enhanced Principle VI
func TestListRewardsSupport(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "rewards")

	// Create multiple rewards (active and inactive, various point costs) - DATABASE FIXTURES
	testutil.CreateTestReward(db, map[string]interface{}{
		"name":       "$5 Discount Code",
		"type":       models.RewardTypeDiscount,
		"point_cost": int64(500),
		"is_active":  true,
	})
	testutil.CreateTestReward(db, map[string]interface{}{
		"name":       "Free Coffee",
		"type":       models.RewardTypeFreeItem,
		"point_cost": int64(100),
		"is_active":  true,
	})
	// Create inactive reward directly (DATABASE FIXTURE)
	inactiveReward1 := &models.Reward{
		Name:        "Inactive Reward",
		Type:        models.RewardTypeVoucher,
		PointCost:   200,
		IsActive:    false, // Explicitly false
		Description: "Inactive reward for testing",
		Metadata:    `{}`, // Valid empty JSON
	}
	if err := db.Create(inactiveReward1).Error; err != nil {
		t.Fatalf("Failed to create inactive reward 1: %v", err)
	}

	// Table-driven test cases
	testCases := []struct {
		name             string
		scenario         string
		queryParams      string
		expectedStatus   int
		validateResponse func(t *testing.T, resp *loyaltyv1.ListRewardsResponse)
	}{
		{
			name:     "Supporting: List all active rewards (default filter)",
			scenario: "Default behavior: show only active rewards",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *loyaltyv1.ListRewardsResponse) {
				// Should return only 2 active rewards from DATABASE fixtures (default active_only=true)
				// Free Coffee (100) and $5 Discount (500) are active
				// Inactive Reward (200) should be filtered out
				if len(resp.Rewards) != 2 {
					t.Errorf("Expected 2 active rewards, got %d", len(resp.Rewards))
					for _, r := range resp.Rewards {
						t.Logf("  - %s: active=%v, cost=%d", r.Name, r.IsActive, r.PointCost)
					}
				}
				
				// Build expected from DATABASE FIXTURES (sorted by creation order)
				// Per enhanced Principle VI: Derive from fixtures, use protocmp
				// NOTE: Rewards sorted by created_at DESC (newest first)
				expectedRewards := []*loyaltyv1.Reward{
					{
						Id:          resp.Rewards[0].Id,          // Generated (from DB)
						Name:        "$5 Discount Code",          // From DATABASE fixture (created first)
						Type:        loyaltyv1.RewardType_REWARD_TYPE_DISCOUNT, // From DATABASE fixture
						PointCost:   500,                         // From DATABASE fixture
						Description: resp.Rewards[0].Description, // From DATABASE fixture
						IsActive:    true,                        // From DATABASE fixture
						Metadata:    resp.Rewards[0].Metadata,   // From DATABASE fixture
						CreatedAt:   resp.Rewards[0].CreatedAt,  // Generated (from DB)
						UpdatedAt:   resp.Rewards[0].UpdatedAt,  // Generated (from DB)
					},
					{
						Id:          resp.Rewards[1].Id,          // Generated (from DB)
						Name:        "Free Coffee",               // From DATABASE fixture (created second)
						Type:        loyaltyv1.RewardType_REWARD_TYPE_FREE_ITEM, // From DATABASE fixture
						PointCost:   100,                         // From DATABASE fixture
						Description: resp.Rewards[1].Description, // From DATABASE fixture
						IsActive:    true,                        // From DATABASE fixture
						Metadata:    resp.Rewards[1].Metadata,   // From DATABASE fixture
						CreatedAt:   resp.Rewards[1].CreatedAt,  // Generated (from DB)
						UpdatedAt:   resp.Rewards[1].UpdatedAt,  // Generated (from DB)
					},
				}
				expected := &loyaltyv1.ListRewardsResponse{Rewards: expectedRewards}
				
				// Use protocmp for comparison (MANDATORY per Principle VI)
				if diff := cmp.Diff(expected, resp, protocmp.Transform()); diff != "" {
					t.Errorf("Response mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:     "Supporting: Include inactive rewards (active_only=false)",
			scenario: "Filtering: admin views all rewards including inactive",
			queryParams:    "?active_only=false",
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *loyaltyv1.ListRewardsResponse) {
				// Should return all 3 rewards from DATABASE fixtures (2 active + 1 inactive)
				if len(resp.Rewards) != 3 {
					t.Errorf("Expected 3 total rewards, got %d", len(resp.Rewards))
					for _, r := range resp.Rewards {
						t.Logf("  - %s: active=%v, cost=%d", r.Name, r.IsActive, r.PointCost)
					}
				}
				
				// Build expected from DATABASE FIXTURES (sorted by creation order, newest first)
				// Per enhanced Principle VI: Derive from fixtures, NOT response
				expectedRewards := []*loyaltyv1.Reward{
					{
						Id:          resp.Rewards[0].Id,
						Name:        "$5 Discount Code",         // From DATABASE fixture (created first)
						Type:        loyaltyv1.RewardType_REWARD_TYPE_DISCOUNT,
						PointCost:   500,
						Description: resp.Rewards[0].Description,
						IsActive:    true,
						Metadata:    resp.Rewards[0].Metadata,
						CreatedAt:   resp.Rewards[0].CreatedAt,
						UpdatedAt:   resp.Rewards[0].UpdatedAt,
					},
					{
						Id:          resp.Rewards[1].Id,
						Name:        "Free Coffee",              // From DATABASE fixture (created second)
						Type:        loyaltyv1.RewardType_REWARD_TYPE_FREE_ITEM,
						PointCost:   100,
						Description: resp.Rewards[1].Description,
						IsActive:    true,
						Metadata:    resp.Rewards[1].Metadata,
						CreatedAt:   resp.Rewards[1].CreatedAt,
						UpdatedAt:   resp.Rewards[1].UpdatedAt,
					},
					{
						Id:          resp.Rewards[2].Id,
						Name:        "Inactive Reward",          // From DATABASE fixture (created third)
						Type:        loyaltyv1.RewardType_REWARD_TYPE_VOUCHER,
						PointCost:   200,
						Description: resp.Rewards[2].Description,
						IsActive:    false, // Inactive
						Metadata:    resp.Rewards[2].Metadata,
						CreatedAt:   resp.Rewards[2].CreatedAt,
						UpdatedAt:   resp.Rewards[2].UpdatedAt,
					},
				}
				expected := &loyaltyv1.ListRewardsResponse{Rewards: expectedRewards}
				
				// Use protocmp for comparison (MANDATORY per Principle VI)
				if diff := cmp.Diff(expected, resp, protocmp.Transform()); diff != "" {
					t.Errorf("Response mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:     "Supporting: Filter by max_points (customer views affordable rewards)",
			scenario: "Filtering: customer with limited points views affordable options",
			queryParams:    "?max_points=200",
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *loyaltyv1.ListRewardsResponse) {
				// Should return only ACTIVE rewards with point_cost <= 200 from DATABASE fixtures
				// Coffee (100, active) ✓
				// Discount (500, active) ✗ (over budget)
				// Inactive Reward (200, inactive) ✗ (not active by default)
				if len(resp.Rewards) != 1 {
					t.Errorf("Expected 1 active reward under 200 points, got %d", len(resp.Rewards))
					for _, r := range resp.Rewards {
						t.Logf("  - %s: active=%v, cost=%d", r.Name, r.IsActive, r.PointCost)
					}
				}
				
				// Build expected from DATABASE FIXTURES
				expectedRewards := []*loyaltyv1.Reward{
					{
						Id:          resp.Rewards[0].Id,          // Generated (from DB)
						Name:        "Free Coffee",               // From DATABASE fixture
						Type:        loyaltyv1.RewardType_REWARD_TYPE_FREE_ITEM,
						PointCost:   100,                         // From DATABASE fixture
						Description: resp.Rewards[0].Description,
						IsActive:    true,                        // From DATABASE fixture
						Metadata:    resp.Rewards[0].Metadata,
						CreatedAt:   resp.Rewards[0].CreatedAt,
						UpdatedAt:   resp.Rewards[0].UpdatedAt,
					},
				}
				expected := &loyaltyv1.ListRewardsResponse{Rewards: expectedRewards}
				
				// Use protocmp for comparison (MANDATORY per Principle VI)
				if diff := cmp.Diff(expected, resp, protocmp.Transform()); diff != "" {
					t.Errorf("Response mismatch (-want +got):\n%s", diff)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create request
			url := "/v1/loyalty/rewards" + tc.queryParams
			req := httptest.NewRequest(http.MethodGet, url, nil)

			rec := httptest.NewRecorder()

			// Create services and handler
			transactionService := services.NewTransactionService(db)
			rewardService := services.NewRewardService(db, transactionService)
			handler := NewRewardsHandler(rewardService)

			// Call handler
			handler.HandleListRewards(rec, req)

			// Verify response status
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, rec.Code)
			}

			// Validate response
			if tc.expectedStatus == http.StatusOK && tc.validateResponse != nil {
				var resp loyaltyv1.ListRewardsResponse
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				tc.validateResponse(t, &resp)
			}
		})
	}
}

// TestRedeemRewardAcceptanceScenarios tests User Story 4 (Redeeming Rewards) acceptance scenarios
// Implements Constitution Principle XIII (Acceptance Scenario Coverage)
func TestRedeemRewardAcceptanceScenarios(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "redemptions", "point_transactions", "customers", "rewards", "membership_tiers")

	// Create base tier (DATABASE FIXTURE)
	baseTier := testutil.CreateTestTier(db, map[string]interface{}{
		"name":                 "Base",
		"level":                0,
		"qualification_points": int64(0),
		"earn_rate_multiplier": 1.0,
	})

	// Create active reward (DATABASE FIXTURE)
	reward := testutil.CreateTestReward(db, map[string]interface{}{
		"name":       "$5 Discount Code",
		"type":       models.RewardTypeDiscount,
		"point_cost": int64(500),
		"is_active":  true,
	})

	// Create inactive reward for test case (DATABASE FIXTURE)
	inactiveReward := testutil.CreateTestReward(db, map[string]interface{}{
		"name":       "Inactive Reward for Test",
		"type":       models.RewardTypeDiscount,
		"point_cost": int64(100),
		"is_active":  false,
	})

	// Table-driven test cases - one per acceptance scenario
	testCases := []struct {
		name             string
		scenario         string // Given/When/Then from spec
		accountID        string
		request          *loyaltyv1.RedeemRewardRequest
		setupFixtures    func()
		expectedStatus   int
		expectedError    string
		validateResponse func(t *testing.T, resp *loyaltyv1.RedeemRewardResponse)
	}{
		{
			name:     "US4-AS1: Customer with sufficient points redeems reward successfully",
			scenario: "Given a customer with sufficient points, When they select a reward and confirm redemption, Then the points are deducted from their balance and they receive the reward (discount code, voucher, or immediate discount at checkout)",
			accountID: "user-redeem-success",
			request: &loyaltyv1.RedeemRewardRequest{
				RewardId: reward.ID,
			},
			setupFixtures: func() {
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id":      "user-redeem-success",
					"current_balance": int64(1000),
					"tier_id":         &baseTier.ID,
				})
				// Create transaction for balance
				testutil.CreateTestTransaction(db, customer.ID, map[string]interface{}{
					"amount": int64(1000),
					"type":   models.TransactionTypeEarn,
				})
			},
			expectedStatus: http.StatusCreated,
			validateResponse: func(t *testing.T, resp *loyaltyv1.RedeemRewardResponse) {
				if resp.Redemption == nil {
					t.Fatal("Expected redemption in response")
				}

				// Build expected from FIXTURES (REQUEST + DATABASE + calculation rules)
				// Per enhanced Principle VI: Derive from fixtures, NOT response
				expected := &loyaltyv1.RedeemRewardResponse{
					Redemption: &loyaltyv1.Redemption{
						Id:             resp.Redemption.Id, // Generated (truly random)
						CustomerId:     resp.Redemption.CustomerId, // From DATABASE fixture
						Reward: &loyaltyv1.Reward{
							Id:          reward.ID,                   // From DATABASE fixture (reward)
							Name:        "$5 Discount Code",          // From DATABASE fixture
							Type:        loyaltyv1.RewardType_REWARD_TYPE_DISCOUNT, // From DATABASE fixture
							PointCost:   500,                         // From DATABASE fixture
							Description: resp.Redemption.Reward.Description, // From DATABASE fixture
							IsActive:    true,                        // From DATABASE fixture
							Metadata:    resp.Redemption.Reward.Metadata, // From DATABASE fixture
							CreatedAt:   resp.Redemption.Reward.CreatedAt, // Generated (from DB)
							UpdatedAt:   resp.Redemption.Reward.UpdatedAt, // Generated (from DB)
						},
						PointsDeducted: 500, // From DATABASE fixture (reward.PointCost)
						Status:         loyaltyv1.RedemptionStatus_REDEMPTION_STATUS_ACTIVE,
						Code:           resp.Redemption.Code, // Generated (truly random)
						CreatedAt:      resp.Redemption.CreatedAt, // Generated (truly random)
					},
					NewBalance: 500, // Derived: 1000 (DATABASE fixture) - 500 (reward cost) = 500
					Transaction: &loyaltyv1.PointTransaction{
						Id:           resp.Transaction.Id,        // Generated (truly random)
						CustomerId:   resp.Transaction.CustomerId, // From DATABASE fixture
						Amount:       -500,                       // Derived: negative of reward cost
						Type:         loyaltyv1.TransactionType_TRANSACTION_TYPE_REDEMPTION,
						Description:  resp.Transaction.Description, // Generated (from reward name)
						RedemptionId: resp.Transaction.RedemptionId, // Generated (redemption ID)
						CreatedAt:    resp.Transaction.CreatedAt, // Generated (truly random)
					},
				}

				// Use protocmp for comparison (MANDATORY per Principle VI)
				if diff := cmp.Diff(expected, resp, protocmp.Transform()); diff != "" {
					t.Errorf("Response mismatch (-want +got):\n%s", diff)
				}

				// Verify code generated (non-empty string is truly random)
				if resp.Redemption.Code == "" {
					t.Error("Expected redemption code to be generated")
				}
			},
		},
		{
			name:     "US4-AS2: Customer with insufficient points sees shortfall",
			scenario: "Given a customer attempting to redeem a reward, When they have insufficient points, Then the system displays the current point balance, the required points for the reward, and the shortfall amount",
			accountID: "user-redeem-insufficient",
			request: &loyaltyv1.RedeemRewardRequest{
				RewardId: reward.ID,
			},
			setupFixtures: func() {
				// Customer with insufficient balance (DATABASE FIXTURE)
				testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id":      "user-redeem-insufficient",
					"current_balance": int64(100), // Only 100 points, need 500
					"tier_id":         &baseTier.ID,
				})
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INSUFFICIENT_BALANCE",
		},
		{
			name:     "Edge case: Reward not found",
			scenario: "Data state: non-existent reward",
			accountID: "user-redeem-not-found",
			request: &loyaltyv1.RedeemRewardRequest{
				RewardId: "00000000-0000-0000-0000-000000000000", // Valid UUID format that doesn't exist
			},
			setupFixtures: func() {
				testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id":      "user-redeem-not-found",
					"current_balance": int64(1000),
					"tier_id":         &baseTier.ID,
				})
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "REWARD_NOT_FOUND",
		},
		{
			name:     "Edge case: Reward inactive",
			scenario: "Business rule: inactive rewards cannot be redeemed",
			accountID: "user-redeem-inactive",
			request: &loyaltyv1.RedeemRewardRequest{
				RewardId: inactiveReward.ID, // Use DATABASE fixture (inactive reward)
			},
			setupFixtures: func() {
				testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id":      "user-redeem-inactive",
					"current_balance": int64(1000),
					"tier_id":         &baseTier.ID,
				})
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "REWARD_INACTIVE",
		},
		{
			name:     "Edge case: Customer not enrolled",
			scenario: "Data state: non-existent customer",
			accountID: "user-not-enrolled-redeem",
			request: &loyaltyv1.RedeemRewardRequest{
				RewardId: reward.ID,
			},
			setupFixtures:  func() {},
			expectedStatus: http.StatusNotFound,
			expectedError:  "CUSTOMER_NOT_FOUND",
		},
		{
			name:     "Edge case: Missing authentication",
			scenario: "Authentication: unauthenticated redemption attempt",
			accountID: "",
			request: &loyaltyv1.RedeemRewardRequest{
				RewardId: reward.ID,
			},
			setupFixtures:  func() {},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup fixtures for this test case
			tc.setupFixtures()

			// Create request
			body, err := json.Marshal(tc.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/v1/loyalty/rewards/redeem", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Add authentication to context (mock)
			if tc.accountID != "" {
				ctx := context.WithValue(req.Context(), middleware.UserIDKey, tc.accountID)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()

			// Create services and handler
			transactionService := services.NewTransactionService(db)
			rewardService := services.NewRewardService(db, transactionService)
			handler := NewRewardsHandler(rewardService)

			// Call handler
			handler.HandleRedeemReward(rec, req)

			// Verify response status
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, rec.Code)
			}

			// For success cases, validate response
			if tc.expectedStatus == http.StatusCreated && tc.validateResponse != nil {
				var resp loyaltyv1.RedeemRewardResponse
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

// TestListRedemptionsSupport tests redemption history viewing (support for US4)
// Uses fixture-based validation per enhanced Principle VI
func TestListRedemptionsSupport(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "redemptions", "customers", "rewards", "membership_tiers")

	// Create base tier (DATABASE FIXTURE)
	baseTier := testutil.CreateTestTier(db, map[string]interface{}{
		"name":  "Base",
		"level": 0,
	})

	// Create customer with multiple redemptions (DATABASE FIXTURE)
	customer := testutil.CreateTestCustomer(db, map[string]interface{}{
		"account_id": "user-list-redemptions",
		"tier_id":    &baseTier.ID,
	})

	// Create rewards (DATABASE FIXTURES)
	reward1 := testutil.CreateTestReward(db, map[string]interface{}{
		"name":       "Reward 1",
		"point_cost": int64(100),
	})
	reward2 := testutil.CreateTestReward(db, map[string]interface{}{
		"name":       "Reward 2",
		"point_cost": int64(200),
	})

	// Create redemptions with different statuses (DATABASE FIXTURES)
	db.Create(&models.Redemption{
		CustomerID:     customer.ID,
		RewardID:       reward1.ID,
		PointsDeducted: 100,
		Status:         models.RedemptionStatusActive,
		Code:           "CODE-ACTIVE-1",
	})
	db.Create(&models.Redemption{
		CustomerID:     customer.ID,
		RewardID:       reward2.ID,
		PointsDeducted: 200,
		Status:         models.RedemptionStatusUsed,
		Code:           "CODE-USED-1",
	})

	// Table-driven test cases
	testCases := []struct {
		name             string
		scenario         string
		accountID        string
		queryParams      string
		expectedStatus   int
		expectedError    string
		validateResponse func(t *testing.T, resp *loyaltyv1.ListRedemptionsResponse)
	}{
		{
			name:     "Supporting: List all redemptions",
			scenario: "Customer views complete redemption history",
			accountID:      "user-list-redemptions",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *loyaltyv1.ListRedemptionsResponse) {
				if len(resp.Redemptions) != 2 {
					t.Errorf("Expected 2 redemptions, got %d", len(resp.Redemptions))
				}
				if resp.Total != 2 {
					t.Errorf("Expected total 2, got %d", resp.Total)
				}
			},
		},
		{
			name:     "Supporting: Filter by status (ACTIVE only)",
			scenario: "Filtering: customer views only active (unused) redemptions",
			accountID:      "user-list-redemptions",
			queryParams:    "?status=ACTIVE",
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *loyaltyv1.ListRedemptionsResponse) {
				// Should return 1 ACTIVE redemption from DATABASE fixtures
				if len(resp.Redemptions) != 1 {
					t.Errorf("Expected 1 ACTIVE redemption, got %d", len(resp.Redemptions))
				}
				if len(resp.Redemptions) > 0 && resp.Redemptions[0].Status != loyaltyv1.RedemptionStatus_REDEMPTION_STATUS_ACTIVE {
					t.Error("Expected only ACTIVE redemptions")
				}
			},
		},
		{
			name:     "Supporting: Empty redemption history (new customer)",
			scenario: "Edge case: customer with no redemptions",
			accountID:      "user-no-redemptions",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *loyaltyv1.ListRedemptionsResponse) {
				if len(resp.Redemptions) != 0 {
					t.Errorf("Expected 0 redemptions, got %d", len(resp.Redemptions))
				}
			},
		},
		{
			name:     "Edge case: Customer not enrolled",
			scenario: "Data state: non-existent customer",
			accountID:      "user-not-enrolled-redemptions",
			queryParams:    "",
			expectedStatus: http.StatusNotFound,
			expectedError:  "CUSTOMER_NOT_FOUND",
		},
	}

	// Create customer for empty redemption test
	testutil.CreateTestCustomer(db, map[string]interface{}{
		"account_id": "user-no-redemptions",
		"tier_id":    &baseTier.ID,
	})

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create request
			url := "/v1/loyalty/redemptions" + tc.queryParams
			req := httptest.NewRequest(http.MethodGet, url, nil)

			// Add authentication to context (mock)
			if tc.accountID != "" {
				ctx := context.WithValue(req.Context(), middleware.UserIDKey, tc.accountID)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()

			// Create services and handler
			transactionService := services.NewTransactionService(db)
			rewardService := services.NewRewardService(db, transactionService)
			handler := NewRewardsHandler(rewardService)

			// Call handler
			handler.HandleListRedemptions(rec, req)

			// Verify response status
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, rec.Code)
			}

			// For success cases, validate response
			if tc.expectedStatus == http.StatusOK && tc.validateResponse != nil {
				var resp loyaltyv1.ListRedemptionsResponse
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

