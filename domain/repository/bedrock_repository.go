package repository

import (
	"time"

	"github.com/ca-srg/tosage/domain/entity"
)

// BedrockRepository defines the interface for retrieving Bedrock usage data
type BedrockRepository interface {
	// GetUsageMetrics retrieves Bedrock usage metrics from CloudWatch
	// for the specified time range and region
	GetUsageMetrics(region string, start, end time.Time) (*entity.BedrockUsage, error)

	// GetDailyUsage retrieves aggregated usage for a specific date
	// Uses JST timezone for date boundaries
	GetDailyUsage(region string, date time.Time) (*entity.BedrockUsage, error)

	// GetCurrentMonthUsage retrieves usage for the current month
	GetCurrentMonthUsage(region string) (*entity.BedrockUsage, error)

	// CheckConnection verifies AWS credentials and CloudWatch access
	CheckConnection() error

	// ListAvailableRegions returns regions with Bedrock activity
	ListAvailableRegions() ([]string, error)
}

// BedrockConfig contains configuration for Bedrock data collection
type BedrockConfig struct {
	// Enabled indicates if Bedrock tracking is enabled
	Enabled bool

	// Regions is the list of AWS regions to monitor
	Regions []string

	// AWSProfile is the AWS profile to use (optional)
	AWSProfile string

	// AssumeRoleARN is the ARN of the role to assume (optional)
	AssumeRoleARN string

	// CollectionInterval is how often to collect metrics
	CollectionInterval time.Duration
}

// DefaultBedrockConfig returns the default configuration
func DefaultBedrockConfig() *BedrockConfig {
	return &BedrockConfig{
		Enabled:            false, // Disabled by default for security
		Regions:            []string{"us-east-1", "us-west-2"},
		CollectionInterval: 15 * time.Minute,
	}
}
