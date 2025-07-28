package repository

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ca-srg/tosage/domain/entity"
	"github.com/ca-srg/tosage/domain/repository"
	"github.com/ca-srg/tosage/domain/valueobject"
)

// JSONLCcRepository implements CcRepository using JSONL files
type JSONLCcRepository struct {
	claudePaths []string
	cache       *ccCache
}

// ccCache holds cached cc entries
type ccCache struct {
	entries      []*entity.CcEntry
	lastModified time.Time
	mu           sync.RWMutex
}

// NewJSONLCcRepository creates a new JSONL-based cc repository
func NewJSONLCcRepository(customPath string) *JSONLCcRepository {
	repo := &JSONLCcRepository{
		cache: &ccCache{},
	}
	repo.claudePaths = repo.getClaudePaths(customPath)
	return repo
}

// getClaudePaths returns the paths to search for Claude data
func (r *JSONLCcRepository) getClaudePaths(customPath string) []string {
	var paths []string

	if customPath != "" {
		// Use custom path if provided
		paths = append(paths, customPath)
	} else {
		// Default paths based on operating system
		home, err := os.UserHomeDir()
		if err != nil {
			return paths
		}

		// Support both old and new Claude Code data locations
		paths = append(paths, filepath.Join(home, ".config", "claude", "projects"))
		paths = append(paths, filepath.Join(home, ".claude", "projects"))
		paths = append(paths, filepath.Join(home, "Library", "Application Support", "claude", "projects"))
	}

	return paths
}

// loadAllEntries loads all cc entries from JSONL files
func (r *JSONLCcRepository) loadAllEntries() ([]*entity.CcEntry, error) {
	// Check cache first
	r.cache.mu.RLock()
	if r.cache.entries != nil && time.Since(r.cache.lastModified) < 5*time.Minute {
		entries := r.cache.entries
		r.cache.mu.RUnlock()
		fmt.Fprintf(os.Stderr, "[DEBUG] Returning %d cached entries\n", len(entries))
		return entries, nil
	}
	r.cache.mu.RUnlock()

	// Load fresh data
	validPaths := r.getValidClaudePaths()
	// fmt.Fprintf(os.Stderr, "[DEBUG] Found %d valid Claude paths: %v\n", len(validPaths), validPaths)
	if len(validPaths) == 0 {
		return nil, fmt.Errorf("no valid Claude data directories found")
	}

	var allEntries []*entity.CcEntry
	processedIDs := make(map[string]bool) // For deduplication

	for _, basePath := range validPaths {
		// fmt.Fprintf(os.Stderr, "[DEBUG] Loading from base path: %s\n", basePath)
		entries, err := r.loadFromPath(basePath, processedIDs)
		if err != nil {
			// Log error but continue with other paths
			fmt.Fprintf(os.Stderr, "Warning: Failed to load from %s: %v\n", basePath, err)
			continue
		}
		// fmt.Fprintf(os.Stderr, "[DEBUG] Loaded %d entries from %s\n", len(entries), basePath)
		allEntries = append(allEntries, entries...)
	}

	// fmt.Fprintf(os.Stderr, "[DEBUG] Total entries loaded: %d\n", len(allEntries))

	// Calculate total tokens and date range
	totalTokens := 0
	var minDate, maxDate time.Time
	if len(allEntries) > 0 {
		minDate = allEntries[0].Timestamp()
		maxDate = allEntries[0].Timestamp()
	}

	for _, entry := range allEntries {
		totalTokens += entry.TotalTokens()
		if entry.Timestamp().Before(minDate) {
			minDate = entry.Timestamp()
		}
		if entry.Timestamp().After(maxDate) {
			maxDate = entry.Timestamp()
		}
	}
	// fmt.Fprintf(os.Stderr, "[DEBUG] Total tokens across all entries: %d\n", totalTokens)
	// if len(allEntries) > 0 {
	// 	fmt.Fprintf(os.Stderr, "[DEBUG] Date range of entries: %v to %v\n", minDate, maxDate)
	// }

	if len(allEntries) == 0 {
		return nil, fmt.Errorf("no cc data found in any Claude directory")
	}

	// Update cache
	r.cache.mu.Lock()
	r.cache.entries = allEntries
	r.cache.lastModified = time.Now()
	r.cache.mu.Unlock()

	return allEntries, nil
}

