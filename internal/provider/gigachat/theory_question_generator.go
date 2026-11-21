package gigachat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"ai-helper-interview/backend/internal/domain"
	"ai-helper-interview/backend/internal/ports"
)

func (c *Client) Generate(
	ctx context.Context,
	input domain.TheoryGenerationInput,
) ([]domain.Question, error) {
	requestBody, err := json.Marshal(newChatRequest(c.model, input))
	if err != nil {
		return nil, fmt.Errorf("%w: encode request: %v", ports.ErrQuestionGeneration, err)
	}

	var generationErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		responseBody, err := c.generate(ctx, requestBody)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ports.ErrQuestionGeneration, err)
		}

		questions, err := parseChatResponse(responseBody, input.Count)
		if err == nil {
			return questions, nil
		}
		generationErr = err

		if attempt < c.maxRetries {
			if err := waitForRetry(ctx, attempt); err != nil {
				return nil, err
			}
		}
	}

	return nil, fmt.Errorf("%w: invalid structured output after retries: %v", ports.ErrQuestionGeneration, generationErr)
}

func (c *Client) generate(ctx context.Context, body []byte) ([]byte, error) {
	forceTokenRefresh := false
	unauthorizedRetried := false

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		accessToken, err := c.getAccessToken(ctx, forceTokenRefresh)
		if err != nil {
			return nil, err
		}
		forceTokenRefresh = false

		request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.chatURL, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("create GigaChat generation request: %w", err)
		}
		request.Header.Set("Accept", "application/json")
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer "+accessToken)

		responseBody, err := c.send(request)
		if err == nil {
			return responseBody, nil
		}
		if isUnauthorized(err) && !unauthorizedRetried {
			unauthorizedRetried = true
			forceTokenRefresh = true
			attempt--
			continue
		}
		if !isRetryable(err) || attempt == c.maxRetries {
			return nil, err
		}
		if err := waitForRetry(ctx, attempt); err != nil {
			return nil, err
		}
	}

	return nil, errors.New("GigaChat generation retries exhausted")
}

type chatRequest struct {
	Model          string         `json:"model"`
	Messages       []chatMessage  `json:"messages"`
	Temperature    float64        `json:"temperature"`
	MaxTokens      int            `json:"max_tokens"`
	ResponseFormat responseFormat `json:"response_format"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type   string         `json:"type"`
	Schema map[string]any `json:"schema"`
	Strict bool           `json:"strict"`
}

func newChatRequest(model string, input domain.TheoryGenerationInput) chatRequest {
	userPrompt := fmt.Sprintf(
		"Составь ровно %d теоретических вопросов для технического собеседования.\n"+
			"Темы: %s.\nУровень кандидата: %s.\nОписание вакансии и требования: %s.\n"+
			"Проверяй понимание принципов и практическое применение. Не добавляй задачи на написание кода, вопросы об опыте, подсказки и повторяющиеся вопросы.",
		input.Count,
		strings.Join(input.Topics, ", "),
		input.Level.DisplayName(),
		fallback(input.Vacancy, "не указаны"),
	)

	return chatRequest{
		Model: model,
		Messages: []chatMessage{
			{
				Role:    "system",
				Content: "Ты составляешь качественные технические собеседования. Пиши по-русски. У каждого вопроса должен быть ровно один бесспорно правильный вариант ответа, ровно четыре непустых уникальных варианта и короткое полезное объяснение.",
			},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
		MaxTokens:   min(input.Count*500+500, 8192),
		ResponseFormat: responseFormat{
			Type:   "json_schema",
			Schema: questionnaireSchema(input.Count),
			Strict: true,
		},
	}
}

func questionnaireSchema(count int) map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"questions"},
		"properties": map[string]any{
			"questions": map[string]any{
				"type":     "array",
				"minItems": count,
				"maxItems": count,
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []string{"prompt", "answers", "correctAnswerIndex", "explanation"},
					"properties": map[string]any{
						"prompt": map[string]any{"type": "string", "minLength": 1},
						"answers": map[string]any{
							"type":        "array",
							"minItems":    4,
							"maxItems":    4,
							"uniqueItems": true,
							"items":       map[string]any{"type": "string", "minLength": 1},
						},
						"correctAnswerIndex": map[string]any{
							"type":    "integer",
							"minimum": 0,
							"maximum": 3,
						},
						"explanation": map[string]any{"type": "string", "minLength": 1},
					},
				},
			},
		},
	}
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

type generatedQuestionnaire struct {
	Questions []struct {
		Prompt             string   `json:"prompt"`
		Answers            []string `json:"answers"`
		CorrectAnswerIndex int      `json:"correctAnswerIndex"`
		Explanation        string   `json:"explanation"`
	} `json:"questions"`
}

func parseChatResponse(body []byte, expectedCount int) ([]domain.Question, error) {
	var response chatResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode GigaChat envelope: %w", err)
	}
	if len(response.Choices) == 0 || strings.TrimSpace(response.Choices[0].Message.Content) == "" {
		return nil, errors.New("GigaChat returned no choices")
	}

	var generated generatedQuestionnaire
	decoder := json.NewDecoder(strings.NewReader(response.Choices[0].Message.Content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&generated); err != nil {
		return nil, fmt.Errorf("decode structured GigaChat output: %w", err)
	}
	if len(generated.Questions) != expectedCount {
		return nil, fmt.Errorf("expected %d generated questions, got %d", expectedCount, len(generated.Questions))
	}

	answerIDs := []string{"a", "b", "c", "d"}
	questions := make([]domain.Question, 0, expectedCount)
	for index, generatedQuestion := range generated.Questions {
		prompt := strings.TrimSpace(generatedQuestion.Prompt)
		explanation := strings.TrimSpace(generatedQuestion.Explanation)
		if prompt == "" || explanation == "" || len(generatedQuestion.Answers) != 4 {
			return nil, fmt.Errorf("question %d has incomplete fields", index+1)
		}
		if generatedQuestion.CorrectAnswerIndex < 0 || generatedQuestion.CorrectAnswerIndex >= len(answerIDs) {
			return nil, fmt.Errorf("question %d has invalid correct answer index", index+1)
		}

		answers := make([]domain.Answer, 0, 4)
		seenAnswers := make(map[string]struct{}, 4)
		for answerIndex, answerText := range generatedQuestion.Answers {
			answerText = strings.TrimSpace(answerText)
			answerKey := strings.ToLower(answerText)
			if answerText == "" {
				return nil, fmt.Errorf("question %d has an empty answer", index+1)
			}
			if _, exists := seenAnswers[answerKey]; exists {
				return nil, fmt.Errorf("question %d has duplicate answers", index+1)
			}
			seenAnswers[answerKey] = struct{}{}
			answers = append(answers, domain.Answer{ID: answerIDs[answerIndex], Text: answerText})
		}

		questions = append(questions, domain.Question{
			ID:              index + 1,
			Type:            "Теоретический вопрос",
			Prompt:          prompt,
			Answers:         answers,
			CorrectAnswerID: answerIDs[generatedQuestion.CorrectAnswerIndex],
			Explanation:     explanation,
		})
	}

	return questions, nil
}

func fallback(value, fallbackValue string) string {
	if strings.TrimSpace(value) == "" {
		return fallbackValue
	}
	return value
}
