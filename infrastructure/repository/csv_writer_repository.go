package repository

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ca-srg/tosage/domain"
	"github.com/ca-srg/tosage/domain/entity"
	"github.com/ca-srg/tosage/domain/repository"
)

// CSVWriterRepositoryImpl implements CSVWriterRepository
type CSVWriterRepositoryImpl struct {
	logger domain.Logger
}

// NewCSVWriterRepository creates a new CSV writer repository
func NewCSVWriterRepository(logger domain.Logger) repository.CSVWriterRepository {
	return &CSVWriterRepositoryImpl{
		logger: logger,
	}
}

// Write writes metric records to a CSV file
func (r *CSVWriterRepositoryImpl) Write(records []*entity.MetricRecord, outputPath string) error {
	// Validate output path
	if err := r.validateOutputPath(outputPath); err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return domain.ErrFileOperationWithCause("create directory", dir, err)
	}

	// Create file with restricted permissions
	file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return domain.ErrFileOperationWithCause("create file", outputPath, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// Log the error but don't override the main error
			r.logger.Error(context.TODO(), "Failed to close CSV file",
				domain.NewField("error", closeErr.Error()),
				domain.NewField("path", outputPath))
		}
	}()

	// Create CSV writer with UTF-8 BOM
	if _, err := file.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return domain.ErrCSVExportWithCause("write BOM", "failed to write UTF-8 BOM", err)
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header - source and project are excluded
	header := []string{"timestamp", "value", "unit"}
	// Get all unique metadata keys (excluding specified fields)
	metadataKeys := r.getUniqueMetadataKeys(records)
	header = append(header, metadataKeys...)

	if err := writer.Write(header); err != nil {
		return domain.ErrCSVExportWithCause("write header", "failed to write CSV header", err)
	}

	// Write records - source and project are excluded
	for _, record := range records {
		row := []string{
			record.Timestamp.Format(time.RFC3339),
			fmt.Sprintf("%.2f", record.Value),
			r.sanitizeCSVField(record.Unit),
		}

		// Add metadata values
		for _, key := range metadataKeys {
			value, exists := record.GetMetadata(key)
			if exists {
				row = append(row, r.sanitizeCSVField(value))
			} else {
				row = append(row, "")
			}
		}

		if err := writer.Write(row); err != nil {
			return domain.ErrCSVExportWithCause("write record", fmt.Sprintf("failed to write record at timestamp %s", record.Timestamp), err)
		}
	}

	// Check for write errors
	if err := writer.Error(); err != nil {
		return domain.ErrCSVExportWithCause("flush", "failed to flush CSV writer", err)
	}

	r.logger.Info(context.TODO(), "CSV export completed",
		domain.NewField("outputPath", outputPath),
		domain.NewField("records", len(records)))

	return nil
}

// validateOutputPath validates the output path for security
func (r *CSVWriterRepositoryImpl) validateOutputPath(path string) error {
	// Clean the path
	cleanPath := filepath.Clean(path)

	// Check for directory traversal attempts
	if strings.Contains(cleanPath, "..") {
		return domain.ErrPathTraversal(path)
	}

	// Ensure it's not an absolute path to system directories
	if filepath.IsAbs(cleanPath) {
		// Allow temp directories
		if strings.HasPrefix(cleanPath, "/tmp/") || strings.HasPrefix(cleanPath, "/var/folders/") || strings.HasPrefix(cleanPath, os.TempDir()) {
			// These are acceptable paths
		} else {
			systemDirs := []string{"/etc", "/usr", "/bin", "/sbin", "/var", "/proc", "/sys", "/dev", "/root"}
			for _, dir := range systemDirs {
				if strings.HasPrefix(cleanPath, dir) {
					return domain.ErrSystemDirectory(path)
				}
			}
		}
	}

	// Check for hidden files (starting with .)
	base := filepath.Base(cleanPath)
	if strings.HasPrefix(base, ".") && base != "." {
		return domain.ErrFileOperation("validatePath", path, "cannot write to hidden files")
	}

	// Ensure the file has .csv extension
	if filepath.Ext(cleanPath) != ".csv" {
		return domain.ErrInvalidInput("outputPath", "file must have .csv extension")
	}

	return nil
}

// sanitizeCSVField sanitizes a field to prevent CSV injection
func (r *CSVWriterRepositoryImpl) sanitizeCSVField(field string) string {
	// Remove any leading characters that could cause formula injection
	dangerousChars := []string{"=", "+", "-", "@", "\t", "\r", "|"}
	for _, char := range dangerousChars {
		if strings.HasPrefix(field, char) {
			field = "'" + field
			break
		}
	}

	// Also check for formula patterns
	dangerousPatterns := []string{"=cmd", "=DDE", "@SUM", "IMPORTXML", "WEBSERVICE"}
	fieldUpper := strings.ToUpper(field)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(fieldUpper, pattern) {
			field = "'" + field
			break
		}
	}

	// Escape quotes
	return strings.ReplaceAll(field, `"`, `""`)
}

// getUniqueMetadataKeys gets all unique metadata keys from records
func (r *CSVWriterRepositoryImpl) getUniqueMetadataKeys(records []*entity.MetricRecord) []string {
	// Define excluded metadata keys
	excludedKeys := map[string]bool{
		"cache_creation_tokens": true,
		"cache_read_tokens":     true,
		"cost":                  true,
		"currency":              true,
		"entry_count":           true,
		"input_tokens":          true,
		"output_tokens":         true,
	}

	keyMap := make(map[string]bool)
	for _, record := range records {
		if record.Metadata != nil {
			for key := range record.Metadata {
				// Skip excluded keys
				if !excludedKeys[key] {
					keyMap[key] = true
				}
			}
		}
	}

	keys := make([]string, 0, len(keyMap))
	for key := range keyMap {
		keys = append(keys, key)
	}

	// Sort keys for consistent output
	// Simple bubble sort for small number of keys
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	return keys
}
