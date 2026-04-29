package api

import (
	"context"
	"embed"
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

//go:embed ui/static/*
var staticFS embed.FS

type Server struct {
	router     *chi.Mux
	config     *config.Config
	queue      *queue.Queue
	vault      *vault.Writer
	vaultReader *vault.Reader
	llm        llm.LLMExt
	logger     *slog.Logger
}

func NewServer(cfg *config.Config, q *queue.Queue, v *vault.Writer, l llm.LLMExt, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Server{
		config: cfg,
		queue:  q,
		vault:  v,
		vaultReader: vault.NewReader(v.BasePath(), cfg.Vault.InboxDir),
		llm:    l,
		logger: logger,
	}
	s.setupRouter()
	return s
}

func (s *Server) setupRouter() {
	s.router = chi.NewRouter()

	s.router.Use(middleware.RequestLogger(s.logger))

		s.router.Route("/v1", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(s.config.Server.Token, WriteError))

		r.Get("/health", s.healthHandler)
		r.Post("/capture", s.captureHandler)
		r.Get("/search", s.searchHandler)
		r.Get("/stats", s.statsHandler)
		r.Get("/notes/{path:.*}", s.noteHandler)
		r.Get("/queue", s.queueListHandler)
		r.Get("/queue/{id}", s.queueGetHandler)
		r.Post("/queue/{id}/retry", s.queueRetryHandler)
		r.Post("/queue/{id}/discard", s.queueDiscardHandler)
	})

	// Static file serving (after API routes)
	s.router.Get("/*", s.staticHandler)
}

func (s *Server) staticHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	// Try to serve the file from embedded static FS
	f, err := staticFS.Open("ui/static" + path)
	if err != nil {
		// File not found - SPA fallback to index.html
		http.ServeFileFS(w, r, staticFS, "ui/static/index.html")
		return
	}
	f.Close()

	// Serve the static file
	http.ServeFileFS(w, r, staticFS, "ui/static"+path)
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)

	srv := &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("server error", "error", err)
		}
	}()

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
