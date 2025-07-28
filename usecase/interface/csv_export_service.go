package usecase

import (
	"time"

	"github.com/ca-srg/tosage/domain/entity"
)

// CSVExportService defines the interface for CSV export use cases
type CSVExportService interface {
	// Export exports metrics data to CSV file
	Export(options CSVExportOptions) error
}

// CSVExportOptions represents options for CSV export
type CSVExportOptions struct {
	OutputPath  string
	StartTime   *time.Time
	EndTime     *time.Time
	MetricTypes []string // claude_code, cursor, bedrock, vertex_ai
}

// MetricsDataCollector defines the interface for collecting metrics data
type MetricsDataCollector interface {
	// Collect collects metrics data from all sources
	Collect(startTime, endTime time.Time, metricTypes []string) ([]*entity.MetricRecord, error)
}
