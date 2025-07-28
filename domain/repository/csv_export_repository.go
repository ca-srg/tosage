package repository

import (
	"time"

	"github.com/ca-srg/tosage/domain/entity"
)

// CSVWriterRepository defines the interface for writing CSV files
type CSVWriterRepository interface {
	Write(records []*entity.MetricRecord, outputPath string) error
}

// MetricsDataCollectorRepository defines the interface for collecting metrics data
type MetricsDataCollectorRepository interface {
	Collect(startTime, endTime time.Time, metricTypes []string) ([]*entity.MetricRecord, error)
}

// CSVExportOptions represents options for CSV export
type CSVExportOptions struct {
	OutputPath  string
	StartTime   time.Time
	EndTime     time.Time
	MetricTypes []string // claude_code, cursor, bedrock, vertex_ai
}

// CSVExportServiceRepository defines the interface for CSV export service
type CSVExportServiceRepository interface {
	Export(options CSVExportOptions) error
}
