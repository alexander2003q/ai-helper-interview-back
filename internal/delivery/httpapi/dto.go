package httpapi

import "ai-helper-interview/backend/internal/domain"

type trainingPartRequest struct {
	Enabled bool `json:"enabled"`
	Count   int  `json:"count"`
}

type hintsConfigRequest struct {
	Enabled bool `json:"enabled"`
	PerTask int  `json:"perTask"`
}

type trainingConfigRequest struct {
	Theory          trainingPartRequest `json:"theory"`
	Code            trainingPartRequest `json:"code"`
	Experience      trainingPartRequest `json:"experience"`
	Hints           hintsConfigRequest  `json:"hints"`
	AttemptsPerTask int                 `json:"attemptsPerTask"`
}

type createQuestionnaireRequest struct {
	Topics         []string              `json:"topics"`
	Vacancy        string                `json:"vacancy"`
	Level          domain.Level          `json:"level"`
	ResponseFormat domain.ResponseFormat `json:"responseFormat"`
	TrainingConfig trainingConfigRequest `json:"trainingConfig"`
}

func (r createQuestionnaireRequest) toDomain() domain.CreateQuestionnaireInput {
	return domain.CreateQuestionnaireInput{
		Topics:         r.Topics,
		Vacancy:        r.Vacancy,
		Level:          r.Level,
		ResponseFormat: r.ResponseFormat,
		Config: domain.TrainingConfig{
			Theory: domain.TrainingPart{
				Enabled: r.TrainingConfig.Theory.Enabled,
				Count:   r.TrainingConfig.Theory.Count,
			},
			Code: domain.TrainingPart{
				Enabled: r.TrainingConfig.Code.Enabled,
				Count:   r.TrainingConfig.Code.Count,
			},
			Experience: domain.TrainingPart{
				Enabled: r.TrainingConfig.Experience.Enabled,
				Count:   r.TrainingConfig.Experience.Count,
			},
			Hints: domain.HintsConfig{
				Enabled: r.TrainingConfig.Hints.Enabled,
				PerTask: r.TrainingConfig.Hints.PerTask,
			},
			AttemptsPerTask: r.TrainingConfig.AttemptsPerTask,
		},
	}
}

type answerResponse struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type questionResponse struct {
	ID              int              `json:"id"`
	Type            string           `json:"type"`
	Prompt          string           `json:"prompt"`
	Answers         []answerResponse `json:"answers"`
	CorrectAnswerID string           `json:"correctAnswerId"`
	Explanation     string           `json:"explanation"`
}

type questionnaireResponse struct {
	ID             string                `json:"id"`
	Topic          string                `json:"topic"`
	Topics         []string              `json:"topics"`
	Level          string                `json:"level"`
	ResponseFormat domain.ResponseFormat `json:"responseFormat"`
	Total          int                   `json:"total"`
	Questions      []questionResponse    `json:"questions"`
}

func toQuestionnaireResponse(questionnaire domain.Questionnaire) questionnaireResponse {
	questions := make([]questionResponse, 0, len(questionnaire.Questions))

	for _, question := range questionnaire.Questions {
		answers := make([]answerResponse, 0, len(question.Answers))
		for _, answer := range question.Answers {
			answers = append(answers, answerResponse{
				ID:   answer.ID,
				Text: answer.Text,
			})
		}

		questions = append(questions, questionResponse{
			ID:              question.ID,
			Type:            question.Type,
			Prompt:          question.Prompt,
			Answers:         answers,
			CorrectAnswerID: question.CorrectAnswerID,
			Explanation:     question.Explanation,
		})
	}

	return questionnaireResponse{
		ID:             questionnaire.ID,
		Topic:          questionnaire.Topic,
		Topics:         questionnaire.Topics,
		Level:          questionnaire.Level,
		ResponseFormat: questionnaire.ResponseFormat,
		Total:          len(questions),
		Questions:      questions,
	}
}
