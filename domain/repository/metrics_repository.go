package repository

// MetricsRepository defines the interface for sending metrics to external systems
type MetricsRepository interface {
	// SendTokenMetric sends the total token count metric with specified metric name
	SendTokenMetric(totalTokens int, hostLabel string, metricName string) error

	// SendTokenMetricWithTimezone sends the total token count metric with timezone information
	SendTokenMetricWithTimezone(totalTokens int, hostLabel string, metricName string, timezoneInfo TimezoneInfo) error

	// Close cleans up any resources used by the metrics repository
	Close() error
}

// MetricsRepositoryError represents errors from the metrics repository
type MetricsRepositoryError struct {
	Operation string
	Err       error
}

func (e *MetricsRepositoryError) Error() string {
	if e.Err != nil {
		return "metrics repository error in " + e.Operation + ": " + e.Err.Error()
	}
	return "metrics repository error in " + e.Operation
}

func (e *MetricsRepositoryError) Unwrap() error {
	return e.Err
}

// NewMetricsRepositoryError creates a new metrics repository error
func NewMetricsRepositoryError(operation string, err error) error {
	return &MetricsRepositoryError{
		Operation: operation,
		Err:       err,
	}
}

// Common error types
var (
	// ErrMetricsConnectionFailed is returned when connection to metrics server fails
	ErrMetricsConnectionFailed = &MetricsRepositoryError{Operation: "connect", Err: nil}

	// ErrMetricsSendFailed is returned when sending metrics fails
	ErrMetricsSendFailed = &MetricsRepositoryError{Operation: "send", Err: nil}

	// ErrMetricsTimeout is returned when metrics operation times out
	ErrMetricsTimeout = &MetricsRepositoryError{Operation: "timeout", Err: nil}
)
