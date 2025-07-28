package controller

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ca-srg/tosage/domain"
	usecase "github.com/ca-srg/tosage/usecase/interface"
)

// ExportController handles export command operations
type ExportController struct {
	exportService usecase.ExportService
	logger        domain.Logger
}

// NewExportController creates a new export controller
func NewExportController(
	exportService usecase.ExportService,
	logger domain.Logger,
) *ExportController {
	return &ExportController{
		exportService: exportService,
		logger:        logger,
	}
}

// Run executes the export command with provided arguments
func (c *ExportController) Run(args []string) error {
	// Create flagset for export-prometheus command
	fs := flag.NewFlagSet("export-prometheus", flag.ExitOnError)

	// Define flags
	var (
		date      = fs.String("date", "", "Export data for a specific date (YYYY-MM-DD)")
		startDate = fs.String("start-date", "", "Start date for export range (YYYY-MM-DD)")
		endDate   = fs.String("end-date", "", "End date for export range (YYYY-MM-DD)")
		output    = fs.String("output", "", "Output file path (default: auto-generated in current directory)")
	)

	// Custom usage
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: tosage export-prometheus [options]\n\n")
		fmt.Fprintf(os.Stderr, "Export metrics data from Prometheus to CSV file.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Export today's data\n")
		fmt.Fprintf(os.Stderr, "  tosage export-prometheus\n\n")
		fmt.Fprintf(os.Stderr, "  # Export specific date\n")
		fmt.Fprintf(os.Stderr, "  tosage export-prometheus --date 2024-01-15\n\n")
		fmt.Fprintf(os.Stderr, "  # Export date range\n")
		fmt.Fprintf(os.Stderr, "  tosage export-prometheus --start-date 2024-01-01 --end-date 2024-01-31\n\n")
		fmt.Fprintf(os.Stderr, "  # Export to specific file\n")
		fmt.Fprintf(os.Stderr, "  tosage export-prometheus --date 2024-01-15 --output /path/to/metrics.csv\n")
	}

	// Parse flags
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	// Create export request
	req := &usecase.ExportRequest{
		OutputPath: *output,
	}

	// Parse date parameters
	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		return fmt.Errorf("failed to load JST timezone: %w", err)
	}

	// Handle date flags
	if *date != "" {
		// Single date specified
		if *startDate != "" || *endDate != "" {
			return fmt.Errorf("cannot use --date with --start-date or --end-date")
		}

		parsedDate, err := time.ParseInLocation("2006-01-02", *date, jst)
		if err != nil {
			return fmt.Errorf("invalid date format (use YYYY-MM-DD): %w", err)
		}
		req.SingleDate = &parsedDate

	} else if *startDate != "" || *endDate != "" {
		// Date range specified
		if *startDate == "" || *endDate == "" {
			return fmt.Errorf("both --start-date and --end-date must be specified for range export")
		}

		start, err := time.ParseInLocation("2006-01-02", *startDate, jst)
		if err != nil {
			return fmt.Errorf("invalid start date format (use YYYY-MM-DD): %w", err)
		}

		end, err := time.ParseInLocation("2006-01-02", *endDate, jst)
		if err != nil {
			return fmt.Errorf("invalid end date format (use YYYY-MM-DD): %w", err)
		}

		// Set end time to end of day
		end = end.Add(24*time.Hour - time.Second)

		req.StartDate = &start
		req.EndDate = &end
	}
	// If no date is specified, the service will default to today

	// Execute export
	ctx := context.Background()
	c.logger.Info(ctx, "Starting metrics export")

	if err := c.exportService.ExportMetricsToCSV(ctx, req); err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	return nil
}
