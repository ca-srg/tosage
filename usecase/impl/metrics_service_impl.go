package impl

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ca-srg/tosage/domain"
	"github.com/ca-srg/tosage/domain/repository"
	"github.com/ca-srg/tosage/infrastructure/config"
	usecase "github.com/ca-srg/tosage/usecase/interface"
)

// MetricsServiceImpl implements the MetricsService interface
type MetricsServiceImpl struct {
	ccService     usecase.CcService
	cursorService usecase.CursorService
	metricsRepo   repository.MetricsRepository
	config        *config.PrometheusConfig
	ticker        *time.Ticker
	stopChan      chan struct{}
	wg            sync.WaitGroup
	mu            sync.Mutex
	isRunning     bool
	logger        domain.Logger
}

// NewMetricsServiceImpl creates a new metrics service implementation
func NewMetricsServiceImpl(
	ccService usecase.CcService,
	cursorService usecase.CursorService,
	metricsRepo repository.MetricsRepository,
	config *config.PrometheusConfig,
	logger domain.Logger,
) usecase.MetricsService {
	return &MetricsServiceImpl{
		ccService:     ccService,
		cursorService: cursorService,
		metricsRepo:   metricsRepo,
		config:        config,
		stopChan:      make(chan struct{}),
		isRunning:     false,
		logger:        logger,
	}
}

// StartPeriodicMetrics starts the periodic metrics collection
func (s *MetricsServiceImpl) StartPeriodicMetrics() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return usecase.NewMetricsServiceError("already_running", "metrics service is already running")
	}

	// Check if config is nil
	if s.config == nil {
		return usecase.NewMetricsServiceError("invalid_config", "prometheus config is nil")
	}

	// Send initial metrics
	if err := s.sendMetrics(); err != nil {
		ctx := context.Background()
		s.logger.Warn(ctx, "Failed to send initial metrics", domain.NewField("error", err.Error()))
		// Don't fail startup due to metrics error
	}

	// Start ticker for periodic collection
	s.ticker = time.NewTicker(time.Duration(s.config.IntervalSec) * time.Second)
	s.isRunning = true

	// Start goroutine for periodic metrics
	s.wg.Add(1)
	go s.runPeriodicMetrics()

	return nil
}

// StopPeriodicMetrics stops the periodic metrics collection
func (s *MetricsServiceImpl) StopPeriodicMetrics() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return nil
	}

	// Stop ticker
	if s.ticker != nil {
		s.ticker.Stop()
	}

	// Signal goroutine to stop
	close(s.stopChan)

	// Wait for goroutine to finish
	s.wg.Wait()

	// Send final metrics before stopping
	if err := s.sendMetrics(); err != nil {
		ctx := context.Background()
		s.logger.Warn(ctx, "Failed to send final metrics", domain.NewField("error", err.Error()))
		// Don't fail shutdown due to metrics error
	}

	s.isRunning = false
	s.stopChan = make(chan struct{}) // Reset for potential restart

	return nil
}

// SendCurrentMetrics sends the current metrics immediately
func (s *MetricsServiceImpl) SendCurrentMetrics() error {
	return s.sendMetrics()
}

// runPeriodicMetrics runs the periodic metrics collection loop
func (s *MetricsServiceImpl) runPeriodicMetrics() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ticker.C:
			if err := s.sendMetrics(); err != nil {
				ctx := context.Background()
				s.logger.Warn(ctx, "Failed to send periodic metrics", domain.NewField("error", err.Error()))
				// Continue running even if metrics fail
			}
		case <-s.stopChan:
			return
		}
	}
}

// sendMetrics calculates and sends the current metrics
func (s *MetricsServiceImpl) sendMetrics() error {
	ctx := context.Background()

	// Claude Code metrics if ClaudeService is available
	if s.ccService != nil {
		// Calculate today's tokens
		totalTokens, err := s.ccService.CalculateTodayTokens()
		if err != nil {
			return fmt.Errorf("failed to calculate today's tokens: %w", err)
		}

		// Send metrics to Prometheus
		if err := s.metricsRepo.SendTokenMetric(totalTokens, s.config.HostLabel, "tosage_cc_token"); err != nil {
			return fmt.Errorf("failed to send token metric: %w", err)
		}

		s.logger.Info(ctx, "Successfully sent Claude Code metrics", domain.NewField("tokens", totalTokens))
	}

	// Send Cursor metrics if CursorService is available
	if s.cursorService != nil {
		// Get aggregated token usage from JST 00:00 to current time
		totalTokens, err := s.cursorService.GetAggregatedTokenUsage()
		if err != nil {
			// Log error but don't fail the entire metrics operation
			s.logger.Warn(ctx, "Failed to get Cursor token usage", domain.NewField("error", err.Error()))
		} else {
			// Send Cursor token metric
			if err := s.metricsRepo.SendTokenMetric(int(totalTokens), s.config.HostLabel, "tosage_cursor_token"); err != nil {
				// Log error but don't fail the entire metrics operation
				s.logger.Warn(ctx, "Failed to send Cursor metrics", domain.NewField("error", err.Error()))
			} else {
				s.logger.Info(ctx, "Successfully sent Cursor metrics",
					domain.NewField("total_tokens", totalTokens),
					domain.NewField("period", "JST 00:00 to now"))
			}
		}
	}

	return nil
}
