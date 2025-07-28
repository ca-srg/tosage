package repository

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ca-srg/tosage/domain/entity"
	"github.com/ca-srg/tosage/domain/repository"
)

// VertexAIMonitoringRepository implements VertexAIRepository using Google Cloud Monitoring
type VertexAIMonitoringRepository struct {
	client                *monitoring.MetricClient
	projectID             string
	serviceAccountKeyPath string
}

// NewVertexAIMonitoringRepository creates a new Vertex AI Monitoring repository
func NewVertexAIMonitoringRepository(projectID, serviceAccountKeyPath string) (*VertexAIMonitoringRepository, error) {
	ctx := context.Background()

	var opts []option.ClientOption
	if serviceAccountKeyPath != "" {
		opts = append(opts, option.WithCredentialsFile(serviceAccountKeyPath))
	}

	client, err := monitoring.NewMetricClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create monitoring client: %w", err)
	}

	return &VertexAIMonitoringRepository{
		client:                client,
		projectID:             projectID,
		serviceAccountKeyPath: serviceAccountKeyPath,
	}, nil
}

// GetUsageMetrics retrieves Vertex AI usage metrics from Cloud Monitoring
func (r *VertexAIMonitoringRepository) GetUsageMetrics(projectID, location string, start, end time.Time) (*entity.VertexAIUsage, error) {
	ctx := context.Background()

	log.Printf("[DEBUG] GetUsageMetrics called with projectID=%s, location=%s, start=%v, end=%v",
		projectID, location, start.Format(time.RFC3339), end.Format(time.RFC3339))

	// Debug: List available metrics
	r.debugListMetrics(ctx, projectID, location, start, end)

	// Get input tokens
	inputTokens, err := r.getMetricValue(ctx, projectID, "aiplatform.googleapis.com/prediction/input_token_count", location, start, end)
	if err != nil {
		// Log the error but continue with 0
		log.Printf("[DEBUG] Failed to get input tokens: %v", err)
		inputTokens = 0
	}

	// Get output tokens
	outputTokens, err := r.getMetricValue(ctx, projectID, "aiplatform.googleapis.com/prediction/output_token_count", location, start, end)
	if err != nil {
		// Log the error but continue with 0
		log.Printf("[DEBUG] Failed to get output tokens: %v", err)
		outputTokens = 0
	}

	// Get model-specific metrics
	modelMetrics, err := r.getModelMetrics(ctx, projectID, location, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get model metrics: %w", err)
	}

	// Calculate estimated cost (simplified - actual cost depends on model pricing)
	totalCost := r.calculateEstimatedCost(inputTokens, outputTokens, modelMetrics)

	return entity.NewVertexAIUsage(
		int64(inputTokens),
		int64(outputTokens),
		totalCost,
		modelMetrics,
		projectID,
		location,
	)
}

// GetDailyUsage retrieves aggregated usage for a specific date
func (r *VertexAIMonitoringRepository) GetDailyUsage(projectID, location string, date time.Time) (*entity.VertexAIUsage, error) {
	// Convert to JST for consistent date boundaries
	jst, _ := time.LoadLocation("Asia/Tokyo")
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, jst)
	endOfDay := startOfDay.Add(24 * time.Hour)

	return r.GetUsageMetrics(projectID, location, startOfDay, endOfDay)
}

// GetCurrentMonthUsage retrieves usage for the current month
func (r *VertexAIMonitoringRepository) GetCurrentMonthUsage(projectID, location string) (*entity.VertexAIUsage, error) {
	jst, _ := time.LoadLocation("Asia/Tokyo")
	now := time.Now().In(jst)
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, jst)

	return r.GetUsageMetrics(projectID, location, startOfMonth, now)
}

