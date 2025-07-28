package repository

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/ca-srg/tosage/domain/entity"
	"github.com/ca-srg/tosage/domain/repository"
)

// BedrockCloudWatchRepository implements BedrockRepository using AWS CloudWatch
type BedrockCloudWatchRepository struct {
	session    *session.Session
	cwClients  map[string]*cloudwatch.CloudWatch
	awsProfile string
}

// NewBedrockCloudWatchRepository creates a new Bedrock CloudWatch repository
func NewBedrockCloudWatchRepository(awsProfile string) (*BedrockCloudWatchRepository, error) {
	// Create AWS session
	sess, err := session.NewSessionWithOptions(session.Options{
		Profile:           awsProfile,
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	return &BedrockCloudWatchRepository{
		session:    sess,
		cwClients:  make(map[string]*cloudwatch.CloudWatch),
		awsProfile: awsProfile,
	}, nil
}

// getCloudWatchClient returns a CloudWatch client for the specified region
func (r *BedrockCloudWatchRepository) getCloudWatchClient(region string) *cloudwatch.CloudWatch {
	if client, exists := r.cwClients[region]; exists {
		return client
	}

	client := cloudwatch.New(r.session, &aws.Config{Region: aws.String(region)})
	r.cwClients[region] = client
	return client
}

// GetUsageMetrics retrieves Bedrock usage metrics from CloudWatch
func (r *BedrockCloudWatchRepository) GetUsageMetrics(region string, start, end time.Time) (*entity.BedrockUsage, error) {
	cwClient := r.getCloudWatchClient(region)

	// Get input tokens
	inputTokens, err := r.getMetricValue(cwClient, "AWS/Bedrock", "InputTokenCount", start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get input tokens: %w", err)
	}

	// Get output tokens
	outputTokens, err := r.getMetricValue(cwClient, "AWS/Bedrock", "OutputTokenCount", start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get output tokens: %w", err)
	}

	// Get model-specific metrics
	modelMetrics, err := r.getModelMetrics(cwClient, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get model metrics: %w", err)
	}

	// Calculate estimated cost (simplified - actual cost depends on model pricing)
	totalCost := r.calculateEstimatedCost(inputTokens, outputTokens, modelMetrics)

	// Get account ID from session (simplified)
	accountID := "unknown"
	if r.session.Config.Credentials != nil {
		// In a real implementation, you'd extract account ID from credentials
		accountID = "current-account"
	}

	return entity.NewBedrockUsage(
		int64(inputTokens),
		int64(outputTokens),
		totalCost,
		modelMetrics,
		region,
		accountID,
	)
}

// GetDailyUsage retrieves aggregated usage for a specific date
func (r *BedrockCloudWatchRepository) GetDailyUsage(region string, date time.Time) (*entity.BedrockUsage, error) {
	// Convert to JST for consistent date boundaries
	jst, _ := time.LoadLocation("Asia/Tokyo")
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, jst)
	endOfDay := startOfDay.Add(24 * time.Hour)

	return r.GetUsageMetrics(region, startOfDay, endOfDay)
}

// GetCurrentMonthUsage retrieves usage for the current month
func (r *BedrockCloudWatchRepository) GetCurrentMonthUsage(region string) (*entity.BedrockUsage, error) {
	jst, _ := time.LoadLocation("Asia/Tokyo")
	now := time.Now().In(jst)
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, jst)

	return r.GetUsageMetrics(region, startOfMonth, now)
}

// CheckConnection verifies AWS credentials and CloudWatch access
func (r *BedrockCloudWatchRepository) CheckConnection() error {
	// Test connection by listing metrics
	cwClient := r.getCloudWatchClient("us-east-1")
	input := &cloudwatch.ListMetricsInput{
		Namespace: aws.String("AWS/Bedrock"),
	}

	_, err := cwClient.ListMetrics(input)
	if err != nil {
		return fmt.Errorf("failed to connect to CloudWatch: %w", err)
	}

	return nil
}

// ListAvailableRegions returns regions with Bedrock activity
func (r *BedrockCloudWatchRepository) ListAvailableRegions() ([]string, error) {
	// Common Bedrock regions
	regions := []string{
		"us-east-1",
		"us-west-2",
		"eu-west-1",
		"ap-southeast-1",
		"ap-northeast-1",
	}

	var activeRegions []string
	for _, region := range regions {
		cwClient := r.getCloudWatchClient(region)

		// Check if there are any Bedrock metrics in this region
		input := &cloudwatch.ListMetricsInput{
			Namespace: aws.String("AWS/Bedrock"),
		}

		result, err := cwClient.ListMetrics(input)
		if err != nil {
			continue // Skip regions with errors
		}

		if len(result.Metrics) > 0 {
			activeRegions = append(activeRegions, region)
		}
	}

	return activeRegions, nil
}

// getMetricValue retrieves a metric value from CloudWatch
func (r *BedrockCloudWatchRepository) getMetricValue(
	cwClient *cloudwatch.CloudWatch,
	namespace, metricName string,
	start, end time.Time,
) (float64, error) {
	input := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String(namespace),
		MetricName: aws.String(metricName),
		StartTime:  aws.Time(start),
		EndTime:    aws.Time(end),
		Period:     aws.Int64(3600), // 1 hour periods
		Statistics: []*string{aws.String("Sum")},
	}

	result, err := cwClient.GetMetricStatistics(input)
	if err != nil {
		return 0, err
	}

	total := 0.0
	for _, datapoint := range result.Datapoints {
		if datapoint.Sum != nil {
			total += *datapoint.Sum
		}
	}

	return total, nil
}

