# API Specification: Customer Loyalty System

**Version**: v1  
**Date**: November 20, 2025  
**Protocol**: REST over HTTP/JSON  
**Data Format**: Protocol Buffers (JSON encoding over HTTP)

## Overview

This document defines the HTTP API contract for the loyalty system. All request/response types are defined in protobuf (`.proto` files) and use JSON encoding over HTTP.

## Base URL

```
Production: https://api.example.com/v1
Development: http://localhost:8080/v1
```

## Authentication

All endpoints (except health check) require authentication:
- Header: `Authorization: Bearer <token>`
- Token validation handled by authentication middleware
- User ID extracted from token and passed to services via context

## Common Headers

### Request Headers
```
Authorization: Bearer <jwt_token>
Content-Type: application/json
X-Request-ID: <uuid>  (optional, for tracing)
```

### Response Headers
```
Content-Type: application/json
X-Request-ID: <uuid>  (echoed from request or generated)
X-Trace-ID: <trace_id>  (OpenTracing trace ID)
```

## Error Response Format

All errors follow this structure (defined in protobuf):

```json
{
  "code": "ERROR_CODE",
  "message": "Human-readable error message",
  "details": {
    "field": "specific_field",
    "constraint": "validation rule violated"
  }
}
```

**HTTP Status Codes**:
- `400 Bad Request`: Validation errors, malformed input
- `401 Unauthorized`: Missing or invalid authentication
- `403 Forbidden`: Insufficient permissions (admin endpoints)
- `404 Not Found`: Resource does not exist
- `409 Conflict`: Duplicate enrollment, SKU already exists
- `500 Internal Server Error`: Server-side failures

## Endpoints

### 1. Customer Enrollment

**Endpoint**: `POST /v1/loyalty/enroll`

**Description**: Enroll a customer in the loyalty program

**Request** (`EnrollCustomerRequest`):
```json
{
  "referral_code": "ABC123XYZ"  // Optional
}
```

**Response** (`EnrollCustomerResponse` - 201 Created):
```json
{
  "customer": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "account_id": "user123",
    "membership_number": "LM-20251120-A8K9",
    "referral_code": "XYZ789ABC",
    "referred_by": "ABC123XYZ",
    "enrolled_at": "2025-11-20T10:30:00Z",
    "current_balance": 0,
    "tier": {
      "id": "base-tier-id",
      "name": "Base",
      "level": 0,
      "earn_rate_multiplier": 1.0
    }
  }
}
```

**User Stories**: US-1  
**Requirements**: FR-001, FR-002, FR-016  
**Success Criteria**: SC-001 (enrollment under 60 seconds)

**Edge Cases**:
- Already enrolled (409 Conflict)
- Invalid referral code (400 Bad Request)
- Missing authentication (401 Unauthorized)

---

### 2. Get Customer Status

**Endpoint**: `GET /v1/loyalty/me`

**Description**: Retrieve current customer's loyalty account details

**Request**: None (customer identified by auth token)

**Response** (`GetCustomerResponse` - 200 OK):
```json
{
  "customer": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "membership_number": "LM-20251120-A8K9",
    "referral_code": "XYZ789ABC",
    "enrolled_at": "2025-11-20T10:30:00Z",
    "current_balance": 1250,
    "tier": {
      "id": "gold-tier-id",
      "name": "Gold",
      "level": 2,
      "earn_rate_multiplier": 1.5,
      "qualification_points": 1000
    },
    "points_to_next_tier": 250,
    "points_expiring_soon": {
      "amount": 100,
      "expiration_date": "2025-12-31T23:59:59Z"
    }
  }
}
```

**User Stories**: US-3  
**Requirements**: FR-018, FR-019  
**Success Criteria**: SC-005 (query under 2 seconds)

**Edge Cases**:
- Not enrolled (404 Not Found)
- Authentication expired (401 Unauthorized)

---

### 3. Earn Points

**Endpoint**: `POST /v1/loyalty/points/earn`

**Description**: Credit points for a qualifying activity (purchase, referral, promotion)

**Request** (`EarnPointsRequest`):
```json
{
  "amount": 1050,  // Transaction amount in cents
  "reference_id": "order-12345",
  "reference_type": "ORDER",
  "description": "Purchase at Store A"
}
```

**Response** (`EarnPointsResponse` - 201 Created):
```json
{
  "transaction": {
    "id": "txn-550e8400-e29b-41d4-a716-446655440000",
    "customer_id": "550e8400-e29b-41d4-a716-446655440000",
    "amount": 10,  // Points earned (1050 cents = $10.50 -> 10 points)
    "type": "EARN",
    "reference_id": "order-12345",
    "reference_type": "ORDER",
    "description": "Purchase at Store A",
    "created_at": "2025-11-20T11:15:00Z",
    "expires_at": "2026-11-20T11:15:00Z"
  },
  "new_balance": 1260,
  "campaign_applied": {
    "id": "campaign-id",
    "name": "Double Points Weekend",
    "bonus_points": 10
  }
}
```

