package main

import (
	"log/slog"
	"time"

	"github.com/titoffon/lru-cache-service/internal/cache"
	"github.com/titoffon/lru-cache-service/internal/server"
)

func main(){
	var (
		defaultHostPort   = "localhost:8080"
		defaultCacheSize  = 10
		defaultCacheTTL   = 60 * time.Second
	)

	logger := slog.Default()

	lru := cache.NewLRUCache(defaultCacheSize, defaultCacheTTL)

	srv := server.NewServer(defaultHostPort, lru)

	if err := srv.Start(); err != nil {
		logger.Error("Failed to start server")
	}
}