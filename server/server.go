package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"rag-document-assistant/config"
)

type Server struct {
	http   *http.Server
	logger *slog.Logger
}

func New(cfg config.ServerConfig, handler http.Handler, logger *slog.Logger) *Server {
	return &Server{http: &http.Server{
		Addr: cfg.Address(), Handler: handler, ReadTimeout: cfg.ReadTimeout(),
		WriteTimeout: cfg.WriteTimeout(), IdleTimeout: cfg.WriteTimeout(),
	}, logger: logger}
}

func (s *Server) Start() error {
	s.logger.Info("HTTP server starting", "address", s.http.Addr)
	if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error { return s.http.Shutdown(ctx) }
