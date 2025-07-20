package repository

import (
	"github.com/ca-srg/tosage/domain/entity"
	"github.com/ca-srg/tosage/domain/valueobject"
)

// CursorAPIRepository defines the interface for communicating with the Cursor API
type CursorAPIRepository interface {
	// GetUsageStats retrieves current usage statistics from the Cursor API
	GetUsageStats(token *valueobject.CursorToken) (*entity.CursorUsage, error)

	// GetUsageLimit retrieves the current usage limit settings
	GetUsageLimit(token *valueobject.CursorToken, teamID *int) (*UsageLimitInfo, error)

	// CheckUsageBasedStatus checks if usage-based pricing is enabled
	CheckUsageBasedStatus(token *valueobject.CursorToken, teamID *int) (*UsageBasedStatus, error)

	// GetAggregatedTokenUsage retrieves aggregated token usage from JST 00:00 to current time
	GetAggregatedTokenUsage(token *valueobject.CursorToken) (int64, error)
}

// UsageLimitInfo contains information about usage limits
type UsageLimitInfo struct {
	HardLimit           *float64
	HardLimitPerUser    *float64
	NoUsageBasedAllowed bool
}

// UsageBasedStatus contains information about usage-based pricing status
type UsageBasedStatus struct {
	IsEnabled bool
	Limit     *float64
}
