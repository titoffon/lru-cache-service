package main

import (
	"log/slog"

	"github.com/titoffon/lru-cache-service/internal/config"
	"github.com/titoffon/lru-cache-service/internal/server"
	"github.com/titoffon/lru-cache-service/pkg/logger"
)

func main(){

	cfg, err := config.ReadConfig()
	if err != nil {
		panic("Failed to parse configuration")
	}

	logger.InitGlobalLogger(cfg.LogLevel)

	srv := server.NewServer(cfg.ServerHostPort, cfg.CacheSize, cfg.DefaultCacheTTL)

	slog.Info("Starting server", slog.String("address", cfg.ServerHostPort))
	if err := srv.Start(); err != nil {
		slog.Error("Failed to start server", slog.String("error", err.Error()))
	}
}