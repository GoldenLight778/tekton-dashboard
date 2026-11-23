package config

import (
	"os"
)

// Config holds all configuration options for the platform backend.
type Config struct {
	Addr       string
	DBPath     string
	JWTSecret  string
	SecretKey  string
	Kubeconfig string
	LogLevel   string
}

// Load loads the configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Addr:       getEnv("PLATFORM_ADDR", ":8080"),
		DBPath:     getEnv("PLATFORM_DB_PATH", "./data/platform.db"),
		JWTSecret:  getEnv("PLATFORM_JWT_SECRET", "dev-secret-change-in-production"),
		SecretKey:  getEnv("PLATFORM_SECRET_KEY", "0123456789abcdef0123456789abcdef"), // 32 bytes AES key for dev
		Kubeconfig: getEnv("PLATFORM_KUBECONFIG", ""),
		LogLevel:   getEnv("PLATFORM_LOG_LEVEL", "info"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
