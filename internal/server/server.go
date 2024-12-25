package server

import (
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

    err := s.httpServer.ListenAndServe()
    if err != nil && err != http.ErrServerClosed {
        return err
    }
	
    return nil
}
