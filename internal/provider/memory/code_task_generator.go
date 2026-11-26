package memory

import (
	"context"

	"ai-helper-interview/backend/internal/domain"
)

type CodeTaskGenerator struct{}

func NewCodeTaskGenerator() *CodeTaskGenerator {
	return &CodeTaskGenerator{}
}

func (g *CodeTaskGenerator) Generate(
	ctx context.Context,
	input domain.CodeGenerationInput,
) ([]domain.CodeTask, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	templates := mockCodeTasks()
	result := make([]domain.CodeTask, 0, input.Count)
	for index := 0; index < input.Count; index++ {
		task := templates[index%len(templates)]
		task.Hints = takeHints(task.Hints, input.HintsPerTask)
		result = append(result, task)
	}

	return result, nil
}

func takeHints(hints []string, count int) []string {
	if count <= 0 {
		return []string{}
	}
	if count > len(hints) {
		count = len(hints)
	}

	selected := make([]string, count)
	copy(selected, hints[:count])
	return selected
}

func mockCodeTasks() []domain.CodeTask {
	return []domain.CodeTask{
		{
			Type:        "Микро-задача с кодом",
			Title:       "Уникальные значения массива",
			Description: "Напишите функцию uniqueValues, которая принимает массив и возвращает новый массив без повторяющихся значений, сохраняя исходный порядок.",
			Language:    "javascript",
			FileName:    "solution.js",
			StarterCode: "function uniqueValues(items) {\n  // Напишите решение\n\n}",
			Examples: []domain.CodeExample{
				{Input: "[1, 2, 2, 3]", Output: "[1, 2, 3]"},
			},
			Tests: []domain.CodeTest{
				{ID: "array", Title: "Обычный массив"},
				{ID: "strings", Title: "Дубликаты строк"},
				{ID: "empty", Title: "Пустой массив"},
				{ID: "mixed", Title: "Смешанные типы"},
			},
			Hints: []string{
				"Объект Set хранит только уникальные значения.",
				"Set можно превратить обратно в массив через spread.",
				"Порядок первого вхождения Set сохраняет сам.",
			},
			Solution: domain.CodeSolution{
				Code:        "return [...new Set(items)]",
				Explanation: "Set удаляет дубликаты, а spread превращает коллекцию обратно в массив.",
			},
		},
		{
			Type:        "Микро-задача с кодом",
			Title:       "Сумма чисел",
			Description: "Напишите функцию sum, которая принимает массив чисел и возвращает их сумму. Для пустого массива верните 0.",
			Language:    "javascript",
			FileName:    "solution.js",
			StarterCode: "function sum(numbers) {\n  // Напишите решение\n\n}",
			Examples: []domain.CodeExample{
				{Input: "[1, 2, 3]", Output: "6"},
			},
			Tests: []domain.CodeTest{
				{ID: "positive", Title: "Положительные числа"},
				{ID: "negative", Title: "Отрицательные числа"},
				{ID: "empty", Title: "Пустой массив"},
				{ID: "single", Title: "Один элемент"},
			},
			Hints: []string{
				"Массив можно обойти методом reduce.",
				"Начальное значение аккумулятора — 0.",
				"Пустой массив с начальным значением 0 сразу вернёт 0.",
			},
			Solution: domain.CodeSolution{
				Code:        "return numbers.reduce((total, value) => total + value, 0)",
				Explanation: "reduce складывает элементы, а нулевой аккумулятор покрывает пустой массив.",
			},
		},
	}
}
