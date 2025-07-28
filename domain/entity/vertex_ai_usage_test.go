package entity

import (
	"testing"
	"time"
)

func TestNewVertexAIUsage(t *testing.T) {
	tests := []struct {
		name           string
		inputTokens    int64
		outputTokens   int64
		totalCost      float64
		modelMetrics   []VertexAIModelMetric
		projectID      string
		location       string
		expectedError  string
	}{
		{
			name:         "valid usage",
			inputTokens:  1000,
			outputTokens: 500,
			totalCost:    5.25,
			modelMetrics: []VertexAIModelMetric{
				{
					ModelID:      "gemini-pro",
					InputTokens:  1000,
					OutputTokens: 500,
					RequestCount: 10,
					Cost:         5.25,
					LatencyMs:    120.5,
				},
			},
			projectID: "test-project",
			location:  "us-central1",
		},
		{
			name:          "negative input tokens",
			inputTokens:   -100,
			outputTokens:  500,
			totalCost:     5.25,
			modelMetrics:  []VertexAIModelMetric{},
			projectID:     "test-project",
			location:      "us-central1",
			expectedError: "input tokens cannot be negative",
		},
		{
			name:          "negative output tokens",
			inputTokens:   1000,
			outputTokens:  -500,
			totalCost:     5.25,
			modelMetrics:  []VertexAIModelMetric{},
			projectID:     "test-project",
			location:      "us-central1",
			expectedError: "output tokens cannot be negative",
		},
		{
			name:          "negative total cost",
			inputTokens:   1000,
			outputTokens:  500,
			totalCost:     -5.25,
			modelMetrics:  []VertexAIModelMetric{},
			projectID:     "test-project",
			location:      "us-central1",
			expectedError: "total cost cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage, err := NewVertexAIUsage(
				tt.inputTokens,
				tt.outputTokens,
				tt.totalCost,
				tt.modelMetrics,
				tt.projectID,
				tt.location,
			)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("expected error %q, got nil", tt.expectedError)
					return
				}
				if err.Error() != tt.expectedError {
					t.Errorf("expected error %q, got %q", tt.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if usage.InputTokens() != tt.inputTokens {
				t.Errorf("expected input tokens %d, got %d", tt.inputTokens, usage.InputTokens())
			}

			if usage.OutputTokens() != tt.outputTokens {
				t.Errorf("expected output tokens %d, got %d", tt.outputTokens, usage.OutputTokens())
			}

			expectedTotal := tt.inputTokens + tt.outputTokens
			if usage.TotalTokens() != expectedTotal {
				t.Errorf("expected total tokens %d, got %d", expectedTotal, usage.TotalTokens())
			}

			if usage.TotalCost() != tt.totalCost {
				t.Errorf("expected total cost %f, got %f", tt.totalCost, usage.TotalCost())
			}

			if usage.ProjectID() != tt.projectID {
				t.Errorf("expected project ID %s, got %s", tt.projectID, usage.ProjectID())
			}

			if usage.Location() != tt.location {
				t.Errorf("expected location %s, got %s", tt.location, usage.Location())
			}

			if len(usage.ModelMetrics()) != len(tt.modelMetrics) {
				t.Errorf("expected %d model metrics, got %d", len(tt.modelMetrics), len(usage.ModelMetrics()))
			}

			// Check timestamp is recent
			if time.Since(usage.Timestamp()) > time.Minute {
				t.Error("timestamp should be recent")
			}
		})
	}
}

