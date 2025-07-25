package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ca-srg/tosage/domain"
	"github.com/ca-srg/tosage/infrastructure/di"
)

func main() {
	// Parse command line flags
	var (
		cliMode    = flag.Bool("cli", false, "Run in CLI mode (default is daemon mode on macOS)")
		daemonMode = flag.Bool("daemon", false, "Run in daemon mode (macOS only)")
		debugMode  = flag.Bool("debug", false, "Enable debug logging to stdout")
	)
	flag.Parse()

	// Create DI container with options
	opts := []di.ContainerOption{}
	if *debugMode {
		opts = append(opts, di.WithDebugMode(true))
	}

	container, err := di.NewContainer(opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize application: %v\n", err)
		os.Exit(1)
	}

	// Get configuration
	config := container.GetConfig()

	// Determine mode based on flags and configuration
	runDaemon := false
	if *daemonMode {
		runDaemon = true
	} else if !*cliMode && config.Daemon != nil && config.Daemon.Enabled {
		runDaemon = true
	}

	// Run in appropriate mode
	if runDaemon {
		runDaemonMode(container)
	} else {
		runCLIMode(container)
	}
}

// handleShutdown handles graceful shutdown with signal handling
func handleShutdown(metricsService interface{ StopPeriodicMetrics() error }, logger domain.Logger) {
	// Create channel to listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for signal
	<-sigChan

	// Stop metrics service gracefully
	ctx := context.Background()
	logger.Info(ctx, "Shutting down metrics service...")
	if err := metricsService.StopPeriodicMetrics(); err != nil {
		logger.Error(ctx, "Error stopping metrics service", domain.NewField("error", err.Error()))
	}

	// Exit gracefully
	os.Exit(0)
}

// runCLIMode runs the application in CLI mode
func runCLIMode(container *di.Container) {
	// Get services
	cliController := container.GetCLIController()
	metricsService := container.GetMetricsService()

	// Get logger
	logger := container.CreateLogger("main")
	ctx := context.Background()

	// Start metrics service if Prometheus is enabled
	if err := metricsService.StartPeriodicMetrics(); err != nil {
		// Log error but don't fail application startup
		logger.Warn(ctx, "Failed to start metrics service", domain.NewField("error", err.Error()))
	}

	// Setup graceful shutdown
	go handleShutdown(metricsService, logger)

	// Run without arguments - always shows today's tokens in JST
	if err := cliController.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

// runDaemonMode runs the application in daemon mode
func runDaemonMode(container *di.Container) {
	// Get daemon controller
	// Get logger
	logger := container.CreateLogger("main")
	ctx := context.Background()

	daemonController := container.GetDaemonController()
	if daemonController == nil {
		logger.Error(ctx, "Daemon mode is not available. Please check your configuration.")
		os.Exit(1)
	}

	// Run the daemon controller on the main thread
	// This is required for macOS GUI components
	daemonController.Run()
}
