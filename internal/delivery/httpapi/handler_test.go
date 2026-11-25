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
		Topic           string `json:"topic"`
		Level           string `json:"level"`
		Total           int    `json:"total"`
		AttemptsPerTask int    `json:"attemptsPerTask"`
		Hints           struct {
			Enabled bool `json:"enabled"`
			PerTask int  `json:"perTask"`
		} `json:"hints"`
		Tasks []struct {
			Kind  string   `json:"kind"`
			Hints []string `json:"hints"`
		} `json:"tasks"`
	}

	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.Topic != "React" || payload.Level != "Middle" {
		t.Fatalf("unexpected questionnaire metadata: %+v", payload)
	}

	if payload.AttemptsPerTask != 3 || !payload.Hints.Enabled || payload.Hints.PerTask != 2 {
		t.Fatalf("unexpected training settings: %+v", payload)
	}

	questions, codeTasks := 0, 0
	for _, task := range payload.Tasks {
		switch task.Kind {
		case "question":
			questions++
		case "code":
			codeTasks++
			if len(task.Hints) != 2 {
				t.Fatalf("expected 2 hints, got %d", len(task.Hints))
			}
		default:
			t.Fatalf("unexpected task kind %q", task.Kind)
		}
	}

	if payload.Total != 10 || questions != 8 || codeTasks != 2 {
		t.Fatalf(
			"expected 8 questions and 2 code tasks, got total=%d questions=%d code=%d",
			payload.Total,
			questions,
			codeTasks,
		)
	}
}

func TestCreateQuestionnaireCodeOnly(t *testing.T) {
	t.Parallel()

	router := newTestRouter()
	body := []byte(`{
      "topics": ["React"],
      "level": "junior",
      "responseFormat": "text",
      "trainingConfig": {
        "theory": {"enabled": false, "count": 8},
        "code": {"enabled": true, "count": 1},
        "hints": {"enabled": false, "perTask": 2},
        "attemptsPerTask": 1
      }
    }`)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/questionnaires", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}

	var payload struct {
		Total int `json:"total"`
		Hints struct {
			Enabled bool `json:"enabled"`
			PerTask int  `json:"perTask"`
		} `json:"hints"`
		Tasks []struct {
			Kind  string   `json:"kind"`
			Hints []string `json:"hints"`
		} `json:"tasks"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.Total != 1 || len(payload.Tasks) != 1 || payload.Tasks[0].Kind != "code" {
		t.Fatalf("expected one code task, got %+v", payload)
	}
	if payload.Hints.Enabled || payload.Hints.PerTask != 0 || len(payload.Tasks[0].Hints) != 0 {
		t.Fatalf("expected hints to be disabled, got %+v", payload)
	}
}

func TestCreateQuestionnaireRequiresTask(t *testing.T) {
	t.Parallel()

	router := newTestRouter()
	body := []byte(`{
      "topics": ["React"],
      "level": "middle",
      "responseFormat": "text",
      "trainingConfig": {
        "theory": {"enabled": false, "count": 8},
        "code": {"enabled": false, "count": 2},
        "hints": {"enabled": true, "perTask": 2},
        "attemptsPerTask": 3
      }
    }`)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/questionnaires", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnprocessableEntity, response.Code, response.Body.String())
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
	questions := memory.NewTheoryQuestionGenerator()
	codeTasks := memory.NewCodeTaskGenerator()
	createQuestionnaire := usecase.NewCreateQuestionnaire(questions, codeTasks)
	handler := httpapi.NewQuestionnaireHandler(createQuestionnaire, logger)

	return httpapi.NewRouter(
		handler,
		"http://localhost:5173,http://localhost:8081",
		logger,
	)
}