**User Stories**: US-2  
**Requirements**: FR-003, FR-017, FR-021  
**Success Criteria**: SC-002 (balance updated within 5 seconds)

**Edge Cases**:
- Duplicate reference_id (idempotent, returns existing transaction)
- Negative amount (400 Bad Request)
- Invalid reference_type (400 Bad Request)
- Customer not enrolled (404 Not Found)

---

### 4. Get Transaction History

**Endpoint**: `GET /v1/loyalty/transactions?limit=50&offset=0&type=EARN`

**Description**: List customer's point transaction history

**Query Parameters**:
- `limit` (int, default: 50, max: 100): Number of transactions to return
- `offset` (int, default: 0): Pagination offset
- `type` (string, optional): Filter by transaction type (EARN, REDEMPTION, ADJUSTMENT, REFERRAL, EXPIRATION)
- `start_date` (ISO8601, optional): Filter transactions after this date
- `end_date` (ISO8601, optional): Filter transactions before this date

**Response** (`ListTransactionsResponse` - 200 OK):
```json
{
  "transactions": [
    {
      "id": "txn-1",
      "customer_id": "customer-id",
      "amount": 10,
      "type": "EARN",
      "reference_id": "order-12345",
      "reference_type": "ORDER",
      "description": "Purchase at Store A",
      "campaign_id": "campaign-1",
      "created_at": "2025-11-20T11:15:00Z",
      "expires_at": "2026-11-20T11:15:00Z"
    },
    {
      "id": "txn-2",
      "customer_id": "customer-id",
      "amount": -500,
      "type": "REDEMPTION",
      "description": "Redeemed: $5 discount code",
      "redemption_id": "redemption-id",
      "created_at": "2025-11-19T14:30:00Z"
    }
  ],
  "total": 127,
  "limit": 50,
  "offset": 0
}
```

**User Stories**: US-3  
**Requirements**: FR-005, FR-018  
**Success Criteria**: SC-005 (query under 2 seconds)

**Edge Cases**:
- Invalid date range (400 Bad Request)
- Limit exceeds maximum (400 Bad Request)
- Empty result set (200 OK with empty array)

---

### 5. List Available Rewards

**Endpoint**: `GET /v1/loyalty/rewards?active_only=true`

**Description**: Get catalog of redeemable rewards

**Query Parameters**:
- `active_only` (bool, default: true): Filter to active rewards only
- `max_points` (int, optional): Filter rewards by maximum point cost

**Response** (`ListRewardsResponse` - 200 OK):
```json
{
  "rewards": [
    {
      "id": "reward-1",
      "name": "$5 Discount Code",
      "description": "Get $5 off your next purchase",
      "type": "DISCOUNT",
      "point_cost": 500,
      "is_active": true,
      "metadata": {
        "discount_percentage": 0,
        "discount_amount": 500,
        "min_purchase": 1000
      }
    },
    {
      "id": "reward-2",
      "name": "Free Coffee",
      "description": "Redeem for one free coffee",
      "type": "FREE_ITEM",
      "point_cost": 100,
      "is_active": true,
      "metadata": {
        "product_id": "coffee-001"
      }
    }
  ]
}
```

**User Stories**: US-4  
**Requirements**: FR-008  

**Edge Cases**:
- No active rewards (200 OK with empty array)

---

### 6. Redeem Reward

**Endpoint**: `POST /v1/loyalty/rewards/redeem`

**Description**: Redeem points for a reward

**Request** (`RedeemRewardRequest`):
```json
{
  "reward_id": "reward-1"
}
```

**Response** (`RedeemRewardResponse` - 201 Created):
```json
{
  "redemption": {
    "id": "redemption-id",
    "customer_id": "customer-id",
    "reward": {
      "id": "reward-1",
      "name": "$5 Discount Code",
      "type": "DISCOUNT",
      "point_cost": 500
    },
    "points_deducted": 500,
    "status": "ACTIVE",
    "code": "DISC-ABC123",
    "created_at": "2025-11-20T12:00:00Z"
  },
  "new_balance": 750,
  "transaction": {
    "id": "txn-id",
    "amount": -500,
    "type": "REDEMPTION",
    "description": "Redeemed: $5 Discount Code",
    "created_at": "2025-11-20T12:00:00Z"
  }
}
```

**User Stories**: US-4  
**Requirements**: FR-007, FR-009  
**Success Criteria**: SC-003 (redemption without assistance), SC-009 (completion under 90 seconds)

**Edge Cases**:
- Insufficient balance (400 Bad Request with shortfall details)
- Reward not found (404 Not Found)
- Reward inactive (400 Bad Request)
- Customer not enrolled (404 Not Found)

