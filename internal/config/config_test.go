package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadYandexConfig(t *testing.T) {
	t.Setenv("YANDEX_API_KEY", " secret ")
	t.Setenv("YANDEX_FOLDER_ID", " folder ")
	t.Setenv("YANDEX_MODEL", "")
	t.Setenv("YANDEX_API_BASE_URL", "")
	t.Setenv("YANDEX_TIMEOUT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Yandex.APIKey != "secret" || cfg.Yandex.FolderID != "folder" {
		t.Fatalf("credentials were not trimmed: %+v", cfg.Yandex)
	}
	if cfg.Yandex.Model != "yandexgpt" {
		t.Fatalf("unexpected model %q", cfg.Yandex.Model)
	}
	if cfg.Yandex.ModelURI() != "gpt://folder/yandexgpt/latest" {
		t.Fatalf("unexpected model URI %q", cfg.Yandex.ModelURI())
	}
	if cfg.Yandex.BaseURL != "https://llm.api.cloud.yandex.net/v1" {
		t.Fatalf("unexpected base URL %q", cfg.Yandex.BaseURL)
	}
	if cfg.Yandex.Timeout != 75*time.Second {
		t.Fatalf("unexpected timeout %s", cfg.Yandex.Timeout)
	}
	if cfg.WriteTimeout != 120*time.Second {
		t.Fatalf("unexpected write timeout %s", cfg.WriteTimeout)
	}
}

func TestLoadYandexModelVersion(t *testing.T) {
	t.Setenv("YANDEX_API_KEY", "secret")
	t.Setenv("YANDEX_FOLDER_ID", "folder")
	t.Setenv("YANDEX_MODEL", "yandexgpt/rc")
	t.Setenv("YANDEX_API_BASE_URL", "https://ai.api.cloud.yandex.net/v1/")
	t.Setenv("YANDEX_TIMEOUT", "45s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Yandex.ModelURI() != "gpt://folder/yandexgpt/rc" {
		t.Fatalf("unexpected model URI %q", cfg.Yandex.ModelURI())
	}
	if cfg.Yandex.BaseURL != "https://ai.api.cloud.yandex.net/v1" {
		t.Fatalf("base URL was not trimmed: %q", cfg.Yandex.BaseURL)
	}
	if cfg.Yandex.Timeout != 45*time.Second {
		t.Fatalf("unexpected timeout %s", cfg.Yandex.Timeout)
	}
}

func TestLoadRejectsMissingYandexCredentials(t *testing.T) {
	t.Setenv("YANDEX_API_KEY", " ")
	t.Setenv("YANDEX_FOLDER_ID", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "YANDEX_API_KEY") {
		t.Fatalf("expected missing credentials error, got %v", err)
	}
}

func TestLoadRejectsInvalidTimeout(t *testing.T) {
	t.Setenv("YANDEX_API_KEY", "secret")
	t.Setenv("YANDEX_FOLDER_ID", "folder")
	t.Setenv("YANDEX_TIMEOUT", "soon")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "YANDEX_TIMEOUT") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}
