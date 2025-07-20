package entity

import (
	"fmt"
	"time"

	"github.com/ca-srg/tosage/domain/valueobject"
)

// CcEntry represents a domain entity for API cc
type CcEntry struct {
	id           string
	timestamp    time.Time
	sessionID    string
	projectPath  string
	model        string
	tokenStats   valueobject.TokenStats
	version      string
	messageID    string
	requestID    string
	userTimezone *time.Location // Optional: user's timezone for date calculations
}

// NewCcEntry creates a new CcEntry entity with validation
func NewCcEntry(
	id string,
	timestamp time.Time,
	sessionID string,
	projectPath string,
	model string,
	tokenStats valueobject.TokenStats,
	version string,
	messageID string,
	requestID string,
) (*CcEntry, error) {
	// Validate required fields
	if id == "" {
		return nil, fmt.Errorf("cc entry ID cannot be empty")
	}
	if timestamp.IsZero() {
		return nil, fmt.Errorf("timestamp cannot be zero")
	}
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}
	if projectPath == "" {
		return nil, fmt.Errorf("project path cannot be empty")
	}

	return &CcEntry{
		id:          id,
		timestamp:   timestamp,
		sessionID:   sessionID,
		projectPath: projectPath,
		model:       model,
		tokenStats:  tokenStats,
		version:     version,
		messageID:   messageID,
		requestID:   requestID,
	}, nil
}

// NewCcEntryWithTimezone creates a new CcEntry entity with timezone context
func NewCcEntryWithTimezone(
	id string,
	timestamp time.Time,
	sessionID string,
	projectPath string,
	model string,
	tokenStats valueobject.TokenStats,
	version string,
	messageID string,
	requestID string,
	userTimezone *time.Location,
) (*CcEntry, error) {
	entry, err := NewCcEntry(id, timestamp, sessionID, projectPath, model, tokenStats, version, messageID, requestID)
	if err != nil {
		return nil, err
	}
	entry.userTimezone = userTimezone
	return entry, nil
}

// ID returns the cc entry ID
func (u *CcEntry) ID() string {
	return u.id
}

// Timestamp returns the timestamp
func (u *CcEntry) Timestamp() time.Time {
	return u.timestamp
}

// SessionID returns the session ID
func (u *CcEntry) SessionID() string {
	return u.sessionID
}

// ProjectPath returns the project path
func (u *CcEntry) ProjectPath() string {
	return u.projectPath
}

// Model returns the model name
func (u *CcEntry) Model() string {
	return u.model
}

// TokenStats returns the token statistics
func (u *CcEntry) TokenStats() valueobject.TokenStats {
	return u.tokenStats
}

// Version returns the version
func (u *CcEntry) Version() string {
	return u.version
}

// MessageID returns the message ID
func (u *CcEntry) MessageID() string {
	return u.messageID
}

// RequestID returns the request ID
func (u *CcEntry) RequestID() string {
	return u.requestID
}

// UserTimezone returns the user's timezone (may be nil)
func (u *CcEntry) UserTimezone() *time.Location {
	return u.userTimezone
}

// SetUserTimezone sets the user's timezone
func (u *CcEntry) SetUserTimezone(loc *time.Location) {
	u.userTimezone = loc
}

// TimestampInUserTimezone returns the timestamp in user's timezone
func (u *CcEntry) TimestampInUserTimezone() time.Time {
	if u.userTimezone != nil {
		return u.timestamp.In(u.userTimezone)
	}
	// Fall back to JST for backward compatibility
	jst, _ := time.LoadLocation("Asia/Tokyo")
	return u.timestamp.In(jst)
}

// DateInUserTimezone returns the date in YYYY-MM-DD format in user's timezone
func (u *CcEntry) DateInUserTimezone() string {
	userTime := u.TimestampInUserTimezone()
	return userTime.Format("2006-01-02")
}

// IsInDateRangeUserTimezone checks if the entry falls within a date range in user's timezone
func (u *CcEntry) IsInDateRangeUserTimezone(start, end time.Time) bool {
	entryDate := u.DateAsTimeInUserTimezone()
	return !entryDate.Before(start) && !entryDate.After(end)
}

// DateAsTimeInUserTimezone returns the date as a time.Time (at midnight) in user's timezone
func (u *CcEntry) DateAsTimeInUserTimezone() time.Time {
	userTime := u.TimestampInUserTimezone()
	year, month, day := userTime.Date()
	loc := userTime.Location()
	return time.Date(year, month, day, 0, 0, 0, 0, loc)
}

// Date returns the date in YYYY-MM-DD format
func (u *CcEntry) Date() string {
	// Use user timezone if available, otherwise fall back to JST for backward compatibility
	if u.userTimezone != nil {
		return u.timestamp.In(u.userTimezone).Format("2006-01-02")
	}
	// Always use JST for date formatting (backward compatibility)
	jst, _ := time.LoadLocation("Asia/Tokyo")
	jstTime := u.timestamp.In(jst)
	return jstTime.Format("2006-01-02")
}

