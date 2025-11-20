package models

import (
	"time"

	"gorm.io/gorm"
)

// RedemptionStatus represents the current status of a redemption
type RedemptionStatus string

const (
	RedemptionStatusActive   RedemptionStatus = "ACTIVE"
	RedemptionStatusUsed     RedemptionStatus = "USED"
	RedemptionStatusReversed RedemptionStatus = "REVERSED"
)

// Redemption records when a customer redeems points for a reward
type Redemption struct {
	ID             string           `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CustomerID     string           `gorm:"type:uuid;not null;index"`
	RewardID       string           `gorm:"type:uuid;not null;index"`
	PointsDeducted int64            `gorm:"not null"`
	Status         RedemptionStatus `gorm:"type:varchar(20);not null;index"`
	Code           string           `gorm:"type:varchar(50);uniqueIndex"` // Discount/voucher code
	UsedAt         *time.Time
	ReversedAt     *time.Time
	ReversalReason *string `gorm:"type:text"`
	CreatedAt      time.Time `gorm:"not null"`
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
	
	// Relationships
	Customer *Customer `gorm:"foreignKey:CustomerID"`
	Reward   *Reward   `gorm:"foreignKey:RewardID"`
}

// TableName specifies the table name for the Redemption model
func (Redemption) TableName() string {
	return "redemptions"
}

