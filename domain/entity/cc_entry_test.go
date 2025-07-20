package entity

import (
	"testing"
	"time"

	"github.com/ca-srg/tosage/domain/valueobject"
)

func TestNewCcEntry(t *testing.T) {
	timestamp := time.Now()
	tokenStats := valueobject.NewTokenStats(100, 50, 20, 30)

	tests := []struct {
		name        string
		id          string
		timestamp   time.Time
		sessionID   string
		projectPath string
		model       string
		tokenStats  valueobject.TokenStats
		version     string
		messageID   string
		requestID   string
		wantErr     bool
	}{
		{
			name:        "valid cc entry",
			id:          "entry-1",
			timestamp:   timestamp,
			sessionID:   "session-1",
			projectPath: "/project/path",
			model:       "claude-3-5-sonnet-20241022",
			tokenStats:  tokenStats,
			version:     "1.0.0",
			messageID:   "msg-123",
			requestID:   "req-456",
			wantErr:     false,
		},
		{
			name:        "empty ID",
			id:          "",
			timestamp:   timestamp,
			sessionID:   "session-1",
			projectPath: "/project/path",
			model:       "claude-3-5-sonnet-20241022",
			tokenStats:  tokenStats,
			version:     "1.0.0",
			messageID:   "msg-123",
			requestID:   "req-456",
			wantErr:     true,
		},
		{
			name:        "zero timestamp",
			id:          "entry-1",
			timestamp:   time.Time{},
			sessionID:   "session-1",
			projectPath: "/project/path",
			model:       "claude-3-5-sonnet-20241022",
			tokenStats:  tokenStats,
			version:     "1.0.0",
			messageID:   "msg-123",
			requestID:   "req-456",
			wantErr:     true,
		},
		{
			name:        "empty session ID",
			id:          "entry-1",
			timestamp:   timestamp,
			sessionID:   "",
			projectPath: "/project/path",
			model:       "claude-3-5-sonnet-20241022",
			tokenStats:  tokenStats,
			version:     "1.0.0",
			messageID:   "msg-123",
			requestID:   "req-456",
			wantErr:     true,
		},
		{
			name:        "empty project path",
			id:          "entry-1",
			timestamp:   timestamp,
			sessionID:   "session-1",
			projectPath: "",
			model:       "claude-3-5-sonnet-20241022",
			tokenStats:  tokenStats,
			version:     "1.0.0",
			messageID:   "msg-123",
			requestID:   "req-456",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCcEntry(tt.id, tt.timestamp, tt.sessionID, tt.projectPath, tt.model, tt.tokenStats, tt.version, tt.messageID, tt.requestID)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCcEntry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("NewCcEntry() returned nil without error")
			}
		})
	}
}

func TestCcEntry_DateMethods(t *testing.T) {
	timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	tokenStats := valueobject.NewTokenStats(100, 50, 20, 30)
	entry, _ := NewCcEntry("entry-1", timestamp, "session-1", "/project", "model", tokenStats, "1.0.0", "", "")

	t.Run("Date", func(t *testing.T) {
		expected := "2024-01-15"
		if got := entry.Date(); got != expected {
			t.Errorf("Date() = %v, want %v", got, expected)
		}
	})

	t.Run("DateAsTime", func(t *testing.T) {
		expected := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		if got := entry.DateAsTime(); !got.Equal(expected) {
			t.Errorf("DateAsTime() = %v, want %v", got, expected)
		}
	})

	t.Run("IsInDateRange", func(t *testing.T) {
		start := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
		end := time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)
		if !entry.IsInDateRange(start, end) {
			t.Error("IsInDateRange() should return true")
		}

		// Test outside range
		start = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		end = time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
		if entry.IsInDateRange(start, end) {
			t.Error("IsInDateRange() should return false")
		}
	})

	t.Run("IsOnDate", func(t *testing.T) {
		date := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
		if !entry.IsOnDate(date) {
			t.Error("IsOnDate() should return true")
		}

		date = time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
		if entry.IsOnDate(date) {
			t.Error("IsOnDate() should return false")
		}
	})
}