// DateAsTime returns the date as a time.Time (at midnight)
func (u *CcEntry) DateAsTime() time.Time {
	// Use user timezone if available, otherwise fall back to JST for backward compatibility
	var loc *time.Location
	if u.userTimezone != nil {
		loc = u.userTimezone
	} else {
		// Always use JST for date comparisons (backward compatibility)
		loc, _ = time.LoadLocation("Asia/Tokyo")
	}
	locTime := u.timestamp.In(loc)
	year, month, day := locTime.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, loc)
}

// IsInDateRange checks if the cc entry falls within a date range
func (u *CcEntry) IsInDateRange(start, end time.Time) bool {
	entryDate := u.DateAsTime()
	return !entryDate.Before(start) && !entryDate.After(end)
}

// IsOnDate checks if the cc entry is on a specific date
func (u *CcEntry) IsOnDate(date time.Time) bool {
	// Use user timezone if available, otherwise fall back to JST for backward compatibility
	var loc *time.Location
	if u.userTimezone != nil {
		loc = u.userTimezone
	} else {
		// Convert both to JST for comparison (backward compatibility)
		loc, _ = time.LoadLocation("Asia/Tokyo")
	}
	dateInLoc := date.In(loc)
	return u.Date() == dateInLoc.Format("2006-01-02")
}

// IsForModel checks if the cc entry is for a specific model
func (u *CcEntry) IsForModel(model string) bool {
	return u.model == model
}

// IsForProject checks if the cc entry is for a specific project
func (u *CcEntry) IsForProject(projectPath string) bool {
	return u.projectPath == projectPath
}

// IsForSession checks if the cc entry is for a specific session
func (u *CcEntry) IsForSession(sessionID string) bool {
	return u.sessionID == sessionID
}

// HasModel checks if the cc entry has a model specified
func (u *CcEntry) HasModel() bool {
	return u.model != ""
}

// TotalTokens returns the total number of tokens
func (u *CcEntry) TotalTokens() int {
	return u.tokenStats.TotalTokens()
}

// IsEmpty checks if the cc entry has no tokens
func (u *CcEntry) IsEmpty() bool {
	return u.tokenStats.IsEmpty()
}

// CreateDeduplicationKey creates a key for deduplication
func (u *CcEntry) CreateDeduplicationKey() string {
	// Prefer message ID if available
	if u.messageID != "" {
		return "msg:" + u.messageID
	}
	// Fall back to request ID
	if u.requestID != "" {
		return "req:" + u.requestID
	}
	// Fall back to ID
	return "id:" + u.id
}

// Equals checks if two cc entries are equal
func (u *CcEntry) Equals(other *CcEntry) bool {
	if other == nil {
		return false
	}
	return u.id == other.id &&
		u.timestamp.Equal(other.timestamp) &&
		u.sessionID == other.sessionID &&
		u.projectPath == other.projectPath &&
		u.model == other.model &&
		u.tokenStats.Equals(other.tokenStats) &&
		u.version == other.version &&
		u.messageID == other.messageID &&
		u.requestID == other.requestID
}

// CcEntryCollection represents a collection of cc entries
type CcEntryCollection struct {
	entries      []*CcEntry
	userTimezone *time.Location // Optional: shared timezone for all entries
}

// NewCcEntryCollection creates a new collection
func NewCcEntryCollection(entries []*CcEntry) *CcEntryCollection {
	return &CcEntryCollection{
		entries: entries,
	}
}

// NewCcEntryCollectionWithTimezone creates a new collection with timezone context
func NewCcEntryCollectionWithTimezone(entries []*CcEntry, timezone *time.Location) *CcEntryCollection {
	collection := &CcEntryCollection{
		entries:      entries,
		userTimezone: timezone,
	}
	// Set timezone for all entries
	collection.ApplyTimezone(timezone)
	return collection
}

// Entries returns all entries
func (c *CcEntryCollection) Entries() []*CcEntry {
	return c.entries
}

// FilterByDateRange filters entries by date range
func (c *CcEntryCollection) FilterByDateRange(start, end time.Time) *CcEntryCollection {
	var filtered []*CcEntry
	for _, entry := range c.entries {
		if entry.IsInDateRange(start, end) {
			filtered = append(filtered, entry)
		}
	}
	return NewCcEntryCollection(filtered)
}

// FilterByDate filters entries by specific date
func (c *CcEntryCollection) FilterByDate(date time.Time) *CcEntryCollection {
	var filtered []*CcEntry
	for _, entry := range c.entries {
		if entry.IsOnDate(date) {
			filtered = append(filtered, entry)
		}
	}
	return NewCcEntryCollection(filtered)
}

