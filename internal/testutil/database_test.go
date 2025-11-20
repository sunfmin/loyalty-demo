package testutil

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/yourorg/loyalty-demo/services"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SetupTestDB creates a PostgreSQL test container and returns a GORM connection
// with automatic cleanup function. This function:
// - Starts a PostgreSQL container with testcontainers
// - Waits for database to be ready
// - Creates a GORM connection
// - Runs AutoMigrate to set up schema
// - Returns cleanup function to terminate container
//
// Usage:
//
//	func TestSomething(t *testing.T) {
//	    db, cleanup := SetupTestDB(t)
//	    defer cleanup()
//	    defer TruncateTables(db, "customers", "point_transactions")
//	    // ... test code ...
//	}
func SetupTestDB(t *testing.T) (*gorm.DB, func()) {
	ctx := context.Background()

	// Create PostgreSQL container
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}

	// Cleanup function to terminate container
	cleanup := func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		cleanup()
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// Connect using GORM with silent logger for tests
	db, err := gorm.Open(postgres.Open(connStr), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		cleanup()
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Run GORM AutoMigrate to set up schema
	if err := services.AutoMigrate(db); err != nil {
		cleanup()
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db, cleanup
}

// TruncateTables truncates specified tables in reverse dependency order with CASCADE
// This ensures foreign key constraints don't prevent truncation
// Tables should be provided in the order they should be truncated (children before parents)
//
// Usage:
//
//	defer TruncateTables(db, "point_transactions", "redemptions", "customers")
func TruncateTables(db *gorm.DB, tables ...string) {
	// Truncate in reverse order (children before parents)
	for i := len(tables) - 1; i >= 0; i-- {
		table := tables[i]
		if err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)).Error; err != nil {
			// Log error but don't fail - test cleanup should be best-effort
			fmt.Printf("Warning: failed to truncate table %s: %v\n", table, err)
		}
	}
}

