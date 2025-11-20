# Research: Customer Loyalty System

**Date**: November 20, 2025  
**Phase**: Phase 0 - Research & Investigation  
**Purpose**: Resolve technical unknowns and establish design decisions

## Research Tasks

### 1. Target Platform & Deployment

**Unknown**: Containerized deployment strategy and cloud platform preference

**Decision**: Docker containerization with platform-agnostic deployment

**Rationale**:
- Docker provides consistent environment across development, testing, and production
- Platform-agnostic approach allows deployment to any cloud provider (AWS ECS/EKS, GCP Cloud Run/GKE, Azure Container Instances/AKS) or on-premises Kubernetes
- 12-factor app principles enable easy configuration through environment variables
- Horizontal scaling through container orchestration handles promotional traffic spikes (SC-012)

**Alternatives Considered**:
- **Serverless (AWS Lambda, Cloud Functions)**: Rejected due to cold start latency concerns for point balance queries (must be <2s per SC-005), and complexity of managing database connections in serverless environment
- **Traditional VM deployment**: Rejected due to slower scaling and less consistent environment management
- **Platform-specific PaaS (Heroku, App Engine)**: Rejected to avoid vendor lock-in and maintain deployment flexibility

**Implementation Approach**:
- Multi-stage Dockerfile for optimal image size
- Health check endpoints for container orchestration
- Configuration via environment variables (database DSN, tracing backend, etc.)
- Docker Compose for local development environment
- Kubernetes-ready with standard readiness/liveness probes

### 2. Point Calculation & Precision

**Unknown**: How to handle point calculations with decimal precision (e.g., $10.50 purchase at 1 point per dollar)

**Decision**: Store points as integers (smallest unit), calculate using integer math with rounding rules

**Rationale**:
- Avoids floating-point precision errors that could cause balance discrepancies
- Integer operations are deterministic and exact
- Satisfies SC-004 requirement for 99.9% accuracy
- Industry standard for financial calculations (similar to storing cents instead of dollars)

**Alternatives Considered**:
- **Decimal type (NUMERIC in PostgreSQL)**: Considered but adds complexity; integer math is simpler and sufficient
- **Floating-point**: Rejected due to precision issues and non-deterministic rounding

**Implementation Approach**:
- Store points as `int64` in database
- Default earn rate: 1 point per whole dollar (truncate cents)
- Tier multipliers applied before truncation (e.g., 1.5x for Gold: $10.50 * 1.5 = 15.75 → 15 points)
- Document rounding behavior in API contracts
- Point expiration works at point level (not fractional)

### 3. Concurrency Control Strategy

**Unknown**: How to prevent race conditions on point balance updates (SC-007: zero concurrency discrepancies)

**Decision**: Optimistic locking with version field + database constraints

**Rationale**:
- Points are event-sourced through transactions table (immutable log)
- Balance is derived value, not authoritative source
- Database unique constraints prevent duplicate transactions
- Optimistic locking on tier changes (rare updates)
- Satisfies SC-007 requirement

**Alternatives Considered**:
- **Pessimistic locking (SELECT FOR UPDATE)**: Rejected due to performance impact and potential deadlocks under high load
- **Distributed locks (Redis)**: Rejected as over-engineering; database transactions sufficient
- **Balance field with row locking**: Rejected in favor of event-sourced approach

**Implementation Approach**:
- Transactions table is append-only (immutable)
- Balance calculated as SUM(amount) from transactions table
- Unique constraint on transaction reference IDs (idempotency)
- Version field on Customer model for tier updates
- Database transaction isolation level: READ COMMITTED (PostgreSQL default)
- Test concurrent redemptions in integration tests

### 4. Point Expiration Strategy

**Unknown**: How to efficiently expire points for 10,000+ members within 1-hour batch window (SC-011)

**Decision**: Periodic batch job with indexed queries + GORM batch processing

**Rationale**:
- Batch processing more efficient than per-customer expiration
- Index on transaction created_at enables fast "expiring soon" queries
- GORM batch operations reduce memory footprint
- 1-hour window allows for retry and error handling
- Can be run as scheduled job (cron, Kubernetes CronJob, etc.)

**Alternatives Considered**:
- **Lazy expiration (on read)**: Rejected due to inconsistent user experience
- **Database triggers**: Rejected due to complexity and difficulty testing
- **Real-time expiration**: Rejected as unnecessary overhead; daily batch sufficient

**Implementation Approach**:
- Scheduled job runs daily (configurable time, e.g., 2 AM)
- Query transactions where created_at < NOW() - 12 months AND expired = false
- Process in batches of 1000 transactions
- Create expiration transaction records (negative amounts)
- Update expired flag on original transactions
- Job completion tracked for monitoring
- Dry-run mode for testing

### 5. Authentication Integration

