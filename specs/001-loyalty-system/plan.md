# Implementation Plan: Customer Loyalty System

**Branch**: `001-loyalty-system` | **Date**: November 20, 2025 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-loyalty-system/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Build a customer loyalty system that enables customers to enroll in a rewards program, earn points through purchases and activities, view their balance and transaction history, redeem rewards, and progress through membership tiers. The system will support administrative management of program rules, promotions, and analytics. Implementation will use Go with PostgreSQL, following strict integration testing principles with real database fixtures, protobuf API contracts, and service layer architecture for reusability.

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: Go 1.21+ (latest stable recommended)  
**HTTP Framework**: Standard library `net/http` with `http.ServeMux` (MANDATORY per constitution - NO external routers)  
**Database**: PostgreSQL 15+ with JSONB support (MANDATORY per constitution)  
**Database Access**: GORM (gorm.io/gorm with gorm.io/driver/postgres) (MANDATORY per constitution)  
**Distributed Tracing**: OpenTracing (github.com/opentracing/opentracing-go) (MANDATORY per constitution)  
**Protocol Buffers**: protoc compiler, protoc-gen-go for API contracts (MANDATORY per constitution)  
**Testing**: Standard library `testing` with `httptest`, testcontainers-go for PostgreSQL (MANDATORY per constitution)  
**Test Comparison**: google/go-cmp with protocmp for protobuf assertions (MANDATORY per constitution)  
**Error Handling**: Standard library fmt.Errorf with %w for wrapping, errors.Is/As for checking (MANDATORY per constitution)  
**Error Testing**: ALL sentinel errors and HTTP error codes MUST be tested (MANDATORY per constitution Principle IX)  
**Context Propagation**: All service methods MUST accept context.Context as first parameter (MANDATORY per constitution)  
**Service Architecture**: Services in public `services/` package (NOT internal/) for external reusability (MANDATORY per constitution Principle VIII)  
**Target Platform**: Docker containerization with platform-agnostic deployment (AWS/GCP/Azure/Kubernetes)  
**Project Type**: Backend API service (Go) with RESTful HTTP endpoints  
**Performance Goals**: Based on success criteria - point balance updates within 5 seconds, dashboard queries under 2 seconds, support 10,000 enrolled members with concurrent transactions  
**Constraints**: 99.9% accuracy between transactions and balances (SC-004), zero concurrency-related discrepancies (SC-007), point expiration batch processing within 1 hour (SC-011)  
**Scale/Scope**: 10,000+ enrolled members, handle transaction volume spikes during promotions, support concurrent point earning/redemption transactions

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle I: Integration Testing First (No Mocking)
- ✅ **PASS**: All tests will use real PostgreSQL via testcontainers-go
- ✅ **PASS**: No mocking of database, HTTP clients, or external services
- ✅ **PASS**: Tests will use real database fixtures with GORM

### Principle II: Table-Driven Test Design
- ✅ **PASS**: All tests will follow table-driven pattern with test case structs
- ✅ **PASS**: Shared setup/teardown extracted to helper functions

### Principle III: Edge Case Coverage (NON-NEGOTIABLE)
- ✅ **PASS**: Tests will cover input validation (empty strings, nil values, SQL injection, XSS)
- ✅ **PASS**: Tests will cover boundary conditions (zero balances, exact matches, negative values)
- ✅ **PASS**: Tests will cover authentication/authorization edge cases
- ✅ **PASS**: Tests will cover concurrency scenarios (simultaneous redemptions, balance updates)

### Principle IV: Real Database Fixtures
- ✅ **PASS**: Fixtures inserted using GORM to real test database
- ✅ **PASS**: Database truncation used for test isolation with defer pattern

### Principle V: ServeHTTP Endpoint Testing
- ✅ **PASS**: Tests will use httptest.ResponseRecorder and httptest.NewRequest
- ✅ **PASS**: Full HTTP stack validation including middleware

### Principle VI: Protobuf Data Structures
- ✅ **PASS**: All API request/response types defined in .proto files
- ✅ **PASS**: Tests will use protobuf structs (NO map[string]interface{})
- ✅ **PASS**: Tests will use cmp.Diff() with protocmp.Transform() for assertions

