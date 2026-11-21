package ports

import (
	"context"
	"errors"

	"ai-helper-interview/backend/internal/domain"
)

var ErrQuestionGeneration = errors.New("question generation failed")

type TheoryQuestionGenerator interface {
	Generate(ctx context.Context, input domain.TheoryGenerationInput) ([]domain.Question, error)
}
