package models

import (
	"time"
)

// TransactionType represents the type of point transaction
type TransactionType string

const (
	TransactionTypeEarn       TransactionType = "EARN"
	TransactionTypeRedemption TransactionType = "REDEMPTION"
	TransactionTypeAdjustment TransactionType = "ADJUSTMENT"
	TransactionTypeReferral   TransactionType = "REFERRAL"
	TransactionTypeExpiration TransactionType = "EXPIRATION"
)

// PointTransaction represents an immutable event log of all point earning and spending activities
type PointTransaction struct {
	ID            string          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CustomerID    string          `gorm:"type:uuid;not null;index:idx_customer_created"`
	Amount        int64           `gorm:"not null"` // Positive for earning, negative for spending
	Type          TransactionType `gorm:"type:varchar(20);not null;index"`
	ReferenceID   *string         `gorm:"type:varchar(255);uniqueIndex"` // External ref (order ID, etc.)
	ReferenceType *string         `gorm:"type:varchar(50)"`              // e.g., "ORDER", "REFERRAL"
	Description   string          `gorm:"type:text;not null"`
	CampaignID    *string         `gorm:"type:uuid;index"`
	RedemptionID  *string         `gorm:"type:uuid"`
	ExpiredAt     *time.Time      // Set when points expire
	ExpiresAt     *time.Time      // When these points will expire
	AdminUserID   *string         `gorm:"type:varchar(255)"` // Set for manual adjustments
	AdminNote     *string         `gorm:"type:text"`         // Audit note for adjustments
	CreatedAt     time.Time       `gorm:"not null;index:idx_customer_created"`
	
	// Relationships
	Customer   *Customer            `gorm:"foreignKey:CustomerID"`
	Campaign   *PromotionalCampaign `gorm:"foreignKey:CampaignID"`
	Redemption *Redemption          `gorm:"foreignKey:RedemptionID"`
}

// TableName specifies the table name for the PointTransaction model
func (PointTransaction) TableName() string {
	return "point_transactions"
}

