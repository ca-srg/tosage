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

// ExportServiceImpl implements ExportService
type ExportServiceImpl struct {
	prometheusQueryRepo repository.PrometheusQueryRepository
	csvExportRepo       repository.CSVExportRepository
	logger              domain.Logger
}

// NewExportService creates a new export service
func NewExportService(
	prometheusQueryRepo repository.PrometheusQueryRepository,
	csvExportRepo repository.CSVExportRepository,
	logger domain.Logger,
) (usecase.ExportService, error) {
	if prometheusQueryRepo == nil {
		return nil, fmt.Errorf("prometheus query repository is required")
	}
	if csvExportRepo == nil {
		return nil, fmt.Errorf("csv export repository is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	return &ExportServiceImpl{
		prometheusQueryRepo: prometheusQueryRepo,
		csvExportRepo:       csvExportRepo,
		logger:              logger,
	}, nil
}

// ExportMetricsToCSV exports metrics from Prometheus to CSV file
func (s *ExportServiceImpl) ExportMetricsToCSV(ctx context.Context, req *usecase.ExportRequest) error {
	// Validate request
	if req == nil {
		return fmt.Errorf("export request is required")
	}

	// Determine date range
	var startDate, endDate time.Time
	if req.SingleDate != nil {
		// Single date export: from 00:00 to 23:59:59 JST
		jst, err := time.LoadLocation("Asia/Tokyo")
		if err != nil {
			return fmt.Errorf("failed to load JST timezone: %w", err)
		}

		// Convert to JST and get start of day
		dateInJST := req.SingleDate.In(jst)
		startDate = time.Date(dateInJST.Year(), dateInJST.Month(), dateInJST.Day(), 0, 0, 0, 0, jst)
		endDate = startDate.Add(24*time.Hour - time.Second)
	} else if req.StartDate != nil && req.EndDate != nil {
		// Date range export
		startDate = *req.StartDate
		endDate = *req.EndDate

		// Validate date range
		if startDate.After(endDate) {
			return fmt.Errorf("start date must be before or equal to end date")
		}
	} else {
		// Default to today
		jst, err := time.LoadLocation("Asia/Tokyo")
		if err != nil {
			return fmt.Errorf("failed to load JST timezone: %w", err)
		}

		now := time.Now().In(jst)
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, jst)
		endDate = now
	}

	s.logger.Info(ctx, "Exporting metrics",
		domain.NewField("start", startDate.Format("2006-01-02 15:04:05")),
		domain.NewField("end", endDate.Format("2006-01-02 15:04:05")),
	)

	// Determine output path
	outputPath := req.OutputPath
	if outputPath == "" {
		// Generate default filename in current directory
		exportReq := entity.NewExportRequest(startDate, endDate, "")
		outputPath = filepath.Join(".", exportReq.GenerateFilename())
	}

	// Validate output path
	if err := s.csvExportRepo.ValidateFilePath(outputPath); err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}

	// Query Claude Code metrics
	ccQuery := `tosage_cc_token`
	ccDataPoints, err := s.prometheusQueryRepo.QueryRange(ctx, ccQuery, startDate, endDate, 5*time.Minute)
	if err != nil {
		s.logger.Warn(ctx, "Failed to query Claude Code metrics", domain.NewField("error", err.Error()))
		ccDataPoints = []*entity.MetricDataPoint{} // Continue with empty data
	}

	// Query Cursor metrics
	cursorQuery := `tosage_cursor_token`
	cursorDataPoints, err := s.prometheusQueryRepo.QueryRange(ctx, cursorQuery, startDate, endDate, 5*time.Minute)
	if err != nil {
		s.logger.Warn(ctx, "Failed to query Cursor metrics", domain.NewField("error", err.Error()))
		cursorDataPoints = []*entity.MetricDataPoint{} // Continue with empty data
	}

	// Merge data points
	mergedDataPoints := s.mergeDataPoints(ccDataPoints, cursorDataPoints)

	// Check if we have any data
	if len(mergedDataPoints) == 0 {
		return fmt.Errorf("no data found for the specified time range")
	}

	// Write to CSV
	if err := s.csvExportRepo.WriteMetricsToCSV(outputPath, mergedDataPoints); err != nil {
		return fmt.Errorf("failed to write CSV: %w", err)
	}

	s.logger.Info(ctx, "Export completed successfully",
		domain.NewField("file", outputPath),
		domain.NewField("records", len(mergedDataPoints)),
	)

	return nil
}

// mergeDataPoints merges Claude Code and Cursor data points by timestamp and host
func (s *ExportServiceImpl) mergeDataPoints(ccPoints, cursorPoints []*entity.MetricDataPoint) []*entity.MetricDataPoint {
	// Create a map to merge points by timestamp and host
	mergedMap := make(map[string]*entity.MetricDataPoint)

	// Process Claude Code points
	for _, point := range ccPoints {
		key := fmt.Sprintf("%d_%s", point.Timestamp.Unix(), point.Host)
		if existing, ok := mergedMap[key]; ok {
			// Update existing point
			existing.ClaudeCodeTokens = point.ClaudeCodeTokens
			existing.TotalTokens = existing.ClaudeCodeTokens + existing.CursorTokens
		} else {
			// Create new point
			mergedMap[key] = &entity.MetricDataPoint{
				Timestamp:        point.Timestamp,
				ClaudeCodeTokens: point.ClaudeCodeTokens,
				CursorTokens:     0,
				TotalTokens:      point.ClaudeCodeTokens,
				Host:             point.Host,
				Timezone:         point.Timezone,
				TimezoneOffset:   point.TimezoneOffset,
				DetectionMethod:  point.DetectionMethod,
			}
		}
	}

	// Process Cursor points
	for _, point := range cursorPoints {
		key := fmt.Sprintf("%d_%s", point.Timestamp.Unix(), point.Host)
		if existing, ok := mergedMap[key]; ok {
			// Update existing point
			existing.CursorTokens = point.CursorTokens
			existing.TotalTokens = existing.ClaudeCodeTokens + existing.CursorTokens
		} else {
			// Create new point
			mergedMap[key] = &entity.MetricDataPoint{
				Timestamp:        point.Timestamp,
				ClaudeCodeTokens: 0,
				CursorTokens:     point.CursorTokens,
				TotalTokens:      point.CursorTokens,
				Host:             point.Host,
				Timezone:         point.Timezone,
				TimezoneOffset:   point.TimezoneOffset,
				DetectionMethod:  point.DetectionMethod,
			}
		}
	}

	// Convert map to slice
	var result []*entity.MetricDataPoint
	for _, point := range mergedMap {
		result = append(result, point)
	}

	return result
}
