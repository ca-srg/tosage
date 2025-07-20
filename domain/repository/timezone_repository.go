package repository

import (
	"time"
)

// TimezoneService defines the interface for timezone-related operations
type TimezoneService interface {
	// GetUserTimezone returns the user's local timezone
	GetUserTimezone() (*time.Location, error)

	// GetConfiguredTimezone returns configured timezone or user's local timezone
	GetConfiguredTimezone() (*time.Location, error)

	// ConvertToUserTime converts UTC time to user's local time
	ConvertToUserTime(utcTime time.Time) time.Time

	// GetDayBoundaries returns start and end of day in user's timezone
	GetDayBoundaries(date time.Time) (start, end time.Time)

	// GetCurrentDayStart returns start of current day in user's timezone
	GetCurrentDayStart() time.Time

	// FormatTimeForUser formats time according to user's timezone
	FormatTimeForUser(t time.Time, layout string) string

	// GetTimezoneInfo returns timezone information for logging/metrics
	GetTimezoneInfo() TimezoneInfo
}

// TimezoneInfo contains timezone information for logging and metrics
type TimezoneInfo struct {
	// Name is the timezone name (e.g., "America/New_York", "Asia/Tokyo")
	Name string

	// Offset is the UTC offset in the format "+09:00" or "-05:00"
	Offset string

	// OffsetSeconds is the offset from UTC in seconds
	OffsetSeconds int

	// IsDST indicates whether daylight saving time is currently active
	IsDST bool

	// DetectionMethod indicates how the timezone was determined
	// Values: "system", "config", "fallback"
	DetectionMethod string
}