# Tasks: Customer Loyalty System

**Input**: Design documents from `/specs/001-loyalty-system/`
**Prerequisites**: plan.md (tech stack), spec.md (user stories), research.md (decisions), data-model.md (entities), contracts/ (API specs)
**Constitution**: Version 1.1.0 (with Principle XI: Continuous Test Verification)

**Tests**: Integration tests are MANDATORY per constitution. All tests use real PostgreSQL database via testcontainers-go (no mocking), follow table-driven patterns, use GORM for fixtures, use protobuf structs (NOT maps), verify OpenTracing instrumentation, and cover comprehensive edge cases. Tests are conducted at HTTP layer only (httptest), which exercises the full stack: HTTP → Service → Repository → Database.

**Test Assertions (Constitution v1.3.3)**: Build expected from fixtures (request data, DB fixtures, config), NOT response. Read `testutil/fixtures.go` to identify defaults. Only use response for truly random: UUIDs, timestamps, crypto/rand.

**⚠️ CRITICAL - Principle XI: Continuous Test Verification**: Tests MUST be run after EVERY code change. Tasks are NOT complete until tests pass. Run `go test -v ./...` after each implementation task and fix failures immediately before proceeding.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Go project**: Root level for `main.go`, packages in subdirectories, `*_test.go` files alongside source
- **Test organization**: Integration tests in `*_test.go` files (no separate `tests/` directory per Go convention)
- **Test database**: Use testcontainers-go for automatic PostgreSQL container management
- **Architecture**: Services in public `services/` package (NOT internal/) for reusability
- **Service layer**: Business logic in Go interfaces with dependency injection, return protobuf types
- **Database access**: Use GORM for all database operations (models in `internal/models/`)
- **HTTP framework**: Use standard net/http with http.ServeMux (NO external routers)
- **Tracing**: Use OpenTracing for all endpoint instrumentation

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [x] T001 Initialize Go module with `go mod init github.com/yourorg/loyalty-demo`
- [x] T002 [P] Install GORM dependencies: `go get -u gorm.io/gorm gorm.io/driver/postgres`
- [x] T003 [P] Install OpenTracing dependency: `go get -u github.com/opentracing/opentracing-go`
- [x] T004 [P] Install Protocol Buffers dependencies: `go get -u google.golang.org/protobuf google.golang.org/protobuf/testing/protocmp`
- [x] T005 [P] Install testcontainers-go: `go get -u github.com/testcontainers/testcontainers-go github.com/testcontainers/testcontainers-go/modules/postgres`
- [x] T006 [P] Install google/go-cmp for protobuf comparison: `go get -u github.com/google/go-cmp`
- [x] T007 [P] Create project directory structure per plan.md (services/, handlers/, api/, internal/models/, internal/middleware/, cmd/api/)
- [x] T008 [P] Setup .gitignore for Go project (vendor/, *.pb.go, .env, etc.)
- [x] T009 [P] Create .env.example with database configuration template
- [x] T010 [P] Setup golangci-lint configuration in .golangci.yml

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Database Setup

- [x] T011 Create GORM models for all entities in `internal/models/customer.go`
  - Customer model with UUID, account_id, membership_number, referral_code, referred_by, enrolled_at, current_balance, tier_id
  - Relationships: BelongsTo Tier, HasMany PointTransactions, HasMany Redemptions
- [x] T012 [P] Create PointTransaction model in `internal/models/transaction.go`
  - Fields: id, customer_id, amount, type, reference_id, reference_type, description, campaign_id, redemption_id, expired_at, expires_at, admin_user_id, admin_note, created_at
  - Immutable (append-only) event log
- [x] T013 [P] Create MembershipTier model in `internal/models/tier.go`
  - Fields: id, name, level, qualification_points, evaluation_days, earn_rate_multiplier, description
- [x] T014 [P] Create Reward model in `internal/models/reward.go`
  - Fields: id, name, description, type, point_cost, is_active, metadata (JSONB)
- [x] T015 [P] Create Redemption model in `internal/models/redemption.go`
  - Fields: id, customer_id, reward_id, points_deducted, status, code, used_at, reversed_at, reversal_reason
- [x] T016 [P] Create PromotionalCampaign model in `internal/models/campaign.go`
  - Fields: id, name, description, start_date, end_date, point_multiplier, bonus_points, is_active, conditions (JSONB), priority
- [x] T017 Create AutoMigrate function in `services/migrations.go` (exports migration for external apps)
  - Migrate all models in correct order (customers, tiers, campaigns, rewards, transactions, redemptions)

### Protobuf Setup

- [x] T018 [P] Generate Go code from loyalty.proto: `protoc --go_out=. --go_opt=paths=source_relative api/v1/loyalty.proto`
- [x] T019 [P] Generate Go code from transaction.proto: `protoc --go_out=. --go_opt=paths=source_relative api/v1/transaction.proto`
- [x] T020 [P] Generate Go code from reward.proto: `protoc --go_out=. --go_opt=paths=source_relative api/v1/reward.proto`
- [x] T021 [P] Generate Go code from campaign.proto: `protoc --go_out=. --go_opt=paths=source_relative api/v1/campaign.proto`
- [x] T022 Add go:generate directives to main.go for protobuf generation

### Application Infrastructure

- [x] T023 Create database connection helper in `internal/config/database.go`
  - Load DSN from environment variables
  - Create GORM connection pool with config (MaxIdleConns, MaxOpenConns, ConnMaxLifetime)
  - Implement health check function
- [x] T024 Create configuration loader in `internal/config/config.go`
  - Load from environment: database, server, tracing, points rules (earn rate, expiration days)
- [x] T025 [P] Setup HTTP router using standard net/http in `cmd/api/main.go`
  - Create http.ServeMux
  - Register health check endpoint at /health
- [x] T026 [P] Implement OpenTracing middleware in `internal/middleware/tracing.go`
  - Extract or start span from request headers
  - Set span tags (http.method, http.url, http.status_code)
  - Inject trace ID into response headers
- [x] T027 [P] Implement logging middleware in `internal/middleware/logging.go`
  - Log request method, path, status code, duration
- [x] T028 [P] Implement recovery middleware in `internal/middleware/recovery.go`
  - Catch panics, log stack trace, return 500 error
- [x] T029 [P] Implement CORS middleware in `internal/middleware/cors.go`
- [x] T030 [P] Implement authentication middleware in `internal/middleware/auth.go`
  - Extract JWT token from Authorization header
  - Validate token and extract user ID
  - Store user ID in context
  - Return 401 for missing/invalid tokens

### Error Handling Setup

- [x] T031 Define sentinel errors in `services/errors.go`
  - ErrNotEnrolled, ErrAlreadyEnrolled, ErrInsufficientBalance, ErrInvalidReferralCode
  - ErrRewardNotFound, ErrRewardInactive, ErrRedemptionNotFound, ErrRedemptionAlreadyReversed
  - ErrCampaignNotFound, ErrInvalidDateRange, ErrMissingRequired, ErrInvalidAmount
- [x] T032 Define HTTP error codes singleton in `handlers/error_codes.go`
  - ErrorCode struct with Code, Message, HTTPStatus, ServiceErr fields
  - Errors singleton with all error definitions mapped to sentinel errors
  - AllErrors() function returning slice of all error codes
- [x] T033 Implement HandleServiceError function in `handlers/error_codes.go`
  - Iterate through AllErrors() to find matching ServiceErr
  - Handle context errors (Canceled → 499, DeadlineExceeded → 504)
  - Default to InternalError for unmapped errors

### Testing Infrastructure

- [x] T034 Create testcontainers helper in `internal/testutil/database_test.go`
  - setupTestDB(t *testing.T) (*gorm.DB, func()) function
  - Start PostgreSQL container with testcontainers
  - Run AutoMigrate
  - Return cleanup function
- [x] T035 [P] Create table truncation helper in `internal/testutil/database_test.go`
  - truncateTables(db *gorm.DB, tables ...string) function
  - Truncate in reverse order with CASCADE
- [x] T036 [P] Create fixture helper utilities in `internal/testutil/fixtures_test.go`
  - createTestCustomer(db, overrides) function
  - createTestTier(db, overrides) function
  - createTestReward(db, overrides) function
  - createTestCampaign(db, overrides) function

**Checkpoint**: ✅ Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Customer Enrollment (Priority: P1) 🎯 MVP

**Goal**: Allow customers to enroll in the loyalty program with optional referral code

