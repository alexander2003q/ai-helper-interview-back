package domain

import (
	"fmt"
	"strings"
)

type Level string

const (
	LevelJunior Level = "junior"
	LevelMiddle Level = "middle"
	LevelSenior Level = "senior"
)

func (l Level) IsValid() bool {
	switch l {
	case LevelJunior, LevelMiddle, LevelSenior:
		return true
	default:
		return false
	}
}

func (l Level) DisplayName() string {
	switch l {
	case LevelJunior:
		return "Junior"
	case LevelMiddle:
		return "Middle"
	case LevelSenior:
		return "Senior"
	default:
		return string(l)
	}
}

type ResponseFormat string

const (
	ResponseFormatText  ResponseFormat = "text"
	ResponseFormatVoice ResponseFormat = "voice"
)

func (f ResponseFormat) IsValid() bool {
	switch f {
	case ResponseFormatText, ResponseFormatVoice:
		return true
	default:
		return false
	}
}

type TrainingPart struct {
	Enabled bool
	Count   int
}

type HintsConfig struct {
	Enabled bool
	PerTask int
}

type TrainingConfig struct {
	Theory          TrainingPart
	Code            TrainingPart
	Experience      TrainingPart
	Hints           HintsConfig
	AttemptsPerTask int
}

type CreateQuestionnaireInput struct {
	Topics         []string
	Vacancy        string
	Level          Level
	ResponseFormat ResponseFormat
	Config         TrainingConfig
}

type TheoryGenerationInput struct {
	Topics  []string
	Vacancy string
	Level   Level
	Count   int
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func (i CreateQuestionnaireInput) Validate() error {
	if len(i.Topics) == 0 || len(i.Topics) > 3 {
		return &ValidationError{
			Field:   "topics",
			Message: "must contain from 1 to 3 topics",
		}
	}

	for _, topic := range i.Topics {
		if strings.TrimSpace(topic) == "" {
			return &ValidationError{
				Field:   "topics",
				Message: "must not contain empty values",
			}
		}
	}

	if !i.Level.IsValid() {
		return &ValidationError{
			Field:   "level",
			Message: "must be one of junior, middle or senior",
		}
	}

	if !i.ResponseFormat.IsValid() {
		return &ValidationError{
			Field:   "responseFormat",
			Message: "must be one of text or voice",
		}
	}

	if !i.Config.Theory.Enabled && !i.Config.Code.Enabled && !i.Config.Experience.Enabled {
		return &ValidationError{
			Field:   "trainingConfig",
			Message: "at least one training section must be enabled",
		}
	}

	if !i.Config.Theory.Enabled {
		return &ValidationError{
			Field:   "trainingConfig.theory.enabled",
			Message: "must be enabled while only theory questions are supported",
		}
	}

	parts := []struct {
		name string
		part TrainingPart
	}{
		{name: "trainingConfig.theory.count", part: i.Config.Theory},
		{name: "trainingConfig.code.count", part: i.Config.Code},
		{name: "trainingConfig.experience.count", part: i.Config.Experience},
	}

	for _, item := range parts {
		if item.part.Enabled && item.part.Count <= 0 {
			return &ValidationError{
				Field:   item.name,
				Message: "must be greater than zero when the section is enabled",
			}
		}
	}

	if i.Config.Theory.Count > 20 {
		return &ValidationError{
			Field:   "trainingConfig.theory.count",
			Message: "must not be greater than 20",
		}
	}

	if i.Config.Hints.Enabled && (i.Config.Hints.PerTask < 1 || i.Config.Hints.PerTask > 3) {
		return &ValidationError{
			Field:   "trainingConfig.hints.perTask",
			Message: "must be between 1 and 3",
		}
	}

	if i.Config.AttemptsPerTask < 1 || i.Config.AttemptsPerTask > 5 {
		return &ValidationError{
			Field:   "trainingConfig.attemptsPerTask",
			Message: "must be between 1 and 5",
		}
	}

	return nil
}

type Answer struct {
	ID   string
	Text string
}

type Question struct {
	ID              int
	Type            string
	Prompt          string
	Answers         []Answer
	CorrectAnswerID string
	Explanation     string
}

type Questionnaire struct {
	ID             string
	Topic          string
	Topics         []string
	Level          string
	ResponseFormat ResponseFormat
	Questions      []Question
}
