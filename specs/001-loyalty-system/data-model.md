# Data Model: Customer Loyalty System

**Date**: November 20, 2025  
**Phase**: Phase 1 - Design & Contracts  
**Source**: Extracted from feature spec requirements and key entities

## Overview

The loyalty system uses an event-sourced approach for point transactions with denormalized balance for query performance. All entities use GORM models internally with protobuf representations for API contracts.

## Entity Diagram

```
Customer (1) ──────< (N) PointTransaction
    │                        │
    │                        │
    ├─< (1) MembershipTier   └─> (0..1) Redemption
    │
    └─< (N) Redemption ──> (1) Reward

PromotionalCampaign (N) ──< (N) EligibleProducts
```

## Core Entities

### 1. Customer

**Purpose**: Represents a loyalty program member with enrollment and tier information

**GORM Model** (`internal/models/customer.go`):
```go
type Customer struct {
    ID                string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    AccountID         string    `gorm:"type:varchar(255);not null;uniqueIndex"` // Link to external account system
    MembershipNumber  string    `gorm:"type:varchar(50);not null;uniqueIndex"`
    ReferralCode      string    `gorm:"type:varchar(20);not null;uniqueIndex"`
    ReferredBy        *string   `gorm:"type:varchar(20);index"`
    EnrolledAt        time.Time `gorm:"not null"`
    CurrentBalance    int64     `gorm:"not null;default:0"` // Cached from transactions
    TierID            *string   `gorm:"type:uuid;index"`
    Tier              *MembershipTier `gorm:"foreignKey:TierID"`
    CreatedAt         time.Time
    UpdatedAt         time.Time
}
```

**Fields**:
- `ID`: UUID primary key
- `AccountID`: External account system reference (unique)
- `MembershipNumber`: Human-readable membership identifier
- `ReferralCode`: Unique code for referring others
- `ReferredBy`: Referral code of who referred this customer (nullable)
- `EnrolledAt`: Enrollment timestamp
- `CurrentBalance`: Cached point balance (denormalized)
- `TierID`: Current membership tier (nullable for base tier)
- `Tier`: Relationship to tier
- Timestamps: `CreatedAt`, `UpdatedAt`

**Validation Rules**:
- AccountID must be unique and non-empty
- MembershipNumber generated on creation (format: "LM-{timestamp}-{random}")
- ReferralCode generated on creation (8-char alphanumeric)
- CurrentBalance must be >= 0
- ReferredBy must reference valid referral code if provided

**Indexes**:
- Unique: AccountID, MembershipNumber, ReferralCode
- Index: ReferredBy, TierID

### 2. PointTransaction

**Purpose**: Immutable event log of all point earning and spending activities

**GORM Model** (`internal/models/transaction.go`):
```go
type TransactionType string

const (
    TransactionTypeEarn       TransactionType = "EARN"
    TransactionTypeRedemption TransactionType = "REDEMPTION"
    TransactionTypeAdjustment TransactionType = "ADJUSTMENT"
    TransactionTypeReferral   TransactionType = "REFERRAL"
    TransactionTypeExpiration TransactionType = "EXPIRATION"
)

type PointTransaction struct {
    ID              string          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    CustomerID      string          `gorm:"type:uuid;not null;index:idx_customer_created"`
    Customer        *Customer       `gorm:"foreignKey:CustomerID"`
    Amount          int64           `gorm:"not null"` // Positive for earning, negative for spending
    Type            TransactionType `gorm:"type:varchar(20);not null;index"`
    ReferenceID     *string         `gorm:"type:varchar(255);uniqueIndex"` // External ref (order ID, etc.)
    ReferenceType   *string         `gorm:"type:varchar(50)"` // e.g., "ORDER", "REFERRAL"
    Description     string          `gorm:"type:text;not null"`
    CampaignID      *string         `gorm:"type:uuid;index"`
    Campaign        *PromotionalCampaign `gorm:"foreignKey:CampaignID"`
    RedemptionID    *string         `gorm:"type:uuid"`
    Redemption      *Redemption     `gorm:"foreignKey:RedemptionID"`
    ExpiredAt       *time.Time      // Set when points expire
    ExpiresAt       *time.Time      // When these points will expire
    AdminUserID     *string         `gorm:"type:varchar(255)"` // Set for manual adjustments
    AdminNote       *string         `gorm:"type:text"` // Audit note for adjustments
    CreatedAt       time.Time       `gorm:"not null;index:idx_customer_created"`
}
```

