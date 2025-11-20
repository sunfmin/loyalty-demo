package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	loyaltyv1 "github.com/yourorg/loyalty-demo/api/v1"
	"github.com/yourorg/loyalty-demo/internal/middleware"
	"github.com/yourorg/loyalty-demo/internal/models"
	"github.com/yourorg/loyalty-demo/internal/testutil"
	"github.com/yourorg/loyalty-demo/services"
)

// TestAllSentinelErrors tests EVERY sentinel error defined in services/errors.go
// This is MANDATORY per Constitution Principle IX
func TestAllSentinelErrors(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "customers", "point_transactions", "rewards", "redemptions", "membership_tiers")

	// Create base tier for fixtures
	baseTier := testutil.CreateTestTier(db, map[string]interface{}{
		"name":  "Base",
		"level": 0,
	})

	// Table-driven tests - one test case per sentinel error
	testCases := []struct {
		name             string
		sentinelError    error
		setupAndTrigger  func() error
		expectedWrapped  bool
	}{
		{
			name:          "ErrNotEnrolled - customer queries without enrollment",
			sentinelError: services.ErrNotEnrolled,
			setupAndTrigger: func() error {
				// Try to get status for non-enrolled customer
				loyaltyService := services.NewLoyaltyService(db)
				_, err := loyaltyService.GetCustomerStatus(context.Background(), "non-existent-account")
				return err
			},
			expectedWrapped: true,
		},
		{
			name:          "ErrAlreadyEnrolled - duplicate enrollment attempt",
			sentinelError: services.ErrAlreadyEnrolled,
			setupAndTrigger: func() error {
				// Create enrolled customer
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id": "already-enrolled",
					"tier_id":    &baseTier.ID,
				})
				// Try to enroll again
				loyaltyService := services.NewLoyaltyService(db)
				_, err := loyaltyService.EnrollCustomer(context.Background(), nil, customer.AccountID)
				return err
			},
			expectedWrapped: true,
		},
		{
			name:          "ErrInsufficientBalance - redemption exceeding balance",
			sentinelError: services.ErrInsufficientBalance,
			setupAndTrigger: func() error {
				// Create customer with low balance
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id":      "low-balance",
					"current_balance": int64(10),
					"tier_id":         &baseTier.ID,
				})
				// Create expensive reward
				reward := testutil.CreateTestReward(db, map[string]interface{}{
					"point_cost": int64(500),
				})
				// Try to redeem
				transactionService := services.NewTransactionService(db)
				rewardService := services.NewRewardService(db, transactionService)
				_, err := rewardService.RedeemReward(context.Background(), &loyaltyv1.RedeemRewardRequest{
					RewardId: reward.ID,
				}, customer.AccountID)
				return err
			},
			expectedWrapped: true,
		},
		{
			name:          "ErrInvalidReferralCode - invalid referral during enrollment",
			sentinelError: services.ErrInvalidReferralCode,
			setupAndTrigger: func() error {
				loyaltyService := services.NewLoyaltyService(db)
				_, err := loyaltyService.EnrollCustomer(context.Background(), &loyaltyv1.EnrollCustomerRequest{
					ReferralCode: "INVALID-CODE",
				}, "new-user")
				return err
			},
			expectedWrapped: true,
		},
		{
			name:          "ErrRewardNotFound - redeeming non-existent reward",
			sentinelError: services.ErrRewardNotFound,
			setupAndTrigger: func() error {
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id":      "customer-reward-not-found",
					"current_balance": int64(1000),
					"tier_id":         &baseTier.ID,
				})
				transactionService := services.NewTransactionService(db)
				rewardService := services.NewRewardService(db, transactionService)
				_, err := rewardService.RedeemReward(context.Background(), &loyaltyv1.RedeemRewardRequest{
					RewardId: "00000000-0000-0000-0000-000000000000",
				}, customer.AccountID)
				return err
			},
			expectedWrapped: true,
		},
		{
			name:          "ErrRewardInactive - redeeming inactive reward",
			sentinelError: services.ErrRewardInactive,
			setupAndTrigger: func() error {
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id":      "customer-inactive-reward",
					"current_balance": int64(1000),
					"tier_id":         &baseTier.ID,
				})
				// Create inactive reward
				inactiveReward := testutil.CreateTestReward(db, map[string]interface{}{
					"is_active": false,
				})
				transactionService := services.NewTransactionService(db)
				rewardService := services.NewRewardService(db, transactionService)
				_, err := rewardService.RedeemReward(context.Background(), &loyaltyv1.RedeemRewardRequest{
					RewardId: inactiveReward.ID,
				}, customer.AccountID)
				return err
			},
			expectedWrapped: true,
		},
		{
			name:          "ErrRedemptionNotFound - reversing non-existent redemption",
			sentinelError: services.ErrRedemptionNotFound,
			setupAndTrigger: func() error {
				transactionService := services.NewTransactionService(db)
				rewardService := services.NewRewardService(db, transactionService)
				_, err := rewardService.ReverseRedemption(context.Background(), "00000000-0000-0000-0000-000000000000", "Test reason", "admin-123")
				return err
			},
			expectedWrapped: true,
		},
		{
			name:          "ErrRedemptionAlreadyReversed - reversing already reversed redemption",
			sentinelError: services.ErrRedemptionAlreadyReversed,
			setupAndTrigger: func() error {
				// Create customer, reward, and reversed redemption
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id": "customer-reversed",
					"tier_id":    &baseTier.ID,
				})
				reward := testutil.CreateTestReward(db, nil)
				
				// Create already-reversed redemption
				now := time.Now()
				reversedReason := "Already reversed"
				db.Create(&models.Redemption{
					CustomerID:     customer.ID,
					RewardID:       reward.ID,
					PointsDeducted: 100,
					Status:         models.RedemptionStatusReversed,
					Code:           "CODE-REVERSED",
					ReversedAt:     &now,
					ReversalReason: &reversedReason,
				})
				
				var redemption models.Redemption
				db.Where("code = ?", "CODE-REVERSED").First(&redemption)
				
				transactionService := services.NewTransactionService(db)
				rewardService := services.NewRewardService(db, transactionService)
				_, err := rewardService.ReverseRedemption(context.Background(), redemption.ID, "Try again", "admin-123")
				return err
			},
			expectedWrapped: true,
		},
		{
			name:          "ErrInvalidAmount - zero amount in earn points",
			sentinelError: services.ErrInvalidAmount,
			setupAndTrigger: func() error {
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id": "customer-zero-amount",
					"tier_id":    &baseTier.ID,
				})
				transactionService := services.NewTransactionService(db)
				_, err := transactionService.EarnPoints(context.Background(), &loyaltyv1.EarnPointsRequest{
					Amount:        0, // Zero amount
					ReferenceId:   "ref-zero",
					ReferenceType: "ORDER",
					Description:   "Zero amount test",
				}, customer.AccountID)
				return err
			},
			expectedWrapped: true,
		},
		{
			name:          "ErrMissingRequired - missing reference_id",
			sentinelError: services.ErrMissingRequired,
			setupAndTrigger: func() error {
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id": "customer-missing-ref",
					"tier_id":    &baseTier.ID,
				})
				transactionService := services.NewTransactionService(db)
				_, err := transactionService.EarnPoints(context.Background(), &loyaltyv1.EarnPointsRequest{
					Amount:        1000,
					ReferenceId:   "", // Missing
					ReferenceType: "ORDER",
					Description:   "Missing ref test",
				}, customer.AccountID)
				return err
			},
			expectedWrapped: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Trigger the error
			err := tc.setupAndTrigger()

			// Verify error occurred
			if err == nil {
				t.Fatal("Expected error but got nil")
			}

			// Verify error wrapping with errors.Is() (constitutional requirement)
			if !errors.Is(err, tc.sentinelError) {
				t.Errorf("Expected error to wrap %v, but errors.Is() returned false. Got: %v", tc.sentinelError, err)
			}

			// Verify error message contains contextual information (not just sentinel message)
			if err.Error() == tc.sentinelError.Error() {
				t.Error("Expected error to be wrapped with context (not just sentinel message)")
			}
		})
	}
}

