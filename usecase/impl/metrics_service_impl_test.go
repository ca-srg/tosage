package impl

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ca-srg/tosage/domain"
	"github.com/ca-srg/tosage/domain/entity"
	"github.com/ca-srg/tosage/domain/repository"
	"github.com/ca-srg/tosage/infrastructure/config"
	usecase "github.com/ca-srg/tosage/usecase/interface"
)

// Mock implementations

// mockLogger is a test logger that does nothing
type mockLogger struct{}

func (m *mockLogger) Debug(ctx context.Context, msg string, fields ...domain.Field) {}
func (m *mockLogger) Info(ctx context.Context, msg string, fields ...domain.Field)  {}
func (m *mockLogger) Warn(ctx context.Context, msg string, fields ...domain.Field)  {}
func (m *mockLogger) Error(ctx context.Context, msg string, fields ...domain.Field) {}
func (m *mockLogger) WithFields(fields ...domain.Field) domain.Logger               { return m }

type mockCcService struct {
	calculateTodayTokensFunc func() (int, error)
	callCount                int
	mu                       sync.Mutex
}

func (m *mockCcService) CalculateDailyTokens(date time.Time) (int, error) {
	return 0, errors.New("not implemented")
}

func (m *mockCcService) CalculateTodayTokens() (int, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.calculateTodayTokensFunc != nil {
		return m.calculateTodayTokensFunc()
	}
	return 1000, nil
}

func (m *mockCcService) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// Implement other required methods with stubs
func (m *mockCcService) CalculateTokenStats(filter usecase.TokenStatsFilter) (*usecase.TokenStatsResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCcService) CalculateCostBreakdown(filter usecase.CostBreakdownFilter) (*usecase.CostBreakdownResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCcService) CalculateModelBreakdown(filter usecase.ModelBreakdownFilter) (*usecase.ModelBreakdownResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCcService) CalculateDateBreakdown(filter usecase.DateBreakdownFilter) (*usecase.DateBreakdownResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCcService) LoadCcData(filter usecase.CcDataFilter) (*usecase.CcDataResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCcService) GetCcSummary(filter usecase.CcSummaryFilter) (*usecase.CcSummaryResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCcService) EstimateMonthlyCost(daysToAverage int) (*usecase.CostEstimateResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCcService) GetAvailableProjects() ([]string, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCcService) GetAvailableModels() ([]string, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCcService) GetDateRange() (start, end time.Time, err error) {
	return time.Time{}, time.Time{}, errors.New("not implemented")
}

func (m *mockCcService) CalculateDailyTokensInUserTimezone(date time.Time) (int, error) {
	return m.CalculateDailyTokens(date)
}

func (m *mockCcService) CalculateTodayTokensInUserTimezone() (int, error) {
	return m.CalculateTodayTokens()
}

func (m *mockCcService) GetDateRangeInUserTimezone() (start, end time.Time, err error) {
	return m.GetDateRange()
}

type mockMetricsRepository struct {
	sendTokenMetricFunc func(totalTokens int, hostLabel string, metricName string) error
	sendCount           int
	mu                  sync.Mutex
}

func (m *mockMetricsRepository) SendTokenMetric(totalTokens int, hostLabel string, metricName string) error {
	m.mu.Lock()
	m.sendCount++
	m.mu.Unlock()

	if m.sendTokenMetricFunc != nil {
		return m.sendTokenMetricFunc(totalTokens, hostLabel, metricName)
	}
	return nil
}

func (m *mockMetricsRepository) SendTokenMetricWithTimezone(totalTokens int, hostLabel string, metricName string, timezone repository.TimezoneInfo) error {
	// For testing, just call the regular SendTokenMetric
	return m.SendTokenMetric(totalTokens, hostLabel, metricName)
}

func (m *mockMetricsRepository) Close() error {
	return nil
}

func (m *mockMetricsRepository) GetSendCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sendCount
}

