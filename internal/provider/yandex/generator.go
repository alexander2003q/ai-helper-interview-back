// Package yandex generates multiple-choice theory questions through
// Yandex AI Studio Chat Completions.
package yandex

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ai-helper-interview/backend/internal/config"
	"ai-helper-interview/backend/internal/domain"
	"ai-helper-interview/backend/internal/ports"
)

const generationAttempts = 2

type TheoryQuestionGenerator struct {
	client  *Client
	timeout time.Duration
}

func NewTheoryQuestionGenerator(cfg config.YandexConfig) *TheoryQuestionGenerator {
	return &TheoryQuestionGenerator{
		client:  NewClient(cfg),
		timeout: cfg.Timeout,
	}
}

func (g *TheoryQuestionGenerator) Generate(
	ctx context.Context,
	input domain.TheoryGenerationInput,
) ([]domain.Question, error) {
	ctx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	messages := theoryMessages(input)
	schema := theoryQuestionsSchema(input.Count)
	var lastErr error

	for attempt := 1; attempt <= generationAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, generationError(err)
		}

		content, err := g.client.Complete(ctx, messages, input.Count, schema)
		if err != nil {
			if !isRetryable(err) {
				return nil, generationError(err)
			}
			lastErr = err
			continue
		}

		questions, err := parseQuestions(content, input.Count)
		if err == nil {
			return questions, nil
		}
		lastErr = err
	}

	return nil, generationError(lastErr)
}

func generationError(err error) error {
	if err == nil {
		err = errors.New("question generation failed")
	}
	if errors.Is(err, context.Canceled) {
		return err
	}

	return fmt.Errorf("%w: %v", ports.ErrQuestionGeneration, err)
}
