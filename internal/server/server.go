// Package server provides HTTP server lifecycle management with graceful shutdown.
package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/raizel/gateway/internal/config"
)

// Server wraps the standard http.Server with graceful shutdown support.
type Server struct {
	httpServer *http.Server
	logger     *zap.Logger
	cfg        *config.ServerConfig
}

// New creates a new Server with the given handler and configuration.
func New(handler http.Handler, cfg *config.ServerConfig, logger *zap.Logger) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
		logger: logger,
		cfg:    cfg,
	}
}

// Start begins listening for HTTP requests and blocks until a shutdown signal is received.
// It handles SIGINT and SIGTERM for graceful shutdown.
func (s *Server) Start() error {
	// Channel to receive OS signals.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Channel to receive server errors.
	errCh := make(chan error, 1)

	go func() {
		s.logger.Info("starting HTTP server",
			zap.String("addr", s.httpServer.Addr),
		)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Block until signal or error.
	select {
	case sig := <-quit:
		s.logger.Info("received shutdown signal", zap.String("signal", sig.String()))
	case err := <-errCh:
		s.logger.Error("server error", zap.Error(err))
		return err
	}

	// Graceful shutdown with timeout.
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
	defer cancel()

	s.logger.Info("shutting down gracefully",
		zap.Duration("timeout", s.cfg.ShutdownTimeout),
	)

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("forced shutdown", zap.Error(err))
		return err
	}

	s.logger.Info("server stopped cleanly")
	return nil
}

// Stop programmatically triggers a graceful shutdown.
func (s *Server) Stop(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}
