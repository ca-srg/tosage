package logging

import (
	"context"
	"testing"

	"github.com/ca-srg/tosage/domain"
)

// MockLogger is a test logger that tracks method calls
type MockLogger struct {
	debugCalls []string
	infoCalls  []string
	warnCalls  []string
	errorCalls []string
	fields     []domain.Field
}

func (m *MockLogger) Debug(ctx context.Context, msg string, fields ...domain.Field) {
	m.debugCalls = append(m.debugCalls, msg)
}

func (m *MockLogger) Info(ctx context.Context, msg string, fields ...domain.Field) {
	m.infoCalls = append(m.infoCalls, msg)
}

func (m *MockLogger) Warn(ctx context.Context, msg string, fields ...domain.Field) {
	m.warnCalls = append(m.warnCalls, msg)
}

func (m *MockLogger) Error(ctx context.Context, msg string, fields ...domain.Field) {
	m.errorCalls = append(m.errorCalls, msg)
}

func (m *MockLogger) WithFields(fields ...domain.Field) domain.Logger {
	newMock := &MockLogger{
		fields: append(m.fields, fields...),
	}
	return newMock
}

func TestDebugLogger_LogMethods(t *testing.T) {
	mockLogger := &MockLogger{}
	debugLogger := NewDebugLogger(mockLogger, "test-component")
	ctx := context.Background()

	tests := []struct {
		name      string
		logFunc   func()
		checkFunc func() bool
	}{
		{
			name: "Debug logging",
			logFunc: func() {
				debugLogger.Debug(ctx, "debug message")
			},
			checkFunc: func() bool {
				return len(mockLogger.debugCalls) == 1 && mockLogger.debugCalls[0] == "debug message"
			},
		},
		{
			name: "Info logging",
			logFunc: func() {
				debugLogger.Info(ctx, "info message")
			},
			checkFunc: func() bool {
				return len(mockLogger.infoCalls) == 1 && mockLogger.infoCalls[0] == "info message"
			},
		},
		{
			name: "Warn logging",
			logFunc: func() {
				debugLogger.Warn(ctx, "warn message")
			},
			checkFunc: func() bool {
				return len(mockLogger.warnCalls) == 1 && mockLogger.warnCalls[0] == "warn message"
			},
		},
		{
			name: "Error logging",
			logFunc: func() {
				debugLogger.Error(ctx, "error message")
			},
			checkFunc: func() bool {
				return len(mockLogger.errorCalls) == 1 && mockLogger.errorCalls[0] == "error message"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.logFunc()
			if !tt.checkFunc() {
				t.Errorf("Log method was not called correctly")
			}
		})
	}
}

func TestDebugLogger_WithFields(t *testing.T) {
	mockLogger := &MockLogger{}
	debugLogger := NewDebugLogger(mockLogger, "test-component")

	field1 := domain.NewField("key1", "value1")
	field2 := domain.NewField("key2", "value2")

	newLogger := debugLogger.WithFields(field1, field2)

	// Verify that WithFields returns a new DebugLogger instance
	if newLogger == debugLogger {
		t.Error("WithFields should return a new logger instance")
	}

	// Verify that the new logger is still a DebugLogger
	if _, ok := newLogger.(*DebugLogger); !ok {
		t.Error("WithFields should return a DebugLogger instance")
	}
}

func TestDebugLogger_Shutdown(t *testing.T) {
	// Test with a logger that implements Shutdown
	promtailLogger := &PromtailLogger{
		component: "test",
	}
	debugLogger := NewDebugLogger(promtailLogger, "test-component")

	err := debugLogger.Shutdown()
	if err != nil {
		t.Errorf("Shutdown should not return error: %v", err)
	}

	// Test with a logger that doesn't implement Shutdown
	mockLogger := &MockLogger{}
	debugLogger2 := NewDebugLogger(mockLogger, "test-component")

	err2 := debugLogger2.Shutdown()
	if err2 != nil {
		t.Errorf("Shutdown should not return error for logger without Shutdown: %v", err2)
	}
}
