package impl

import (
	"testing"
	"time"

	"github.com/ca-srg/tosage/domain/entity"
	"github.com/ca-srg/tosage/domain/repository"
	"github.com/ca-srg/tosage/domain/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockCcRepository is a mock implementation of CcRepository
type MockCcRepository struct {
	mock.Mock
}

func (m *MockCcRepository) FindAll() ([]*entity.CcEntry, error) {
	args := m.Called()
	return args.Get(0).([]*entity.CcEntry), args.Error(1)
}

func (m *MockCcRepository) FindByDateRange(start, end time.Time) ([]*entity.CcEntry, error) {
	args := m.Called(start, end)
	return args.Get(0).([]*entity.CcEntry), args.Error(1)
}

func (m *MockCcRepository) FindByProject(projectPath string) ([]*entity.CcEntry, error) {
	args := m.Called(projectPath)
	return args.Get(0).([]*entity.CcEntry), args.Error(1)
}

func (m *MockCcRepository) FindByProjectAndDateRange(projectPath string, start, end time.Time) ([]*entity.CcEntry, error) {
	args := m.Called(projectPath, start, end)
	return args.Get(0).([]*entity.CcEntry), args.Error(1)
}

func (m *MockCcRepository) GetDateRange() (start, end time.Time, err error) {
	args := m.Called()
	return args.Get(0).(time.Time), args.Get(1).(time.Time), args.Error(2)
}

func (m *MockCcRepository) CountAll() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

func (m *MockCcRepository) CountByDateRange(start, end time.Time) (int, error) {
	args := m.Called(start, end)
	return args.Int(0), args.Error(1)
}

func (m *MockCcRepository) FindByID(id string) (*entity.CcEntry, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.CcEntry), args.Error(1)
}

func (m *MockCcRepository) FindByDate(date time.Time) ([]*entity.CcEntry, error) {
	args := m.Called(date)
	return args.Get(0).([]*entity.CcEntry), args.Error(1)
}

func (m *MockCcRepository) FindBySession(sessionID string) ([]*entity.CcEntry, error) {
	args := m.Called(sessionID)
	return args.Get(0).([]*entity.CcEntry), args.Error(1)
}

func (m *MockCcRepository) FindByModel(model string) ([]*entity.CcEntry, error) {
	args := m.Called(model)
	return args.Get(0).([]*entity.CcEntry), args.Error(1)
}

func (m *MockCcRepository) ExistsByID(id string) (bool, error) {
	args := m.Called(id)
	return args.Bool(0), args.Error(1)
}

func (m *MockCcRepository) ExistsByMessageID(messageID string) (bool, error) {
	args := m.Called(messageID)
	return args.Bool(0), args.Error(1)
}

func (m *MockCcRepository) ExistsByRequestID(requestID string) (bool, error) {
	args := m.Called(requestID)
	return args.Bool(0), args.Error(1)
}

func (m *MockCcRepository) GetDistinctProjects() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockCcRepository) GetDistinctModels() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockCcRepository) GetDistinctSessions() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockCcRepository) Save(entry *entity.CcEntry) error {
	args := m.Called(entry)
	return args.Error(0)
}

func (m *MockCcRepository) SaveAll(entries []*entity.CcEntry) error {
	args := m.Called(entries)
	return args.Error(0)
}

func (m *MockCcRepository) DeleteByID(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockCcRepository) DeleteByDateRange(start, end time.Time) error {
	args := m.Called(start, end)
	return args.Error(0)
}

// MockTimezoneService is a mock implementation for testing
type MockTimezoneService struct {
	Location        *time.Location
	TimezoneInfo    repository.TimezoneInfo
	Error           error
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

func TestCcServiceImpl_CalculateDailyTokensInUserTimezone(t *testing.T) {
	// Setup
	mockRepo := new(MockCcRepository)
	mockTimezoneService := &MockTimezoneService{
		Location: time.FixedZone("EST", -5*3600), // EST timezone
	}
	
	service := NewCcServiceImpl(mockRepo, mockTimezoneService)

	// Test data
	tokenStats := valueobject.NewTokenStats(100, 200, 50, 25)
	
	// Create test entries
	// Entry at EST 2024-01-15 23:00 (UTC 2024-01-16 04:00)
	entry1, _ := entity.NewCcEntry(
		"id1",
		time.Date(2024, 1, 16, 4, 0, 0, 0, time.UTC),
		"session1",
		"/project1",
		"gpt-4",
		tokenStats,
		"1.0",
		"msg1",
		"req1",
	)
	
	// Entry at EST 2024-01-15 10:00 (UTC 2024-01-15 15:00)
	entry2, _ := entity.NewCcEntry(
		"id2",
		time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC),
		"session2",
		"/project2",
		"gpt-4",
		tokenStats,
		"1.0",
		"msg2",
		"req2",
	)

	// Mock repository behavior
	// When querying for EST 2024-01-15, it should convert to UTC range
	estDate := time.Date(2024, 1, 15, 12, 0, 0, 0, time.FixedZone("EST", -5*3600))
	expectedStart := time.Date(2024, 1, 15, 5, 0, 0, 0, time.UTC) // EST 00:00 = UTC 05:00
	expectedEnd := time.Date(2024, 1, 16, 4, 59, 59, 999999999, time.UTC) // EST 23:59:59 = UTC 04:59:59 next day
	
	mockRepo.On("FindByDateRange", 
		mock.MatchedBy(func(t time.Time) bool { return t.Unix() == expectedStart.Unix() }),
		mock.MatchedBy(func(t time.Time) bool { return t.Unix() == expectedEnd.Unix() }),
	).Return([]*entity.CcEntry{entry1, entry2}, nil)

	// Execute
	totalTokens, err := service.CalculateDailyTokensInUserTimezone(estDate)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 750, totalTokens) // 375 + 375
	mockRepo.AssertExpectations(t)
}

