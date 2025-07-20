package usecase

import (
	"github.com/ca-srg/tosage/domain/entity"
	"github.com/ca-srg/tosage/domain/repository"
)

// CursorService defines the interface for Cursor-related operations
type CursorService interface {
	// GetCurrentUsage retrieves the current Cursor usage statistics
	GetCurrentUsage() (*entity.CursorUsage, error)

	// GetUsageLimit retrieves the current usage limit settings
	GetUsageLimit() (*repository.UsageLimitInfo, error)

	// IsUsageBasedPricingEnabled checks if usage-based pricing is enabled
	IsUsageBasedPricingEnabled() (bool, error)

	// GetAggregatedTokenUsage retrieves aggregated token usage from JST 00:00 to current time
	GetAggregatedTokenUsage() (int64, error)
}
