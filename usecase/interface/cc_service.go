package usecase

import (
	"time"
)

// CcService defines the interface for cc-related use cases
type CcService interface {
	// CalculateDailyTokens calculates total token count for a specific date
	CalculateDailyTokens(date time.Time) (int, error)

	// CalculateTodayTokens calculates total token count for today
	CalculateTodayTokens() (int, error)

	// CalculateTokenStats calculates aggregated token statistics
	CalculateTokenStats(filter TokenStatsFilter) (*TokenStatsResult, error)

	// CalculateCostBreakdown calculates cost breakdown by various dimensions
	CalculateCostBreakdown(filter CostBreakdownFilter) (*CostBreakdownResult, error)

	// CalculateModelBreakdown calculates cc breakdown by model
	CalculateModelBreakdown(filter ModelBreakdownFilter) (*ModelBreakdownResult, error)

	// CalculateDateBreakdown calculates cc breakdown by date
	CalculateDateBreakdown(filter DateBreakdownFilter) (*DateBreakdownResult, error)

	// LoadCcData loads cc data with optional filters
	LoadCcData(filter CcDataFilter) (*CcDataResult, error)

	// GetCcSummary returns a summary of cc statistics
	GetCcSummary(filter CcSummaryFilter) (*CcSummaryResult, error)

	// EstimateMonthlyCost estimates monthly cost based on recent cc
	EstimateMonthlyCost(daysToAverage int) (*CostEstimateResult, error)

	// GetAvailableProjects returns list of available projects
	GetAvailableProjects() ([]string, error)

	// GetAvailableModels returns list of available models
	GetAvailableModels() ([]string, error)

	// GetDateRange returns the date range of available data
	GetDateRange() (start, end time.Time, err error)

	// Timezone-aware methods

	// CalculateDailyTokensInUserTimezone calculates total token count for a specific date in user's timezone
	CalculateDailyTokensInUserTimezone(date time.Time) (int, error)

	// CalculateTodayTokensInUserTimezone calculates total token count for today in user's timezone
	CalculateTodayTokensInUserTimezone() (int, error)

	// GetDateRangeInUserTimezone returns the date range of available data in user's timezone
	GetDateRangeInUserTimezone() (start, end time.Time, err error)
}

// TokenStatsFilter defines filters for token statistics calculation
type TokenStatsFilter struct {
	StartDate   *time.Time
	EndDate     *time.Time
	ProjectPath string
	Model       string
	SessionID   string
}

// TokenStatsResult contains the result of token statistics calculation
type TokenStatsResult struct {
	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int
	CacheReadTokens     int
	TotalTokens         int
	Cost                float64
	Currency            string
	EntryCount          int
	DateRange           DateRange
}

// CostBreakdownFilter defines filters for cost breakdown calculation
type CostBreakdownFilter struct {
	StartDate   *time.Time
	EndDate     *time.Time
	ProjectPath string
	GroupBy     GroupByType
}

// GroupByType defines how to group the breakdown
type GroupByType string

const (
	GroupByModel   GroupByType = "model"
	GroupByDate    GroupByType = "date"
	GroupByProject GroupByType = "project"
	GroupBySession GroupByType = "session"
)

// CostBreakdownResult contains the result of cost breakdown calculation
type CostBreakdownResult struct {
	Breakdowns []CostBreakdownItem
	Total      TokenStatsResult
}

// CostBreakdownItem represents a single item in cost breakdown
type CostBreakdownItem struct {
	Key                 string // Model name, date, project path, or session ID
	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int
	CacheReadTokens     int
	TotalTokens         int
	Cost                float64
	Currency            string
	EntryCount          int
	Percentage          float64 // Percentage of total cost
}

// ModelBreakdownFilter defines filters for model breakdown
type ModelBreakdownFilter struct {
	StartDate   *time.Time
	EndDate     *time.Time
	ProjectPath string
}

// ModelBreakdownResult contains the result of model breakdown
type ModelBreakdownResult struct {
	Models []ModelBreakdownItem
	Total  TokenStatsResult
}

// ModelBreakdownItem represents cc for a single model
type ModelBreakdownItem struct {
	ModelName           string
	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int
	CacheReadTokens     int
	TotalTokens         int
	Cost                float64
	Currency            string
	EntryCount          int
	TokenPercentage     float64
	CostPercentage      float64
}

// DateBreakdownFilter defines filters for date breakdown
type DateBreakdownFilter struct {
	StartDate   *time.Time
	EndDate     *time.Time
	ProjectPath string
	Model       string
}

// DateBreakdownResult contains the result of date breakdown
type DateBreakdownResult struct {
	Dates []DateBreakdownItem
	Total TokenStatsResult
}

// DateBreakdownItem represents cc for a single date
type DateBreakdownItem struct {
	Date                string // YYYY-MM-DD format
	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int
	CacheReadTokens     int
	TotalTokens         int
	Cost                float64
	Currency            string
	EntryCount          int
}

// CcDataFilter defines filters for loading cc data
type CcDataFilter struct {
	StartDate   *time.Time
	EndDate     *time.Time
	ProjectPath string
	Model       string
	SessionID   string
	Limit       int
	Offset      int
}

// CcDataResult contains loaded cc data
type CcDataResult struct {
	Entries    []CcDataEntry
	TotalCount int
	HasMore    bool
}

// CcDataEntry represents a single cc entry
type CcDataEntry struct {
	ID                  string
	Timestamp           time.Time
	Date                string
	SessionID           string
	ProjectPath         string
	Model               string
	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int
	CacheReadTokens     int
	TotalTokens         int
	Cost                float64
	Currency            string
	Version             string
	MessageID           string
	RequestID           string
}

// CcSummaryFilter defines filters for cc summary
type CcSummaryFilter struct {
	StartDate   *time.Time
	EndDate     *time.Time
	ProjectPath string
	Model       string
}

// CcSummaryResult contains cc summary information
type CcSummaryResult struct {
	TotalTokens        int
	TotalCost          float64
	Currency           string
	EntryCount         int
	UniqueProjects     int
	UniqueModels       int
	UniqueSessions     int
	DateRange          DateRange
	AverageDailyTokens int
	AverageDailyCost   float64
	MostUsedModel      string
	MostActiveProject  string
	TokenDistribution  TokenDistribution
}

// DateRange represents a date range
type DateRange struct {
	Start time.Time
	End   time.Time
	Days  int
}

// TokenDistribution represents the distribution of token types
type TokenDistribution struct {
	InputPercentage         float64
	OutputPercentage        float64
	CacheCreationPercentage float64
	CacheReadPercentage     float64
}

// CostEstimateResult contains monthly cost estimate
type CostEstimateResult struct {
	EstimatedMonthlyCost float64
	Currency             string
	BasedOnDays          int
	AverageDailyCost     float64
	Confidence           float64 // 0-1, based on data availability
}

// UseCaseError represents an error from use case operations
type UseCaseError struct {
	Code    string
	Message string
	Details map[string]interface{}
}

func (e *UseCaseError) Error() string {
	return e.Message
}

// NewUseCaseError creates a new use case error
func NewUseCaseError(code, message string) *UseCaseError {
	return &UseCaseError{
		Code:    code,
		Message: message,
		Details: make(map[string]interface{}),
	}
}

// WithDetail adds a detail to the error
func (e *UseCaseError) WithDetail(key string, value interface{}) *UseCaseError {
	e.Details[key] = value
	return e
}
