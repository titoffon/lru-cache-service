package server

import (
	"encoding/json"
	"log/slog"
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
	start := time.Now()

	var req requestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("Invalid JSON in POST request",
			slog.String("error", err.Error()),
			slog.String("method", r.Method),
			slog.String("url", r.URL.Path),
		)
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Key == "" {
		slog.Warn("Missing key in POST request",
			slog.String("method", r.Method),
			slog.String("url", r.URL.Path),
		)
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	ttl := time.Duration(0)
	if req.TTLSeconds != nil {
		if *req.TTLSeconds < 0 {
			slog.Warn("Invalid TTL in POST request",
				slog.String("key", req.Key),
				slog.Int64("ttl_seconds", *req.TTLSeconds),
			)
			http.Error(w, "ttl_seconds must be >= 0", http.StatusBadRequest)
			return
		}
		ttl = time.Duration(*req.TTLSeconds) * time.Second
	}

	ctx := context.Background()
	if err := s.cache.Put(ctx, req.Key, req.Value, ttl); err != nil {
		slog.Error("Failed to store data in cache",
			slog.String("key", req.Key),
			slog.String("error", err.Error()),
		)
		http.Error(w, "failed to put data", http.StatusInternalServerError)
		return
	}

		slog.Info("Data stored successfully",
		slog.String("key", req.Key),
		slog.Duration("ttl", ttl),
		slog.Duration("duration", time.Since(start)),
	)

	w.WriteHeader(http.StatusCreated) // 201
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	key := chi.URLParam(r, "key")
	if key == "" {
		slog.Warn("Missing key in GET request",
			slog.String("method", r.Method),
			slog.String("url", r.URL.Path),
		)
		http.Error(w, "Missing key in GET request", http.StatusBadRequest)
		return
	}

	value, expiresAt, err := s.cache.Get(r.Context(), key)
	if err != nil {
		if err == cache.ErrKeyNotFound {
			slog.Warn("Key not found in GET request",
				slog.String("key", key),
			)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("Failed to retrieve data",
			slog.String("key", key),
			slog.String("error", err.Error()),
		)
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

	slog.Info("Data retrieved successfully",
		slog.String("key", key),
		slog.Duration("duration", time.Since(start)),
	)
}

func (s *Server) handleGetAll(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	keys, values, err := s.cache.GetAll(r.Context())
	if err != nil {
		slog.Error("Failed to retrieve all data from cache",
			slog.String("error", err.Error()),
		)
		http.Error(w, "Failed to retrieve all data from cache", http.StatusInternalServerError)
		return
	}
	if len(keys) == 0 {
		// 204 - нет контента
		slog.Info("No content in GET all request")
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

	slog.Info("All data retrieved successfully",
		slog.Int("keys_count", len(keys)),
		slog.Duration("duration", time.Since(start)),
	)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	key := chi.URLParam(r, "key")
	if key == "" {
		slog.Warn("Missing key in DELETE request",
			slog.String("method", r.Method),
			slog.String("url", r.URL.Path),
		)
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}
	_, err := s.cache.Evict(r.Context(), key)
	if err != nil {
		if err == cache.ErrKeyNotFound {
			slog.Warn("Key not found in DELETE request",
				slog.String("key", key),
			)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Key not found", http.StatusInternalServerError)
		return
	}

		slog.Info("Key deleted successfully",
		slog.String("key", key),
		slog.Duration("duration", time.Since(start)),
	)

	// 204 - успешное удаление
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteAll(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if err := s.cache.EvictAll(r.Context()); err != nil {
		slog.Error("Failed to evict all data",
			slog.String("error", err.Error()),
		)
		http.Error(w, "Failed to evict all data", http.StatusInternalServerError)
		return
	}

	slog.Info("All data evicted successfully",
		slog.Duration("duration", time.Since(start)),
	)
	
	w.WriteHeader(http.StatusNoContent)
}