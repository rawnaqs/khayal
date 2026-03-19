package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rawnaqs/khayal/internal/api/middleware"
	"github.com/rawnaqs/khayal/internal/config"
	"github.com/rawnaqs/khayal/internal/llm"
	"github.com/rawnaqs/khayal/internal/queue"
	"github.com/rawnaqs/khayal/internal/vault"
)

type Server struct {
	router *chi.Mux
	config *config.Config
	queue  *queue.Queue
	vault  *vault.Writer
	llm    llm.LLMExt
	logger *slog.Logger
}

func NewServer(cfg *config.Config, q *queue.Queue, v *vault.Writer, l llm.LLMExt, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Server{
		config: cfg,
		queue:  q,
		vault:  v,
		llm:    l,
		logger: logger,
	}
	s.setupRouter()
	return s
}

func (s *Server) setupRouter() {
	s.router = chi.NewRouter()

	s.router.Use(middleware.RequestLogger(s.logger))

	s.router.Get("/v1/health", s.healthHandler)

	s.router.Route("/v1", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(s.config.Server.Token, WriteError))

		r.Post("/capture", s.captureHandler)
		r.Get("/search", s.searchHandler)
		r.Get("/queue", s.queueListHandler)
		r.Get("/queue/{id}", s.queueGetHandler)
		r.Post("/queue/{id}/retry", s.queueRetryHandler)
		r.Post("/queue/{id}/discard", s.queueDiscardHandler)
	})
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)

	srv := &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	go srv.ListenAndServe()

	s.logger.Info("server started", "addr", addr)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	s.logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.config.Server.ShutdownTimeoutS)*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	s.logger.Info("server stopped")
	return nil
}

func (s *Server) Close() error {
	return s.queue.Close()
}
