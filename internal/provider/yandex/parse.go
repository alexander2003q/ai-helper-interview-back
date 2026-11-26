package yandex

import (
	"encoding/json"
	"fmt"
	"strings"

	"ai-helper-interview/backend/internal/domain"
)

const theoryQuestionType = "Теоретический вопрос"

var answerOrder = []string{"a", "b", "c", "d"}

type generatedQuestionnaire struct {
	Questions []generatedQuestion `json:"questions"`
}

type generatedQuestion struct {
	Prompt          string            `json:"prompt"`
	Answers         []generatedAnswer `json:"answers"`
	CorrectAnswerID string            `json:"correctAnswerId"`
	Explanation     string            `json:"explanation"`
}

type generatedAnswer struct {
	ID          string `json:"id"`
	Text        string `json:"text"`
	Explanation string `json:"explanation"`
}

func parseQuestions(raw string, count int) ([]domain.Question, error) {
	decoder := json.NewDecoder(strings.NewReader(unwrapJSON(raw)))
	decoder.DisallowUnknownFields()

	var payload generatedQuestionnaire
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("model response is not valid question JSON: %w", err)
	}
	if decoder.More() {
		return nil, fmt.Errorf("model response contains extra JSON")
	}
	if len(payload.Questions) != count {
		return nil, fmt.Errorf("expected %d questions, got %d", count, len(payload.Questions))
	}

	questions := make([]domain.Question, 0, count)
	seenPrompts := make(map[string]struct{}, count)
	for index, question := range payload.Questions {
		parsed, err := parseQuestion(question)
		if err != nil {
			return nil, fmt.Errorf("question %d: %w", index+1, err)
		}

		promptKey := strings.ToLower(parsed.Prompt)
		if _, exists := seenPrompts[promptKey]; exists {
			return nil, fmt.Errorf("question %d repeats an earlier prompt", index+1)
		}
		seenPrompts[promptKey] = struct{}{}
		questions = append(questions, parsed)
	}

	return questions, nil
}

func parseQuestion(question generatedQuestion) (domain.Question, error) {
	prompt := strings.TrimSpace(question.Prompt)
	explanation := strings.TrimSpace(question.Explanation)
	correctAnswerID := strings.ToLower(strings.TrimSpace(question.CorrectAnswerID))
	if prompt == "" {
		return domain.Question{}, fmt.Errorf("prompt is empty")
	}
	if explanation == "" {
		return domain.Question{}, fmt.Errorf("explanation is empty")
	}
	if !isAnswerID(correctAnswerID) {
		return domain.Question{}, fmt.Errorf("correct answer is invalid")
	}
	if len(question.Answers) != len(answerOrder) {
		return domain.Question{}, fmt.Errorf("expected 4 answers, got %d", len(question.Answers))
	}

	answers := make([]domain.Answer, len(answerOrder))
	seen := make(map[string]struct{}, len(answerOrder))
	for _, answer := range question.Answers {
		id := strings.ToLower(strings.TrimSpace(answer.ID))
		if !isAnswerID(id) {
			return domain.Question{}, fmt.Errorf("answer id is invalid")
		}
		if _, exists := seen[id]; exists {
			return domain.Question{}, fmt.Errorf("answer id %s is duplicated", id)
		}
		seen[id] = struct{}{}

		text := strings.TrimSpace(answer.Text)
		answerExplanation := strings.TrimSpace(answer.Explanation)
		if text == "" {
			return domain.Question{}, fmt.Errorf("answer %s text is empty", id)
		}
		if id != correctAnswerID && answerExplanation == "" {
			return domain.Question{}, fmt.Errorf("answer %s explanation is empty", id)
		}
		if id == correctAnswerID {
			answerExplanation = ""
		}

		answers[answerIndex(id)] = domain.Answer{
			ID:          id,
			Text:        text,
			Explanation: answerExplanation,
		}
	}

	return domain.Question{
		Type:            theoryQuestionType,
		Prompt:          prompt,
		Answers:         answers,
		CorrectAnswerID: correctAnswerID,
		Explanation:     explanation,
	}, nil
}

func unwrapJSON(raw string) string {
	text := strings.TrimSpace(raw)
	if !strings.HasPrefix(text, "```") {
		return text
	}

	text = strings.TrimPrefix(text, "```")
	if newline := strings.IndexByte(text, '\n'); newline >= 0 {
		text = text[newline+1:]
	}
	text = strings.TrimSpace(text)
	text = strings.TrimSuffix(text, "```")
	return strings.TrimSpace(text)
}

func isAnswerID(id string) bool {
	return answerIndex(id) >= 0
}

func answerIndex(id string) int {
	for index, answerID := range answerOrder {
		if id == answerID {
			return index
		}
	}

	return -1
}
