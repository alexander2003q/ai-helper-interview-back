package httpapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-helper-interview/backend/internal/delivery/httpapi"
	"ai-helper-interview/backend/internal/provider/memory"
	"ai-helper-interview/backend/internal/usecase"
)

func TestCreateQuestionnaire(t *testing.T) {
	t.Parallel()

	router := newTestRouter()
	body := []byte(`{
      "topics": ["React"],
      "vacancy": "React, TypeScript",
      "level": "middle",
      "responseFormat": "text",
      "trainingConfig": {
        "theory": {"enabled": true, "count": 8},
        "code": {"enabled": true, "count": 2},
        "experience": {"enabled": true, "count": 3},
        "hints": {"enabled": true, "perTask": 2},
        "attemptsPerTask": 3
      }
    }`)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/questionnaires", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://localhost:5173")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}

	if origin := response.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:5173" {
		t.Fatalf("unexpected CORS origin: %q", origin)
	}

	var payload struct {
		Topic     string `json:"topic"`
		Level     string `json:"level"`
		Total     int    `json:"total"`
		Questions []struct {
			Prompt string `json:"prompt"`
		} `json:"questions"`
	}

	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.Topic != "React" || payload.Level != "Middle" {
		t.Fatalf("unexpected questionnaire metadata: %+v", payload)
	}

	if payload.Total != 8 || len(payload.Questions) != 8 {
		t.Fatalf("expected 8 questions, got total=%d length=%d", payload.Total, len(payload.Questions))
	}
}

func TestCreateQuestionnaireValidationError(t *testing.T) {
	t.Parallel()

	router := newTestRouter()
	body := []byte(`{"topics":[],"level":"middle","responseFormat":"text"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/questionnaires", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnprocessableEntity, response.Code, response.Body.String())
	}
}

func TestCORSAllowsSwaggerUI(t *testing.T) {
	t.Parallel()

	router := newTestRouter()
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/questionnaires", nil)
	request.Header.Set("Origin", "http://localhost:8081")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
	if origin := response.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:8081" {
		t.Fatalf("unexpected CORS origin: %q", origin)
	}
}

func newTestRouter() http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	generator := memory.NewTheoryQuestionGenerator()
	createQuestionnaire := usecase.NewCreateQuestionnaire(generator)
	handler := httpapi.NewQuestionnaireHandler(createQuestionnaire, logger)

	return httpapi.NewRouter(
		handler,
		"http://localhost:5173,http://localhost:8081",
		logger,
	)
}
