package impl

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ca-srg/tosage/domain"
	"github.com/ca-srg/tosage/domain/entity"
	usecase "github.com/ca-srg/tosage/usecase/interface"
)

// MetricsDataCollectorImpl implements MetricsDataCollector
type MetricsDataCollectorImpl struct {
	ccService       usecase.CcService
	cursorService   usecase.CursorService
	bedrockService  usecase.BedrockService
	vertexAIService usecase.VertexAIService
	logger          domain.Logger
}

// NewMetricsDataCollector creates a new MetricsDataCollector
func NewMetricsDataCollector(
	ccService usecase.CcService,
	cursorService usecase.CursorService,
	bedrockService usecase.BedrockService,
	vertexAIService usecase.VertexAIService,
	logger domain.Logger,
) usecase.MetricsDataCollector {
	return &MetricsDataCollectorImpl{
		ccService:       ccService,
		cursorService:   cursorService,
		bedrockService:  bedrockService,
		vertexAIService: vertexAIService,
		logger:          logger,
	}
}

// Collect collects metrics data from all sources
func (c *MetricsDataCollectorImpl) Collect(startTime, endTime time.Time, metricTypes []string) ([]*entity.MetricRecord, error) {
	c.logger.Info(context.TODO(), "Starting metrics collection",
		domain.NewField("startTime", startTime),
		domain.NewField("endTime", endTime),
		domain.NewField("metricTypes", metricTypes))

	// Validate metric types
	validTypes := map[string]bool{
		"claude_code": true,
		"cursor":      true,
		"bedrock":     true,
		"vertex_ai":   true,
		"all":         true,
	}

	typesToCollect := make(map[string]bool)
	if len(metricTypes) == 0 {
		// Collect all types if none specified
		for t := range validTypes {
			if t != "all" {
				typesToCollect[t] = true
			}
		}
	} else {
		for _, t := range metricTypes {
			if !validTypes[t] {
				return nil, domain.ErrInvalidInput("metricTypes", fmt.Sprintf("invalid metric type: %s", t))
			}
			if t == "all" {
				// Collect all types if "all" is specified
				for validType := range validTypes {
					if validType != "all" {
						typesToCollect[validType] = true
					}
				}
			} else {
				typesToCollect[t] = true
			}
		}
	}

	// Collect metrics concurrently
	var wg sync.WaitGroup
	results := make(chan []*entity.MetricRecord, len(typesToCollect))
	errors := make(chan error, len(typesToCollect))

	for metricType := range typesToCollect {
		wg.Add(1)
		go func(mType string) {
			defer wg.Done()

			records, err := c.collectMetricType(mType, startTime, endTime)
			if err != nil {
				errors <- fmt.Errorf("%s: %w", mType, err)
				return
			}
			results <- records
		}(metricType)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(results)
		close(errors)
	}()

	// Collect results
	var allRecords []*entity.MetricRecord
	var collectionErrors []error

	for {
		select {
		case records, ok := <-results:
			if !ok {
				results = nil
			} else {
				allRecords = append(allRecords, records...)
			}
		case err, ok := <-errors:
			if !ok {
				errors = nil
			} else {
				collectionErrors = append(collectionErrors, err)
			}
		}

		if results == nil && errors == nil {
			break
		}
	}

	// Log warnings for collection errors but don't fail the entire operation
	for _, err := range collectionErrors {
		c.logger.Warn(context.TODO(), "Failed to collect metrics",
			domain.NewField("error", err.Error()))
	}

	// If all collections failed, return error
	if len(collectionErrors) == len(typesToCollect) && len(collectionErrors) > 0 {
		return nil, fmt.Errorf("all metric collections failed")
	}

	c.logger.Info(context.TODO(), "Metrics collection completed",
		domain.NewField("totalRecords", len(allRecords)),
		domain.NewField("errors", len(collectionErrors)))

	return allRecords, nil
}

// collectMetricType collects metrics for a specific type
func (c *MetricsDataCollectorImpl) collectMetricType(metricType string, startTime, endTime time.Time) ([]*entity.MetricRecord, error) {
	switch metricType {
	case "claude_code":
		return c.collectClaudeCode(startTime, endTime)
	case "cursor":
		return c.collectCursor(startTime, endTime)
	case "bedrock":
		return c.collectBedrock(startTime, endTime)
	case "vertex_ai":
		return c.collectVertexAI(startTime, endTime)
	default:
		return nil, fmt.Errorf("unknown metric type: %s", metricType)
	}
}

// collectClaudeCode collects Claude Code metrics
func (c *MetricsDataCollectorImpl) collectClaudeCode(startTime, endTime time.Time) ([]*entity.MetricRecord, error) {
	// Check if Claude Code service is available
	if c.ccService == nil {
		return nil, nil // No Claude Code service configured
	}

	// Get date breakdown for the time range
	filter := usecase.DateBreakdownFilter{
		StartDate: &startTime,
		EndDate:   &endTime,
	}

	breakdown, err := c.ccService.CalculateDateBreakdown(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get date breakdown for claude_code: %w", err)
	}

	var records []*entity.MetricRecord
	for _, item := range breakdown.Dates {
		// Parse date string
		date, err := time.Parse("2006-01-02", item.Date)
		if err != nil {
			c.logger.Warn(context.TODO(), "Failed to parse date",
				domain.NewField("date", item.Date),
				domain.NewField("error", err.Error()))
			continue
		}

		// Create metric record
		record := entity.NewMetricRecord(
			date,
			"claude_code",
			"all_projects", // Aggregated across all projects
			float64(item.TotalTokens),
			"tokens",
		)

		// Add metadata
		record.AddMetadata("input_tokens", fmt.Sprintf("%d", item.InputTokens))
		record.AddMetadata("output_tokens", fmt.Sprintf("%d", item.OutputTokens))
		record.AddMetadata("cache_creation_tokens", fmt.Sprintf("%d", item.CacheCreationTokens))
		record.AddMetadata("cache_read_tokens", fmt.Sprintf("%d", item.CacheReadTokens))
		record.AddMetadata("cost", fmt.Sprintf("%.4f", item.Cost))
		record.AddMetadata("currency", item.Currency)
		record.AddMetadata("entry_count", fmt.Sprintf("%d", item.EntryCount))

		records = append(records, record)
	}

	return records, nil
}