**Unknown**: How to integrate with existing customer account system (Assumption #1 from spec)

**Decision**: Middleware-based authentication with user ID in context

**Rationale**:
- Constitution Principle VIII requires services NOT depend on HTTP types
- Authentication is HTTP layer concern (middleware)
- User ID passed through context to services
- Flexible integration with JWT, OAuth2, API keys, or session-based auth

**Alternatives Considered**:
- **Service-level authentication**: Rejected per constitution (services should not handle HTTP concerns)
- **Direct database user lookup in services**: Rejected to maintain clean separation

**Implementation Approach**:
- Authentication middleware extracts user ID from request (JWT claims, session, etc.)
- User ID stored in context: `ctx = context.WithValue(ctx, "user_id", userID)`
- Services receive authenticated context
- Services validate user owns the loyalty account they're accessing
- Tests mock authentication by setting user ID in context
- Pluggable authentication strategy pattern for different auth methods

### 6. Notification System Integration

**Unknown**: How to send notifications for enrollments, expirations, tier changes (Dependency #3 from spec)

**Decision**: Event-driven with channel abstraction (email, SMS, in-app)

**Rationale**:
- Decouples loyalty service from notification delivery mechanism
- Allows async processing (notifications don't block transactions)
- Multiple notification channels supported
- Constitutional approach: services emit events, handlers/workers send notifications

**Alternatives Considered**:
- **Synchronous notification in service**: Rejected due to latency and coupling
- **Direct email library in service**: Rejected per constitution (external dependency coupling)

**Implementation Approach**:
- Service layer emits domain events (EnrollmentCreated, PointsExpiring, TierChanged)
- Event bus interface (simple Go channel or message queue)
- Notification worker consumes events and sends notifications
- Worker configures delivery method (SMTP, Twilio, push notifications)
- Graceful degradation: notification failure doesn't fail transaction
- Tests mock notification worker

### 7. Referral Tracking

**Unknown**: How to implement referral point bonuses (FR-020)

**Decision**: Referral code system with transaction reference linking

**Rationale**:
- Each customer receives unique referral code on enrollment
- Referred customer provides code during enrollment
- Referral bonus credited when referred customer completes qualifying action
- Transaction records maintain referral relationship

**Alternatives Considered**:
- **Email-based referral**: Rejected as less flexible and harder to track
- **Link-based referral with tokens**: Considered but adds complexity

**Implementation Approach**:
- Customer model includes referral_code (unique) and referred_by (nullable)
- Referral code generated on enrollment (short alphanumeric, e.g., "JOHN2K5")
- Enrollment endpoint accepts optional referral_code parameter
- Qualifying action (first purchase) triggers referral bonus transaction
- Both referrer and referee can see referral relationship in dashboard
- Protobuf messages include referral fields

### 8. Admin Role Permissions

**Unknown**: How to restrict administrative endpoints (FR-003, FR-024)

**Decision**: Role-based authorization middleware

**Rationale**:
- Extends authentication middleware with role checking
- Admin role required for configuration, manual adjustments, analytics
- Audit log tracks admin actions with user ID

**Alternatives Considered**:
- **Service-level authorization**: Rejected per constitution
- **Fine-grained permissions (RBAC)**: Deferred for future; simple admin role sufficient initially

**Implementation Approach**:
- Authentication context includes roles: `ctx.Value("roles").([]string)`
- Authorization middleware checks for "admin" role on admin endpoints
- 403 Forbidden returned for insufficient permissions
- All admin actions logged to audit table
- Tests verify authorization on each admin endpoint

## Technology Best Practices

### PostgreSQL for Loyalty System

**Best Practices**:
- Use JSONB for flexible attributes (promotional campaign conditions, reward metadata)
- Partial indexes for active campaigns, non-expired points
- Composite indexes on (customer_id, created_at) for transaction history queries
- Use PostgreSQL advisory locks if needed for critical sections
- Connection pooling via GORM (default pool settings sufficient for 10k users)

**Schema Design**:
- Immutable transaction log (event sourcing for points)
- Denormalized balance field on Customer model (cached calculation)
- Cascade delete from Customer to prevent orphaned records
- Check constraints for business rules (points >= 0 after redemption validation in code)

### GORM Best Practices for Loyalty System

**Best Practices**:
- Use `db.WithContext(ctx)` for all queries (Principle X)
- Preload relationships efficiently (Preload("Tier"), Preload("Transactions"))
- Use `Select()` to limit columns on large queries
- Batch inserts for bulk operations (e.g., expiration job)
- Avoid N+1 queries in list endpoints

**Transaction Management**:
- Wrap multi-table operations in transactions (Begin/Commit/Rollback)
- Use `defer tx.Rollback()` pattern for automatic cleanup
- Keep transactions short (avoid external API calls inside transactions)

### OpenTracing for Loyalty System

**Best Practices**:
- Trace enrollment flow: HTTP → EnrollmentHandler → LoyaltyService → DB
- Trace earn points: HTTP → PointsHandler → TransactionService → DB
- Trace redemption: HTTP → RewardHandler → RewardService + TransactionService → DB
- Tag spans with customer_id, transaction_id for debugging
- Separate spans for tier evaluation (can be expensive)

**Tracing Setup**:
- Development: NoopTracer (no external dependency)
- Testing: NoopTracer or MockTracer for validation
- Production: Jaeger or Zipkin configured via environment variables

## Summary

All technical unknowns have been resolved:

1. ✅ **Target Platform**: Docker containers, platform-agnostic deployment
2. ✅ **Point Calculation**: Integer math with documented rounding
3. ✅ **Concurrency Control**: Event-sourced transactions + optimistic locking
4. ✅ **Point Expiration**: Batch job with indexed queries
5. ✅ **Authentication**: Middleware-based with context user ID
6. ✅ **Notifications**: Event-driven with channel abstraction
7. ✅ **Referrals**: Referral code system with transaction linking
8. ✅ **Admin Permissions**: Role-based authorization middleware

These decisions enable implementation to proceed to Phase 1 (data model and contracts design).

