package impl

import (
	"fmt"
	"time"

	"github.com/ca-srg/tosage/domain/entity"
	"github.com/ca-srg/tosage/domain/repository"
	usecase "github.com/ca-srg/tosage/usecase/interface"
)

// CcServiceImpl implements the CcService interface
type CcServiceImpl struct {
	ccRepo          repository.CcRepository
	loadCcData      *LoadCcDataUseCase
	timezoneService repository.TimezoneService
}

// NewCcServiceImpl creates a new instance of CcServiceImpl
func NewCcServiceImpl(
	ccRepo repository.CcRepository,
	timezoneService repository.TimezoneService,
) *CcServiceImpl {
	return &CcServiceImpl{
		ccRepo:          ccRepo,
		loadCcData:      NewLoadCcDataUseCase(ccRepo),
		timezoneService: timezoneService,
	}
}

// CalculateDailyTokens calculates total token count for a specific date
func (s *CcServiceImpl) CalculateDailyTokens(date time.Time) (int, error) {
	// If timezone service is available, use timezone-aware method
	if s.timezoneService != nil {
		return s.CalculateDailyTokensInUserTimezone(date)
	}

	// Get entries for the specific date
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	entries, err := s.ccRepo.FindByDateRange(startOfDay, endOfDay)
	if err != nil {
		return 0, fmt.Errorf("failed to get entries for date: %w", err)
	}

	// Calculate total tokens
	totalTokens := 0
	for _, entry := range entries {
		tokens := entry.TotalTokens()
		totalTokens += tokens
	}

	return totalTokens, nil
}

// CalculateTodayTokens calculates total token count for today
func (s *CcServiceImpl) CalculateTodayTokens() (int, error) {
	return s.CalculateDailyTokens(time.Now())
}

// CalculateTokenStats calculates aggregated token statistics
func (s *CcServiceImpl) CalculateTokenStats(filter usecase.TokenStatsFilter) (*usecase.TokenStatsResult, error) {
	// Get filtered entries
	entries, err := s.getFilteredEntries(filter.StartDate, filter.EndDate, filter.ProjectPath, filter.Model, filter.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get filtered entries: %w", err)
	}

	if len(entries) == 0 {
		return &usecase.TokenStatsResult{
			Currency: "USD",
		}, nil
	}

	// Calculate stats without cost
	inputTokens := 0
	outputTokens := 0
	cacheCreationTokens := 0
	cacheReadTokens := 0
	totalTokens := 0

	for _, entry := range entries {
		stats := entry.TokenStats()
		inputTokens += stats.InputTokens()
		outputTokens += stats.OutputTokens()
		cacheCreationTokens += stats.CacheCreationTokens()
		cacheReadTokens += stats.CacheReadTokens()
		totalTokens += stats.TotalTokens()
	}

	// Get date range
	var dateRange usecase.DateRange
	if filter.StartDate != nil && filter.EndDate != nil {
		dateRange = usecase.DateRange{
			Start: *filter.StartDate,
			End:   *filter.EndDate,
			Days:  int(filter.EndDate.Sub(*filter.StartDate).Hours()/24) + 1,
		}
	} else if len(entries) > 0 {
		// Calculate from entries
		minDate := entries[0].Timestamp()
		maxDate := entries[0].Timestamp()
		for _, entry := range entries {
			if entry.Timestamp().Before(minDate) {
				minDate = entry.Timestamp()
			}
			if entry.Timestamp().After(maxDate) {
				maxDate = entry.Timestamp()
			}
		}
		dateRange = usecase.DateRange{
			Start: minDate,
			End:   maxDate,
			Days:  int(maxDate.Sub(minDate).Hours()/24) + 1,
		}
	}

	return &usecase.TokenStatsResult{
		InputTokens:         inputTokens,
		OutputTokens:        outputTokens,
		CacheCreationTokens: cacheCreationTokens,
		CacheReadTokens:     cacheReadTokens,
		TotalTokens:         totalTokens,
		Cost:                0,
		Currency:            "USD",
		EntryCount:          len(entries),
		DateRange:           dateRange,
	}, nil
}

// CalculateCostBreakdown calculates cost breakdown by various dimensions
func (s *CcServiceImpl) CalculateCostBreakdown(filter usecase.CostBreakdownFilter) (*usecase.CostBreakdownResult, error) {
	// Return empty result as we don't calculate costs anymore
	return &usecase.CostBreakdownResult{
		Breakdowns: []usecase.CostBreakdownItem{},
		Total:      usecase.TokenStatsResult{Currency: "USD"},
	}, nil
}

