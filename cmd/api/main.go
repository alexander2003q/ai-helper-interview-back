package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ai-helper-interview/backend/internal/config"
	"ai-helper-interview/backend/internal/delivery/httpapi"
	"ai-helper-interview/backend/internal/provider/memory"
	"ai-helper-interview/backend/internal/provider/yandex"
	"ai-helper-interview/backend/internal/usecase"
)

func main() {
	cfg, err := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err != nil {
		logger.Error("configuration is invalid", "error", err)
		os.Exit(1)
	}

	questions := yandex.NewTheoryQuestionGenerator(cfg.Yandex)
	codeTasks := memory.NewCodeTaskGenerator()
	createQuestionnaire := usecase.NewCreateQuestionnaire(questions, codeTasks)
	handler := httpapi.NewQuestionnaireHandler(createQuestionnaire, logger)
	router := httpapi.NewRouter(handler, cfg.AllowedOrigins, logger)

	server := &http.Server{
		Addr:              cfg.ServerAddr,
		Handler:           router,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server started", "address", cfg.ServerAddr)
		serverErrors <- server.ListenAndServe()
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	case receivedSignal := <-signals:
		logger.Info("shutdown signal received", "signal", receivedSignal.String())
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		_ = server.Close()
		os.Exit(1)
	}

	logger.Info("server stopped", "time", time.Now().UTC())
}
