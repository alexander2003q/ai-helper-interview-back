package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ai-helper-interview/backend/internal/domain"
	"ai-helper-interview/backend/internal/ports"
)

type CreateQuestionnaire struct {
	generator ports.TheoryQuestionGenerator
	now       func() time.Time
}

func NewCreateQuestionnaire(generator ports.TheoryQuestionGenerator) *CreateQuestionnaire {
	return &CreateQuestionnaire{generator: generator, now: time.Now}
}

func (uc *CreateQuestionnaire) Execute(
	ctx context.Context,
	input domain.CreateQuestionnaireInput,
) (domain.Questionnaire, error) {
	if err := input.Validate(); err != nil {
		return domain.Questionnaire{}, err
	}

	topics := make([]string, 0, len(input.Topics))
	for _, topic := range input.Topics {
		topics = append(topics, strings.TrimSpace(topic))
	}

	questions, err := uc.generator.Generate(ctx, domain.TheoryGenerationInput{
		Topics:  topics,
		Vacancy: strings.TrimSpace(input.Vacancy),
		Level:   input.Level,
		Count:   input.Config.Theory.Count,
	})
	if err != nil {
		return domain.Questionnaire{}, err
	}
	if len(questions) != input.Config.Theory.Count {
		return domain.Questionnaire{}, fmt.Errorf(
			"%w: expected %d questions, got %d",
			ports.ErrQuestionGeneration,
			input.Config.Theory.Count,
			len(questions),
		)
	}

	return domain.Questionnaire{
		ID:             fmt.Sprintf("questionnaire-%d", uc.now().UTC().UnixNano()),
		Topic:          topics[0],
		Topics:         topics,
		Level:          input.Level.DisplayName(),
		ResponseFormat: input.ResponseFormat,
		Questions:      questions,
	}, nil
}
