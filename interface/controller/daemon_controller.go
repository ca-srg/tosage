//go:build darwin
// +build darwin

package controller

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/ca-srg/tosage/domain"
	"github.com/ca-srg/tosage/infrastructure/config"
	usecase "github.com/ca-srg/tosage/usecase/interface"
	"github.com/getlantern/systray"
)

// DaemonController manages the daemon lifecycle
type DaemonController struct {
	config         *config.AppConfig
	configService  usecase.ConfigService
	ccService      usecase.CcService
	statusService  usecase.StatusService
	metricsService usecase.MetricsService
	systrayCtrl    *SystrayController

	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	logger          domain.Logger
	pidFile         string
	metricsTicker   *time.Ticker
	metricsTickerMu sync.Mutex
	isPaused        bool
}

// NewDaemonController creates a new daemon controller
func NewDaemonController(
	cfg *config.AppConfig,
	configService usecase.ConfigService,
	ccService usecase.CcService,
	statusService usecase.StatusService,
	metricsService usecase.MetricsService,
	systrayCtrl *SystrayController,
	logger domain.Logger,
) *DaemonController {
	return &DaemonController{
		config:         cfg,
		configService:  configService,
		ccService:      ccService,
		statusService:  statusService,
		metricsService: metricsService,
		systrayCtrl:    systrayCtrl,
		logger:         logger,
	}
}

// Start starts the daemon
func (d *DaemonController) Start() error {
	return d.startInternal()
}

// startInternal starts the daemon without running the system tray
func (d *DaemonController) startInternal() error {
	d.logger.Info(d.ctx, "Starting tosage daemon...")

	// Create context for cancellation
	d.ctx, d.cancel = context.WithCancel(context.Background())

	// Write PID file
	if err := d.writePIDFile(); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// Update status service
	if err := d.statusService.SetDaemonStarted(time.Now()); err != nil {
		return fmt.Errorf("failed to update daemon status: %w", err)
	}

	// Start the daemon run loop in a goroutine
	d.wg.Add(1)
	go d.run()

	// Register for system events
	if err := RegisterSystemEventHandler(d); err != nil {
		d.logger.Warn(d.ctx, "Failed to register for system events", domain.NewField("error", err.Error()))
	}

	// Setup signal handlers
	d.setupSignalHandlers()

	d.logger.Info(d.ctx, "Daemon started successfully")
	return nil
}

// Stop stops the daemon gracefully
func (d *DaemonController) Stop() error {
	d.logger.Info(d.ctx, "Stopping tosage daemon...")

	// Cancel the context to stop all goroutines
	if d.cancel != nil {
		d.cancel()
	}

	// Wait for all goroutines to finish
	d.wg.Wait()

	// Update status service
	if err := d.statusService.SetDaemonStopped(); err != nil {
		d.logger.Error(d.ctx, "Failed to update daemon status", domain.NewField("error", err.Error()))
	}

	// Remove PID file
	if err := d.removePIDFile(); err != nil {
		d.logger.Error(d.ctx, "Failed to remove PID file", domain.NewField("error", err.Error()))
	}

	// Unregister system event handler
	UnregisterSystemEventHandler(d)

	d.logger.Info(d.ctx, "Daemon stopped successfully")
	return nil
}

// Run starts the daemon main loop
func (d *DaemonController) Run() {
	// Apply Dock visibility setting before starting the system tray
	if d.config.Daemon != nil && d.config.Daemon.HideFromDock {
		HideFromDock()
		d.logger.Info(d.ctx, "Application hidden from Dock")
	}

	// This method blocks until the daemon is stopped
	if err := d.startInternal(); err != nil {
		d.logger.Error(d.ctx, "Failed to start daemon", domain.NewField("error", err.Error()))
		return
	}

	// Run system tray on the main thread (this blocks until quit is called)
	// This is required for macOS
	systray.Run(func() {
		d.systrayCtrl.OnReady()
	}, func() {
		d.systrayCtrl.OnExit()
		// Perform cleanup after systray exits
		d.wg.Wait()
		_ = d.statusService.SetDaemonStopped()
		_ = d.removePIDFile()
		UnregisterSystemEventHandler(d)
		d.logger.Info(d.ctx, "Daemon stopped successfully")
	})
}

