package impl

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/ca-srg/tosage/domain"
	"github.com/ca-srg/tosage/domain/entity"
	"github.com/ca-srg/tosage/domain/repository"
	usecase "github.com/ca-srg/tosage/usecase/interface"
)

// CSVExportServiceImpl implements CSVExportService
type CSVExportServiceImpl struct {
	metricsCollector usecase.MetricsDataCollector
	csvWriter        repository.CSVWriterRepository
	logger           domain.Logger
}

// NewCSVExportService creates a new CSV export service
func NewCSVExportService(
	metricsCollector usecase.MetricsDataCollector,
	csvWriter repository.CSVWriterRepository,
	logger domain.Logger,
) usecase.CSVExportService {
	return &CSVExportServiceImpl{
		metricsCollector: metricsCollector,
		csvWriter:        csvWriter,
		logger:           logger,
	}
}

// Export exports metrics data to CSV file
func (s *CSVExportServiceImpl) Export(options usecase.CSVExportOptions) error {
	s.logger.Info(context.TODO(), "Starting CSV export",
		domain.NewField("outputPath", options.OutputPath),
		domain.NewField("startTime", options.StartTime),
		domain.NewField("endTime", options.EndTime),
		domain.NewField("metricTypes", options.MetricTypes))

	// Validate options
	if err := s.validateOptions(options); err != nil {
		return err
	}

	// Set default values
	now := time.Now()
	startTime := s.getStartTime(options.StartTime, now)
	endTime := s.getEndTime(options.EndTime, now)
	outputPath := s.getOutputPath(options.OutputPath, now)

	// Validate time range
	if endTime.Before(startTime) {
		return domain.ErrInvalidInput("time range", "end time must be after start time")
	}

	// Collect metrics data
	records, err := s.metricsCollector.Collect(startTime, endTime, options.MetricTypes)
	if err != nil {
		return domain.ErrCSVExportWithCause("collect metrics", "failed to collect metrics data", err)
	}

	if len(records) == 0 {
		s.logger.Warn(context.TODO(), "No metrics data found for the specified criteria",
			domain.NewField("startTime", startTime),
			domain.NewField("endTime", endTime),
			domain.NewField("metricTypes", options.MetricTypes))
	}

	// Sort records by timestamp
	s.sortRecordsByTimestamp(records)

	// Write to CSV
	if err := s.csvWriter.Write(records, outputPath); err != nil {
		return domain.ErrCSVExportWithCause("write CSV", "failed to write CSV file", err)
	}

	s.logger.Info(context.TODO(), "CSV export completed successfully",
		domain.NewField("outputPath", outputPath),
		domain.NewField("recordCount", len(records)),
		domain.NewField("startTime", startTime),
		domain.NewField("endTime", endTime),
		domain.NewField("metricTypes", options.MetricTypes))

	return nil
}

// validateOptions validates export options
func (s *CSVExportServiceImpl) validateOptions(options usecase.CSVExportOptions) error {
	// Output path validation is done in csvWriter
	// Time validation is done after default values are set
	// Metric types validation is done in metricsCollector
	return nil
}

// getStartTime returns start time with defaults
func (s *CSVExportServiceImpl) getStartTime(optionTime *time.Time, now time.Time) time.Time {
	if optionTime != nil {
		return *optionTime
	}
	// Default: 30 days ago from start of day
	return now.AddDate(0, 0, -30).Truncate(24 * time.Hour)
}

// getEndTime returns end time with defaults
func (s *CSVExportServiceImpl) getEndTime(optionTime *time.Time, now time.Time) time.Time {
	if optionTime != nil {
		return *optionTime
	}
	// Default: end of current day
	return now.Truncate(24 * time.Hour).Add(24*time.Hour - time.Nanosecond)
}

// getOutputPath returns output path with defaults
func (s *CSVExportServiceImpl) getOutputPath(optionPath string, now time.Time) string {
	if optionPath != "" {
		return optionPath
	}
	// Default: metrics_YYYYMMDD_HHMMSS.csv in current directory
	return fmt.Sprintf("metrics_%s.csv", now.Format("20060102_150405"))
}

// sortRecordsByTimestamp sorts records by timestamp
func (s *CSVExportServiceImpl) sortRecordsByTimestamp(records []*entity.MetricRecord) {
	// Simple bubble sort for small datasets
	// For larger datasets, could use sort.Slice
	n := len(records)
	for i := 0; i < n; i++ {
		for j := 0; j < n-i-1; j++ {
			if records[j].Timestamp.After(records[j+1].Timestamp) {
				records[j], records[j+1] = records[j+1], records[j]
			}
		}
	}
}

// GenerateExportOptions creates export options with validation
func GenerateExportOptions(outputPath string, startTimeStr, endTimeStr string, metricTypes []string) (*usecase.CSVExportOptions, error) {
	options := &usecase.CSVExportOptions{
		OutputPath:  outputPath,
		MetricTypes: metricTypes,
	}

	// Parse start time if provided
	if startTimeStr != "" {
		startTime, err := parseTimeString(startTimeStr)
		if err != nil {
			return nil, domain.ErrInvalidInput("start time", fmt.Sprintf("invalid time format: %v", err))
		}
		options.StartTime = &startTime
	}

	// Parse end time if provided
	if endTimeStr != "" {
		endTime, err := parseTimeString(endTimeStr)
		if err != nil {
			return nil, domain.ErrInvalidInput("end time", fmt.Sprintf("invalid time format: %v", err))
		}
		options.EndTime = &endTime
	}

	// Validate output path extension
	if outputPath != "" && filepath.Ext(outputPath) != ".csv" {
		return nil, domain.ErrInvalidInput("output path", "file must have .csv extension")
	}

	return options, nil
}

// parseTimeString parses time string in various formats
func parseTimeString(timeStr string) (time.Time, error) {
	// Try ISO 8601 formats
	formats := []string{
		time.RFC3339,                // 2006-01-02T15:04:05Z07:00
		"2006-01-02T15:04:05",       // 2006-01-02T15:04:05
		"2006-01-02 15:04:05",       // 2006-01-02 15:04:05
		"2006-01-02",                // 2006-01-02
		"2006/01/02",                // 2006/01/02
		"2006-01-02T15:04:05Z",      // 2006-01-02T15:04:05Z
		"2006-01-02T15:04:05+09:00", // 2006-01-02T15:04:05+09:00
	}

	var lastErr error
	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		} else {
			lastErr = err
		}
	}

	return time.Time{}, domain.ErrInvalidInput("time string", fmt.Sprintf("unable to parse '%s': %v", timeStr, lastErr))
}