// CheckConnection verifies Google Cloud credentials and Cloud Monitoring access
func (r *VertexAIMonitoringRepository) CheckConnection() error {
	ctx := context.Background()

	// Test connection by listing metric descriptors
	projectName := fmt.Sprintf("projects/%s", r.projectID)
	req := &monitoringpb.ListMetricDescriptorsRequest{
		Name:   projectName,
		Filter: "metric.type=starts_with(\"aiplatform.googleapis.com/\")",
	}

	it := r.client.ListMetricDescriptors(ctx, req)
	_, err := it.Next()
	if err != nil && err != iterator.Done {
		return fmt.Errorf("failed to connect to Cloud Monitoring: %w", err)
	}

	return nil
}

// ListAvailableLocations returns locations with Vertex AI activity
func (r *VertexAIMonitoringRepository) ListAvailableLocations(projectID string) ([]string, error) {
	// Common Vertex AI locations
	locations := []string{
		"us-central1",
		"us-east1",
		"us-west1",
		"europe-west1",
		"europe-west4",
		"asia-northeast1",
		"asia-southeast1",
	}

	var activeLocations []string
	for _, location := range locations {
		// Check if there are any Vertex AI metrics in this location
		hasActivity, err := r.checkLocationActivity(projectID, location)
		if err != nil {
			continue // Skip locations with errors
		}

		if hasActivity {
			activeLocations = append(activeLocations, location)
		}
	}

	return activeLocations, nil
}

// checkLocationActivity checks if there's any Vertex AI activity in a location
func (r *VertexAIMonitoringRepository) checkLocationActivity(projectID, location string) (bool, error) {
	ctx := context.Background()

	// Look for recent activity in the last 7 days
	end := time.Now()
	start := end.Add(-7 * 24 * time.Hour)

	projectName := fmt.Sprintf("projects/%s", projectID)
	filter := fmt.Sprintf(`metric.type="aiplatform.googleapis.com/prediction/request_count" AND resource.labels.location="%s"`, location)

	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   projectName,
		Filter: filter,
		Interval: &monitoringpb.TimeInterval{
			StartTime: timestamppb.New(start),
			EndTime:   timestamppb.New(end),
		},
	}

	it := r.client.ListTimeSeries(ctx, req)
	_, err := it.Next()
	if err != nil {
		if err == iterator.Done {
			return false, nil // No activity found
		}
		return false, err
	}

	return true, nil // Activity found
}

// debugListMetrics lists available metrics for debugging
func (r *VertexAIMonitoringRepository) debugListMetrics(ctx context.Context, projectID, location string, start, end time.Time) {
	projectName := fmt.Sprintf("projects/%s", projectID)
	// Try different filters to find metrics
	filters := []string{
		fmt.Sprintf(`resource.type=~".*aiplatform.*" AND resource.labels.location="%s"`, location),
		fmt.Sprintf(`metric.type=~".*aiplatform.*" AND resource.labels.location="%s"`, location),
		`metric.type=~".*aiplatform.*"`,
		`resource.type=~".*aiplatform.*"`,
	}

	for _, filter := range filters {
		log.Printf("[DEBUG] Trying filter: %s", filter)

		req := &monitoringpb.ListTimeSeriesRequest{
			Name:   projectName,
			Filter: filter,
			Interval: &monitoringpb.TimeInterval{
				StartTime: timestamppb.New(start),
				EndTime:   timestamppb.New(end),
			},
			View: monitoringpb.ListTimeSeriesRequest_HEADERS,
		}

		it := r.client.ListTimeSeries(ctx, req)
		count := 0
		for {
			ts, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[DEBUG] Error with filter '%s': %v", filter, err)
				break
			}

			log.Printf("[DEBUG] Found metric: type=%s, resource.type=%s, resource.labels=%v",
				ts.Metric.Type, ts.Resource.Type, ts.Resource.Labels)
			count++
			if count > 5 {
				break
			}
		}

		if count > 0 {
			log.Printf("[DEBUG] Found %d metrics with filter: %s", count, filter)
			return
		}
	}

	log.Printf("[DEBUG] No metrics found in any filter for project %s", projectID)
}