// CalculateModelBreakdown calculates usage breakdown by model
func (s *CcServiceImpl) CalculateModelBreakdown(filter usecase.ModelBreakdownFilter) (*usecase.ModelBreakdownResult, error) {
	// Get filtered entries
	entries, err := s.getFilteredEntries(filter.StartDate, filter.EndDate, filter.ProjectPath, "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to get filtered entries: %w", err)
	}

	// Group by model
	modelStats := make(map[string]*struct {
		inputTokens         int
		outputTokens        int
		cacheCreationTokens int
		cacheReadTokens     int
		totalTokens         int
		entryCount          int
	})

	for _, entry := range entries {
		model := entry.Model()
		if _, exists := modelStats[model]; !exists {
			modelStats[model] = &struct {
				inputTokens         int
				outputTokens        int
				cacheCreationTokens int
				cacheReadTokens     int
				totalTokens         int
				entryCount          int
			}{}
		}

		stats := entry.TokenStats()
		modelStats[model].inputTokens += stats.InputTokens()
		modelStats[model].outputTokens += stats.OutputTokens()
		modelStats[model].cacheCreationTokens += stats.CacheCreationTokens()
		modelStats[model].cacheReadTokens += stats.CacheReadTokens()
		modelStats[model].totalTokens += stats.TotalTokens()
		modelStats[model].entryCount++
	}

	// Calculate totals
	totalInputTokens := 0
	totalOutputTokens := 0
	totalCacheCreationTokens := 0
	totalCacheReadTokens := 0
	totalTokens := 0

	for _, stats := range modelStats {
		totalInputTokens += stats.inputTokens
		totalOutputTokens += stats.outputTokens
		totalCacheCreationTokens += stats.cacheCreationTokens
		totalCacheReadTokens += stats.cacheReadTokens
		totalTokens += stats.totalTokens
	}

	// Build result
	result := &usecase.ModelBreakdownResult{
		Total: usecase.TokenStatsResult{
			InputTokens:         totalInputTokens,
			OutputTokens:        totalOutputTokens,
			CacheCreationTokens: totalCacheCreationTokens,
			CacheReadTokens:     totalCacheReadTokens,
			TotalTokens:         totalTokens,
			Cost:                0,
			Currency:            "USD",
			EntryCount:          len(entries),
		},
		Models: make([]usecase.ModelBreakdownItem, 0, len(modelStats)),
	}

	for model, stats := range modelStats {
		tokenPercentage := 0.0
		if totalTokens > 0 {
			tokenPercentage = (float64(stats.totalTokens) / float64(totalTokens)) * 100
		}

		result.Models = append(result.Models, usecase.ModelBreakdownItem{
			ModelName:           model,
			InputTokens:         stats.inputTokens,
			OutputTokens:        stats.outputTokens,
			CacheCreationTokens: stats.cacheCreationTokens,
			CacheReadTokens:     stats.cacheReadTokens,
			TotalTokens:         stats.totalTokens,
			Cost:                0,
			Currency:            "USD",
			EntryCount:          stats.entryCount,
			TokenPercentage:     tokenPercentage,
			CostPercentage:      0,
		})
	}

	return result, nil
}

// CalculateDateBreakdown calculates usage breakdown by date
func (s *CcServiceImpl) CalculateDateBreakdown(filter usecase.DateBreakdownFilter) (*usecase.DateBreakdownResult, error) {
	// Get filtered entries
	entries, err := s.getFilteredEntries(filter.StartDate, filter.EndDate, filter.ProjectPath, filter.Model, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get filtered entries: %w", err)
	}

	// Group by date
	dateStats := make(map[string]*struct {
		inputTokens         int
		outputTokens        int
		cacheCreationTokens int
		cacheReadTokens     int
		totalTokens         int
		entryCount          int
	})

	for _, entry := range entries {
		date := entry.Timestamp().Format("2006-01-02")
		if _, exists := dateStats[date]; !exists {
			dateStats[date] = &struct {
				inputTokens         int
				outputTokens        int
				cacheCreationTokens int
				cacheReadTokens     int
				totalTokens         int
				entryCount          int
			}{}
		}

		stats := entry.TokenStats()
		dateStats[date].inputTokens += stats.InputTokens()
		dateStats[date].outputTokens += stats.OutputTokens()
		dateStats[date].cacheCreationTokens += stats.CacheCreationTokens()
		dateStats[date].cacheReadTokens += stats.CacheReadTokens()
		dateStats[date].totalTokens += stats.TotalTokens()
		dateStats[date].entryCount++
	}

	// Calculate totals
	totalInputTokens := 0
	totalOutputTokens := 0
	totalCacheCreationTokens := 0
	totalCacheReadTokens := 0
	totalTokens := 0

	for _, stats := range dateStats {
		totalInputTokens += stats.inputTokens
		totalOutputTokens += stats.outputTokens
		totalCacheCreationTokens += stats.cacheCreationTokens
		totalCacheReadTokens += stats.cacheReadTokens
		totalTokens += stats.totalTokens
	}

	// Build result
	result := &usecase.DateBreakdownResult{
		Total: usecase.TokenStatsResult{
			InputTokens:         totalInputTokens,
			OutputTokens:        totalOutputTokens,
			CacheCreationTokens: totalCacheCreationTokens,
			CacheReadTokens:     totalCacheReadTokens,
			TotalTokens:         totalTokens,
			Cost:                0,
			Currency:            "USD",
			EntryCount:          len(entries),
		},
		Dates: make([]usecase.DateBreakdownItem, 0, len(dateStats)),
	}

	for date, stats := range dateStats {
		result.Dates = append(result.Dates, usecase.DateBreakdownItem{
			Date:                date,
			InputTokens:         stats.inputTokens,
			OutputTokens:        stats.outputTokens,
			CacheCreationTokens: stats.cacheCreationTokens,
			CacheReadTokens:     stats.cacheReadTokens,
			TotalTokens:         stats.totalTokens,
			Cost:                0,
			Currency:            "USD",
			EntryCount:          stats.entryCount,
		})
	}

	return result, nil
}

