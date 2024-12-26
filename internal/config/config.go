// Package config реализует логику, отвечающую за конфигурацию
package config

import (
	"flag"
	"log/slog"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
  	ServerHostPort  string        `env:"SERVER_HOST_PORT" envDefault:"localhost:8080"`
	CacheSize       int           `env:"CACHE_SIZE" envDefault:"10"`
	DefaultCacheTTL time.Duration `env:"DEFAULT_CACHE_TTL" envDefault:"60s"`
	LogLevel        string        `env:"LOG_LEVEL" envDefault:"WARN"`
}

func ReadConfig() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}

	serverHostPortFlag := flag.String("server-host-port", cfg.ServerHostPort, "server host and port")
	cacheSizeFlag := flag.Int("cache-size", cfg.CacheSize, "LRU cache size")
	cacheTTLFlag := flag.String("default-cache-ttl", cfg.DefaultCacheTTL.String(), "default TTL (e.g. 30s, 1m, 2m30s)")
	logLevelFlag := flag.String("log-level", cfg.LogLevel, "log level (DEBUG|INFO|WARN|ERROR)")

	flag.Parse()

	cfg.ServerHostPort = *serverHostPortFlag
	cfg.CacheSize = *cacheSizeFlag
	cfg.LogLevel = *logLevelFlag

	ttl, err := time.ParseDuration(*cacheTTLFlag)
	if err != nil || ttl <= 0 {
		slog.Warn("Invalid TTL provided, using default value", slog.String("ttl", *cacheTTLFlag))
	} else {
		cfg.DefaultCacheTTL = ttl
	}

	slog.Debug("Application configuration",
		slog.String("server_host_port", cfg.ServerHostPort),
		slog.Int("cache_size", cfg.CacheSize),
		slog.String("cache_ttl", cfg.DefaultCacheTTL.String()),
		slog.String("log_level", cfg.LogLevel),
	)

	return &cfg, nil
}