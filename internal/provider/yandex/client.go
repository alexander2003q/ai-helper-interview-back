package yandex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"ai-helper-interview/backend/internal/config"
)

const (
	maxResponseBytes = 1 << 20
	temperature      = 0.6
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	folderID   string
	modelURI   string
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string         `json:"model"`
	Messages       []chatMessage  `json:"messages"`
	Temperature    float64        `json:"temperature"`
	MaxTokens      int            `json:"max_tokens"`
	Stream         bool           `json:"stream"`
	ResponseFormat responseFormat `json:"response_format"`
}

type responseFormat struct {
	Type       string           `json:"type"`
	JSONSchema jsonSchemaFormat `json:"json_schema"`
}

type jsonSchemaFormat struct {
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema"`
}

type chatCompletion struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type apiErrorBody struct {
	Message string `json:"message"`
	Error   struct {
		Message string `json:"message"`
	} `json:"error"`
}

type retryableError struct {
	err error
}

func (e *retryableError) Error() string {
	return e.err.Error()
}

func (e *retryableError) Unwrap() error {
	return e.err
}

func isRetryable(err error) bool {
	var retryable *retryableError
	return errors.As(err, &retryable)
}

func NewClient(cfg config.YandexConfig) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		baseURL:  cfg.BaseURL,
		apiKey:   cfg.APIKey,
		folderID: cfg.FolderID,
		modelURI: cfg.ModelURI(),
	}
}

func (c *Client) Complete(
	ctx context.Context,
	messages []chatMessage,
	questionCount int,
	schema json.RawMessage,
) (string, error) {
	payload := chatRequest{
		Model:       c.modelURI,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   completionTokens(questionCount),
		ResponseFormat: responseFormat{
			Type: "json_schema",
			JSONSchema: jsonSchemaFormat{
				Name:   theorySchemaName,
				Schema: schema,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", err
	}

	request.Header.Set("Authorization", "Api-Key "+c.apiKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("x-folder-id", c.folderID)
	request.Header.Set("x-data-logging-enabled", "false")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
	if err != nil {
		return "", err
	}

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"yandex ai studio returned %d: %s",
			response.StatusCode,
			apiErrorMessage(responseBody),
		)
	}

	var completion chatCompletion
	if err := json.Unmarshal(responseBody, &completion); err != nil {
		return "", &retryableError{err: errors.New("yandex ai studio returned unexpected JSON")}
	}
	if len(completion.Choices) == 0 {
		return "", &retryableError{err: errors.New("yandex ai studio returned no choices")}
	}

	content := strings.TrimSpace(completion.Choices[0].Message.Content)
	if content == "" {
		return "", &retryableError{err: errors.New("yandex ai studio returned an empty completion")}
	}

	return content, nil
}

func completionTokens(questionCount int) int {
	tokens := 500 * questionCount
	if tokens < 1200 {
		tokens = 1200
	}
	if tokens > 8000 {
		tokens = 8000
	}

	return tokens
}

func apiErrorMessage(body []byte) string {
	var payload apiErrorBody
	if err := json.Unmarshal(body, &payload); err == nil {
		if message := strings.TrimSpace(payload.Error.Message); message != "" {
			return limitText(message, 300)
		}
		if message := strings.TrimSpace(payload.Message); message != "" {
			return limitText(message, 300)
		}
	}

	text := strings.TrimSpace(string(body))
	if text == "" {
		return "empty error response"
	}

	return limitText(text, 300)
}

func limitText(value string, limit int) string {
	if len(value) <= limit {
		return value
	}

	trimmed := value[:limit]
	for !utf8.ValidString(trimmed) {
		trimmed = trimmed[:len(trimmed)-1]
	}

	return trimmed
}
