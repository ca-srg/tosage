package entity

import (
	"time"
)

// ExportRequest represents a request to export metrics data
type ExportRequest struct {
	StartDate    time.Time
	EndDate      time.Time
	OutputPath   string
	IncludeHosts []string // Empty means all hosts
}

// NewExportRequest creates a new export request
func NewExportRequest(startDate, endDate time.Time, outputPath string) *ExportRequest {
	return &ExportRequest{
		StartDate:    startDate,
		EndDate:      endDate,
		OutputPath:   outputPath,
		IncludeHosts: []string{},
	}
}

// WithHosts sets the hosts to include in the export
func (e *ExportRequest) WithHosts(hosts []string) *ExportRequest {
	e.IncludeHosts = hosts
	return e
}

// GetDateRange returns a formatted date range string
func (e *ExportRequest) GetDateRange() string {
	if e.StartDate.Format("2006-01-02") == e.EndDate.Format("2006-01-02") {
		return e.StartDate.Format("2006-01-02")
	}
	return e.StartDate.Format("2006-01-02") + "_to_" + e.EndDate.Format("2006-01-02")
}

// GenerateFilename generates a filename for the export
func (e *ExportRequest) GenerateFilename() string {
	timestamp := time.Now().Format("20060102_150405")
	dateRange := e.GetDateRange()
	return "tosage_metrics_" + dateRange + "_" + timestamp + ".csv"
}
