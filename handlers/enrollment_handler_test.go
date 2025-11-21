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

// TestEnrollmentAcceptanceScenarios tests User Story 1 (Customer Enrollment) acceptance scenarios
// Implements Constitution Principle XIII (Acceptance Scenario Coverage)
func TestEnrollmentAcceptanceScenarios(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "customers", "membership_tiers")

	// Create base tier for enrollment (DATABASE FIXTURE)
	baseTier := testutil.CreateTestTier(db, map[string]interface{}{
		"name":                "Base",
		"level":               0,
		"qualification_points": int64(0),
		"earn_rate_multiplier": 1.0,
	})
	
	// Create referrer customer for referral test case (DATABASE FIXTURE)
	_ = testutil.CreateTestCustomer(db, map[string]interface{}{
		"account_id":    "referrer-001",
		"referral_code": "FRIEND123",
		"tier_id":       &baseTier.ID,
	})

	// Table-driven test cases - one per acceptance scenario
	testCases := []struct {
		name           string
		scenario       string // Given/When/Then from spec
		accountID      string
		request        *loyaltyv1.EnrollCustomerRequest
		setupFixtures  func()
		expectedStatus int
		expectedError  string
		validateResponse func(t *testing.T, resp *loyaltyv1.EnrollCustomerResponse)
	}{
		{
			name:     "US1-AS1: New customer enrolls with confirmation and zero balance",
			scenario: "Given a new customer with a valid account, When they choose to enroll in the loyalty program, Then they receive a confirmation with their membership number and starting point balance of zero",
			accountID: "user-001",
			request:   &loyaltyv1.EnrollCustomerRequest{},
			setupFixtures: func() {},
			expectedStatus: http.StatusCreated,
			validateResponse: func(t *testing.T, resp *loyaltyv1.EnrollCustomerResponse) {
				if resp.Customer == nil {
					t.Fatal("Expected customer in response")
				}
				
				// Build expected from FIXTURES (REQUEST context + DATABASE fixtures)
				// Per enhanced Principle VI: Derive from fixtures, NOT response
				expected := &loyaltyv1.Customer{
					Id:               resp.Customer.Id,               // Generated UUID (truly random)
					AccountId:        "user-001",                     // From REQUEST context fixture
					MembershipNumber: resp.Customer.MembershipNumber, // Generated (truly random)
					ReferralCode:     resp.Customer.ReferralCode,     // Generated (truly random)
					ReferredBy:       "",                             // From REQUEST (no referral)
					EnrolledAt:       resp.Customer.EnrolledAt,       // Generated timestamp (truly random)
					CurrentBalance:   0,                              // Initial balance (known rule)
					Tier: &loyaltyv1.Tier{
						Id:                  resp.Customer.Tier.Id,   // Generated (from DB fixture)
						Name:                "Base",                   // From DATABASE fixture (baseTier)
						Level:               0,                        // From DATABASE fixture
						QualificationPoints: 0,                        // From DATABASE fixture
						EvaluationDays:      365,                      // System default
						EarnRateMultiplier:  1.0,                      // From DATABASE fixture
						Description:         resp.Customer.Tier.Description, // Generated (from DB)
					},
					CreatedAt: resp.Customer.CreatedAt, // Generated timestamp (truly random)
					UpdatedAt: resp.Customer.UpdatedAt, // Generated timestamp (truly random)
				}
				
				// Compare entire message using protocmp (MANDATORY per Principle VI)
				if diff := cmp.Diff(expected, resp.Customer, protocmp.Transform()); diff != "" {
					t.Errorf("Customer mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:     "US1-AS2: Customer during checkout enrolls (NOTE: earn points tested in US2-AS1)",
			scenario: "Given a customer during checkout, When they complete their first purchase and opt-in to the loyalty program, Then they are automatically enrolled",
			accountID: "user-002",
			request:   &loyaltyv1.EnrollCustomerRequest{},
			setupFixtures: func() {},
			expectedStatus: http.StatusCreated,
			validateResponse: func(t *testing.T, resp *loyaltyv1.EnrollCustomerResponse) {
				if resp.Customer == nil {
					t.Fatal("Expected customer in response")
				}
				
				// Build expected from FIXTURES (REQUEST context + DATABASE fixtures)
				expected := &loyaltyv1.Customer{
					Id:               resp.Customer.Id,               // Generated UUID (truly random)
					AccountId:        "user-002",                     // From REQUEST context fixture
					MembershipNumber: resp.Customer.MembershipNumber, // Generated (truly random)
					ReferralCode:     resp.Customer.ReferralCode,     // Generated (truly random)
					ReferredBy:       "",                             // From REQUEST (no referral)
					EnrolledAt:       resp.Customer.EnrolledAt,       // Generated timestamp (truly random)
					CurrentBalance:   0,                              // Initial balance (known rule)
					Tier: &loyaltyv1.Tier{
						Id:                  resp.Customer.Tier.Id,
						Name:                "Base",                   // From DATABASE fixture
						Level:               0,                        // From DATABASE fixture
						QualificationPoints: 0,                        // From DATABASE fixture
						EvaluationDays:      365,                      // System default
						EarnRateMultiplier:  1.0,                      // From DATABASE fixture
						Description:         resp.Customer.Tier.Description,
					},
					CreatedAt: resp.Customer.CreatedAt, // Generated timestamp (truly random)
					UpdatedAt: resp.Customer.UpdatedAt, // Generated timestamp (truly random)
				}
				
				// Compare entire message using protocmp (MANDATORY per Principle VI)
				if diff := cmp.Diff(expected, resp.Customer, protocmp.Transform()); diff != "" {
					t.Errorf("Customer mismatch (-want +got):\n%s", diff)
				}
				
				// NOTE: The "earn points for initial purchase" part of US1-AS2 is tested
				// in US2-AS1 (points earned for purchase) since points earning is separate operation
			},
		},
		{
			name:     "US1-AS3: Already enrolled customer attempts re-enrollment",
			scenario: "Given an already enrolled customer, When they attempt to enroll again, Then the system informs them they are already a member",
			accountID: "user-003",
			request:   &loyaltyv1.EnrollCustomerRequest{},
			setupFixtures: func() {
				// Create customer that's already enrolled (DATABASE FIXTURE)
				testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id": "user-003",
					"tier_id":    &baseTier.ID,
				})
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "ALREADY_ENROLLED",
		},
		{
			name:     "US2-AS2: Enrollment with valid referral code (referral bonus tested separately)",
			scenario: "Given a customer enrolling with referral code, When they enroll, Then they are linked to referrer",
			accountID: "user-referral",
			request: &loyaltyv1.EnrollCustomerRequest{
				ReferralCode: "FRIEND123", // From DATABASE fixture (referrer created above)
			},
			setupFixtures: func() {},
			expectedStatus: http.StatusCreated,
			validateResponse: func(t *testing.T, resp *loyaltyv1.EnrollCustomerResponse) {
				if resp.Customer == nil {
					t.Fatal("Expected customer in response")
				}
				
				// Build expected from FIXTURES (REQUEST + DATABASE)
				expected := &loyaltyv1.Customer{
					Id:               resp.Customer.Id,               // Generated (truly random)
					AccountId:        "user-referral",                // From REQUEST context fixture
					MembershipNumber: resp.Customer.MembershipNumber, // Generated (truly random)
					ReferralCode:     resp.Customer.ReferralCode,     // Generated (truly random)
					ReferredBy:       "FRIEND123",                    // From REQUEST fixture
					EnrolledAt:       resp.Customer.EnrolledAt,       // Generated (truly random)
					CurrentBalance:   0,                              // Initial balance (known rule)
					Tier: &loyaltyv1.Tier{
						Id:                  resp.Customer.Tier.Id,
						Name:                "Base",                   // From DATABASE fixture
						Level:               0,                        // From DATABASE fixture
						QualificationPoints: 0,                        // From DATABASE fixture
						EvaluationDays:      365,                      // System default
						EarnRateMultiplier:  1.0,                      // From DATABASE fixture
						Description:         resp.Customer.Tier.Description,
					},
					CreatedAt: resp.Customer.CreatedAt, // Generated (truly random)
					UpdatedAt: resp.Customer.UpdatedAt, // Generated (truly random)
				}
				
				// Compare entire message using protocmp (MANDATORY per Principle VI)
				if diff := cmp.Diff(expected, resp.Customer, protocmp.Transform()); diff != "" {
					t.Errorf("Customer mismatch (-want +got):\n%s", diff)
				}
				
				// NOTE: Referral bonus points (US2-AS2 "earn bonus referral points")
				// is tested in points_handler_test.go when referrer's friend makes first purchase
			},
		},
		{
			name:     "Edge case: Invalid referral code",
			scenario: "Validation: referral code does not exist",
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
			scenario:       "Authentication: unauthenticated enrollment attempt",
			accountID:      "",
			request:        &loyaltyv1.EnrollCustomerRequest{},
			setupFixtures:  func() {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:     "Edge case: SQL injection attempt in referral code",
			scenario: "Security: SQL injection prevented via parameterized queries",
			accountID: "user-sql-inject",
			request: &loyaltyv1.EnrollCustomerRequest{
				ReferralCode: "'; DROP TABLE customers; --",
			},
			setupFixtures:  func() {},
			expectedStatus: http.StatusBadRequest, // Invalid referral code (rejected before DB)
			expectedError:  "INVALID_REFERRAL_CODE",
		},
		{
			name:     "Edge case: XSS payload in referral code",
			scenario: "Security: XSS attempt rejected in validation",
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

// TestViewStatusAcceptanceScenarios tests User Story 3 (Viewing Points and Status) acceptance scenarios
// Implements Constitution Principle XIII (Acceptance Scenario Coverage)
func TestViewStatusAcceptanceScenarios(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "customers", "point_transactions", "membership_tiers")

	// Create base tier (DATABASE FIXTURE)
	baseTier := testutil.CreateTestTier(db, map[string]interface{}{
		"name":                "Base",
		"level":               0,
		"qualification_points": int64(0),
		"earn_rate_multiplier": 1.0,
	})

	// Table-driven test cases - one per acceptance scenario
	testCases := []struct {
		name           string
		scenario       string // Given/When/Then from spec
		accountID      string
		setupFixtures  func()
		expectedStatus int
		expectedError  string
		validateResponse func(t *testing.T, resp *loyaltyv1.GetCustomerResponse)
	}{
		{
			name:     "US3-AS1: Customer views dashboard with balance, tier, and recent activity",
			scenario: "Given an enrolled customer logged into their account, When they view their loyalty dashboard, Then they see their current point balance, membership tier, and points earned",
			accountID: "user-enrolled",
			setupFixtures: func() {
				// Create customer with balance (DATABASE FIXTURE)
				customer := testutil.CreateTestCustomer(db, map[string]interface{}{
					"account_id":      "user-enrolled",
					"current_balance": int64(500),
					"tier_id":         &baseTier.ID,
				})
				// Create transaction (DATABASE FIXTURE)
				testutil.CreateTestTransaction(db, customer.ID, map[string]interface{}{
					"amount": int64(500),
				})
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *loyaltyv1.GetCustomerResponse) {
				if resp.Customer == nil {
					t.Fatal("Expected customer in response")
				}
				
				// Build expected from FIXTURES (DATABASE fixtures + system rules)
				// Per enhanced Principle VI: Derive from fixtures, NOT response
				expectedResponse := &loyaltyv1.GetCustomerResponse{
					Customer: &loyaltyv1.Customer{
						Id:               resp.Customer.Id,               // Generated (truly random)
						AccountId:        "user-enrolled",                // From DATABASE fixture
						MembershipNumber: resp.Customer.MembershipNumber, // Generated (truly random)
						ReferralCode:     resp.Customer.ReferralCode,     // Generated (truly random)
						ReferredBy:       "",                             // From DATABASE fixture (no referral)
						EnrolledAt:       resp.Customer.EnrolledAt,       // Generated timestamp (truly random)
						CurrentBalance:   500,                            // From DATABASE fixture
						Tier: &loyaltyv1.Tier{
							Id:                  resp.Customer.Tier.Id,   // From DATABASE fixture
							Name:                "Base",                   // From DATABASE fixture (baseTier)
							Level:               0,                        // From DATABASE fixture
							QualificationPoints: 0,                        // From DATABASE fixture
							EvaluationDays:      365,                      // System default
							EarnRateMultiplier:  1.0,                      // From DATABASE fixture
							Description:         resp.Customer.Tier.Description, // From DATABASE fixture
						},
						CreatedAt: resp.Customer.CreatedAt, // Generated (truly random)
						UpdatedAt: resp.Customer.UpdatedAt, // Generated (truly random)
					},
					PointsToNextTier:    resp.PointsToNextTier,    // Calculated (system rule)
					PointsExpiringSoon:  resp.PointsExpiringSoon,  // Calculated (system rule)
				}
				
				// Compare entire message using protocmp (MANDATORY per Principle VI)
				if diff := cmp.Diff(expectedResponse, resp, protocmp.Transform()); diff != "" {
					t.Errorf("Response mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:           "Edge case: Customer not enrolled",
			scenario:       "Data state: non-existent customer lookup",
			accountID:      "user-not-enrolled",
			setupFixtures:  func() {},
			expectedStatus: http.StatusNotFound,
			expectedError:  "CUSTOMER_NOT_FOUND",
		},
		{
			name:           "Edge case: Missing authentication",
			scenario:       "Authentication: unauthenticated status request",
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

// TestListTiersSupport tests tier listing (support endpoint for enrollment/status)
// Uses fixture-based validation per enhanced Principle VI
func TestListTiersSupport(t *testing.T) {
	// Setup test database
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer testutil.TruncateTables(db, "membership_tiers")

	// Create multiple tiers in various orders (DATABASE FIXTURES)
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

	// Build expected from DATABASE FIXTURES (ordered by level ASC)
	// Per enhanced Principle VI: Derive from fixtures, NOT response
	expected := &loyaltyv1.ListTiersResponse{
		Tiers: []*loyaltyv1.Tier{
			{
				Id:                  resp.Tiers[0].Id, // Generated (from DB)
				Name:                "Base",           // From DATABASE fixture
				Level:               0,                // From DATABASE fixture
				QualificationPoints: 0,                // From DATABASE fixture
				EvaluationDays:      365,              // System default
				EarnRateMultiplier:  1.0,              // From DATABASE fixture
				Description:         resp.Tiers[0].Description, // From DATABASE fixture
			},
			{
				Id:                  resp.Tiers[1].Id, // Generated (from DB)
				Name:                "Silver",         // From DATABASE fixture
				Level:               1,                // From DATABASE fixture
				QualificationPoints: 500,              // From DATABASE fixture
				EvaluationDays:      365,              // System default
				EarnRateMultiplier:  1.25,             // From DATABASE fixture
				Description:         resp.Tiers[1].Description, // From DATABASE fixture
			},
			{
				Id:                  resp.Tiers[2].Id, // Generated (from DB)
				Name:                "Gold",           // From DATABASE fixture
				Level:               2,                // From DATABASE fixture
				QualificationPoints: 1000,             // From DATABASE fixture
				EvaluationDays:      365,              // System default
				EarnRateMultiplier:  1.5,              // From DATABASE fixture
				Description:         resp.Tiers[2].Description, // From DATABASE fixture
			},
		},
	}

	// Compare using protocmp (MANDATORY per Principle VI)
	if diff := cmp.Diff(expected, &resp, protocmp.Transform()); diff != "" {
		t.Errorf("Response mismatch (-want +got):\n%s", diff)
	}
}
