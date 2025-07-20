package impl

import (
	"fmt"
	"sync"
	"time"

	"github.com/ca-srg/tosage/domain"
	"github.com/ca-srg/tosage/domain/entity"
	"github.com/ca-srg/tosage/domain/repository"
	"github.com/ca-srg/tosage/infrastructure/config"
	usecase "github.com/ca-srg/tosage/usecase/interface"
)

// CursorServiceImpl implements the CursorService interface
type CursorServiceImpl struct {
	tokenRepo repository.CursorTokenRepository
	apiRepo   repository.CursorAPIRepository
	config    *config.CursorConfig

	// Cache fields
	cacheMutex   sync.RWMutex
	cachedUsage  *entity.CursorUsage
	cachedLimit  *repository.UsageLimitInfo
	cachedStatus *repository.UsageBasedStatus
	cacheExpiry  time.Time
}

// NewCursorService creates a new CursorServiceImpl instance
func NewCursorService(
	tokenRepo repository.CursorTokenRepository,
	apiRepo repository.CursorAPIRepository,
	config *config.CursorConfig,
) usecase.CursorService {
	return &CursorServiceImpl{
		tokenRepo: tokenRepo,
		apiRepo:   apiRepo,
		config:    config,
	}
}

// GetCurrentUsage retrieves the current Cursor usage statistics
func (s *CursorServiceImpl) GetCurrentUsage() (*entity.CursorUsage, error) {
	// Check cache first
	s.cacheMutex.RLock()
	if s.cachedUsage != nil && time.Now().Before(s.cacheExpiry) {
		usage := s.cachedUsage
		s.cacheMutex.RUnlock()
		return usage, nil
	}
	s.cacheMutex.RUnlock()

	// Get token from repository
	token, err := s.tokenRepo.GetToken()
	if err != nil {
		if domain.IsErrorCode(err, domain.ErrCodeCursorDatabase) {
			return nil, fmt.Errorf("failed to retrieve Cursor token: %w", err)
		}
		return nil, err
	}

	// Check if token is expired
	if token.IsExpired() {
		return nil, domain.ErrCursorToken("token has expired").
			WithDetails("expiresAt", token.ExpiresAt())
	}

	// Fetch usage statistics from API
	usage, err := s.apiRepo.GetUsageStats(token)
	if err != nil {
		if domain.IsErrorCode(err, domain.ErrCodeCursorAPI) {
			return nil, fmt.Errorf("failed to fetch usage statistics: %w", err)
		}
		return nil, err
	}

	// Validate usage data
	if err := usage.Validate(); err != nil {
		return nil, domain.ErrBusinessRule("usage data validation", err.Error())
	}

	// Update cache
	s.updateUsageCache(usage)

	return usage, nil
}

// GetUsageLimit retrieves the current usage limit settings
func (s *CursorServiceImpl) GetUsageLimit() (*repository.UsageLimitInfo, error) {
	// Check cache first
	s.cacheMutex.RLock()
	if s.cachedLimit != nil && time.Now().Before(s.cacheExpiry) {
		limit := s.cachedLimit
		s.cacheMutex.RUnlock()
		return limit, nil
	}
	s.cacheMutex.RUnlock()

	// Get token from repository
	token, err := s.tokenRepo.GetToken()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve Cursor token: %w", err)
	}

	// Check if token is expired
	if token.IsExpired() {
		return nil, domain.ErrCursorToken("token has expired").
			WithDetails("expiresAt", token.ExpiresAt())
	}

	// Check if user is a team member
	var teamID *int
	usage, err := s.apiRepo.GetUsageStats(token)
	if err == nil && usage.IsTeamMember() && usage.TeamInfo() != nil {
		id := usage.TeamInfo().TeamID
		teamID = &id
	}

	// Fetch usage limit from API
	limit, err := s.apiRepo.GetUsageLimit(token, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch usage limit: %w", err)
	}

	// Update cache
	s.updateLimitCache(limit)

	return limit, nil
}

// IsUsageBasedPricingEnabled checks if usage-based pricing is enabled
func (s *CursorServiceImpl) IsUsageBasedPricingEnabled() (bool, error) {
	// Check cache first
	s.cacheMutex.RLock()
	if s.cachedStatus != nil && time.Now().Before(s.cacheExpiry) {
		status := s.cachedStatus
		s.cacheMutex.RUnlock()
		return status.IsEnabled, nil
	}
	s.cacheMutex.RUnlock()

	// Get token from repository
	token, err := s.tokenRepo.GetToken()
	if err != nil {
		return false, fmt.Errorf("failed to retrieve Cursor token: %w", err)
	}

	// Check if token is expired
	if token.IsExpired() {
		return false, domain.ErrCursorToken("token has expired").
			WithDetails("expiresAt", token.ExpiresAt())
	}

	// Check if user is a team member
	var teamID *int
	usage, err := s.apiRepo.GetUsageStats(token)
	if err == nil && usage.IsTeamMember() && usage.TeamInfo() != nil {
		id := usage.TeamInfo().TeamID
		teamID = &id
	}

	// Check usage-based status from API
	status, err := s.apiRepo.CheckUsageBasedStatus(token, teamID)
	if err != nil {
		return false, fmt.Errorf("failed to check usage-based pricing status: %w", err)
	}

	// Update cache
	s.updateStatusCache(status)

	return status.IsEnabled, nil
}

// updateUsageCache updates the cached usage data
func (s *CursorServiceImpl) updateUsageCache(usage *entity.CursorUsage) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.cachedUsage = usage
	s.cacheExpiry = time.Now().Add(time.Duration(s.config.CacheTimeout) * time.Second)
}

// updateLimitCache updates the cached limit data
func (s *CursorServiceImpl) updateLimitCache(limit *repository.UsageLimitInfo) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.cachedLimit = limit
	s.cacheExpiry = time.Now().Add(time.Duration(s.config.CacheTimeout) * time.Second)
}

// updateStatusCache updates the cached status data
func (s *CursorServiceImpl) updateStatusCache(status *repository.UsageBasedStatus) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.cachedStatus = status
	s.cacheExpiry = time.Now().Add(time.Duration(s.config.CacheTimeout) * time.Second)
}

// ClearCache clears all cached data
func (s *CursorServiceImpl) ClearCache() {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.cachedUsage = nil
	s.cachedLimit = nil
	s.cachedStatus = nil
	s.cacheExpiry = time.Time{}
}

// GetAggregatedTokenUsage retrieves aggregated token usage from JST 00:00 to current time
func (s *CursorServiceImpl) GetAggregatedTokenUsage() (int64, error) {
	// Get token from repository
	token, err := s.tokenRepo.GetToken()
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve Cursor token: %w", err)
	}

	// Check if token is expired
	if token.IsExpired() {
		return 0, domain.ErrCursorToken("token has expired").
			WithDetails("expiresAt", token.ExpiresAt())
	}

	// Get aggregated token usage from API
	totalTokens, err := s.apiRepo.GetAggregatedTokenUsage(token)
	if err != nil {
		return 0, fmt.Errorf("failed to get aggregated token usage: %w", err)
	}

	return totalTokens, nil
}