---

### 7. Get Redemption History

**Endpoint**: `GET /v1/loyalty/redemptions?limit=50&offset=0`

**Description**: List customer's redemption history

**Query Parameters**:
- `limit` (int, default: 50, max: 100)
- `offset` (int, default: 0)
- `status` (string, optional): Filter by status (ACTIVE, USED, REVERSED)

**Response** (`ListRedemptionsResponse` - 200 OK):
```json
{
  "redemptions": [
    {
      "id": "redemption-1",
      "reward": {
        "id": "reward-1",
        "name": "$5 Discount Code",
        "type": "DISCOUNT"
      },
      "points_deducted": 500,
      "status": "USED",
      "code": "DISC-ABC123",
      "created_at": "2025-11-20T12:00:00Z",
      "used_at": "2025-11-20T14:30:00Z"
    }
  ],
  "total": 15,
  "limit": 50,
  "offset": 0
}
```

**User Stories**: US-4  
**Requirements**: FR-018  

---

### 8. Reverse Redemption (Admin)

**Endpoint**: `POST /v1/admin/loyalty/redemptions/{redemption_id}/reverse`

**Description**: Reverse a redemption and restore points (for order cancellation/refund)

**Authorization**: Requires admin role

**Request** (`ReverseRedemptionRequest`):
```json
{
  "reason": "Order cancelled by customer"
}
```

**Response** (`ReverseRedemptionResponse` - 200 OK):
```json
{
  "redemption": {
    "id": "redemption-id",
    "status": "REVERSED",
    "reversed_at": "2025-11-20T15:00:00Z",
    "reversal_reason": "Order cancelled by customer"
  },
  "points_restored": 500,
  "new_balance": 1250,
  "transaction": {
    "id": "txn-restore-id",
    "amount": 500,
    "type": "ADJUSTMENT",
    "description": "Redemption reversal: Order cancelled by customer"
  }
}
```

**User Stories**: US-4  
**Requirements**: FR-010  

**Edge Cases**:
- Redemption not found (404 Not Found)
- Already reversed (409 Conflict)
- Missing admin role (403 Forbidden)

---

### 9. Manual Point Adjustment (Admin)

**Endpoint**: `POST /v1/admin/loyalty/customers/{customer_id}/adjust`

**Description**: Manually adjust a customer's point balance with audit trail

**Authorization**: Requires admin role

**Request** (`AdjustPointsRequest`):
```json
{
  "amount": -100,  // Positive to add, negative to deduct
  "reason": "Compensation for service issue"
}
```

**Response** (`AdjustPointsResponse` - 201 Created):
```json
{
  "transaction": {
    "id": "txn-adj-id",
    "customer_id": "customer-id",
    "amount": -100,
    "type": "ADJUSTMENT",
    "description": "Manual adjustment by admin",
    "admin_user_id": "admin-user-123",
    "admin_note": "Compensation for service issue",
    "created_at": "2025-11-20T16:00:00Z"
  },
  "new_balance": 1150
}
```

**User Stories**: US-6  
**Requirements**: FR-004, FR-024  

**Edge Cases**:
- Customer not found (404 Not Found)
- Amount is zero (400 Bad Request)
- Adjustment would make balance negative (400 Bad Request)
- Missing admin role (403 Forbidden)
- Missing reason (400 Bad Request)

---

### 10. List Membership Tiers

**Endpoint**: `GET /v1/loyalty/tiers`

**Description**: Get all available membership tiers

**Response** (`ListTiersResponse` - 200 OK):
```json
{
  "tiers": [
    {
      "id": "tier-base",
      "name": "Base",
      "level": 0,
      "qualification_points": 0,
      "evaluation_days": 365,
      "earn_rate_multiplier": 1.0,
      "description": "Standard membership benefits"
    },
    {
      "id": "tier-silver",
      "name": "Silver",
      "level": 1,
      "qualification_points": 500,
      "evaluation_days": 365,
      "earn_rate_multiplier": 1.25,
      "description": "Earn 1.25x points on all purchases"
    },
    {
      "id": "tier-gold",
      "name": "Gold",
      "level": 2,
      "qualification_points": 1000,
      "evaluation_days": 365,
      "earn_rate_multiplier": 1.5,
      "description": "Earn 1.5x points on all purchases"
    }
  ]
}
```

**User Stories**: US-5  
**Requirements**: FR-019  

---

### 11. Create Campaign (Admin)

**Endpoint**: `POST /v1/admin/loyalty/campaigns`

**Description**: Create a promotional campaign with bonus points or multipliers

**Authorization**: Requires admin role

