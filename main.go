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
	"github.com/ca-srg/tosage/interface/cli"
)

func main() {
	// Parse command line flags
	var (
		cliMode         = flag.Bool("cli", false, "Run in CLI mode (default is daemon mode on macOS)")
		daemonMode      = flag.Bool("daemon", false, "Run in daemon mode (macOS only)")
		debugMode       = flag.Bool("debug", false, "Enable debug logging to stdout")
		includeBedrock  = flag.Bool("bedrock", false, "Include AWS Bedrock usage metrics (requires AWS credentials)")
		includeVertexAI = flag.Bool("vertex-ai", false, "Include Google Vertex AI usage metrics (requires Google Cloud credentials)")
	)
	flag.Parse()

	// Create DI container with options
	opts := []di.ContainerOption{}
	if *debugMode {
		opts = append(opts, di.WithDebugMode(true))
	}
	if *includeBedrock {
		opts = append(opts, di.WithBedrockEnabled(true))
	}
	if *includeVertexAI {
		opts = append(opts, di.WithVertexAIEnabled(true))
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

	// Daemon mode is not supported when Bedrock or Vertex AI flags are set
	if runDaemon && (*includeBedrock || *includeVertexAI) {
		fmt.Fprintf(os.Stderr, "Daemon mode is not supported when --bedrock or --vertex-ai flags are set\n")
		os.Exit(1)
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
	cliControllerIface := container.GetCLIController()
	cliController, ok := cliControllerIface.(*cli.CLIController)
	if !ok || cliController == nil {
		fmt.Fprintf(os.Stderr, "CLI controller not available\n")
		os.Exit(1)
	}

	// Skip Claude Code and Cursor metrics if Bedrock or Vertex AI is enabled
	config := container.GetConfig()
	if (config.Bedrock != nil && config.Bedrock.Enabled) || (config.VertexAI != nil && config.VertexAI.Enabled) {
		cliController.SetSkipCCMetrics(true)
	}

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
