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

	"github.com/titoffon/lru-cache-service/internal/cache"
)

// Server хранит ссылки на http.Server и реализацию кэша.
type Server struct {
	httpServer *http.Server
	cache      cache.ILRUCache
}

// NewServer создаёт новый Server, регистрирует все HTTP-эндпоинты.
// Возвращает ссылку на сконфигурированный Server.
func NewServer(addr string, cache cache.ILRUCache) *Server {
	r := chi.NewRouter()

	s := &Server{
		cache: cache,
		httpServer: &http.Server{
			Addr:    			addr,
			Handler: 			r,
			ReadHeaderTimeout:	10 * time.Second, // например
    		IdleTimeout:   		60 * time.Second,
		},
	}

	//Настраиваем роуты
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
	//канал для приёма системных сигналов.
	stop := make(chan os.Signal, 1)
	//подписываемся на сигналы SIGINT (Ctrl+C) и SIGTERM
	signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Канал для передачи ошибок, которые могут возникнуть при ListenAndServe.
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
		// Пришёл сигнал (SIGINT или SIGTERM)
		slog.Info("Received shutdown signal",
			slog.String("signal", sig.String()),
		)

		shutdownStart := time.Now()

		// Создаём контекст с таймаутом
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Вызываем Shutdown — это говорит серверу не принимать новых запросов
		// и подождать, пока все активные запросы завершатся или пока не истечёт таймаут.
		if err := s.httpServer.Shutdown(ctx); err != nil {
			slog.Error("Graceful shutdown failed", slog.String("error", err.Error()))
			return err
		}
		// Логируем, сколько заняла остановка
		shutdownElapsed := time.Since(shutdownStart)
		slog.Info("Server gracefully stopped",
			slog.Duration("shutdown_time", shutdownElapsed),
		)
		return nil
			
	case err := <-errChan:
		// Если сервер упал не по причине Shutdown, возвращаем ошибку
		return err
	}
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//start := time.Now()
		next.ServeHTTP(w, r)
		//duration := time.Since(start)

		slog.Debug("Incoming request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			//slog.Duration("duration", duration),
			slog.String("remote_addr", r.RemoteAddr))
	})
}