// getMetricValue retrieves a metric value from Cloud Monitoring
func (r *VertexAIMonitoringRepository) getMetricValue(
	ctx context.Context,
	projectID, metricType, location string,
	start, end time.Time,
) (float64, error) {
	projectName := fmt.Sprintf("projects/%s", projectID)
	// Try both resource types for compatibility
	filter := fmt.Sprintf(`metric.type="%s" AND (resource.type="aiplatform.googleapis.com/Model" OR resource.type="aiplatform.googleapis.com/PublisherModel") AND resource.labels.location="%s"`, metricType, location)

	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   projectName,
		Filter: filter,
		Interval: &monitoringpb.TimeInterval{
			StartTime: timestamppb.New(start),
			EndTime:   timestamppb.New(end),
		},
		Aggregation: &monitoringpb.Aggregation{
			AlignmentPeriod:  durationpb.New(time.Hour), // 1 hour periods
			PerSeriesAligner: monitoringpb.Aggregation_ALIGN_RATE,
		},
	}

	it := r.client.ListTimeSeries(ctx, req)
	total := 0.0

	for {
		ts, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}

		for _, point := range ts.Points {
			if point.Value != nil && point.Value.GetDoubleValue() != 0 {
				total += point.Value.GetDoubleValue()
			}
		}
	}

	return total, nil
}

// getModelMetrics retrieves model-specific metrics
func (r *VertexAIMonitoringRepository) getModelMetrics(
	ctx context.Context,
	projectID, location string,
	start, end time.Time,
) ([]entity.VertexAIModelMetric, error) {
	projectName := fmt.Sprintf("projects/%s", projectID)

	// Get all unique model IDs
	modelIDs, err := r.getUniqueModelIDs(ctx, projectName, location, start, end)
	if err != nil {
		return nil, err
	}

	var metrics []entity.VertexAIModelMetric
	for _, modelID := range modelIDs {
		metric := entity.VertexAIModelMetric{
			ModelID: modelID,
		}

		// Get input tokens for this model
		inputTokens, err := r.getModelMetricValue(ctx, projectName, "aiplatform.googleapis.com/prediction/input_token_count", modelID, location, start, end)
		if err == nil {
			metric.InputTokens = int64(inputTokens)
		}

		// Get output tokens for this model
		outputTokens, err := r.getModelMetricValue(ctx, projectName, "aiplatform.googleapis.com/prediction/output_token_count", modelID, location, start, end)
		if err == nil {
			metric.OutputTokens = int64(outputTokens)
		}

		// Get request count for this model
		requestCount, err := r.getModelMetricValue(ctx, projectName, "aiplatform.googleapis.com/prediction/request_count", modelID, location, start, end)
		if err == nil {
			metric.RequestCount = int64(requestCount)
		}

		// Get latency for this model
		latency, err := r.getModelMetricValue(ctx, projectName, "aiplatform.googleapis.com/prediction/response_latencies", modelID, location, start, end)
		if err == nil {
			metric.LatencyMs = latency
		}

		// Calculate cost (simplified)
		metric.Cost = r.calculateModelCost(metric.InputTokens, metric.OutputTokens, modelID)

		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// getUniqueModelIDs retrieves unique model IDs from metrics
func (r *VertexAIMonitoringRepository) getUniqueModelIDs(
	ctx context.Context,
	projectName, location string,
	start, end time.Time,
) ([]string, error) {
	filter := fmt.Sprintf(`metric.type="aiplatform.googleapis.com/prediction/request_count" AND resource.labels.location="%s"`, location)

	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   projectName,
		Filter: filter,
		Interval: &monitoringpb.TimeInterval{
			StartTime: timestamppb.New(start),
			EndTime:   timestamppb.New(end),
		},
	}

	it := r.client.ListTimeSeries(ctx, req)
	modelIDSet := make(map[string]bool)

	for {
		ts, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		// Extract model ID from labels
		if ts.Resource != nil && ts.Resource.Labels != nil {
			if modelID, exists := ts.Resource.Labels["model_id"]; exists {
				modelIDSet[modelID] = true
			}
		}
	}

	var modelIDs []string
	for modelID := range modelIDSet {
		modelIDs = append(modelIDs, modelID)
	}

	return modelIDs, nil
}

// getModelMetricValue retrieves a metric value for a specific model
func (r *VertexAIMonitoringRepository) getModelMetricValue(
	ctx context.Context,
	projectName, metricType, modelID, location string,
	start, end time.Time,
) (float64, error) {
	filter := fmt.Sprintf(`metric.type="%s" AND resource.labels.model_id="%s" AND resource.labels.location="%s"`, metricType, modelID, location)

	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   projectName,
		Filter: filter,
		Interval: &monitoringpb.TimeInterval{
			StartTime: timestamppb.New(start),
			EndTime:   timestamppb.New(end),
		},
		Aggregation: &monitoringpb.Aggregation{
			AlignmentPeriod:  durationpb.New(time.Hour), // 1 hour periods
			PerSeriesAligner: monitoringpb.Aggregation_ALIGN_RATE,
		},
	}

	it := r.client.ListTimeSeries(ctx, req)
	total := 0.0

	for {
		ts, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}

		for _, point := range ts.Points {
			if point.Value != nil && point.Value.GetDoubleValue() != 0 {
				total += point.Value.GetDoubleValue()
			}
		}
	}

	return total, nil
}