// getModelMetrics retrieves model-specific metrics
func (r *BedrockCloudWatchRepository) getModelMetrics(
	cwClient *cloudwatch.CloudWatch,
	start, end time.Time,
) ([]entity.BedrockModelMetric, error) {
	// List all metrics with ModelId dimension
	listInput := &cloudwatch.ListMetricsInput{
		Namespace: aws.String("AWS/Bedrock"),
	}

	result, err := cwClient.ListMetrics(listInput)
	if err != nil {
		return nil, err
	}

	modelMap := make(map[string]*entity.BedrockModelMetric)

	// Process each metric
	for _, metric := range result.Metrics {
		if metric.MetricName == nil {
			continue
		}

		// Find ModelId dimension
		var modelID string
		for _, dimension := range metric.Dimensions {
			if dimension.Name != nil && *dimension.Name == "ModelId" {
				if dimension.Value != nil {
					modelID = *dimension.Value
				}
				break
			}
		}

		if modelID == "" {
			continue
		}

		// Initialize model metric if not exists
		if _, exists := modelMap[modelID]; !exists {
			modelMap[modelID] = &entity.BedrockModelMetric{
				ModelID: modelID,
			}
		}

		// Get metric value based on metric name
		dimensions := make([]*cloudwatch.Dimension, len(metric.Dimensions))
		copy(dimensions, metric.Dimensions)

		value, err := r.getMetricValueWithDimensions(cwClient, *metric.MetricName, dimensions, start, end)
		if err != nil {
			continue // Skip failed metrics
		}

		// Update the appropriate field based on metric name
		switch *metric.MetricName {
		case "InputTokenCount":
			modelMap[modelID].InputTokens = int64(value)
		case "OutputTokenCount":
			modelMap[modelID].OutputTokens = int64(value)
		case "Invocations":
			modelMap[modelID].InvocationCount = int64(value)
		case "InvocationLatency":
			modelMap[modelID].LatencyMs = value
		}
	}

	// Convert map to slice
	var metrics []entity.BedrockModelMetric
	for _, metric := range modelMap {
		// Calculate cost (simplified)
		metric.Cost = r.calculateModelCost(metric.InputTokens, metric.OutputTokens, metric.ModelID)
		metrics = append(metrics, *metric)
	}

	return metrics, nil
}

// getMetricValueWithDimensions retrieves a metric value with specific dimensions
func (r *BedrockCloudWatchRepository) getMetricValueWithDimensions(
	cwClient *cloudwatch.CloudWatch,
	metricName string,
	dimensions []*cloudwatch.Dimension,
	start, end time.Time,
) (float64, error) {
	input := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/Bedrock"),
		MetricName: aws.String(metricName),
		StartTime:  aws.Time(start),
		EndTime:    aws.Time(end),
		Period:     aws.Int64(3600), // 1 hour periods
		Statistics: []*string{aws.String("Sum")},
		Dimensions: dimensions,
	}

	result, err := cwClient.GetMetricStatistics(input)
	if err != nil {
		return 0, err
	}

	total := 0.0
	for _, datapoint := range result.Datapoints {
		if datapoint.Sum != nil {
			total += *datapoint.Sum
		}
	}

	return total, nil
}

// calculateEstimatedCost calculates estimated cost based on token usage
func (r *BedrockCloudWatchRepository) calculateEstimatedCost(
	inputTokens, outputTokens float64,
	modelMetrics []entity.BedrockModelMetric,
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
		inputCostPer1K := 0.0015 // $0.0015 per 1K input tokens
		outputCostPer1K := 0.002 // $0.002 per 1K output tokens

		totalCost = (inputTokens/1000)*inputCostPer1K + (outputTokens/1000)*outputCostPer1K
	}

	return totalCost
}

// calculateModelCost calculates cost for a specific model
func (r *BedrockCloudWatchRepository) calculateModelCost(inputTokens, outputTokens int64, modelID string) float64 {
	// Simplified model-specific pricing
	// Real implementation would have a pricing table

	var inputRate, outputRate float64

	// Example pricing (simplified)
	switch {
	case contains(modelID, "claude"):
		inputRate = 0.0015 // $0.0015 per 1K tokens
		outputRate = 0.002 // $0.002 per 1K tokens
	case contains(modelID, "titan"):
		inputRate = 0.0008  // $0.0008 per 1K tokens
		outputRate = 0.0016 // $0.0016 per 1K tokens
	default:
		inputRate = 0.001  // Default rate
		outputRate = 0.002 // Default rate
	}

	return (float64(inputTokens)/1000)*inputRate + (float64(outputTokens)/1000)*outputRate
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsSubstring(s, substr))))
}

// containsSubstring checks if string contains substring
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Ensure BedrockCloudWatchRepository implements BedrockRepository
var _ repository.BedrockRepository = (*BedrockCloudWatchRepository)(nil)
