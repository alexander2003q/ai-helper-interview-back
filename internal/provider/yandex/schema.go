package yandex

import "encoding/json"

const theorySchemaName = "theory_questions"

func theoryQuestionsSchema(count int) json.RawMessage {
	answer := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"id", "text", "explanation"},
		"properties": map[string]any{
			"id": map[string]any{
				"type": "string",
				"enum": []string{"a", "b", "c", "d"},
			},
			"text":        map[string]any{"type": "string"},
			"explanation": map[string]any{"type": "string"},
		},
	}

	question := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"prompt", "answers", "correctAnswerId", "explanation"},
		"properties": map[string]any{
			"prompt":      map[string]any{"type": "string"},
			"explanation": map[string]any{"type": "string"},
			"correctAnswerId": map[string]any{
				"type": "string",
				"enum": []string{"a", "b", "c", "d"},
			},
			"answers": map[string]any{
				"type":     "array",
				"minItems": 4,
				"maxItems": 4,
				"items":    answer,
			},
		},
	}

	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"questions"},
		"properties": map[string]any{
			"questions": map[string]any{
				"type":     "array",
				"minItems": count,
				"maxItems": count,
				"items":    question,
			},
		},
	}

	encoded, err := json.Marshal(schema)
	if err != nil {
		panic(err)
	}

	return encoded
}