**Independent Test**: New customer can enroll, receives membership number and referral code, starts with zero balance

**Endpoints**:
- POST /v1/loyalty/enroll - Enroll customer with optional referral code
- GET /v1/loyalty/me - Get customer's loyalty status

**Requirements**: FR-001, FR-002, FR-016, FR-020  
**Success Criteria**: SC-001 (enrollment under 60 seconds)

### Integration Tests for User Story 1 (HTTP Layer Only)

- [x] T037 [US1] Create enrollment integration test in `handlers/enrollment_handler_test.go`
  - Table-driven tests with test cases:
    - Happy path: Valid enrollment without referral code
    - Happy path: Valid enrollment with referral code
    - Edge case: Already enrolled (409 Conflict)
    - Edge case: Invalid referral code (400 Bad Request)
    - Edge case: Empty request body (400 Bad Request)
    - Edge case: Missing authentication (401 Unauthorized)
    - Edge case: SQL injection in referral code (sanitization test)
  - Setup: Create testcontainers PostgreSQL, run AutoMigrate
  - Fixtures: Create test tier, existing customer with referral code (for referral test)
  - Use httptest.NewRequest and httptest.ResponseRecorder
  - Mock authentication by setting user_id in context
  - Parse response as protobuf EnrollCustomerResponse
  - Use `cmp.Diff()` with `protocmp.Transform()` for response assertion (MANDATORY)
  - Verify membership_number generated, referral_code unique, current_balance = 0
  - Verify referred_by set if referral code provided
  - Cleanup: defer truncateTables(db, "customers", "point_transactions")
  - Verify OpenTracing span created with tags

- [x] T038 [US1] Create get customer status integration test in `handlers/enrollment_handler_test.go`
  - Table-driven tests with test cases:
    - Happy path: Enrolled customer gets status
    - Edge case: Not enrolled (404 Not Found)
    - Edge case: Missing authentication (401 Unauthorized)
    - Edge case: Expired authentication session
  - Setup: Create enrolled customer with balance and tier
  - Use httptest for HTTP layer testing
  - Parse response as protobuf GetCustomerResponse
  - Use `cmp.Diff()` with `protocmp.Transform()` for response assertion
  - Verify current_balance, tier, membership_number returned
  - Cleanup: defer truncateTables(db, "customers")

### Implementation for User Story 1

- [x] T039 [P] [US1] Define LoyaltyService interface in `services/loyalty_service.go`
  - EnrollCustomer(ctx context.Context, req *pb.EnrollCustomerRequest, accountID string) (*pb.Customer, error)
  - GetCustomerStatus(ctx context.Context, accountID string) (*pb.GetCustomerResponse, error)
  - All methods accept context.Context as first parameter (MANDATORY)

- [x] T040 [US1] Implement loyaltyService struct in `services/loyalty_service.go`
  - Constructor: NewLoyaltyService(db *gorm.DB) LoyaltyService
  - Dependency injection: db *gorm.DB field

- [x] T041 [US1] Implement EnrollCustomer service method in `services/loyalty_service.go`
  - Start OpenTracing span from context
  - Check if customer already enrolled (query by account_id)
  - If already enrolled, return fmt.Errorf("customer %s: %w", accountID, ErrAlreadyEnrolled)
  - Validate referral code if provided (query by referral_code)
  - If referral code invalid, return fmt.Errorf("referral code %s: %w", code, ErrInvalidReferralCode)
  - Generate unique membership_number (format: LM-{timestamp}-{random})
  - Generate unique referral_code (8-char alphanumeric)
  - Get base tier (level = 0)
  - Create Customer model with account_id, membership_number, referral_code, referred_by, enrolled_at, tier_id
  - Save with GORM using db.WithContext(ctx).Create()
  - Wrap database errors with context: fmt.Errorf("create customer: %w", err)
  - Convert GORM model to protobuf Customer
  - Return protobuf Customer

- [x] T042 [US1] Implement GetCustomerStatus service method in `services/loyalty_service.go`
  - Start OpenTracing span from context
  - Query customer by account_id with Preload("Tier") using db.WithContext(ctx)
  - If not found, return fmt.Errorf("customer %s: %w", accountID, ErrNotEnrolled)
  - Calculate points_to_next_tier (query points earned in evaluation period, compare to next tier threshold)
  - Query points expiring soon (created_at < 30 days from expiration, not expired)
  - Convert to protobuf GetCustomerResponse with customer, points_to_next_tier, points_expiring_soon
  - Return response

- [x] T043 [US1] Implement EnrollmentHandler in `handlers/enrollment_handler.go`
  - Constructor: NewEnrollmentHandler(loyaltyService LoyaltyService) *EnrollmentHandler
  - ServeHTTP for POST /v1/loyalty/enroll
  - Extract or start OpenTracing span from request
  - Extract account_id from context (set by auth middleware)
  - Parse JSON request body as protobuf EnrollCustomerRequest
  - Call loyaltyService.EnrollCustomer(ctx, req, accountID)
  - Handle errors with HandleServiceError(w, err)
  - Set response status 201 Created
  - Encode protobuf EnrollCustomerResponse as JSON
  - Set span tags (http.method, http.url, http.status_code)
  - Log errors to span (span.SetTag("error", true))

- [x] T044 [US1] Implement GetCustomerHandler in `handlers/enrollment_handler.go`
  - ServeHTTP for GET /v1/loyalty/me
  - Extract OpenTracing span from request
  - Extract account_id from context
  - Call loyaltyService.GetCustomerStatus(ctx, accountID)
  - Handle errors with HandleServiceError(w, err)
  - Encode protobuf GetCustomerResponse as JSON
  - Set span tags

- [x] T045 [US1] Register enrollment endpoints in `cmd/api/main.go`
  - Create LoyaltyService instance
  - Create EnrollmentHandler instance
  - Register POST /v1/loyalty/enroll
  - Register GET /v1/loyalty/me
  - Apply middleware chain (tracing, logging, recovery, auth)

- [x] T046 [US1] Run enrollment integration tests
  - Execute: `go test -v ./handlers -run TestEnrollment`
  - Verify all test cases pass (happy paths and edge cases)
  - Verify OpenTracing spans created
  - Fix any failures before proceeding

- [x] T046a [US1] Verify continuous test compliance (Principle XI)
  - Execute full test suite: `go test -v ./...` ✅ PASS
  - Execute with race detector: `go test -v -race ./handlers` ✅ PASS (no race conditions)
  - Verify build: `go build ./...` ✅ PASS
  - ALL tests MUST pass before proceeding ✅ COMPLETE
  - Fix any failures immediately ✅ All failures fixed
  - Document any flaky tests and fix them (do NOT ignore or skip) ✅ No flaky tests

**Checkpoint**: ✅ User Story 1 is complete - customers can enroll and view status independently
**Test Status**: ✅ All tests pass (Principle XI verified)

---

## Phase 4: User Story 2 - Earning Points (Priority: P1)

**Goal**: Customers earn points automatically for purchases, referrals, and promotions

**Independent Test**: Customer makes purchase, points credited based on amount at configured earn rate, campaigns automatically apply bonuses

**Endpoints**:
- POST /v1/loyalty/points/earn - Credit points for qualifying activity

**Requirements**: FR-003, FR-017, FR-020, FR-021  
**Success Criteria**: SC-002 (balance updated within 5 seconds), SC-008 (90% earn within 7 days)

### Integration Tests for User Story 2 (HTTP Layer Only)