**Fields**:
- `ID`: UUID primary key
- `CustomerID`: Owner of transaction
- `Amount`: Points (positive = earned, negative = redeemed/expired)
- `Type`: Transaction type (EARN, REDEMPTION, ADJUSTMENT, REFERRAL, EXPIRATION)
- `ReferenceID`: External reference (e.g., order ID) for idempotency
- `ReferenceType`: Type of reference (e.g., "ORDER", "REFERRAL")
- `Description`: Human-readable description
- `CampaignID`: Promotional campaign if applicable
- `RedemptionID`: Link to redemption if type is REDEMPTION
- `ExpiredAt`: When points were expired (for expiration records)
- `ExpiresAt`: When these earned points will expire
- `AdminUserID`: Admin who made adjustment (for audit)
- `AdminNote`: Explanation for manual adjustment
- `CreatedAt`: Transaction timestamp

**Validation Rules**:
- Amount must not be zero
- Type must be valid enum value
- EARN transactions must have positive amount
- REDEMPTION/EXPIRATION transactions must have negative amount
- ReferenceID must be unique if provided (idempotency)
- AdminUserID required if Type is ADJUSTMENT

**Indexes**:
- Composite: (CustomerID, CreatedAt) for history queries
- Index: Type, CampaignID
- Unique: ReferenceID (for idempotency)

**State Transitions**:
- Transactions are immutable once created
- Reversals create new offsetting transactions

### 3. MembershipTier

**Purpose**: Defines tier levels with qualification thresholds and benefits

**GORM Model** (`internal/models/tier.go`):
```go
type MembershipTier struct {
    ID                  string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    Name                string    `gorm:"type:varchar(50);not null;uniqueIndex"`
    Level               int       `gorm:"not null;uniqueIndex"` // 0=Base, 1=Silver, 2=Gold, etc.
    QualificationPoints int64     `gorm:"not null"` // Points needed in evaluation period
    EvaluationDays      int       `gorm:"not null;default:365"` // Rolling period (default 12 months)
    EarnRateMultiplier  float64   `gorm:"type:decimal(3,2);not null;default:1.0"` // e.g., 1.5 for 1.5x
    Description         string    `gorm:"type:text"`
    CreatedAt           time.Time
    UpdatedAt           time.Time
}
```

**Fields**:
- `ID`: UUID primary key
- `Name`: Tier name (e.g., "Silver", "Gold", "Platinum")
- `Level`: Numeric level for ordering (0 = base)
- `QualificationPoints`: Points earned in evaluation period to qualify
- `EvaluationDays`: Rolling evaluation period in days (default 365)
- `EarnRateMultiplier`: Point earning multiplier (1.0 = 1x, 1.5 = 1.5x)
- `Description`: Tier benefits description
- Timestamps: `CreatedAt`, `UpdatedAt`

**Validation Rules**:
- Name must be unique
- Level must be unique and >= 0
- QualificationPoints must be >= 0
- EvaluationDays must be > 0
- EarnRateMultiplier must be > 0

**Indexes**:
- Unique: Name, Level

### 4. Reward

**Purpose**: Defines available rewards that can be redeemed with points

**GORM Model** (`internal/models/reward.go`):
```go
type RewardType string

const (
    RewardTypeDiscount  RewardType = "DISCOUNT"
    RewardTypeVoucher   RewardType = "VOUCHER"
    RewardTypeFreeItem  RewardType = "FREE_ITEM"
)

type Reward struct {
    ID          string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    Name        string     `gorm:"type:varchar(255);not null"`
    Description string     `gorm:"type:text"`
    Type        RewardType `gorm:"type:varchar(20);not null"`
    PointCost   int64      `gorm:"not null;index"`
    IsActive    bool       `gorm:"not null;default:true;index"`
    Metadata    string     `gorm:"type:jsonb"` // Flexible data (discount %, product ID, etc.)
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

**Fields**:
- `ID`: UUID primary key
- `Name`: Reward name
- `Description`: Detailed description
- `Type`: Reward type (DISCOUNT, VOUCHER, FREE_ITEM)
- `PointCost`: Points required to redeem
- `IsActive`: Whether reward is currently available
- `Metadata`: JSONB for flexible reward data
- Timestamps: `CreatedAt`, `UpdatedAt`

**Validation Rules**:
- Name must not be empty
- Type must be valid enum value
- PointCost must be > 0
- Metadata must be valid JSON

**Indexes**:
- Index: PointCost (for filtering by cost range)
- Index: IsActive (for filtering available rewards)

### 5. Redemption

**Purpose**: Records when a customer redeems points for a reward

**GORM Model** (`internal/models/redemption.go`):
```go
type RedemptionStatus string

