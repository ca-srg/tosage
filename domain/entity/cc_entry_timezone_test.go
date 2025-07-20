package entity

import (
	"testing"
	"time"

	"github.com/ca-srg/tosage/domain/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCcEntry_TimezoneSupport(t *testing.T) {
	// Test data
	tokenStats := valueobject.NewTokenStats(100, 200, 50, 25)
	timestamp := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)

	// Create entry with timezone
	jst, _ := time.LoadLocation("Asia/Tokyo")
	entry, err := NewCcEntryWithTimezone(
		"test-id",
		timestamp,
		"session-123",
		"/path/to/project",
		"gpt-4",
		tokenStats,
		"1.0.0",
		"msg-123",
		"req-123",
		jst,
	)
	require.NoError(t, err)

	t.Run("UserTimezone", func(t *testing.T) {
		assert.Equal(t, jst, entry.UserTimezone())
	})

	t.Run("TimestampInUserTimezone", func(t *testing.T) {
		userTime := entry.TimestampInUserTimezone()
		// UTC 14:30 should be JST 23:30
		assert.Equal(t, 23, userTime.Hour())
		assert.Equal(t, 30, userTime.Minute())
		assert.Equal(t, jst, userTime.Location())
	})

	t.Run("DateInUserTimezone", func(t *testing.T) {
		date := entry.DateInUserTimezone()
		// UTC 2024-01-15 14:30 is JST 2024-01-15 23:30
		assert.Equal(t, "2024-01-15", date)
	})

	t.Run("DateAsTimeInUserTimezone", func(t *testing.T) {
		dateTime := entry.DateAsTimeInUserTimezone()
		assert.Equal(t, 2024, dateTime.Year())
		assert.Equal(t, time.January, dateTime.Month())
		assert.Equal(t, 15, dateTime.Day())
		assert.Equal(t, 0, dateTime.Hour())
		assert.Equal(t, 0, dateTime.Minute())
		assert.Equal(t, 0, dateTime.Second())
		assert.Equal(t, jst, dateTime.Location())
	})

	t.Run("IsInDateRangeUserTimezone", func(t *testing.T) {
		// Create date range in JST
		start := time.Date(2024, 1, 15, 0, 0, 0, 0, jst)
		end := time.Date(2024, 1, 15, 23, 59, 59, 999999999, jst)

		assert.True(t, entry.IsInDateRangeUserTimezone(start, end))

		// Test outside range
		start2 := time.Date(2024, 1, 16, 0, 0, 0, 0, jst)
		end2 := time.Date(2024, 1, 16, 23, 59, 59, 999999999, jst)
		assert.False(t, entry.IsInDateRangeUserTimezone(start2, end2))
	})
}

func TestCcEntry_TimezoneCompatibility(t *testing.T) {
	// Test backward compatibility with JST
	tokenStats := valueobject.NewTokenStats(100, 200, 50, 25)
	timestamp := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)

	// Create entry without timezone (should fall back to JST)
	entry, err := NewCcEntry(
		"test-id",
		timestamp,
		"session-123",
		"/path/to/project",
		"gpt-4",
		tokenStats,
		"1.0.0",
		"msg-123",
		"req-123",
	)
	require.NoError(t, err)

	t.Run("Date falls back to JST", func(t *testing.T) {
		date := entry.Date()
		// UTC 2024-01-15 14:30 is JST 2024-01-15 23:30
		assert.Equal(t, "2024-01-15", date)
	})

	t.Run("DateAsTime falls back to JST", func(t *testing.T) {
		dateTime := entry.DateAsTime()
		jst, _ := time.LoadLocation("Asia/Tokyo")
		assert.Equal(t, jst, dateTime.Location())
		assert.Equal(t, 2024, dateTime.Year())
		assert.Equal(t, time.January, dateTime.Month())
		assert.Equal(t, 15, dateTime.Day())
	})

	t.Run("TimestampInUserTimezone falls back to JST", func(t *testing.T) {
		userTime := entry.TimestampInUserTimezone()
		// Should be in JST
		assert.Equal(t, 23, userTime.Hour())
		assert.Equal(t, 30, userTime.Minute())
	})
}

func TestCcEntry_SetUserTimezone(t *testing.T) {
	tokenStats := valueobject.NewTokenStats(100, 200, 50, 25)
	timestamp := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)

	entry, err := NewCcEntry(
		"test-id",
		timestamp,
		"session-123",
		"/path/to/project",
		"gpt-4",
		tokenStats,
		"1.0.0",
		"msg-123",
		"req-123",
	)
	require.NoError(t, err)

	// Set timezone to EST
	est, _ := time.LoadLocation("America/New_York")
	entry.SetUserTimezone(est)

	t.Run("Timezone is updated", func(t *testing.T) {
		assert.Equal(t, est, entry.UserTimezone())
	})

	t.Run("Date uses new timezone", func(t *testing.T) {
		date := entry.Date()
		// UTC 2024-01-15 14:30 is EST 2024-01-15 09:30
		assert.Equal(t, "2024-01-15", date)
	})
}

