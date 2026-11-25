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
	questions ports.TheoryQuestionGenerator
	codeTasks ports.CodeTaskGenerator
	now       func() time.Time
}

func NewCreateQuestionnaire(
	questions ports.TheoryQuestionGenerator,
	codeTasks ports.CodeTaskGenerator,
) *CreateQuestionnaire {
	return &CreateQuestionnaire{
		questions: questions,
		codeTasks: codeTasks,
		now:       time.Now,
	}
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

	questions, err := uc.generateQuestions(ctx, input, topics)
	if err != nil {
		return domain.Questionnaire{}, err
	}

	codeTasks, err := uc.generateCodeTasks(ctx, input, topics)
	if err != nil {
		return domain.Questionnaire{}, err
	}

	assignTaskIDs(questions, codeTasks)

	hints := input.Config.Hints
	if !hints.Enabled {
		hints.PerTask = 0
	}

	return domain.Questionnaire{
		ID:              fmt.Sprintf("questionnaire-%d", uc.now().UTC().UnixNano()),
		Topic:           topics[0],
		Topics:          topics,
		Level:           input.Level.DisplayName(),
		ResponseFormat:  input.ResponseFormat,
		AttemptsPerTask: input.Config.AttemptsPerTask,
		Hints:           hints,
		Questions:       questions,
		CodeTasks:       codeTasks,
	}, nil
}

func (uc *CreateQuestionnaire) generateQuestions(
	ctx context.Context,
	input domain.CreateQuestionnaireInput,
	topics []string,
) ([]domain.Question, error) {
	if !domain.TrainingRequested(input.Config.Theory) {
		return []domain.Question{}, nil
	}

	questions, err := uc.questions.Generate(ctx, domain.TheoryGenerationInput{
		Topics:  topics,
		Vacancy: strings.TrimSpace(input.Vacancy),
		Level:   input.Level,
		Count:   input.Config.Theory.Count,
	})
	if err != nil {
		return nil, err
	}
	if len(questions) != input.Config.Theory.Count {
		return nil, fmt.Errorf(
			"%w: expected %d questions, got %d",
			ports.ErrQuestionGeneration,
			input.Config.Theory.Count,
			len(questions),
		)
	}

	return questions, nil
}

func (uc *CreateQuestionnaire) generateCodeTasks(
	ctx context.Context,
	input domain.CreateQuestionnaireInput,
	topics []string,
) ([]domain.CodeTask, error) {
	if !domain.TrainingRequested(input.Config.Code) {
		return []domain.CodeTask{}, nil
	}

	hintsPerTask := 0
	if input.Config.Hints.Enabled {
		hintsPerTask = input.Config.Hints.PerTask
	}

	codeTasks, err := uc.codeTasks.Generate(ctx, domain.CodeGenerationInput{
		Topics:       topics,
		Vacancy:      strings.TrimSpace(input.Vacancy),
		Level:        input.Level,
		Count:        input.Config.Code.Count,
		HintsPerTask: hintsPerTask,
	})
	if err != nil {
		return nil, err
	}
	if len(codeTasks) != input.Config.Code.Count {
		return nil, fmt.Errorf(
			"%w: expected %d code tasks, got %d",
			ports.ErrQuestionGeneration,
			input.Config.Code.Count,
			len(codeTasks),
		)
	}

	return codeTasks, nil
}

func assignTaskIDs(questions []domain.Question, codeTasks []domain.CodeTask) {
	nextID := 1
	for index := range questions {
		questions[index].ID = nextID
		nextID++
	}
	for index := range codeTasks {
		codeTasks[index].ID = nextID
		nextID++
	}
}
