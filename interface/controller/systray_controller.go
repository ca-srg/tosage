//go:build darwin
// +build darwin

package controller

import (
	"fmt"
	"time"

	"github.com/ca-srg/tosage/assets"
	usecase "github.com/ca-srg/tosage/usecase/interface"
	"github.com/getlantern/systray"
)

// SystrayController manages the system tray menu and interactions
type SystrayController struct {
	ccService      usecase.CcService
	statusService  usecase.StatusService
	metricsService usecase.MetricsService
	configService  usecase.ConfigService

	// Menu items
	sendNowItem      *systray.MenuItem
	settingsItem     *systray.MenuItem
	startAtLoginItem *systray.MenuItem
	quitItem         *systray.MenuItem

	// Channels for menu actions
	sendNowChan      chan struct{}
	settingsChan     chan struct{}
	startAtLoginChan chan struct{}
	quitChan         chan struct{}

	// Login item manager
	loginItemManager *LoginItemManager
}

// NewSystrayController creates a new system tray controller
func NewSystrayController(
	ccService usecase.CcService,
	statusService usecase.StatusService,
	metricsService usecase.MetricsService,
	configService usecase.ConfigService,
) *SystrayController {
	loginItemManager, _ := NewLoginItemManager()

	return &SystrayController{
		ccService:        ccService,
		statusService:    statusService,
		metricsService:   metricsService,
		configService:    configService,
		sendNowChan:      make(chan struct{}),
		settingsChan:     make(chan struct{}),
		startAtLoginChan: make(chan struct{}),
		quitChan:         make(chan struct{}),
		loginItemManager: loginItemManager,
	}
}

// OnReady is called when the system tray is ready
func (s *SystrayController) OnReady() {
	// Set up the system tray icon and tooltip
	systray.SetIcon(assets.IconData)
	systray.SetTooltip("tosage - token cc tracker")

	// Create menu items
	s.sendNowItem = systray.AddMenuItem("Send Metrics Now", "Send metrics to Prometheus immediately")
	systray.AddSeparator()
	s.settingsItem = systray.AddMenuItem("Settings...", "Open settings dialog")
	s.startAtLoginItem = systray.AddMenuItemCheckbox("Start at Login", "Start tosage when you log in", false)

	// Set initial state for start at login
	if s.loginItemManager != nil {
		if isLoginItem, _ := s.loginItemManager.IsLoginItem(); isLoginItem {
			s.startAtLoginItem.Check()
		}
	}

	systray.AddSeparator()
	s.quitItem = systray.AddMenuItem("Quit", "Quit the application")

	// Start handling menu clicks
	go s.handleMenuClicks()
}

// OnExit is called when the system tray is exiting
func (s *SystrayController) OnExit() {
	// Clean up resources
	close(s.sendNowChan)
	close(s.settingsChan)
	close(s.startAtLoginChan)
	close(s.quitChan)
}

// handleMenuClicks handles clicks on menu items
func (s *SystrayController) handleMenuClicks() {
	for {
		select {
		case <-s.sendNowItem.ClickedCh:
			s.sendNowChan <- struct{}{}

		case <-s.settingsItem.ClickedCh:
			s.settingsChan <- struct{}{}

		case <-s.startAtLoginItem.ClickedCh:
			s.handleStartAtLoginToggle()

		case <-s.quitItem.ClickedCh:
			s.quitChan <- struct{}{}
			return
		}
	}
}

// GetSendNowChannel returns the channel that signals when "Send Metrics Now" is clicked
func (s *SystrayController) GetSendNowChannel() <-chan struct{} {
	return s.sendNowChan
}

// GetSettingsChannel returns the channel that signals when "Settings..." is clicked
func (s *SystrayController) GetSettingsChannel() <-chan struct{} {
	return s.settingsChan
}

// GetQuitChannel returns the channel that signals when "Quit" is clicked
func (s *SystrayController) GetQuitChannel() <-chan struct{} {
	return s.quitChan
}

// UpdateStatus updates the status display in the menu
func (s *SystrayController) UpdateStatus(status *usecase.StatusInfo) {
	if status == nil {
		return
	}

	// Update tooltip with current status
	tooltip := "tosage\n"
	if status.IsRunning {
		tooltip += "Status: Running\n"
		if status.DaemonStartedAt != nil {
			tooltip += fmt.Sprintf("Started: %s\n", status.DaemonStartedAt.Format("15:04:05"))
		}
		tooltip += fmt.Sprintf("Today's tokens: %d\n", status.TodayTokenCount)
		if status.LastMetricsSentAt != nil {
			tooltip += fmt.Sprintf("Last sent: %s", status.LastMetricsSentAt.Format("15:04:05"))
		}
	} else {
		tooltip += "Status: Stopped"
	}

	systray.SetTooltip(tooltip)
}

// ShowNotification shows a notification to the user
func (s *SystrayController) ShowNotification(title, message string) {
	// Note: systray doesn't directly support notifications
	// This would need to be implemented using platform-specific APIs
	// For now, we'll just update the tooltip
	systray.SetTooltip(fmt.Sprintf("%s: %s", title, message))

	// Reset tooltip after 3 seconds
	go func() {
		time.Sleep(3 * time.Second)
		status, _ := s.statusService.GetStatus()
		s.UpdateStatus(status)
	}()
}

// handleStartAtLoginToggle handles toggling the start at login setting
func (s *SystrayController) handleStartAtLoginToggle() {
	if s.loginItemManager == nil {
		s.ShowNotification("Error", "Login item management not available")
		return
	}

	// Check current state
	isLoginItem, err := s.loginItemManager.IsLoginItem()
	if err != nil {
		s.ShowNotification("Error", fmt.Sprintf("Failed to check login item status: %v", err))
		return
	}

	// Toggle the state
	newState := !isLoginItem
	err = s.loginItemManager.SetLoginItem(newState)
	if err != nil {
		s.ShowNotification("Error", fmt.Sprintf("Failed to update login item: %v", err))
		// Revert the checkbox state
		if isLoginItem {
			s.startAtLoginItem.Check()
		} else {
			s.startAtLoginItem.Uncheck()
		}
		return
	}

	// Update checkbox state
	if newState {
		s.startAtLoginItem.Check()
		s.ShowNotification("Success", "tosage will start at login")
	} else {
		s.startAtLoginItem.Uncheck()
		s.ShowNotification("Success", "tosage will not start at login")
	}
}
