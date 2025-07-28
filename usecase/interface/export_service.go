package usecase

import (
	"context"
	"time"
)

// ExportRequest represents a request to export metrics
type ExportRequest struct {
	StartDate  *time.Time
	EndDate    *time.Time
	OutputPath string
	SingleDate *time.Time // For single date export
}

// ExportService defines the interface for exporting metrics data
type ExportService interface {
	// ExportMetricsToCSV exports metrics from Prometheus to CSV file
	ExportMetricsToCSV(ctx context.Context, req *ExportRequest) error
}
