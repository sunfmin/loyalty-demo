package services

import (
	"github.com/yourorg/loyalty-demo/internal/models"
	"gorm.io/gorm"
)

// AutoMigrate runs all necessary database migrations for loyalty services.
// External applications using this package MUST call this function before using services.
// This allows external apps to set up the database schema without importing internal models.
//
// Example usage:
//
//	db, _ := gorm.Open(...)
//	if err := services.AutoMigrate(db); err != nil {
//	    log.Fatal(err)
//	}
//	loyaltyService := services.NewLoyaltyService(db)
func AutoMigrate(db *gorm.DB) error {
	// Migrate all models in correct dependency order
	// Base tables first, then tables with foreign keys
	return db.AutoMigrate(
		&models.MembershipTier{},
		&models.Customer{},
		&models.PromotionalCampaign{},
		&models.Reward{},
		&models.Redemption{},
		&models.PointTransaction{},
	)
}