// getValidClaudePaths returns only the Claude paths that exist
func (r *JSONLCcRepository) getValidClaudePaths() []string {
	var validPaths []string
	for _, path := range r.claudePaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			validPaths = append(validPaths, path)
		}
	}
	return validPaths
}

// loadFromPath loads cc data from a specific Claude projects path
func (r *JSONLCcRepository) loadFromPath(basePath string, processedIDs map[string]bool) ([]*entity.CcEntry, error) {
	var entries []*entity.CcEntry

	// Walk through all JSONL files in the projects directory
	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible paths
		}

		// Only process .jsonl files
		if !strings.HasSuffix(path, ".jsonl") || info.IsDir() {
			return nil
		}

		// Extract session and project info from path
		relPath, _ := filepath.Rel(basePath, path)
		parts := strings.Split(relPath, string(filepath.Separator))

		if len(parts) >= 2 {
			projectPath := parts[0]
			sessionID := parts[1]

			// Load entries from this file
			fileEntries, err := r.loadJSONLFile(path, projectPath, sessionID, processedIDs)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to load %s: %v\n", path, err)
				return nil // Continue with other files
			}

			entries = append(entries, fileEntries...)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return entries, nil
}

// loadJSONLFile loads and parses a single JSONL file
func (r *JSONLCcRepository) loadJSONLFile(filePath, projectPath, sessionID string, processedIDs map[string]bool) ([]*entity.CcEntry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	// fmt.Fprintf(os.Stderr, "[DEBUG] Loading JSONL file: %s\n", filePath)

	var entries []*entity.CcEntry
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024) // Handle large lines up to 10MB

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var data ccData
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			// Skip malformed lines
			// fmt.Fprintf(os.Stderr, "[DEBUG] Failed to parse JSON at line %d: %v\n", lineNum, err)
			continue
		}

		// Debug log token values
		// fmt.Fprintf(os.Stderr, "[DEBUG] Line %d - Input tokens: %d, Output tokens: %d, Cache creation: %d, Cache read: %d\n",
		// 	lineNum,
		// 	data.Message.Usage.InputTokens,
		// 	data.Message.Usage.OutputTokens,
		// 	data.Message.Usage.CacheCreationInputTokens,
		// 	data.Message.Usage.CacheReadInputTokens)

		// Create deduplication key
		dedupKey := r.createDedupKey(&data)
		if dedupKey != "" && processedIDs[dedupKey] {
			// fmt.Fprintf(os.Stderr, "[DEBUG] Skipping duplicate entry with key: %s\n", dedupKey)
			continue // Skip duplicate
		}
		if dedupKey != "" {
			processedIDs[dedupKey] = true
		}

		// Convert to domain entity
		entry, err := r.convertToCcEntry(&data, projectPath, sessionID)
		if err != nil {
			// fmt.Fprintf(os.Stderr, "[DEBUG] Failed to convert to entry at line %d: %v\n", lineNum, err)
			continue // Skip invalid entries
		}

		// fmt.Fprintf(os.Stderr, "[DEBUG] Created entry with total tokens: %d, timestamp: %v\n", entry.TotalTokens(), entry.Timestamp())
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return entries, fmt.Errorf("error reading file: %w", err)
	}

	// fmt.Fprintf(os.Stderr, "[DEBUG] Loaded %d entries from file: %s\n", len(entries), filePath)
	return entries, nil
}

// convertToCcEntry converts raw cc data to domain entity
func (r *JSONLCcRepository) convertToCcEntry(data *ccData, projectPath, sessionID string) (*entity.CcEntry, error) {
	// Parse timestamp
	timestamp, err := time.Parse(time.RFC3339, data.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp: %w", err)
	}

	// Create token stats value object
	tokenStats := valueobject.NewTokenStats(
		data.Message.Usage.InputTokens,
		data.Message.Usage.OutputTokens,
		data.Message.Usage.CacheCreationInputTokens,
		data.Message.Usage.CacheReadInputTokens,
	)

	// Generate ID if not provided
	id := data.Message.ID
	if id == "" {
		id = fmt.Sprintf("%s-%s-%d", sessionID, timestamp.Format("20060102150405"), lineNum)
	}

	// Create domain entity
	entry, err := entity.NewCcEntry(
		id,
		timestamp,
		sessionID,
		projectPath,
		data.Message.Model,
		tokenStats,
		data.Version,
		data.Message.ID,
		data.RequestID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cc entry: %w", err)
	}

	return entry, nil
}

