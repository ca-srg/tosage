package impl

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ca-srg/tosage/domain"
	"github.com/ca-srg/tosage/domain/entity"
	usecase "github.com/ca-srg/tosage/usecase/interface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock implementations
type MockCSVExportLogger struct{}

func (m *MockCSVExportLogger) Debug(ctx context.Context, msg string, fields ...domain.Field) {}
func (m *MockCSVExportLogger) Info(ctx context.Context, msg string, fields ...domain.Field)  {}
func (m *MockCSVExportLogger) Warn(ctx context.Context, msg string, fields ...domain.Field)  {}
func (m *MockCSVExportLogger) Error(ctx context.Context, msg string, fields ...domain.Field) {}
func (m *MockCSVExportLogger) WithFields(fields ...domain.Field) domain.Logger {
	return m
}

type MockMetricsDataCollector struct {
	mock.Mock
}

func (m *MockMetricsDataCollector) Collect(startTime, endTime time.Time, metricTypes []string) ([]*entity.MetricRecord, error) {
	args := m.Called(startTime, endTime, metricTypes)
	if result := args.Get(0); result != nil {
		return result.([]*entity.MetricRecord), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockCSVWriter struct {
	mock.Mock
}

func (m *MockCSVWriter) Write(records []*entity.MetricRecord, outputPath string) error {
	args := m.Called(records, outputPath)
	return args.Error(0)
}

func TestCSVExportService_Export_Success(t *testing.T) {
	mockCollector := new(MockMetricsDataCollector)
	mockWriter := new(MockCSVWriter)
	logger := &MockCSVExportLogger{}

	service := NewCSVExportService(mockCollector, mockWriter, logger)

	// Test data
	startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
	outputPath := "/tmp/test_metrics.csv"

	records := []*entity.MetricRecord{
		{
			Timestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			Source:    "claude_code",
			Project:   "test-project",
			Value:     1000.0,
			Unit:      "tokens",
		},
		{
			Timestamp: time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC),
			Source:    "cursor",
			Project:   "test-project",
			Value:     500.0,
			Unit:      "tokens",
		},
	}

	// Mock expectations
	mockCollector.On("Collect", startTime, endTime, []string{"claude_code", "cursor"}).
		Return(records, nil)
	mockWriter.On("Write", mock.AnythingOfType("[]*entity.MetricRecord"), outputPath).
		Return(nil)

	// Execute
	options := usecase.CSVExportOptions{
		OutputPath:  outputPath,
		StartTime:   &startTime,
		EndTime:     &endTime,
		MetricTypes: []string{"claude_code", "cursor"},
	}

	err := service.Export(options)

	// Verify
	require.NoError(t, err)
	mockCollector.AssertExpectations(t)
	mockWriter.AssertExpectations(t)

	// Verify records are sorted by timestamp
	writtenRecords := mockWriter.Calls[0].Arguments.Get(0).([]*entity.MetricRecord)
	assert.Len(t, writtenRecords, 2)
	assert.True(t, writtenRecords[0].Timestamp.Before(writtenRecords[1].Timestamp) ||
		writtenRecords[0].Timestamp.Equal(writtenRecords[1].Timestamp))
}

func TestCSVExportService_Export_DefaultValues(t *testing.T) {
	mockCollector := new(MockMetricsDataCollector)
	mockWriter := new(MockCSVWriter)
	logger := &MockCSVExportLogger{}

	service := NewCSVExportService(mockCollector, mockWriter, logger)

	// Mock expectations with any time values
	mockCollector.On("Collect", mock.Anything, mock.Anything, mock.Anything).
		Return([]*entity.MetricRecord{}, nil)
	mockWriter.On("Write", mock.AnythingOfType("[]*entity.MetricRecord"), mock.AnythingOfType("string")).
		Return(nil)

	// Execute with no optional values
	options := usecase.CSVExportOptions{}
	err := service.Export(options)

	// Verify
	require.NoError(t, err)
	mockCollector.AssertExpectations(t)
	mockWriter.AssertExpectations(t)

	// Verify default output path format
	outputPath := mockWriter.Calls[0].Arguments.Get(1).(string)
	assert.Regexp(t, `^metrics_\d{8}_\d{6}\.csv$`, outputPath)

	// Verify default time range (30 days)
	collectCall := mockCollector.Calls[0]
	startTime := collectCall.Arguments.Get(0).(time.Time)
	endTime := collectCall.Arguments.Get(1).(time.Time)
	duration := endTime.Sub(startTime)
	assert.InDelta(t, 30*24*time.Hour, duration, float64(25*time.Hour)) // Allow some flexibility
}

func TestCSVExportService_Export_InvalidTimeRange(t *testing.T) {
	mockCollector := new(MockMetricsDataCollector)
	mockWriter := new(MockCSVWriter)
	logger := &MockCSVExportLogger{}

	service := NewCSVExportService(mockCollector, mockWriter, logger)

	// End time before start time
	startTime := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	options := usecase.CSVExportOptions{
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	err := service.Export(options)

	// Verify
	require.Error(t, err)
	assert.True(t, domain.IsErrorCode(err, domain.ErrCodeInvalidInput))
	mockCollector.AssertNotCalled(t, "Collect", mock.Anything, mock.Anything, mock.Anything)
	mockWriter.AssertNotCalled(t, "Write", mock.Anything, mock.Anything)
}

func TestCSVExportService_Export_CollectorError(t *testing.T) {
	mockCollector := new(MockMetricsDataCollector)
	mockWriter := new(MockCSVWriter)
	logger := &MockCSVExportLogger{}

	service := NewCSVExportService(mockCollector, mockWriter, logger)

	// Mock collector error
	mockCollector.On("Collect", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("collection failed"))

	options := usecase.CSVExportOptions{
		OutputPath: "/tmp/test.csv",
	}

	err := service.Export(options)

	// Verify
	require.Error(t, err)
	assert.True(t, domain.IsErrorCode(err, domain.ErrCodeCSVExport))
	mockCollector.AssertExpectations(t)
	mockWriter.AssertNotCalled(t, "Write", mock.Anything, mock.Anything)
}

func TestCSVExportService_Export_WriterError(t *testing.T) {
	mockCollector := new(MockMetricsDataCollector)
	mockWriter := new(MockCSVWriter)
	logger := &MockCSVExportLogger{}

	service := NewCSVExportService(mockCollector, mockWriter, logger)

	records := []*entity.MetricRecord{
		{
			Timestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			Source:    "test",
			Project:   "test",
			Value:     100.0,
			Unit:      "tokens",
		},
	}

	// Mock expectations
	mockCollector.On("Collect", mock.Anything, mock.Anything, mock.Anything).
		Return(records, nil)
	mockWriter.On("Write", mock.AnythingOfType("[]*entity.MetricRecord"), mock.Anything).
		Return(errors.New("write failed"))

	options := usecase.CSVExportOptions{
		OutputPath: "/tmp/test.csv",
	}

	err := service.Export(options)

	// Verify
	require.Error(t, err)
	assert.True(t, domain.IsErrorCode(err, domain.ErrCodeCSVExport))
	mockCollector.AssertExpectations(t)
	mockWriter.AssertExpectations(t)
}

func TestCSVExportService_Export_NoData(t *testing.T) {
	mockCollector := new(MockMetricsDataCollector)
	mockWriter := new(MockCSVWriter)
	logger := &MockCSVExportLogger{}

	service := NewCSVExportService(mockCollector, mockWriter, logger)

	// Mock empty data
	mockCollector.On("Collect", mock.Anything, mock.Anything, mock.Anything).
		Return([]*entity.MetricRecord{}, nil)
	mockWriter.On("Write", mock.AnythingOfType("[]*entity.MetricRecord"), mock.Anything).
		Return(nil)

	options := usecase.CSVExportOptions{
		OutputPath: "/tmp/test.csv",
	}

	err := service.Export(options)

	// Verify - should succeed even with no data
	require.NoError(t, err)
	mockCollector.AssertExpectations(t)
	mockWriter.AssertExpectations(t)
}

func TestCSVExportService_Export_SortRecords(t *testing.T) {
	mockCollector := new(MockMetricsDataCollector)
	mockWriter := new(MockCSVWriter)
	logger := &MockCSVExportLogger{}

	service := NewCSVExportService(mockCollector, mockWriter, logger)

	// Create unsorted records
	records := []*entity.MetricRecord{
		{
			Timestamp: time.Date(2024, 1, 3, 10, 0, 0, 0, time.UTC),
			Source:    "test",
			Value:     300.0,
		},
		{
			Timestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			Source:    "test",
			Value:     100.0,
		},
		{
			Timestamp: time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC),
			Source:    "test",
			Value:     200.0,
		},
	}

	// Mock expectations
	mockCollector.On("Collect", mock.Anything, mock.Anything, mock.Anything).
		Return(records, nil)

	var capturedRecords []*entity.MetricRecord
	mockWriter.On("Write", mock.AnythingOfType("[]*entity.MetricRecord"), mock.Anything).
		Run(func(args mock.Arguments) {
			capturedRecords = args.Get(0).([]*entity.MetricRecord)
		}).
		Return(nil)

	options := usecase.CSVExportOptions{
		OutputPath: "/tmp/test.csv",
	}

	err := service.Export(options)

	// Verify
	require.NoError(t, err)
	require.Len(t, capturedRecords, 3)

	// Verify records are sorted by timestamp
	assert.Equal(t, float64(100), capturedRecords[0].Value)
	assert.Equal(t, float64(200), capturedRecords[1].Value)
	assert.Equal(t, float64(300), capturedRecords[2].Value)

	mockCollector.AssertExpectations(t)
	mockWriter.AssertExpectations(t)
}

func TestGenerateExportOptions_Success(t *testing.T) {
	tests := []struct {
		name         string
		outputPath   string
		startTimeStr string
		endTimeStr   string
		metricTypes  []string
		expectError  bool
	}{
		{
			name:         "all fields valid",
			outputPath:   "/tmp/metrics.csv",
			startTimeStr: "2024-01-01T00:00:00Z",
			endTimeStr:   "2024-01-31T23:59:59Z",
			metricTypes:  []string{"claude_code", "cursor"},
			expectError:  false,
		},
		{
			name:         "date only format",
			outputPath:   "/tmp/metrics.csv",
			startTimeStr: "2024-01-01",
			endTimeStr:   "2024-01-31",
			metricTypes:  []string{"bedrock"},
			expectError:  false,
		},
		{
			name:         "empty fields",
			outputPath:   "",
			startTimeStr: "",
			endTimeStr:   "",
			metricTypes:  nil,
			expectError:  false,
		},
		{
			name:         "invalid output extension",
			outputPath:   "/tmp/metrics.txt",
			startTimeStr: "",
			endTimeStr:   "",
			metricTypes:  nil,
			expectError:  true,
		},
		{
			name:         "invalid start time",
			outputPath:   "/tmp/metrics.csv",
			startTimeStr: "invalid-time",
			endTimeStr:   "",
			metricTypes:  nil,
			expectError:  true,
		},
		{
			name:         "invalid end time",
			outputPath:   "/tmp/metrics.csv",
			startTimeStr: "",
			endTimeStr:   "invalid-time",
			metricTypes:  nil,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options, err := GenerateExportOptions(tt.outputPath, tt.startTimeStr, tt.endTimeStr, tt.metricTypes)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, options)
			} else {
				require.NoError(t, err)
				require.NotNil(t, options)

				assert.Equal(t, tt.outputPath, options.OutputPath)
				assert.Equal(t, tt.metricTypes, options.MetricTypes)

				if tt.startTimeStr != "" {
					assert.NotNil(t, options.StartTime)
				}
				if tt.endTimeStr != "" {
					assert.NotNil(t, options.EndTime)
				}
			}
		})
	}
}

func TestParseTimeString_Formats(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		hasError bool
	}{
		{"2024-01-01T15:04:05Z", "2024-01-01T15:04:05Z", false},
		{"2024-01-01T15:04:05", "2024-01-01T15:04:05Z", false},
		{"2024-01-01 15:04:05", "2024-01-01T15:04:05Z", false},
		{"2024-01-01", "2024-01-01T00:00:00Z", false},
		{"2024/01/01", "2024-01-01T00:00:00Z", false},
		{"2024-01-01T15:04:05+09:00", "2024-01-01T06:04:05Z", false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseTimeString(tt.input)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result.UTC().Format(time.RFC3339))
			}
		})
	}
}
