package httpapi

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
)

func NewRouter(
	handler *QuestionnaireHandler,
	allowedOrigins string,
	logger *slog.Logger,
) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/questionnaires", handler.Create)

	return recoveryMiddleware(
		logger,
		loggingMiddleware(
			logger,
			corsMiddleware(allowedOrigins, mux),
		),
	)
}

func corsMiddleware(allowedOrigins string, next http.Handler) http.Handler {
	origins := make(map[string]struct{})
	allowAll := false
	for _, origin := range strings.Split(allowedOrigins, ",") {
		origin = strings.TrimSpace(origin)
		if origin == "*" {
			allowAll = true
			continue
		}
		if origin != "" {
			origins[origin] = struct{}{}
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		_, isAllowed := origins[origin]
		if origin != "" && (isAllowed || allowAll) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		logger.Info(
			"request started",
			"method", r.Method,
			"path", r.URL.Path,
		)

		next.ServeHTTP(recorder, r)

		logger.Info(
			"request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"duration", time.Since(startedAt),
		)
	})
}

func recoveryMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("panic recovered", "error", recovered)
				writeError(
					w,
					http.StatusInternalServerError,
					"internal_error",
					"internal server error",
					"",
				)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