// createDedupKey creates a deduplication key from cc data
func (r *JSONLCcRepository) createDedupKey(data *ccData) string {
	// Use message ID if available
	if data.Message.ID != "" {
		return "msg:" + data.Message.ID
	}

	// Use request ID if available
	if data.RequestID != "" {
		return "req:" + data.RequestID
	}

	// No deduplication key available
	return ""
}

// Repository interface implementations

// FindAll returns all cc entries
func (r *JSONLCcRepository) FindAll() ([]*entity.CcEntry, error) {
	return r.loadAllEntries()
}

// FindByID returns a cc entry by its ID
func (r *JSONLCcRepository) FindByID(id string) (*entity.CcEntry, error) {
	entries, err := r.loadAllEntries()
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.ID() == id {
			return entry, nil
		}
	}

	return nil, repository.ErrCcNotFound
}

// FindByDateRange returns cc entries within a date range
func (r *JSONLCcRepository) FindByDateRange(start, end time.Time) ([]*entity.CcEntry, error) {
	entries, err := r.loadAllEntries()
	if err != nil {
		return nil, err
	}

	// fmt.Fprintf(os.Stderr, "[DEBUG] FindByDateRange - Start: %v, End: %v\n", start, end)

	var result []*entity.CcEntry
	for _, entry := range entries {
		if entry.IsInDateRange(start, end) {
			result = append(result, entry)
		}
	}

	// fmt.Fprintf(os.Stderr, "[DEBUG] FindByDateRange - Found %d entries in date range out of %d total\n", len(result), len(entries))
	return result, nil
}

// FindByDate returns cc entries for a specific date
func (r *JSONLCcRepository) FindByDate(date time.Time) ([]*entity.CcEntry, error) {
	entries, err := r.loadAllEntries()
	if err != nil {
		return nil, err
	}

	var result []*entity.CcEntry
	for _, entry := range entries {
		if entry.IsOnDate(date) {
			result = append(result, entry)
		}
	}

	return result, nil
}

// FindByProject returns cc entries for a specific project
func (r *JSONLCcRepository) FindByProject(projectPath string) ([]*entity.CcEntry, error) {
	entries, err := r.loadAllEntries()
	if err != nil {
		return nil, err
	}

	var result []*entity.CcEntry
	for _, entry := range entries {
		if entry.IsForProject(projectPath) {
			result = append(result, entry)
		}
	}

	return result, nil
}

// FindBySession returns cc entries for a specific session
func (r *JSONLCcRepository) FindBySession(sessionID string) ([]*entity.CcEntry, error) {
	entries, err := r.loadAllEntries()
	if err != nil {
		return nil, err
	}

	var result []*entity.CcEntry
	for _, entry := range entries {
		if entry.IsForSession(sessionID) {
			result = append(result, entry)
		}
	}

	return result, nil
}

// FindByModel returns cc entries for a specific model
func (r *JSONLCcRepository) FindByModel(model string) ([]*entity.CcEntry, error) {
	entries, err := r.loadAllEntries()
	if err != nil {
		return nil, err
	}

	var result []*entity.CcEntry
	for _, entry := range entries {
		if entry.IsForModel(model) {
			result = append(result, entry)
		}
	}

	return result, nil
}

// FindByProjectAndDateRange returns cc entries for a project within a date range
func (r *JSONLCcRepository) FindByProjectAndDateRange(projectPath string, start, end time.Time) ([]*entity.CcEntry, error) {
	entries, err := r.loadAllEntries()
	if err != nil {
		return nil, err
	}

	var result []*entity.CcEntry
	for _, entry := range entries {
		if entry.IsForProject(projectPath) && entry.IsInDateRange(start, end) {
			result = append(result, entry)
		}
	}

	return result, nil
}

// ExistsByID checks if a cc entry exists with the given ID
func (r *JSONLCcRepository) ExistsByID(id string) (bool, error) {
	entries, err := r.loadAllEntries()
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if entry.ID() == id {
			return true, nil
		}
	}

	return false, nil
}

// ExistsByMessageID checks if a cc entry exists with the given message ID
func (r *JSONLCcRepository) ExistsByMessageID(messageID string) (bool, error) {
	entries, err := r.loadAllEntries()
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if entry.MessageID() == messageID {
			return true, nil
		}
	}

	return false, nil
}