type mockCursorService struct {
	getCurrentUsageFunc         func() (*entity.CursorUsage, error)
	getAggregatedTokenUsageFunc func() (int64, error)
	callCount                   int
	mu                          sync.Mutex
}

func (m *mockCursorService) GetCurrentUsage() (*entity.CursorUsage, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.getCurrentUsageFunc != nil {
		return m.getCurrentUsageFunc()
	}

	// Return default cursor usage
	return entity.NewCursorUsage(
		entity.PremiumRequestsInfo{
			Current:      350,
			Limit:        500,
			StartOfMonth: "2023-01-01",
		},
		entity.UsageBasedPricingInfo{},
		nil,
	), nil
}

func (m *mockCursorService) GetUsageLimit() (*repository.UsageLimitInfo, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCursorService) IsUsageBasedPricingEnabled() (bool, error) {
	return false, errors.New("not implemented")
}

func (m *mockCursorService) GetAggregatedTokenUsage() (int64, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.getAggregatedTokenUsageFunc != nil {
		return m.getAggregatedTokenUsageFunc()
	}
	return 0, errors.New("not implemented")
}

func (m *mockCursorService) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// Tests

func TestNewMetricsServiceImpl(t *testing.T) {
	ccService := &mockCcService{}
	metricsRepo := &mockMetricsRepository{}
	config := &config.PrometheusConfig{
		IntervalSec: 600,
		HostLabel:   "test-host",
	}

	timezoneService := &MockTimezoneService{Location: time.UTC}
	service := NewMetricsServiceImpl(ccService, nil, nil, nil, metricsRepo, config, &mockLogger{}, timezoneService)
	if service == nil {
		t.Error("NewMetricsServiceImpl returned nil")
	}
}

func TestMetricsServiceImpl_StartPeriodicMetrics(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.PrometheusConfig
		wantErr bool
	}{
		{
			name: "successful start",
			config: &config.PrometheusConfig{
				IntervalSec: 1, // 1 second for testing
				HostLabel:   "test-host",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ccService := &mockCcService{}
			metricsRepo := &mockMetricsRepository{}
			timezoneService := &MockTimezoneService{Location: time.UTC}
			service := NewMetricsServiceImpl(ccService, nil, nil, nil, metricsRepo, tt.config, &mockLogger{}, timezoneService)

			err := service.StartPeriodicMetrics()
			if (err != nil) != tt.wantErr {
				t.Errorf("StartPeriodicMetrics() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Give time for initial metric to be sent
				time.Sleep(100 * time.Millisecond)

				// Check that metrics were sent
				if metricsRepo.GetSendCount() == 0 {
					t.Error("No metrics were sent")
				}

				// Stop the service
				_ = service.StopPeriodicMetrics()
			}
		})
	}
}

func TestMetricsServiceImpl_StopPeriodicMetrics(t *testing.T) {
	ccService := &mockCcService{}
	metricsRepo := &mockMetricsRepository{}
	config := &config.PrometheusConfig{
		IntervalSec: 1,
		HostLabel:   "test-host",
	}

	timezoneService := &MockTimezoneService{Location: time.UTC}
	service := NewMetricsServiceImpl(ccService, nil, nil, nil, metricsRepo, config, &mockLogger{}, timezoneService)

	// Start the service
	err := service.StartPeriodicMetrics()
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}

	// Give time for metrics to be sent
	time.Sleep(100 * time.Millisecond)

	initialCount := metricsRepo.GetSendCount()

	// Stop the service
	err = service.StopPeriodicMetrics()
	if err != nil {
		t.Errorf("StopPeriodicMetrics() returned error: %v", err)
	}

	// Final metrics should be sent
	finalCount := metricsRepo.GetSendCount()
	if finalCount <= initialCount {
		t.Error("Final metrics were not sent on stop")
	}

	// Try stopping again - should not error
	err = service.StopPeriodicMetrics()
	if err != nil {
		t.Errorf("StopPeriodicMetrics() on stopped service returned error: %v", err)
	}
}

