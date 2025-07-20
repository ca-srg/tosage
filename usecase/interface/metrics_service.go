package usecase

// MetricsService defines the interface for metrics collection and reporting
type MetricsService interface {
	// StartPeriodicMetrics starts the periodic metrics collection
	StartPeriodicMetrics() error

	// StopPeriodicMetrics stops the periodic metrics collection
	StopPeriodicMetrics() error

	// SendCurrentMetrics sends the current metrics immediately
	SendCurrentMetrics() error
}

// MetricsServiceError represents an error from metrics service operations
type MetricsServiceError struct {
	Code    string
	Message string
	Details map[string]interface{}
}

func (e *MetricsServiceError) Error() string {
	return e.Message
}

// NewMetricsServiceError creates a new metrics service error
func NewMetricsServiceError(code, message string) *MetricsServiceError {
	return &MetricsServiceError{
		Code:    code,
		Message: message,
		Details: make(map[string]interface{}),
	}
}

// WithDetail adds a detail to the error
func (e *MetricsServiceError) WithDetail(key string, value interface{}) *MetricsServiceError {
	e.Details[key] = value
	return e
}
