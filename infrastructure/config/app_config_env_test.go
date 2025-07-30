package config

import (
	"os"
	"testing"
)

func TestSplitCommaSeparated(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single region",
			input:    "us-east-1",
			expected: []string{"us-east-1"},
		},
		{
			name:     "multiple regions without spaces",
			input:    "us-east-1,us-west-2,eu-west-1",
			expected: []string{"us-east-1", "us-west-2", "eu-west-1"},
		},
		{
			name:     "multiple regions with spaces",
			input:    "us-east-1, us-west-2,  eu-west-1",
			expected: []string{"us-east-1", "us-west-2", "eu-west-1"},
		},
		{
			name:     "trailing comma",
			input:    "us-east-1,us-west-2,",
			expected: []string{"us-east-1", "us-west-2"},
		},
		{
			name:     "only commas",
			input:    ",,,",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitCommaSeparated(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d elements, got %d", len(tt.expected), len(result))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("element %d: expected %s, got %s", i, tt.expected[i], v)
				}
			}
		})
	}
}

func TestSlicesEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{
			name:     "both empty",
			a:        []string{},
			b:        []string{},
			expected: true,
		},
		{
			name:     "same elements",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "b", "c"},
			expected: true,
		},
		{
			name:     "different length",
			a:        []string{"a", "b"},
			b:        []string{"a", "b", "c"},
			expected: false,
		},
		{
			name:     "different elements",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "b", "d"},
			expected: false,
		},
		{
			name:     "different order",
			a:        []string{"a", "b", "c"},
			b:        []string{"c", "b", "a"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := slicesEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBedrockRegionsEnvironmentVariable(t *testing.T) {
	// Save original env
	originalRegions := os.Getenv("TOSAGE_BEDROCK_REGIONS")
	defer func() { _ = os.Setenv("TOSAGE_BEDROCK_REGIONS", originalRegions) }()

	// Set test env
	testRegions := "us-east-1, us-west-2, ap-northeast-1"
	if err := os.Setenv("TOSAGE_BEDROCK_REGIONS", testRegions); err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
	}

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	expected := []string{"us-east-1", "us-west-2", "ap-northeast-1"}
	if !slicesEqual(config.Bedrock.Regions, expected) {
		t.Errorf("Expected regions %v, got %v", expected, config.Bedrock.Regions)
	}

	// Check config source
	if source, ok := config.ConfigSources["Bedrock.Regions"]; !ok || source != SourceEnvironment {
		t.Errorf("Expected Bedrock.Regions source to be SourceEnvironment, got %v", source)
	}
}

// TestVertexAILocationsEnvironmentVariable is removed as Locations field is no longer supported
