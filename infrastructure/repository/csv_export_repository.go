package repository

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/ca-srg/tosage/domain/entity"
	"github.com/ca-srg/tosage/domain/repository"
)

// CSVExportRepository implements repository.CSVExportRepository
type CSVExportRepository struct{}

// NewCSVExportRepository creates a new CSV export repository
func NewCSVExportRepository() repository.CSVExportRepository {
	return &CSVExportRepository{}
}

// WriteMetricsToCSV writes metric data points to a CSV file
func (r *CSVExportRepository) WriteMetricsToCSV(filepath string, dataPoints []*entity.MetricDataPoint) error {
	// Validate input
	if len(dataPoints) == 0 {
		return repository.NewCSVExportError("write", fmt.Errorf("no data points to export"))
	}

	// Sort data points by timestamp and host
	sort.Slice(dataPoints, func(i, j int) bool {
		if dataPoints[i].Timestamp.Equal(dataPoints[j].Timestamp) {
			return dataPoints[i].Host < dataPoints[j].Host
		}
		return dataPoints[i].Timestamp.Before(dataPoints[j].Timestamp)
	})

	// Create CSV file
	file, err := os.Create(filepath)
	if err != nil {
		return repository.NewCSVExportError("write", fmt.Errorf("failed to create file: %w", err))
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Log error but don't fail the operation
			fmt.Fprintf(os.Stderr, "Warning: failed to close file: %v\n", err)
		}
	}()

	// Create CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	headers := []string{
		"timestamp",
		"claude_code_tokens",
		"cursor_tokens",
		"total_tokens",
		"host",
		"timezone",
		"timezone_offset",
		"detection_method",
	}
	if err := writer.Write(headers); err != nil {
		return repository.NewCSVExportError("write", fmt.Errorf("failed to write header: %w", err))
	}

	// Write data rows
	for _, dp := range dataPoints {
		row := []string{
			dp.Timestamp.Format("2006-01-02 15:04:05"),
			strconv.Itoa(dp.ClaudeCodeTokens),
			strconv.Itoa(dp.CursorTokens),
			strconv.Itoa(dp.TotalTokens),
			dp.Host,
			dp.Timezone,
			dp.TimezoneOffset,
			dp.DetectionMethod,
		}
		if err := writer.Write(row); err != nil {
			return repository.NewCSVExportError("write", fmt.Errorf("failed to write row: %w", err))
		}
	}

	return nil
}

// ValidateFilePath validates that the file path is writable
func (r *CSVExportRepository) ValidateFilePath(path string) error {
	// Get directory path
	dir := filepath.Dir(path)

	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return repository.NewCSVExportError("validate", fmt.Errorf("directory does not exist: %s", dir))
		}
		return repository.NewCSVExportError("validate", fmt.Errorf("failed to check directory: %w", err))
	}

	// Check if it's a directory
	if !info.IsDir() {
		return repository.NewCSVExportError("validate", fmt.Errorf("path is not a directory: %s", dir))
	}

	// Check if directory is writable by creating a temp file
	testFile := filepath.Join(dir, ".tosage_test_write")
	file, err := os.Create(testFile)
	if err != nil {
		return repository.NewCSVExportError("validate", fmt.Errorf("directory is not writable: %s", dir))
	}
	if err := file.Close(); err != nil {
		// Try to remove anyway
		_ = os.Remove(testFile)
		return repository.NewCSVExportError("validate", fmt.Errorf("failed to close test file: %w", err))
	}
	if err := os.Remove(testFile); err != nil {
		// Non-critical error, just log it
		fmt.Fprintf(os.Stderr, "Warning: failed to remove test file: %v\n", err)
	}

	// Check file extension
	ext := filepath.Ext(path)
	if ext != ".csv" {
		return repository.NewCSVExportError("validate", fmt.Errorf("file must have .csv extension, got: %s", ext))
	}

	return nil
}
