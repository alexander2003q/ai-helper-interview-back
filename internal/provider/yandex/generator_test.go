package yandex

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"ai-helper-interview/backend/internal/config"
	"ai-helper-interview/backend/internal/domain"
	"ai-helper-interview/backend/internal/ports"
)

func TestGenerateQuestions(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Api-Key test-key" {
			t.Errorf("unexpected authorization %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("x-folder-id") != "folder" {
			t.Errorf("unexpected folder %q", r.Header.Get("x-folder-id"))
		}
		if r.Header.Get("x-data-logging-enabled") != "false" {
			t.Errorf("data logging was not disabled")
		}

		var request chatRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if request.Model != "gpt://folder/yandexgpt/latest" {
			t.Errorf("unexpected model %s", request.Model)
		}
		if request.MaxTokens != 1200 || request.Temperature != temperature || request.Stream {
			t.Errorf("unexpected generation options: %+v", request)
		}
		if request.ResponseFormat.Type != "json_schema" || request.ResponseFormat.JSONSchema.Name != theorySchemaName {
			t.Errorf("unexpected response format: %+v", request.ResponseFormat)
		}

		var schema struct {
			Properties struct {
				Questions struct {
					MinItems int `json:"minItems"`
					MaxItems int `json:"maxItems"`
				} `json:"questions"`
			} `json:"properties"`
		}
		if err := json.Unmarshal(request.ResponseFormat.JSONSchema.Schema, &schema); err != nil {
			t.Errorf("decode schema: %v", err)
		}
		if schema.Properties.Questions.MinItems != 2 || schema.Properties.Questions.MaxItems != 2 {
			t.Errorf("schema does not require 2 questions: %+v", schema.Properties.Questions)
		}

		foundTopic := false
		for _, message := range request.Messages {
			if message.Role == "user" && strings.Contains(message.Content, "React") && strings.Contains(message.Content, "Middle") {
				foundTopic = true
			}
		}
		if !foundTopic {
			t.Errorf("prompt does not contain the interview settings: %+v", request.Messages)
		}

		writeCompletion(w, validQuestionsJSON(2))
	}))
	t.Cleanup(server.Close)

	questions, err := newTestGenerator(server.URL).Generate(context.Background(), testInput(2))
	if err != nil {
		t.Fatalf("generate questions: %v", err)
	}
	if requests.Load() != 1 || len(questions) != 2 {
		t.Fatalf("requests=%d questions=%d", requests.Load(), len(questions))
	}
	if questions[0].CorrectAnswerID != "b" || questions[1].Prompt == questions[0].Prompt {
		t.Fatalf("unexpected questions: %+v", questions)
	}
}

func TestGenerateRetriesInvalidModelResponse(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			writeCompletion(w, `{"questions":[{"prompt":"Повтор"}]}`)
			return
		}
		writeCompletion(w, validQuestionsJSON(1))
	}))
	t.Cleanup(server.Close)

	questions, err := newTestGenerator(server.URL).Generate(context.Background(), testInput(1))
	if err != nil {
		t.Fatalf("generate questions: %v", err)
	}
	if requests.Load() != 2 || len(questions) != 1 {
		t.Fatalf("requests=%d questions=%d", requests.Load(), len(questions))
	}
}

func TestGenerateDoesNotRetryProviderError(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, `{"error":{"message":"invalid api key"}}`, http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)

	_, err := newTestGenerator(server.URL).Generate(context.Background(), testInput(1))
	if !errors.Is(err, ports.ErrQuestionGeneration) {
		t.Fatalf("expected generation error, got %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("provider error was retried %d times", requests.Load())
	}
}

func TestGenerateFailsAfterRepeatedInvalidResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeCompletion(w, "")
	}))
	t.Cleanup(server.Close)

	_, err := newTestGenerator(server.URL).Generate(context.Background(), testInput(1))
	if !errors.Is(err, ports.ErrQuestionGeneration) {
		t.Fatalf("expected generation error, got %v", err)
	}
}

func TestGeneratePreservesCallerCancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Errorf("request was sent after cancellation")
	}))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := newTestGenerator(server.URL).Generate(ctx, testInput(1))
	if !errors.Is(err, context.Canceled) || errors.Is(err, ports.ErrQuestionGeneration) {
		t.Fatalf("expected caller cancellation, got %v", err)
	}
}

func newTestGenerator(baseURL string) *TheoryQuestionGenerator {
	return NewTheoryQuestionGenerator(config.YandexConfig{
		APIKey:   "test-key",
		FolderID: "folder",
		Model:    "yandexgpt",
		BaseURL:  baseURL,
		Timeout:  2 * time.Second,
	})
}

func testInput(count int) domain.TheoryGenerationInput {
	return domain.TheoryGenerationInput{
		Topics:  []string{"React"},
		Vacancy: "TypeScript",
		Level:   domain.LevelMiddle,
		Count:   count,
	}
}

func writeCompletion(w http.ResponseWriter, content string) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"choices": []map[string]any{
			{"message": map[string]string{"content": content}},
		},
	})
}
