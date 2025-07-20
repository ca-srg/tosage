package domain

import (
	"context"
)

type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

type Field struct {
	Key   string
	Value interface{}
}

type Logger interface {
	Debug(ctx context.Context, msg string, fields ...Field)
	Info(ctx context.Context, msg string, fields ...Field)
	Warn(ctx context.Context, msg string, fields ...Field)
	Error(ctx context.Context, msg string, fields ...Field)

	WithFields(fields ...Field) Logger
}

func NewField(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}
