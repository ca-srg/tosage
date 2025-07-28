package impl

import (
	"fmt"
	"sync"
	"time"

	"github.com/ca-srg/tosage/domain"
	"github.com/ca-srg/tosage/domain/entity"
	"github.com/ca-srg/tosage/domain/repository"
	usecase "github.com/ca-srg/tosage/usecase/interface"
)

// VertexAIServiceImpl implements the VertexAIService interface
type VertexAIServiceImpl struct {
	vertexAIRepo repository.VertexAIRepository
	config       *repository.VertexAIConfig

	// Cache fields
	cacheMutex   sync.RWMutex
	cachedUsage  map[string]*entity.VertexAIUsage // keyed by "projectID:location"
	cacheExpiry  time.Time
	cacheTimeout time.Duration
}

// NewVertexAIService creates a new VertexAIServiceImpl instance
func NewVertexAIService(
	vertexAIRepo repository.VertexAIRepository,
	config *repository.VertexAIConfig,
) usecase.VertexAIService {
	return &VertexAIServiceImpl{
		vertexAIRepo: vertexAIRepo,
		config:       config,
		cachedUsage:  make(map[string]*entity.VertexAIUsage),
		cacheTimeout: 5 * time.Minute, // 5 minute cache
	}
}

// IsEnabled checks if Vertex AI tracking is enabled in configuration
func (s *VertexAIServiceImpl) IsEnabled() bool {
	return s.config.Enabled
}

// GetCurrentUsage retrieves the current Vertex AI usage statistics for all configured projects and locations
func (s *VertexAIServiceImpl) GetCurrentUsage() (*entity.VertexAIUsage, error) {
	if !s.IsEnabled() {
		return nil, domain.ErrBusinessRule("vertex ai disabled", "Vertex AI tracking is disabled in configuration")
	}

	var totalInputTokens int64
	var totalOutputTokens int64
	var totalCost float64
	var allModelMetrics []entity.VertexAIModelMetric
	var primaryProjectID string
	var primaryLocation string

	// If no project ID is configured, return empty usage
	if s.config.ProjectID == "" {
		return entity.NewVertexAIUsage(0, 0, 0, []entity.VertexAIModelMetric{}, "unknown", "unknown")
	}

	// Collect usage from all configured locations
	for _, location := range s.config.Locations {
		usage, err := s.GetUsageForProject(s.config.ProjectID, location)
		if err != nil {
			// Log error but continue with other locations
			continue
		}

		if usage != nil && !usage.IsEmpty() {
			totalInputTokens += usage.InputTokens()
			totalOutputTokens += usage.OutputTokens()
			totalCost += usage.TotalCost()
			allModelMetrics = append(allModelMetrics, usage.ModelMetrics()...)

			// Use first non-empty location as primary
			if primaryProjectID == "" {
				primaryProjectID = usage.ProjectID()
				primaryLocation = usage.Location()
			}
		}
	}

	if primaryProjectID == "" {
		primaryProjectID = s.config.ProjectID
		primaryLocation = "us-central1" // Default location
	}

	// Create consolidated usage
	return entity.NewVertexAIUsage(
		totalInputTokens,
		totalOutputTokens,
		totalCost,
		allModelMetrics,
		primaryProjectID,
		primaryLocation,
	)
}

// GetUsageForProject retrieves usage statistics for a specific project and location
func (s *VertexAIServiceImpl) GetUsageForProject(projectID, location string) (*entity.VertexAIUsage, error) {
	if !s.IsEnabled() {
		return nil, domain.ErrBusinessRule("vertex ai disabled", "Vertex AI tracking is disabled in configuration")
	}

	cacheKey := fmt.Sprintf("%s:%s", projectID, location)

	// Check cache first
	s.cacheMutex.RLock()
	if cachedUsage, exists := s.cachedUsage[cacheKey]; exists && time.Now().Before(s.cacheExpiry) {
		s.cacheMutex.RUnlock()
		return cachedUsage, nil
	}
	s.cacheMutex.RUnlock()

	// Get current date in JST
	jst, _ := time.LoadLocation("Asia/Tokyo")
	now := time.Now().In(jst)
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, jst)

	// Fetch usage from repository
	usage, err := s.vertexAIRepo.GetUsageMetrics(projectID, location, startOfDay, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get Vertex AI usage for project %s, location %s: %w", projectID, location, err)
	}

	// Validate usage data
	if err := usage.Validate(); err != nil {
		return nil, domain.ErrBusinessRule("usage data validation", err.Error())
	}

	// Update cache
	s.updateUsageCache(cacheKey, usage)

	return usage, nil
}

