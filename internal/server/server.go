package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/titoffon/lru-cache-service/internal/cache"
)

//Server хранит кэш и конфиг, содержит HTTP-сервер
type Server struct {
	httpServer *http.Server
	cache      cache.ILRUCache
}

func NewServer(addr string, cache cache.ILRUCache) *Server {
	r := chi.NewRouter()

	s := &Server{
		cache: cache,
		httpServer: &http.Server{
			Addr:    addr,
			Handler: r,
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

func (s *Server) Start() error {
	s.httpServer.Handler = s.loggingMiddleware(s.httpServer.Handler)
	err := s.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
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