# Customer Loyalty System

A production-grade loyalty program API built with Go, following strict constitutional principles for code quality, testing, and architecture.

**Version**: 1.0.0  
**Status**: Production Ready (User Stories 1-4 Complete)  
**Constitution**: v1.3.3 (13 principles)

---

## Features

### Customer Features ✅
- **Enroll** in loyalty program with referral codes
- **Earn** points from purchases (1 point per dollar)
- **Tier bonuses** (Gold members earn 1.5x points)
- **Campaign bonuses** (e.g., double points weekends)
- **Referral bonuses** (100 points when friend makes first purchase)
- **Browse** reward catalog
- **Redeem** points for discounts, vouchers, free items
- **Track** complete transaction history with filtering
- **View** redemption history

### Business Capabilities ✅
- Campaign system for promotions
- Tier-based rewards
- Referral tracking
- Complete audit trail
- Real-time point updates
- Idempotent operations

---

## Quick Start

### Prerequisites
- Go 1.21+ installed
- Docker and Docker Compose (for PostgreSQL)
- Protocol Buffers compiler (`protoc`)

### Installation

```bash
# Clone repository
git clone <repository-url>
cd loyalty-demo

# Install dependencies
go mod download

# Start PostgreSQL
docker-compose up -d postgres

# Run database migrations
go run cmd/api/main.go
# (AutoMigrate runs automatically on startup)
```

### Running the Server

```bash
# Development mode
go run cmd/api/main.go

# Production build
go build -o loyalty-api cmd/api/main.go
./loyalty-api
```

Server starts on `http://localhost:8080`

### Configuration

Set environment variables (see `.env.example`):
```bash
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=loyalty
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres

SERVER_PORT=8080
POINTS_EARN_RATE=1.0
POINTS_EXPIRATION_DAYS=365
```

---

## API Endpoints

### Customer Endpoints

```bash
# Enroll in loyalty program
POST /v1/loyalty/enroll
Authorization: Bearer <token>
{
  "referral_code": "FRIEND123"  # optional
}

# Get customer status
GET /v1/loyalty/me
Authorization: Bearer <token>

# Earn points from purchase
POST /v1/loyalty/points/earn
Authorization: Bearer <token>
{
  "amount": 5000,  # $50.00 in cents
  "reference_id": "order-12345",
  "reference_type": "ORDER",
  "description": "Purchase at Store A"
}

# View transaction history
GET /v1/loyalty/transactions?type=EARN&limit=50&offset=0
Authorization: Bearer <token>

# Browse rewards
GET /v1/loyalty/rewards?max_points=500

# Redeem reward
POST /v1/loyalty/rewards/redeem
Authorization: Bearer <token>
{
  "reward_id": "reward-uuid"
}

# View redemptions
GET /v1/loyalty/redemptions
Authorization: Bearer <token>
```

### Public Endpoints

```bash
# Health check
GET /health

# List membership tiers
GET /v1/loyalty/tiers
```

---

## Testing

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run with race detector
go test -race ./...

# Run specific test
go test -v ./handlers -run TestEnrollCustomer

