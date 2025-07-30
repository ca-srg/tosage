package repository

import (
	"time"

	"github.com/ca-srg/tosage/domain/entity"
)

// VertexAIRepository defines the interface for retrieving Vertex AI usage data
type VertexAIRepository interface {
	// GetUsageMetrics retrieves Vertex AI usage metrics from Cloud Monitoring
	// for the specified time range and project ID
	GetUsageMetrics(projectID string, start, end time.Time) (*entity.VertexAIUsage, error)

	// GetDailyUsage retrieves aggregated usage for a specific date
	// Uses JST timezone for date boundaries
	GetDailyUsage(projectID string, date time.Time) (*entity.VertexAIUsage, error)

	// GetCurrentMonthUsage retrieves usage for the current month
	GetCurrentMonthUsage(projectID string) (*entity.VertexAIUsage, error)

	// CheckConnection verifies Google Cloud credentials and Cloud Monitoring access
	CheckConnection() error
}

// VertexAIConfig contains configuration for Vertex AI data collection
type VertexAIConfig struct {
	// Enabled indicates if Vertex AI tracking is enabled
	Enabled bool

	// ProjectID is the Google Cloud Project ID
	ProjectID string

	// ServiceAccountKeyPath is the path to the service account key file (optional)
	ServiceAccountKeyPath string

	// ServiceAccountKey is the service account key JSON content (optional)
	ServiceAccountKey string

	// CollectionInterval is how often to collect metrics
	CollectionInterval time.Duration
}

// DefaultVertexAIConfig returns the default configuration
func DefaultVertexAIConfig() *VertexAIConfig {
	return &VertexAIConfig{
		Enabled:            false, // Disabled by default for security
		ProjectID:          "",
		CollectionInterval: 15 * time.Minute,
	}
}