func TestCcEntry_DSTHandling(t *testing.T) {
	tokenStats := valueobject.NewTokenStats(100, 200, 50, 25)
	
	// Test spring forward (2AM -> 3AM)
	// March 10, 2024, 7:00 UTC = 2:00 EST -> 3:00 EDT
	springForward := time.Date(2024, 3, 10, 7, 0, 0, 0, time.UTC)
	
	est, _ := time.LoadLocation("America/New_York")
	entry1, err := NewCcEntryWithTimezone(
		"test-id-1",
		springForward,
		"session-123",
		"/path/to/project",
		"gpt-4",
		tokenStats,
		"1.0.0",
		"msg-123",
		"req-123",
		est,
	)
	require.NoError(t, err)

	t.Run("Spring forward date", func(t *testing.T) {
		date := entry1.DateInUserTimezone()
		assert.Equal(t, "2024-03-10", date)
	})

	// Test fall back (2AM -> 1AM)
	// November 3, 2024, 6:00 UTC = 2:00 EDT -> 1:00 EST
	fallBack := time.Date(2024, 11, 3, 6, 0, 0, 0, time.UTC)
	
	entry2, err := NewCcEntryWithTimezone(
		"test-id-2",
		fallBack,
		"session-123",
		"/path/to/project",
		"gpt-4",
		tokenStats,
		"1.0.0",
		"msg-123",
		"req-123",
		est,
	)
	require.NoError(t, err)

	t.Run("Fall back date", func(t *testing.T) {
		date := entry2.DateInUserTimezone()
		assert.Equal(t, "2024-11-03", date)
	})
}

func TestCcEntryCollection_TimezoneSupport(t *testing.T) {
	tokenStats := valueobject.NewTokenStats(100, 200, 50, 25)
	jst, _ := time.LoadLocation("Asia/Tokyo")
	
	// Create entries at different times
	entries := []*CcEntry{}
	
	// Entry 1: UTC 2024-01-15 14:00 = JST 2024-01-15 23:00
	entry1, _ := NewCcEntry("id1", time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC), "s1", "/p1", "gpt-4", tokenStats, "1.0", "m1", "r1")
	entries = append(entries, entry1)
	
	// Entry 2: UTC 2024-01-15 15:00 = JST 2024-01-16 00:00
	entry2, _ := NewCcEntry("id2", time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC), "s2", "/p2", "gpt-4", tokenStats, "1.0", "m2", "r2")
	entries = append(entries, entry2)
	
	// Entry 3: UTC 2024-01-16 01:00 = JST 2024-01-16 10:00
	entry3, _ := NewCcEntry("id3", time.Date(2024, 1, 16, 1, 0, 0, 0, time.UTC), "s3", "/p3", "gpt-4", tokenStats, "1.0", "m3", "r3")
	entries = append(entries, entry3)

	// Create collection with timezone
	collection := NewCcEntryCollectionWithTimezone(entries, jst)

	t.Run("ApplyTimezone", func(t *testing.T) {
		// All entries should have JST timezone
		for _, entry := range collection.Entries() {
			assert.Equal(t, jst, entry.UserTimezone())
		}
	})

	t.Run("FilterByDateInUserTimezone", func(t *testing.T) {
		// Filter for JST 2024-01-15
		date := time.Date(2024, 1, 15, 12, 0, 0, 0, jst)
		filtered := collection.FilterByDateInUserTimezone(date)
		
		// Should only include entry1
		assert.Equal(t, 1, filtered.Count())
		assert.Equal(t, "id1", filtered.Entries()[0].ID())
	})

	t.Run("FilterByDateRangeInUserTimezone", func(t *testing.T) {
		// Filter for JST 2024-01-16 00:00 to 23:59
		start := time.Date(2024, 1, 16, 0, 0, 0, 0, jst)
		end := time.Date(2024, 1, 16, 23, 59, 59, 999999999, jst)
		filtered := collection.FilterByDateRangeInUserTimezone(start, end)
		
		// Should include entry2 and entry3
		assert.Equal(t, 2, filtered.Count())
	})

	t.Run("GroupByDateInUserTimezone", func(t *testing.T) {
		groups := collection.GroupByDateInUserTimezone()
		
		// Should have 2 groups: 2024-01-15 and 2024-01-16
		assert.Equal(t, 2, len(groups))
		assert.NotNil(t, groups["2024-01-15"])
		assert.NotNil(t, groups["2024-01-16"])
		
		// 2024-01-15 should have 1 entry
		assert.Equal(t, 1, groups["2024-01-15"].Count())
		
		// 2024-01-16 should have 2 entries
		assert.Equal(t, 2, groups["2024-01-16"].Count())
	})
}