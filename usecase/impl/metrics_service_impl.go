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
	ccService       usecase.CcService
	cursorService   usecase.CursorService
	bedrockService  usecase.BedrockService
	vertexAIService usecase.VertexAIService
	metricsRepo     repository.MetricsRepository
	config          *config.PrometheusConfig
	ticker          *time.Ticker
	stopChan        chan struct{}
	wg              sync.WaitGroup
	mu              sync.Mutex
	isRunning       bool
	logger          domain.Logger
	timezoneService repository.TimezoneService
}

// NewMetricsServiceImpl creates a new metrics service implementation
func NewMetricsServiceImpl(
	ccService usecase.CcService,
	cursorService usecase.CursorService,
	bedrockService usecase.BedrockService,
	vertexAIService usecase.VertexAIService,
	metricsRepo repository.MetricsRepository,
	config *config.PrometheusConfig,
	logger domain.Logger,
	timezoneService repository.TimezoneService,
) usecase.MetricsService {
	return &MetricsServiceImpl{
		ccService:       ccService,
		cursorService:   cursorService,
		bedrockService:  bedrockService,
		vertexAIService: vertexAIService,
		metricsRepo:     metricsRepo,
		config:          config,
		stopChan:        make(chan struct{}),
		isRunning:       false,
		logger:          logger,
		timezoneService: timezoneService,
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
		if s.timezoneService != nil {
			// Send with timezone information
			timezoneInfo := s.timezoneService.GetTimezoneInfo()
			if err := s.metricsRepo.SendTokenMetricWithTimezone(totalTokens, s.config.HostLabel, "tosage_cc_token", timezoneInfo); err != nil {
				return fmt.Errorf("failed to send token metric with timezone: %w", err)
			}
		} else {
			// Fall back to sending without timezone information
			if err := s.metricsRepo.SendTokenMetric(totalTokens, s.config.HostLabel, "tosage_cc_token"); err != nil {
				return fmt.Errorf("failed to send token metric: %w", err)
			}
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
			if s.timezoneService != nil {
				// Send with timezone information
				timezoneInfo := s.timezoneService.GetTimezoneInfo()
				if err := s.metricsRepo.SendTokenMetricWithTimezone(int(totalTokens), s.config.HostLabel, "tosage_cursor_token", timezoneInfo); err != nil {
					// Log error but don't fail the entire metrics operation
					s.logger.Warn(ctx, "Failed to send Cursor metrics with timezone", domain.NewField("error", err.Error()))
				} else {
					s.logger.Info(ctx, "Successfully sent Cursor metrics",
						domain.NewField("total_tokens", totalTokens),
						domain.NewField("period", "JST 00:00 to now"))
				}
			} else {
				// Fall back to sending without timezone information
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
	}

	// Send Bedrock metrics if BedrockService is available and enabled
	if s.bedrockService != nil && s.bedrockService.IsEnabled() {
		// Get today's Bedrock usage
		jst, _ := time.LoadLocation("Asia/Tokyo")
		today := time.Now().In(jst)
		bedrockUsage, err := s.bedrockService.GetDailyUsage(today)
		if err != nil {
			// Log error but don't fail the entire metrics operation
			s.logger.Warn(ctx, "Failed to get Bedrock usage", domain.NewField("error", err.Error()))
		} else if bedrockUsage != nil && !bedrockUsage.IsEmpty() {
			// Send Bedrock token metrics (separate input/output metrics)
			if s.timezoneService != nil {
				timezoneInfo := s.timezoneService.GetTimezoneInfo()

				// Send input tokens
				if err := s.metricsRepo.SendTokenMetricWithTimezone(int(bedrockUsage.InputTokens()), s.config.HostLabel, "tosage_bedrock_input_token", timezoneInfo); err != nil {
					s.logger.Warn(ctx, "Failed to send Bedrock input token metrics", domain.NewField("error", err.Error()))
				}

				// Send output tokens
				if err := s.metricsRepo.SendTokenMetricWithTimezone(int(bedrockUsage.OutputTokens()), s.config.HostLabel, "tosage_bedrock_output_token", timezoneInfo); err != nil {
					s.logger.Warn(ctx, "Failed to send Bedrock output token metrics", domain.NewField("error", err.Error()))
				}

				// Send total tokens
				if err := s.metricsRepo.SendTokenMetricWithTimezone(int(bedrockUsage.TotalTokens()), s.config.HostLabel, "tosage_bedrock_total_token", timezoneInfo); err != nil {
					s.logger.Warn(ctx, "Failed to send Bedrock total token metrics", domain.NewField("error", err.Error()))
				} else {
					s.logger.Info(ctx, "Successfully sent Bedrock metrics",
						domain.NewField("input_tokens", bedrockUsage.InputTokens()),
						domain.NewField("output_tokens", bedrockUsage.OutputTokens()),
						domain.NewField("total_tokens", bedrockUsage.TotalTokens()),
						domain.NewField("total_cost", bedrockUsage.TotalCost()),
						domain.NewField("period", "JST today"))
				}
			} else {
				// Fall back to sending without timezone information
				if err := s.metricsRepo.SendTokenMetric(int(bedrockUsage.InputTokens()), s.config.HostLabel, "tosage_bedrock_input_token"); err != nil {
					s.logger.Warn(ctx, "Failed to send Bedrock input token metrics", domain.NewField("error", err.Error()))
				}
				if err := s.metricsRepo.SendTokenMetric(int(bedrockUsage.OutputTokens()), s.config.HostLabel, "tosage_bedrock_output_token"); err != nil {
					s.logger.Warn(ctx, "Failed to send Bedrock output token metrics", domain.NewField("error", err.Error()))
				}
				if err := s.metricsRepo.SendTokenMetric(int(bedrockUsage.TotalTokens()), s.config.HostLabel, "tosage_bedrock_total_token"); err != nil {
					s.logger.Warn(ctx, "Failed to send Bedrock total token metrics", domain.NewField("error", err.Error()))
				} else {
					s.logger.Info(ctx, "Successfully sent Bedrock metrics",
						domain.NewField("input_tokens", bedrockUsage.InputTokens()),
						domain.NewField("output_tokens", bedrockUsage.OutputTokens()),
						domain.NewField("total_tokens", bedrockUsage.TotalTokens()),
						domain.NewField("total_cost", bedrockUsage.TotalCost()),
						domain.NewField("period", "JST today"))
				}
			}
		}
	}

	// Send Vertex AI metrics if VertexAIService is available and enabled
	if s.vertexAIService != nil && s.vertexAIService.IsEnabled() {
		// Get today's Vertex AI usage
		jst, _ := time.LoadLocation("Asia/Tokyo")
		today := time.Now().In(jst)
		vertexAIUsage, err := s.vertexAIService.GetDailyUsage(today)
		if err != nil {
			// Log error but don't fail the entire metrics operation
			s.logger.Warn(ctx, "Failed to get Vertex AI usage", domain.NewField("error", err.Error()))
		} else if vertexAIUsage != nil && !vertexAIUsage.IsEmpty() {
			// Send Vertex AI token metrics (separate input/output metrics)
			if s.timezoneService != nil {
				timezoneInfo := s.timezoneService.GetTimezoneInfo()

				// Send input tokens
				if err := s.metricsRepo.SendTokenMetricWithTimezone(int(vertexAIUsage.InputTokens()), s.config.HostLabel, "tosage_vertex_ai_input_token", timezoneInfo); err != nil {
					s.logger.Warn(ctx, "Failed to send Vertex AI input token metrics", domain.NewField("error", err.Error()))
				}

				// Send output tokens
				if err := s.metricsRepo.SendTokenMetricWithTimezone(int(vertexAIUsage.OutputTokens()), s.config.HostLabel, "tosage_vertex_ai_output_token", timezoneInfo); err != nil {
					s.logger.Warn(ctx, "Failed to send Vertex AI output token metrics", domain.NewField("error", err.Error()))
				}

				// Send total tokens
				if err := s.metricsRepo.SendTokenMetricWithTimezone(int(vertexAIUsage.TotalTokens()), s.config.HostLabel, "tosage_vertex_ai_total_token", timezoneInfo); err != nil {
					s.logger.Warn(ctx, "Failed to send Vertex AI total token metrics", domain.NewField("error", err.Error()))
				} else {
					s.logger.Info(ctx, "Successfully sent Vertex AI metrics",
						domain.NewField("input_tokens", vertexAIUsage.InputTokens()),
						domain.NewField("output_tokens", vertexAIUsage.OutputTokens()),
						domain.NewField("total_tokens", vertexAIUsage.TotalTokens()),
						domain.NewField("total_cost", vertexAIUsage.TotalCost()),
						domain.NewField("period", "JST today"))
				}
			} else {
				// Fall back to sending without timezone information
				if err := s.metricsRepo.SendTokenMetric(int(vertexAIUsage.InputTokens()), s.config.HostLabel, "tosage_vertex_ai_input_token"); err != nil {
					s.logger.Warn(ctx, "Failed to send Vertex AI input token metrics", domain.NewField("error", err.Error()))
				}
				if err := s.metricsRepo.SendTokenMetric(int(vertexAIUsage.OutputTokens()), s.config.HostLabel, "tosage_vertex_ai_output_token"); err != nil {
					s.logger.Warn(ctx, "Failed to send Vertex AI output token metrics", domain.NewField("error", err.Error()))
				}
				if err := s.metricsRepo.SendTokenMetric(int(vertexAIUsage.TotalTokens()), s.config.HostLabel, "tosage_vertex_ai_total_token"); err != nil {
					s.logger.Warn(ctx, "Failed to send Vertex AI total token metrics", domain.NewField("error", err.Error()))
				} else {
					s.logger.Info(ctx, "Successfully sent Vertex AI metrics",
						domain.NewField("input_tokens", vertexAIUsage.InputTokens()),
						domain.NewField("output_tokens", vertexAIUsage.OutputTokens()),
						domain.NewField("total_tokens", vertexAIUsage.TotalTokens()),
						domain.NewField("total_cost", vertexAIUsage.TotalCost()),
						domain.NewField("period", "JST today"))
				}
			}
		}
	}

	return nil
}
