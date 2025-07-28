package entity

import (
	"fmt"
	"time"
)

// VertexAIUsage represents Google Cloud Vertex AI usage data
type VertexAIUsage struct {
	inputTokens  int64
	outputTokens int64
	totalTokens  int64
	totalCost    float64
	modelMetrics []VertexAIModelMetric
	timestamp    time.Time
	projectID    string
	location     string
}

// VertexAIModelMetric represents usage metrics for a specific Vertex AI model
type VertexAIModelMetric struct {
	ModelID      string
	InputTokens  int64
	OutputTokens int64
	RequestCount int64
	Cost         float64
	LatencyMs    float64
}

// NewVertexAIUsage creates a new VertexAIUsage instance
func NewVertexAIUsage(
	inputTokens int64,
	outputTokens int64,
	totalCost float64,
	modelMetrics []VertexAIModelMetric,
	projectID string,
	location string,
) (*VertexAIUsage, error) {
	if inputTokens < 0 {
		return nil, fmt.Errorf("input tokens cannot be negative")
	}
	if outputTokens < 0 {
		return nil, fmt.Errorf("output tokens cannot be negative")
	}
	if totalCost < 0 {
		return nil, fmt.Errorf("total cost cannot be negative")
	}

	return &VertexAIUsage{
		inputTokens:  inputTokens,
		outputTokens: outputTokens,
		totalTokens:  inputTokens + outputTokens,
		totalCost:    totalCost,
		modelMetrics: modelMetrics,
		timestamp:    time.Now(),
		projectID:    projectID,
		location:     location,
	}, nil
}

// InputTokens returns the total input tokens
func (v *VertexAIUsage) InputTokens() int64 {
	return v.inputTokens
}

// OutputTokens returns the total output tokens
func (v *VertexAIUsage) OutputTokens() int64 {
	return v.outputTokens
}

// TotalTokens returns the total tokens (input + output)
func (v *VertexAIUsage) TotalTokens() int64 {
	return v.totalTokens
}

// TotalCost returns the total cost
func (v *VertexAIUsage) TotalCost() float64 {
	return v.totalCost
}

// ModelMetrics returns the model-specific metrics
func (v *VertexAIUsage) ModelMetrics() []VertexAIModelMetric {
	return v.modelMetrics
}

// Timestamp returns when this usage data was created
func (v *VertexAIUsage) Timestamp() time.Time {
	return v.timestamp
}

// ProjectID returns the Google Cloud project ID
func (v *VertexAIUsage) ProjectID() string {
	return v.projectID
}

// Location returns the Google Cloud location/region
func (v *VertexAIUsage) Location() string {
	return v.location
}

// GetModelMetric returns the metric for a specific model
func (v *VertexAIUsage) GetModelMetric(modelID string) *VertexAIModelMetric {
	for _, metric := range v.modelMetrics {
		if metric.ModelID == modelID {
			return &metric
		}
	}
	return nil
}

// GetTopModels returns the top N models by token usage
func (v *VertexAIUsage) GetTopModels(n int) []VertexAIModelMetric {
	if len(v.modelMetrics) == 0 {
		return []VertexAIModelMetric{}
	}

	// Create a copy and sort by total tokens (input + output)
	metrics := make([]VertexAIModelMetric, len(v.modelMetrics))
	copy(metrics, v.modelMetrics)

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

// CalculateTotalRequests returns the total number of requests
func (v *VertexAIUsage) CalculateTotalRequests() int64 {
	total := int64(0)
	for _, metric := range v.modelMetrics {
		total += metric.RequestCount
	}
	return total
}

// CalculateAverageLatency returns the average latency across all models
func (v *VertexAIUsage) CalculateAverageLatency() float64 {
	if len(v.modelMetrics) == 0 {
		return 0
	}

	totalLatency := 0.0
	totalRequests := int64(0)

	for _, metric := range v.modelMetrics {
		if metric.RequestCount > 0 {
			totalLatency += metric.LatencyMs * float64(metric.RequestCount)
			totalRequests += metric.RequestCount
		}
	}

	if totalRequests == 0 {
		return 0
	}

	return totalLatency / float64(totalRequests)
}

// IsEmpty checks if the usage data is empty
func (v *VertexAIUsage) IsEmpty() bool {
	return v.totalTokens == 0 && v.totalCost == 0
}

// Validate checks if the usage data is valid
func (v *VertexAIUsage) Validate() error {
	if v.inputTokens < 0 {
		return fmt.Errorf("input tokens cannot be negative")
	}
	if v.outputTokens < 0 {
		return fmt.Errorf("output tokens cannot be negative")
	}
	if v.totalCost < 0 {
		return fmt.Errorf("total cost cannot be negative")
	}
	if v.totalTokens != v.inputTokens+v.outputTokens {
		return fmt.Errorf("total tokens mismatch: expected %d, got %d",
			v.inputTokens+v.outputTokens, v.totalTokens)
	}

	// Validate model metrics
	for i, metric := range v.modelMetrics {
		if err := v.validateModelMetric(metric, i); err != nil {
			return fmt.Errorf("model metric %d validation failed: %w", i, err)
		}
	}

	return nil
}

// validateModelMetric validates a single model metric
func (v *VertexAIUsage) validateModelMetric(metric VertexAIModelMetric, index int) error {
	if metric.ModelID == "" {
		return fmt.Errorf("model ID cannot be empty")
	}
	if metric.InputTokens < 0 {
		return fmt.Errorf("input tokens cannot be negative")
	}
	if metric.OutputTokens < 0 {
		return fmt.Errorf("output tokens cannot be negative")
	}
	if metric.RequestCount < 0 {
		return fmt.Errorf("request count cannot be negative")
	}
	if metric.Cost < 0 {
		return fmt.Errorf("cost cannot be negative")
	}
	if metric.LatencyMs < 0 {
		return fmt.Errorf("latency cannot be negative")
	}

	return nil
}
