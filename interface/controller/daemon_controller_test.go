//go:build darwin
// +build darwin

package controller

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ca-srg/tosage/domain"
	"github.com/ca-srg/tosage/infrastructure/config"
	"github.com/ca-srg/tosage/usecase/impl"
	usecase "github.com/ca-srg/tosage/usecase/interface"
)

// mockLogger is a test logger that does nothing
type mockLogger struct{}

func (m *mockLogger) Debug(ctx context.Context, msg string, fields ...domain.Field) {}
func (m *mockLogger) Info(ctx context.Context, msg string, fields ...domain.Field)  {}
func (m *mockLogger) Warn(ctx context.Context, msg string, fields ...domain.Field)  {}
func (m *mockLogger) Error(ctx context.Context, msg string, fields ...domain.Field) {}
func (m *mockLogger) WithFields(fields ...domain.Field) domain.Logger               { return m }

// MockSystrayController is a mock implementation of SystrayController for testing
type MockSystrayController struct {
	sendNowChan   chan struct{}
	statusChan    chan struct{}
	settingsChan  chan struct{}
	quitChan      chan struct{}
	onReadyCalled bool
	onExitCalled  bool
}

func NewMockSystrayController() *MockSystrayController {
	return &MockSystrayController{
		sendNowChan:  make(chan struct{}, 1),
		statusChan:   make(chan struct{}, 1),
		settingsChan: make(chan struct{}, 1),
		quitChan:     make(chan struct{}, 1),
	}
}

func (m *MockSystrayController) OnReady() {
	m.onReadyCalled = true
}

func (m *MockSystrayController) OnExit() {
	m.onExitCalled = true
}

func (m *MockSystrayController) GetSendNowChannel() <-chan struct{} {
	return m.sendNowChan
}

func (m *MockSystrayController) GetStatusChannel() <-chan struct{} {
	return m.statusChan
}

func (m *MockSystrayController) GetSettingsChannel() <-chan struct{} {
	return m.settingsChan
}

func (m *MockSystrayController) GetQuitChannel() <-chan struct{} {
	return m.quitChan
}

func (m *MockSystrayController) ShowNotification(title, message string) {
	// Mock implementation - do nothing
}

func (m *MockSystrayController) UpdateStatus(status *usecase.StatusInfo) {
	// Mock implementation - do nothing
}

func TestDaemonController_StartStop(t *testing.T) {
	// Create test configuration
	cfg := &config.AppConfig{
		Daemon: &config.DaemonConfig{
			Enabled: true,
			LogPath: "/tmp/test-tosage.log",
			PidFile: "/tmp/test-tosage.pid",
		},
		Prometheus: &config.PrometheusConfig{
			IntervalSec: 5, // Short interval for testing
			TimeoutSec:  2,
		},
	}

	// Create mock services
	ccService := &MockCcService{}
	statusService := impl.NewStatusService()
	metricsService := &MockMetricsService{}
	configService := &MockConfigService{}
	systrayCtrl := NewSystrayController(ccService, statusService, metricsService, configService, nil)

	// Create daemon controller
	daemon := NewDaemonController(cfg, configService, ccService, statusService, metricsService, systrayCtrl, &mockLogger{})

	// Test Start
	err := daemon.Start()
	if err != nil {
		t.Fatalf("Failed to start daemon: %v", err)
	}

	// Give daemon time to initialize
	time.Sleep(100 * time.Millisecond)

	// Check status
	status, err := statusService.GetStatus()
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}
	if !status.IsRunning {
		t.Error("Expected daemon to be running")
	}

	// Test Stop
	err = daemon.Stop()
	if err != nil {
		t.Fatalf("Failed to stop daemon: %v", err)
	}

	// Check status after stop
	status, err = statusService.GetStatus()
	if err != nil {
		t.Fatalf("Failed to get status after stop: %v", err)
	}
	if status.IsRunning {
		t.Error("Expected daemon to be stopped")
	}
}

