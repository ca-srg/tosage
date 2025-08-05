package logging

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ca-srg/tosage/domain"
)

type DebugLogger struct {
	wrapped   domain.Logger
	component string
	mu        sync.Mutex
}

func NewDebugLogger(wrapped domain.Logger, component string) *DebugLogger {
	return &DebugLogger{
		wrapped:   wrapped,
		component: component,
	}
}

func (d *DebugLogger) Debug(ctx context.Context, msg string, fields ...domain.Field) {
	d.wrapped.Debug(ctx, msg, fields...)
	d.printToStdout(domain.LogLevelDebug, msg, fields...)
}

func (d *DebugLogger) Info(ctx context.Context, msg string, fields ...domain.Field) {
	d.wrapped.Info(ctx, msg, fields...)
	d.printToStdout(domain.LogLevelInfo, msg, fields...)
}

func (d *DebugLogger) Warn(ctx context.Context, msg string, fields ...domain.Field) {
	d.wrapped.Warn(ctx, msg, fields...)
	d.printToStdout(domain.LogLevelWarn, msg, fields...)
}

func (d *DebugLogger) Error(ctx context.Context, msg string, fields ...domain.Field) {
	d.wrapped.Error(ctx, msg, fields...)
	d.printToStdout(domain.LogLevelError, msg, fields...)
}

func (d *DebugLogger) WithFields(fields ...domain.Field) domain.Logger {
	wrappedWithFields := d.wrapped.WithFields(fields...)
	return &DebugLogger{
		wrapped:   wrappedWithFields,
		component: d.component,
	}
}

func (d *DebugLogger) printToStdout(level domain.LogLevel, msg string, fields ...domain.Field) {
	// Only print debug logs when debug level is Debug
	if level == domain.LogLevelDebug {
		// Skip debug logs - they're already sent to promtail
		return
	}
	
	d.mu.Lock()
	defer d.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02T15:04:05.000Z07:00")
	levelStr := levelToString(level)

	output := fmt.Sprintf("[%s] [%s] [%s] %s", timestamp, levelStr, d.component, msg)

	if len(fields) > 0 {
		output += " {"
		for i, field := range fields {
			if i > 0 {
				output += ", "
			}
			output += fmt.Sprintf("%s=%v", field.Key, field.Value)
		}
		output += "}"
	}

	_, _ = fmt.Fprintln(os.Stdout, output)
}

func (d *DebugLogger) Shutdown() error {
	if shutdowner, ok := d.wrapped.(interface{ Shutdown() error }); ok {
		return shutdowner.Shutdown()
	}
	return nil
}
