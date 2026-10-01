package config

import (
	"os"
)

// Config holds application-level configuration.
type Config struct {
	Port        string
	Environment string
}

// Load reads configuration from environment variables and returns a Config.
func Load() Config {
	return Config{
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENV", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
