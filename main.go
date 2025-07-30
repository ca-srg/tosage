package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ca-srg/tosage/domain"
	infraConfig "github.com/ca-srg/tosage/infrastructure/config"
	"github.com/ca-srg/tosage/infrastructure/di"
	"github.com/ca-srg/tosage/interface/cli"
	"github.com/ca-srg/tosage/usecase/impl"
)

func main() {
	// Parse command line flags
	var (
		cliMode         = flag.Bool("cli", false, "Run in CLI mode (default is daemon mode on macOS)")
		daemonMode      = flag.Bool("daemon", false, "Run in daemon mode (macOS only)")
		debugMode       = flag.Bool("debug", false, "Enable debug logging to stdout")
		includeBedrock  = flag.Bool("bedrock", false, "Include AWS Bedrock usage metrics (requires AWS credentials)")
		includeVertexAI = flag.Bool("vertex-ai", false, "Include Google Vertex AI usage metrics (requires Google Cloud credentials)")

		// CSV export flags
		exportCSV   = flag.Bool("export-csv", false, "Export metrics to CSV file")
		output      = flag.String("output", "", "Output CSV file path (default: metrics_YYYYMMDD_HHMMSS.csv)")
		startTime   = flag.String("start-time", "", "Start time in ISO 8601 format (default: 30 days ago)")
		endTime     = flag.String("end-time", "", "End time in ISO 8601 format (default: now)")
		metricTypes = flag.String("metrics-types", "", "Comma-separated list of metric types to export (claude_code,cursor,bedrock,vertex_ai,all)")
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

	// Check if CSV export mode is requested
	if *exportCSV {
		runCSVExportMode(container, *output, *startTime, *endTime, *metricTypes)
		return
	}

	// Determine mode based on flags and configuration
	runDaemon := false
	if *daemonMode {
		runDaemon = true
		// Force enable daemon in config when --daemon flag is used
		if config.Daemon == nil {
			config.Daemon = &infraConfig.DaemonConfig{
				Enabled:      true,
				StartAtLogin: false,
				LogPath:      "/tmp/tosage.log",
				PidFile:      "/tmp/tosage.pid",
			}
		} else {
			config.Daemon.Enabled = true
		}
		// Re-initialize daemon components
		if err := container.InitDaemonComponents(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize daemon components: %v\n", err)
			os.Exit(1)
		}
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
	bedrockEnabled := config.Bedrock != nil && config.Bedrock.Enabled
	vertexAIEnabled := config.VertexAI != nil && config.VertexAI.Enabled

	if bedrockEnabled || vertexAIEnabled {
		cliController.SetSkipCCMetrics(true)

		// Check if services were properly initialized and set them to CLI controller
		bedrockService := container.GetBedrockService()
		vertexAIService := container.GetVertexAIService()

		// Set services to CLI controller
		cliController.SetBedrockService(bedrockService)
		cliController.SetVertexAIService(vertexAIService)

		// Provide feedback if services failed to initialize
		if bedrockEnabled && bedrockService == nil {
			fmt.Fprintf(os.Stderr, "Warning: Bedrock was enabled but service initialization failed\n")
		}
		if vertexAIEnabled && vertexAIService == nil {
			fmt.Fprintf(os.Stderr, "Warning: Vertex AI was enabled but service initialization failed\n")
		}
	}

	metricsService := container.GetMetricsService()

	// Get logger
	logger := container.CreateLogger("main")
	ctx := context.Background()

	// Check if vertex-ai flag is set
	if vertexAIEnabled {
		// Send metrics once to Prometheus
		if err := metricsService.SendCurrentMetrics(); err != nil {
			logger.Error(ctx, "Failed to send metrics to Prometheus", domain.NewField("error", err.Error()))
			fmt.Fprintf(os.Stderr, "Failed to send metrics to Prometheus: %v\n", err)
			// Continue to display token count even if sending fails
		} else {
			logger.Info(ctx, "Successfully sent Vertex AI metrics to Prometheus")
		}
		
		// Display token count and exit
		if err := cliController.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		return
	}

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

	// Call Run() using a helper function to avoid platform-specific type issues
	runDaemonController(daemonController, logger, ctx)
}

// runDaemonController is a helper function to run the daemon controller
// It handles platform-specific type differences
func runDaemonController(daemonController interface{}, logger domain.Logger, ctx context.Context) {
	// Use reflection to call Run() method regardless of the concrete type
	// This works for both *controller.DaemonController (Darwin) and interface{} (non-Darwin)
	if runner, ok := daemonController.(interface{ Run() }); ok {
		runner.Run()
	} else {
		logger.Error(ctx, "Daemon controller does not implement Run method or daemon mode is not supported on this platform")
		os.Exit(1)
	}
}

// runCSVExportMode runs the application in CSV export mode
func runCSVExportMode(container *di.Container, outputPath, startTimeStr, endTimeStr, metricTypesStr string) {
	// Get logger
	logger := container.CreateLogger("main")
	ctx := context.Background()

	// Validate that --metrics-types is specified
	if metricTypesStr == "" {
		fmt.Fprintf(os.Stderr, "Error: --metrics-types is required when using --export-csv\n")
		fmt.Fprintf(os.Stderr, "Available metric types: claude_code, cursor, bedrock, vertex_ai, all\n")
		fmt.Fprintf(os.Stderr, "Example: tosage --export-csv --metrics-types \"claude_code,cursor\"\n")
		os.Exit(1)
	}

	// Parse metric types
	var metricTypes []string
	if metricTypesStr != "" {
		metricTypes = strings.Split(metricTypesStr, ",")
		// Trim spaces
		for i := range metricTypes {
			metricTypes[i] = strings.TrimSpace(metricTypes[i])
		}
	}

	// Generate export options
	options, err := impl.GenerateExportOptions(outputPath, startTimeStr, endTimeStr, metricTypes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid export options: %v\n", err)
		os.Exit(1)
	}

	// Get CSV export service
	csvExportService := container.GetCSVExportService()
	if csvExportService == nil {
		fmt.Fprintf(os.Stderr, "CSV export service not available\n")
		os.Exit(1)
	}

	// Perform export
	logger.Info(ctx, "Starting CSV export",
		domain.NewField("output", outputPath),
		domain.NewField("startTime", startTimeStr),
		domain.NewField("endTime", endTimeStr),
		domain.NewField("metricTypes", metricTypes))

	if err := csvExportService.Export(*options); err != nil {
		fmt.Fprintf(os.Stderr, "Export failed: %v\n", err)
		os.Exit(1)
	}

	// Display the output path that was actually used
	actualOutputPath := outputPath
	if actualOutputPath == "" {
		actualOutputPath = options.OutputPath
	}

	fmt.Printf("Successfully exported metrics to: %s\n", actualOutputPath)
}
