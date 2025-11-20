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

func TestListRewards(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "rewards")

	// Create multiple rewards (active and inactive, various point costs)
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
	// Create inactive reward directly to avoid fixture bool issue
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
		queryParams      string
		expectedStatus   int
		validateResponse func(t *testing.T, resp *loyaltyv1.ListRewardsResponse)
	}{
		{
			name:           "Happy path: List all active rewards",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *loyaltyv1.ListRewardsResponse) {
				// Should return only 2 active rewards (default active_only=true)
				// Free Coffee (100) and $5 Discount (500) are active
				// Inactive Reward (200) should be filtered out
				if len(resp.Rewards) != 2 {
					t.Errorf("Expected 2 active rewards, got %d", len(resp.Rewards))
					for _, r := range resp.Rewards {
						t.Logf("  - %s: active=%v, cost=%d", r.Name, r.IsActive, r.PointCost)
					}
				}
				// Verify all returned rewards are active
				for _, reward := range resp.Rewards {
					if !reward.IsActive {
						t.Errorf("Reward %s should be active but is not", reward.Name)
					}
				}
			},
		},
		{
			name:           "Happy path: Include inactive rewards",
			queryParams:    "?active_only=false",
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *loyaltyv1.ListRewardsResponse) {
				// Should return all 3 rewards (2 active + 1 inactive) when active_only=false
				if len(resp.Rewards) != 3 {
					t.Errorf("Expected 3 total rewards, got %d", len(resp.Rewards))
					for _, r := range resp.Rewards {
						t.Logf("  - %s: active=%v, cost=%d", r.Name, r.IsActive, r.PointCost)
					}
				}
				// Verify we have mix of active and inactive
				hasActive := false
				hasInactive := false
				for _, r := range resp.Rewards {
					if r.IsActive {
						hasActive = true
					} else {
						hasInactive = true
					}
				}
				if !hasActive || !hasInactive {
					t.Error("Expected mix of active and inactive rewards when active_only=false")
				}
			},
		},
		{
			name:           "Happy path: Filter by max_points (under budget)",
			queryParams:    "?max_points=200",
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *loyaltyv1.ListRewardsResponse) {
				// Should return only ACTIVE rewards with point_cost <= 200
				// Coffee (100, active) ✓
				// Discount (500, active) ✗ (over budget)
				// Inactive Reward (200, inactive) ✗ (not active by default)
				if len(resp.Rewards) != 1 {
					t.Errorf("Expected 1 active reward under 200 points, got %d", len(resp.Rewards))
					for _, r := range resp.Rewards {
						t.Logf("  - %s: active=%v, cost=%d", r.Name, r.IsActive, r.PointCost)
					}
				}
				// Verify all returned rewards are within budget and active
				for _, reward := range resp.Rewards {
					if reward.PointCost > 200 {
						t.Errorf("Reward %s has cost %d > 200", reward.Name, reward.PointCost)
					}
					if !reward.IsActive {
						t.Error("Expected only active rewards by default")
					}
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

func TestRedeemReward(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "redemptions", "point_transactions", "customers", "rewards", "membership_tiers")

	// Create base tier
	baseTier := testutil.CreateTestTier(db, map[string]interface{}{
		"name":                 "Base",
		"level":                0,
		"qualification_points": int64(0),
		"earn_rate_multiplier": 1.0,
	})

	// Create active reward
	reward := testutil.CreateTestReward(db, map[string]interface{}{
		"name":       "$5 Discount Code",
		"type":       models.RewardTypeDiscount,
		"point_cost": int64(500),
		"is_active":  true,
	})

	// Create inactive reward for test case
	inactiveReward := testutil.CreateTestReward(db, map[string]interface{}{
		"name":       "Inactive Reward for Test",
		"type":       models.RewardTypeDiscount,
		"point_cost": int64(100),
		"is_active":  false, // Now works properly after model fix
	})

	// Table-driven test cases
	testCases := []struct {
		name             string
		accountID        string
		request          *loyaltyv1.RedeemRewardRequest
		setupFixtures    func()
		expectedStatus   int
		expectedError    string
		validateResponse func(t *testing.T, resp *loyaltyv1.RedeemRewardResponse)
	}{
		{
			name:      "Happy path: Customer with sufficient points redeems reward",
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

				// Build expected from REQUEST data
				expected := &loyaltyv1.RedeemRewardResponse{
					Redemption: &loyaltyv1.Redemption{
						Id:             resp.Redemption.Id, // Generated
						CustomerId:     resp.Redemption.CustomerId,
						Reward: &loyaltyv1.Reward{
							Id:        reward.ID,
							Name:      "$5 Discount Code",
							Type:      loyaltyv1.RewardType_REWARD_TYPE_DISCOUNT,
							PointCost: 500,
							// Other fields from response
							Description: resp.Redemption.Reward.Description,
							IsActive:    resp.Redemption.Reward.IsActive,
							Metadata:    resp.Redemption.Reward.Metadata,
							CreatedAt:   resp.Redemption.Reward.CreatedAt,
							UpdatedAt:   resp.Redemption.Reward.UpdatedAt,
						},
						PointsDeducted: 500, // From reward.PointCost
						Status:         loyaltyv1.RedemptionStatus_REDEMPTION_STATUS_ACTIVE,
						Code:           resp.Redemption.Code, // Generated code
						CreatedAt:      resp.Redemption.CreatedAt,
					},
					NewBalance: 500, // 1000 - 500 = 500
					Transaction: resp.Transaction, // Use from response
				}

				// Compare using protocmp
				if diff := cmp.Diff(expected, resp, protocmp.Transform()); diff != "" {
					t.Errorf("Response mismatch (-want +got):\n%s", diff)
				}

				// Verify code generated (non-empty)
				if resp.Redemption.Code == "" {
					t.Error("Expected redemption code to be generated")
				}
			},
		},
		{
			name:      "Edge case: Insufficient balance",
			accountID: "user-redeem-insufficient",
			request: &loyaltyv1.RedeemRewardRequest{
				RewardId: reward.ID,
			},
			setupFixtures: func() {
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
			name:      "Edge case: Reward not found",
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
			name:      "Edge case: Reward inactive",
			accountID: "user-redeem-inactive",
			request: &loyaltyv1.RedeemRewardRequest{
				RewardId: inactiveReward.ID, // Use inactive reward created above
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
			name:      "Edge case: Customer not enrolled",
			accountID: "user-not-enrolled-redeem",
			request: &loyaltyv1.RedeemRewardRequest{
				RewardId: reward.ID,
			},
			setupFixtures:  func() {},
			expectedStatus: http.StatusNotFound,
			expectedError:  "CUSTOMER_NOT_FOUND",
		},
		{
			name:      "Edge case: Missing authentication",
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

func TestListRedemptions(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "redemptions", "customers", "rewards", "membership_tiers")

	// Create base tier
	baseTier := testutil.CreateTestTier(db, map[string]interface{}{
		"name":  "Base",
		"level": 0,
	})

	// Create customer with multiple redemptions
	customer := testutil.CreateTestCustomer(db, map[string]interface{}{
		"account_id": "user-list-redemptions",
		"tier_id":    &baseTier.ID,
	})

	// Create rewards
	reward1 := testutil.CreateTestReward(db, map[string]interface{}{
		"name":       "Reward 1",
		"point_cost": int64(100),
	})
	reward2 := testutil.CreateTestReward(db, map[string]interface{}{
		"name":       "Reward 2",
		"point_cost": int64(200),
	})

	// Create redemptions with different statuses
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
		accountID        string
		queryParams      string
		expectedStatus   int
		expectedError    string
		validateResponse func(t *testing.T, resp *loyaltyv1.ListRedemptionsResponse)
	}{
		{
			name:           "Happy path: List all redemptions",
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
			name:           "Happy path: Filter by status (ACTIVE only)",
			accountID:      "user-list-redemptions",
			queryParams:    "?status=ACTIVE",
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *loyaltyv1.ListRedemptionsResponse) {
				if len(resp.Redemptions) != 1 {
					t.Errorf("Expected 1 ACTIVE redemption, got %d", len(resp.Redemptions))
				}
				if len(resp.Redemptions) > 0 && resp.Redemptions[0].Status != loyaltyv1.RedemptionStatus_REDEMPTION_STATUS_ACTIVE {
					t.Error("Expected only ACTIVE redemptions")
				}
			},
		},
		{
			name:           "Happy path: Empty redemption history",
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
			name:           "Edge case: Customer not enrolled",
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

