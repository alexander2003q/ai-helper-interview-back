package httpapi

import (
	"encoding/json"

	"ai-helper-interview/backend/internal/domain"
)

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
			Hints: domain.HintsConfig{
				Enabled: r.TrainingConfig.Hints.Enabled,
				PerTask: r.TrainingConfig.Hints.PerTask,
			},
			AttemptsPerTask: r.TrainingConfig.AttemptsPerTask,
		},
	}
}

type answerResponse struct {
	ID          string `json:"id"`
	Text        string `json:"text"`
	Explanation string `json:"explanation,omitempty"`
}

type questionResponse struct {
	Kind            string           `json:"kind"`
	ID              int              `json:"id"`
	Type            string           `json:"type"`
	Prompt          string           `json:"prompt"`
	Answers         []answerResponse `json:"answers"`
	CorrectAnswerID string           `json:"correctAnswerId"`
	Explanation     string           `json:"explanation"`
}

type codeExampleResponse struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

type codeTestResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type codeSolutionResponse struct {
	Code        string `json:"code"`
	Explanation string `json:"explanation"`
}

type codeTaskResponse struct {
	Kind        string                `json:"kind"`
	ID          int                   `json:"id"`
	Type        string                `json:"type"`
	Title       string                `json:"title"`
	Description string                `json:"description"`
	Language    string                `json:"language"`
	FileName    string                `json:"fileName"`
	StarterCode string                `json:"starterCode"`
	Examples    []codeExampleResponse `json:"examples"`
	Tests       []codeTestResponse    `json:"tests"`
	Hints       []string              `json:"hints"`
	Solution    codeSolutionResponse  `json:"solution"`
}

type taskResponse struct {
	question *questionResponse
	codeTask *codeTaskResponse
}

func (task taskResponse) MarshalJSON() ([]byte, error) {
	if task.codeTask != nil {
		return json.Marshal(task.codeTask)
	}
	return json.Marshal(task.question)
}

type questionnaireResponse struct {
	ID              string                `json:"id"`
	Topic           string                `json:"topic"`
	Topics          []string              `json:"topics"`
	Level           string                `json:"level"`
	ResponseFormat  domain.ResponseFormat `json:"responseFormat"`
	Total           int                   `json:"total"`
	AttemptsPerTask int                   `json:"attemptsPerTask"`
	Hints           hintsConfigRequest    `json:"hints"`
	Tasks           []taskResponse        `json:"tasks"`
}

func toQuestionnaireResponse(questionnaire domain.Questionnaire) questionnaireResponse {
	tasks := make([]taskResponse, 0, len(questionnaire.Questions)+len(questionnaire.CodeTasks))

	for _, question := range questionnaire.Questions {
		answers := make([]answerResponse, 0, len(question.Answers))
		for _, answer := range question.Answers {
			answers = append(answers, answerResponse{
				ID:          answer.ID,
				Text:        answer.Text,
				Explanation: answer.Explanation,
			})
		}

		tasks = append(tasks, taskResponse{
			question: &questionResponse{
				Kind:            "question",
				ID:              question.ID,
				Type:            question.Type,
				Prompt:          question.Prompt,
				Answers:         answers,
				CorrectAnswerID: question.CorrectAnswerID,
				Explanation:     question.Explanation,
			},
		})
	}

	for _, task := range questionnaire.CodeTasks {
		examples := make([]codeExampleResponse, 0, len(task.Examples))
		for _, example := range task.Examples {
			examples = append(examples, codeExampleResponse{
				Input:  example.Input,
				Output: example.Output,
			})
		}

		tests := make([]codeTestResponse, 0, len(task.Tests))
		for _, test := range task.Tests {
			tests = append(tests, codeTestResponse{
				ID:    test.ID,
				Title: test.Title,
			})
		}

		hints := task.Hints
		if hints == nil {
			hints = []string{}
		}

		tasks = append(tasks, taskResponse{
			codeTask: &codeTaskResponse{
				Kind:        "code",
				ID:          task.ID,
				Type:        task.Type,
				Title:       task.Title,
				Description: task.Description,
				Language:    task.Language,
				FileName:    task.FileName,
				StarterCode: task.StarterCode,
				Examples:    examples,
				Tests:       tests,
				Hints:       hints,
				Solution: codeSolutionResponse{
					Code:        task.Solution.Code,
					Explanation: task.Solution.Explanation,
				},
			},
		})
	}

	return questionnaireResponse{
		ID:              questionnaire.ID,
		Topic:           questionnaire.Topic,
		Topics:          questionnaire.Topics,
		Level:           questionnaire.Level,
		ResponseFormat:  questionnaire.ResponseFormat,
		Total:           len(tasks),
		AttemptsPerTask: questionnaire.AttemptsPerTask,
		Hints: hintsConfigRequest{
			Enabled: questionnaire.Hints.Enabled,
			PerTask: questionnaire.Hints.PerTask,
		},
		Tasks: tasks,
	}
}