### Principle VII: Distributed Tracing (OpenTracing)
- ✅ **PASS**: All endpoints will be instrumented with OpenTracing spans
- ✅ **PASS**: Service operations will create child spans
- ✅ **PASS**: Database operations traced at transaction level (NOT per query)

### Principle VIII: Service Layer Architecture (Dependency Injection)
- ✅ **PASS**: Services in public services/ package (NOT internal/)
- ✅ **PASS**: Services will NOT depend on HTTP types
- ✅ **PASS**: Services will return protobuf types
- ✅ **PASS**: AutoMigrate() function will be exported for external apps

### Principle IX: Comprehensive Error Handling
- ✅ **PASS**: Sentinel errors defined for domain-specific errors
- ✅ **PASS**: HTTP error codes using singleton struct instances
- ✅ **PASS**: ErrorCode.ServiceErr field for automatic error mapping
- ✅ **PASS**: ALL sentinel errors and HTTP error codes will be tested

### Principle X: Context-Aware Operations
- ✅ **PASS**: All service methods will accept context.Context as first parameter
- ✅ **PASS**: Database operations will use db.WithContext(ctx)
- ✅ **PASS**: Context cancellation will be tested

**Overall Status**: ✅ **ALL GATES PASS** - Proceeding to Phase 0

---

**Post-Design Re-Evaluation (Phase 1 Complete)**:
- ✅ Data model follows event-sourced pattern with immutable transactions
- ✅ API contracts defined in protobuf (loyalty.proto, transaction.proto, reward.proto, campaign.proto)
- ✅ Service architecture maintains separation from HTTP layer
- ✅ Error handling strategy defined with sentinel errors and HTTP error code mapping
- ✅ All constitutional principles remain satisfied
- ✅ Research resolved all technical unknowns
- ✅ Design supports all functional requirements and success criteria

**Final Status**: ✅ **ALL GATES PASS** - Ready for implementation (Phase 2: Tasks)

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
# Loyalty System Project Structure
├── .specify/              # Spec-kit configuration
│   ├── memory/
│   │   └── constitution.md
│   ├── scripts/
│   └── templates/
├── services/              # PUBLIC - reusable business logic
│   ├── loyalty_service.go      # Customer enrollment, points, tiers
│   ├── transaction_service.go  # Point earning/redemption tracking
│   ├── reward_service.go       # Reward catalog management
│   ├── campaign_service.go     # Promotional campaigns
│   ├── errors.go               # Sentinel errors
│   └── migrations.go           # AutoMigrate for external apps
├── handlers/              # PUBLIC - HTTP handlers
│   ├── enrollment_handler.go   # Customer enrollment endpoints
│   ├── points_handler.go       # Earn/redeem points endpoints
│   ├── rewards_handler.go      # Reward catalog endpoints
│   ├── campaign_handler.go     # Campaign management endpoints
│   ├── admin_handler.go        # Administrative endpoints
│   └── error_codes.go          # HTTP error singleton
├── api/                   # PUBLIC - protobuf definitions and generated code
│   ├── v1/
│   │   ├── loyalty.proto       # Enrollment, membership
│   │   ├── transaction.proto   # Point transactions
│   │   ├── reward.proto        # Rewards
│   │   └── campaign.proto      # Campaigns
│   └── gen/v1/
│       ├── loyalty.pb.go
│       ├── transaction.pb.go
│       ├── reward.pb.go
│       └── campaign.pb.go
├── internal/              # INTERNAL - implementation details
│   ├── models/            # GORM models (not exposed)
│   │   ├── customer.go
│   │   ├── transaction.go
│   │   ├── reward.go
│   │   ├── tier.go
│   │   ├── campaign.go
│   │   └── redemption.go
│   ├── middleware/        # HTTP middleware
│   │   ├── auth.go
│   │   ├── tracing.go
│   │   └── logging.go
│   └── config/            # Configuration
│       └── config.go
├── cmd/                   # Application entry points
│   └── api/
│       └── main.go
└── tests/
    └── integration/
        ├── enrollment_test.go
        ├── earn_points_test.go
        ├── redeem_points_test.go
        ├── tiers_test.go
        ├── campaigns_test.go
        └── helpers.go
```

**Structure Decision**: Service layer organized by domain concepts (loyalty, transaction, reward, campaign) rather than single monolithic service. This supports independent testing and future microservice extraction if needed.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No violations - all constitutional principles will be followed.
