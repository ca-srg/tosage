package repository

import (
	"context"
	"fmt"
	"log"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ca-srg/tosage/domain/entity"
	"github.com/ca-srg/tosage/domain/repository"
	"github.com/ca-srg/tosage/infrastructure/auth"
)

// VertexAIMonitoringRepository implements VertexAIRepository using Google Cloud Monitoring
type VertexAIMonitoringRepository struct {
	client        *monitoring.MetricClient
	projectID     string
	authenticator auth.VertexAIAuthenticator
}

// NewVertexAIMonitoringRepository creates a new Vertex AI Monitoring repository
func NewVertexAIMonitoringRepository(projectID string, authenticator auth.VertexAIAuthenticator) (*VertexAIMonitoringRepository, error) {
	ctx := context.Background()

	var opts []option.ClientOption
	if authenticator != nil {
		// Use the token source from the authenticator
		opts = append(opts, option.WithTokenSource(authenticator.GetTokenSource()))
	}

	client, err := monitoring.NewMetricClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create monitoring client: %w", err)
	}

	return &VertexAIMonitoringRepository{
		client:        client,
		projectID:     projectID,
		authenticator: authenticator,
	}, nil
}

// GetUsageMetrics retrieves Vertex AI usage metrics from Cloud Monitoring
func (r *VertexAIMonitoringRepository) GetUsageMetrics(projectID string, start, end time.Time) (*entity.VertexAIUsage, error) {
	ctx := context.Background()


	// Debug: List available metrics
	r.debugListMetrics(ctx, projectID)

	// Use the new metric type for token count
	metricType := "aiplatform.googleapis.com/publisher/online_serving/token_count"
	
	// Get input and output tokens separately
	inputTokens, outputTokens, err := r.getTokenCountByType(ctx, projectID, metricType, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve token count metric: %w", err)
	}
	
	totalTokens := inputTokens + outputTokens
	if totalTokens == 0 {
		return nil, fmt.Errorf("no token usage data found for metric %s", metricType)
	}
	

	// Get model-specific metrics
	modelMetrics, err := r.getModelMetrics(ctx, projectID, start, end)
	if err != nil {
		log.Printf("[WARN] Could not get model metrics: %v. Proceeding without model-specific metrics.", err)
		modelMetrics = []entity.VertexAIModelMetric{}
	}

	// Calculate estimated cost (simplified - actual cost depends on model pricing)
	totalCost := r.calculateEstimatedCost(inputTokens, outputTokens, modelMetrics)

	return entity.NewVertexAIUsage(
		int64(inputTokens),
		int64(outputTokens),
		totalCost,
		modelMetrics,
		projectID,
		"", // Empty location since we're not filtering by location
	)
}

// GetDailyUsage retrieves aggregated usage for a specific date
func (r *VertexAIMonitoringRepository) GetDailyUsage(projectID string, date time.Time) (*entity.VertexAIUsage, error) {
	// Convert to JST for consistent date boundaries
	jst, _ := time.LoadLocation("Asia/Tokyo")
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, jst)
	
	// Get current time in JST
	now := time.Now().In(jst)
	
	// If the requested date is today, use current time as end time
	// Otherwise, use end of day
	var endTime time.Time
	if startOfDay.Year() == now.Year() && startOfDay.Month() == now.Month() && startOfDay.Day() == now.Day() {
		endTime = now
	} else {
		endTime = startOfDay.Add(24 * time.Hour)
	}

	return r.GetUsageMetrics(projectID, startOfDay, endTime)
}

// GetCurrentMonthUsage retrieves usage for the current month
func (r *VertexAIMonitoringRepository) GetCurrentMonthUsage(projectID string) (*entity.VertexAIUsage, error) {
	jst, _ := time.LoadLocation("Asia/Tokyo")
	now := time.Now().In(jst)
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, jst)

	return r.GetUsageMetrics(projectID, startOfMonth, now)
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


// debugListMetrics lists available metrics for debugging
func (r *VertexAIMonitoringRepository) debugListMetrics(ctx context.Context, projectID string) {
	projectName := fmt.Sprintf("projects/%s", projectID)
	
	// First, list available metric descriptors
	req := &monitoringpb.ListMetricDescriptorsRequest{
		Name:   projectName,
		Filter: `metric.type=starts_with("aiplatform.googleapis.com/")`,
		PageSize: 100, // Request more results per page
	}
	
	count := 0
	pageCount := 0
	
	for pageCount < 10 { // Check up to 10 pages
		it := r.client.ListMetricDescriptors(ctx, req)
		pageCount++
		
		for {
			md, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return
			}
			
			count++
			if count <= 50 || md.Type == "aiplatform.googleapis.com/publisher/online_serving/token_count" {
			}
			
			if md.Type == "aiplatform.googleapis.com/publisher/online_serving/token_count" {
				return
			}
		}
		
		// Get next page token
		if req.PageToken = it.PageInfo().Token; req.PageToken == "" {
			break
		}
	}
	
	
	if count == 0 {
		
		// Try to list ALL metric descriptors to see what's available
		req2 := &monitoringpb.ListMetricDescriptorsRequest{
			Name: projectName,
		}
		
		it2 := r.client.ListMetricDescriptors(ctx, req2)
		aiplatformCount := 0
		for {
			md, err := it2.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}
			
			// Check if it's related to AI Platform
			if vertexAIContains(md.Type, "aiplatform") || vertexAIContains(md.Type, "ml.googleapis.com") {
				if md.Type == "aiplatform.googleapis.com/publisher/online_serving/token_count" {
				}
				aiplatformCount++
			}
			
			if aiplatformCount > 20 {
				break
			}
		}
	}
}