func TestVertexAIUsage_GetModelMetric(t *testing.T) {
	modelMetrics := []VertexAIModelMetric{
		{
			ModelID:      "gemini-pro",
			InputTokens:  1000,
			OutputTokens: 500,
			RequestCount: 10,
			Cost:         5.25,
			LatencyMs:    120.5,
		},
		{
			ModelID:      "palm-2",
			InputTokens:  500,
			OutputTokens: 300,
			RequestCount: 5,
			Cost:         2.10,
			LatencyMs:    90.0,
		},
	}

	usage, err := NewVertexAIUsage(1500, 800, 7.35, modelMetrics, "test-project", "us-central1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test existing model
	metric := usage.GetModelMetric("gemini-pro")
	if metric == nil {
		t.Fatal("expected to find gemini-pro metric")
	}
	if metric.ModelID != "gemini-pro" {
		t.Errorf("expected model ID gemini-pro, got %s", metric.ModelID)
	}
	if metric.InputTokens != 1000 {
		t.Errorf("expected input tokens 1000, got %d", metric.InputTokens)
	}

	// Test non-existing model
	metric = usage.GetModelMetric("non-existing")
	if metric != nil {
		t.Error("expected nil for non-existing model")
	}
}

func TestVertexAIUsage_GetTopModels(t *testing.T) {
	modelMetrics := []VertexAIModelMetric{
		{
			ModelID:      "gemini-pro",
			InputTokens:  1000,
			OutputTokens: 500,
			RequestCount: 10,
			Cost:         5.25,
			LatencyMs:    120.5,
		},
		{
			ModelID:      "palm-2",
			InputTokens:  2000,
			OutputTokens: 1000,
			RequestCount: 15,
			Cost:         10.50,
			LatencyMs:    150.0,
		},
		{
			ModelID:      "gemini-flash",
			InputTokens:  500,
			OutputTokens: 200,
			RequestCount: 8,
			Cost:         1.75,
			LatencyMs:    80.0,
		},
	}

	usage, err := NewVertexAIUsage(3500, 1700, 17.50, modelMetrics, "test-project", "us-central1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test getting top 2 models
	topModels := usage.GetTopModels(2)
	if len(topModels) != 2 {
		t.Errorf("expected 2 top models, got %d", len(topModels))
	}

	// Should be sorted by total tokens (palm-2: 3000, gemini-pro: 1500, gemini-flash: 700)
	if topModels[0].ModelID != "palm-2" {
		t.Errorf("expected first model to be palm-2, got %s", topModels[0].ModelID)
	}
	if topModels[1].ModelID != "gemini-pro" {
		t.Errorf("expected second model to be gemini-pro, got %s", topModels[1].ModelID)
	}

	// Test getting more models than available
	allModels := usage.GetTopModels(10)
	if len(allModels) != 3 {
		t.Errorf("expected 3 models when requesting 10, got %d", len(allModels))
	}

	// Test empty metrics
	emptyUsage, _ := NewVertexAIUsage(0, 0, 0, []VertexAIModelMetric{}, "test-project", "us-central1")
	emptyTop := emptyUsage.GetTopModels(5)
	if len(emptyTop) != 0 {
		t.Errorf("expected 0 models for empty usage, got %d", len(emptyTop))
	}
}

func TestVertexAIUsage_CalculateTotalRequests(t *testing.T) {
	modelMetrics := []VertexAIModelMetric{
		{ModelID: "gemini-pro", RequestCount: 10},
		{ModelID: "palm-2", RequestCount: 15},
		{ModelID: "gemini-flash", RequestCount: 8},
	}

	usage, err := NewVertexAIUsage(1000, 500, 5.25, modelMetrics, "test-project", "us-central1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	totalRequests := usage.CalculateTotalRequests()
	expected := int64(33) // 10 + 15 + 8
	if totalRequests != expected {
		t.Errorf("expected total requests %d, got %d", expected, totalRequests)
	}
}

func TestVertexAIUsage_CalculateAverageLatency(t *testing.T) {
	tests := []struct {
		name            string
		modelMetrics    []VertexAIModelMetric
		expectedLatency float64
	}{
		{
			name: "with requests",
			modelMetrics: []VertexAIModelMetric{
				{ModelID: "gemini-pro", RequestCount: 10, LatencyMs: 100.0},
				{ModelID: "palm-2", RequestCount: 20, LatencyMs: 200.0},
			},
			expectedLatency: (10*100.0 + 20*200.0) / 30, // (1000 + 4000) / 30 = 166.67
		},
		{
			name:            "empty metrics",
			modelMetrics:    []VertexAIModelMetric{},
			expectedLatency: 0,
		},
		{
			name: "zero requests",
			modelMetrics: []VertexAIModelMetric{
				{ModelID: "gemini-pro", RequestCount: 0, LatencyMs: 100.0},
			},
			expectedLatency: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage, err := NewVertexAIUsage(1000, 500, 5.25, tt.modelMetrics, "test-project", "us-central1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			avgLatency := usage.CalculateAverageLatency()
			if avgLatency != tt.expectedLatency {
				t.Errorf("expected average latency %f, got %f", tt.expectedLatency, avgLatency)
			}
		})
	}
}

func TestVertexAIUsage_IsEmpty(t *testing.T) {
	tests := []struct {
		name         string
		inputTokens  int64
		outputTokens int64
		totalCost    float64
		expectedEmpty bool
	}{
		{
			name:         "empty usage",
			inputTokens:  0,
			outputTokens: 0,
			totalCost:    0,
			expectedEmpty: true,
		},
		{
			name:         "non-empty tokens",
			inputTokens:  100,
			outputTokens: 0,
			totalCost:    0,
			expectedEmpty: false,
		},
		{
			name:         "non-empty cost",
			inputTokens:  0,
			outputTokens: 0,
			totalCost:    1.50,
			expectedEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage, err := NewVertexAIUsage(tt.inputTokens, tt.outputTokens, tt.totalCost, []VertexAIModelMetric{}, "test-project", "us-central1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			isEmpty := usage.IsEmpty()
			if isEmpty != tt.expectedEmpty {
				t.Errorf("expected isEmpty %t, got %t", tt.expectedEmpty, isEmpty)
			}
		})
	}
}

func TestVertexAIUsage_Validate(t *testing.T) {
	tests := []struct {
		name          string
		inputTokens   int64
		outputTokens  int64
		totalCost     float64
		modelMetrics  []VertexAIModelMetric
		expectedError string
	}{
		{
			name:         "valid usage",
			inputTokens:  1000,
			outputTokens: 500,
			totalCost:    5.25,
			modelMetrics: []VertexAIModelMetric{
				{
					ModelID:      "gemini-pro",
					InputTokens:  500,
					OutputTokens: 250,
					RequestCount: 10,
					Cost:         2.50,
					LatencyMs:    120.5,
				},
			},
		},
		{
			name:         "invalid model metric - empty model ID",
			inputTokens:  1000,
			outputTokens: 500,
			totalCost:    5.25,
			modelMetrics: []VertexAIModelMetric{
				{
					ModelID:      "",
					InputTokens:  500,
					OutputTokens: 250,
					RequestCount: 10,
					Cost:         2.50,
					LatencyMs:    120.5,
				},
			},
			expectedError: "model metric 0 validation failed: model ID cannot be empty",
		},
		{
			name:         "invalid model metric - negative cost",
			inputTokens:  1000,
			outputTokens: 500,
			totalCost:    5.25,
			modelMetrics: []VertexAIModelMetric{
				{
					ModelID:      "gemini-pro",
					InputTokens:  500,
					OutputTokens: 250,
					RequestCount: 10,
					Cost:         -2.50,
					LatencyMs:    120.5,
				},
			},
			expectedError: "model metric 0 validation failed: cost cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage, err := NewVertexAIUsage(tt.inputTokens, tt.outputTokens, tt.totalCost, tt.modelMetrics, "test-project", "us-central1")
			if err != nil {
				t.Fatalf("unexpected error during creation: %v", err)
			}

			err = usage.Validate()
			if tt.expectedError == "" {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error %q, got nil", tt.expectedError)
				} else if err.Error() != tt.expectedError {
					t.Errorf("expected error %q, got %q", tt.expectedError, err.Error())
				}
			}
		})
	}
}