package main

import (
	"fmt"
	"log"

	"github.com/yourorg/loyalty-demo/internal/config"
	"github.com/yourorg/loyalty-demo/internal/models"
	"github.com/yourorg/loyalty-demo/services"
)

func main() {
	// Load configuration
	dbConfig := config.LoadDatabaseConfig()

	// Connect to database
	db, err := config.NewDatabaseConnection(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	if err := services.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Seeding membership tiers...")

	// Create Base tier (Level 0)
	baseTier := &models.MembershipTier{
		Name:                "Base",
		Level:               0,
		QualificationPoints: 0,
		EvaluationDays:      365,
		EarnRateMultiplier:  1.0,
		Description:         "Base tier for all new members. Earn 1x points on all purchases.",
	}
	result := db.Where("level = ?", 0).FirstOrCreate(baseTier)
	if result.Error != nil {
		log.Fatalf("Failed to create Base tier: %v", result.Error)
	}
	fmt.Printf("✓ Base tier (Level 0, 1.0x multiplier)\n")

	// Create Silver tier (Level 1)
	silverTier := &models.MembershipTier{
		Name:                "Silver",
		Level:               1,
		QualificationPoints: 500,
		EvaluationDays:      365,
		EarnRateMultiplier:  1.25,
		Description:         "Silver tier for loyal customers. Earn 1.25x points on all purchases. Requires 500 points earned in the last 12 months.",
	}
	result = db.Where("level = ?", 1).FirstOrCreate(silverTier)
	if result.Error != nil {
		log.Fatalf("Failed to create Silver tier: %v", result.Error)
	}
	fmt.Printf("✓ Silver tier (Level 1, 1.25x multiplier, 500 points required)\n")

	// Create Gold tier (Level 2)
	goldTier := &models.MembershipTier{
		Name:                "Gold",
		Level:               2,
		QualificationPoints: 1000,
		EvaluationDays:      365,
		EarnRateMultiplier:  1.5,
		Description:         "Gold tier for VIP customers. Earn 1.5x points on all purchases. Requires 1000 points earned in the last 12 months.",
	}
	result = db.Where("level = ?", 2).FirstOrCreate(goldTier)
	if result.Error != nil {
		log.Fatalf("Failed to create Gold tier: %v", result.Error)
	}
	fmt.Printf("✓ Gold tier (Level 2, 1.5x multiplier, 1000 points required)\n")

	// Create Platinum tier (Level 3)
	platinumTier := &models.MembershipTier{
		Name:                "Platinum",
		Level:               3,
		QualificationPoints: 2000,
		EvaluationDays:      365,
		EarnRateMultiplier:  2.0,
		Description:         "Platinum tier for elite customers. Earn 2.0x points on all purchases. Requires 2000 points earned in the last 12 months.",
	}
	result = db.Where("level = ?", 3).FirstOrCreate(platinumTier)
	if result.Error != nil {
		log.Fatalf("Failed to create Platinum tier: %v", result.Error)
	}
	fmt.Printf("✓ Platinum tier (Level 3, 2.0x multiplier, 2000 points required)\n")

	log.Println("\n✅ Tier seeding complete! 4 tiers created.")
	log.Println("\nTier Progression:")
	log.Println("  Base (0 points) → Silver (500 points) → Gold (1000 points) → Platinum (2000 points)")
}

