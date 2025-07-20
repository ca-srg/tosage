package logging

import (
	"context"
	"testing"

	"github.com/ca-srg/tosage/domain"
)

func TestPromtailLogger_LogMethods(t *testing.T) {
	// Note: This test requires a running promtail instance for integration testing
	// For unit testing, we would typically mock the promtail client

	tests := []struct {
		name    string
		logFunc func(logger domain.Logger, ctx context.Context, msg string, fields ...domain.Field)
		level   string
		message string
		fields  []domain.Field
	}{
		{
			name: "Debug log",
			logFunc: func(logger domain.Logger, ctx context.Context, msg string, fields ...domain.Field) {
				logger.Debug(ctx, msg, fields...)
			},
			level:   "DEBUG",
			message: "Debug message",
			fields: []domain.Field{
				domain.NewField("user", "test"),
				domain.NewField("action", "debug_test"),
			},
		},
		{
			name: "Info log",
			logFunc: func(logger domain.Logger, ctx context.Context, msg string, fields ...domain.Field) {
				logger.Info(ctx, msg, fields...)
			},
			level:   "INFO",
			message: "Info message",
			fields: []domain.Field{
				domain.NewField("status", "success"),
			},
		},
		{
			name: "Warn log",
			logFunc: func(logger domain.Logger, ctx context.Context, msg string, fields ...domain.Field) {
				logger.Warn(ctx, msg, fields...)
			},
			level:   "WARN",
			message: "Warning message",
			fields: []domain.Field{
				domain.NewField("threshold", 80),
			},
		},
		{
			name: "Error log",
			logFunc: func(logger domain.Logger, ctx context.Context, msg string, fields ...domain.Field) {
				logger.Error(ctx, msg, fields...)
			},
			level:   "ERROR",
			message: "Error message",
			fields: []domain.Field{
				domain.NewField("error", "test error"),
			},
		},
	}

	// Skip if promtail is not available
	logger, err := NewPromtailLogger("http://localhost:3100", "", "", "test-component")
	if err != nil {
		t.Skip("Promtail not available, skipping integration test")
	}
	defer func() {
		if err := logger.Shutdown(); err != nil {
			t.Logf("Failed to shutdown logger: %v", err)
		}
	}()

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test verifies that the methods don't panic
			// Actual log delivery verification would require a promtail mock
			tt.logFunc(logger, ctx, tt.message, tt.fields...)
		})
	}
}

func TestPromtailLogger_WithFields(t *testing.T) {
	// Create a logger with fallback to stderr (promtail not required)
	logger := &PromtailLogger{
		component: "test",
		fields:    []domain.Field{},
	}

	// Add base fields
	baseLogger := logger.WithFields(
		domain.NewField("app", "tosage"),
		domain.NewField("version", "1.0.0"),
	)

	// Add more fields
	childLogger := baseLogger.WithFields(
		domain.NewField("module", "test"),
	)

	// Verify that WithFields returns a new logger instance
	if logger == baseLogger {
		t.Error("WithFields should return a new logger instance")
	}

	// Verify that fields are accumulated
	childLoggerImpl := childLogger.(*PromtailLogger)
	if len(childLoggerImpl.fields) != 3 {
		t.Errorf("Expected 3 fields, got %d", len(childLoggerImpl.fields))
	}
}

func TestLevelToString(t *testing.T) {
	tests := []struct {
		level    domain.LogLevel
		expected string
	}{
		{domain.LogLevelDebug, "DEBUG"},
		{domain.LogLevelInfo, "INFO"},
		{domain.LogLevelWarn, "WARN"},
		{domain.LogLevelError, "ERROR"},
		{domain.LogLevel(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := levelToString(tt.level)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
