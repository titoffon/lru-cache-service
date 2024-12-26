package main

import (
	"log/slog"

	"github.com/titoffon/lru-cache-service/internal/config"
	"github.com/titoffon/lru-cache-service/internal/server"
	"github.com/titoffon/lru-cache-service/pkg/cache"
	"github.com/titoffon/lru-cache-service/pkg/logger"
)

func main(){

	cfg, err := config.ReadConfig()
	if err != nil {
		slog.Error("Failed to parse configuration")
		return
	}

	logger.InitGlobalLogger(cfg.LogLevel)

	lru := cache.NewLRUCache(cfg.CacheSize, cfg.DefaultCacheTTL) // в Newserver

	srv := server.NewServer(cfg.ServerHostPort, lru)

	slog.Info("Starting server", slog.String("address", cfg.ServerHostPort))
	if err := srv.Start(); err != nil {
		slog.Error("Failed to start server", slog.String("error", err.Error()))
	}
}