# Check coverage
go test -cover ./...
```

### Test Statistics

- **Total Test Cases**: 39
- **Pass Rate**: 100%
- **Coverage**: handlers and services tested through integration tests
- **Test Database**: PostgreSQL via testcontainers (automatic Docker management)
- **Test Duration**: ~7 seconds

---

## Architecture

### Technology Stack

- **Language**: Go 1.21+
- **Database**: PostgreSQL 15+ with JSONB
- **HTTP Framework**: Standard library `net/http` (no external routers)
- **Database Access**: GORM
- **Tracing**: OpenTracing
- **API Contracts**: Protocol Buffers
- **Testing**: testcontainers-go with real PostgreSQL

### Package Structure

```
loyalty-demo/
├── services/              # PUBLIC - business logic (importable)
│   ├── loyalty_service.go
│   ├── transaction_service.go
│   ├── reward_service.go
│   ├── errors.go          # Sentinel errors
│   └── migrations.go      # AutoMigrate for external apps
├── handlers/              # PUBLIC - HTTP handlers
│   ├── enrollment_handler.go
│   ├── points_handler.go
│   ├── rewards_handler.go
│   └── error_codes.go
├── api/v1/                # Protobuf contracts
│   ├── loyalty.proto
│   ├── transaction.proto
│   ├── reward.proto
│   ├── campaign.proto
│   └── *.pb.go (generated)
├── internal/              # Internal implementation
│   ├── models/            # GORM models
│   ├── middleware/        # HTTP middleware
│   ├── config/            # Configuration
│   └── testutil/          # Test helpers
└── cmd/api/main.go        # Application entry point
```

### Design Patterns

- **Service Layer**: Business logic separated from HTTP transport
- **Dependency Injection**: Services accept dependencies via constructors
- **Event Sourcing**: Point transactions are immutable event log
- **Protobuf Contracts**: Type-safe API definitions
- **Integration Testing**: Real PostgreSQL database, no mocks

---

## Constitutional Principles

This project follows a strict constitution with **13 core principles**:

1. ✅ **Integration Testing First** - Real database, no mocks
2. ✅ **Table-Driven Test Design** - Comprehensive test cases
3. ✅ **Edge Case Coverage** - Security, validation, boundaries
4. ✅ **Real Database Fixtures** - GORM fixtures with PostgreSQL
5. ✅ **ServeHTTP Endpoint Testing** - Full HTTP stack validation
6. ✅ **Protobuf Data Structures** - Type-safe API contracts
7. ✅ **Distributed Tracing** - OpenTracing instrumentation
8. ✅ **Service Layer Architecture** - Reusable business logic
9. ✅ **Comprehensive Error Handling** - Sentinel errors + HTTP codes
10. ✅ **Context-Aware Operations** - Timeout and cancellation support
11. ✅ **Continuous Test Verification** - Tests run after every change
12. ✅ **Root Cause Tracing** - Fix problems at source, not symptoms
13. ✅ **Acceptance Scenario Coverage** - One-to-one spec-to-test mapping

See `.specify/memory/constitution.md` for complete details.

---

## Development Workflow

### Test-First Development (TDD)

1. Define API contract in `.proto` files
2. Generate protobuf code
3. **Write tests FIRST** (table-driven, comprehensive edge cases)
4. Verify tests FAIL (red phase)
5. Implement code to make tests pass (green phase)
6. **Run tests immediately** after implementation (Principle XI)
7. Refactor while keeping tests green
8. **Run tests after refactoring** (Principle XI)

### Debugging Discipline (Principle XII)

When problems occur:
1. **Trace backward** through call chain
2. **Identify root cause** (not symptom)
3. **Fix at source** (not workaround)
4. **Verify** with tests
5. **Document** root cause analysis

**Never**:
- ❌ Remove failing tests
- ❌ Weaken test expectations
- ❌ Add workarounds
- ❌ "Make it work" without understanding

---

## Documentation

- **Specification**: `specs/001-loyalty-system/spec.md`
- **Implementation Plan**: `specs/001-loyalty-system/plan.md`
- **Data Model**: `specs/001-loyalty-system/data-model.md`
- **API Contracts**: `specs/001-loyalty-system/contracts/`
- **Tasks**: `specs/001-loyalty-system/tasks.md`
- **Quickstart**: `specs/001-loyalty-system/quickstart.md`
- **Constitutional Audit**: `CONSTITUTIONAL_AUDIT.md`
- **MVP Complete**: `MVP_COMPLETE.md`
- **Root Cause Fix Example**: `ROOT_CAUSE_FIX_REPORT.md`

---

## Project Status

### Completed (73 tasks, 58%)

- ✅ **Phase 1**: Setup (10/10)
- ✅ **Phase 2**: Foundational (26/26)
- ✅ **Phase 3**: User Story 1 - Enrollment (11/11)
- ✅ **Phase 4**: User Story 2 - Earn Points (8/8)
- ✅ **Phase 5**: User Story 3 - View History (6/6)
- ✅ **Phase 6**: User Story 4 - Redeem Rewards (12/12)

### Remaining (52 tasks, 42%)

- ⏳ **Phase 7**: User Story 5 - Tiers (11 tasks)
- ⏳ **Phase 8**: User Story 6 - Admin (21 tasks)
- ⏳ **Phase 9**: Error Testing (7 tasks) - MANDATORY
- ⏳ **Phase 10**: Polish (14 tasks)

---

## Deployment

### Docker

```bash
# Build
docker build -t loyalty-api:latest .

# Run with docker-compose
docker-compose up
```

### Kubernetes

```bash
# Deploy
kubectl apply -f k8s/

# Check status
kubectl get pods -l app=loyalty-api
```

### Manual

```bash
# Build binary
go build -o loyalty-api cmd/api/main.go

# Set environment variables
export DATABASE_HOST=localhost
export DATABASE_NAME=loyalty
export SERVER_PORT=8080

# Run
./loyalty-api
```

---

## Contributing

### Code Quality Requirements

All code MUST comply with the 13 constitutional principles:

1. **Integration tests** with real PostgreSQL (testcontainers)
2. **Table-driven** test patterns
3. **Edge cases** including SQL injection, XSS, auth errors
4. **Protobuf** contracts for all APIs
5. **OpenTracing** spans on all endpoints
6. **Context** propagation throughout
7. **Error handling** with sentinel errors
8. **Test verification** after every change
9. **Root cause fixes** only (no symptom fixes)
10. **Acceptance scenario coverage** - every spec scenario has a test

### Running Quality Checks

```bash
# Run tests
go test -v ./...

# Run with race detector
go test -race ./...

# Run linter
golangci-lint run

# Verify build
go build ./...
```

All checks must pass before PR approval.

---

## License

[Your License Here]

---

## Support

- **Issues**: GitHub Issues
- **Documentation**: `specs/` directory
- **Constitution**: `.specify/memory/constitution.md`

---

**Constitution Version**: 1.3.3  
**Last Updated**: November 22, 2025