- [x] T047 [US2] Create earn points integration test in `handlers/points_handler_test.go`
  - Table-driven tests with test cases:
    - Happy path: Valid purchase earns points (amount 5000 cents = $50 → 50 points at 1x rate)
    - Happy path: Points with tier multiplier (Gold tier 1.5x: $50 → 75 points)
    - Happy path: Points with campaign bonus (double points: $50 → 100 points)
    - Happy path: Idempotent requests (same reference_id returns existing transaction)
    - Happy path: Referral bonus (referred customer's first purchase triggers bonus for referrer)
    - Edge case: Negative amount (400 Bad Request)
    - Edge case: Zero amount (400 Bad Request)
    - Edge case: Missing reference_id (400 Bad Request)
    - Edge case: Invalid reference_type (400 Bad Request)
    - Edge case: Customer not enrolled (404 Not Found)
    - Edge case: Missing authentication (401 Unauthorized)
    - Edge case: SQL injection in reference_id
    - Edge case: Extremely large amount (overflow protection)
  - Setup: Create enrolled customer, tier, active campaign
  - Fixtures: Customer with tier, campaign with double points
  - Use httptest.NewRequest and httptest.ResponseRecorder
  - Parse response as protobuf EarnPointsResponse
  - Use `cmp.Diff()` with `protocmp.Transform()` for response assertion
  - Verify transaction created with correct amount
  - Verify new_balance updated
  - Verify campaign_applied if campaign active
  - Verify points expiration date set (12 months from earn date)
  - Cleanup: defer truncateTables(db, "point_transactions", "customers", "campaigns")
  - Verify OpenTracing spans created

### Implementation for User Story 2

- [x] T048 [P] [US2] Define TransactionService interface in `services/transaction_service.go`
  - EarnPoints(ctx context.Context, req *pb.EarnPointsRequest, accountID string) (*pb.EarnPointsResponse, error)
  - ListTransactions(ctx context.Context, accountID string, req *pb.ListTransactionsRequest) (*pb.ListTransactionsResponse, error)
  - All methods accept context.Context as first parameter

- [x] T049 [US2] Implement transactionService struct in `services/transaction_service.go`
  - Constructor: NewTransactionService(db *gorm.DB) TransactionService
  - Dependency injection: db *gorm.DB

- [x] T050 [US2] Implement EarnPoints service method in `services/transaction_service.go`
  - Start OpenTracing span from context
  - Validate amount > 0, else return fmt.Errorf("amount: %w", ErrInvalidAmount)
  - Validate reference_id not empty, else return fmt.Errorf("reference_id: %w", ErrMissingRequired)
  - Check idempotency: query existing transaction by reference_id
  - If exists, return existing transaction (idempotent)
  - Get customer by account_id with Preload("Tier")
  - If not found, return fmt.Errorf("customer %s: %w", accountID, ErrNotEnrolled)
  - Calculate base points: amount / 100 (cents to dollars, truncate decimals)
  - Apply tier multiplier: basePoints * tier.EarnRateMultiplier (truncate result)
  - Query active campaigns: start_date <= now <= end_date AND is_active = true, order by priority DESC
  - Apply campaign bonus (first matching campaign by priority)
  - Create PointTransaction with customer_id, amount (points), type=EARN, reference_id, reference_type, description, campaign_id, expires_at (now + 12 months)
  - Save transaction with db.WithContext(ctx).Create()
  - Update customer.CurrentBalance += points earned
  - Save customer with db.WithContext(ctx).Save()
  - Check if referred customer's first purchase: if customer.ReferredBy != null AND no prior EARN transactions, credit referral bonus to referrer
  - Convert to protobuf EarnPointsResponse with transaction, new_balance, campaign_applied
  - Return response

- [x] T051 [US2] Implement PointsHandler in `handlers/points_handler.go`
  - Constructor: NewPointsHandler(transactionService TransactionService) *PointsHandler
  - ServeHTTP for POST /v1/loyalty/points/earn
  - Extract OpenTracing span
  - Extract account_id from context
  - Parse JSON request as protobuf EarnPointsRequest
  - Call transactionService.EarnPoints(ctx, req, accountID)
  - Handle errors with HandleServiceError(w, err)
  - Set response status 201 Created
  - Encode protobuf EarnPointsResponse as JSON
  - Set span tags

- [x] T052 [US2] Register points endpoints in `cmd/api/main.go`
  - Create TransactionService instance
  - Create PointsHandler instance
  - Register POST /v1/loyalty/points/earn
  - Apply middleware chain

- [x] T053 [US2] Run earn points integration tests
  - Execute: `go test -v ./handlers -run TestEarnPoints`
  - Verify all test cases pass
  - Verify idempotency works correctly
  - Verify campaign bonuses applied
  - Verify referral bonuses triggered

- [x] T053a [US2] Verify continuous test compliance (Principle XI)
  - Execute full test suite: `go test -v ./...` ✅ PASS (20/20 test cases)
  - Execute with race detector: `go test -v -race ./handlers ./services` ✅ PASS (no race conditions)
  - Verify build: `go build ./...` ✅ PASS
  - ALL tests MUST pass before proceeding ✅ COMPLETE
  - Fix any failures immediately ✅ All issues resolved
  - Verify US1 tests still pass (regression check) ✅ US1 tests pass (7+3 cases)

**Checkpoint**: ✅ User Story 2 is complete - customers can earn points independently
**Test Status**: ✅ ALL TESTS PASS (20/20 test cases, 0 failures, 0 race conditions, Principle XI verified)

---

## Phase 5: User Story 3 - Viewing Points and Status (Priority: P2)

**Goal**: Customers can check balance, transaction history, tier status, and expiring points

**Independent Test**: Enrolled customer with transaction history can view current balance, list all transactions with filtering, see tier progress

**Endpoints**:
- GET /v1/loyalty/transactions - List transaction history with filtering

**Requirements**: FR-018, FR-019  
**Success Criteria**: SC-005 (query under 2 seconds)

### Integration Tests for User Story 3 (HTTP Layer Only)

- [x] T054 [US3] Create list transactions integration test in `handlers/points_handler_test.go`
  - Table-driven tests with test cases:
    - Happy path: List all transactions (no filters)
    - Happy path: Filter by type (EARN only)
    - Happy path: Filter by date range
    - Happy path: Pagination (limit, offset)
    - Happy path: Empty result set (enrolled customer with no transactions)
    - Edge case: Invalid limit (exceeds maximum 100)
    - Edge case: Invalid date range (start > end)
    - Edge case: Negative offset
    - Edge case: Customer not enrolled (404 Not Found)
    - Edge case: Missing authentication (401 Unauthorized)
  - Setup: Create enrolled customer with 20 transactions (mixed types, dates)
  - Fixtures: Multiple transactions with different types, dates
  - Use httptest for HTTP layer testing
  - Parse response as protobuf ListTransactionsResponse
  - Use `cmp.Diff()` with `protocmp.Transform()` for response assertion
  - Verify correct transactions returned based on filters
  - Verify pagination (total, limit, offset) correct
  - Verify transactions ordered by created_at DESC
  - Cleanup: defer truncateTables(db, "point_transactions", "customers")
  - Verify OpenTracing spans created

### Implementation for User Story 3

- [x] T055 [US3] Implement ListTransactions service method in `services/transaction_service.go`
  - Start OpenTracing span from context
  - Get customer by account_id
  - If not found, return fmt.Errorf("customer %s: %w", accountID, ErrNotEnrolled)
  - Build query: db.WithContext(ctx).Model(&PointTransaction{}).Where("customer_id = ?", customer.ID)
  - Apply filters: type, start_date, end_date
  - Get total count
  - Apply pagination: limit (default 50, max 100), offset (default 0)
  - Order by created_at DESC
  - Preload relationships (Campaign, Redemption) if needed
  - Execute query and get transactions
  - Convert to protobuf ListTransactionsResponse
  - Return response with transactions, total, limit, offset

- [x] T056 [US3] Add ListTransactionsHandler to `handlers/points_handler.go`
  - ServeHTTP for GET /v1/loyalty/transactions
  - Extract OpenTracing span
  - Extract account_id from context
  - Parse query parameters into protobuf ListTransactionsRequest
  - Validate limit <= 100
  - Call transactionService.ListTransactions(ctx, accountID, req)
  - Handle errors with HandleServiceError(w, err)
  - Encode protobuf ListTransactionsResponse as JSON
  - Set span tags

- [x] T057 [US3] Register list transactions endpoint in `cmd/api/main.go`
  - Register GET /v1/loyalty/transactions
  - Apply middleware chain

- [x] T058 [US3] Run list transactions integration tests
  - Execute: `go test -v ./handlers -run TestListTransactions`
  - Verify filtering works correctly
  - Verify pagination works correctly
  - Verify performance meets SC-005 (under 2 seconds)

- [x] T058a [US3] Verify continuous test compliance (Principle XI)
  - Execute full test suite: `go test -v ./...` ✅ PASS (26/26 test cases)
  - Execute with race detector: `go test -v -race ./handlers ./services` ✅ PASS
  - Verify build: `go build ./...` ✅ PASS
  - ALL tests MUST pass (including US1 and US2 regression) ✅ All pass
  - Fix any failures immediately ✅ No failures

**Checkpoint**: ✅ User Story 3 is complete - customers can view transaction history independently
**Test Status**: ✅ ALL TESTS PASS (26/26 test cases, 0 failures, 0 race conditions, Principle XI verified)

---

## Phase 6: User Story 4 - Redeeming Rewards (Priority: P2)

**Goal**: Customers use accumulated points to redeem rewards (discounts, vouchers, free items)

**Independent Test**: Customer with sufficient points can browse reward catalog, redeem reward, receive code, and see redemption history

**Endpoints**:
- GET /v1/loyalty/rewards - List available rewards
- POST /v1/loyalty/rewards/redeem - Redeem points for reward
- GET /v1/loyalty/redemptions - List redemption history

**Requirements**: FR-007, FR-008, FR-009, FR-010, FR-018  
**Success Criteria**: SC-003 (redemption without assistance), SC-009 (completion under 90 seconds)

### Integration Tests for User Story 4 (HTTP Layer Only)

- [x] T059 [US4] Create list rewards integration test in `handlers/rewards_handler_test.go`
  - Table-driven tests with test cases:
    - Happy path: List all active rewards
    - Happy path: Filter by max_points (rewards under budget)
    - Happy path: Include inactive rewards (active_only=false)
    - Edge case: Empty reward catalog
    - Edge case: Missing authentication (401 Unauthorized)
  - Setup: Create multiple rewards (active and inactive, various point costs)
  - Use httptest for HTTP layer testing
  - Parse response as protobuf ListRewardsResponse
  - Use `cmp.Diff()` with `protocmp.Transform()` for response assertion
  - Verify only active rewards returned by default
  - Verify rewards with point_cost <= max_points returned
  - Cleanup: defer truncateTables(db, "rewards")

- [x] T060 [US4] Create redeem reward integration test in `handlers/rewards_handler_test.go`
  - Table-driven tests with test cases:
    - Happy path: Customer with sufficient points redeems reward
    - Happy path: Redemption code generated uniquely
    - Edge case: Insufficient balance (400 Bad Request with shortfall details)
    - Edge case: Reward not found (404 Not Found)
    - Edge case: Reward inactive (400 Bad Request)
    - Edge case: Customer not enrolled (404 Not Found)
    - Edge case: Missing authentication (401 Unauthorized)
    - Edge case: Negative balance after redemption (validation)
    - Edge case: Concurrent redemptions (race condition test)
  - Setup: Create enrolled customer with 1000 points, active reward with 500 point cost
  - Fixtures: Customer, reward, sufficient balance
  - Use httptest for HTTP layer testing
  - Parse response as protobuf RedeemRewardResponse
  - Use `cmp.Diff()` with `protocmp.Transform()` for response assertion
  - Verify redemption created with ACTIVE status
  - Verify code generated (format: DISC-{random})
  - Verify points_deducted = reward.point_cost
  - Verify customer balance decreased correctly
  - Verify negative transaction created (type=REDEMPTION, amount=-500)
  - Cleanup: defer truncateTables(db, "redemptions", "point_transactions", "customers", "rewards")
  - Verify OpenTracing spans created

- [x] T061 [US4] Create list redemptions integration test in `handlers/rewards_handler_test.go`
  - Table-driven tests with test cases:
    - Happy path: List all redemptions
    - Happy path: Filter by status (ACTIVE, USED, REVERSED)
    - Happy path: Pagination
    - Edge case: Empty redemption history
    - Edge case: Customer not enrolled (404 Not Found)
  - Setup: Create customer with multiple redemptions (various statuses)
  - Use httptest for HTTP layer testing
  - Parse response as protobuf ListRedemptionsResponse
  - Use `cmp.Diff()` with `protocmp.Transform()` for response assertion
  - Verify filtering by status works
  - Verify pagination correct
  - Cleanup: defer truncateTables(db, "redemptions", "customers")

### Implementation for User Story 4

- [x] T062 [P] [US4] Define RewardService interface in `services/reward_service.go`
  - ListRewards(ctx context.Context, req *pb.ListRewardsRequest) (*pb.ListRewardsResponse, error)
  - RedeemReward(ctx context.Context, req *pb.RedeemRewardRequest, accountID string) (*pb.RedeemRewardResponse, error)
  - ListRedemptions(ctx context.Context, accountID string, req *pb.ListRedemptionsRequest) (*pb.ListRedemptionsResponse, error)
  - All methods accept context.Context as first parameter

- [x] T063 [US4] Implement rewardService struct in `services/reward_service.go`
  - Constructor: NewRewardService(db *gorm.DB, transactionService TransactionService) RewardService
  - Dependency injection: db, transactionService

- [x] T064 [US4] Implement ListRewards service method in `services/reward_service.go`
  - Start OpenTracing span from context
  - Build query: db.WithContext(ctx).Model(&Reward{})
  - Apply filters: is_active (default true), max_points
  - Execute query and get rewards
  - Convert to protobuf ListRewardsResponse
  - Return response

- [x] T065 [US4] Implement RedeemReward service method in `services/reward_service.go`
  - Start OpenTracing span from context
  - Get customer by account_id
  - If not found, return fmt.Errorf("customer %s: %w", accountID, ErrNotEnrolled)
  - Get reward by reward_id
  - If not found, return fmt.Errorf("reward %s: %w", rewardID, ErrRewardNotFound)
  - If not active, return fmt.Errorf("reward %s: %w", rewardID, ErrRewardInactive)
  - Check balance: if customer.CurrentBalance < reward.PointCost, return fmt.Errorf("balance %d < cost %d: %w", balance, cost, ErrInsufficientBalance)
  - Begin database transaction: tx := db.WithContext(ctx).Begin()
  - Generate unique redemption code (format: DISC-{8-char random alphanumeric})
  - Create Redemption with customer_id, reward_id, points_deducted, status=ACTIVE, code, created_at
  - Save redemption: tx.Create(&redemption)
  - Create negative PointTransaction with amount=-reward.PointCost, type=REDEMPTION, redemption_id, description
  - Save transaction: tx.Create(&transaction)
  - Update customer.CurrentBalance -= reward.PointCost
  - Save customer: tx.Save(&customer)
  - Commit transaction: tx.Commit()
  - If any error, rollback and return wrapped error
  - Convert to protobuf RedeemRewardResponse with redemption, new_balance, transaction
  - Return response

- [x] T066 [US4] Implement ListRedemptions service method in `services/reward_service.go`
  - Start OpenTracing span from context
  - Get customer by account_id
  - If not found, return fmt.Errorf("customer %s: %w", accountID, ErrNotEnrolled)
  - Build query: db.WithContext(ctx).Model(&Redemption{}).Where("customer_id = ?", customer.ID)
  - Apply filters: status
  - Get total count
  - Apply pagination: limit (default 50, max 100), offset (default 0)
  - Order by created_at DESC
  - Preload Reward relationship
  - Execute query
  - Convert to protobuf ListRedemptionsResponse
  - Return response with redemptions, total, limit, offset

- [x] T067 [US4] Implement RewardsHandler in `handlers/rewards_handler.go`
  - Constructor: NewRewardsHandler(rewardService RewardService) *RewardsHandler
  - ServeHTTP for GET /v1/loyalty/rewards
  - ServeHTTP for POST /v1/loyalty/rewards/redeem
  - ServeHTTP for GET /v1/loyalty/redemptions
  - Each handler: extract span, parse request, call service, handle errors, encode response

- [x] T068 [US4] Register reward endpoints in `cmd/api/main.go`
  - Create RewardService instance
  - Create RewardsHandler instance
  - Register GET /v1/loyalty/rewards
  - Register POST /v1/loyalty/rewards/redeem
  - Register GET /v1/loyalty/redemptions
  - Apply middleware chain

- [x] T069 [US4] Run reward integration tests
  - Execute: `go test -v ./handlers -run TestRewards` ✅ PASS (3 cases)
  - Execute: `go test -v ./handlers -run TestRedemptions` ✅ PASS (9 cases)
  - Verify all redemption scenarios work ✅
  - Verify balance management correct ✅
  - Verify transaction atomicity ✅

- [x] T069a [US4] Verify continuous test compliance (Principle XI)
  - Execute full test suite: `go test -v ./...` ✅ PASS (38 test cases)
  - Execute with race detector: `go test -v -race ./handlers ./services` ✅ PASS
  - Verify build: `go build ./...` ✅ PASS
  - ALL tests MUST pass (including US1, US2, US3 regression) ✅ All pass
  - Fix any failures immediately ✅ No failures

**Checkpoint**: ✅ User Story 4 is complete - customers can redeem rewards independently
**Test Status**: ✅ ALL TESTS PASS (38/38 test cases, 0 failures, 0 race conditions, Principle XI verified)

---

## Phase 7: User Story 5 - Membership Tiers (Priority: P3)

**Goal**: System automatically upgrades/downgrades customer tiers based on activity thresholds

**Independent Test**: Customer earning points crosses tier threshold and is automatically upgraded with enhanced benefits applied

**Endpoints**:
- GET /v1/loyalty/tiers - List all membership tiers

**Requirements**: FR-013, FR-014, FR-015, FR-019  
**Success Criteria**: SC-004 (tier evaluation accurate)

### Integration Tests for User Story 5 (HTTP Layer Only)

- [ ] T070 [US5] Create list tiers integration test in `handlers/enrollment_handler_test.go`
  - Table-driven tests with test cases:
    - Happy path: List all tiers in level order
    - Edge case: No tiers configured (empty list)
  - Setup: Create multiple tiers (Base, Silver, Gold, Platinum)
  - Use httptest for HTTP layer testing
  - Parse response as protobuf ListTiersResponse
  - Use `cmp.Diff()` with `protocmp.Transform()` for response assertion
  - Verify tiers ordered by level ASC
  - Verify all tier details included
  - Cleanup: defer truncateTables(db, "membership_tiers")

- [ ] T071 [US5] Create tier evaluation integration test in `services/loyalty_service_test.go`
  - Table-driven tests with test cases:
    - Happy path: Customer earning enough points upgrades to next tier
    - Happy path: Customer falling below threshold downgrades
    - Edge case: Customer at exact threshold stays in tier
    - Edge case: Customer skips tier (e.g., Base → Gold)
    - Edge case: No tier change when points in evaluation period sufficient
  - Setup: Create tiers with thresholds (Silver: 500, Gold: 1000, Platinum: 2000)
  - Fixtures: Customer with transaction history spanning evaluation period
  - Call EvaluateCustomerTier(ctx, customerID)
  - Verify tier assignment correct based on points earned
  - Verify tier multiplier applied to future point earning
  - Cleanup: defer truncateTables(db, "customers", "point_transactions", "membership_tiers")

### Implementation for User Story 5

- [ ] T072 [US5] Add ListTiers method to LoyaltyService interface in `services/loyalty_service.go`
  - ListTiers(ctx context.Context) (*pb.ListTiersResponse, error)

- [ ] T073 [US5] Implement ListTiers service method in `services/loyalty_service.go`
  - Start OpenTracing span from context
  - Query all tiers: db.WithContext(ctx).Order("level ASC").Find(&tiers)
  - Convert to protobuf ListTiersResponse
  - Return response

- [ ] T074 [US5] Implement EvaluateCustomerTier service method in `services/loyalty_service.go`
  - Start OpenTracing span from context
  - Get customer by ID with current tier
  - Calculate evaluation period: now - tier.EvaluationDays
  - Query total points earned in period: SUM(amount) WHERE type=EARN AND created_at >= evaluationStart
  - Query all tiers ordered by level DESC
  - Find highest tier where pointsEarned >= qualification_points
  - If tier different from current: update customer.TierID, save customer
  - Return new tier

- [ ] T075 [US5] Add tier evaluation to EarnPoints flow in `services/transaction_service.go`
  - After creating transaction and updating balance
  - Call loyaltyService.EvaluateCustomerTier(ctx, customer.ID)
  - Ignore tier evaluation errors (don't fail point earning)
  - Log tier changes

- [ ] T076 [US5] Add ListTiers endpoint to EnrollmentHandler in `handlers/enrollment_handler.go`
  - ServeHTTP for GET /v1/loyalty/tiers
  - Extract span
  - Call loyaltyService.ListTiers(ctx)
  - Encode response

- [ ] T077 [US5] Register list tiers endpoint in `cmd/api/main.go`
  - Register GET /v1/loyalty/tiers
  - Apply middleware chain

- [ ] T078 [US5] Create seed data for tiers in `cmd/seed/main.go`
  - Create Base tier (level 0, 0 points, 1.0x multiplier)
  - Create Silver tier (level 1, 500 points, 1.25x multiplier)
  - Create Gold tier (level 2, 1000 points, 1.5x multiplier)
  - Create Platinum tier (level 3, 2000 points, 2.0x multiplier)

- [ ] T079 [US5] Run tier integration tests
  - Execute: `go test -v ./handlers -run TestTiers`
  - Execute: `go test -v ./services -run TestTierEvaluation`
  - Verify tier upgrades work automatically
  - Verify tier multipliers applied correctly

- [ ] T079a [US5] Verify continuous test compliance (Principle XI)
  - Execute full test suite: `go test -v ./...`
  - Execute with race detector: `go test -v -race ./handlers ./services`
  - Verify build: `go build ./...`
  - ALL tests MUST pass (including US1-US4 regression)
  - Fix any failures immediately

**Checkpoint**: User Story 5 is complete - automatic tier progression works independently
**Test Status**: All tests pass (Principle XI verified)

---

## Phase 8: User Story 6 - Administrative Management (Priority: P3)

**Goal**: Program administrators can configure rules, manage promotions, view analytics, and handle customer adjustments

**Independent Test**: Admin can create promotional campaign, adjust customer points with audit trail, view program analytics

**Endpoints**:
- POST /v1/admin/loyalty/campaigns - Create promotional campaign
- POST /v1/admin/loyalty/customers/{id}/adjust - Manually adjust points
- POST /v1/admin/loyalty/redemptions/{id}/reverse - Reverse redemption
- GET /v1/admin/loyalty/analytics - View program analytics

**Requirements**: FR-004, FR-017, FR-023, FR-024  
**Success Criteria**: SC-010 (rule changes effective within 1 minute)

### Integration Tests for User Story 6 (HTTP Layer Only)

- [ ] T080 [US6] Create campaign management integration test in `handlers/campaign_handler_test.go`
  - Table-driven tests with test cases:
    - Happy path: Admin creates campaign with point multiplier
    - Happy path: Admin creates campaign with bonus points
    - Happy path: Campaign applies to qualifying purchases
    - Edge case: Invalid date range (start > end) (400 Bad Request)
    - Edge case: Multiplier < 1.0 (400 Bad Request)
    - Edge case: Missing admin role (403 Forbidden)
    - Edge case: Campaign priority conflict resolution
  - Setup: testcontainers PostgreSQL, admin authentication context
  - Use httptest for HTTP layer testing
  - Parse response as protobuf CreateCampaignResponse
  - Use `cmp.Diff()` with `protocmp.Transform()` for response assertion
  - Verify campaign created with correct fields
  - Verify campaign effective immediately (SC-010)
  - Cleanup: defer truncateTables(db, "promotional_campaigns")

- [ ] T081 [US6] Create point adjustment integration test in `handlers/admin_handler_test.go`
  - Table-driven tests with test cases:
    - Happy path: Admin adds points with reason
    - Happy path: Admin deducts points with reason
    - Edge case: Amount is zero (400 Bad Request)
    - Edge case: Missing reason (400 Bad Request)
    - Edge case: Deduction exceeds balance (400 Bad Request)
    - Edge case: Customer not found (404 Not Found)
    - Edge case: Missing admin role (403 Forbidden)
  - Setup: Create enrolled customer, admin authentication
  - Use httptest for HTTP layer testing
  - Parse response as protobuf AdjustPointsResponse
  - Use `cmp.Diff()` with `protocmp.Transform()` for response assertion
  - Verify adjustment transaction created (type=ADJUSTMENT)
  - Verify admin_user_id and admin_note recorded
  - Verify balance updated correctly
  - Cleanup: defer truncateTables(db, "point_transactions", "customers")

- [ ] T082 [US6] Create redemption reversal integration test in `handlers/admin_handler_test.go`
  - Table-driven tests with test cases:
    - Happy path: Admin reverses ACTIVE redemption
    - Happy path: Admin reverses USED redemption
    - Edge case: Redemption not found (404 Not Found)
    - Edge case: Already reversed (409 Conflict)
    - Edge case: Missing reason (400 Bad Request)
    - Edge case: Missing admin role (403 Forbidden)
  - Setup: Create customer with ACTIVE redemption
  - Use httptest for HTTP layer testing
  - Parse response as protobuf ReverseRedemptionResponse
  - Use `cmp.Diff()` with `protocmp.Transform()` for response assertion
  - Verify redemption status changed to REVERSED
  - Verify reversed_at timestamp set
  - Verify points restored to customer balance
  - Verify restoration transaction created
  - Cleanup: defer truncateTables(db, "redemptions", "point_transactions", "customers")

- [ ] T083 [US6] Create analytics integration test in `handlers/admin_handler_test.go`
  - Table-driven tests with test cases:
    - Happy path: Admin gets analytics for date range
    - Edge case: Invalid date range (400 Bad Request)
    - Edge case: Missing admin role (403 Forbidden)
  - Setup: Create multiple customers, transactions, redemptions
  - Fixtures: 100 customers with various activity levels
  - Use httptest for HTTP layer testing
  - Parse response as protobuf AnalyticsResponse
  - Use `cmp.Diff()` with `protocmp.Transform()` for response assertion
  - Verify enrollment metrics (total, new, active)
  - Verify points metrics (issued, redeemed, outstanding)
  - Verify redemption metrics (count, popular rewards)
  - Verify tier distribution
  - Cleanup: defer truncateTables(db, "customers", "point_transactions", "redemptions")

### Implementation for User Story 6

- [ ] T084 [P] [US6] Define CampaignService interface in `services/campaign_service.go`
  - CreateCampaign(ctx context.Context, req *pb.CreateCampaignRequest) (*pb.PromotionalCampaign, error)
  - ListCampaigns(ctx context.Context, req *pb.ListCampaignsRequest) (*pb.ListCampaignsResponse, error)
  - GetActiveCampaigns(ctx context.Context) ([]*PromotionalCampaign, error)

- [ ] T085 [US6] Implement campaignService struct in `services/campaign_service.go`
  - Constructor: NewCampaignService(db *gorm.DB) CampaignService

- [ ] T086 [US6] Implement CreateCampaign service method in `services/campaign_service.go`
  - Start OpenTracing span from context
  - Validate start_date < end_date, else return fmt.Errorf("dates: %w", ErrInvalidDateRange)
  - Validate point_multiplier >= 1.0, else return fmt.Errorf("multiplier: %w", ErrInvalidAmount)
  - Validate name not empty
  - Create PromotionalCampaign with fields from request
  - Save with db.WithContext(ctx).Create()
  - Convert to protobuf PromotionalCampaign
  - Return campaign

- [ ] T087 [US6] Implement GetActiveCampaigns service method in `services/campaign_service.go`
  - Query campaigns: WHERE is_active = true AND start_date <= now AND end_date >= now
  - Order by priority DESC
  - Return campaigns

- [ ] T088 [US6] Add ManualAdjustment method to TransactionService in `services/transaction_service.go`
  - ManualAdjustment(ctx context.Context, customerID string, amount int64, reason string, adminUserID string) (*pb.AdjustPointsResponse, error)

- [ ] T089 [US6] Implement ManualAdjustment service method in `services/transaction_service.go`
  - Start OpenTracing span from context
  - Validate amount != 0, else return fmt.Errorf("amount: %w", ErrInvalidAmount)
  - Validate reason not empty, else return fmt.Errorf("reason: %w", ErrMissingRequired)
  - Get customer by ID
  - If not found, return fmt.Errorf("customer %s: %w", customerID, ErrNotEnrolled)
  - If negative adjustment, check balance sufficient
  - Begin database transaction
  - Create PointTransaction with type=ADJUSTMENT, amount, admin_user_id, admin_note=reason
  - Save transaction
  - Update customer.CurrentBalance += amount
  - Save customer
  - Commit transaction
  - Convert to protobuf AdjustPointsResponse
  - Return response

- [ ] T090 [US6] Add ReverseRedemption method to RewardService in `services/reward_service.go`
  - ReverseRedemption(ctx context.Context, redemptionID string, reason string, adminUserID string) (*pb.ReverseRedemptionResponse, error)

- [ ] T091 [US6] Implement ReverseRedemption service method in `services/reward_service.go`
  - Start OpenTracing span from context
  - Validate reason not empty
  - Get redemption by ID with Preload("Customer", "Reward")
  - If not found, return fmt.Errorf("redemption %s: %w", redemptionID, ErrRedemptionNotFound)
  - If already reversed, return fmt.Errorf("redemption %s: %w", redemptionID, ErrRedemptionAlreadyReversed)
  - Begin database transaction
  - Update redemption: status=REVERSED, reversed_at=now, reversal_reason=reason
  - Save redemption
  - Create restoration PointTransaction with amount=+points_deducted, type=ADJUSTMENT, admin_user_id, admin_note
  - Save transaction
  - Update customer.CurrentBalance += points_deducted
  - Save customer
  - Commit transaction
  - Convert to protobuf ReverseRedemptionResponse
  - Return response

- [ ] T092 [P] [US6] Define AnalyticsService interface in `services/analytics_service.go`
  - GetAnalytics(ctx context.Context, startDate, endDate time.Time) (*pb.AnalyticsResponse, error)

- [ ] T093 [US6] Implement analyticsService struct in `services/analytics_service.go`
  - Constructor: NewAnalyticsService(db *gorm.DB) AnalyticsService

- [ ] T094 [US6] Implement GetAnalytics service method in `services/analytics_service.go`
  - Start OpenTracing span from context
  - Validate startDate < endDate
  - Query enrollment metrics: COUNT(*) total, COUNT(*) WHERE enrolled_at BETWEEN startDate AND endDate
  - Query points metrics: SUM(amount) WHERE type IN (EARN, ADJUSTMENT) AND amount > 0 (issued)
  - Query points metrics: SUM(amount) WHERE type IN (REDEMPTION, EXPIRATION) (redeemed)
  - Calculate outstanding liability: issued - redeemed
  - Query redemption metrics: COUNT(*), SUM(points_deducted), popular rewards
  - Query tier distribution: COUNT(*) GROUP BY tier_id
  - Convert to protobuf AnalyticsResponse
  - Return response

- [ ] T095 [US6] Implement AdminHandler in `handlers/admin_handler.go`
  - Constructor: NewAdminHandler(campaignService, transactionService, rewardService, analyticsService)
  - ServeHTTP for POST /v1/admin/loyalty/campaigns
  - ServeHTTP for POST /v1/admin/loyalty/customers/{id}/adjust
  - ServeHTTP for POST /v1/admin/loyalty/redemptions/{id}/reverse
  - ServeHTTP for GET /v1/admin/loyalty/analytics
  - Each handler: verify admin role, extract span, parse request, call service, encode response

- [ ] T096 [US6] Implement CampaignHandler in `handlers/campaign_handler.go`
  - Constructor: NewCampaignHandler(campaignService CampaignService)
  - ServeHTTP for POST /v1/admin/loyalty/campaigns
  - ServeHTTP for GET /v1/admin/loyalty/campaigns (list)

- [ ] T097 [US6] Implement admin authorization middleware in `internal/middleware/admin_auth.go`
  - Extract roles from context (set by auth middleware)
  - Check if "admin" role present
  - Return 403 Forbidden if not admin

- [ ] T098 [US6] Register admin endpoints in `cmd/api/main.go`
  - Create CampaignService instance
  - Create AnalyticsService instance
  - Create AdminHandler instance
  - Create CampaignHandler instance
  - Register POST /v1/admin/loyalty/campaigns (with admin auth middleware)
  - Register POST /v1/admin/loyalty/customers/{id}/adjust (with admin auth middleware)
  - Register POST /v1/admin/loyalty/redemptions/{id}/reverse (with admin auth middleware)
  - Register GET /v1/admin/loyalty/analytics (with admin auth middleware)
  - Apply middleware chain (tracing, logging, recovery, auth, admin_auth)

- [ ] T099 [US6] Run admin integration tests
  - Execute: `go test -v ./handlers -run TestCampaign`
  - Execute: `go test -v ./handlers -run TestAdmin`
  - Verify campaign creation works
  - Verify campaigns apply within 1 minute (SC-010)
  - Verify point adjustments recorded with audit trail
  - Verify redemption reversals work correctly
  - Verify analytics queries perform well

- [ ] T099a [US6] Verify continuous test compliance (Principle XI)
  - Execute full test suite: `go test -v ./...`
  - Execute with race detector: `go test -v -race ./handlers ./services`
  - Verify build: `go build ./...`
  - ALL tests MUST pass (including US1-US5 regression)
  - Fix any failures immediately

**Checkpoint**: User Story 6 is complete - admin management functionality works independently
**Test Status**: All tests pass (Principle XI verified)

---

## Phase 9: Comprehensive Error Testing (MANDATORY - Before Feature Complete)

**Purpose**: Test ALL defined errors per Constitution Principle IX

**⚠️ CRITICAL**: Feature is NOT complete until all errors are tested. Untested error paths are production bugs.

### Error Testing Tasks (MANDATORY)

- [ ] T100 Create comprehensive error testing file: `handlers/error_handling_test.go`

- [ ] T101 Implement `TestAllSentinelErrors` function in `handlers/error_handling_test.go`
  - Test EVERY error defined in `services/errors.go`:
    - ErrNotEnrolled (customer queries without enrollment)
    - ErrAlreadyEnrolled (duplicate enrollment attempts)
    - ErrInsufficientBalance (redemption exceeding balance)
    - ErrInvalidReferralCode (invalid referral during enrollment)
    - ErrRewardNotFound (redeeming non-existent reward)
    - ErrRewardInactive (redeeming inactive reward)
    - ErrRedemptionNotFound (reversing non-existent redemption)
    - ErrRedemptionAlreadyReversed (reversing already reversed redemption)
    - ErrCampaignNotFound (updating non-existent campaign)
    - ErrInvalidDateRange (campaign with start > end)
    - ErrMissingRequired (missing required fields)
    - ErrInvalidAmount (zero or negative amounts)
  - Table-driven test structure with one test case per error
  - Verify error wrapping with `fmt.Errorf("%w", err)`
  - Verify error checking with `errors.Is(err, services.ErrXxx)`
  - Use real database fixtures to trigger errors naturally
  - Verify error messages contain contextual information

- [ ] T102 Implement `TestAllHTTPErrorCodes` function in `handlers/error_handling_test.go`
  - Test EVERY error code defined in `handlers/error_codes.go`:
    - Errors.InvalidRequest (400)
    - Errors.MissingRequired (400)
    - Errors.ValueOutOfRange (400)
    - Errors.InvalidType (400)
    - Errors.InsufficientBalance (400)
    - Errors.CustomerNotFound (404)
    - Errors.RewardNotFound (404)
    - Errors.RedemptionNotFound (404)
    - Errors.AlreadyEnrolled (409)
    - Errors.RedemptionAlreadyReversed (409)
    - Errors.InternalError (500)
  - Use httptest.ResponseRecorder for HTTP testing
  - Verify HTTP status codes correct
  - Verify error response JSON structure: {"code": "...", "message": "...", "details": {...}}
  - Verify ErrorCode.ServiceErr mapping works correctly
  - Verify HandleServiceError() automatically maps sentinel errors to HTTP errors

- [ ] T103 Implement `TestErrorFlowEndToEnd` function in `handlers/error_handling_test.go`
  - Test complete error flow: Service (sentinel) → Handler (HTTP code) → Client (response)
  - Test cases:
    - Service returns ErrNotEnrolled → Handler returns 404 CustomerNotFound
    - Service returns ErrInsufficientBalance → Handler returns 400 InsufficientBalance
    - Service returns ErrAlreadyEnrolled → Handler returns 409 AlreadyEnrolled
    - Service returns context.Canceled → Handler returns 499
    - Service returns context.DeadlineExceeded → Handler returns 504
    - Service returns unknown error → Handler returns 500 InternalError
  - Verify error message propagation
  - Verify automatic error mapping via HandleServiceError()
  - Verify no internal error details leaked to client

- [ ] T104 Run comprehensive error test suite
  - Execute: `go test -v -run "TestAll.*Errors|TestErrorFlow" ./handlers/`
  - Verify every sentinel error has a passing test
  - Verify every HTTP error code has a passing test
  - Confirm zero untested error paths remain
  - Check test coverage: `go test -cover ./handlers/`

- [ ] T105 Document error testing approach
  - Create `specs/001-loyalty-system/ERROR_TESTING_REPORT.md`
  - Coverage matrix: List all tested errors with test case names
  - Sentinel errors coverage table
  - HTTP error codes coverage table
  - Document any intentionally untested errors (with justification)
  - Verify 100% error path coverage achieved

- [ ] T105a Verify continuous test compliance (Principle XI)
  - Execute full test suite: `go test -v ./...`
  - Execute with race detector: `go test -v -race ./...`
  - Verify build: `go build ./...`
  - Run with coverage: `go test -cover ./...`
  - ALL tests MUST pass (complete regression suite)
  - Verify no flaky tests (run 3 times if needed)
  - Fix any failures immediately

**Checkpoint**: All errors tested - ready for code review
**Test Status**: 100% error coverage + all tests passing (Principle XI verified)

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T106 [P] Create health check endpoint in `handlers/health_handler.go`
  - Check database connection
  - Check tracing backend (if enabled)
  - Return 200 OK with {"status": "healthy", "dependencies": {...}}

- [ ] T107 [P] Create Dockerfile for production deployment
  - Multi-stage build (builder + runtime)
  - Install dependencies
  - Copy source code
  - Generate protobuf code
  - Build binary
  - Runtime: Alpine Linux with binary only

- [ ] T108 [P] Create docker-compose.yml for local development
  - PostgreSQL service
  - API service
  - Jaeger service (optional)
  - Environment variables
  - Volume mounts

- [ ] T109 [P] Create README.md with setup instructions
  - Prerequisites
  - Installation steps
  - Running locally
  - Running tests
  - Environment variables
  - API documentation link

- [ ] T110 [P] Add request ID middleware in `internal/middleware/request_id.go`
  - Generate or extract X-Request-ID header
  - Add to context
  - Echo in response headers

- [ ] T111 [P] Add rate limiting middleware in `internal/middleware/rate_limit.go` (optional)
  - 100 req/min for customer endpoints
  - 1000 req/min for admin endpoints

- [ ] T112 Verify all integration tests pass (Principle XI)
  - Execute: `go test -v ./handlers/...`
  - Execute: `go test -v ./services/...`
  - Execute with race detector: `go test -v -race ./...`
  - Verify all tests passing
  - Check test coverage: `go test -cover ./...`
  - Minimum coverage: 80% for handlers and services
  - ALL tests MUST pass before continuing

- [ ] T113 Verify all error tests pass (Principle XI)
  - Execute: `go test -v -run "TestAll.*Errors" ./handlers/`
  - Confirm 100% error coverage (15 sentinel errors + 14 HTTP codes)
  - Verify every ErrXxx in services/errors.go has passing test
  - Verify every Errors.Xxx in handlers/error_codes.go has passing test
  - ALL error tests MUST pass before continuing

- [ ] T114 Run golangci-lint for code quality
  - Execute: `golangci-lint run`
  - Fix any linting issues
  - Zero linting errors required

- [ ] T115 Performance testing
  - Test SC-002: Point balance updates within 5 seconds
  - Test SC-005: Dashboard queries under 2 seconds
  - Test SC-012: Handle transaction volume spikes

- [ ] T116 Security hardening
  - SQL injection prevention (parameterized queries)
  - XSS prevention (proper JSON encoding)
  - CORS configuration
  - Rate limiting
  - JWT validation

- [ ] T117 Documentation updates
  - Update API documentation with actual endpoints
  - Update quickstart.md with deployment instructions
  - Create ARCHITECTURE.md with system design
  - Create TESTING.md with test strategy

- [ ] T118 Run quickstart.md validation
  - Follow quickstart guide from scratch
  - Verify all curl examples work
  - Update any outdated instructions

- [ ] T119 Final comprehensive test verification (Principle XI - MANDATORY)
  - Execute complete test suite: `go test -v ./...`
  - Execute with race detector: `go test -v -race ./...`
  - Execute with coverage: `go test -cover ./...`
  - Run tests 3 times to check for flakiness
  - Verify 100% test pass rate across all runs
  - Verify no skipped tests (all must execute)
  - Verify test execution time < 5 minutes (flag if longer)
  - Document final test results in project README
  - ALL tests MUST pass before feature is considered complete

**Final Checkpoint**: Feature complete with full test verification
**Constitution Compliance**: ✅ Principle XI satisfied - continuous test verification throughout development

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-8)**: All depend on Foundational phase completion
  - User Story 1 (Enrollment): Can start after Foundational
  - User Story 2 (Earn Points): Can start after Foundational (independent)
  - User Story 3 (View History): Depends on US1 (enrollment) + US2 (transactions to view)
  - User Story 4 (Redeem Rewards): Depends on US1 (enrollment) + US2 (points to redeem)
  - User Story 5 (Tiers): Depends on US1 (enrollment) + US2 (points for tier evaluation)
  - User Story 6 (Admin): Can start after Foundational (independent admin functions)
- **Error Testing (Phase 9)**: Depends on all user stories being complete
- **Polish (Phase 10)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: REQUIRED - No dependencies on other stories (base functionality)
- **User Story 2 (P1)**: REQUIRED - No dependencies (can work independently)
- **User Story 3 (P2)**: Soft dependency on US1+US2 (needs enrollment and transactions to view)
- **User Story 4 (P2)**: Soft dependency on US1+US2 (needs enrollment and points to redeem)
- **User Story 5 (P3)**: Soft dependency on US1+US2 (needs enrollment and point earning for tier progression)
- **User Story 6 (P3)**: Independent admin functions (can be developed in parallel)

### Within Each User Story

- Integration tests MUST be written and FAIL before implementation
- Models before services (data layer first)
- Services before handlers (business logic before HTTP)
- Handlers registered in main.go last (wiring)
- Story complete and independently testable before moving to next priority

### Parallel Opportunities

- **Phase 1 (Setup)**: All tasks marked [P] can run in parallel (T002-T010)
- **Phase 2 (Foundational)**: Database models (T011-T016), protobuf generation (T018-T021), middleware (T026-T030) can run in parallel
- **User Stories**: Once Foundational completes:
  - US1 and US2 can be developed in parallel (both P1, no dependencies)
  - US3, US4, US5 can be developed in parallel after US1+US2 complete
  - US6 can be developed in parallel with any other story (independent admin functions)
- **Within User Story**: Models marked [P] can be created in parallel

---

## Parallel Example: User Story 2 (Earn Points)

```bash
# After Foundational phase completes:

# Developer A works on User Story 1 (Enrollment)
Task T037: Write enrollment integration tests
Task T039-T045: Implement enrollment service and handlers

# Developer B works on User Story 2 (Earn Points) in parallel
Task T047: Write earn points integration tests
Task T048-T053: Implement transaction service and handlers

# Both stories are independently testable and deployable
```

---

## Implementation Strategy

### MVP First (User Stories 1 & 2 Only)

1. Complete Phase 1: Setup (T001-T010)
2. Complete Phase 2: Foundational (T011-T036) - CRITICAL BLOCKER
3. Complete Phase 3: User Story 1 - Enrollment (T037-T046)
4. Complete Phase 4: User Story 2 - Earn Points (T047-T053)
5. **STOP and VALIDATE**: Test US1+US2 independently
6. **Deploy/Demo MVP**: Customers can enroll and earn points!

**MVP Scope**: Customers can join program and earn points from purchases. This is the minimum viable loyalty system.

### Incremental Delivery

1. Foundation (Phase 1+2) → Infrastructure ready ✓
2. Add US1 (Enrollment) → Customers can join ✓
3. Add US2 (Earn Points) → Customers earn rewards ✓ → **Deploy MVP**
4. Add US3 (View History) → Customers see activity ✓ → **Deploy v1.1**
5. Add US4 (Redeem Rewards) → Customers use points ✓ → **Deploy v1.2**
6. Add US5 (Tiers) → Gamification & engagement ✓ → **Deploy v1.3**
7. Add US6 (Admin) → Full program management ✓ → **Deploy v2.0**

Each deployment adds value without breaking previous functionality.

### Parallel Team Strategy

With 3 developers after Foundational phase:

1. **Team completes Setup + Foundational together** (1-2 days)
2. **Parallel development phase**:
   - Developer A: User Story 1 (Enrollment)
   - Developer B: User Story 2 (Earn Points)
   - Developer C: User Story 6 (Admin - independent)
3. **Integration phase**: US3, US4, US5 (require US1+US2 as foundation)
4. **Testing phase**: Comprehensive error testing (Phase 9)
5. **Polish phase**: Documentation, deployment, optimization

---

## Notes

- [P] tasks = different files, no dependencies, can run in parallel
- [Story] label maps task to specific user story (US1, US2, etc.)
- Each user story should be independently completable and testable
- **CRITICAL**: Write tests FIRST, ensure they FAIL, then implement
- **⚠️ PRINCIPLE XI**: Run tests after EVERY code change - tasks NOT complete until tests pass
- Use real PostgreSQL database for all tests (testcontainers-go)
- Use protobuf structs (NOT maps) for all API contracts
- Use `cmp.Diff()` with `protocmp.Transform()` for ALL protobuf assertions
- Use database truncation for test cleanup (defer truncateTables pattern)
- All services MUST accept context.Context as first parameter
- All errors MUST be wrapped with `fmt.Errorf("%w", err)` for error chains
- All error checking MUST use `errors.Is()` and `errors.As()`
- **ALL sentinel errors and HTTP error codes MUST be tested** (Phase 9)
- **Run `go test -v ./...` after each implementation task** (Principle XI)
- **Fix test failures immediately before proceeding** (Principle XI)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Each user story is a potentially shippable increment

---

## Summary

- **Total Tasks**: 125 tasks (updated with Principle XI test verification tasks)
- **Setup Phase**: 10 tasks (parallel opportunities: 8 tasks)
- **Foundational Phase**: 26 tasks (BLOCKING - must complete before user stories)
- **User Story 1 (Enrollment)**: 11 tasks (P1 - MVP critical) [includes T046a test verification]
- **User Story 2 (Earn Points)**: 8 tasks (P1 - MVP critical) [includes T053a test verification]
- **User Story 3 (View History)**: 6 tasks (P2) [includes T058a test verification]
- **User Story 4 (Redeem Rewards)**: 12 tasks (P2) [includes T069a test verification]
- **User Story 5 (Tiers)**: 11 tasks (P3) [includes T079a test verification]
- **User Story 6 (Admin)**: 21 tasks (P3) [includes T099a test verification]
- **Error Testing Phase**: 7 tasks (MANDATORY) [includes T105a test verification]
- **Polish Phase**: 14 tasks [includes T119 final test verification]

**Constitution**: Version 1.1.0 with Principle XI (Continuous Test Verification)
**Test Verification Tasks**: 7 new tasks (T046a, T053a, T058a, T069a, T079a, T105a, T119)

**MVP Scope**: User Stories 1 + 2 (28 tasks after Foundational) - Enrollment and point earning
**Full Feature**: All user stories (96 tasks after Foundational)

**Parallel Opportunities**: Foundational phase has 15+ parallelizable tasks. After Foundational, US1 and US2 can be developed in parallel, followed by US3/US4/US5 in parallel.

**Independent Test Criteria**:
- US1: New customer enrolls → receives membership number → balance is zero
- US2: Enrolled customer makes purchase → points credited → balance updated
- US3: Enrolled customer with history → can list transactions → filtered correctly
- US4: Customer with points → redeems reward → receives code → balance decreased
- US5: Customer earning points → crosses tier threshold → automatically upgraded
- US6: Admin creates campaign → campaign applies to purchases → analytics shows metrics

**Constitutional Compliance**: All tasks follow TDD approach with integration tests first, use real PostgreSQL, protobuf contracts, service layer architecture, comprehensive error handling, and OpenTracing instrumentation.

**Principle XI - Continuous Test Verification**: After EVERY implementation task, run `go test -v ./...` and verify all tests pass before proceeding. This is NON-NEGOTIABLE - tasks are incomplete if tests fail. Test verification tasks (T046a, T053a, T058a, T069a, T079a, T099a, T105a, T119) enforce this discipline at checkpoints.

## Test Verification Workflow (Principle XI)

**After Each Implementation Task**:
1. Run tests: `go test -v ./...`
2. Check for failures
3. If failures exist: FIX IMMEDIATELY (do not proceed)
4. If all pass: Proceed to next task
5. Commit after verification passes

**At Each Checkpoint** (end of user story):
1. Run full test suite: `go test -v ./...`
2. Run with race detector: `go test -v -race ./...`
3. Verify build: `go build ./...`
4. Check coverage: `go test -cover ./...`
5. Run regression checks (verify previous stories still pass)
6. Document test results
7. ALL tests MUST pass before marking story complete

**Before Feature Complete**:
1. Execute comprehensive test suite (T119)
2. Run tests 3 times to check for flakiness
3. Verify 100% pass rate
4. Verify no skipped tests
5. Document final test status

**Failure Handling**:
- Flaky tests MUST be fixed (not ignored or re-run)
- Test failures BLOCK all subsequent work
- Debug and fix immediately
- Re-run tests to verify fix
- Only proceed after clean test run