func TestDaemonController_MetricsSending(t *testing.T) {
	// Create test configuration with short interval
	cfg := &config.AppConfig{
		Daemon: &config.DaemonConfig{
			Enabled: true,
		},
		Prometheus: &config.PrometheusConfig{
			IntervalSec: 1, // 1 second for quick testing
			TimeoutSec:  1,
		},
	}

	// Create mock services
	ccService := &MockCcService{tokenCount: 12345}
	statusService := impl.NewStatusService()
	metricsService := &MockMetricsService{}
	configService := &MockConfigService{}
	systrayCtrl := NewSystrayController(ccService, statusService, metricsService, configService, nil)

	// Create daemon controller
	daemon := NewDaemonController(cfg, configService, ccService, statusService, metricsService, systrayCtrl, &mockLogger{})

	// Start daemon
	err := daemon.Start()
	if err != nil {
		t.Fatalf("Failed to start daemon: %v", err)
	}
	defer func() {
		_ = daemon.Stop()
	}()

	// Wait for automatic metrics send
	time.Sleep(1500 * time.Millisecond)

	// Check that metrics were sent
	if metricsService.GetSendCount() < 1 {
		t.Error("Expected metrics to be sent at least once")
	}

	// Check status was updated
	status, _ := statusService.GetStatus()
	if status.TodayTokenCount != 12345 {
		t.Errorf("Expected token count to be 12345, got %d", status.TodayTokenCount)
	}
	if status.LastMetricsSentAt == nil {
		t.Error("Expected LastMetricsSentAt to be set")
	}
}

func TestDaemonController_ManualMetricsSend(t *testing.T) {
	// Create test configuration
	cfg := &config.AppConfig{
		Daemon: &config.DaemonConfig{
			Enabled: true,
		},
		Prometheus: &config.PrometheusConfig{
			IntervalSec: 600, // Long interval so automatic send doesn't interfere
			TimeoutSec:  30,
		},
	}

	// Create mock services
	ccService := &MockCcService{tokenCount: 54321}
	statusService := impl.NewStatusService()
	metricsService := &MockMetricsService{}
	configService := &MockConfigService{}
	systrayCtrl := NewSystrayController(ccService, statusService, metricsService, configService, nil)

	// Create daemon controller
	daemon := NewDaemonController(cfg, configService, ccService, statusService, metricsService, systrayCtrl, &mockLogger{})

	// Start daemon
	err := daemon.Start()
	if err != nil {
		t.Fatalf("Failed to start daemon: %v", err)
	}
	defer func() {
		_ = daemon.Stop()
	}()

	// Give daemon time to initialize
	time.Sleep(100 * time.Millisecond)

	// Trigger manual metrics send
	systrayCtrl.sendNowChan <- struct{}{}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Check that metrics were sent
	sendCount := metricsService.GetSendCount()
	if sendCount != 2 { // 1 initial + 1 manual
		t.Errorf("Expected 2 metrics sends, got %d", sendCount)
	}

	// Check status was updated
	status, _ := statusService.GetStatus()
	if status.TodayTokenCount != 54321 {
		t.Errorf("Expected token count to be 54321, got %d", status.TodayTokenCount)
	}
}

func TestDaemonController_SystemEvents(t *testing.T) {
	// Skip if not on Darwin
	if !isDarwin() {
		t.Skip("System events only supported on Darwin")
	}

	// Create test configuration
	cfg := &config.AppConfig{
		Daemon: &config.DaemonConfig{
			Enabled: true,
		},
		Prometheus: &config.PrometheusConfig{
			IntervalSec: 10,
			TimeoutSec:  5,
		},
	}

	// Create services
	ccService := &MockCcService{}
	statusService := impl.NewStatusService()
	metricsService := &MockMetricsService{}
	configService := &MockConfigService{}
	systrayCtrl := NewSystrayController(ccService, statusService, metricsService, configService, nil)

	// Create daemon controller
	daemon := NewDaemonController(cfg, configService, ccService, statusService, metricsService, systrayCtrl, &mockLogger{})

	// Start daemon
	err := daemon.Start()
	if err != nil {
		t.Fatalf("Failed to start daemon: %v", err)
	}
	defer func() {
		_ = daemon.Stop()
	}()

	// Simulate system sleep
	daemon.OnSystemSleep()

	// Check that daemon is paused
	if !daemon.isPaused {
		t.Error("Expected daemon to be paused after system sleep")
	}

	// Check error was recorded
	status, _ := statusService.GetStatus()
	if status.LastError == nil {
		t.Error("Expected error to be recorded for system sleep")
	}

	// Simulate system wake
	daemon.OnSystemWake()

	// Check that daemon is resumed
	if daemon.isPaused {
		t.Error("Expected daemon to be resumed after system wake")
	}

	// Wait for catch-up metrics
	time.Sleep(6 * time.Second)

	// Check that catch-up metrics were sent
	if metricsService.GetSendCount() < 2 {
		t.Error("Expected catch-up metrics to be sent after wake")
	}
}