// collectCursor collects Cursor metrics
func (c *MetricsDataCollectorImpl) collectCursor(startTime, endTime time.Time) ([]*entity.MetricRecord, error) {
	// Check if Cursor service is available
	if c.cursorService == nil {
		return nil, nil // No Cursor service configured
	}

	// For Cursor, we'll create records based on available data
	var records []*entity.MetricRecord

	// Get current usage to at least provide some data
	usage, err := c.cursorService.GetCurrentUsage()
	if err != nil {
		return nil, fmt.Errorf("failed to get cursor usage: %w", err)
	}

	// If we have usage data, create a record for current month usage
	if usage != nil {
		// Calculate total requests from usage-based pricing items
		totalRequests := 0
		for _, item := range usage.UsageBasedPricing().CurrentMonth.Items {
			totalRequests += item.RequestCount
		}

		if totalRequests > 0 {
			record := entity.NewMetricRecord(
				time.Now(),
				"cursor",
				"all_workspaces",
				float64(totalRequests),
				"requests",
			)

			// Add metadata
			record.AddMetadata("total_cost", fmt.Sprintf("%.2f", usage.CurrentMonthTotalCost()))
			record.AddMetadata("premium_requests_current", fmt.Sprintf("%d", usage.PremiumRequests().Current))
			record.AddMetadata("premium_requests_limit", fmt.Sprintf("%d", usage.PremiumRequests().Limit))

			records = append(records, record)
		}
	}

	return records, nil
}

// collectBedrock collects Bedrock metrics
func (c *MetricsDataCollectorImpl) collectBedrock(startTime, endTime time.Time) ([]*entity.MetricRecord, error) {
	// Check if Bedrock service is available
	if c.bedrockService == nil {
		return nil, nil // No Bedrock service configured
	}

	// Since GetUsageForPeriod is not available in the interface,
	// we'll collect daily usage for each day in the period
	var records []*entity.MetricRecord

	current := startTime
	for !current.After(endTime) {
		usage, err := c.bedrockService.GetDailyUsage(current)
		if err != nil {
			c.logger.Warn(context.TODO(), "Failed to get Bedrock daily usage",
				domain.NewField("date", current),
				domain.NewField("error", err.Error()))
			current = current.AddDate(0, 0, 1)
			continue
		}

		if usage != nil && (usage.InputTokens() > 0 || usage.OutputTokens() > 0) {
			record := entity.NewMetricRecord(
				current,
				"bedrock",
				"all_models",
				float64(usage.TotalTokens()),
				"tokens",
			)

			// Add metadata
			record.AddMetadata("input_tokens", fmt.Sprintf("%d", usage.InputTokens()))
			record.AddMetadata("output_tokens", fmt.Sprintf("%d", usage.OutputTokens()))
			record.AddMetadata("region", usage.Region())
			record.AddMetadata("total_cost", fmt.Sprintf("%.4f", usage.TotalCost()))

			records = append(records, record)
		}

		current = current.AddDate(0, 0, 1)
	}

	return records, nil
}

// collectVertexAI collects Vertex AI metrics
func (c *MetricsDataCollectorImpl) collectVertexAI(startTime, endTime time.Time) ([]*entity.MetricRecord, error) {
	// Check if Vertex AI service is available
	if c.vertexAIService == nil {
		return nil, nil // No Vertex AI service configured
	}

	// Since GetUsageForPeriod is not available in the interface,
	// we'll collect daily usage for each day in the period
	var records []*entity.MetricRecord

	current := startTime
	for !current.After(endTime) {
		usage, err := c.vertexAIService.GetDailyUsage(current)
		if err != nil {
			c.logger.Warn(context.TODO(), "Failed to get Vertex AI daily usage",
				domain.NewField("date", current),
				domain.NewField("error", err.Error()))
			current = current.AddDate(0, 0, 1)
			continue
		}

		if usage != nil && (usage.InputTokens() > 0 || usage.OutputTokens() > 0) {
			record := entity.NewMetricRecord(
				current,
				"vertex_ai",
				"all_models",
				float64(usage.TotalTokens()),
				"tokens",
			)

			// Add metadata
			record.AddMetadata("input_tokens", fmt.Sprintf("%d", usage.InputTokens()))
			record.AddMetadata("output_tokens", fmt.Sprintf("%d", usage.OutputTokens()))
			record.AddMetadata("project_id", usage.ProjectID())
			record.AddMetadata("location", usage.Location())
			record.AddMetadata("total_cost", fmt.Sprintf("%.4f", usage.TotalCost()))

			records = append(records, record)
		}

		current = current.AddDate(0, 0, 1)
	}

	return records, nil
}