// GetDailyUsage retrieves aggregated usage for a specific date
func (s *VertexAIServiceImpl) GetDailyUsage(date time.Time) (*entity.VertexAIUsage, error) {
	if !s.IsEnabled() {
		return nil, domain.ErrBusinessRule("vertex ai disabled", "Vertex AI tracking is disabled in configuration")
	}

	var totalInputTokens int64
	var totalOutputTokens int64
	var totalCost float64
	var allModelMetrics []entity.VertexAIModelMetric
	var primaryProjectID string
	var primaryLocation string

	// If no project ID is configured, return empty usage
	if s.config.ProjectID == "" {
		return entity.NewVertexAIUsage(0, 0, 0, []entity.VertexAIModelMetric{}, "unknown", "unknown")
	}

	// Collect daily usage from all configured locations
	for _, location := range s.config.Locations {
		usage, err := s.vertexAIRepo.GetDailyUsage(s.config.ProjectID, location, date)
		if err != nil {
			// Log error but continue with other locations
			continue
		}

		if usage != nil && !usage.IsEmpty() {
			totalInputTokens += usage.InputTokens()
			totalOutputTokens += usage.OutputTokens()
			totalCost += usage.TotalCost()
			allModelMetrics = append(allModelMetrics, usage.ModelMetrics()...)

			// Use first non-empty location as primary
			if primaryProjectID == "" {
				primaryProjectID = usage.ProjectID()
				primaryLocation = usage.Location()
			}
		}
	}

	if primaryProjectID == "" {
		primaryProjectID = s.config.ProjectID
		primaryLocation = "us-central1" // Default location
	}

	// Create consolidated daily usage
	return entity.NewVertexAIUsage(
		totalInputTokens,
		totalOutputTokens,
		totalCost,
		allModelMetrics,
		primaryProjectID,
		primaryLocation,
	)
}

// GetCurrentMonthUsage retrieves usage for the current month
func (s *VertexAIServiceImpl) GetCurrentMonthUsage() (*entity.VertexAIUsage, error) {
	if !s.IsEnabled() {
		return nil, domain.ErrBusinessRule("vertex ai disabled", "Vertex AI tracking is disabled in configuration")
	}

	var totalInputTokens int64
	var totalOutputTokens int64
	var totalCost float64
	var allModelMetrics []entity.VertexAIModelMetric
	var primaryProjectID string
	var primaryLocation string

	// If no project ID is configured, return empty usage
	if s.config.ProjectID == "" {
		return entity.NewVertexAIUsage(0, 0, 0, []entity.VertexAIModelMetric{}, "unknown", "unknown")
	}

	// Collect monthly usage from all configured locations
	for _, location := range s.config.Locations {
		usage, err := s.vertexAIRepo.GetCurrentMonthUsage(s.config.ProjectID, location)
		if err != nil {
			// Log error but continue with other locations
			continue
		}

		if usage != nil && !usage.IsEmpty() {
			totalInputTokens += usage.InputTokens()
			totalOutputTokens += usage.OutputTokens()
			totalCost += usage.TotalCost()
			allModelMetrics = append(allModelMetrics, usage.ModelMetrics()...)

			// Use first non-empty location as primary
			if primaryProjectID == "" {
				primaryProjectID = usage.ProjectID()
				primaryLocation = usage.Location()
			}
		}
	}

	if primaryProjectID == "" {
		primaryProjectID = s.config.ProjectID
		primaryLocation = "us-central1" // Default location
	}

	// Create consolidated monthly usage
	return entity.NewVertexAIUsage(
		totalInputTokens,
		totalOutputTokens,
		totalCost,
		allModelMetrics,
		primaryProjectID,
		primaryLocation,
	)
}

// CheckConnection verifies Google Cloud credentials and Cloud Monitoring access
func (s *VertexAIServiceImpl) CheckConnection() error {
	if !s.IsEnabled() {
		return domain.ErrBusinessRule("vertex ai disabled", "Vertex AI tracking is disabled in configuration")
	}

	return s.vertexAIRepo.CheckConnection()
}

// GetConfiguredProjects returns the list of configured project IDs
func (s *VertexAIServiceImpl) GetConfiguredProjects() []string {
	if s.config.ProjectID == "" {
		return []string{}
	}
	return []string{s.config.ProjectID}
}

// GetConfiguredLocations returns the list of configured locations
func (s *VertexAIServiceImpl) GetConfiguredLocations() []string {
	return s.config.Locations
}

// GetAvailableLocations returns locations with Vertex AI activity for a project
func (s *VertexAIServiceImpl) GetAvailableLocations(projectID string) ([]string, error) {
	if !s.IsEnabled() {
		return nil, domain.ErrBusinessRule("vertex ai disabled", "Vertex AI tracking is disabled in configuration")
	}

	return s.vertexAIRepo.ListAvailableLocations(projectID)
}

// updateUsageCache updates the cached usage data for a specific project and location
func (s *VertexAIServiceImpl) updateUsageCache(cacheKey string, usage *entity.VertexAIUsage) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.cachedUsage[cacheKey] = usage
	s.cacheExpiry = time.Now().Add(s.cacheTimeout)
}

// ClearCache clears all cached data
func (s *VertexAIServiceImpl) ClearCache() {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.cachedUsage = make(map[string]*entity.VertexAIUsage)
	s.cacheExpiry = time.Time{}
}

// SetCacheTimeout sets the cache timeout duration
func (s *VertexAIServiceImpl) SetCacheTimeout(timeout time.Duration) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.cacheTimeout = timeout
}