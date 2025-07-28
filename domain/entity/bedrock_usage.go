package entity

import (
	"fmt"
	"time"
)

// BedrockUsage represents AWS Bedrock usage data
type BedrockUsage struct {
	inputTokens  int64
	outputTokens int64
	totalTokens  int64
	totalCost    float64
	modelMetrics []BedrockModelMetric
	timestamp    time.Time
	region       string
	accountID    string
}

// BedrockModelMetric represents usage metrics for a specific Bedrock model
type BedrockModelMetric struct {
	ModelID         string
	InputTokens     int64
	OutputTokens    int64
	InvocationCount int64
	Cost            float64
	LatencyMs       float64
}

// NewBedrockUsage creates a new BedrockUsage instance
func NewBedrockUsage(
	inputTokens int64,
	outputTokens int64,
	totalCost float64,
	modelMetrics []BedrockModelMetric,
	region string,
	accountID string,
) (*BedrockUsage, error) {
	if inputTokens < 0 {
		return nil, fmt.Errorf("input tokens cannot be negative")
	}
	if outputTokens < 0 {
		return nil, fmt.Errorf("output tokens cannot be negative")
	}
	if totalCost < 0 {
		return nil, fmt.Errorf("total cost cannot be negative")
	}

	return &BedrockUsage{
		inputTokens:  inputTokens,
		outputTokens: outputTokens,
		totalTokens:  inputTokens + outputTokens,
		totalCost:    totalCost,
		modelMetrics: modelMetrics,
		timestamp:    time.Now(),
		region:       region,
		accountID:    accountID,
	}, nil
}

// InputTokens returns the total input tokens
func (b *BedrockUsage) InputTokens() int64 {
	return b.inputTokens
}

// OutputTokens returns the total output tokens
func (b *BedrockUsage) OutputTokens() int64 {
	return b.outputTokens
}

// TotalTokens returns the total tokens (input + output)
func (b *BedrockUsage) TotalTokens() int64 {
	return b.totalTokens
}

// TotalCost returns the total cost
func (b *BedrockUsage) TotalCost() float64 {
	return b.totalCost
}

// ModelMetrics returns the model-specific metrics
func (b *BedrockUsage) ModelMetrics() []BedrockModelMetric {
	return b.modelMetrics
}

// Timestamp returns when this usage data was created
func (b *BedrockUsage) Timestamp() time.Time {
	return b.timestamp
}

// Region returns the AWS region
func (b *BedrockUsage) Region() string {
	return b.region
}

// AccountID returns the AWS account ID
func (b *BedrockUsage) AccountID() string {
	return b.accountID
}

// GetModelMetric returns the metric for a specific model
func (b *BedrockUsage) GetModelMetric(modelID string) *BedrockModelMetric {
	for _, metric := range b.modelMetrics {
		if metric.ModelID == modelID {
			return &metric
		}
	}
	return nil
}

// GetTopModels returns the top N models by token usage
func (b *BedrockUsage) GetTopModels(n int) []BedrockModelMetric {
	if len(b.modelMetrics) == 0 {
		return []BedrockModelMetric{}
	}

	// Create a copy and sort by total tokens (input + output)
	metrics := make([]BedrockModelMetric, len(b.modelMetrics))
	copy(metrics, b.modelMetrics)

	// Simple bubble sort by total tokens
	for i := 0; i < len(metrics)-1; i++ {
		for j := 0; j < len(metrics)-i-1; j++ {
			totalJ := metrics[j].InputTokens + metrics[j].OutputTokens
			totalJPlus1 := metrics[j+1].InputTokens + metrics[j+1].OutputTokens
			if totalJ < totalJPlus1 {
				metrics[j], metrics[j+1] = metrics[j+1], metrics[j]
			}
		}
	}

	if n > len(metrics) {
		n = len(metrics)
	}
	return metrics[:n]
}

// CalculateTotalInvocations returns the total number of model invocations
func (b *BedrockUsage) CalculateTotalInvocations() int64 {
	total := int64(0)
	for _, metric := range b.modelMetrics {
		total += metric.InvocationCount
	}
	return total
}

// CalculateAverageLatency returns the average latency across all models
func (b *BedrockUsage) CalculateAverageLatency() float64 {
	if len(b.modelMetrics) == 0 {
		return 0
	}

	totalLatency := 0.0
	totalInvocations := int64(0)

	for _, metric := range b.modelMetrics {
		if metric.InvocationCount > 0 {
			totalLatency += metric.LatencyMs * float64(metric.InvocationCount)
			totalInvocations += metric.InvocationCount
		}
	}

	if totalInvocations == 0 {
		return 0
	}

	return totalLatency / float64(totalInvocations)
}

// IsEmpty checks if the usage data is empty
func (b *BedrockUsage) IsEmpty() bool {
	return b.totalTokens == 0 && b.totalCost == 0
}

// Validate checks if the usage data is valid
func (b *BedrockUsage) Validate() error {
	if b.inputTokens < 0 {
		return fmt.Errorf("input tokens cannot be negative")
	}
	if b.outputTokens < 0 {
		return fmt.Errorf("output tokens cannot be negative")
	}
	if b.totalCost < 0 {
		return fmt.Errorf("total cost cannot be negative")
	}
	if b.totalTokens != b.inputTokens+b.outputTokens {
		return fmt.Errorf("total tokens mismatch: expected %d, got %d",
			b.inputTokens+b.outputTokens, b.totalTokens)
	}

	// Validate model metrics
	for i, metric := range b.modelMetrics {
		if err := b.validateModelMetric(metric, i); err != nil {
			return fmt.Errorf("model metric %d validation failed: %w", i, err)
		}
	}

	return nil
}

// validateModelMetric validates a single model metric
func (b *BedrockUsage) validateModelMetric(metric BedrockModelMetric, index int) error {
	if metric.ModelID == "" {
		return fmt.Errorf("model ID cannot be empty")
	}
	if metric.InputTokens < 0 {
		return fmt.Errorf("input tokens cannot be negative")
	}
	if metric.OutputTokens < 0 {
		return fmt.Errorf("output tokens cannot be negative")
	}
	if metric.InvocationCount < 0 {
		return fmt.Errorf("invocation count cannot be negative")
	}
	if metric.Cost < 0 {
		return fmt.Errorf("cost cannot be negative")
	}
	if metric.LatencyMs < 0 {
		return fmt.Errorf("latency cannot be negative")
	}

	return nil
}
