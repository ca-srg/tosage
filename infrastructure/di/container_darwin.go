//go:build darwin
// +build darwin

package di

import (
	"github.com/ca-srg/tosage/interface/controller"
)

// DarwinContainer holds Darwin-specific components
type DarwinContainer struct {
	systrayController *controller.SystrayController
	daemonController  *controller.DaemonController
}

// initDaemonPlatform initializes daemon components for Darwin
func (c *Container) initDaemonPlatform() error {
	// Only initialize if daemon mode is configured
	if c.config.Daemon == nil || !c.config.Daemon.Enabled {
		return nil
	}

	// Initialize systray controller
	systrayController := controller.NewSystrayController(
		c.ccService,
		c.statusService,
		c.metricsService,
		c.configService,
		c.bedrockService,
		c.vertexAIService,
	)

	// Initialize daemon controller
	daemonController := controller.NewDaemonController(
		c.config,
		c.configService,
		c.ccService,
		c.statusService,
		c.metricsService,
		systrayController,
		c.CreateLogger("daemon"),
	)

	// Store in Darwin-specific container
	c.darwinContainer = &DarwinContainer{
		systrayController: systrayController,
		daemonController:  daemonController,
	}

	return nil
}

// GetSystrayController returns the systray controller (Darwin only)
func (c *Container) GetSystrayController() *controller.SystrayController {
	if c.darwinContainer != nil {
		return c.darwinContainer.systrayController
	}
	return nil
}

// GetDaemonController returns the daemon controller (Darwin only)
func (c *Container) GetDaemonController() *controller.DaemonController {
	if c.darwinContainer != nil {
		return c.darwinContainer.daemonController
	}
	return nil
}