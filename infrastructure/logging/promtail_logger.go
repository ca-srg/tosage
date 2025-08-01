package logging

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ca-srg/tosage/domain"
	"github.com/ic2hrmk/promtail"
)

type PromtailLogger struct {
	client    promtail.Client
	component string
	fields    []domain.Field
	mu        sync.RWMutex
}

func NewPromtailLogger(url, username, password, component string) (*PromtailLogger, error) {
	// Default labels for all logs
	defaultLabels := map[string]string{
		"app":       "tosage",
		"component": component,
	}

	client, err := promtail.NewJSONv1Client(
		url,
		defaultLabels,
		promtail.WithSendBatchSize(100),
		promtail.WithSendBatchTimeout(1*time.Second),
		promtail.WithBasicAuth(username, password),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create promtail client: %w", err)
	}

	return &PromtailLogger{
		client:    client,
		component: component,
		fields:    []domain.Field{},
	}, nil
}

func (p *PromtailLogger) Debug(ctx context.Context, msg string, fields ...domain.Field) {
	p.log(ctx, domain.LogLevelDebug, msg, fields...)
}

func (p *PromtailLogger) Info(ctx context.Context, msg string, fields ...domain.Field) {
	p.log(ctx, domain.LogLevelInfo, msg, fields...)
}

func (p *PromtailLogger) Warn(ctx context.Context, msg string, fields ...domain.Field) {
	p.log(ctx, domain.LogLevelWarn, msg, fields...)
}

func (p *PromtailLogger) Error(ctx context.Context, msg string, fields ...domain.Field) {
	p.log(ctx, domain.LogLevelError, msg, fields...)
}

func (p *PromtailLogger) WithFields(fields ...domain.Field) domain.Logger {
	p.mu.RLock()
	defer p.mu.RUnlock()

	newFields := make([]domain.Field, len(p.fields)+len(fields))
	copy(newFields, p.fields)
	copy(newFields[len(p.fields):], fields)

	return &PromtailLogger{
		client:    p.client,
		component: p.component,
		fields:    newFields,
	}
}

func (p *PromtailLogger) log(ctx context.Context, level domain.LogLevel, msg string, fields ...domain.Field) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	labels := map[string]string{
		"level": levelToString(level),
	}

	allFields := append(p.fields, fields...)
	for _, field := range allFields {
		labels[field.Key] = fmt.Sprintf("%v", field.Value)
	}

	// Convert domain log level to promtail log level
	var promtailLevel promtail.Level
	switch level {
	case domain.LogLevelDebug:
		promtailLevel = promtail.Debug
	case domain.LogLevelInfo:
		promtailLevel = promtail.Info
	case domain.LogLevelWarn:
		promtailLevel = promtail.Warn
	case domain.LogLevelError:
		promtailLevel = promtail.Error
	default:
		promtailLevel = promtail.Info
	}

	// Send log with labels
	p.client.LogfWithLabels(promtailLevel, labels, "%s", msg)
}

func (p *PromtailLogger) Shutdown() error {
	if p.client != nil {
		p.client.Close()
	}
	return nil
}

