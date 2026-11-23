package config

import (
	"os"
	"time"
)

type Config struct {
	ServerAddr        string
	AllowedOrigins    string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

func Load() Config {
	return Config{
		ServerAddr: getEnv("SERVER_ADDR", ":8080"),
		AllowedOrigins: getEnv(
			"CORS_ALLOWED_ORIGINS",
			getEnv("CORS_ALLOWED_ORIGIN", "http://localhost:5173,http://localhost:8081"),
		),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      90 * time.Second,
		IdleTimeout:       60 * time.Second,
		ShutdownTimeout:   10 * time.Second,
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
