package models

import (
	"time"
)

// PromotionalCampaign defines temporary bonus earning opportunities
type PromotionalCampaign struct {
	ID              string  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name            string  `gorm:"type:varchar(255);not null"`
	Description     string  `gorm:"type:text"`
	StartDate       time.Time `gorm:"not null;index:idx_campaign_dates"`
	EndDate         time.Time `gorm:"not null;index:idx_campaign_dates"`
	PointMultiplier float64 `gorm:"type:decimal(3,2);not null;default:1.0"` // e.g., 2.0 for double points
	BonusPoints     int64   `gorm:"default:0"`                              // Fixed bonus (alternative to multiplier)
	IsActive        bool    `gorm:"not null;default:true;index"`
	Conditions      string  `gorm:"type:jsonb"`                            // Flexible rules (min purchase, eligible products, etc.)
	Priority        int     `gorm:"not null;default:0"`                    // Higher priority when multiple campaigns overlap
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// TableName specifies the table name for the PromotionalCampaign model
func (PromotionalCampaign) TableName() string {
	return "promotional_campaigns"
}

