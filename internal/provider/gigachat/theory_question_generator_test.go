package gigachat_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"ai-helper-interview/backend/internal/domain"
	"ai-helper-interview/backend/internal/provider/gigachat"
)

func TestGenerateUsesOAuthStructuredOutputAndCachesToken(t *testing.T) {
	t.Parallel()

	var oauthCalls atomic.Int32
	var chatCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth":
			oauthCalls.Add(1)
			if authorization := r.Header.Get("Authorization"); authorization != "Basic test-key" {
				t.Errorf("unexpected OAuth authorization: %q", authorization)
			}
			if r.Header.Get("RqUID") == "" {
				t.Error("OAuth request has no RqUID")
			}
			if err := r.ParseForm(); err != nil {
				t.Errorf("parse OAuth form: %v", err)
			}
			if scope := r.Form.Get("scope"); scope != "GIGACHAT_API_PERS" {
				t.Errorf("unexpected scope: %q", scope)
			}
			writeJSON(t, w, map[string]any{
				"access_token": "test-access-token",
				"expires_at":   time.Now().Add(time.Hour).Unix(),
			})
		case "/chat":
			chatCall := chatCalls.Add(1)
			if authorization := r.Header.Get("Authorization"); authorization != "Bearer test-access-token" {
				t.Errorf("unexpected chat authorization: %q", authorization)
			}

			var requestBody struct {
				Model    string `json:"model"`
				Messages []struct {
					Content string `json:"content"`
				} `json:"messages"`
				ResponseFormat struct {
					Type   string `json:"type"`
					Strict bool   `json:"strict"`
				} `json:"response_format"`
			}
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				t.Errorf("decode chat request: %v", err)
			}
			if requestBody.Model != "GigaChat-2" {
				t.Errorf("unexpected model: %q", requestBody.Model)
			}
			if len(requestBody.Messages) != 2 || !strings.Contains(requestBody.Messages[1].Content, "React, TypeScript") {
				t.Errorf("prompt does not contain topics: %+v", requestBody.Messages)
			}
			if requestBody.ResponseFormat.Type != "json_schema" || !requestBody.ResponseFormat.Strict {
				t.Errorf("unexpected response format: %+v", requestBody.ResponseFormat)
			}

			structuredOutput := structuredQuestions()
			if chatCall == 1 {
				structuredOutput = strings.Replace(structuredOutput, "Запускает эффект", "", 1)
			}

			writeJSON(t, w, map[string]any{
				"choices": []any{map[string]any{
					"message": map[string]any{
						"role":    "assistant",
						"content": structuredOutput,
					},
					"finish_reason": "stop",
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := gigachat.NewClient(gigachat.Config{
		AuthorizationKey: "test-key",
		Model:            "GigaChat-2",
		Scope:            "GIGACHAT_API_PERS",
		OAuthURL:         server.URL + "/oauth",
		ChatURL:          server.URL + "/chat",
		HTTPClient:       server.Client(),
		MaxRetries:       1,
	})
	if err != nil {
		t.Fatalf("create GigaChat client: %v", err)
	}

	input := domain.TheoryGenerationInput{
		Topics:  []string{"React", "TypeScript"},
		Vacancy: "React, REST API",
		Level:   domain.LevelMiddle,
		Count:   2,
	}
	for call := 0; call < 2; call++ {
		questions, generateErr := client.Generate(context.Background(), input)
		if generateErr != nil {
			t.Fatalf("generate questions: %v", generateErr)
		}
		if len(questions) != 2 {
			t.Fatalf("expected 2 questions, got %d", len(questions))
		}
		if questions[0].CorrectAnswerID != "b" || questions[1].CorrectAnswerID != "c" {
			t.Fatalf("unexpected correct answer mapping: %+v", questions)
		}
	}

	if calls := oauthCalls.Load(); calls != 1 {
		t.Fatalf("expected one OAuth call, got %d", calls)
	}
	if calls := chatCalls.Load(); calls != 3 {
		t.Fatalf("expected three chat calls including one validation retry, got %d", calls)
	}
}

func TestNewClientRequiresAuthorizationKey(t *testing.T) {
	t.Parallel()

	_, err := gigachat.NewClient(gigachat.Config{
		Model:    "GigaChat-2",
		Scope:    "GIGACHAT_API_PERS",
		OAuthURL: "https://example.com/oauth",
		ChatURL:  "https://example.com/chat",
	})
	if err == nil {
		t.Fatal("expected missing authorization key error")
	}
}

func structuredQuestions() string {
	return `{
		"questions": [
			{
				"prompt": "Что делает useMemo?",
				"answers": ["Запускает эффект", "Кэширует вычисление", "Создаёт state", "Меняет props"],
				"correctAnswerIndex": 1,
				"explanation": "useMemo кэширует результат вычисления между рендерами."
			},
			{
				"prompt": "Для чего нужен key?",
				"answers": ["Для CSS", "Для props", "Для сопоставления элементов", "Для запросов"],
				"correctAnswerIndex": 2,
				"explanation": "key помогает React сопоставлять элементы между рендерами."
			}
		]
	}`
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Errorf("encode response: %v", err)
	}
}