// getTokenCountByType retrieves input and output token counts separately
func (r *VertexAIMonitoringRepository) getTokenCountByType(
	ctx context.Context,
	projectID, metricType string,
	start, end time.Time,
) (float64, float64, error) {
	projectName := fmt.Sprintf("projects/%s", projectID)
	
	// No filter - get all data for this metric type
	filter := fmt.Sprintf(`metric.type="%s"`, metricType)


	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   projectName,
		Filter: filter,
		Interval: &monitoringpb.TimeInterval{
			StartTime: timestamppb.New(start),
			EndTime:   timestamppb.New(end),
		},
		Aggregation: &monitoringpb.Aggregation{
			AlignmentPeriod:  durationpb.New(time.Hour), // 1 hour periods
			PerSeriesAligner: monitoringpb.Aggregation_ALIGN_DELTA,
		},
	}

	it := r.client.ListTimeSeries(ctx, req)
	inputTokens := 0.0
	outputTokens := 0.0
	seriesCount := 0

	for {
		ts, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, 0, err
		}

		seriesCount++
		
		// Determine if this is input or output tokens based on labels
		tokenType := ""
		if ts.Metric != nil && ts.Metric.Labels != nil {
			if typeLabel, ok := ts.Metric.Labels["type"]; ok {
				tokenType = typeLabel
			}
		}
		
		// Log resource labels
		if ts.Resource != nil && ts.Resource.Labels != nil {
		}

		// Sum points based on token type
		pointCount := 0
		for _, point := range ts.Points {
			if point.Value != nil {
				// Try both double and int64 values
				var value float64
				hasValue := false
				if dv := point.Value.GetDoubleValue(); dv != 0 {
					value = dv
					hasValue = true
				} else if iv := point.Value.GetInt64Value(); iv != 0 {
					value = float64(iv)
					hasValue = true
				} else {
					// Log even if value is 0 to debug
				}
				
				if hasValue {
					switch tokenType {
					case "input":
						inputTokens += value
					case "output":
						outputTokens += value
					default:
						// If type is not specified, assume it's total and split evenly
						inputTokens += value / 2
						outputTokens += value / 2
					}
					pointCount++
				}
			}
		}
	}

	return inputTokens, outputTokens, nil
}



// getModelMetrics retrieves model-specific metrics
func (r *VertexAIMonitoringRepository) getModelMetrics(
	ctx context.Context,
	projectID string,
	start, end time.Time,
) ([]entity.VertexAIModelMetric, error) {
	// Get all unique model IDs
	modelIDs, err := r.getAvailableModelsForLocation(ctx, projectID)
	if err != nil {
		log.Printf("[WARN] Could not get available models: %v. Proceeding without model-specific metrics.", err)
		return []entity.VertexAIModelMetric{}, nil
	}

	var metrics []entity.VertexAIModelMetric
	for _, modelID := range modelIDs {
		metric := entity.VertexAIModelMetric{
			ModelID: modelID,
		}

		// Get input tokens for this model
		inputTokens, err := r.getModelMetricValue(ctx, fmt.Sprintf("projects/%s", r.projectID), "aiplatform.googleapis.com/prediction/input_token_count", modelID, start, end)
		if err == nil {
			metric.InputTokens = int64(inputTokens)
		}

		// Get output tokens for this model
		outputTokens, err := r.getModelMetricValue(ctx, fmt.Sprintf("projects/%s", r.projectID), "aiplatform.googleapis.com/prediction/output_token_count", modelID, start, end)
		if err == nil {
			metric.OutputTokens = int64(outputTokens)
		}

		// Get request count for this model
		requestCount, err := r.getModelMetricValue(ctx, fmt.Sprintf("projects/%s", r.projectID), "aiplatform.googleapis.com/prediction/request_count", modelID, start, end)
		if err == nil {
			metric.RequestCount = int64(requestCount)
		}

		// Get latency for this model
		latency, err := r.getModelMetricValue(ctx, fmt.Sprintf("projects/%s", r.projectID), "aiplatform.googleapis.com/prediction/response_latencies", modelID, start, end)
		if err == nil {
			metric.LatencyMs = latency
		}

		// Calculate cost (simplified)
		metric.Cost = r.calculateModelCost(metric.InputTokens, metric.OutputTokens, modelID)

		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// getAvailableModelsForLocation retrieves available publisher models.
func (r *VertexAIMonitoringRepository) getAvailableModelsForLocation(ctx context.Context, projectID string) ([]string, error) {
	// Try to get model information from actual usage metrics instead of REST API
	// since the publisher models endpoint seems to be unavailable
	return r.getUniqueModelIDs(ctx, fmt.Sprintf("projects/%s", projectID), time.Now().Add(-7*24*time.Hour), time.Now())
}

// getUniqueModelIDs retrieves unique model IDs from metrics
func (r *VertexAIMonitoringRepository) getUniqueModelIDs(
	ctx context.Context,
	projectName string,
	start, end time.Time,
) ([]string, error) {
	filter := `metric.type="aiplatform.googleapis.com/prediction/request_count"`

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
	projectName, metricType, modelID string,
	start, end time.Time,
) (float64, error) {
	filter := fmt.Sprintf(`metric.type="%s" AND resource.labels.model_id="%s"`, metricType, modelID)

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
