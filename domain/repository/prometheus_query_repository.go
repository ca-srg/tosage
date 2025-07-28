package repository

import (
	"context"
	"time"

	"github.com/ca-srg/tosage/domain/entity"
)

// PrometheusQueryRepository defines the interface for querying Prometheus
type PrometheusQueryRepository interface {
	// QueryRange queries Prometheus for time series data within a time range
	QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) ([]*entity.MetricDataPoint, error)

	// QueryInstant queries Prometheus for instant vector at a specific time
	QueryInstant(ctx context.Context, query string, timestamp time.Time) ([]*entity.MetricDataPoint, error)
}

// PrometheusQueryError represents an error from Prometheus query operations
type PrometheusQueryError struct {
	Op  string
	Err error
}

func (e *PrometheusQueryError) Error() string {
	return "prometheus query error in " + e.Op + ": " + e.Err.Error()
}

// NewPrometheusQueryError creates a new Prometheus query error
func NewPrometheusQueryError(op string, err error) error {
	return &PrometheusQueryError{Op: op, Err: err}
}
