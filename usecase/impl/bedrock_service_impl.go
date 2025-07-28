package impl

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ca-srg/tosage/domain"
	"github.com/ca-srg/tosage/domain/entity"
	"github.com/ca-srg/tosage/domain/repository"
	usecase "github.com/ca-srg/tosage/usecase/interface"
)

// BedrockServiceImpl implements the BedrockService interface
type BedrockServiceImpl struct {
	bedrockRepo repository.BedrockRepository
	config      *repository.BedrockConfig
	logger      domain.Logger

	// Cache fields
	cacheMutex   sync.RWMutex
	cachedUsage  map[string]*entity.BedrockUsage // keyed by region
	cacheExpiry  time.Time
	cacheTimeout time.Duration
}

// NewBedrockService creates a new BedrockServiceImpl instance
func NewBedrockService(
	bedrockRepo repository.BedrockRepository,
	config *repository.BedrockConfig,
	logger domain.Logger,
) usecase.BedrockService {
	return &BedrockServiceImpl{
		bedrockRepo:  bedrockRepo,
		config:       config,
		logger:       logger,
		cachedUsage:  make(map[string]*entity.BedrockUsage),
		cacheTimeout: 5 * time.Minute, // 5 minute cache
	}
}

// IsEnabled checks if Bedrock tracking is enabled in configuration
func (s *BedrockServiceImpl) IsEnabled() bool {
	return s.config.Enabled
}

// GetCurrentUsage retrieves the current Bedrock usage statistics for all configured regions
func (s *BedrockServiceImpl) GetCurrentUsage() (*entity.BedrockUsage, error) {
	if !s.IsEnabled() {
		return nil, domain.ErrBusinessRule("bedrock disabled", "Bedrock tracking is disabled in configuration")
	}

	var totalInputTokens int64
	var totalOutputTokens int64
	var totalCost float64
	var allModelMetrics []entity.BedrockModelMetric
	var primaryRegion string
	var accountID string

	// Collect usage from all configured regions
	for _, region := range s.config.Regions {
		usage, err := s.GetUsageForRegion(region)
		if err != nil {
			// Log error but continue with other regions
			s.logger.Error(context.TODO(), "Failed to get Bedrock usage for region",
				domain.NewField("region", region),
				domain.NewField("error", err.Error()))
			continue
		}

		if usage != nil && !usage.IsEmpty() {
			totalInputTokens += usage.InputTokens()
			totalOutputTokens += usage.OutputTokens()
			totalCost += usage.TotalCost()
			allModelMetrics = append(allModelMetrics, usage.ModelMetrics()...)

			// Use first non-empty region as primary
			if primaryRegion == "" {
				primaryRegion = usage.Region()
				accountID = usage.AccountID()
			}
		}
	}

	if primaryRegion == "" {
		primaryRegion = "us-east-1" // Default region
		accountID = "current-account"
	}

	// Create consolidated usage
	return entity.NewBedrockUsage(
		totalInputTokens,
		totalOutputTokens,
		totalCost,
		allModelMetrics,
		primaryRegion,
		accountID,
	)
}

// GetUsageForRegion retrieves usage statistics for a specific region
func (s *BedrockServiceImpl) GetUsageForRegion(region string) (*entity.BedrockUsage, error) {
	if !s.IsEnabled() {
		return nil, domain.ErrBusinessRule("bedrock disabled", "Bedrock tracking is disabled in configuration")
	}

	// Check cache first
	s.cacheMutex.RLock()
	if cachedUsage, exists := s.cachedUsage[region]; exists && time.Now().Before(s.cacheExpiry) {
		s.cacheMutex.RUnlock()
		return cachedUsage, nil
	}
	s.cacheMutex.RUnlock()

	// Get current date in JST
	jst, _ := time.LoadLocation("Asia/Tokyo")
	now := time.Now().In(jst)
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, jst)

	// Fetch usage from repository
	usage, err := s.bedrockRepo.GetUsageMetrics(region, startOfDay, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get Bedrock usage for region %s: %w", region, err)
	}

	// Validate usage data
	if err := usage.Validate(); err != nil {
		return nil, domain.ErrBusinessRule("usage data validation", err.Error())
	}

	// Update cache
	s.updateUsageCache(region, usage)

	return usage, nil
}