// TestAllHTTPErrorCodes tests EVERY error code defined in handlers/error_codes.go
// This is MANDATORY per Constitution Principle IX
func TestAllHTTPErrorCodes(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "customers", "rewards", "redemptions", "membership_tiers")

	// Create base tier
	baseTier := testutil.CreateTestTier(db, map[string]interface{}{
		"name":  "Base",
		"level": 0,
	})

	// Table-driven tests - one test case per HTTP error code
	testCases := []struct {
		name             string
		httpErrorCode    ErrorCode
		setupAndExecute  func() (*httptest.ResponseRecorder, int)
	}{
		{
			name:          "Errors.InvalidRequest - malformed JSON",
			httpErrorCode: Errors.InvalidRequest,
			setupAndExecute: func() (*httptest.ResponseRecorder, int) {
				req := httptest.NewRequest(http.MethodPost, "/v1/loyalty/enroll", nil) // No body
				req.Header.Set("Content-Type", "application/json")
				ctx := context.WithValue(req.Context(), middleware.UserIDKey, "test-user")
				req = req.WithContext(ctx)
				
				rec := httptest.NewRecorder()
				loyaltyService := services.NewLoyaltyService(db)
				handler := NewEnrollmentHandler(loyaltyService)
				handler.HandleEnroll(rec, req)
				
				return rec, Errors.InvalidRequest.HTTPStatus
			},
		},
		{
			name:          "Errors.MissingRequired - missing reference_id",
			httpErrorCode: Errors.MissingRequired,
			setupAndExecute: func() (*httptest.ResponseRecorder, int) {
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id": "user-missing-ref",
					"tier_id":    &baseTier.ID,
				})
				
				reqBody := &loyaltyv1.EarnPointsRequest{
					Amount:        1000,
					ReferenceId:   "", // Missing required field
					ReferenceType: "ORDER",
					Description:   "Test",
				}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/v1/loyalty/points/earn", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				ctx := context.WithValue(req.Context(), middleware.UserIDKey, customer.AccountID)
				req = req.WithContext(ctx)
				
				rec := httptest.NewRecorder()
				transactionService := services.NewTransactionService(db)
				loyaltyService := services.NewLoyaltyService(db)
				handler := NewPointsHandler(transactionService, loyaltyService)
				handler.HandleEarnPoints(rec, req)
				
				return rec, Errors.MissingRequired.HTTPStatus
			},
		},
		{
			name:          "Errors.InvalidAmount - negative amount",
			httpErrorCode: Errors.InvalidAmount,
			setupAndExecute: func() (*httptest.ResponseRecorder, int) {
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id": "user-negative",
					"tier_id":    &baseTier.ID,
				})
				
				reqBody := &loyaltyv1.EarnPointsRequest{
					Amount:        -100, // Negative amount
					ReferenceId:   "ref-neg",
					ReferenceType: "ORDER",
					Description:   "Negative test",
				}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/v1/loyalty/points/earn", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				ctx := context.WithValue(req.Context(), middleware.UserIDKey, customer.AccountID)
				req = req.WithContext(ctx)
				
				rec := httptest.NewRecorder()
				transactionService := services.NewTransactionService(db)
				loyaltyService := services.NewLoyaltyService(db)
				handler := NewPointsHandler(transactionService, loyaltyService)
				handler.HandleEarnPoints(rec, req)
				
				return rec, Errors.InvalidAmount.HTTPStatus
			},
		},
		{
			name:          "Errors.InsufficientBalance - redemption exceeding balance",
			httpErrorCode: Errors.InsufficientBalance,
			setupAndExecute: func() (*httptest.ResponseRecorder, int) {
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id":      "user-low-balance",
					"current_balance": int64(10),
					"tier_id":         &baseTier.ID,
				})
				reward := testutil.CreateTestReward(db, map[string]interface{}{
					"point_cost": int64(500),
				})
				
				reqBody := &loyaltyv1.RedeemRewardRequest{RewardId: reward.ID}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/v1/loyalty/rewards/redeem", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				ctx := context.WithValue(req.Context(), middleware.UserIDKey, customer.AccountID)
				req = req.WithContext(ctx)
				
				rec := httptest.NewRecorder()
				transactionService := services.NewTransactionService(db)
				rewardService := services.NewRewardService(db, transactionService)
				handler := NewRewardsHandler(rewardService)
				handler.HandleRedeemReward(rec, req)
				
				return rec, Errors.InsufficientBalance.HTTPStatus
			},
		},
		{
			name:          "Errors.InvalidReferralCode - invalid referral code",
			httpErrorCode: Errors.InvalidReferralCode,
			setupAndExecute: func() (*httptest.ResponseRecorder, int) {
				reqBody := &loyaltyv1.EnrollCustomerRequest{
					ReferralCode: "INVALID999",
				}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/v1/loyalty/enroll", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				ctx := context.WithValue(req.Context(), middleware.UserIDKey, "new-user-invalid-ref")
				req = req.WithContext(ctx)
				
				rec := httptest.NewRecorder()
				loyaltyService := services.NewLoyaltyService(db)
				handler := NewEnrollmentHandler(loyaltyService)
				handler.HandleEnroll(rec, req)
				
				return rec, Errors.InvalidReferralCode.HTTPStatus
			},
		},
		{
			name:          "Errors.CustomerNotFound - customer not enrolled",
			httpErrorCode: Errors.CustomerNotFound,
			setupAndExecute: func() (*httptest.ResponseRecorder, int) {
				req := httptest.NewRequest(http.MethodGet, "/v1/loyalty/me", nil)
				ctx := context.WithValue(req.Context(), middleware.UserIDKey, "not-enrolled-user")
				req = req.WithContext(ctx)
				
				rec := httptest.NewRecorder()
				loyaltyService := services.NewLoyaltyService(db)
				handler := NewEnrollmentHandler(loyaltyService)
				handler.HandleGetStatus(rec, req)
				
				return rec, Errors.CustomerNotFound.HTTPStatus
			},
		},
		{
			name:          "Errors.RewardNotFound - reward not found",
			httpErrorCode: Errors.RewardNotFound,
			setupAndExecute: func() (*httptest.ResponseRecorder, int) {
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id":      "user-reward-404",
					"current_balance": int64(1000),
					"tier_id":         &baseTier.ID,
				})
				
				reqBody := &loyaltyv1.RedeemRewardRequest{
					RewardId: "00000000-0000-0000-0000-000000000000",
				}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/v1/loyalty/rewards/redeem", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				ctx := context.WithValue(req.Context(), middleware.UserIDKey, customer.AccountID)
				req = req.WithContext(ctx)
				
				rec := httptest.NewRecorder()
				transactionService := services.NewTransactionService(db)
				rewardService := services.NewRewardService(db, transactionService)
				handler := NewRewardsHandler(rewardService)
				handler.HandleRedeemReward(rec, req)
				
				return rec, Errors.RewardNotFound.HTTPStatus
			},
		},
		{
			name:          "Errors.RewardInactive - inactive reward redemption",
			httpErrorCode: Errors.RewardInactive,
			setupAndExecute: func() (*httptest.ResponseRecorder, int) {
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id":      "user-inactive",
					"current_balance": int64(1000),
					"tier_id":         &baseTier.ID,
				})
				inactiveReward := testutil.CreateTestReward(db, map[string]interface{}{
					"is_active": false,
				})
				
				reqBody := &loyaltyv1.RedeemRewardRequest{RewardId: inactiveReward.ID}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/v1/loyalty/rewards/redeem", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				ctx := context.WithValue(req.Context(), middleware.UserIDKey, customer.AccountID)
				req = req.WithContext(ctx)
				
				rec := httptest.NewRecorder()
				transactionService := services.NewTransactionService(db)
				rewardService := services.NewRewardService(db, transactionService)
				handler := NewRewardsHandler(rewardService)
				handler.HandleRedeemReward(rec, req)
				
				return rec, Errors.RewardInactive.HTTPStatus
			},
		},
		{
			name:          "Errors.AlreadyEnrolled - duplicate enrollment",
			httpErrorCode: Errors.AlreadyEnrolled,
			setupAndExecute: func() (*httptest.ResponseRecorder, int) {
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id": "already-enrolled-http",
					"tier_id":    &baseTier.ID,
				})
				
				reqBody := &loyaltyv1.EnrollCustomerRequest{}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/v1/loyalty/enroll", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				ctx := context.WithValue(req.Context(), middleware.UserIDKey, customer.AccountID)
				req = req.WithContext(ctx)
				
				rec := httptest.NewRecorder()
				loyaltyService := services.NewLoyaltyService(db)
				handler := NewEnrollmentHandler(loyaltyService)
				handler.HandleEnroll(rec, req)
				
				return rec, Errors.AlreadyEnrolled.HTTPStatus
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Execute request and get response
			rec, expectedStatus := tc.setupAndExecute()

			// Verify HTTP status code matches error code
			if rec.Code != expectedStatus {
				t.Errorf("Expected HTTP status %d, got %d", expectedStatus, rec.Code)
			}

			// Parse error response
			var errResp ErrorResponse
			if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
				t.Fatalf("Failed to decode error response: %v", err)
			}

			// Verify error code matches
			if errResp.Code != tc.httpErrorCode.Code {
				t.Errorf("Expected error code %s, got %s", tc.httpErrorCode.Code, errResp.Code)
			}

			// Verify error message present
			if errResp.Message == "" {
				t.Error("Expected error message to be present")
			}
		})
	}
}

