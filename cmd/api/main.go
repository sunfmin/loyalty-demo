package main

//go:generate protoc --go_out=. --go_opt=paths=source_relative ../../api/v1/loyalty.proto
//go:generate protoc --go_out=. --go_opt=paths=source_relative ../../api/v1/transaction.proto
//go:generate protoc --go_out=. --go_opt=paths=source_relative ../../api/v1/reward.proto
//go:generate protoc --go_out=. --go_opt=paths=source_relative ../../api/v1/campaign.proto

import (
	"fmt"
	"log"
	"net/http"
	"time"
	
	"github.com/opentracing/opentracing-go"
	"github.com/yourorg/loyalty-demo/handlers"
	"github.com/yourorg/loyalty-demo/internal/config"
	"github.com/yourorg/loyalty-demo/internal/middleware"
	"github.com/yourorg/loyalty-demo/services"
)

func main() {
	// Load configuration from environment
	cfg := config.Load()
	
	// Initialize NoopTracer for development (production will use Jaeger/Zipkin)
	opentracing.SetGlobalTracer(opentracing.NoopTracer{})
	
	// Connect to database
	dbConfig := config.LoadDatabaseConfig()
	db, err := config.NewDatabaseConnection(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	
	// Run migrations
	if err := services.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	
	log.Println("Database connection established and migrations complete")
	
	// Create services
	loyaltyService := services.NewLoyaltyService(db)
	transactionService := services.NewTransactionService(db)
	rewardService := services.NewRewardService(db, transactionService)
	campaignService := services.NewCampaignService(db)
	analyticsService := services.NewAnalyticsService(db)
	
	// Create handlers
	enrollmentHandler := handlers.NewEnrollmentHandler(loyaltyService)
	pointsHandler := handlers.NewPointsHandler(transactionService, loyaltyService) // Pass loyaltyService for tier evaluation
	rewardsHandler := handlers.NewRewardsHandler(rewardService)
	adminUIHandler := handlers.NewAdminUIHandler(loyaltyService, transactionService, rewardService, campaignService, analyticsService)
	
	// Create HTTP multiplexer
	mux := http.NewServeMux()
	
	// Register health check endpoint (no auth required)
	mux.HandleFunc("/health", healthCheckHandler)
	
	// Register loyalty endpoints (with auth middleware)
	mux.Handle("/v1/loyalty/enroll", middleware.Authentication(http.HandlerFunc(enrollmentHandler.HandleEnroll)))
	mux.Handle("/v1/loyalty/me", middleware.Authentication(http.HandlerFunc(enrollmentHandler.HandleGetStatus)))
	mux.HandleFunc("/v1/loyalty/tiers", enrollmentHandler.HandleListTiers) // Public endpoint
	
	// Register points endpoints (with auth middleware)
	mux.Handle("/v1/loyalty/points/earn", middleware.Authentication(http.HandlerFunc(pointsHandler.HandleEarnPoints)))
	mux.Handle("/v1/loyalty/transactions", middleware.Authentication(http.HandlerFunc(pointsHandler.HandleListTransactions)))
	
	// Register rewards endpoints
	mux.HandleFunc("/v1/loyalty/rewards", rewardsHandler.HandleListRewards) // Public catalog
	mux.Handle("/v1/loyalty/rewards/redeem", middleware.Authentication(http.HandlerFunc(rewardsHandler.HandleRedeemReward)))
	mux.Handle("/v1/loyalty/redemptions", middleware.Authentication(http.HandlerFunc(rewardsHandler.HandleListRedemptions)))
	
	// Register admin UI endpoints (wrapped with admin authentication middleware)
	mux.Handle("/admin", middleware.AdminAuthentication(http.HandlerFunc(adminUIHandler.HandleDashboard)))
	mux.Handle("/admin/adjust-points", middleware.AdminAuthentication(http.HandlerFunc(adminUIHandler.HandleAdjustPoints)))
	mux.Handle("/admin/campaigns/new", middleware.AdminAuthentication(http.HandlerFunc(adminUIHandler.HandleCreateCampaign)))
	mux.Handle("/admin/customers", middleware.AdminAuthentication(http.HandlerFunc(adminUIHandler.HandleCustomerSearch)))
	
	// Register admin API endpoints for HTMX (return HTML fragments, also need admin auth)
	mux.Handle("/admin/api/adjust-points", middleware.AdminAuthentication(http.HandlerFunc(adminUIHandler.HandleAdjustPointsSubmit)))
	mux.Handle("/admin/api/campaigns", middleware.AdminAuthentication(http.HandlerFunc(adminUIHandler.HandleCreateCampaignSubmit)))
	mux.Handle("/admin/api/customers/lookup", middleware.AdminAuthentication(http.HandlerFunc(adminUIHandler.HandleCustomerLookup)))
	
	// Apply middleware chain
	// Order: Recovery → Logging → Tracing → CORS
	handler := middleware.Recovery(
		middleware.Logging(
			middleware.Tracing(
				middleware.CORS(mux),
			),
		),
	)
	
	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}
	
	log.Printf("Starting loyalty API server on port %s", cfg.Server.Port)
	log.Printf("Environment: %s, Debug: %v", cfg.App.Environment, cfg.App.Debug)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","version":"1.0.0","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