func TestCcServiceImpl_CalculateTodayTokensInUserTimezone(t *testing.T) {
	// Setup
	mockRepo := new(MockCcRepository)
	jst := time.FixedZone("JST", 9*3600)
	mockTimezoneService := &MockTimezoneService{
		Location: jst,
	}
	
	service := NewCcServiceImpl(mockRepo, mockTimezoneService)

	// Test data
	tokenStats := valueobject.NewTokenStats(100, 200, 50, 25)
	
	// Create test entry for today in JST
	now := time.Now()
	todayStart, todayEnd := mockTimezoneService.GetDayBoundaries(now)
	
	entry, _ := entity.NewCcEntry(
		"id1",
		now,
		"session1",
		"/project1",
		"gpt-4",
		tokenStats,
		"1.0",
		"msg1",
		"req1",
	)

	// Mock repository behavior
	mockRepo.On("FindByDateRange", 
		mock.MatchedBy(func(t time.Time) bool { 
			return t.Unix() >= todayStart.Unix()-1 && t.Unix() <= todayStart.Unix()+1 
		}),
		mock.MatchedBy(func(t time.Time) bool { 
			return t.Unix() >= todayEnd.Unix()-1 && t.Unix() <= todayEnd.Unix()+1 
		}),
	).Return([]*entity.CcEntry{entry}, nil)

	// Execute
	totalTokens, err := service.CalculateTodayTokensInUserTimezone()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 375, totalTokens)
	mockRepo.AssertExpectations(t)
}

func TestCcServiceImpl_GetDateRangeInUserTimezone(t *testing.T) {
	// Setup
	mockRepo := new(MockCcRepository)
	pst := time.FixedZone("PST", -8*3600)
	mockTimezoneService := &MockTimezoneService{
		Location: pst,
	}
	
	service := NewCcServiceImpl(mockRepo, mockTimezoneService)

	// Mock repository to return UTC times
	utcStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	utcEnd := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
	mockRepo.On("GetDateRange").Return(utcStart, utcEnd, nil)

	// Execute
	start, end, err := service.GetDateRangeInUserTimezone()

	// Assert
	require.NoError(t, err)
	
	// Verify times are converted to PST
	assert.Equal(t, pst, start.Location())
	assert.Equal(t, pst, end.Location())
	
	// UTC 2024-01-01 00:00 = PST 2023-12-31 16:00
	assert.Equal(t, 2023, start.Year())
	assert.Equal(t, time.December, start.Month())
	assert.Equal(t, 31, start.Day())
	assert.Equal(t, 16, start.Hour())
	
	// UTC 2024-01-31 23:59 = PST 2024-01-31 15:59
	assert.Equal(t, 2024, end.Year())
	assert.Equal(t, time.January, end.Month())
	assert.Equal(t, 31, end.Day())
	assert.Equal(t, 15, end.Hour())
	assert.Equal(t, 59, end.Minute())
	
	mockRepo.AssertExpectations(t)
}

func TestCcServiceImpl_TimezoneServiceFallback(t *testing.T) {
	// Test that service works without timezone service (backward compatibility)
	mockRepo := new(MockCcRepository)
	service := NewCcServiceImpl(mockRepo, nil) // No timezone service

	// Test data
	tokenStats := valueobject.NewTokenStats(100, 200, 50, 25)
	date := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	
	entry, _ := entity.NewCcEntry(
		"id1",
		date,
		"session1",
		"/project1",
		"gpt-4",
		tokenStats,
		"1.0",
		"msg1",
		"req1",
	)

	// Mock repository behavior - should use date as-is
	expectedStart := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	expectedEnd := expectedStart.Add(24 * time.Hour)
	
	mockRepo.On("FindByDateRange", 
		mock.MatchedBy(func(t time.Time) bool { return t.Equal(expectedStart) }),
		mock.MatchedBy(func(t time.Time) bool { return t.Equal(expectedEnd) }),
	).Return([]*entity.CcEntry{entry}, nil)

	// Execute methods without timezone service
	t.Run("CalculateDailyTokensInUserTimezone falls back", func(t *testing.T) {
		totalTokens, err := service.CalculateDailyTokensInUserTimezone(date)
		require.NoError(t, err)
		assert.Equal(t, 375, totalTokens)
	})

	t.Run("GetDateRangeInUserTimezone returns as-is", func(t *testing.T) {
		utcStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		utcEnd := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
		mockRepo.On("GetDateRange").Return(utcStart, utcEnd, nil)

		start, end, err := service.GetDateRangeInUserTimezone()
		require.NoError(t, err)
		assert.Equal(t, utcStart, start)
		assert.Equal(t, utcEnd, end)
	})

	mockRepo.AssertExpectations(t)
}