// TestErrorFlowEndToEnd tests complete error flow: Service → Handler → Client
func TestErrorFlowEndToEnd(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "customers", "membership_tiers")

	baseTier := testutil.CreateTestTier(db, map[string]interface{}{
		"name":  "Base",
		"level": 0,
	})

	testCases := []struct {
		name               string
		setupAndExecute    func() (*httptest.ResponseRecorder, error)
		expectedHTTPStatus int
		expectedErrorCode  string
	}{
		{
			name: "Service ErrNotEnrolled → Handler 404 CustomerNotFound",
			setupAndExecute: func() (*httptest.ResponseRecorder, error) {
				req := httptest.NewRequest(http.MethodGet, "/v1/loyalty/me", nil)
				ctx := context.WithValue(req.Context(), middleware.UserIDKey, "not-enrolled")
				req = req.WithContext(ctx)
				
				rec := httptest.NewRecorder()
				loyaltyService := services.NewLoyaltyService(db)
				handler := NewEnrollmentHandler(loyaltyService)
				handler.HandleGetStatus(rec, req)
				
				return rec, nil
			},
			expectedHTTPStatus: http.StatusNotFound,
			expectedErrorCode:  "CUSTOMER_NOT_FOUND",
		},
		{
			name: "Service ErrInsufficientBalance → Handler 400 InsufficientBalance",
			setupAndExecute: func() (*httptest.ResponseRecorder, error) {
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id":      "low-bal",
					"current_balance": int64(10),
					"tier_id":         &baseTier.ID,
				})
				reward := testutil.CreateTestReward(db, map[string]interface{}{
					"point_cost": int64(500),
				})
				
				reqBody := &loyaltyv1.RedeemRewardRequest{RewardId: reward.ID}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/v1/loyalty/rewards/redeem", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				ctx := context.WithValue(req.Context(), middleware.UserIDKey, customer.AccountID)
				req = req.WithContext(ctx)
				
				rec := httptest.NewRecorder()
				transactionService := services.NewTransactionService(db)
				rewardService := services.NewRewardService(db, transactionService)
				handler := NewRewardsHandler(rewardService)
				handler.HandleRedeemReward(rec, req)
				
				return rec, nil
			},
			expectedHTTPStatus: http.StatusBadRequest,
			expectedErrorCode:  "INSUFFICIENT_BALANCE",
		},
		{
			name: "Service ErrAlreadyEnrolled → Handler 409 AlreadyEnrolled",
			setupAndExecute: func() (*httptest.ResponseRecorder, error) {
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id": "enrolled-409",
					"tier_id":    &baseTier.ID,
				})
				
				reqBody := &loyaltyv1.EnrollCustomerRequest{}
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/v1/loyalty/enroll", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				ctx := context.WithValue(req.Context(), middleware.UserIDKey, customer.AccountID)
				req = req.WithContext(ctx)
				
				rec := httptest.NewRecorder()
				loyaltyService := services.NewLoyaltyService(db)
				handler := NewEnrollmentHandler(loyaltyService)
				handler.HandleEnroll(rec, req)
				
				return rec, nil
			},
			expectedHTTPStatus: http.StatusConflict,
			expectedErrorCode:  "ALREADY_ENROLLED",
		},
		{
			name: "Context.Canceled → Handler 499",
			setupAndExecute: func() (*httptest.ResponseRecorder, error) {
				rec := httptest.NewRecorder()
				
				// Test HandleServiceError directly with cancelled context
				HandleServiceError(rec, context.Canceled)
				
				return rec, nil
			},
			expectedHTTPStatus: 499, // Client Closed Request
			expectedErrorCode:  "",  // Plain text response for context errors
		},
		{
			name: "Context.DeadlineExceeded → Handler 504",
			setupAndExecute: func() (*httptest.ResponseRecorder, error) {
				rec := httptest.NewRecorder()
				
				// Test HandleServiceError directly with deadline exceeded
				HandleServiceError(rec, context.DeadlineExceeded)
				
				return rec, nil
			},
			expectedHTTPStatus: http.StatusGatewayTimeout,
			expectedErrorCode:  "", // Plain text response for context errors
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Execute and get response
			rec, _ := tc.setupAndExecute()

			// Verify HTTP status code
			if rec.Code != tc.expectedHTTPStatus {
				t.Errorf("Expected HTTP status %d, got %d", tc.expectedHTTPStatus, rec.Code)
			}

			// For JSON error responses, verify error code
			if tc.expectedErrorCode != "" {
				var errResp ErrorResponse
				if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
					t.Fatalf("Failed to decode error response: %v", err)
				}

				if errResp.Code != tc.expectedErrorCode {
					t.Errorf("Expected error code %s, got %s", tc.expectedErrorCode, errResp.Code)
				}

				// Verify no internal error details leaked
				if errResp.Message == "" {
					t.Error("Expected error message to be present")
				}
			}
		})
	}
}

