package usecase

import (
	"time"

	"github.com/ca-srg/tosage/domain/entity"
)

// VertexAIService defines the interface for Vertex AI-related operations
type VertexAIService interface {
	// GetCurrentUsage retrieves the current Vertex AI usage statistics
	// for all configured projects and locations
	GetCurrentUsage() (*entity.VertexAIUsage, error)

	// GetUsageForProject retrieves usage statistics for a specific project and location
	GetUsageForProject(projectID, location string) (*entity.VertexAIUsage, error)

	// GetDailyUsage retrieves aggregated usage for a specific date
	// Uses JST timezone for date boundaries
	GetDailyUsage(date time.Time) (*entity.VertexAIUsage, error)

	// GetCurrentMonthUsage retrieves usage for the current month
	GetCurrentMonthUsage() (*entity.VertexAIUsage, error)

	// IsEnabled checks if Vertex AI tracking is enabled in configuration
	IsEnabled() bool

	// CheckConnection verifies Google Cloud credentials and Cloud Monitoring access
	CheckConnection() error

	// GetConfiguredProjects returns the list of configured project IDs
	GetConfiguredProjects() []string

	// GetConfiguredLocations returns the list of configured locations
	GetConfiguredLocations() []string

	// GetAvailableLocations returns locations with Vertex AI activity for a project
	GetAvailableLocations(projectID string) ([]string, error)
}