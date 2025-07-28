package usecase

import (
	"time"

	"github.com/ca-srg/tosage/domain/entity"
)

// BedrockService defines the interface for Bedrock-related operations
type BedrockService interface {
	// GetCurrentUsage retrieves the current Bedrock usage statistics
	// for all configured regions
	GetCurrentUsage() (*entity.BedrockUsage, error)

	// GetUsageForRegion retrieves usage statistics for a specific region
	GetUsageForRegion(region string) (*entity.BedrockUsage, error)

	// GetDailyUsage retrieves aggregated usage for a specific date
	// Uses JST timezone for date boundaries
	GetDailyUsage(date time.Time) (*entity.BedrockUsage, error)

	// GetCurrentMonthUsage retrieves usage for the current month
	GetCurrentMonthUsage() (*entity.BedrockUsage, error)

	// IsEnabled checks if Bedrock tracking is enabled in configuration
	IsEnabled() bool

	// CheckConnection verifies AWS credentials and CloudWatch access
	CheckConnection() error

	// GetConfiguredRegions returns the list of configured regions
	GetConfiguredRegions() []string

	// GetAvailableRegions returns regions with Bedrock activity
	GetAvailableRegions() ([]string, error)
}