const (
    RedemptionStatusActive   RedemptionStatus = "ACTIVE"
    RedemptionStatusUsed     RedemptionStatus = "USED"
    RedemptionStatusReversed RedemptionStatus = "REVERSED"
)

type Redemption struct {
    ID               string           `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    CustomerID       string           `gorm:"type:uuid;not null;index"`
    Customer         *Customer        `gorm:"foreignKey:CustomerID"`
    RewardID         string           `gorm:"type:uuid;not null;index"`
    Reward           *Reward          `gorm:"foreignKey:RewardID"`
    PointsDeducted   int64            `gorm:"not null"`
    Status           RedemptionStatus `gorm:"type:varchar(20);not null;index"`
    Code             string           `gorm:"type:varchar(50);uniqueIndex"` // Discount/voucher code
    UsedAt           *time.Time
    ReversedAt       *time.Time
    ReversalReason   *string          `gorm:"type:text"`
    CreatedAt        time.Time        `gorm:"not null"`
    UpdatedAt        time.Time
}
```

**Fields**:
- `ID`: UUID primary key
- `CustomerID`: Customer who redeemed
- `RewardID`: Reward that was redeemed
- `PointsDeducted`: Points spent (positive value)
- `Status`: Current status (ACTIVE, USED, REVERSED)
- `Code`: Generated code for applying reward (e.g., discount code)
- `UsedAt`: When reward was applied/used
- `ReversedAt`: When redemption was reversed
- `ReversalReason`: Explanation for reversal
- Timestamps: `CreatedAt`, `UpdatedAt`

**Validation Rules**:
- PointsDeducted must be > 0
- Status must be valid enum value
- Code must be unique
- UsedAt required if Status is USED
- ReversedAt required if Status is REVERSED

**Indexes**:
- Index: CustomerID, RewardID, Status
- Unique: Code

**State Transitions**:
- ACTIVE → USED (when applied to order)
- ACTIVE → REVERSED (when order cancelled)
- USED → REVERSED (when order refunded)

### 6. PromotionalCampaign

**Purpose**: Defines temporary bonus earning opportunities

**GORM Model** (`internal/models/campaign.go`):
```go
type PromotionalCampaign struct {
    ID              string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    Name            string    `gorm:"type:varchar(255);not null"`
    Description     string    `gorm:"type:text"`
    StartDate       time.Time `gorm:"not null;index:idx_campaign_dates"`
    EndDate         time.Time `gorm:"not null;index:idx_campaign_dates"`
    PointMultiplier float64   `gorm:"type:decimal(3,2);not null;default:1.0"` // e.g., 2.0 for double points
    BonusPoints     int64     `gorm:"default:0"` // Fixed bonus (alternative to multiplier)
    IsActive        bool      `gorm:"not null;default:true;index"`
    Conditions      string    `gorm:"type:jsonb"` // Flexible rules (min purchase, eligible products, etc.)
    Priority        int       `gorm:"not null;default:0"` // Higher priority when multiple campaigns overlap
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

**Fields**:
- `ID`: UUID primary key
- `Name`: Campaign name
- `Description`: Campaign details
- `StartDate`: Campaign start (inclusive)
- `EndDate`: Campaign end (inclusive)
- `PointMultiplier`: Multiplier for earned points (e.g., 2.0 = double)
- `BonusPoints`: Fixed bonus points (alternative to multiplier)
- `IsActive`: Whether campaign is currently enabled
- `Conditions`: JSONB for flexible eligibility rules
- `Priority`: Used when multiple campaigns overlap (higher wins)
- Timestamps: `CreatedAt`, `UpdatedAt`

**Validation Rules**:
- Name must not be empty
- StartDate must be before EndDate
- PointMultiplier must be >= 1.0
- BonusPoints must be >= 0
- Priority must be >= 0
- Conditions must be valid JSON

**Indexes**:
- Composite: (StartDate, EndDate) for finding active campaigns
- Index: IsActive

## Relationships

### Customer Relationships
- Customer → PointTransaction (1:N) - All transactions for a customer
- Customer → MembershipTier (N:1) - Current tier assignment
- Customer → Redemption (1:N) - All redemptions by customer
- Customer → Customer (self-referral) - ReferredBy links to another customer's ReferralCode

### Transaction Relationships
- PointTransaction → Customer (N:1) - Owner of transaction
- PointTransaction → PromotionalCampaign (N:1) - Campaign that generated bonus
- PointTransaction → Redemption (1:1) - For REDEMPTION type transactions

### Redemption Relationships
- Redemption → Customer (N:1) - Who redeemed
- Redemption → Reward (N:1) - What was redeemed
- Redemption → PointTransaction (1:1) - Corresponding deduction transaction

### Campaign Relationships
- PromotionalCampaign → PointTransaction (1:N) - Transactions created under campaign

## Database Constraints

### Foreign Key Cascades
- Customer deletion → CASCADE to PointTransaction, Redemption (delete all customer data)
- Reward deletion → RESTRICT if active Redemptions exist
- Campaign deletion → SET NULL in PointTransaction (preserve historical data)
- Tier deletion → RESTRICT if Customers assigned

### Check Constraints
- Customer.CurrentBalance >= 0 (enforced in application, not database)
- PointTransaction.Amount != 0
- Reward.PointCost > 0
- Redemption.PointsDeducted > 0
- MembershipTier.Level >= 0
- PromotionalCampaign.StartDate < EndDate

### Unique Constraints
- Customer: AccountID, MembershipNumber, ReferralCode
- PointTransaction: ReferenceID (if not null)
- MembershipTier: Name, Level
- Redemption: Code

## Derived Values

### Customer Balance Calculation
```sql
SELECT SUM(amount) 
FROM point_transactions 
WHERE customer_id = ? AND expired_at IS NULL
```
- Cached in `Customer.CurrentBalance` for performance
- Recalculated on every transaction
- Verified in tests for accuracy (SC-004)

### Tier Qualification
```sql
SELECT SUM(amount) 
FROM point_transactions 
WHERE customer_id = ? 
  AND type = 'EARN' 
  AND created_at >= NOW() - INTERVAL '{evaluation_days} days'
```
- Calculated during tier evaluation job
- Based on points earned (not current balance)
- Uses rolling window defined by tier

## Migration Strategy

### Initial Schema Creation
```go
// services/migrations.go
func AutoMigrate(db *gorm.DB) error {
    return db.AutoMigrate(
        &Customer{},
        &MembershipTier{},
        &PointTransaction{},
        &Reward{},
        &Redemption{},
        &PromotionalCampaign{},
    )
}
```

### Seed Data (Development/Testing)
- Base tier (Level 0, 1.0x multiplier, 0 points required)
- Sample reward catalog
- Test customers with various balances and tiers

## Performance Considerations

### Query Optimization
- Composite index on (customer_id, created_at) for transaction history
- Partial index on IsActive for rewards and campaigns
- JSONB GIN index on Metadata and Conditions if complex queries needed

### Scaling Strategies
- Partition PointTransaction table by created_at (monthly) if volume exceeds 10M rows
- Read replicas for analytics queries (dashboard, reports)
- Cache customer balance in memory (Redis) for high-traffic scenarios

### Data Retention
- Keep all PointTransaction records indefinitely (audit requirement)
- Archive old Redemption records (USED/REVERSED older than 2 years)
- Soft delete inactive campaigns (preserve for historical analysis)

## Summary

The data model supports all functional requirements with:
- ✅ Event-sourced point transactions (FR-005, FR-006, SC-004)
- ✅ Immutable transaction log for audit (FR-024)
- ✅ Tier progression with rolling evaluation (FR-013, FR-014)
- ✅ Referral tracking via referral codes (FR-020)
- ✅ Promotional campaigns with flexible conditions (FR-017)
- ✅ Redemption lifecycle management (FR-008, FR-009, FR-010)
- ✅ Concurrency-safe balance calculations (SC-007)

