package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/titoffon/lru-cache-service/pkg/cache"
)

func TestHandlePost(t *testing.T) {
	mockCache := cache.NewLRUCache(10, time.Minute)

	srv := &Server{cache: mockCache}

	tests := []struct {
		name       string
		body       string
		statusCode int
	}{
		{
			name:       "Valid request",
			body:       `{"key": "testKey", "value": "testValue", "ttl_seconds": 60}`,
			statusCode: http.StatusCreated,
		},
		{
			name:       "Missing key",
			body:       `{"value": "testValue"}`,
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "Invalid TTL",
			body:       `{"key": "testKey", "value": "testValue", "ttl_seconds": -10}`,
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "Invalid JSON",
			body:       `invalid_json`,
			statusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/lru", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()

			srv.handlePost(rec, req)

			assert.Equal(t, tt.statusCode, rec.Code)
		})
	}
}

func TestHandleGet(t *testing.T) {

	mockCache := cache.NewLRUCache(10, time.Minute)
	mockCache.Put(context.Background(), "testKey", "testValue", time.Minute)

	srv := &Server{cache: mockCache}

	// Test cases
	tests := []struct {
		name       string
		key        string
		statusCode int
		response   responseBody
	}{
		{
			name:       "Valid key",
			key:        "testKey",
			statusCode: http.StatusOK,
			response: responseBody{
				Key:       "testKey",
				Value:     "testValue",
				ExpiresAt: time.Now().Add(time.Minute).Unix(),
			},
		},
		{
			name:       "Key not found",
			key:        "missingKey",
			statusCode: http.StatusNotFound,
		},
		{
			name:       "Missing key",
			key:        "",
			statusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//создаёт базовый запрос с методом и URL
			req := httptest.NewRequest(http.MethodGet, "/api/lru/"+tt.key, nil)
			rec := httptest.NewRecorder()
			//чтобы эмулировать поведение маршрутизатора и предоставить параметры пути обработчику
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
				URLParams: chi.RouteParams{Keys: []string{"key"}, Values: []string{tt.key}},
			}))

			srv.handleGet(rec, req)

			assert.Equal(t, tt.statusCode, rec.Code)

			if tt.statusCode == http.StatusOK {
				var resp responseBody
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, tt.response.Key, resp.Key)
				assert.Equal(t, tt.response.Value, resp.Value)
			}
		})
	}
}

func TestHandleGetAll(t *testing.T) {

	mockCache := cache.NewLRUCache(10, time.Minute)
	mockCache.Put(context.Background(), "key1", "value1", time.Minute)
	mockCache.Put(context.Background(), "key2", "value2", time.Minute)

	srv := &Server{cache: mockCache}

	t.Run("Retrieve all keys and values", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/lru", nil)
		rec := httptest.NewRecorder()

		srv.handleGetAll(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp struct {
			Keys   []string      `json:"keys"`
			Values []interface{} `json:"values"`
		}
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NoError(t, err)

		//валидация ответа
		assert.ElementsMatch(t, []string{"key1", "key2"}, resp.Keys)
		assert.ElementsMatch(t, []interface{}{"value1", "value2"}, resp.Values)
	})

	t.Run("No content in cache", func(t *testing.T) {

		mockCache.EvictAll(context.Background())

		req := httptest.NewRequest(http.MethodGet, "/api/lru", nil)
		rec := httptest.NewRecorder()

		srv.handleGetAll(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Empty(t, rec.Body.String())
	})
}

func TestHandleDelete(t *testing.T) {
	mockCache := cache.NewLRUCache(10, time.Minute)
	mockCache.Put(context.Background(), "testKey", "testValue", time.Minute)

	srv := &Server{cache: mockCache}

	tests := []struct {
		name       string
		key        string
		statusCode int
	}{
		{
			name:       "Valid key",
			key:        "testKey",
			statusCode: http.StatusNoContent,
		},
		{
			name:       "Key not found",
			key:        "missingKey",
			statusCode: http.StatusNotFound,
		},
		{
			name:       "Missing key",
			key:        "",
			statusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/lru/"+tt.key, nil)
			rec := httptest.NewRecorder()

			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
				URLParams: chi.RouteParams{Keys: []string{"key"}, Values: []string{tt.key}},
			}))

			srv.handleDelete(rec, req)

			assert.Equal(t, tt.statusCode, rec.Code)
		})
	}
}

func TestHandleDeleteAll(t *testing.T) {
	mockCache := cache.NewLRUCache(10, time.Minute)
	mockCache.Put(context.Background(), "key1", "value1", time.Minute)
	mockCache.Put(context.Background(), "key2", "value2", time.Minute)

	srv := &Server{cache: mockCache}

	t.Run("Delete all keys and values", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/lru", nil)
		rec := httptest.NewRecorder()

		srv.handleDeleteAll(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)

		keys, values, err := mockCache.GetAll(context.Background())
		assert.NoError(t, err)
		assert.Empty(t, keys)
		assert.Empty(t, values)
	})

	//убедиться, что вызов обработчика с пустым кэшем также возвращает 204
	t.Run("Delete all when cache is empty", func(t *testing.T) {

		mockCache.EvictAll(context.Background())

		req := httptest.NewRequest(http.MethodDelete, "/api/lru", nil)
		rec := httptest.NewRecorder()

		srv.handleDeleteAll(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)

		keys, values, err := mockCache.GetAll(context.Background())
		assert.NoError(t, err)
		assert.Empty(t, keys)
		assert.Empty(t, values)
	})
}
