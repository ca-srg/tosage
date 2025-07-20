package service

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ca-srg/tosage/domain"
	"github.com/ca-srg/tosage/domain/repository"
	"github.com/ca-srg/tosage/infrastructure/config"
)

// TimezoneServiceImpl implements the TimezoneService interface
type TimezoneServiceImpl struct {
	config       *config.AppConfig
	logger       domain.Logger
	locationMu   sync.RWMutex
	userLocation *time.Location
	detectionMu  sync.Mutex
	detected     bool
}

// NewTimezoneServiceImpl creates a new instance of TimezoneServiceImpl
func NewTimezoneServiceImpl(config *config.AppConfig, logger domain.Logger) *TimezoneServiceImpl {
	return &TimezoneServiceImpl{
		config: config,
		logger: logger,
	}
}

// GetUserTimezone returns the user's local timezone
func (s *TimezoneServiceImpl) GetUserTimezone() (*time.Location, error) {
	s.locationMu.RLock()
	if s.userLocation != nil {
		s.locationMu.RUnlock()
		return s.userLocation, nil
	}
	s.locationMu.RUnlock()

	return s.detectSystemTimezone()
}

// GetConfiguredTimezone returns configured timezone or user's local timezone
func (s *TimezoneServiceImpl) GetConfiguredTimezone() (*time.Location, error) {
	// Always use system timezone
	return s.GetUserTimezone()
}

// ConvertToUserTime converts UTC time to user's local time
func (s *TimezoneServiceImpl) ConvertToUserTime(utcTime time.Time) time.Time {
	loc, err := s.GetConfiguredTimezone()
	if err != nil {
		s.logger.Warn(context.Background(), "Failed to get user timezone, using UTC",
			domain.NewField("error", err.Error()))
		return utcTime
	}
	return utcTime.In(loc)
}

// GetDayBoundaries returns start and end of day in user's timezone
func (s *TimezoneServiceImpl) GetDayBoundaries(date time.Time) (start, end time.Time) {
	loc, err := s.GetConfiguredTimezone()
	if err != nil {
		s.logger.Warn(context.Background(), "Failed to get user timezone, using UTC",
			domain.NewField("error", err.Error()))
		loc = time.UTC
	}

	// Convert to user's timezone
	userTime := date.In(loc)
	year, month, day := userTime.Date()

	// Start of day (00:00:00) in user's timezone
	start = time.Date(year, month, day, 0, 0, 0, 0, loc)

	// End of day (23:59:59.999999999) in user's timezone
	end = time.Date(year, month, day, 23, 59, 59, 999999999, loc)

	return start, end
}

// GetCurrentDayStart returns start of current day in user's timezone
func (s *TimezoneServiceImpl) GetCurrentDayStart() time.Time {
	loc, err := s.GetConfiguredTimezone()
	if err != nil {
		s.logger.Warn(context.Background(), "Failed to get user timezone, using UTC",
			domain.NewField("error", err.Error()))
		loc = time.UTC
	}

	now := time.Now().In(loc)
	year, month, day := now.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, loc)
}

// FormatTimeForUser formats time according to user's timezone
func (s *TimezoneServiceImpl) FormatTimeForUser(t time.Time, layout string) string {
	userTime := s.ConvertToUserTime(t)
	return userTime.Format(layout)
}

// GetTimezoneInfo returns timezone information for logging/metrics
func (s *TimezoneServiceImpl) GetTimezoneInfo() repository.TimezoneInfo {
	loc, err := s.GetConfiguredTimezone()
	if err != nil {
		// Return UTC info if timezone detection fails
		return repository.TimezoneInfo{
			Name:            "UTC",
			Offset:          "+00:00",
			OffsetSeconds:   0,
			IsDST:           false,
			DetectionMethod: "fallback",
		}
	}

	// Get current time in the timezone
	now := time.Now().In(loc)
	_, offset := now.Zone()

	// Format offset as +HH:MM or -HH:MM
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	hours := offset / 3600
	minutes := (offset % 3600) / 60
	offsetStr := fmt.Sprintf("%s%02d:%02d", sign, hours, minutes)

	return repository.TimezoneInfo{
		Name:            loc.String(),
		Offset:          offsetStr,
		OffsetSeconds:   offset,
		IsDST:           now.IsDST(),
		DetectionMethod: "system",
	}
}

// detectSystemTimezone detects the system timezone
func (s *TimezoneServiceImpl) detectSystemTimezone() (*time.Location, error) {
	s.detectionMu.Lock()
	defer s.detectionMu.Unlock()

	// Check if already detected
	s.locationMu.RLock()
	if s.detected && s.userLocation != nil {
		s.locationMu.RUnlock()
		return s.userLocation, nil
	}
	s.locationMu.RUnlock()

	// Try multiple detection methods
	var loc *time.Location
	var err error

	// Method 1: Use time.Local (most reliable)
	loc = time.Local
	if loc != nil && loc.String() != "Local" {
		s.logger.Debug(context.Background(), "Detected timezone using time.Local",
			domain.NewField("timezone", loc.String()))
		s.setUserLocation(loc)
		return loc, nil
	}

	// Method 2: Check TZ environment variable
	if tzEnv := os.Getenv("TZ"); tzEnv != "" {
		loc, err = time.LoadLocation(tzEnv)
		if err == nil {
			s.logger.Debug(context.Background(), "Detected timezone from TZ environment variable",
				domain.NewField("timezone", loc.String()))
			s.setUserLocation(loc)
			return loc, nil
		}
		s.logger.Warn(context.Background(), "Failed to load timezone from TZ environment variable",
			domain.NewField("TZ", tzEnv),
			domain.NewField("error", err.Error()))
	}

	// Method 3: Read /etc/localtime symlink (Unix/Linux)
	if linkPath, err := os.Readlink("/etc/localtime"); err == nil {
		// Extract timezone name from path (e.g., /usr/share/zoneinfo/America/New_York)
		parts := strings.Split(linkPath, "/zoneinfo/")
		if len(parts) > 1 {
			tzName := parts[1]
			loc, err = time.LoadLocation(tzName)
			if err == nil {
				s.logger.Debug(context.Background(), "Detected timezone from /etc/localtime",
					domain.NewField("timezone", loc.String()))
				s.setUserLocation(loc)
				return loc, nil
			}
		}
	}

	// Method 4: Try to get timezone from system (macOS specific)
	if output, err := s.getSystemTimezone(); err == nil && output != "" {
		loc, err = time.LoadLocation(output)
		if err == nil {
			s.logger.Debug(context.Background(), "Detected timezone from system command",
				domain.NewField("timezone", loc.String()))
			s.setUserLocation(loc)
			return loc, nil
		}
	}

	// Fallback to UTC
	s.logger.Warn(context.Background(), "Failed to detect system timezone, using UTC as fallback")
	loc = time.UTC
	s.setUserLocation(loc)
	return loc, domain.ErrTimezoneDetection("UTC")
}

// getSystemTimezone tries to get timezone from system commands
func (s *TimezoneServiceImpl) getSystemTimezone() (string, error) {
	// This is a placeholder for system-specific timezone detection
	// On macOS, we could use: systemsetup -gettimezone
	// On Linux, we could read /etc/timezone
	// For now, return empty to use other detection methods
	return "", fmt.Errorf("system timezone detection not implemented")
}

// setUserLocation sets the user location with proper locking
func (s *TimezoneServiceImpl) setUserLocation(loc *time.Location) {
	s.locationMu.Lock()
	defer s.locationMu.Unlock()
	s.userLocation = loc
	s.detected = true
}