func TestMetricsServiceImpl_SendCurrentMetrics(t *testing.T) {
	tests := []struct {
		name            string
		ccServiceFunc   func() (int, error)
		metricsRepoFunc func(int, string, string) error
		wantErr         bool
	}{
		{
			name: "successful send",
			ccServiceFunc: func() (int, error) {
				return 12345, nil
			},
			metricsRepoFunc: func(tokens int, host string, metricName string) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "cc service error",
			ccServiceFunc: func() (int, error) {
				return 0, errors.New("cc error")
			},
			metricsRepoFunc: func(tokens int, host string, metricName string) error {
				return nil
			},
			wantErr: true,
		},
		{
			name: "metrics repo error",
			ccServiceFunc: func() (int, error) {
				return 12345, nil
			},
			metricsRepoFunc: func(tokens int, host string, metricName string) error {
				return errors.New("send error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ccService := &mockCcService{
				calculateTodayTokensFunc: tt.ccServiceFunc,
			}
			metricsRepo := &mockMetricsRepository{
				sendTokenMetricFunc: tt.metricsRepoFunc,
			}
			config := &config.PrometheusConfig{
				IntervalSec: 600,
				HostLabel:   "test-host",
			}

			timezoneService := &MockTimezoneService{Location: time.UTC}
			service := NewMetricsServiceImpl(ccService, nil, nil, nil, metricsRepo, config, &mockLogger{}, timezoneService)

			err := service.SendCurrentMetrics()
			if (err != nil) != tt.wantErr {
				t.Errorf("SendCurrentMetrics() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMetricsServiceImpl_PeriodicExecution(t *testing.T) {
	ccService := &mockCcService{}
	metricsRepo := &mockMetricsRepository{}
	config := &config.PrometheusConfig{
		IntervalSec: 1, // 1 second interval for testing
		HostLabel:   "test-host",
	}

	timezoneService := &MockTimezoneService{Location: time.UTC}
	service := NewMetricsServiceImpl(ccService, nil, nil, nil, metricsRepo, config, &mockLogger{}, timezoneService)

	// Start periodic metrics
	err := service.StartPeriodicMetrics()
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}

	// Wait for multiple intervals
	time.Sleep(3500 * time.Millisecond)

	// Stop the service
	_ = service.StopPeriodicMetrics()

	// Check that metrics were sent multiple times
	sendCount := metricsRepo.GetSendCount()
	if sendCount < 3 {
		t.Errorf("Expected at least 3 metrics sends, got %d", sendCount)
	}
}

func TestMetricsServiceImpl_ErrorHandling(t *testing.T) {
	// Test that errors don't stop periodic execution
	errorCount := 0
	ccService := &mockCcService{
		calculateTodayTokensFunc: func() (int, error) {
			errorCount++
			if errorCount%2 == 0 {
				return 1000, nil
			}
			return 0, errors.New("intermittent error")
		},
	}

	successCount := 0
	metricsRepo := &mockMetricsRepository{
		sendTokenMetricFunc: func(tokens int, host string, metricName string) error {
			successCount++
			return nil
		},
	}

	config := &config.PrometheusConfig{
		IntervalSec: 1,
		HostLabel:   "test-host",
	}

	timezoneService := &MockTimezoneService{Location: time.UTC}
	service := NewMetricsServiceImpl(ccService, nil, nil, nil, metricsRepo, config, &mockLogger{}, timezoneService)

	// Start periodic metrics
	err := service.StartPeriodicMetrics()
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}

	// Wait for multiple intervals
	time.Sleep(4500 * time.Millisecond)

	// Stop the service
	_ = service.StopPeriodicMetrics()

	// Check that some metrics were sent despite errors
	if successCount < 2 {
		t.Errorf("Expected at least 2 successful sends despite errors, got %d", successCount)
	}
}

func TestMetricsServiceImpl_ConcurrentStartStop(t *testing.T) {
	ccService := &mockCcService{}
	metricsRepo := &mockMetricsRepository{}
	config := &config.PrometheusConfig{
		IntervalSec: 1,
		HostLabel:   "test-host",
	}

	timezoneService := &MockTimezoneService{Location: time.UTC}
	service := NewMetricsServiceImpl(ccService, nil, nil, nil, metricsRepo, config, &mockLogger{}, timezoneService)

	// Try starting multiple times concurrently
	var wg sync.WaitGroup
	errors := make([]error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errors[idx] = service.StartPeriodicMetrics()
		}(i)
	}

	wg.Wait()

	// Only one should succeed
	successCount := 0
	for _, err := range errors {
		if err == nil {
			successCount++
		}
	}

	if successCount != 1 {
		t.Errorf("Expected exactly 1 successful start, got %d", successCount)
	}

	// Stop the service
	_ = service.StopPeriodicMetrics()
}

func TestMetricsServiceImpl_CursorMetrics(t *testing.T) {
	tests := []struct {
		name             string
		cursorService    *mockCursorService
		expectCursorCall bool
	}{
		{
			name: "with cursor service - individual user",
			cursorService: &mockCursorService{
				getCurrentUsageFunc: func() (*entity.CursorUsage, error) {
					return entity.NewCursorUsage(
						entity.PremiumRequestsInfo{
							Current:      200,
							Limit:        500,
							StartOfMonth: "2023-01-01",
						},
						entity.UsageBasedPricingInfo{},
						nil, // No team info - individual user
					), nil
				},
				getAggregatedTokenUsageFunc: func() (int64, error) {
					return 200, nil
				},
			},
			expectCursorCall: true,
		},
		{
			name: "with cursor service - team member",
			cursorService: &mockCursorService{
				getCurrentUsageFunc: func() (*entity.CursorUsage, error) {
					return entity.NewCursorUsage(
						entity.PremiumRequestsInfo{
							Current:      450,
							Limit:        1000,
							StartOfMonth: "2023-01-01",
						},
						entity.UsageBasedPricingInfo{},
						&entity.TeamInfo{
							TeamID:   123,
							UserID:   456,
							TeamName: "Engineering",
							Role:     "member",
						},
					), nil
				},
				getAggregatedTokenUsageFunc: func() (int64, error) {
					return 450, nil
				},
			},
			expectCursorCall: true,
		},
		{
			name: "with cursor service error",
			cursorService: &mockCursorService{
				getCurrentUsageFunc: func() (*entity.CursorUsage, error) {
					return nil, errors.New("cursor API error")
				},
				getAggregatedTokenUsageFunc: func() (int64, error) {
					return 0, errors.New("cursor API error")
				},
			},
			expectCursorCall: true,
		},
		{
			name:             "without cursor service",
			cursorService:    nil,
			expectCursorCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ccService := &mockCcService{}
			metricsRepo := &mockMetricsRepository{}
			config := &config.PrometheusConfig{
				IntervalSec: 600,
				HostLabel:   "test-host",
			}

			var service usecase.MetricsService
			timezoneService := &MockTimezoneService{Location: time.UTC}
			if tt.cursorService != nil {
				service = NewMetricsServiceImpl(ccService, tt.cursorService, nil, nil, metricsRepo, config, &mockLogger{}, timezoneService)
			} else {
				service = NewMetricsServiceImpl(ccService, nil, nil, nil, metricsRepo, config, &mockLogger{}, timezoneService)
			}

			// Send metrics
			err := service.SendCurrentMetrics()
			if err != nil {
				t.Errorf("SendCurrentMetrics() returned unexpected error: %v", err)
			}

			// Check if cursor service was called
			if tt.cursorService != nil {
				callCount := tt.cursorService.GetCallCount()
				if tt.expectCursorCall && callCount == 0 {
					t.Error("Expected cursor service to be called, but it wasn't")
				} else if !tt.expectCursorCall && callCount > 0 {
					t.Error("Expected cursor service not to be called, but it was")
				}
			}

			// Note: Cursor metrics are now sent through the same SendTokenMetric method
			// We can verify this by checking the total send count

			// Token metric should always be sent
			tokenMetricCount := metricsRepo.GetSendCount()
			if tokenMetricCount == 0 {
				t.Error("Expected token metric to be sent, but it wasn't")
			}
		})
	}
}

func TestMetricsServiceImpl_CursorMetrics_Values(t *testing.T) {
	var capturedTokens int
	var capturedHostLabel string
	var capturedMetricName string

	metricsRepo := &mockMetricsRepository{
		sendTokenMetricFunc: func(totalTokens int, hostLabel string, metricName string) error {
			capturedTokens = totalTokens
			capturedHostLabel = hostLabel
			capturedMetricName = metricName
			return nil
		},
	}

	tests := []struct {
		name               string
		expectedTokens     int
		expectedMetricName string
	}{
		{
			name:               "cursor token metric",
			expectedTokens:     150,
			expectedMetricName: "tosage_cursor_token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursorService := &mockCursorService{
				getAggregatedTokenUsageFunc: func() (int64, error) {
					return int64(tt.expectedTokens), nil
				},
			}

			ccService := &mockCcService{}
			config := &config.PrometheusConfig{
				IntervalSec: 600,
				HostLabel:   "test-host",
			}

			timezoneService := &MockTimezoneService{Location: time.UTC}
			service := NewMetricsServiceImpl(ccService, cursorService, nil, nil, metricsRepo, config, &mockLogger{}, timezoneService)

			// Send metrics
			_ = service.SendCurrentMetrics()

			// Verify captured values
			if capturedTokens != tt.expectedTokens {
				t.Errorf("Expected tokens %d, got %d", tt.expectedTokens, capturedTokens)
			}

			if capturedHostLabel != config.HostLabel {
				t.Errorf("Expected host label %s, got %s", config.HostLabel, capturedHostLabel)
			}

			if capturedMetricName != tt.expectedMetricName {
				t.Errorf("Expected metric name %s, got %s", tt.expectedMetricName, capturedMetricName)
			}
		})
	}
}

