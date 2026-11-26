package yandex

import (
	"fmt"
	"strings"

	"ai-helper-interview/backend/internal/domain"
)

const theorySystemPrompt = `Ты составляешь теоретические вопросы технического интервью на русском языке.
У каждого вопроса ровно четыре варианта с id a, b, c и d и один правильный ответ.
explanation объясняет, почему верный ответ верен.
У каждого неверного варианта explanation объясняет, почему он неверен.
У верного варианта explanation оставь пустым.
Вопросы разные, соответствуют темам, описанию вакансии и уровню.
Не пиши задачи с кодом и не добавляй текст вне JSON.`

func theoryMessages(input domain.TheoryGenerationInput) []chatMessage {
	return []chatMessage{
		{Role: "system", Content: theorySystemPrompt},
		{Role: "user", Content: userPrompt(input)},
	}
}

func userPrompt(input domain.TheoryGenerationInput) string {
	vacancy := strings.TrimSpace(input.Vacancy)
	if vacancy == "" {
		vacancy = "не указано"
	}

	return fmt.Sprintf(
		"Составь %d теоретических вопросов.\nТемы: %s.\nУровень: %s.\nОписание вакансии: %s.\nЯзык вопросов и ответов: русский.",
		input.Count,
		strings.Join(input.Topics, ", "),
		input.Level.DisplayName(),
		vacancy,
	)
}
