//go:build integration
// +build integration

package integration

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ca-srg/tosage/infrastructure/di"
	"github.com/ca-srg/tosage/usecase/impl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSVExportIntegration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory for test outputs
	tempDir, err := os.MkdirTemp("", "tosage_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create DI container
	container, err := di.NewContainer(di.WithDebugMode(true))
	require.NoError(t, err)

	// Get CSV export service
	csvExportService := container.GetCSVExportService()
	require.NotNil(t, csvExportService)

	t.Run("ExportWithDefaultOptions", func(t *testing.T) {
		outputPath := filepath.Join(tempDir, "test_default.csv")

		options, err := impl.GenerateExportOptions(outputPath, "", "", nil)
		require.NoError(t, err)

		err = csvExportService.Export(*options)
		require.NoError(t, err)

		// Verify file was created
		info, err := os.Stat(outputPath)
		require.NoError(t, err)
		assert.True(t, info.Mode().IsRegular())

		// Check file permissions (should be 0600)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

		// Read and verify content
		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)

		// Check for UTF-8 BOM
		assert.True(t, len(content) >= 3)
		assert.Equal(t, []byte{0xEF, 0xBB, 0xBF}, content[:3])

		// Check CSV header
		lines := strings.Split(string(content[3:]), "\n")
		assert.True(t, len(lines) > 0)
		assert.Contains(t, lines[0], "timestamp,value,unit")
	})

	t.Run("ExportWithTimeRange", func(t *testing.T) {
		outputPath := filepath.Join(tempDir, "test_timerange.csv")

		// Export last 7 days
		endTime := time.Now()
		startTime := endTime.AddDate(0, 0, -7)

		options, err := impl.GenerateExportOptions(
			outputPath,
			startTime.Format(time.RFC3339),
			endTime.Format(time.RFC3339),
			[]string{"claude_code"},
		)
		require.NoError(t, err)

		err = csvExportService.Export(*options)
		require.NoError(t, err)

		// Verify file was created
		_, err = os.Stat(outputPath)
		require.NoError(t, err)
	})

	t.Run("ExportSpecificMetricTypes", func(t *testing.T) {
		outputPath := filepath.Join(tempDir, "test_metrics.csv")

		options, err := impl.GenerateExportOptions(
			outputPath,
			"",
			"",
			[]string{"claude_code", "cursor"},
		)
		require.NoError(t, err)

		err = csvExportService.Export(*options)
		require.NoError(t, err)

		// Read and verify content
		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)

		// Skip BOM
		csvContent := string(content[3:])

		// Check that we have data from the specified sources
		lines := strings.Split(csvContent, "\n")
		hasClaudeCode := false
		hasCursor := false

		for _, line := range lines[1:] { // Skip header
			if strings.Contains(line, "claude_code") {
				hasClaudeCode = true
			}
			if strings.Contains(line, "cursor") {
				hasCursor = true
			}
		}

		// May or may not have data depending on test environment
		t.Logf("Has Claude Code data: %v", hasClaudeCode)
		t.Logf("Has Cursor data: %v", hasCursor)
	})

	t.Run("SecurityValidations", func(t *testing.T) {
		testCases := []struct {
			name        string
			outputPath  string
			expectError string
		}{
			{
				name:        "DirectoryTraversal",
				outputPath:  "../../../etc/passwd.csv",
				expectError: "path contains directory traversal",
			},
			{
				name:        "SystemDirectory",
				outputPath:  "/etc/test.csv",
				expectError: "cannot write to system directory",
			},
			{
				name:        "HiddenFile",
				outputPath:  filepath.Join(tempDir, ".hidden.csv"),
				expectError: "cannot write to hidden files",
			},
			{
				name:        "InvalidExtension",
				outputPath:  filepath.Join(tempDir, "test.txt"),
				expectError: "file must have .csv extension",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				options, err := impl.GenerateExportOptions(tc.outputPath, "", "", nil)

				// Some validations happen during option generation
				if err != nil {
					assert.Contains(t, err.Error(), tc.expectError)
					return
				}

				// Others happen during export
				err = csvExportService.Export(*options)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectError)
			})
		}
	})

	t.Run("InvalidTimeFormat", func(t *testing.T) {
		outputPath := filepath.Join(tempDir, "test_invalid_time.csv")

		_, err := impl.GenerateExportOptions(
			outputPath,
			"invalid-time-format",
			"",
			nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid start time")
	})

	t.Run("InvalidMetricType", func(t *testing.T) {
		outputPath := filepath.Join(tempDir, "test_invalid_metric.csv")

		options, err := impl.GenerateExportOptions(
			outputPath,
			"",
			"",
			[]string{"invalid_metric"},
		)
		require.NoError(t, err)

		err = csvExportService.Export(*options)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid metric type")
	})
}

func TestMain(m *testing.M) {
	// Parse flags to support -short flag
	flag.Parse()

	// Run tests
	code := m.Run()

	// Clean up
	os.Exit(code)
}