// run is the main daemon loop
func (d *DaemonController) run() {
	defer d.wg.Done()

	// Start periodic metrics if configured
	if d.config.Prometheus != nil && d.metricsService != nil {
		interval := time.Duration(d.config.Prometheus.IntervalSec) * time.Second
		d.metricsTickerMu.Lock()
		d.metricsTicker = time.NewTicker(interval)
		d.metricsTickerMu.Unlock()
		defer d.metricsTicker.Stop()

		// Send initial metrics
		d.sendMetrics()
		d.updateNextSendTime(interval)
	}

	// Main loop
	for {
		select {
		case <-d.ctx.Done():
			return

		case <-d.metricsTicker.C:
			if !d.isPaused {
				d.sendMetrics()
				d.updateNextSendTime(time.Duration(d.config.Prometheus.IntervalSec) * time.Second)
			}

		case <-d.systrayCtrl.GetSendNowChannel():
			d.logger.Info(d.ctx, "Manual metrics send requested")
			d.sendMetrics()
			d.systrayCtrl.ShowNotification("Metrics Sent", "Token cc metrics sent successfully")

		case <-d.systrayCtrl.GetSettingsChannel():
			d.openSettings()

		case <-d.systrayCtrl.GetQuitChannel():
			d.logger.Info(d.ctx, "Quit button clicked")
			d.cancel()
			// Quit system tray to unblock the main thread
			go func() {
				// Give time for goroutines to finish
				time.Sleep(100 * time.Millisecond)
				systray.Quit()
			}()
			return
		}
	}
}

// sendMetrics sends current metrics
func (d *DaemonController) sendMetrics() {
	d.logger.Debug(d.ctx, "Sending metrics...")

	// Get current cc
	tokens, err := d.ccService.CalculateTodayTokens()
	if err != nil {
		d.logger.Error(d.ctx, "Failed to get cc", domain.NewField("error", err.Error()))
		_ = d.statusService.RecordError(err)
		return
	}

	// Update token count in status
	if err := d.statusService.UpdateTodayTokenCount(int64(tokens)); err != nil {
		d.logger.Error(d.ctx, "Failed to update token count", domain.NewField("error", err.Error()))
	}

	// Send metrics
	if err := d.metricsService.SendCurrentMetrics(); err != nil {
		d.logger.Error(d.ctx, "Failed to send metrics", domain.NewField("error", err.Error()))
		_ = d.statusService.RecordError(err)
		return
	}

	// Update status
	if err := d.statusService.UpdateLastMetricsSent(time.Now()); err != nil {
		d.logger.Error(d.ctx, "Failed to update last sent time", domain.NewField("error", err.Error()))
	}
	_ = d.statusService.ClearError()

	// Update system tray
	status, _ := d.statusService.GetStatus()
	d.systrayCtrl.UpdateStatus(status)
}

// updateNextSendTime updates the next metrics send time
func (d *DaemonController) updateNextSendTime(interval time.Duration) {
	nextTime := time.Now().Add(interval)
	if err := d.statusService.UpdateNextMetricsSend(nextTime); err != nil {
		d.logger.Error(d.ctx, "Failed to update next send time", domain.NewField("error", err.Error()))
	}
}

// setupSignalHandlers sets up signal handlers for graceful shutdown
func (d *DaemonController) setupSignalHandlers() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		d.logger.Info(d.ctx, "Received signal", domain.NewField("signal", sig.String()))
		d.cancel()
		// Wait for run goroutine to finish
		d.wg.Wait()
		// Clean up
		_ = d.statusService.SetDaemonStopped()
		_ = d.removePIDFile()
		UnregisterSystemEventHandler(d)
		// Quit system tray to unblock the main thread
		systray.Quit()
	}()
}

