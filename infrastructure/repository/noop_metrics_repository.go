package repository

import (
	"github.com/ca-srg/tosage/domain/repository"
)

// NoOpMetricsRepository is a no-op implementation of MetricsRepository
// Used when Prometheus is not configured
type NoOpMetricsRepository struct{}

// NewNoOpMetricsRepository creates a new no-op metrics repository
func NewNoOpMetricsRepository() repository.MetricsRepository {
	return &NoOpMetricsRepository{}
}

// SendTokenMetric does nothing
func (r *NoOpMetricsRepository) SendTokenMetric(totalTokens int, hostLabel string, metricName string) error {
	// No-op: do nothing
	return nil
}

// SendTokenMetricWithTimezone does nothing
func (r *NoOpMetricsRepository) SendTokenMetricWithTimezone(totalTokens int, hostLabel string, metricName string, timezoneInfo repository.TimezoneInfo) error {
	// No-op: do nothing
	return nil
}

// Close does nothing
func (r *NoOpMetricsRepository) Close() error {
	// No-op: do nothing
	return nil
}
