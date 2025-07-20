package entity

import (
	"fmt"
	"time"
)

// CursorUsage represents Cursor API usage data
type CursorUsage struct {
	premiumRequests   PremiumRequestsInfo
	usageBasedPricing UsageBasedPricingInfo
	teamInfo          *TeamInfo
	timestamp         time.Time
}

// PremiumRequestsInfo contains information about premium (GPT-4) requests
type PremiumRequestsInfo struct {
	Current      int
	Limit        int
	StartOfMonth string
}

// UsageBasedPricingInfo contains usage-based pricing information
type UsageBasedPricingInfo struct {
	CurrentMonth MonthlyUsage
	LastMonth    MonthlyUsage
}

// MonthlyUsage represents usage data for a specific month
type MonthlyUsage struct {
	Month            int
	Year             int
	Items            []UsageItem
	MidMonthPayment  float64
	HasUnpaidInvoice bool
}

// UsageItem represents a single usage item
type UsageItem struct {
	RequestCount   int
	Model          string
	CostPerRequest float64
	TotalCost      float64
	Description    string
	IsDiscounted   bool
	IsToolCall     bool
}

// TeamInfo contains team membership information
type TeamInfo struct {
	TeamID   int
	UserID   int
	TeamName string
	Role     string
}

// NewCursorUsage creates a new CursorUsage instance
func NewCursorUsage(
	premiumRequests PremiumRequestsInfo,
	usageBasedPricing UsageBasedPricingInfo,
	teamInfo *TeamInfo,
) *CursorUsage {
	return &CursorUsage{
		premiumRequests:   premiumRequests,
		usageBasedPricing: usageBasedPricing,
		teamInfo:          teamInfo,
		timestamp:         time.Now(),
	}
}

// PremiumRequests returns the premium requests information
func (u *CursorUsage) PremiumRequests() PremiumRequestsInfo {
	return u.premiumRequests
}

// UsageBasedPricing returns the usage-based pricing information
func (u *CursorUsage) UsageBasedPricing() UsageBasedPricingInfo {
	return u.usageBasedPricing
}

// TeamInfo returns the team information (may be nil)
func (u *CursorUsage) TeamInfo() *TeamInfo {
	return u.teamInfo
}

// IsTeamMember checks if the user is a team member
func (u *CursorUsage) IsTeamMember() bool {
	return u.teamInfo != nil
}

// Timestamp returns when this usage data was created
func (u *CursorUsage) Timestamp() time.Time {
	return u.timestamp
}

// PremiumRequestsPercentage calculates the percentage of premium requests used
func (u *CursorUsage) PremiumRequestsPercentage() float64 {
	if u.premiumRequests.Limit == 0 {
		return 0
	}
	return float64(u.premiumRequests.Current) / float64(u.premiumRequests.Limit) * 100
}

// CurrentMonthTotalCost calculates the total cost for the current month
func (u *CursorUsage) CurrentMonthTotalCost() float64 {
	total := 0.0
	for _, item := range u.usageBasedPricing.CurrentMonth.Items {
		total += item.TotalCost
	}
	// Subtract mid-month payment as it's already been paid
	total -= u.usageBasedPricing.CurrentMonth.MidMonthPayment
	return total
}

// LastMonthTotalCost calculates the total cost for the last month
func (u *CursorUsage) LastMonthTotalCost() float64 {
	total := 0.0
	for _, item := range u.usageBasedPricing.LastMonth.Items {
		total += item.TotalCost
	}
	return total
}

// Validate checks if the usage data is valid
func (u *CursorUsage) Validate() error {
	if u.premiumRequests.Current < 0 {
		return fmt.Errorf("premium requests current cannot be negative")
	}
	if u.premiumRequests.Limit < 0 {
		return fmt.Errorf("premium requests limit cannot be negative")
	}
	if u.premiumRequests.Current > u.premiumRequests.Limit && u.premiumRequests.Limit != 0 {
		return fmt.Errorf("premium requests current (%d) exceeds limit (%d)",
			u.premiumRequests.Current, u.premiumRequests.Limit)
	}

	// Validate monthly usage data
	if err := u.validateMonthlyUsage(u.usageBasedPricing.CurrentMonth); err != nil {
		return fmt.Errorf("current month validation failed: %w", err)
	}
	if err := u.validateMonthlyUsage(u.usageBasedPricing.LastMonth); err != nil {
		return fmt.Errorf("last month validation failed: %w", err)
	}

	return nil
}

// validateMonthlyUsage validates monthly usage data
func (u *CursorUsage) validateMonthlyUsage(monthly MonthlyUsage) error {
	if monthly.Month < 1 || monthly.Month > 12 {
		return fmt.Errorf("invalid month: %d", monthly.Month)
	}
	if monthly.Year < 2020 {
		return fmt.Errorf("invalid year: %d", monthly.Year)
	}
	if monthly.MidMonthPayment < 0 {
		return fmt.Errorf("mid-month payment cannot be negative")
	}

	for i, item := range monthly.Items {
		if item.RequestCount < 0 {
			return fmt.Errorf("item %d: request count cannot be negative", i)
		}
		if item.CostPerRequest < 0 {
			return fmt.Errorf("item %d: cost per request cannot be negative", i)
		}
		if item.TotalCost < 0 {
			return fmt.Errorf("item %d: total cost cannot be negative", i)
		}
	}

	return nil
}
