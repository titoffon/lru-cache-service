package server

import (
	"encoding/json"
	"net/http"
	"time"

	"context"

	"github.com/go-chi/chi/v5"

	"github.com/titoffon/lru-cache-service/internal/cache"
)

//тело POST-запроса
type requestBody struct {
	Key        string      `json:"key"`
	Value      interface{} `json:"value"`
	TTLSeconds *int64      `json:"ttl_seconds,omitempty"`
}

//тело GET-ответа
type responseBody struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	ExpiresAt int64       `json:"expires_at"`
}

func (s *Server) handlePost(w http.ResponseWriter, r *http.Request) {

	var req requestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	ttl := time.Duration(0)
	if req.TTLSeconds != nil {
		if *req.TTLSeconds < 0 {
			http.Error(w, "ttl_seconds must be >= 0", http.StatusBadRequest)
			return
		}
		ttl = time.Duration(*req.TTLSeconds) * time.Second
	}

	ctx := context.Background()
	if err := s.cache.Put(ctx, req.Key, req.Value, ttl); err != nil {
		http.Error(w, "failed to put data", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated) // 201
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {

	key := chi.URLParam(r, "key")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	value, expiresAt, err := s.cache.Get(r.Context(), key)
	if err != nil {
		if err == cache.ErrKeyNotFound {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get data", http.StatusInternalServerError)
		return
	}

	resp := responseBody{
		Key:       key,
		Value:     value,
		ExpiresAt: expiresAt.Unix(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleGetAll(w http.ResponseWriter, r *http.Request) {

	keys, values, err := s.cache.GetAll(r.Context())
	if err != nil {
		http.Error(w, "failed to get data", http.StatusInternalServerError)
		return
	}
	if len(keys) == 0 {
		// 204 - нет контента
		w.WriteHeader(http.StatusNoContent)
		return
	}

	resp := struct {
		Keys   []string      `json:"keys"`
		Values []interface{} `json:"values"`
	}{
		Keys:   keys,
		Values: values,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {

	key := chi.URLParam(r, "key")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}
	_, err := s.cache.Evict(r.Context(), key)
	if err != nil {
		if err == cache.ErrKeyNotFound {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to evict data", http.StatusInternalServerError)
		return
	}
	// 204 - успешное удаление
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteAll(w http.ResponseWriter, r *http.Request) {

	if err := s.cache.EvictAll(r.Context()); err != nil {
		http.Error(w, "failed to evict all data", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}