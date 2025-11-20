package config

import (
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Tracing  TracingConfig
	Loyalty  LoyaltyConfig
	Auth     AuthConfig
	App      AppConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         string
	Host         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// TracingConfig holds distributed tracing configuration
type TracingConfig struct {
	Enabled          bool
	Backend          string // "jaeger", "zipkin", etc.
	JaegerAgentHost  string
	JaegerAgentPort  string
	JaegerSamplerType string
	JaegerSamplerParam float64
}

// LoyaltyConfig holds loyalty program business rules configuration
type LoyaltyConfig struct {
	PointsEarnRate       float64
	PointsExpirationDays int
	TierEvaluationDays   int
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret     string
	JWTExpiration int
	AdminEmails   string
}

// AppConfig holds general application configuration
type AppConfig struct {
	Environment string
	Debug       bool
	LogLevel    string
}

// Load loads all configuration from environment variables
func Load() *Config {
	return &Config{
		Server:   loadServerConfig(),
		Database: *LoadDatabaseConfig(),
		Tracing:  loadTracingConfig(),
		Loyalty:  loadLoyaltyConfig(),
		Auth:     loadAuthConfig(),
		App:      loadAppConfig(),
	}
}

func loadServerConfig() ServerConfig {
	readTimeout, _ := strconv.Atoi(getEnv("SERVER_READ_TIMEOUT", "30"))
	writeTimeout, _ := strconv.Atoi(getEnv("SERVER_WRITE_TIMEOUT", "30"))

	return ServerConfig{
		Port:         getEnv("SERVER_PORT", "8080"),
		Host:         getEnv("SERVER_HOST", "0.0.0.0"),
		ReadTimeout:  time.Duration(readTimeout) * time.Second,
		WriteTimeout: time.Duration(writeTimeout) * time.Second,
	}
}

func loadTracingConfig() TracingConfig {
	enabled, _ := strconv.ParseBool(getEnv("TRACING_ENABLED", "false"))
	samplerParam, _ := strconv.ParseFloat(getEnv("JAEGER_SAMPLER_PARAM", "1"), 64)

	return TracingConfig{
		Enabled:            enabled,
		Backend:            getEnv("TRACING_BACKEND", "jaeger"),
		JaegerAgentHost:    getEnv("JAEGER_AGENT_HOST", "localhost"),
		JaegerAgentPort:    getEnv("JAEGER_AGENT_PORT", "6831"),
		JaegerSamplerType:  getEnv("JAEGER_SAMPLER_TYPE", "const"),
		JaegerSamplerParam: samplerParam,
	}
}

func loadLoyaltyConfig() LoyaltyConfig {
	earnRate, _ := strconv.ParseFloat(getEnv("POINTS_EARN_RATE", "1.0"), 64)
	expirationDays, _ := strconv.Atoi(getEnv("POINTS_EXPIRATION_DAYS", "365"))
	evaluationDays, _ := strconv.Atoi(getEnv("TIER_EVALUATION_DAYS", "365"))

	return LoyaltyConfig{
		PointsEarnRate:       earnRate,
		PointsExpirationDays: expirationDays,
		TierEvaluationDays:   evaluationDays,
	}
}

func loadAuthConfig() AuthConfig {
	expiration, _ := strconv.Atoi(getEnv("JWT_EXPIRATION", "3600"))

	return AuthConfig{
		JWTSecret:     getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		JWTExpiration: expiration,
		AdminEmails:   getEnv("ADMIN_EMAILS", "admin@example.com"),
	}
}

func loadAppConfig() AppConfig {
	debug, _ := strconv.ParseBool(getEnv("APP_DEBUG", "true"))

	return AppConfig{
		Environment: getEnv("APP_ENV", "development"),
		Debug:       debug,
		LogLevel:    getEnv("LOG_LEVEL", "debug"),
	}
}

