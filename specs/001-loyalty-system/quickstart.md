# Quickstart Guide: Customer Loyalty System

**Date**: November 20, 2025  
**Phase**: Phase 1 - Design & Contracts  
**Audience**: Developers implementing or integrating with the loyalty system

## Overview

This guide helps you get started with the Customer Loyalty System, whether you're implementing the service or integrating as a client.

## Prerequisites

### For Implementation
- Go 1.21+ installed
- Docker and Docker Compose (for PostgreSQL test database)
- Protocol Buffers compiler (`protoc`) with Go plugins
- Git for version control

### For Client Integration
- Access to loyalty API (URL and authentication token)
- HTTP client (curl, Postman, or SDK)
- Understanding of REST APIs and JSON

## Quick Setup (Implementation)

### 1. Clone Repository

```bash
git clone <repository-url>
cd loyalty-demo
git checkout 001-loyalty-system
```

### 2. Install Dependencies

```bash
# Install Go dependencies
go mod download

# Install protoc plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### 3. Start Local Database

```bash
# Start PostgreSQL via Docker Compose
docker-compose up -d postgres

# Verify database is running
docker-compose ps
```

### 4. Generate Protobuf Code

```bash
# Generate Go code from .proto files
protoc --go_out=. --go_opt=paths=source_relative \
  api/v1/*.proto
```

### 5. Run Migrations

```bash
# Run database migrations
go run cmd/migrate/main.go

# Or use GORM AutoMigrate in application startup
```

### 6. Start Application

```bash
# Development mode
go run cmd/api/main.go

# Application starts on http://localhost:8080
```

### 7. Verify Health

```bash
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy","version":"1.0.0","dependencies":{"database":"healthy"}}
```

## Quick Setup (Client Integration)

### 1. Obtain API Credentials

Contact your administrator to get:
- API base URL (e.g., `https://api.example.com/v1`)
- Authentication token or API key

### 2. Test Connection

```bash
export LOYALTY_API_URL="https://api.example.com/v1"
export LOYALTY_TOKEN="your-jwt-token"

curl -H "Authorization: Bearer $LOYALTY_TOKEN" \
  $LOYALTY_API_URL/loyalty/me
```

### 3. Install SDK (Optional)

```bash
# If Go SDK is available
go get github.com/yourorg/loyalty-sdk-go

# If JavaScript SDK is available
npm install @yourorg/loyalty-sdk
```

## Common Workflows

### Workflow 1: Customer Enrollment

**Scenario**: A new customer signs up and wants to join the loyalty program.

```bash
# Step 1: Customer enrolls (optionally with referral code)
curl -X POST $LOYALTY_API_URL/loyalty/enroll \
  -H "Authorization: Bearer $LOYALTY_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "referral_code": "FRIEND123"
  }'

# Response:
{
  "customer": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "membership_number": "LM-20251120-A8K9",
    "referral_code": "XYZ789",
    "referred_by": "FRIEND123",
    "enrolled_at": "2025-11-20T10:30:00Z",
    "current_balance": 0,
    "tier": {
      "name": "Base",
      "level": 0,
      "earn_rate_multiplier": 1.0
    }
  }
}
```

**Integration Points**:
- Call during user registration flow
- Store `membership_number` in your customer record
- Display `referral_code` to customer for sharing

### Workflow 2: Earning Points from Purchase

**Scenario**: Customer makes a purchase and earns loyalty points.

```bash
# Step 1: After order completion, credit points
curl -X POST $LOYALTY_API_URL/loyalty/points/earn \
  -H "Authorization: Bearer $LOYALTY_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 5000,
    "reference_id": "order-12345",
    "reference_type": "ORDER",
    "description": "Purchase at Main Street Store"
  }'

# Response:
{
  "transaction": {
    "id": "txn-abc123",
    "amount": 50,
    "type": "EARN",
    "reference_id": "order-12345",
    "description": "Purchase at Main Street Store",
    "created_at": "2025-11-20T11:15:00Z",
    "expires_at": "2026-11-20T11:15:00Z"
  },
  "new_balance": 50,
  "campaign_applied": {
    "name": "Holiday Double Points",
    "bonus_points": 50
  }
}
```

**Integration Points**:
- Call after successful payment processing
- Use order ID as `reference_id` for idempotency
- Amount in cents ($50.00 = 5000)
- Handle retries gracefully (idempotent)

### Workflow 3: Viewing Balance and History

**Scenario**: Customer checks their loyalty status and transaction history.

```bash
# Step 1: Get current status
curl -H "Authorization: Bearer $LOYALTY_TOKEN" \
  $LOYALTY_API_URL/loyalty/me

# Response:
{
  "customer": {
    "membership_number": "LM-20251120-A8K9",
    "current_balance": 1250,
    "tier": {
      "name": "Gold",
      "level": 2,
      "earn_rate_multiplier": 1.5
    },
    "points_to_next_tier": 250,
    "points_expiring_soon": {
      "amount": 100,
      "expiration_date": "2025-12-31T23:59:59Z"
    }
  }
}

# Step 2: Get transaction history
curl -H "Authorization: Bearer $LOYALTY_TOKEN" \
  "$LOYALTY_API_URL/loyalty/transactions?limit=10&type=EARN"

# Response:
{
  "transactions": [
    {
      "id": "txn-1",
      "amount": 50,
      "type": "EARN",
      "description": "Purchase at Main Street Store",
      "created_at": "2025-11-20T11:15:00Z"
    },
    ...
  ],
  "total": 127,
  "limit": 10,
  "offset": 0
}
```

**Integration Points**:
- Display balance in customer dashboard
- Show transaction history in loyalty section
- Alert customers about expiring points

### Workflow 4: Redeeming Rewards

**Scenario**: Customer uses points to get a discount code.

```bash
# Step 1: List available rewards
curl -H "Authorization: Bearer $LOYALTY_TOKEN" \
  $LOYALTY_API_URL/loyalty/rewards

# Response:
{
  "rewards": [
    {
      "id": "reward-1",
      "name": "$5 Discount Code",
      "description": "Get $5 off your next purchase",
      "type": "DISCOUNT",
      "point_cost": 500,
      "is_active": true
    }
  ]
}

# Step 2: Redeem reward
curl -X POST $LOYALTY_API_URL/loyalty/rewards/redeem \
  -H "Authorization: Bearer $LOYALTY_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "reward_id": "reward-1"
  }'

# Response:
{
  "redemption": {
    "id": "redemption-123",
    "reward": {
      "name": "$5 Discount Code",
      "type": "DISCOUNT",
      "point_cost": 500
    },
    "points_deducted": 500,
    "status": "ACTIVE",
    "code": "DISC-ABC123",
    "created_at": "2025-11-20T12:00:00Z"
  },
  "new_balance": 750
}

# Step 3: Apply discount code at checkout
# Customer enters code "DISC-ABC123" during checkout
# Your checkout system validates and applies $5 discount
```

**Integration Points**:
- Display rewards catalog in loyalty section
- Handle insufficient balance errors gracefully
- Store `code` for applying discount at checkout
- Validate codes in your checkout system
- Mark redemption as USED after order completion

### Workflow 5: Handling Refunds

**Scenario**: Customer returns an item; points and redemptions must be reversed.

```bash
# Step 1: Order is refunded in your system
# Step 2: Call loyalty API to adjust points

# If order earned points, they were already deducted during refund processing
# Points earned from order-12345 will be negative (deduction)

# If redemption was used, reverse it
curl -X POST $LOYALTY_API_URL/admin/loyalty/redemptions/redemption-123/reverse \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Order refunded by customer"
  }'

# Response:
{
  "redemption": {
    "id": "redemption-123",
    "status": "REVERSED",
    "reversed_at": "2025-11-20T15:00:00Z"
  },
  "points_restored": 500,
  "new_balance": 1250
}
```

**Integration Points**:
- Call when processing refunds
- Requires admin authentication
- Restores points to customer balance

## Testing

### Running Integration Tests

```bash
# Run all tests
go test ./tests/integration/...

# Run specific test
go test ./tests/integration -run TestEnrollment

# Run with verbose output
go test -v ./tests/integration/...

# Run with coverage
go test -cover ./tests/integration/...
```

### Manual API Testing

```bash
# Test enrollment
curl -X POST http://localhost:8080/v1/loyalty/enroll \
  -H "Authorization: Bearer test-token" \
  -H "Content-Type: application/json" \
  -d '{"referral_code": "TEST123"}'

# Test earning points
curl -X POST http://localhost:8080/v1/loyalty/points/earn \
  -H "Authorization: Bearer test-token" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 5000,
    "reference_id": "test-order-001",
    "reference_type": "ORDER",
    "description": "Test purchase"
  }'

# Test getting status
curl -H "Authorization: Bearer test-token" \
  http://localhost:8080/v1/loyalty/me
```

## Configuration

### Environment Variables

```bash
# Database
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=loyalty
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres
DATABASE_SSL_MODE=disable

# Server
SERVER_PORT=8080
SERVER_HOST=0.0.0.0

# Tracing
TRACING_ENABLED=true
TRACING_BACKEND=jaeger
JAEGER_AGENT_HOST=localhost
JAEGER_AGENT_PORT=6831

# Application
POINTS_EARN_RATE=1.0  # 1 point per dollar
POINTS_EXPIRATION_DAYS=365
TIER_EVALUATION_DAYS=365

# Security
JWT_SECRET=your-secret-key
ADMIN_EMAILS=admin@example.com,support@example.com
```

### Docker Compose Configuration

```yaml
# docker-compose.yml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: loyalty
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  api:
    build: .
    ports:
      - "8080:8080"
    environment:
      DATABASE_HOST: postgres
      DATABASE_PORT: 5432
      DATABASE_NAME: loyalty
      DATABASE_USER: postgres
      DATABASE_PASSWORD: postgres
    depends_on:
      - postgres

  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "6831:6831/udp"  # Agent
      - "16686:16686"    # UI
    environment:
      COLLECTOR_ZIPKIN_HOST_PORT: 9411

volumes:
  postgres_data:
```

## Common Issues

### Issue 1: Database Connection Fails

**Symptom**: `failed to connect to database: connection refused`

**Solution**:
```bash
# Verify PostgreSQL is running
docker-compose ps

# Check connection parameters
psql -h localhost -U postgres -d loyalty

# Restart PostgreSQL
docker-compose restart postgres
```

### Issue 2: Protobuf Generation Errors

**Symptom**: `protoc-gen-go: program not found`

**Solution**:
```bash
# Install missing plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# Verify installation
which protoc-gen-go

# Ensure GOPATH/bin is in PATH
export PATH="$PATH:$(go env GOPATH)/bin"
```

### Issue 3: Duplicate Transaction Error

**Symptom**: `409 Conflict: reference_id already exists`

**Solution**:
- This is expected behavior (idempotency)
- Use unique reference IDs for each transaction
- For retries, use same reference_ID to get existing transaction

### Issue 4: Insufficient Points for Redemption

**Symptom**: `400 Bad Request: insufficient balance`

**Solution**:
```bash
# Check customer balance first
curl -H "Authorization: Bearer $TOKEN" \
  $LOYALTY_API_URL/loyalty/me

# Response shows current_balance
# Ensure customer has enough points before redemption
```

## Development Tips

### 1. Use Table-Driven Tests

```go
func TestEarnPoints(t *testing.T) {
    testCases := []struct {
        name           string
        amount         int64
        referenceID    string
        expectedPoints int64
        expectError    bool
    }{
        {
            name:           "Valid purchase",
            amount:         5000,
            referenceID:    "order-1",
            expectedPoints: 50,
            expectError:    false,
        },
        {
            name:           "Negative amount",
            amount:         -100,
            referenceID:    "order-2",
            expectedPoints: 0,
            expectError:    true,
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### 2. Use Protobuf Comparison

```go
import (
    "github.com/google/go-cmp/cmp"
    "google.golang.org/protobuf/testing/protocmp"
)

// Compare protobuf messages
if diff := cmp.Diff(expected, actual, protocmp.Transform()); diff != "" {
    t.Errorf("Mismatch (-want +got):\n%s", diff)
}
```

### 3. Test Database Truncation

```go
func TestSomething(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()
    defer truncateTables(db, "customers", "point_transactions", "redemptions")
    
    // Your test code
}
```

## Next Steps

1. **Review Architecture**: Read `plan.md` for design decisions
2. **Understand Data Model**: Review `data-model.md` for entity relationships
3. **API Reference**: Consult `contracts/api-spec.md` for detailed endpoints
4. **Implement Services**: Start with `services/loyalty_service.go`
5. **Write Tests First**: Follow TDD workflow per constitution
6. **Add Handlers**: Create HTTP handlers after services are tested
7. **Deploy**: Use Docker Compose for staging, Kubernetes for production

## Support

- **Documentation**: See `/specs/001-loyalty-system/` directory
- **Constitution**: `.specify/memory/constitution.md` for development guidelines
- **Issues**: Submit to project issue tracker
- **Questions**: Contact development team

## Summary

This quickstart guide covers:
- ✅ Local development setup
- ✅ Common integration workflows
- ✅ Testing strategies
- ✅ Configuration options
- ✅ Troubleshooting common issues
- ✅ Development best practices

You're now ready to implement or integrate with the loyalty system!

