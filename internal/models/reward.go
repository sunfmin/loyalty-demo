package models

import (
	"time"
)

// RewardType represents the type of reward
type RewardType string

const (
	RewardTypeDiscount RewardType = "DISCOUNT"
	RewardTypeVoucher  RewardType = "VOUCHER"
	RewardTypeFreeItem RewardType = "FREE_ITEM"
)

// Reward defines available rewards that can be redeemed with points
type Reward struct {
	ID          string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string     `gorm:"type:varchar(255);not null"`
	Description string     `gorm:"type:text"`
	Type        RewardType `gorm:"type:varchar(20);not null"`
	PointCost   int64      `gorm:"not null;index"`
	IsActive    bool       `gorm:"not null;index"` // Removed default:true to allow explicit false values
	Metadata    string     `gorm:"type:jsonb"` // Flexible data (discount %, product ID, etc.)
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TableName specifies the table name for the Reward model
func (Reward) TableName() string {
	return "rewards"
}