// ExistsByRequestID checks if a cc entry exists with the given request ID
func (r *JSONLCcRepository) ExistsByRequestID(requestID string) (bool, error) {
	entries, err := r.loadAllEntries()
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if entry.RequestID() == requestID {
			return true, nil
		}
	}

	return false, nil
}

// CountAll returns the total number of cc entries
func (r *JSONLCcRepository) CountAll() (int, error) {
	entries, err := r.loadAllEntries()
	if err != nil {
		return 0, err
	}

	return len(entries), nil
}

// CountByDateRange returns the number of entries within a date range
func (r *JSONLCcRepository) CountByDateRange(start, end time.Time) (int, error) {
	entries, err := r.loadAllEntries()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, entry := range entries {
		if entry.IsInDateRange(start, end) {
			count++
		}
	}

	return count, nil
}

// GetDistinctProjects returns all unique project paths
func (r *JSONLCcRepository) GetDistinctProjects() ([]string, error) {
	entries, err := r.loadAllEntries()
	if err != nil {
		return nil, err
	}

	projects := make(map[string]bool)
	for _, entry := range entries {
		projects[entry.ProjectPath()] = true
	}

	result := make([]string, 0, len(projects))
	for project := range projects {
		result = append(result, project)
	}

	return result, nil
}

// GetDistinctModels returns all unique model names
func (r *JSONLCcRepository) GetDistinctModels() ([]string, error) {
	entries, err := r.loadAllEntries()
	if err != nil {
		return nil, err
	}

	models := make(map[string]bool)
	for _, entry := range entries {
		if model := entry.Model(); model != "" {
			models[model] = true
		}
	}

	result := make([]string, 0, len(models))
	for model := range models {
		result = append(result, model)
	}

	return result, nil
}

// GetDistinctSessions returns all unique session IDs
func (r *JSONLCcRepository) GetDistinctSessions() ([]string, error) {
	entries, err := r.loadAllEntries()
	if err != nil {
		return nil, err
	}

	sessions := make(map[string]bool)
	for _, entry := range entries {
		sessions[entry.SessionID()] = true
	}

	result := make([]string, 0, len(sessions))
	for session := range sessions {
		result = append(result, session)
	}

	return result, nil
}

// GetDateRange returns the earliest and latest dates with cc entries
func (r *JSONLCcRepository) GetDateRange() (start, end time.Time, err error) {
	entries, err := r.loadAllEntries()
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	if len(entries) == 0 {
		return time.Time{}, time.Time{}, nil
	}

	start = entries[0].Timestamp()
	end = entries[0].Timestamp()

	for _, entry := range entries {
		timestamp := entry.Timestamp()
		if timestamp.Before(start) {
			start = timestamp
		}
		if timestamp.After(end) {
			end = timestamp
		}
	}

	return start, end, nil
}

// Write operations (not implemented for read-only JSONL repository)

// Save persists a cc entry (not implemented)
func (r *JSONLCcRepository) Save(entry *entity.CcEntry) error {
	return fmt.Errorf("save operation not supported for JSONL repository")
}

// SaveAll persists multiple cc entries (not implemented)
func (r *JSONLCcRepository) SaveAll(entries []*entity.CcEntry) error {
	return fmt.Errorf("save operation not supported for JSONL repository")
}

// DeleteByID deletes a cc entry by ID (not implemented)
func (r *JSONLCcRepository) DeleteByID(id string) error {
	return fmt.Errorf("delete operation not supported for JSONL repository")
}

// DeleteByDateRange deletes entries within a date range (not implemented)
func (r *JSONLCcRepository) DeleteByDateRange(start, end time.Time) error {
	return fmt.Errorf("delete operation not supported for JSONL repository")
}

// ccData represents the raw cc data parsed from JSONL files
type ccData struct {
	Timestamp string `json:"timestamp"`
	Version   string `json:"version,omitempty"`
	Message   struct {
		Usage struct {
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
		} `json:"usage"`
		Model string `json:"model,omitempty"`
		ID    string `json:"id,omitempty"`
	} `json:"message"`
	CostUSD   *float64 `json:"costUSD,omitempty"`
	RequestID string   `json:"requestId,omitempty"`
}

// Local variable for line number tracking during file processing
var lineNum int