// Mock implementations for testing

type MockCcService struct {
	tokenCount int
	err        error
}

func (m *MockCcService) CalculateDailyTokens(date time.Time) (int, error) {
	return m.tokenCount, m.err
}

func (m *MockCcService) CalculateTodayTokens() (int, error) {
	return m.tokenCount, m.err
}

func (m *MockCcService) CalculateTokenStats(filter usecase.TokenStatsFilter) (*usecase.TokenStatsResult, error) {
	return nil, nil
}

func (m *MockCcService) CalculateCostBreakdown(filter usecase.CostBreakdownFilter) (*usecase.CostBreakdownResult, error) {
	return nil, nil
}

func (m *MockCcService) CalculateModelBreakdown(filter usecase.ModelBreakdownFilter) (*usecase.ModelBreakdownResult, error) {
	return nil, nil
}

func (m *MockCcService) CalculateDateBreakdown(filter usecase.DateBreakdownFilter) (*usecase.DateBreakdownResult, error) {
	return nil, nil
}

func (m *MockCcService) LoadCcData(filter usecase.CcDataFilter) (*usecase.CcDataResult, error) {
	return nil, nil
}

func (m *MockCcService) GetCcSummary(filter usecase.CcSummaryFilter) (*usecase.CcSummaryResult, error) {
	return nil, nil
}

func (m *MockCcService) EstimateMonthlyCost(daysToAverage int) (*usecase.CostEstimateResult, error) {
	return nil, nil
}

func (m *MockCcService) GetAvailableProjects() ([]string, error) {
	return nil, nil
}

func (m *MockCcService) GetAvailableModels() ([]string, error) {
	return nil, nil
}

func (m *MockCcService) GetDateRange() (start, end time.Time, err error) {
	return time.Now(), time.Now(), nil
}

func (m *MockCcService) CalculateDailyTokensInUserTimezone(date time.Time) (int, error) {
	return m.tokenCount, m.err
}

func (m *MockCcService) CalculateTodayTokensInUserTimezone() (int, error) {
	return m.tokenCount, m.err
}

func (m *MockCcService) GetDateRangeInUserTimezone() (start, end time.Time, err error) {
	return time.Now(), time.Now(), nil
}

type MockMetricsService struct {
	mu        sync.Mutex
	sendCount int
	err       error
}

func (m *MockMetricsService) StartPeriodicMetrics() error {
	return nil
}

func (m *MockMetricsService) StopPeriodicMetrics() error {
	return nil
}

func (m *MockMetricsService) SendCurrentMetrics() error {
	m.mu.Lock()
	m.sendCount++
	m.mu.Unlock()
	return m.err
}

func (m *MockMetricsService) GetSendCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sendCount
}

type MockConfigService struct {
	config *config.AppConfig
}

func (m *MockConfigService) GetConfig() *config.AppConfig {
	if m.config == nil {
		return config.DefaultConfig()
	}
	return m.config
}

func (m *MockConfigService) UpdateConfig(newConfig *config.AppConfig) error {
	m.config = newConfig
	return nil
}

func (m *MockConfigService) GetConfigWithSources() (*config.AppConfig, config.ConfigSourceMap) {
	return m.GetConfig(), make(config.ConfigSourceMap)
}

func (m *MockConfigService) SaveConfig() error {
	return nil
}

func (m *MockConfigService) ReloadConfig() error {
	return nil
}

func (m *MockConfigService) GetConfigPath() string {
	return "/tmp/test-config.json"
}

func (m *MockConfigService) CreateDefaultConfig() error {
	return nil
}

func (m *MockConfigService) ExportConfig() map[string]interface{} {
	return make(map[string]interface{})
}

func (m *MockConfigService) EnsureConfigExists() error {
	return nil
}

func (m *MockConfigService) CreateTemplateConfig() error {
	return nil
}

func (m *MockConfigService) LoadConfigWithFallback() (*config.AppConfig, error) {
	return m.GetConfig(), nil
}

// Helper function to check if running on Darwin
func isDarwin() bool {
	// This would normally check runtime.GOOS == "darwin"
	// For testing purposes, we'll return true on Darwin platforms
	return true
}
