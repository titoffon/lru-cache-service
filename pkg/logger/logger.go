// Package logger хранит в себе логику установки глобального уровня логировния
package logger

import (
	"log/slog"
	"os"
)

// InitGlobalLogger инициализирует глобальный логер с заданным уровнем.
func InitGlobalLogger(levelStr string) {
	level := ParseLogLevel(levelStr)
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))
}

// ParseLogLevel преобразует строковое значение уровня логирования в тип slog.Level.
func ParseLogLevel(level string) slog.Level {
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