// calculateEstimatedCost calculates estimated cost based on token usage
func (r *VertexAIMonitoringRepository) calculateEstimatedCost(
	inputTokens, outputTokens float64,
	modelMetrics []entity.VertexAIModelMetric,
) float64 {
	// Simplified cost calculation
	// Real implementation would use actual pricing per model

	totalCost := 0.0
	for _, metric := range modelMetrics {
		totalCost += metric.Cost
	}

	// If no model-specific costs, use approximate rates
	if totalCost == 0 {
		// Approximate costs per 1000 tokens (varies by model)
		inputCostPer1K := 0.0025  // $0.0025 per 1K input tokens (Gemini Pro)
		outputCostPer1K := 0.0075 // $0.0075 per 1K output tokens (Gemini Pro)

		totalCost = (inputTokens/1000)*inputCostPer1K + (outputTokens/1000)*outputCostPer1K
	}

	return totalCost
}

// calculateModelCost calculates cost for a specific model
func (r *VertexAIMonitoringRepository) calculateModelCost(inputTokens, outputTokens int64, modelID string) float64 {
	// Simplified model-specific pricing
	// Real implementation would have a pricing table

	var inputRate, outputRate float64

	// Example pricing (simplified)
	switch {
	case vertexAIContains(modelID, "gemini"):
		inputRate = 0.0025  // $0.0025 per 1K tokens
		outputRate = 0.0075 // $0.0075 per 1K tokens
	case vertexAIContains(modelID, "palm"):
		inputRate = 0.0005  // $0.0005 per 1K tokens
		outputRate = 0.0015 // $0.0015 per 1K tokens
	default:
		inputRate = 0.002  // Default rate
		outputRate = 0.006 // Default rate
	}

	return (float64(inputTokens)/1000)*inputRate + (float64(outputTokens)/1000)*outputRate
}

// vertexAIContains checks if a string contains a substring (case-insensitive)
func vertexAIContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				vertexAIContainsSubstring(s, substr))))
}

// vertexAIContainsSubstring checks if string contains substring
func vertexAIContainsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Close closes the monitoring client
func (r *VertexAIMonitoringRepository) Close() error {
	return r.client.Close()
}

// Ensure VertexAIMonitoringRepository implements VertexAIRepository
var _ repository.VertexAIRepository = (*VertexAIMonitoringRepository)(nil)
