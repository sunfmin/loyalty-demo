package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host            string
	Port            string
	Database        string
	User            string
	Password        string
	SSLMode         string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
}

// LoadDatabaseConfig loads database configuration from environment variables
func LoadDatabaseConfig() *DatabaseConfig {
	maxIdleConns, _ := strconv.Atoi(getEnv("DATABASE_MAX_IDLE_CONNS", "10"))
	maxOpenConns, _ := strconv.Atoi(getEnv("DATABASE_MAX_OPEN_CONNS", "100"))
	connMaxLifetime, _ := strconv.Atoi(getEnv("DATABASE_CONN_MAX_LIFETIME", "3600"))

	return &DatabaseConfig{
		Host:            getEnv("DATABASE_HOST", "localhost"),
		Port:            getEnv("DATABASE_PORT", "5432"),
		Database:        getEnv("DATABASE_NAME", "loyalty"),
		User:            getEnv("DATABASE_USER", "postgres"),
		Password:        getEnv("DATABASE_PASSWORD", "postgres"),
		SSLMode:         getEnv("DATABASE_SSL_MODE", "disable"),
		MaxIdleConns:    maxIdleConns,
		MaxOpenConns:    maxOpenConns,
		ConnMaxLifetime: time.Duration(connMaxLifetime) * time.Second,
	}
}

// NewDatabaseConnection creates a new GORM database connection with connection pooling
func NewDatabaseConnection(config *DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.Database,
		config.SSLMode,
	)

	// Configure GORM logger based on environment
	logLevel := logger.Silent
	if getEnv("APP_DEBUG", "false") == "true" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL database for connection pool configuration
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)

	log.Printf("Database connection established: %s:%s/%s", config.Host, config.Port, config.Database)
	return db, nil
}

// HealthCheck performs a database health check
func HealthCheck(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// getEnv gets environment variable with a default fallback
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