// writePIDFile writes the process ID to a file
func (d *DaemonController) writePIDFile() error {
	if d.config.Daemon == nil || d.config.Daemon.PidFile == "" {
		return nil
	}

	pid := os.Getpid()
	pidStr := strconv.Itoa(pid)

	if err := os.WriteFile(d.config.Daemon.PidFile, []byte(pidStr), 0644); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	d.pidFile = d.config.Daemon.PidFile
	return nil
}

// removePIDFile removes the PID file
func (d *DaemonController) removePIDFile() error {
	if d.pidFile == "" {
		return nil
	}

	if err := os.Remove(d.pidFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}

	return nil
}

// OnSystemSleep handles system sleep events
func (d *DaemonController) OnSystemSleep() {
	d.logger.Info(d.ctx, "System going to sleep, pausing metrics collection")

	d.metricsTickerMu.Lock()
	d.isPaused = true
	d.metricsTickerMu.Unlock()

	// Update status to indicate sleep
	_ = d.statusService.RecordError(fmt.Errorf("system sleeping"))
}

// OnSystemWake handles system wake events
func (d *DaemonController) OnSystemWake() {
	d.logger.Info(d.ctx, "System waking up, resuming metrics collection")

	d.metricsTickerMu.Lock()
	d.isPaused = false
	d.metricsTickerMu.Unlock()

	// Clear sleep error
	_ = d.statusService.ClearError()

	// Send catch-up metrics after a brief delay to allow network to stabilize
	go func() {
		time.Sleep(5 * time.Second)
		d.logger.Info(d.ctx, "Sending catch-up metrics after wake")
		d.sendMetrics()
		d.updateNextSendTime(time.Duration(d.config.Prometheus.IntervalSec) * time.Second)
	}()
}

// openSettings opens the settings dialog
func (d *DaemonController) openSettings() {
	d.logger.Info(d.ctx, "Opening settings in external editor")

	// 設定ファイルのパスを取得
	configPath := d.configService.GetConfigPath()

	// 設定ファイルが存在しない場合は作成
	err := d.configService.SaveConfig()
	if err != nil {
		d.logger.Error(d.ctx, "Failed to ensure config file", domain.NewField("error", err.Error()))
		d.systrayCtrl.ShowNotification("Error", "Failed to ensure configuration file")
		return
	}

	// 外部エディタで設定ファイルを開く
	if err := d.openInExternalEditor(configPath); err != nil {
		d.logger.Error(d.ctx, "Failed to open settings", domain.NewField("error", err.Error()))
		d.systrayCtrl.ShowNotification("Error", fmt.Sprintf("Failed to open settings: %v", err))
		return
	}

	// 成功通知
	d.systrayCtrl.ShowNotification("Settings", "Configuration opened in editor. Restart required after changes.")
	d.logger.Info(d.ctx, "Settings opened successfully", domain.NewField("config_path", configPath))
}

// openInExternalEditor は指定されたファイルを外部エディタで開く
func (d *DaemonController) openInExternalEditor(filePath string) error {
	var cmd *exec.Cmd

	// プラットフォームに応じて適切なコマンドを選択
	switch runtime.GOOS {
	case "darwin":
		// macOSでは 'open' コマンドを使用
		// -t フラグでテキストエディタを指定
		cmd = exec.Command("open", "-t", filePath)
	case "linux":
		// Linuxでは xdg-open を使用
		cmd = exec.Command("xdg-open", filePath)
	case "windows":
		// Windowsでは notepad を使用
		cmd = exec.Command("notepad", filePath)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// エディタを起動（非同期）
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}

	// プロセスの終了を待たない（エディタは独立して動作）
	go func() {
		// エディタプロセスをクリーンアップ
		_ = cmd.Wait()
	}()

	return nil
}
