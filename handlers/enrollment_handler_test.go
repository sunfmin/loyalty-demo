package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/yourorg/loyalty-demo/internal/middleware"
	"github.com/yourorg/loyalty-demo/internal/testutil"
	"github.com/yourorg/loyalty-demo/services"
	"google.golang.org/protobuf/testing/protocmp"
	loyaltyv1 "github.com/yourorg/loyalty-demo/api/v1"
)

func TestEnrollCustomer(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "customers", "membership_tiers")

	// Create base tier for enrollment
	baseTier := testutil.CreateTestTier(db, map[string]interface{}{
		"name":                "Base",
		"level":               0,
		"qualification_points": int64(0),
		"earn_rate_multiplier": 1.0,
	})

	// Table-driven test cases
	testCases := []struct {
		name           string
		accountID      string
		request        *loyaltyv1.EnrollCustomerRequest
		setupFixtures  func()
		expectedStatus int
		expectedError  string
		validateResponse func(t *testing.T, resp *loyaltyv1.EnrollCustomerResponse)
	}{
		{
			name:      "Happy path: Valid enrollment without referral code",
			accountID: "user-001",
			request:   &loyaltyv1.EnrollCustomerRequest{},
			setupFixtures: func() {},
			expectedStatus: http.StatusCreated,
			validateResponse: func(t *testing.T, resp *loyaltyv1.EnrollCustomerResponse) {
				if resp.Customer == nil {
					t.Fatal("Expected customer in response")
				}
				if resp.Customer.AccountId != "user-001" {
					t.Errorf("Expected account_id user-001, got %s", resp.Customer.AccountId)
				}
				if resp.Customer.MembershipNumber == "" {
					t.Error("Expected membership_number to be generated")
				}
				if resp.Customer.ReferralCode == "" {
					t.Error("Expected referral_code to be generated")
				}
				if resp.Customer.CurrentBalance != 0 {
					t.Errorf("Expected initial balance 0, got %d", resp.Customer.CurrentBalance)
				}
				if resp.Customer.Tier == nil || resp.Customer.Tier.Name != "Base" {
					t.Error("Expected Base tier assigned")
				}
			},
		},
		{
			name:      "Happy path: Valid enrollment with referral code",
			accountID: "user-002",
			request: &loyaltyv1.EnrollCustomerRequest{
				ReferralCode: "",
			},
			setupFixtures: func() {
				// Create existing customer with referral code
				referrer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id":    "referrer-001",
					"referral_code": "FRIEND123",
					"tier_id":       &baseTier.ID,
				})
				// Store referral code for test
				testCases[1].request.ReferralCode = referrer.ReferralCode
			},
			expectedStatus: http.StatusCreated,
			validateResponse: func(t *testing.T, resp *loyaltyv1.EnrollCustomerResponse) {
				if resp.Customer == nil {
					t.Fatal("Expected customer in response")
				}
				if resp.Customer.ReferredBy != "FRIEND123" {
					t.Errorf("Expected referred_by to be FRIEND123, got %s", resp.Customer.ReferredBy)
				}
			},
		},
		{
			name:      "Edge case: Already enrolled",
			accountID: "user-003",
			request:   &loyaltyv1.EnrollCustomerRequest{},
			setupFixtures: func() {
				// Create customer that's already enrolled
				testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id": "user-003",
					"tier_id":    &baseTier.ID,
				})
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "ALREADY_ENROLLED",
		},
		{
			name:      "Edge case: Invalid referral code",
			accountID: "user-004",
			request: &loyaltyv1.EnrollCustomerRequest{
				ReferralCode: "INVALID999",
			},
			setupFixtures:  func() {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_REQUEST",
		},
		{
			name:           "Edge case: Missing authentication",
			accountID:      "",
			request:        &loyaltyv1.EnrollCustomerRequest{},
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

			req := httptest.NewRequest(http.MethodPost, "/v1/loyalty/enroll", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Add authentication to context (mock)
			if tc.accountID != "" {
				ctx := context.WithValue(req.Context(), middleware.UserIDKey, tc.accountID)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()

			// Create service and handler
			loyaltyService := services.NewLoyaltyService(db)
			handler := NewEnrollmentHandler(loyaltyService)

			// Call handler
			handler.HandleEnroll(rec, req)

			// Verify response status
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, rec.Code)
			}

			// For success cases, validate response
			if tc.expectedStatus == http.StatusCreated && tc.validateResponse != nil {
				var resp loyaltyv1.EnrollCustomerResponse
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

func TestGetCustomerStatus(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "customers", "point_transactions", "membership_tiers")

	// Create base tier
	baseTier := testutil.CreateTestTier(db, map[string]interface{}{
		"name":                "Base",
		"level":               0,
		"qualification_points": int64(0),
		"earn_rate_multiplier": 1.0,
	})

	// Table-driven test cases
	testCases := []struct {
		name           string
		accountID      string
		setupFixtures  func()
		expectedStatus int
		expectedError  string
		validateResponse func(t *testing.T, resp *loyaltyv1.GetCustomerResponse)
	}{
		{
			name:      "Happy path: Enrolled customer gets status",
			accountID: "user-enrolled",
			setupFixtures: func() {
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id":      "user-enrolled",
					"current_balance": int64(500),
					"tier_id":         &baseTier.ID,
				})
				// Create some transactions
				testutil.CreateTestTransaction(db, customer.ID, map[string]interface{}{
					"amount": int64(500),
				})
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *loyaltyv1.GetCustomerResponse) {
				if resp.Customer == nil {
					t.Fatal("Expected customer in response")
				}
				if resp.Customer.CurrentBalance != 500 {
					t.Errorf("Expected balance 500, got %d", resp.Customer.CurrentBalance)
				}
				if resp.Customer.Tier == nil || resp.Customer.Tier.Name != "Base" {
					t.Error("Expected Base tier")
				}
			},
		},
		{
			name:           "Edge case: Not enrolled",
			accountID:      "user-not-enrolled",
			setupFixtures:  func() {},
			expectedStatus: http.StatusNotFound,
			expectedError:  "CUSTOMER_NOT_FOUND",
		},
		{
			name:           "Edge case: Missing authentication",
			accountID:      "",
			setupFixtures:  func() {},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup fixtures for this test case
			tc.setupFixtures()

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/v1/loyalty/me", nil)

			// Add authentication to context (mock)
			if tc.accountID != "" {
				ctx := context.WithValue(req.Context(), middleware.UserIDKey, tc.accountID)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()

			// Create service and handler
			loyaltyService := services.NewLoyaltyService(db)
			handler := NewEnrollmentHandler(loyaltyService)

			// Call handler
			handler.HandleGetStatus(rec, req)

			// Verify response status
			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, rec.Code)
			}

			// For success cases, validate response
			if tc.expectedStatus == http.StatusOK && tc.validateResponse != nil {
				var resp loyaltyv1.GetCustomerResponse
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