// FilterByModel filters entries by model
func (c *CcEntryCollection) FilterByModel(model string) *CcEntryCollection {
	var filtered []*CcEntry
	for _, entry := range c.entries {
		if entry.IsForModel(model) {
			filtered = append(filtered, entry)
		}
	}
	return NewCcEntryCollection(filtered)
}

// FilterByProject filters entries by project
func (c *CcEntryCollection) FilterByProject(projectPath string) *CcEntryCollection {
	var filtered []*CcEntry
	for _, entry := range c.entries {
		if entry.IsForProject(projectPath) {
			filtered = append(filtered, entry)
		}
	}
	return NewCcEntryCollection(filtered)
}

// FilterBySession filters entries by session
func (c *CcEntryCollection) FilterBySession(sessionID string) *CcEntryCollection {
	var filtered []*CcEntry
	for _, entry := range c.entries {
		if entry.IsForSession(sessionID) {
			filtered = append(filtered, entry)
		}
	}
	return NewCcEntryCollection(filtered)
}

// GroupByModel groups entries by model
func (c *CcEntryCollection) GroupByModel() map[string]*CcEntryCollection {
	groups := make(map[string]*CcEntryCollection)

	for _, entry := range c.entries {
		model := entry.Model()
		if model == "" {
			model = "unknown"
		}

		if group, exists := groups[model]; exists {
			group.entries = append(group.entries, entry)
		} else {
			groups[model] = NewCcEntryCollection([]*CcEntry{entry})
		}
	}

	return groups
}

// GroupByDate groups entries by date
func (c *CcEntryCollection) GroupByDate() map[string]*CcEntryCollection {
	groups := make(map[string]*CcEntryCollection)

	for _, entry := range c.entries {
		date := entry.Date()

		if group, exists := groups[date]; exists {
			group.entries = append(group.entries, entry)
		} else {
			groups[date] = NewCcEntryCollection([]*CcEntry{entry})
		}
	}

	return groups
}

// TotalTokenStats calculates total token statistics for the collection
func (c *CcEntryCollection) TotalTokenStats() (valueobject.TokenStats, error) {
	if len(c.entries) == 0 {
		return valueobject.NewEmptyTokenStats(), nil
	}

	var stats []valueobject.TokenStats
	for _, entry := range c.entries {
		stats = append(stats, entry.TokenStats())
	}

	return valueobject.MergeMultipleTokenStats(stats), nil
}

// IsEmpty checks if the collection is empty
func (c *CcEntryCollection) IsEmpty() bool {
	return len(c.entries) == 0
}

// Count returns the number of entries
func (c *CcEntryCollection) Count() int {
	return len(c.entries)
}

// ApplyTimezone applies the timezone to all entries in the collection
func (c *CcEntryCollection) ApplyTimezone(timezone *time.Location) {
	c.userTimezone = timezone
	for _, entry := range c.entries {
		entry.SetUserTimezone(timezone)
	}
}

// FilterByDateRangeInUserTimezone filters entries by date range in user's timezone
func (c *CcEntryCollection) FilterByDateRangeInUserTimezone(start, end time.Time) *CcEntryCollection {
	var filtered []*CcEntry
	for _, entry := range c.entries {
		if entry.IsInDateRangeUserTimezone(start, end) {
			filtered = append(filtered, entry)
		}
	}
	result := NewCcEntryCollection(filtered)
	result.userTimezone = c.userTimezone
	return result
}

// FilterByDateInUserTimezone filters entries by specific date in user's timezone
func (c *CcEntryCollection) FilterByDateInUserTimezone(date time.Time) *CcEntryCollection {
	var filtered []*CcEntry
	for _, entry := range c.entries {
		// Use timezone-aware comparison
		if c.userTimezone != nil {
			entry.SetUserTimezone(c.userTimezone)
		}
		if entry.IsOnDate(date) {
			filtered = append(filtered, entry)
		}
	}
	result := NewCcEntryCollection(filtered)
	result.userTimezone = c.userTimezone
	return result
}

// GroupByDateInUserTimezone groups entries by date in user's timezone
func (c *CcEntryCollection) GroupByDateInUserTimezone() map[string]*CcEntryCollection {
	groups := make(map[string]*CcEntryCollection)

	for _, entry := range c.entries {
		// Ensure entry has timezone set
		if c.userTimezone != nil {
			entry.SetUserTimezone(c.userTimezone)
		}
		date := entry.DateInUserTimezone()

		if group, exists := groups[date]; exists {
			group.entries = append(group.entries, entry)
		} else {
			newGroup := NewCcEntryCollection([]*CcEntry{entry})
			newGroup.userTimezone = c.userTimezone
			groups[date] = newGroup
		}
	}

	return groups
}
