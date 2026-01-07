// Package config handles configuration loading for blayzen-sip
package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for blayzen-sip
type Config struct {
	// SIP Server
	SIPHost      string
	SIPPort      int
	SIPTransport string
	RTPPortMin   int
	RTPPortMax   int

	// REST API
	APIHost string
	APIPort int
	GinMode string

	// Database
	DatabaseURL        string
	DBMaxOpenConns     int
	DBMaxIdleConns     int
	DBConnMaxLifetime  time.Duration

	// Cache
	ValkeyURL      string
	ValkeyPassword string
	ValkeyDB       int
	CacheRouteTTL  time.Duration

	// WebSocket
	DefaultWebSocketURL string
	WSReadTimeout       time.Duration
	WSWriteTimeout      time.Duration
	WSPingInterval      time.Duration

	// Logging
	LogLevel  string
	LogFormat string

	// Security
	APIAuthEnabled bool

	// Metrics
	MetricsEnabled bool
	MetricsPath    string
}

// Load loads configuration from environment variables
func Load() *Config {
	// Load .env file if it exists
	_ = godotenv.Load()

	return &Config{
		// SIP Server
		SIPHost:      getEnv("SIP_HOST", "0.0.0.0"),
		SIPPort:      getEnvInt("SIP_PORT", 5060),
		SIPTransport: getEnv("SIP_TRANSPORT", "udp"),
		RTPPortMin:   getEnvInt("RTP_PORT_MIN", 10000),
		RTPPortMax:   getEnvInt("RTP_PORT_MAX", 10100),

		// REST API
		APIHost: getEnv("API_HOST", "0.0.0.0"),
		APIPort: getEnvInt("API_PORT", 8080),
		GinMode: getEnv("GIN_MODE", "debug"),

		// Database
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://blayzen:blayzen@localhost:5432/blayzen_sip?sslmode=disable"),
		DBMaxOpenConns:     getEnvInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:     getEnvInt("DB_MAX_IDLE_CONNS", 5),
		DBConnMaxLifetime:  getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),

		// Cache
		ValkeyURL:      getEnv("VALKEY_URL", "localhost:6379"),
		ValkeyPassword: getEnv("VALKEY_PASSWORD", ""),
		ValkeyDB:       getEnvInt("VALKEY_DB", 0),
		CacheRouteTTL:  getEnvDuration("CACHE_ROUTE_TTL", 5*time.Minute),

		// WebSocket
		DefaultWebSocketURL: getEnv("DEFAULT_WEBSOCKET_URL", "ws://localhost:8081/ws"),
		WSReadTimeout:       getEnvDuration("WS_READ_TIMEOUT", 60*time.Second),
		WSWriteTimeout:      getEnvDuration("WS_WRITE_TIMEOUT", 10*time.Second),
		WSPingInterval:      getEnvDuration("WS_PING_INTERVAL", 30*time.Second),

		// Logging
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "text"),

		// Security
		APIAuthEnabled: getEnvBool("API_AUTH_ENABLED", true),

		// Metrics
		MetricsEnabled: getEnvBool("METRICS_ENABLED", true),
		MetricsPath:    getEnv("METRICS_PATH", "/metrics"),
	}
}

// getEnv returns environment variable or default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt returns environment variable as int or default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

// getEnvBool returns environment variable as bool or default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

// getEnvDuration returns environment variable as duration or default value
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

