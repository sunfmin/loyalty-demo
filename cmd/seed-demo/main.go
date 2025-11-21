package main

import (
	"context"
	"fmt"
	"log"
	"time"

	loyaltyv1 "github.com/yourorg/loyalty-demo/api/v1"
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

	log.Println("Seeding demo data...")

	// Get base tier
	var baseTier models.MembershipTier
	db.Where("level = ?", 0).First(&baseTier)

	// Create services
	loyaltyService := services.NewLoyaltyService(db)
	transactionService := services.NewTransactionService(db)
	rewardService := services.NewRewardService(db, transactionService)

	ctx := context.Background()

	// 1. Create sample customers
	log.Println("\n1. Creating sample customers...")
	
	customers := []struct {
		accountID string
		name      string
	}{
		{"alice", "Alice Johnson"},
		{"bob", "Bob Smith"},
		{"carol", "Carol Davis"},
	}

	for _, c := range customers {
		customer, err := loyaltyService.EnrollCustomer(ctx, &loyaltyv1.EnrollCustomerRequest{}, c.accountID)
		if err != nil {
			log.Printf("   Customer %s already exists, skipping", c.name)
		} else {
			fmt.Printf("   ✓ %s enrolled (ID: %s, Referral Code: %s)\n", c.name, customer.AccountId, customer.ReferralCode)
		}
	}

	// 2. Give customers some points
	log.Println("\n2. Adding points to customers...")
	
	purchases := []struct {
		accountID string
		amount    int64
		desc      string
	}{
		{"alice", 10000, "Purchase at Main Store - $100"},
		{"alice", 5000, "Purchase at Online Store - $50"},
		{"bob", 7500, "Purchase at Main Store - $75"},
		{"carol", 3000, "Purchase at Mall Location - $30"},
	}

	for _, p := range purchases {
		referenceID := fmt.Sprintf("order-%s-%d", p.accountID, time.Now().UnixNano())
		_, err := transactionService.EarnPoints(ctx, &loyaltyv1.EarnPointsRequest{
			Amount:        p.amount,
			ReferenceId:   referenceID,
			ReferenceType: "ORDER",
			Description:   p.desc,
		}, p.accountID)
		if err != nil {
			log.Printf("   Error adding points for %s: %v", p.accountID, err)
		} else {
			points := p.amount / 100
			fmt.Printf("   ✓ %s earned %d points from %s\n", p.accountID, points, p.desc)
		}
	}

	// 3. Create sample rewards
	log.Println("\n3. Creating sample rewards...")
	
	rewards := []struct {
		name      string
		rewardType models.RewardType
		pointCost int64
	}{
		{"$5 Discount Code", models.RewardTypeDiscount, 500},
		{"$10 Discount Code", models.RewardTypeDiscount, 1000},
		{"Free Coffee", models.RewardTypeFreeItem, 100},
		{"Free Dessert", models.RewardTypeFreeItem, 150},
		{"$25 Voucher", models.RewardTypeVoucher, 2500},
	}

	for _, r := range rewards {
		reward := &models.Reward{
			Name:        r.name,
			Description: fmt.Sprintf("Redeem %d points for %s", r.pointCost, r.name),
			Type:        r.rewardType,
			PointCost:   r.pointCost,
			IsActive:    true,
			Metadata:    `{}`,
		}
		if err := db.FirstOrCreate(reward, "name = ?", r.name).Error; err != nil {
			log.Printf("   Error creating reward %s: %v", r.name, err)
		} else {
			fmt.Printf("   ✓ %s (%d points)\n", r.name, r.pointCost)
		}
	}

	// 4. Create a sample campaign
	log.Println("\n4. Creating sample campaign...")
	
	now := time.Now()
	campaign := &models.PromotionalCampaign{
		Name:            "Holiday Double Points",
		Description:     "Earn 2x points on all purchases during the holiday season",
		StartDate:       now.Add(-24 * time.Hour), // Started yesterday
		EndDate:         now.Add(30 * 24 * time.Hour), // Ends in 30 days
		PointMultiplier: 2.0,
		BonusPoints:     0,
		IsActive:        true,
		Conditions:      `{}`,
		Priority:        10,
	}
	
	if err := db.FirstOrCreate(campaign, "name = ?", campaign.Name).Error; err != nil {
		log.Printf("   Error creating campaign: %v", err)
	} else {
		fmt.Printf("   ✓ %s (2.0x multiplier, active now)\n", campaign.Name)
	}

	// 5. Create some sample redemptions
	log.Println("\n5. Creating sample redemptions...")
	
	// Get Alice's customer record
	var alice models.Customer
	if err := db.Where("account_id = ?", "alice").First(&alice).Error; err == nil {
		// Get Free Coffee reward
		var freeCoffee models.Reward
		if err := db.Where("name = ?", "Free Coffee").First(&freeCoffee).Error; err == nil {
			// Check if Alice has enough points
			if alice.CurrentBalance >= freeCoffee.PointCost {
				_, err := rewardService.RedeemReward(ctx, &loyaltyv1.RedeemRewardRequest{
					RewardId: freeCoffee.ID,
				}, "alice")
				if err != nil {
					log.Printf("   Redemption already exists or error: %v", err)
				} else {
					fmt.Printf("   ✓ Alice redeemed Free Coffee (100 points)\n")
				}
			}
		}
	}

	log.Println("\n✅ Demo data seeding complete!")
	log.Println("\nYou can now test the system with these accounts:")
	log.Println("  - alice (has points and transactions)")
	log.Println("  - bob (has some points)")
	log.Println("  - carol (has fewer points)")
	log.Println("\nUse these account IDs when testing the admin interface or API.")
}

