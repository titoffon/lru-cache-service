package main

import (
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/titoffon/lru-cache-service/internal/cache"
	"github.com/titoffon/lru-cache-service/internal/server"
)

type config struct {
  	ServerHostPort  string        `env:"SERVER_HOST_PORT" envDefault:"localhost:8080"`
	CacheSize       int           `env:"CACHE_SIZE" envDefault:"10"`
	DefaultCacheTTL time.Duration `env:"DEFAULT_CACHE_TTL" envDefault:"60s"`
	LogLevel        string        `env:"LOG_LEVEL" envDefault:"DEBUG"`
}

func main(){
	var cfg config
	if err := env.Parse(&cfg); err != nil {
		slog.Error("Failed to parse configuration")
	}

	
	serverHostPortFlag := flag.String("server-host-port", cfg.ServerHostPort, "server host and port")
	cacheSizeFlag := flag.Int("cache-size", cfg.CacheSize, "LRU cache size")
	cacheTTLFlag := flag.String("default-cache-ttl", cfg.DefaultCacheTTL.String(), "default TTL (e.g. 30s, 1m, 2m30s)")
	logLevelFlag := flag.String("log-level", cfg.LogLevel, "log level (DEBUG|INFO|WARN|ERROR)")

	flag.Parse()

	cfg.ServerHostPort = *serverHostPortFlag
	cfg.CacheSize = *cacheSizeFlag
	cfg.LogLevel = *logLevelFlag

	level := parseLogLevel(cfg.LogLevel)
	//устанавливаем глобальный логгер, 
	//&slog.HandlerOptions указывает уровень логирования. Логи ниже этого уровня игнорируются
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))


	ttl, err := time.ParseDuration(*cacheTTLFlag)
	if err != nil || ttl <= 0 {
		//если некорректное значение, оставим дефолт (из ENV)
		slog.Warn("Invalid TTL provided, using default value", slog.String("ttl", *cacheTTLFlag))
	} else {
		cfg.DefaultCacheTTL = ttl
	}
	
	//логируем конфигурацию на уровне DEBUG
	slog.Debug("Application configuration",
		slog.String("server_host_port", cfg.ServerHostPort),
		slog.Int("cache_size", cfg.CacheSize),
		slog.String("cache_ttl", cfg.DefaultCacheTTL.String()),
		slog.String("log_level", cfg.LogLevel),
	)

	lru := cache.NewLRUCache(cfg.CacheSize, cfg.DefaultCacheTTL)

	srv := server.NewServer(cfg.ServerHostPort, lru)

	slog.Info("Starting server", slog.String("address", cfg.ServerHostPort))
	if err := srv.Start(); err != nil {
		slog.Error("Failed to start server", slog.String("error", err.Error()))
	}
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelWarn
	}
}