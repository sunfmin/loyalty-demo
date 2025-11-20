package models

import (
	"time"
)

// MembershipTier defines tier levels with qualification thresholds and benefits
type MembershipTier struct {
	ID                  string  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name                string  `gorm:"type:varchar(50);not null;uniqueIndex"`
	Level               int     `gorm:"not null;uniqueIndex"` // 0=Base, 1=Silver, 2=Gold, etc.
	QualificationPoints int64   `gorm:"not null"`             // Points needed in evaluation period
	EvaluationDays      int     `gorm:"not null;default:365"` // Rolling period (default 12 months)
	EarnRateMultiplier  float64 `gorm:"type:decimal(3,2);not null;default:1.0"` // e.g., 1.5 for 1.5x
	Description         string  `gorm:"type:text"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// TableName specifies the table name for the MembershipTier model
func (MembershipTier) TableName() string {
	return "membership_tiers"
}

