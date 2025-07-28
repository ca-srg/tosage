package repository

import (
	"github.com/ca-srg/tosage/domain/entity"
)

// CSVExportRepository defines the interface for exporting data to CSV
type CSVExportRepository interface {
	// WriteMetricsToCSV writes metric data points to a CSV file
	WriteMetricsToCSV(filepath string, dataPoints []*entity.MetricDataPoint) error

	// ValidateFilePath validates that the file path is writable
	ValidateFilePath(filepath string) error
}

// CSVExportError represents an error from CSV export operations
type CSVExportError struct {
	Op  string
	Err error
}

func (e *CSVExportError) Error() string {
	return "csv export error in " + e.Op + ": " + e.Err.Error()
}

// NewCSVExportError creates a new CSV export error
func NewCSVExportError(op string, err error) error {
	return &CSVExportError{Op: op, Err: err}
}