// LoadCcData loads usage data with optional filters
func (s *CcServiceImpl) LoadCcData(filter usecase.CcDataFilter) (*usecase.CcDataResult, error) {
	return s.loadCcData.Execute(filter)
}

// GetCcSummary returns a summary of cc statistics
func (s *CcServiceImpl) GetCcSummary(filter usecase.CcSummaryFilter) (*usecase.CcSummaryResult, error) {
	// Get filtered entries
	entries, err := s.getFilteredEntries(filter.StartDate, filter.EndDate, filter.ProjectPath, filter.Model, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get filtered entries: %w", err)
	}

	if len(entries) == 0 {
		return &usecase.CcSummaryResult{
			Currency: "USD",
		}, nil
	}

	// Calculate total stats without cost
	inputTokens := 0
	outputTokens := 0
	cacheCreationTokens := 0
	cacheReadTokens := 0
	totalTokens := 0

	for _, entry := range entries {
		stats := entry.TokenStats()
		inputTokens += stats.InputTokens()
		outputTokens += stats.OutputTokens()
		cacheCreationTokens += stats.CacheCreationTokens()
		cacheReadTokens += stats.CacheReadTokens()
		totalTokens += stats.TotalTokens()
	}

	// Get unique counts
	projects := make(map[string]bool)
	models := make(map[string]bool)
	sessions := make(map[string]bool)
	modelCounts := make(map[string]int)
	projectCounts := make(map[string]int)

	for _, entry := range entries {
		projects[entry.ProjectPath()] = true
		models[entry.Model()] = true
		sessions[entry.SessionID()] = true
		modelCounts[entry.Model()]++
		projectCounts[entry.ProjectPath()]++
	}

	// Find most used model and project
	mostUsedModel := ""
	maxModelCount := 0
	for model, count := range modelCounts {
		if count > maxModelCount {
			maxModelCount = count
			mostUsedModel = model
		}
	}

	mostActiveProject := ""
	maxProjectCount := 0
	for project, count := range projectCounts {
		if count > maxProjectCount {
			maxProjectCount = count
			mostActiveProject = project
		}
	}

	// Calculate date range
	minDate := entries[0].Timestamp()
	maxDate := entries[0].Timestamp()
	for _, entry := range entries {
		if entry.Timestamp().Before(minDate) {
			minDate = entry.Timestamp()
		}
		if entry.Timestamp().After(maxDate) {
			maxDate = entry.Timestamp()
		}
	}

	dateRange := usecase.DateRange{
		Start: minDate,
		End:   maxDate,
		Days:  int(maxDate.Sub(minDate).Hours()/24) + 1,
	}

	// Calculate averages
	avgDailyTokens := 0
	avgDailyCost := 0.0
	if dateRange.Days > 0 {
		avgDailyTokens = totalTokens / dateRange.Days
		avgDailyCost = 0
	}

	// Calculate token distribution
	tokenDist := usecase.TokenDistribution{}
	if totalTokens > 0 {
		tokenDist.InputPercentage = (float64(inputTokens) / float64(totalTokens)) * 100
		tokenDist.OutputPercentage = (float64(outputTokens) / float64(totalTokens)) * 100
		tokenDist.CacheCreationPercentage = (float64(cacheCreationTokens) / float64(totalTokens)) * 100
		tokenDist.CacheReadPercentage = (float64(cacheReadTokens) / float64(totalTokens)) * 100
	}

	return &usecase.CcSummaryResult{
		TotalTokens:        totalTokens,
		TotalCost:          0,
		Currency:           "USD",
		EntryCount:         len(entries),
		UniqueProjects:     len(projects),
		UniqueModels:       len(models),
		UniqueSessions:     len(sessions),
		DateRange:          dateRange,
		AverageDailyTokens: avgDailyTokens,
		AverageDailyCost:   avgDailyCost,
		MostUsedModel:      mostUsedModel,
		MostActiveProject:  mostActiveProject,
		TokenDistribution:  tokenDist,
	}, nil
}

