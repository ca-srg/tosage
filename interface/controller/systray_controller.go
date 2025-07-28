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
	ccService       usecase.CcService
	statusService   usecase.StatusService
	metricsService  usecase.MetricsService
	configService   usecase.ConfigService
	bedrockService  usecase.BedrockService
	vertexAIService usecase.VertexAIService

	// Menu items
	sendNowItem      *systray.MenuItem
	settingsItem     *systray.MenuItem
	startAtLoginItem *systray.MenuItem
	bedrockItem      *systray.MenuItem
	vertexAIItem     *systray.MenuItem
	quitItem         *systray.MenuItem

	// Channels for menu actions
	sendNowChan      chan struct{}
	settingsChan     chan struct{}
	startAtLoginChan chan struct{}
	bedrockChan      chan struct{}
	vertexAIChan     chan struct{}
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
	bedrockService usecase.BedrockService,
	vertexAIService usecase.VertexAIService,
) *SystrayController {
	loginItemManager, _ := NewLoginItemManager()

	return &SystrayController{
		ccService:        ccService,
		statusService:    statusService,
		metricsService:   metricsService,
		configService:    configService,
		bedrockService:   bedrockService,
		vertexAIService:  vertexAIService,
		sendNowChan:      make(chan struct{}),
		settingsChan:     make(chan struct{}),
		startAtLoginChan: make(chan struct{}),
		bedrockChan:      make(chan struct{}),
		vertexAIChan:     make(chan struct{}),
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

	// Bedrock tracking checkbox
	bedrockEnabled := s.bedrockService != nil && s.bedrockService.IsEnabled()
	s.bedrockItem = systray.AddMenuItemCheckbox("Include Bedrock Metrics", "Include AWS Bedrock usage in metrics (requires AWS credentials)", bedrockEnabled)

	// Vertex AI tracking checkbox
	vertexAIEnabled := s.vertexAIService != nil && s.vertexAIService.IsEnabled()
	s.vertexAIItem = systray.AddMenuItemCheckbox("Include Vertex AI Metrics", "Include Google Cloud Vertex AI usage in metrics (requires GCP credentials)", vertexAIEnabled)

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
	close(s.bedrockChan)
	close(s.vertexAIChan)
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

		case <-s.bedrockItem.ClickedCh:
			s.handleBedrockToggle()

		case <-s.vertexAIItem.ClickedCh:
			s.handleVertexAIToggle()

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

// GetBedrockChannel returns the channel that signals when Bedrock checkbox is toggled
func (s *SystrayController) GetBedrockChannel() <-chan struct{} {
	return s.bedrockChan
}

// GetVertexAIChannel returns the channel that signals when Vertex AI checkbox is toggled
func (s *SystrayController) GetVertexAIChannel() <-chan struct{} {
	return s.vertexAIChan
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

// handleBedrockToggle handles toggling the Bedrock tracking setting
func (s *SystrayController) handleBedrockToggle() {
	if s.bedrockService == nil {
		s.ShowNotification("Error", "Bedrock service not available")
		return
	}

	// Check current state
	currentState := s.bedrockService.IsEnabled()
	newState := !currentState

	// Update configuration through config service
	// This is a simplified implementation - in practice, you'd update the config file
	if newState {
		// Check AWS credentials before enabling
		if err := s.bedrockService.CheckConnection(); err != nil {
			s.ShowNotification("Error", "AWS credentials not configured or CloudWatch access denied")
			// Revert checkbox state
			if currentState {
				s.bedrockItem.Check()
			} else {
				s.bedrockItem.Uncheck()
			}
			return
		}
		s.bedrockItem.Check()
		s.ShowNotification("Success", "Bedrock metrics enabled")
	} else {
		s.bedrockItem.Uncheck()
		s.ShowNotification("Success", "Bedrock metrics disabled")
	}

	// Signal configuration change
	s.bedrockChan <- struct{}{}
}

// handleVertexAIToggle handles toggling the Vertex AI tracking setting
func (s *SystrayController) handleVertexAIToggle() {
	if s.vertexAIService == nil {
		s.ShowNotification("Error", "Vertex AI service not available")
		return
	}

	// Check current state
	currentState := s.vertexAIService.IsEnabled()
	newState := !currentState

	// Update configuration through config service
	// This is a simplified implementation - in practice, you'd update the config file
	if newState {
		// Check GCP credentials before enabling
		if err := s.vertexAIService.CheckConnection(); err != nil {
			s.ShowNotification("Error", "GCP credentials not configured or Cloud Monitoring access denied")
			// Revert checkbox state
			if currentState {
				s.vertexAIItem.Check()
			} else {
				s.vertexAIItem.Uncheck()
			}
			return
		}
		s.vertexAIItem.Check()
		s.ShowNotification("Success", "Vertex AI metrics enabled")
	} else {
		s.vertexAIItem.Uncheck()
		s.ShowNotification("Success", "Vertex AI metrics disabled")
	}

	// Signal configuration change
	s.vertexAIChan <- struct{}{}
}