func TestCcEntryCollection_Filters(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)
	tomorrow := today.AddDate(0, 0, 1)

	tokenStats := valueobject.NewTokenStats(100, 50, 20, 30)

	entries := []*CcEntry{
		func() *CcEntry {
			e, _ := NewCcEntry("1", today.Add(10*time.Hour), "session-1", "/project1", "model-1", tokenStats, "1.0", "", "")
			return e
		}(),
		func() *CcEntry {
			e, _ := NewCcEntry("2", today.Add(11*time.Hour), "session-1", "/project2", "model-2", tokenStats, "1.0", "", "")
			return e
		}(),
		func() *CcEntry {
			e, _ := NewCcEntry("3", yesterday.Add(10*time.Hour), "session-2", "/project1", "model-1", tokenStats, "1.0", "", "")
			return e
		}(),
		func() *CcEntry {
			e, _ := NewCcEntry("4", tomorrow.Add(10*time.Hour), "session-2", "/project2", "model-2", tokenStats, "1.0", "", "")
			return e
		}(),
	}

	collection := NewCcEntryCollection(entries)

	t.Run("FilterByDate", func(t *testing.T) {
		filtered := collection.FilterByDate(today)
		if filtered.Count() != 2 {
			t.Errorf("FilterByDate() count = %v, want 2", filtered.Count())
		}
	})

	t.Run("FilterByDateRange", func(t *testing.T) {
		filtered := collection.FilterByDateRange(yesterday, today)
		if filtered.Count() != 3 {
			t.Errorf("FilterByDateRange() count = %v, want 3", filtered.Count())
		}
	})

	t.Run("FilterByModel", func(t *testing.T) {
		filtered := collection.FilterByModel("model-1")
		if filtered.Count() != 2 {
			t.Errorf("FilterByModel() count = %v, want 2", filtered.Count())
		}
	})

	t.Run("FilterByProject", func(t *testing.T) {
		filtered := collection.FilterByProject("/project1")
		if filtered.Count() != 2 {
			t.Errorf("FilterByProject() count = %v, want 2", filtered.Count())
		}
	})

	t.Run("FilterBySession", func(t *testing.T) {
		filtered := collection.FilterBySession("session-1")
		if filtered.Count() != 2 {
			t.Errorf("FilterBySession() count = %v, want 2", filtered.Count())
		}
	})
}

func TestCcEntryCollection_GroupBy(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)

	tokenStats := valueobject.NewTokenStats(100, 50, 20, 30)

	entries := []*CcEntry{
		func() *CcEntry {
			e, _ := NewCcEntry("1", today, "s1", "/p1", "model-1", tokenStats, "1.0", "", "")
			return e
		}(),
		func() *CcEntry {
			e, _ := NewCcEntry("2", today, "s1", "/p1", "model-2", tokenStats, "1.0", "", "")
			return e
		}(),
		func() *CcEntry {
			e, _ := NewCcEntry("3", yesterday, "s1", "/p1", "model-1", tokenStats, "1.0", "", "")
			return e
		}(),
		func() *CcEntry {
			e, _ := NewCcEntry("4", yesterday, "s1", "/p1", "", tokenStats, "1.0", "", "")
			return e
		}(),
	}

	collection := NewCcEntryCollection(entries)

	t.Run("GroupByModel", func(t *testing.T) {
		groups := collection.GroupByModel()
		if len(groups) != 3 { // model-1, model-2, unknown
			t.Errorf("GroupByModel() groups = %v, want 3", len(groups))
		}
		if groups["model-1"].Count() != 2 {
			t.Errorf("GroupByModel()[model-1] count = %v, want 2", groups["model-1"].Count())
		}
		if groups["unknown"].Count() != 1 {
			t.Errorf("GroupByModel()[unknown] count = %v, want 1", groups["unknown"].Count())
		}
	})

	t.Run("GroupByDate", func(t *testing.T) {
		groups := collection.GroupByDate()
		if len(groups) != 2 {
			t.Errorf("GroupByDate() groups = %v, want 2", len(groups))
		}
		todayStr := today.Format("2006-01-02")
		yesterdayStr := yesterday.Format("2006-01-02")
		if groups[todayStr].Count() != 2 {
			t.Errorf("GroupByDate()[today] count = %v, want 2", groups[todayStr].Count())
		}
		if groups[yesterdayStr].Count() != 2 {
			t.Errorf("GroupByDate()[yesterday] count = %v, want 2", groups[yesterdayStr].Count())
		}
	})
}

func TestCcEntryCollection_TotalTokenStats(t *testing.T) {
	tokenStats1 := valueobject.NewTokenStats(100, 50, 20, 30)
	tokenStats2 := valueobject.NewTokenStats(50, 25, 10, 15)

	entries := []*CcEntry{
		func() *CcEntry {
			e, _ := NewCcEntry("1", time.Now(), "s1", "/p1", "m1", tokenStats1, "1.0", "", "")
			return e
		}(),
		func() *CcEntry {
			e, _ := NewCcEntry("2", time.Now(), "s1", "/p1", "m1", tokenStats2, "1.0", "", "")
			return e
		}(),
	}

	collection := NewCcEntryCollection(entries)

	total, err := collection.TotalTokenStats()
	if err != nil {
		t.Fatalf("TotalTokenStats() error = %v", err)
	}

	expected := valueobject.NewTokenStats(150, 75, 30, 45)
	if !total.Equals(expected) {
		t.Errorf("TotalTokenStats() = %v, want %v", total, expected)
	}
}
