package config

import (
	"os"
	"time"
)

type Config struct {
	ServerAddr        string
	AllowedOrigins    string
	AIProvider        string
	GigaChatKey       string
	GigaChatModel     string
	GigaChatScope     string
	GigaChatOAuthURL  string
	GigaChatChatURL   string
	GigaChatTimeout   time.Duration
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
		AIProvider:        getEnv("AI_PROVIDER", "mock"),
		GigaChatKey:       os.Getenv("GIGACHAT_AUTHORIZATION_KEY"),
		GigaChatModel:     getEnv("GIGACHAT_MODEL", "GigaChat-2"),
		GigaChatScope:     getEnv("GIGACHAT_SCOPE", "GIGACHAT_API_PERS"),
		GigaChatOAuthURL:  getEnv("GIGACHAT_OAUTH_URL", "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"),
		GigaChatChatURL:   getEnv("GIGACHAT_CHAT_URL", "https://api.giga.chat/v1/chat/completions"),
		GigaChatTimeout:   60 * time.Second,
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