// GetDailyUsage retrieves aggregated usage for a specific date
func (s *BedrockServiceImpl) GetDailyUsage(date time.Time) (*entity.BedrockUsage, error) {
	if !s.IsEnabled() {
		return nil, domain.ErrBusinessRule("bedrock disabled", "Bedrock tracking is disabled in configuration")
	}

	var totalInputTokens int64
	var totalOutputTokens int64
	var totalCost float64
	var allModelMetrics []entity.BedrockModelMetric
	var primaryRegion string
	var accountID string

	// Collect daily usage from all configured regions
	for _, region := range s.config.Regions {
		usage, err := s.bedrockRepo.GetDailyUsage(region, date)
		if err != nil {
			// Log error but continue with other regions
			s.logger.Error(context.TODO(), "Failed to get Bedrock daily usage",
				domain.NewField("region", region),
				domain.NewField("date", date.Format("2006-01-02")),
				domain.NewField("error", err.Error()))
			continue
		}

		if usage != nil && !usage.IsEmpty() {
			totalInputTokens += usage.InputTokens()
			totalOutputTokens += usage.OutputTokens()
			totalCost += usage.TotalCost()
			allModelMetrics = append(allModelMetrics, usage.ModelMetrics()...)

			// Use first non-empty region as primary
			if primaryRegion == "" {
				primaryRegion = usage.Region()
				accountID = usage.AccountID()
			}
		}
	}

	if primaryRegion == "" {
		primaryRegion = "us-east-1" // Default region
		accountID = "current-account"
	}

	// Create consolidated daily usage
	return entity.NewBedrockUsage(
		totalInputTokens,
		totalOutputTokens,
		totalCost,
		allModelMetrics,
		primaryRegion,
		accountID,
	)
}

// GetCurrentMonthUsage retrieves usage for the current month
func (s *BedrockServiceImpl) GetCurrentMonthUsage() (*entity.BedrockUsage, error) {
	if !s.IsEnabled() {
		return nil, domain.ErrBusinessRule("bedrock disabled", "Bedrock tracking is disabled in configuration")
	}

	var totalInputTokens int64
	var totalOutputTokens int64
	var totalCost float64
	var allModelMetrics []entity.BedrockModelMetric
	var primaryRegion string
	var accountID string

	// Collect monthly usage from all configured regions
	for _, region := range s.config.Regions {
		usage, err := s.bedrockRepo.GetCurrentMonthUsage(region)
		if err != nil {
			// Log error but continue with other regions
			s.logger.Error(context.TODO(), "Failed to get Bedrock monthly usage",
				domain.NewField("region", region),
				domain.NewField("error", err.Error()))
			continue
		}

		if usage != nil && !usage.IsEmpty() {
			totalInputTokens += usage.InputTokens()
			totalOutputTokens += usage.OutputTokens()
			totalCost += usage.TotalCost()
			allModelMetrics = append(allModelMetrics, usage.ModelMetrics()...)

			// Use first non-empty region as primary
			if primaryRegion == "" {
				primaryRegion = usage.Region()
				accountID = usage.AccountID()
			}
		}
	}

	if primaryRegion == "" {
		primaryRegion = "us-east-1" // Default region
		accountID = "current-account"
	}

	// Create consolidated monthly usage
	return entity.NewBedrockUsage(
		totalInputTokens,
		totalOutputTokens,
		totalCost,
		allModelMetrics,
		primaryRegion,
		accountID,
	)
}

// CheckConnection verifies AWS credentials and CloudWatch access
func (s *BedrockServiceImpl) CheckConnection() error {
	if !s.IsEnabled() {
		return domain.ErrBusinessRule("bedrock disabled", "Bedrock tracking is disabled in configuration")
	}

	return s.bedrockRepo.CheckConnection()
}

// GetConfiguredRegions returns the list of configured regions
func (s *BedrockServiceImpl) GetConfiguredRegions() []string {
	return s.config.Regions
}

// GetAvailableRegions returns regions with Bedrock activity
func (s *BedrockServiceImpl) GetAvailableRegions() ([]string, error) {
	if !s.IsEnabled() {
		return nil, domain.ErrBusinessRule("bedrock disabled", "Bedrock tracking is disabled in configuration")
	}

	return s.bedrockRepo.ListAvailableRegions()
}

// updateUsageCache updates the cached usage data for a specific region
func (s *BedrockServiceImpl) updateUsageCache(region string, usage *entity.BedrockUsage) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.cachedUsage[region] = usage
	s.cacheExpiry = time.Now().Add(s.cacheTimeout)
}

// ClearCache clears all cached data
func (s *BedrockServiceImpl) ClearCache() {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.cachedUsage = make(map[string]*entity.BedrockUsage)
	s.cacheExpiry = time.Time{}
}

// SetCacheTimeout sets the cache timeout duration
func (s *BedrockServiceImpl) SetCacheTimeout(timeout time.Duration) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.cacheTimeout = timeout
}
