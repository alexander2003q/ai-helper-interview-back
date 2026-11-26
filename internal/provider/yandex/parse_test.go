package yandex

import (
	"strings"
	"testing"
)

func TestParseQuestionsNormalizesAnswers(t *testing.T) {
	t.Parallel()

	questions, err := parseQuestions(`{
		"questions": [{
			"prompt": "  Зачем нужен key?  ",
			"correctAnswerId": "C",
			"explanation": "  Key сопоставляет элементы.  ",
			"answers": [
				{"id": "c", "text": " Для сопоставления ", "explanation": "лишнее"},
				{"id": "a", "text": "Для стилей", "explanation": " Key не задаёт стили. "},
				{"id": "d", "text": "Для props", "explanation": "Key не попадает в props."},
				{"id": "b", "text": "Для сортировки", "explanation": "Key не сортирует данные."}
			]
		}]
	}`, 1)
	if err != nil {
		t.Fatalf("parse questions: %v", err)
	}

	question := questions[0]
	if question.Type != theoryQuestionType || question.ID != 0 {
		t.Fatalf("unexpected question metadata: %+v", question)
	}
	if question.Prompt != "Зачем нужен key?" || question.Explanation != "Key сопоставляет элементы." {
		t.Fatalf("unexpected prompt or explanation: %+v", question)
	}
	if question.CorrectAnswerID != "c" || question.Answers[2].Explanation != "" {
		t.Fatalf("correct answer was not normalized: %+v", question)
	}

	for index, id := range answerOrder {
		if question.Answers[index].ID != id {
			t.Fatalf("answer %d has id %s", index, question.Answers[index].ID)
		}
	}
	if question.Answers[0].Explanation != "Key не задаёт стили." {
		t.Fatalf("wrong answer explanation was lost: %+v", question.Answers[0])
	}
}

func TestParseQuestionsAcceptsFencedJSON(t *testing.T) {
	t.Parallel()

	questions, err := parseQuestions("```json\n"+validQuestionsJSON(1)+"\n```", 1)
	if err != nil {
		t.Fatalf("parse fenced questions: %v", err)
	}
	if len(questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(questions))
	}
}

func TestParseQuestionsRejectsInvalidQuestion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		count   int
		message string
	}{
		{
			name:    "wrong count",
			raw:     validQuestionsJSON(1),
			count:   2,
			message: "expected 2 questions",
		},
		{
			name:    "duplicate prompt",
			raw:     `{"questions":[` + questionJSON("Один вопрос", "b") + `,` + questionJSON("Один вопрос", "a") + `]}`,
			count:   2,
			message: "repeats an earlier prompt",
		},
		{
			name:    "missing wrong explanation",
			raw:     `{"questions":[{"prompt":"Вопрос","correctAnswerId":"a","explanation":"Потому что","answers":[{"id":"a","text":"Верно","explanation":""},{"id":"b","text":"Неверно","explanation":"  "},{"id":"c","text":"Тоже нет","explanation":"Причина"},{"id":"d","text":"И это нет","explanation":"Причина"}]}]}`,
			count:   1,
			message: "answer b explanation is empty",
		},
		{
			name:    "unknown field",
			raw:     `{"questions":[],"extra":true}`,
			count:   0,
			message: "unknown field",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseQuestions(test.raw, test.count)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("expected %q, got %v", test.message, err)
			}
		})
	}
}

func questionJSON(prompt, correctID string) string {
	return `{
		"prompt": "` + prompt + `",
		"correctAnswerId": "` + correctID + `",
		"explanation": "Пояснение",
		"answers": [
			{"id": "a", "text": "Вариант A", "explanation": "Неверно A"},
			{"id": "b", "text": "Вариант B", "explanation": "Неверно B"},
			{"id": "c", "text": "Вариант C", "explanation": "Неверно C"},
			{"id": "d", "text": "Вариант D", "explanation": "Неверно D"}
		]
	}`
}

func validQuestionsJSON(count int) string {
	questions := make([]string, 0, count)
	for index := 0; index < count; index++ {
		questions = append(questions, questionJSON("Вопрос "+string(rune('A'+index)), "b"))
	}

	return `{"questions":[` + strings.Join(questions, ",") + `]}`
}
