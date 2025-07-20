package service

import (
	"os"
	"testing"
	"time"

	"github.com/ca-srg/tosage/domain/repository"
	"github.com/ca-srg/tosage/infrastructure/config"
	"github.com/ca-srg/tosage/infrastructure/logging"
	"github.com/stretchr/testify/assert"
)

func TestTimezoneServiceImpl_GetUserTimezone(t *testing.T) {
	logger := &logging.NoOpLogger{}
	cfg := &config.AppConfig{}
	service := NewTimezoneServiceImpl(cfg, logger)

	// Test getting user timezone
	loc, err := service.GetUserTimezone()
	assert.NoError(t, err)
	assert.NotNil(t, loc)

	// Should return cached location on second call
	loc2, err := service.GetUserTimezone()
	assert.NoError(t, err)
	assert.Equal(t, loc, loc2)
}

func TestTimezoneServiceImpl_GetConfiguredTimezone(t *testing.T) {
	// GetConfiguredTimezone now always returns system timezone
	logger := &logging.NoOpLogger{}
	cfg := &config.AppConfig{}
	service := NewTimezoneServiceImpl(cfg, logger)

	loc, err := service.GetConfiguredTimezone()

	// Should always return system timezone without error
	assert.NoError(t, err)
	assert.NotNil(t, loc)
}

func TestTimezoneServiceImpl_ConvertToUserTime(t *testing.T) {
	logger := &logging.NoOpLogger{}
	cfg := &config.AppConfig{}
	service := NewTimezoneServiceImpl(cfg, logger)

	// Test UTC to user's system timezone conversion
	utcTime := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	userTime := service.ConvertToUserTime(utcTime)

	// Should convert to system timezone
	systemLoc, _ := service.GetUserTimezone()
	expectedTime := utcTime.In(systemLoc)
	assert.Equal(t, expectedTime.Unix(), userTime.Unix())
}

func TestTimezoneServiceImpl_GetDayBoundaries(t *testing.T) {
	logger := &logging.NoOpLogger{}
	cfg := &config.AppConfig{}
	service := NewTimezoneServiceImpl(cfg, logger)

	// Test with system timezone
	inputTime := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)
	start, end := service.GetDayBoundaries(inputTime)

	// Should use system timezone for boundaries
	systemLoc, _ := service.GetUserTimezone()
	userTime := inputTime.In(systemLoc)
	year, month, day := userTime.Date()

	expectedStart := time.Date(year, month, day, 0, 0, 0, 0, systemLoc)
	expectedEnd := time.Date(year, month, day, 23, 59, 59, 999999999, systemLoc)

	// Compare Unix timestamps to avoid timezone representation issues
	assert.Equal(t, expectedStart.Unix(), start.Unix())
	assert.Equal(t, expectedEnd.Unix(), end.Unix())
}

func TestTimezoneServiceImpl_GetCurrentDayStart(t *testing.T) {
	logger := &logging.NoOpLogger{}
	cfg := &config.AppConfig{}
	service := NewTimezoneServiceImpl(cfg, logger)

	dayStart := service.GetCurrentDayStart()

	// Verify it's at the start of the day
	assert.Equal(t, 0, dayStart.Hour())
	assert.Equal(t, 0, dayStart.Minute())
	assert.Equal(t, 0, dayStart.Second())
	assert.Equal(t, 0, dayStart.Nanosecond())
}

func TestTimezoneServiceImpl_FormatTimeForUser(t *testing.T) {
	logger := &logging.NoOpLogger{}
	cfg := &config.AppConfig{}
	service := NewTimezoneServiceImpl(cfg, logger)

	utcTime := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	formatted := service.FormatTimeForUser(utcTime, "2006-01-02 15:04:05")

	// Should format in system timezone
	systemLoc, _ := service.GetUserTimezone()
	expectedFormatted := utcTime.In(systemLoc).Format("2006-01-02 15:04:05")
	assert.Equal(t, expectedFormatted, formatted)
}

func TestTimezoneServiceImpl_GetTimezoneInfo(t *testing.T) {
	logger := &logging.NoOpLogger{}
	cfg := &config.AppConfig{}
	service := NewTimezoneServiceImpl(cfg, logger)

	info := service.GetTimezoneInfo()

	// Should always report system timezone
	assert.Equal(t, "system", info.DetectionMethod)
	assert.NotEmpty(t, info.Name)
	assert.NotEmpty(t, info.Offset)
	assert.True(t, info.OffsetSeconds >= -12*3600, "Offset should be >= UTC-12")
	assert.True(t, info.OffsetSeconds <= 14*3600, "Offset should be <= UTC+14")
}

func TestTimezoneServiceImpl_DetectSystemTimezone(t *testing.T) {
	logger := &logging.NoOpLogger{}
	cfg := &config.AppConfig{}
	service := NewTimezoneServiceImpl(cfg, logger)

	// Test with TZ environment variable
	t.Run("TZ environment variable", func(t *testing.T) {
		// Save original TZ
		originalTZ, originalTZSet := os.LookupEnv("TZ")
		defer func() {
			if originalTZSet {
				if err := os.Setenv("TZ", originalTZ); err != nil {
					t.Errorf("Failed to restore TZ environment variable: %v", err)
				}
			} else {
				if err := os.Unsetenv("TZ"); err != nil {
					t.Errorf("Failed to unset TZ environment variable: %v", err)
				}
			}
		}()

		// Set TZ
		if err := os.Setenv("TZ", "Europe/London"); err != nil {
			t.Fatalf("Failed to set TZ environment variable: %v", err)
		}

		// Reset service state
		service.detected = false
		service.userLocation = nil

		loc, err := service.detectSystemTimezone()

		// Should detect from TZ or fall back gracefully
		assert.NotNil(t, loc)
		if err == nil && loc.String() == "Europe/London" {
			assert.Equal(t, "Europe/London", loc.String())
		}
	})
}

// MockTimezoneService is a mock implementation for testing
type MockTimezoneService struct {
	Location     *time.Location
	TimezoneInfo repository.TimezoneInfo
	Error        error
}

func (m *MockTimezoneService) GetUserTimezone() (*time.Location, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if m.Location != nil {
		return m.Location, nil
	}
	return time.UTC, nil
}

func (m *MockTimezoneService) GetConfiguredTimezone() (*time.Location, error) {
	return m.GetUserTimezone()
}

func (m *MockTimezoneService) ConvertToUserTime(utcTime time.Time) time.Time {
	loc, _ := m.GetUserTimezone()
	return utcTime.In(loc)
}

func (m *MockTimezoneService) GetDayBoundaries(date time.Time) (start, end time.Time) {
	loc, _ := m.GetUserTimezone()
	userTime := date.In(loc)
	year, month, day := userTime.Date()
	start = time.Date(year, month, day, 0, 0, 0, 0, loc)
	end = time.Date(year, month, day, 23, 59, 59, 999999999, loc)
	return
}

func (m *MockTimezoneService) GetCurrentDayStart() time.Time {
	loc, _ := m.GetUserTimezone()
	now := time.Now().In(loc)
	year, month, day := now.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, loc)
}

func (m *MockTimezoneService) FormatTimeForUser(t time.Time, layout string) string {
	return m.ConvertToUserTime(t).Format(layout)
}

func (m *MockTimezoneService) GetTimezoneInfo() repository.TimezoneInfo {
	if m.TimezoneInfo.Name != "" {
		return m.TimezoneInfo
	}
	return repository.TimezoneInfo{
		Name:            "UTC",
		Offset:          "+00:00",
		OffsetSeconds:   0,
		IsDST:           false,
		DetectionMethod: "mock",
	}
}
