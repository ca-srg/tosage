package impl

import (
	"fmt"

	"github.com/ca-srg/tosage/domain/entity"
	"github.com/ca-srg/tosage/domain/repository"
	usecase "github.com/ca-srg/tosage/usecase/interface"
)

// LoadCcDataUseCase implements the use case for loading cc data
type LoadCcDataUseCase struct {
	ccRepo repository.CcRepository
}

// NewLoadCcDataUseCase creates a new instance of the use case
func NewLoadCcDataUseCase(ccRepo repository.CcRepository) *LoadCcDataUseCase {
	return &LoadCcDataUseCase{
		ccRepo: ccRepo,
	}
}

// Execute loads cc data based on the provided filter
func (uc *LoadCcDataUseCase) Execute(filter usecase.CcDataFilter) (*usecase.CcDataResult, error) {
	// Get entries based on filter
	entries, err := uc.getFilteredEntries(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get filtered entries: %w", err)
	}

	// Get total count for pagination
	totalCount := len(entries)

	// Apply pagination
	start := filter.Offset
	if start < 0 {
		start = 0
	}
	if start >= totalCount {
		return &usecase.CcDataResult{
			Entries:    []usecase.CcDataEntry{},
			TotalCount: totalCount,
			HasMore:    false,
		}, nil
	}

	end := start + filter.Limit
	if filter.Limit <= 0 || end > totalCount {
		end = totalCount
	}

	// Slice entries for pagination
	paginatedEntries := entries[start:end]

	// Convert to result format
	result := &usecase.CcDataResult{
		Entries:    make([]usecase.CcDataEntry, len(paginatedEntries)),
		TotalCount: totalCount,
		HasMore:    end < totalCount,
	}

	for i, entry := range paginatedEntries {
		result.Entries[i] = uc.convertToCcDataEntry(entry)
	}

	return result, nil
}

// ExecuteAll loads all cc data without pagination
func (uc *LoadCcDataUseCase) ExecuteAll() (*usecase.CcDataResult, error) {
	entries, err := uc.ccRepo.FindAll()
	if err != nil {
		return nil, fmt.Errorf("failed to find all entries: %w", err)
	}

	result := &usecase.CcDataResult{
		Entries:    make([]usecase.CcDataEntry, len(entries)),
		TotalCount: len(entries),
		HasMore:    false,
	}

	for i, entry := range entries {
		result.Entries[i] = uc.convertToCcDataEntry(entry)
	}

	return result, nil
}

// getFilteredEntries retrieves entries based on the filter
func (uc *LoadCcDataUseCase) getFilteredEntries(filter usecase.CcDataFilter) ([]*entity.CcEntry, error) {
	var entries []*entity.CcEntry
	var err error

	// Start with all entries or date-filtered entries
	if filter.StartDate != nil && filter.EndDate != nil {
		entries, err = uc.ccRepo.FindByDateRange(*filter.StartDate, *filter.EndDate)
	} else {
		entries, err = uc.ccRepo.FindAll()
	}

	if err != nil {
		return nil, err
	}

	// Apply additional filters using collection
	collection := entity.NewCcEntryCollection(entries)

	if filter.ProjectPath != "" {
		collection = collection.FilterByProject(filter.ProjectPath)
	}

	if filter.Model != "" {
		collection = collection.FilterByModel(filter.Model)
	}

	if filter.SessionID != "" {
		collection = collection.FilterBySession(filter.SessionID)
	}

	return collection.Entries(), nil
}

// convertToCcDataEntry converts domain entity to use case DTO
func (uc *LoadCcDataUseCase) convertToCcDataEntry(entry *entity.CcEntry) usecase.CcDataEntry {
	stats := entry.TokenStats()

	return usecase.CcDataEntry{
		ID:                  entry.ID(),
		Timestamp:           entry.Timestamp(),
		Date:                entry.Date(),
		SessionID:           entry.SessionID(),
		ProjectPath:         entry.ProjectPath(),
		Model:               entry.Model(),
		InputTokens:         stats.InputTokens(),
		OutputTokens:        stats.OutputTokens(),
		CacheCreationTokens: stats.CacheCreationTokens(),
		CacheReadTokens:     stats.CacheReadTokens(),
		TotalTokens:         stats.TotalTokens(),
		Cost:                0,
		Currency:            "USD",
		Version:             entry.Version(),
		MessageID:           entry.MessageID(),
		RequestID:           entry.RequestID(),
	}
}

// GetAvailableProjects returns list of unique project paths
func (uc *LoadCcDataUseCase) GetAvailableProjects() ([]string, error) {
	projects, err := uc.ccRepo.GetDistinctProjects()
	if err != nil {
		return nil, fmt.Errorf("failed to get distinct projects: %w", err)
	}
	return projects, nil
}

// GetAvailableModels returns list of unique model names
func (uc *LoadCcDataUseCase) GetAvailableModels() ([]string, error) {
	models, err := uc.ccRepo.GetDistinctModels()
	if err != nil {
		return nil, fmt.Errorf("failed to get distinct models: %w", err)
	}
	return models, nil
}

// GetAvailableSessions returns list of unique session IDs
func (uc *LoadCcDataUseCase) GetAvailableSessions() ([]string, error) {
	sessions, err := uc.ccRepo.GetDistinctSessions()
	if err != nil {
		return nil, fmt.Errorf("failed to get distinct sessions: %w", err)
	}
	return sessions, nil
}

// GetDateRange returns the date range of available data
func (uc *LoadCcDataUseCase) GetDateRange() (start, end string, err error) {
	startTime, endTime, err := uc.ccRepo.GetDateRange()
	if err != nil {
		return "", "", fmt.Errorf("failed to get date range: %w", err)
	}

	return startTime.Format("2006-01-02"), endTime.Format("2006-01-02"), nil
}

// ValidateEntry validates a cc entry
func (uc *LoadCcDataUseCase) ValidateEntry(entry *entity.CcEntry) error {
	// Check if entry already exists
	if entry.MessageID() != "" {
		exists, err := uc.ccRepo.ExistsByMessageID(entry.MessageID())
		if err != nil {
			return fmt.Errorf("failed to check message ID existence: %w", err)
		}
		if exists {
			return fmt.Errorf("entry with message ID %s already exists", entry.MessageID())
		}
	}

	if entry.RequestID() != "" {
		exists, err := uc.ccRepo.ExistsByRequestID(entry.RequestID())
		if err != nil {
			return fmt.Errorf("failed to check request ID existence: %w", err)
		}
		if exists {
			return fmt.Errorf("entry with request ID %s already exists", entry.RequestID())
		}
	}

	// Validate entry data
	if entry.IsEmpty() {
		return fmt.Errorf("entry has no token data")
	}

	return nil
}