**Request** (`CreateCampaignRequest`):
```json
{
  "name": "Holiday Double Points",
  "description": "Earn 2x points on all purchases",
  "start_date": "2025-12-01T00:00:00Z",
  "end_date": "2025-12-31T23:59:59Z",
  "point_multiplier": 2.0,
  "bonus_points": 0,
  "conditions": {
    "min_purchase_amount": 1000,
    "eligible_categories": ["electronics", "apparel"]
  },
  "priority": 10
}
```

**Response** (`CreateCampaignResponse` - 201 Created):
```json
{
  "campaign": {
    "id": "campaign-id",
    "name": "Holiday Double Points",
    "description": "Earn 2x points on all purchases",
    "start_date": "2025-12-01T00:00:00Z",
    "end_date": "2025-12-31T23:59:59Z",
    "point_multiplier": 2.0,
    "bonus_points": 0,
    "is_active": true,
    "conditions": {
      "min_purchase_amount": 1000,
      "eligible_categories": ["electronics", "apparel"]
    },
    "priority": 10,
    "created_at": "2025-11-20T17:00:00Z"
  }
}
```

**User Stories**: US-6  
**Requirements**: FR-017  
**Success Criteria**: SC-010 (rule changes effective within 1 minute)

**Edge Cases**:
- Invalid date range (400 Bad Request)
- Missing admin role (403 Forbidden)
- Multiplier < 1.0 (400 Bad Request)

---

### 12. Analytics Dashboard (Admin)

**Endpoint**: `GET /v1/admin/loyalty/analytics?start_date=2025-01-01&end_date=2025-12-31`

**Description**: Get program analytics and metrics

**Authorization**: Requires admin role

**Query Parameters**:
- `start_date` (ISO8601, required): Analysis period start
- `end_date` (ISO8601, required): Analysis period end

**Response** (`AnalyticsResponse` - 200 OK):
```json
{
  "period": {
    "start_date": "2025-01-01T00:00:00Z",
    "end_date": "2025-12-31T23:59:59Z"
  },
  "enrollment": {
    "total_members": 10542,
    "new_enrollments": 1253,
    "active_members": 8721
  },
  "points": {
    "total_points_issued": 2450000,
    "total_points_redeemed": 1120000,
    "outstanding_liability": 1330000,
    "avg_balance_per_member": 126
  },
  "redemptions": {
    "total_redemptions": 3456,
    "total_points_redeemed": 1120000,
    "avg_points_per_redemption": 324,
    "most_popular_rewards": [
      {
        "reward_id": "reward-1",
        "reward_name": "$5 Discount",
        "redemption_count": 1234
      }
    ]
  },
  "tiers": {
    "base": 7542,
    "silver": 2100,
    "gold": 800,
    "platinum": 100
  }
}
```

**User Stories**: US-6  
**Requirements**: FR-023  

**Edge Cases**:
- Invalid date range (400 Bad Request)
- Missing admin role (403 Forbidden)

---

### 13. Health Check

**Endpoint**: `GET /health`

**Description**: Health check endpoint for monitoring and container orchestration

**Authentication**: None required

**Response** (200 OK):
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "timestamp": "2025-11-20T18:00:00Z",
  "dependencies": {
    "database": "healthy",
    "tracing": "healthy"
  }
}
```

---

## Rate Limiting

- **Customer endpoints**: 100 requests/minute per authenticated user
- **Admin endpoints**: 1000 requests/minute per admin user
- **Exceeded**: 429 Too Many Requests

## Pagination

List endpoints use offset-based pagination:
- `limit`: Items per page (default: 50, max: 100)
- `offset`: Number of items to skip (default: 0)
- Response includes `total` count for calculating page counts

## Idempotency

Endpoints that create resources (earn points, redeem rewards) support idempotency:
- Earn points: `reference_id` ensures duplicate requests return existing transaction
- Redeem reward: Multiple requests return existing redemption if called within 5 minutes

## Caching

- **Rewards catalog**: Cache for 5 minutes
- **Tiers list**: Cache for 1 hour
- **Customer balance**: No cache (always fresh)

## Protobuf Definitions

All request/response types will be defined in these proto files:

- `api/v1/loyalty.proto` - Enrollment, customer status
- `api/v1/transaction.proto` - Point transactions, earn, history
- `api/v1/reward.proto` - Rewards, redemptions
- `api/v1/campaign.proto` - Campaigns
- `api/v1/admin.proto` - Administrative endpoints
- `api/v1/common.proto` - Shared types (pagination, errors)

## Summary

This API provides:
- ✅ Customer enrollment with referrals (US-1)
- ✅ Point earning with campaigns (US-2)
- ✅ Balance and transaction history viewing (US-3)
- ✅ Reward redemption and reversal (US-4)
- ✅ Tier progression tracking (US-5)
- ✅ Administrative management (US-6)
- ✅ All functional requirements (FR-001 through FR-025)
- ✅ Edge case coverage per constitution
- ✅ Protobuf contract foundation

