package logging

import (
	"context"
	"strings"

	"github.com/ca-srg/tosage/domain"
	"github.com/ca-srg/tosage/infrastructure/config"
)

type LoggerFactoryImpl struct {
	config *config.LoggingConfig
}

func NewLoggerFactory(config *config.LoggingConfig) domain.LoggerFactory {
	return &LoggerFactoryImpl{
		config: config,
	}
}

func (f *LoggerFactoryImpl) CreateLogger(component string) domain.Logger {
	promtailLogger, err := NewPromtailLogger(f.config.Promtail.URL, f.config.Promtail.Username, f.config.Promtail.Password, component)
	if err != nil {
		// Fallback to a no-op logger if promtail is not available
		return &NoOpLogger{}
	}

	// Apply log level filtering
	var logger domain.Logger = promtailLogger
	logger = NewLevelFilterLogger(logger, f.parseLogLevel(f.config.Level))

	// Wrap with debug logger if debug mode is enabled
	if f.config.Debug {
		logger = NewDebugLogger(logger, component)
	}

	return logger
}

func (f *LoggerFactoryImpl) parseLogLevel(level string) domain.LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return domain.LogLevelDebug
	case "info":
		return domain.LogLevelInfo
	case "warn":
		return domain.LogLevelWarn
	case "error":
		return domain.LogLevelError
	default:
		return domain.LogLevelInfo
	}
}

// LevelFilterLogger filters log messages based on minimum level
type LevelFilterLogger struct {
	wrapped  domain.Logger
	minLevel domain.LogLevel
}

func NewLevelFilterLogger(wrapped domain.Logger, minLevel domain.LogLevel) *LevelFilterLogger {
	return &LevelFilterLogger{
		wrapped:  wrapped,
		minLevel: minLevel,
	}
}

func (l *LevelFilterLogger) Debug(ctx context.Context, msg string, fields ...domain.Field) {
	if domain.LogLevelDebug >= l.minLevel {
		l.wrapped.Debug(ctx, msg, fields...)
	}
}

func (l *LevelFilterLogger) Info(ctx context.Context, msg string, fields ...domain.Field) {
	if domain.LogLevelInfo >= l.minLevel {
		l.wrapped.Info(ctx, msg, fields...)
	}
}

func (l *LevelFilterLogger) Warn(ctx context.Context, msg string, fields ...domain.Field) {
	if domain.LogLevelWarn >= l.minLevel {
		l.wrapped.Warn(ctx, msg, fields...)
	}
}

func (l *LevelFilterLogger) Error(ctx context.Context, msg string, fields ...domain.Field) {
	if domain.LogLevelError >= l.minLevel {
		l.wrapped.Error(ctx, msg, fields...)
	}
}

func (l *LevelFilterLogger) WithFields(fields ...domain.Field) domain.Logger {
	return &LevelFilterLogger{
		wrapped:  l.wrapped.WithFields(fields...),
		minLevel: l.minLevel,
	}
}

// NoOpLogger is a logger that does nothing
type NoOpLogger struct{}

func (n *NoOpLogger) Debug(ctx context.Context, msg string, fields ...domain.Field) {}
func (n *NoOpLogger) Info(ctx context.Context, msg string, fields ...domain.Field)  {}
func (n *NoOpLogger) Warn(ctx context.Context, msg string, fields ...domain.Field)  {}
func (n *NoOpLogger) Error(ctx context.Context, msg string, fields ...domain.Field) {}
func (n *NoOpLogger) WithFields(fields ...domain.Field) domain.Logger {
	return n
}
