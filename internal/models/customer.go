package models

import (
	"time"

	"gorm.io/gorm"
)

// Customer represents a loyalty program member with enrollment and tier information
type Customer struct {
	ID               string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AccountID        string    `gorm:"type:varchar(255);not null;uniqueIndex"`
	MembershipNumber string    `gorm:"type:varchar(50);not null;uniqueIndex"`
	ReferralCode     string    `gorm:"type:varchar(20);not null;uniqueIndex"`
	ReferredBy       *string   `gorm:"type:varchar(20);index"`
	EnrolledAt       time.Time `gorm:"not null"`
	CurrentBalance   int64     `gorm:"not null;default:0"`
	TierID           *string   `gorm:"type:uuid;index"`
	
	// Relationships
	Tier              *MembershipTier    `gorm:"foreignKey:TierID"`
	PointTransactions []PointTransaction `gorm:"foreignKey:CustomerID"`
	Redemptions       []Redemption       `gorm:"foreignKey:CustomerID"`
	
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for the Customer model
func (Customer) TableName() string {
	return "customers"
}

