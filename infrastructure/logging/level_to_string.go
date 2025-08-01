package logging

import "github.com/ca-srg/tosage/domain"

func levelToString(level domain.LogLevel) string {
	switch level {
	case domain.LogLevelDebug:
		return "DEBUG"
	case domain.LogLevelInfo:
		return "INFO"
	case domain.LogLevelWarn:
		return "WARN"
	case domain.LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}