package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"ai-helper-interview/backend/internal/domain"
	"ai-helper-interview/backend/internal/ports"
	"ai-helper-interview/backend/internal/usecase"
)

const maxRequestBodySize = 1 << 20

type QuestionnaireHandler struct {
	createQuestionnaire *usecase.CreateQuestionnaire
	logger              *slog.Logger
}

func NewQuestionnaireHandler(
	createQuestionnaire *usecase.CreateQuestionnaire,
	logger *slog.Logger,
) *QuestionnaireHandler {
	return &QuestionnaireHandler{
		createQuestionnaire: createQuestionnaire,
		logger:              logger,
	}
}

func (h *QuestionnaireHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}

	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		writeError(
			w,
			http.StatusUnsupportedMediaType,
			"unsupported_media_type",
			"Content-Type must be application/json",
			"",
		)
		return
	}

	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var request createQuestionnaireRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body contains invalid JSON", "")
		return
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must contain one JSON object", "")
		return
	}

	questionnaire, err := h.createQuestionnaire.Execute(r.Context(), request.toDomain())
	if err != nil {
		var validationError *domain.ValidationError
		if errors.As(err, &validationError) {
			writeError(
				w,
				http.StatusUnprocessableEntity,
				"validation_error",
				validationError.Message,
				validationError.Field,
			)
			return
		}

		h.logger.Error("create questionnaire failed", "error", err)

		if errors.Is(err, ports.ErrQuestionGeneration) {
			writeError(
				w,
				http.StatusBadGateway,
				"question_generation_failed",
				"could not generate questionnaire",
				"",
			)
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error", "")
		return
	}

	writeJSON(w, http.StatusCreated, toQuestionnaireResponse(questionnaire))
}
