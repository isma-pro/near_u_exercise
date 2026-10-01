package config

import (
	"os"
	"time"
)

// Config holds application-level configuration.
type Config struct {
	Port            string
	Environment     string
	ShutdownTimeout time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	DatabaseURL     string
}

// Load reads configuration from environment variables and returns a Config.
func Load() Config {
	return Config{
		Port:            getEnv("PORT", "8080"),
		Environment:     getEnv("ENV", "development"),
		ShutdownTimeout: getDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		ReadTimeout:     getDuration("SERVER_READ_TIMEOUT", 5*time.Second),
		WriteTimeout:    getDuration("SERVER_WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:     getDuration("SERVER_IDLE_TIMEOUT", 120*time.Second),
		DatabaseURL:     getEnv("DATABASE_URL", ""),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return d
}