// EstimateMonthlyCost estimates monthly cost based on recent usage
func (s *CcServiceImpl) EstimateMonthlyCost(daysToAverage int) (*usecase.CostEstimateResult, error) {
	// Return empty result as we don't calculate costs anymore
	return &usecase.CostEstimateResult{
		EstimatedMonthlyCost: 0,
		Currency:             "USD",
		BasedOnDays:          0,
		AverageDailyCost:     0,
		Confidence:           0,
	}, nil
}

// GetAvailableProjects returns list of available projects
func (s *CcServiceImpl) GetAvailableProjects() ([]string, error) {
	return s.loadCcData.GetAvailableProjects()
}

// GetAvailableModels returns list of available models
func (s *CcServiceImpl) GetAvailableModels() ([]string, error) {
	return s.loadCcData.GetAvailableModels()
}

// GetDateRange returns the date range of available data
func (s *CcServiceImpl) GetDateRange() (start, end time.Time, err error) {
	return s.ccRepo.GetDateRange()
}

// getFilteredEntries is a helper method to get filtered entries
func (s *CcServiceImpl) getFilteredEntries(
	startDate, endDate *time.Time,
	projectPath, model, sessionID string,
) ([]*entity.CcEntry, error) {
	var entries []*entity.CcEntry
	var err error

	// Apply date range filter
	if startDate != nil && endDate != nil {
		if projectPath != "" {
			entries, err = s.ccRepo.FindByProjectAndDateRange(projectPath, *startDate, *endDate)
		} else {
			entries, err = s.ccRepo.FindByDateRange(*startDate, *endDate)
		}
	} else if projectPath != "" {
		entries, err = s.ccRepo.FindByProject(projectPath)
	} else {
		entries, err = s.ccRepo.FindAll()
	}

	if err != nil {
		return nil, err
	}

	// Apply additional filters
	collection := entity.NewCcEntryCollection(entries)

	if model != "" {
		collection = collection.FilterByModel(model)
	}

	if sessionID != "" {
		collection = collection.FilterBySession(sessionID)
	}

	return collection.Entries(), nil
}

// Timezone-aware methods

// CalculateDailyTokensInUserTimezone calculates total token count for a specific date in user's timezone
func (s *CcServiceImpl) CalculateDailyTokensInUserTimezone(date time.Time) (int, error) {
	if s.timezoneService == nil {
		// Fall back to existing method if timezone service not available
		return s.CalculateDailyTokens(date)
	}

	// Get user's timezone
	userTimezone, err := s.timezoneService.GetConfiguredTimezone()
	if err != nil {
		return 0, fmt.Errorf("failed to get user timezone: %w", err)
	}

	// Get day boundaries in user's timezone
	startOfDay, endOfDay := s.timezoneService.GetDayBoundaries(date)

	// Get entries for the date range
	entries, err := s.ccRepo.FindByDateRange(startOfDay, endOfDay)
	if err != nil {
		return 0, fmt.Errorf("failed to get entries for date: %w", err)
	}

	// Create collection with timezone context
	collection := entity.NewCcEntryCollectionWithTimezone(entries, userTimezone)

	// Calculate total tokens
	totalTokens := 0
	for _, entry := range collection.Entries() {
		totalTokens += entry.TotalTokens()
	}

	return totalTokens, nil
}

// CalculateTodayTokensInUserTimezone calculates total token count for today in user's timezone
func (s *CcServiceImpl) CalculateTodayTokensInUserTimezone() (int, error) {
	if s.timezoneService == nil {
		// Fall back to existing method if timezone service not available
		return s.CalculateTodayTokens()
	}

	// Use current time in user's timezone
	return s.CalculateDailyTokensInUserTimezone(time.Now())
}

// GetDateRangeInUserTimezone returns the date range of available data in user's timezone
func (s *CcServiceImpl) GetDateRangeInUserTimezone() (start, end time.Time, err error) {
	// Get the date range in UTC
	start, end, err = s.GetDateRange()
	if err != nil {
		return start, end, err
	}

	if s.timezoneService == nil {
		// Return as-is if timezone service not available
		return start, end, nil
	}

	// Convert to user's timezone
	start = s.timezoneService.ConvertToUserTime(start)
	end = s.timezoneService.ConvertToUserTime(end)

	return start, end, nil
}
