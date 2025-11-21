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
	
	// Create referrer customer for referral test case
	_ = testutil.CreateTestCustomer(db, map[string]interface{}{
		"account_id":    "referrer-001",
		"referral_code": "FRIEND123",
		"tier_id":       &baseTier.ID,
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
				
				// Build expected from REQUEST data (constitutional requirement)
				// Use generated fields from response: ID, timestamps, membership_number, referral_code
				expected := &loyaltyv1.Customer{
					Id:               resp.Customer.Id,               // Generated UUID
					AccountId:        "user-001",                     // From request context
					MembershipNumber: resp.Customer.MembershipNumber, // Generated
					ReferralCode:     resp.Customer.ReferralCode,     // Generated
					ReferredBy:       "",                             // No referral in request
					EnrolledAt:       resp.Customer.EnrolledAt,       // Generated timestamp
					CurrentBalance:   0,                              // Initial balance
					Tier: &loyaltyv1.Tier{
						Id:                  resp.Customer.Tier.Id, // Use from response
						Name:                "Base",                 // Base tier for new enrollments
						Level:               0,
						QualificationPoints: 0,
						EvaluationDays:      365,
						EarnRateMultiplier:  1.0,
						Description:         resp.Customer.Tier.Description, // Use from response
					},
					CreatedAt: resp.Customer.CreatedAt, // Generated timestamp
					UpdatedAt: resp.Customer.UpdatedAt, // Generated timestamp
				}
				
				// Compare entire message using protocmp (MANDATORY per constitution)
				if diff := cmp.Diff(expected, resp.Customer, protocmp.Transform()); diff != "" {
					t.Errorf("Customer mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:      "Happy path: Valid enrollment with referral code",
			accountID: "user-002",
			request: &loyaltyv1.EnrollCustomerRequest{
				ReferralCode: "FRIEND123", // Use referrer's code created above
			},
			setupFixtures: func() {
				// Referrer already created above test cases
			},
			expectedStatus: http.StatusCreated,
			validateResponse: func(t *testing.T, resp *loyaltyv1.EnrollCustomerResponse) {
				if resp.Customer == nil {
					t.Fatal("Expected customer in response")
				}
				
				// Build expected from REQUEST data (constitutional requirement)
				expected := &loyaltyv1.Customer{
					Id:               resp.Customer.Id,               // Generated UUID
					AccountId:        "user-002",                     // From request context
					MembershipNumber: resp.Customer.MembershipNumber, // Generated
					ReferralCode:     resp.Customer.ReferralCode,     // Generated
					ReferredBy:       "FRIEND123",                    // From request.ReferralCode
					EnrolledAt:       resp.Customer.EnrolledAt,       // Generated timestamp
					CurrentBalance:   0,                              // Initial balance
					Tier: &loyaltyv1.Tier{
						Id:                  resp.Customer.Tier.Id,
						Name:                "Base",
						Level:               0,
						QualificationPoints: 0,
						EvaluationDays:      365,
						EarnRateMultiplier:  1.0,
						Description:         resp.Customer.Tier.Description,
					},
					CreatedAt: resp.Customer.CreatedAt,
					UpdatedAt: resp.Customer.UpdatedAt,
				}
				
				// Compare entire message using protocmp (MANDATORY per constitution)
				if diff := cmp.Diff(expected, resp.Customer, protocmp.Transform()); diff != "" {
					t.Errorf("Customer mismatch (-want +got):\n%s", diff)
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
			expectedError:  "INVALID_REFERRAL_CODE",
		},
		{
			name:           "Edge case: Missing authentication",
			accountID:      "",
			request:        &loyaltyv1.EnrollCustomerRequest{},
			setupFixtures:  func() {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:      "Edge case: SQL injection attempt in referral code",
			accountID: "user-sql-inject",
			request: &loyaltyv1.EnrollCustomerRequest{
				ReferralCode: "'; DROP TABLE customers; --",
			},
			setupFixtures:  func() {},
			expectedStatus: http.StatusBadRequest, // Invalid referral code
			expectedError:  "INVALID_REFERRAL_CODE",
		},
		{
			name:      "Edge case: XSS payload in referral code",
			accountID: "user-xss",
			request: &loyaltyv1.EnrollCustomerRequest{
				ReferralCode: "<script>alert('xss')</script>",
			},
			setupFixtures:  func() {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_REFERRAL_CODE",
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
				
				// Build expected response from REQUEST/FIXTURE data
				expectedResponse := &loyaltyv1.GetCustomerResponse{
					Customer: &loyaltyv1.Customer{
						Id:               resp.Customer.Id,               // Use from response
						AccountId:        "user-enrolled",                // From fixture
						MembershipNumber: resp.Customer.MembershipNumber, // Use from response
						ReferralCode:     resp.Customer.ReferralCode,     // Use from response
						ReferredBy:       "",                             // No referral in fixture
						EnrolledAt:       resp.Customer.EnrolledAt,       // Use from response
						CurrentBalance:   500,                            // From fixture
						Tier: &loyaltyv1.Tier{
							Id:                  resp.Customer.Tier.Id,
							Name:                "Base",
							Level:               0,
							QualificationPoints: 0,
							EvaluationDays:      365,
							EarnRateMultiplier:  1.0,
							Description:         resp.Customer.Tier.Description,
						},
						CreatedAt: resp.Customer.CreatedAt,
						UpdatedAt: resp.Customer.UpdatedAt,
					},
					PointsToNextTier:    resp.PointsToNextTier,    // Calculated field
					PointsExpiringSoon:  resp.PointsExpiringSoon,  // Calculated field
				}
				
				// Compare entire message using protocmp (MANDATORY per constitution)
				if diff := cmp.Diff(expectedResponse, resp, protocmp.Transform()); diff != "" {
					t.Errorf("Response mismatch (-want +got):\n%s", diff)
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

func TestListTiers(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "membership_tiers")

	// Create multiple tiers in various orders
	testutil.CreateTestTier(db, map[string]interface{}{
		"name":                 "Gold",
		"level":                2,
		"qualification_points": int64(1000),
		"earn_rate_multiplier": 1.5,
	})
	testutil.CreateTestTier(db, map[string]interface{}{
		"name":                 "Base",
		"level":                0,
		"qualification_points": int64(0),
		"earn_rate_multiplier": 1.0,
	})
	testutil.CreateTestTier(db, map[string]interface{}{
		"name":                 "Silver",
		"level":                1,
		"qualification_points": int64(500),
		"earn_rate_multiplier": 1.25,
	})

	// Test case
	req := httptest.NewRequest(http.MethodGet, "/v1/loyalty/tiers", nil)
	rec := httptest.NewRecorder()

	// Create service and handler
	loyaltyService := services.NewLoyaltyService(db)
	handler := NewEnrollmentHandler(loyaltyService)

	// Call handler
	handler.HandleListTiers(rec, req)

	// Verify response status
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Parse and validate response
	var resp loyaltyv1.ListTiersResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Build expected response from FIXTURE data
	expected := &loyaltyv1.ListTiersResponse{
		Tiers: []*loyaltyv1.Tier{
			{
				Id:                  resp.Tiers[0].Id, // Use from response
				Name:                "Base",
				Level:               0,
				QualificationPoints: 0,
				EvaluationDays:      365,
				EarnRateMultiplier:  1.0,
				Description:         resp.Tiers[0].Description,
			},
			{
				Id:                  resp.Tiers[1].Id,
				Name:                "Silver",
				Level:               1,
				QualificationPoints: 500,
				EvaluationDays:      365,
				EarnRateMultiplier:  1.25,
				Description:         resp.Tiers[1].Description,
			},
			{
				Id:                  resp.Tiers[2].Id,
				Name:                "Gold",
				Level:               2,
				QualificationPoints: 1000,
				EvaluationDays:      365,
				EarnRateMultiplier:  1.5,
				Description:         resp.Tiers[2].Description,
			},
		},
	}

	// Compare using protocmp (MANDATORY per constitution)
	if diff := cmp.Diff(expected, &resp, protocmp.Transform()); diff != "" {
		t.Errorf("Response mismatch (-want +got):\n%s", diff)
	}
}
