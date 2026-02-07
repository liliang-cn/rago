package log

import (
	"fmt"
	"log/slog"
	"os"
)

var (
	defaultLogger *slog.Logger
	levelVar      *slog.LevelVar
)

func init() {
	levelVar = &slog.LevelVar{}
	levelVar.Set(slog.LevelInfo)

	opts := &slog.HandlerOptions{
		Level: levelVar,
	}

	handler := slog.NewTextHandler(os.Stderr, opts)
	defaultLogger = slog.New(handler)
}

func SetLevel(level slog.Level) { levelVar.Set(level) }

func SetDebug(enabled bool) {
	if enabled {
		SetLevel(slog.LevelDebug)
	} else {
		SetLevel(slog.LevelInfo)
	}
}

func IsDebug() bool { return levelVar.Level() == slog.LevelDebug }

func GetLogger() *slog.Logger { return defaultLogger }

func WithModule(module string) *slog.Logger {
	return defaultLogger.With(slog.String("module", module))
}

// Structured Logging
func Debug(msg string, args ...any) { defaultLogger.Debug(msg, args...) }
func Info(msg string, args ...any)  { defaultLogger.Info(msg, args...) }
func Warn(msg string, args ...any)  { defaultLogger.Warn(msg, args...) }
func Error(msg string, args ...any) { defaultLogger.Error(msg, args...) }

// Format-style Logging (Compatibility)
func Debugf(format string, args ...any) {
	defaultLogger.Debug(fmt.Sprintf(format, args...))
}
func Infof(format string, args ...any) {
	defaultLogger.Info(fmt.Sprintf(format, args...))
}
func Warnf(format string, args ...any) {
	defaultLogger.Warn(fmt.Sprintf(format, args...))
}
func Errf(format string, args ...any) {
	defaultLogger.Error(fmt.Sprintf(format, args...))
}
