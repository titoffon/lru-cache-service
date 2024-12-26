// Package server реализует HTTP-сервер, предоставляющий API для работы с LRU-кешем.
package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/titoffon/lru-cache-service/pkg/cache"
)

// Server хранит ссылки на http.Server и реализацию кэша.
type Server struct {
	httpServer *http.Server
	cache      cache.ILRUCache
}

// NewServer создаёт новый Server, регистрирует все HTTP-эндпоинты.
// Возвращает ссылку на сконфигурированный Server.
func NewServer(addr string, cacheSize int, defaultCacheTTL time.Duration) *Server {
	r := chi.NewRouter()

	lru := cache.NewLRUCache(cacheSize, defaultCacheTTL)

	s := &Server{
		cache: lru,
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           r,
			ReadHeaderTimeout: 10 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}

	r.Post("/api/lru", s.handlePost)
	r.Get("/api/lru/{key}", s.handleGet)
	r.Get("/api/lru", s.handleGetAll)
	r.Delete("/api/lru/{key}", s.handleDelete)
	r.Delete("/api/lru", s.handleDeleteAll)

	return s
}

// Start запускает HTTP-сервер в текущем горутине (блокирует).
// Возвращает ошибку, если сервер не смог стартовать или завершился с ошибкой.
func (s *Server) Start() error {

	stop := make(chan os.Signal, 1)

	signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)

	s.httpServer.Handler = s.loggingMiddleware(s.httpServer.Handler)
	go func() {
		slog.Info("Server is starting", slog.String("addr", s.httpServer.Addr))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case sig := <-stop:
		slog.Info("Received shutdown signal",
			slog.String("signal", sig.String()),
		)

		shutdownStart := time.Now()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(ctx); err != nil {
			slog.Error("Graceful shutdown failed", slog.String("error", err.Error()))
			return err
		}

		shutdownElapsed := time.Since(shutdownStart)
		slog.Info("Server gracefully stopped",
			slog.Duration("shutdown_time", shutdownElapsed),
		)
		return nil

	case err := <-errChan:

		return err
	}
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)

		slog.Debug("Incoming request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remote_addr", r.RemoteAddr))
	})
}
