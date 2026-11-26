package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
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
	Yandex            YandexConfig
}

type YandexConfig struct {
	APIKey   string
	FolderID string
	Model    string
	BaseURL  string
	Timeout  time.Duration
}

func (c YandexConfig) ModelURI() string {
	model := c.Model
	if !strings.Contains(model, "/") {
		model += "/latest"
	}

	return "gpt://" + c.FolderID + "/" + model
}

func Load() (Config, error) {
	yandex, err := loadYandex()
	if err != nil {
		return Config{}, err
	}

	return Config{
		ServerAddr: getEnv("SERVER_ADDR", ":8080"),
		AllowedOrigins: getEnv(
			"CORS_ALLOWED_ORIGINS",
			getEnv("CORS_ALLOWED_ORIGIN", "http://localhost:5173,http://localhost:8081"),
		),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       60 * time.Second,
		ShutdownTimeout:   10 * time.Second,
		Yandex:            yandex,
	}, nil
}

func loadYandex() (YandexConfig, error) {
	apiKey := strings.TrimSpace(os.Getenv("YANDEX_API_KEY"))
	folderID := strings.TrimSpace(os.Getenv("YANDEX_FOLDER_ID"))
	if apiKey == "" || folderID == "" {
		return YandexConfig{}, errors.New("YANDEX_API_KEY and YANDEX_FOLDER_ID are required")
	}

	timeout := 75 * time.Second
	if rawTimeout := strings.TrimSpace(os.Getenv("YANDEX_TIMEOUT")); rawTimeout != "" {
		parsedTimeout, err := time.ParseDuration(rawTimeout)
		if err != nil {
			return YandexConfig{}, fmt.Errorf("YANDEX_TIMEOUT: %w", err)
		}
		if parsedTimeout <= 0 {
			return YandexConfig{}, errors.New("YANDEX_TIMEOUT must be positive")
		}
		timeout = parsedTimeout
	}

	model := strings.TrimSpace(getEnv("YANDEX_MODEL", "yandexgpt"))
	if model == "" {
		return YandexConfig{}, errors.New("YANDEX_MODEL must not be empty")
	}

	baseURL := strings.TrimRight(strings.TrimSpace(getEnv(
		"YANDEX_API_BASE_URL",
		"https://llm.api.cloud.yandex.net/v1",
	)), "/")
	if baseURL == "" {
		return YandexConfig{}, errors.New("YANDEX_API_BASE_URL must not be empty")
	}

	return YandexConfig{
		APIKey:   apiKey,
		FolderID: folderID,
		Model:    model,
		BaseURL:  baseURL,
		Timeout:  timeout,
	}, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
