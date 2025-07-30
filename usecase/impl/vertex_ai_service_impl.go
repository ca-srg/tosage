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
	vertexAIRepo           repository.VertexAIRepository
	vertexAIMonitoringRepo repository.VertexAIRepository
	config                 *repository.VertexAIConfig

	// Cache fields
	cacheMutex   sync.RWMutex
	cachedUsage  map[string]*entity.VertexAIUsage // keyed by "projectID:location"
	cacheExpiry  time.Time
	cacheTimeout time.Duration
}

// NewVertexAIService creates a new VertexAIServiceImpl instance
func NewVertexAIService(
	vertexAIRepo repository.VertexAIRepository,
	vertexAIMonitoringRepo repository.VertexAIRepository,
	config *repository.VertexAIConfig,
) usecase.VertexAIService {
	return &VertexAIServiceImpl{
		vertexAIRepo:           vertexAIRepo,
		vertexAIMonitoringRepo: vertexAIMonitoringRepo,
		config:                 config,
		cachedUsage:            make(map[string]*entity.VertexAIUsage),
		cacheTimeout:           5 * time.Minute, // 5 minute cache
	}
}

// IsEnabled checks if Vertex AI tracking is enabled in configuration
func (s *VertexAIServiceImpl) IsEnabled() bool {
	return s.config.Enabled
}

// GetCurrentUsage retrieves the current Vertex AI usage statistics for the configured project
func (s *VertexAIServiceImpl) GetCurrentUsage() (*entity.VertexAIUsage, error) {
	if !s.IsEnabled() {
		return nil, domain.ErrBusinessRule("vertex ai disabled", "Vertex AI tracking is disabled in configuration")
	}

	// If no project ID is configured, return empty usage
	if s.config.ProjectID == "" {
		return entity.NewVertexAIUsage(0, 0, 0, []entity.VertexAIModelMetric{}, "unknown", "unknown")
	}

	// Get usage without location filter
	usage, err := s.GetUsageForProject(s.config.ProjectID)
	if err != nil {
		return nil, err
	}

	return usage, nil
}

// GetUsageForProject retrieves usage statistics for a specific project
func (s *VertexAIServiceImpl) GetUsageForProject(projectID string) (*entity.VertexAIUsage, error) {
	if !s.IsEnabled() {
		return nil, domain.ErrBusinessRule("vertex ai disabled", "Vertex AI tracking is disabled in configuration")
	}

	cacheKey := projectID

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
	usage, err := s.vertexAIMonitoringRepo.GetUsageMetrics(projectID, startOfDay, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get Vertex AI usage for project %s: %w", projectID, err)
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

	// If no project ID is configured, return error
	if s.config.ProjectID == "" {
		return nil, domain.ErrBusinessRule("project id required", "Vertex AI project ID is required but not configured")
	}

	// Get daily usage without location filter
	usage, err := s.vertexAIRepo.GetDailyUsage(s.config.ProjectID, date)
	if err != nil {
		return nil, err
	}

	return usage, nil
}

// GetCurrentMonthUsage retrieves usage for the current month
func (s *VertexAIServiceImpl) GetCurrentMonthUsage() (*entity.VertexAIUsage, error) {
	if !s.IsEnabled() {
		return nil, domain.ErrBusinessRule("vertex ai disabled", "Vertex AI tracking is disabled in configuration")
	}

	// If no project ID is configured, return empty usage
	if s.config.ProjectID == "" {
		return entity.NewVertexAIUsage(0, 0, 0, []entity.VertexAIModelMetric{}, "unknown", "unknown")
	}

	// Get monthly usage without location filter
	usage, err := s.vertexAIRepo.GetCurrentMonthUsage(s.config.ProjectID)
	if err != nil {
		return nil, err
	}

	return usage, nil
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