func TestMetricsServiceImpl_GracefulDegradation(t *testing.T) {
	tests := []struct {
		name             string
		ccService        usecase.CcService
		cursorService    usecase.CursorService
		expectCcCall     bool
		expectCursorCall bool
	}{
		{
			name:             "both services nil",
			ccService:        nil,
			cursorService:    nil,
			expectCcCall:     false,
			expectCursorCall: false,
		},
		{
			name:      "only ccService nil",
			ccService: nil,
			cursorService: &mockCursorService{
				getAggregatedTokenUsageFunc: func() (int64, error) {
					return 5000, nil
				},
			},
			expectCcCall:     false,
			expectCursorCall: true,
		},
		{
			name: "only cursorService nil",
			ccService: &mockCcService{
				calculateTodayTokensFunc: func() (int, error) {
					return 1000, nil
				},
			},
			cursorService:    nil,
			expectCcCall:     true,
			expectCursorCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			metricsRepo := &mockMetricsRepository{
				sendTokenMetricFunc: func(tokens int, hostLabel, metricName string) error {
					callCount++
					return nil
				},
			}
			config := &config.PrometheusConfig{
				IntervalSec: 600,
				HostLabel:   "test-host",
			}

			timezoneService := &MockTimezoneService{Location: time.UTC}
			service := NewMetricsServiceImpl(tt.ccService, tt.cursorService, nil, nil, metricsRepo, config, &mockLogger{}, timezoneService)

			// Send metrics
			err := service.SendCurrentMetrics()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check expected number of metric sends
			expectedCalls := 0
			if tt.expectCcCall {
				expectedCalls++
			}
			if tt.expectCursorCall {
				expectedCalls++
			}

			if callCount != expectedCalls {
				t.Errorf("Expected %d metric send calls, got %d", expectedCalls, callCount)
			}
		})
	}
}
