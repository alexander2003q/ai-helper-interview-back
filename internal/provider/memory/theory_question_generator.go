package memory

import (
	"context"

	"ai-helper-interview/backend/internal/domain"
)

type TheoryQuestionGenerator struct{}

func NewTheoryQuestionGenerator() *TheoryQuestionGenerator {
	return &TheoryQuestionGenerator{}
}

func (g *TheoryQuestionGenerator) Generate(
	ctx context.Context,
	input domain.TheoryGenerationInput,
) ([]domain.Question, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	questions := mockQuestions()
	result := make([]domain.Question, 0, input.Count)
	for index := 0; index < input.Count; index++ {
		question := questions[index%len(questions)]
		question.ID = index + 1
		result = append(result, question)
	}

	return result, nil
}

func mockQuestions() []domain.Question {
	return []domain.Question{
		newQuestion(
			1,
			"Что произойдёт при обновлении state внутри useEffect без массива зависимостей?",
			"b",
			"Эффект запускается после каждого рендера, а обновление state вызывает следующий рендер.",
			"Эффект выполнится только один раз",
			"Возникнет цикл перерендеров",
			"React проигнорирует setState",
			"Компонент сразу размонтируется",
		),
		newQuestion(
			2,
			"Для чего React использует свойство key при рендере списка?",
			"c",
			"Стабильный key помогает React понять, какой элемент изменился, добавился или был удалён.",
			"Для стилизации элементов списка",
			"Для сортировки данных по алфавиту",
			"Для сопоставления элементов между рендерами",
			"Для передачи значения в props",
		),
		newQuestion(
			3,
			"Когда имеет смысл использовать useMemo?",
			"b",
			"useMemo полезен, когда повторное вычисление действительно дорого и зависимости меняются нечасто.",
			"Для каждого вычисления в компоненте",
			"Для кэширования дорогого вычисления",
			"Для выполнения сетевого запроса",
			"Для замены локального state",
		),
		newQuestion(
			4,
			"Что гарантирует функциональная форма setState?",
			"a",
			"Функция обновления получает последнее доступное значение state, что важно для зависимых обновлений.",
			"Доступ к актуальному предыдущему значению",
			"Синхронный рендер компонента",
			"Отмену следующего обновления",
			"Сохранение state в localStorage",
		),
		newQuestion(
			5,
			"Что делает React.memo?",
			"b",
			"React.memo выполняет поверхностное сравнение props и позволяет не рендерить компонент повторно.",
			"Запоминает значение локального state",
			"Сравнивает props и может пропустить рендер",
			"Оборачивает компонент в Suspense",
			"Удаляет неиспользуемые эффекты",
		),
		newQuestion(
			6,
			"Где допустимо вызывать React-хуки?",
			"c",
			"Порядок вызова хуков должен оставаться одинаковым между рендерами.",
			"Внутри условий и циклов",
			"Только в классовых компонентах",
			"На верхнем уровне компонента или своего хука",
			"В любом обработчике события",
		),
		newQuestion(
			7,
			"Зачем useEffect может возвращать функцию?",
			"a",
			"Cleanup-функция удаляет подписки, таймеры и другие ресурсы перед новым эффектом или размонтированием.",
			"Чтобы очистить созданный эффектом ресурс",
			"Чтобы вернуть JSX",
			"Чтобы изменить props",
			"Чтобы остановить первый рендер",
		),
		newQuestion(
			8,
			"Как Context влияет на подписанные компоненты?",
			"a",
			"Потребители контекста перерендериваются, когда меняется значение ближайшего Provider.",
			"Они обновляются при изменении value провайдера",
			"Они никогда не перерендериваются",
			"Они получают value только один раз",
			"Они становятся серверными компонентами",
		),
		newQuestion(
			9,
			"Для чего используется useRef?",
			"b",
			"Изменение поля current не вызывает рендер; ref также часто связывают с DOM-элементами.",
			"Только для хранения DOM-узла",
			"Для изменяемого значения без запуска рендера",
			"Для объявления глобальной переменной",
			"Для мемоизации компонента",
		),
		newQuestion(
			10,
			"Что такое controlled input в React?",
			"a",
			"Управляемое поле получает value из React-state и обновляет его через обработчик.",
			"Поле, значение которого хранится в state",
			"Поле только для чтения",
			"Поле без обработчика событий",
			"Нативный элемент вне React",
		),
		newQuestion(
			11,
			"Что делает StrictMode в режиме разработки?",
			"b",
			"StrictMode включает дополнительные проверки и может повторять некоторые вызовы в development.",
			"Оптимизирует production bundle",
			"Помогает обнаруживать небезопасные побочные эффекты",
			"Запрещает использовать хуки",
			"Автоматически исправляет ошибки",
		),
		newQuestion(
			12,
			"Когда React выполняет reconciliation?",
			"a",
			"Во время reconciliation React определяет минимальный набор изменений для интерфейса.",
			"При сравнении предыдущего и нового дерева элементов",
			"Только при загрузке страницы",
			"Только после сетевого запроса",
			"До создания JSX",
		),
		newQuestion(
			13,
			"Как избежать устаревшего значения в обработчике обновления state?",
			"b",
			"Функциональное обновление получает актуальное предыдущее значение независимо от замыкания.",
			"Всегда использовать setTimeout",
			"Использовать функциональное обновление",
			"Мутировать state напрямую",
			"Удалить StrictMode",
		),
	}
}

func newQuestion(
	id int,
	prompt string,
	correctAnswerID string,
	explanation string,
	answers ...string,
) domain.Question {
	answerIDs := []string{"a", "b", "c", "d"}
	options := make([]domain.Answer, 0, len(answers))

	for index, answer := range answers {
		options = append(options, domain.Answer{
			ID:   answerIDs[index],
			Text: answer,
		})
	}

	return domain.Question{
		ID:              id,
		Type:            "Теоретический вопрос",
		Prompt:          prompt,
		Answers:         options,
		CorrectAnswerID: correctAnswerID,
		Explanation:     explanation,
	}
}
