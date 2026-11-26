package domain_test

import (
	"errors"
	"testing"

	"ai-helper-interview/backend/internal/domain"
)

func TestCreateQuestionnaireInputValidate(t *testing.T) {
	t.Parallel()

	validInput := domain.CreateQuestionnaireInput{
		Topics:         []string{"React"},
		Level:          domain.LevelMiddle,
		ResponseFormat: domain.ResponseFormatText,
		Config: domain.TrainingConfig{
			Theory:          domain.TrainingPart{Enabled: true, Count: 8},
			Hints:           domain.HintsConfig{Enabled: true, PerTask: 2},
			AttemptsPerTask: 3,
		},
	}

	tests := []struct {
		name      string
		mutate    func(*domain.CreateQuestionnaireInput)
		wantField string
	}{
		{
			name: "topics are required",
			mutate: func(input *domain.CreateQuestionnaireInput) {
				input.Topics = nil
			},
			wantField: "topics",
		},
		{
			name: "level must be supported",
			mutate: func(input *domain.CreateQuestionnaireInput) {
				input.Level = "lead"
			},
			wantField: "level",
		},
		{
			name: "at least one section is required",
			mutate: func(input *domain.CreateQuestionnaireInput) {
				input.Config.Theory.Enabled = false
			},
			wantField: "trainingConfig",
		},
		{
			name: "code count is limited",
			mutate: func(input *domain.CreateQuestionnaireInput) {
				input.Config.Code = domain.TrainingPart{Enabled: true, Count: 11}
			},
			wantField: "trainingConfig.code.count",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validInput
			test.mutate(&input)

			err := input.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}

			var validationError *domain.ValidationError
			if !errors.As(err, &validationError) {
				t.Fatalf("expected ValidationError, got %T", err)
			}

			if validationError.Field != test.wantField {
				t.Fatalf("expected field %q, got %q", test.wantField, validationError.Field)
			}
		})
	}

	if err := validInput.Validate(); err != nil {
		t.Fatalf("valid input returned error: %v", err)
	}

	codeOnly := validInput
	codeOnly.Config.Theory.Enabled = false
	codeOnly.Config.Code = domain.TrainingPart{Enabled: true, Count: 1}
	if err := codeOnly.Validate(); err != nil {
		t.Fatalf("code-only input returned error: %v", err)
	}
}
