package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application.
type Config struct {
	App        AppConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	JWT        JWTConfig
	Threads    ThreadsConfig
	AI         AIConfig
	Shortener  ShortenerConfig
	Encryption EncryptionConfig
}

// AppConfig holds general application settings.
type AppConfig struct {
	Name   string
	Env    string
	Port   string
	Secret string
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// JWTConfig holds JWT authentication settings.
type JWTConfig struct {
	Secret        string
	Expiry        time.Duration
	RefreshExpiry time.Duration
}

// ThreadsConfig holds Meta Threads API settings.
type ThreadsConfig struct {
	AppID       string
	AppSecret   string
	RedirectURI string
}

// AIConfig holds AI service settings.
type AIConfig struct {
	APIURL string
	APIKey string
}

// ShortenerConfig holds URL shortener settings.
type ShortenerConfig struct {
	Domain string
	Port   string
}

// EncryptionConfig holds encryption settings.
type EncryptionConfig struct {
	Key string
}

// Load reads configuration from environment variables and returns a *Config.
// It attempts to load a .env file if present (errors are silently ignored).
func Load() *Config {
	// Load .env file if it exists; ignore error if not found
	_ = godotenv.Load()

	return &Config{
		App: AppConfig{
			Name:   getEnv("APP_NAME", "threads-affiliate-automation"),
			Env:    getEnv("APP_ENV", "development"),
			Port:   getEnv("APP_PORT", "3000"),
			Secret: getEnv("APP_SECRET", ""),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "threads_affiliate"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:        getEnv("JWT_SECRET", ""),
			Expiry:        getEnvDuration("JWT_EXPIRY", 15*time.Minute),
			RefreshExpiry: getEnvDuration("JWT_REFRESH_EXPIRY", 7*24*time.Hour),
		},
		Threads: ThreadsConfig{
			AppID:       getEnv("THREADS_APP_ID", ""),
			AppSecret:   getEnv("THREADS_APP_SECRET", ""),
			RedirectURI: getEnv("THREADS_REDIRECT_URI", "http://localhost:3000/auth/callback"),
		},
		AI: AIConfig{
			APIURL: getEnv("AI_API_URL", ""),
			APIKey: getEnv("AI_API_KEY", ""),
		},
		Shortener: ShortenerConfig{
			Domain: getEnv("SHORTENER_DOMAIN", "localhost"),
			Port:   getEnv("SHORTENER_PORT", "3001"),
		},
		Encryption: EncryptionConfig{
			Key: getEnv("ENCRYPTION_KEY", ""),
		},
	}
}

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt retrieves an environment variable as an integer or returns a default.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getEnvDuration retrieves an environment variable as a time.Duration or returns a default.
// Accepts Go duration strings (e.g., "15m", "24h", "720h").
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
