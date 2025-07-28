package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/ca-srg/tosage/interface/presenter"
	usecase "github.com/ca-srg/tosage/usecase/interface"
)

// CLIController handles command-line interface operations
type CLIController struct {
	ccService        usecase.CcService
	consolePresenter presenter.ConsolePresenter
	jsonPresenter    presenter.JSONPresenter
	skipCCMetrics    bool
	bedrockService   usecase.BedrockService
	vertexAIService  usecase.VertexAIService
}

// NewCLIController creates a new CLI controller
func NewCLIController(
	ccService usecase.CcService,
	consolePresenter presenter.ConsolePresenter,
	jsonPresenter presenter.JSONPresenter,
) *CLIController {
	return &CLIController{
		ccService:        ccService,
		consolePresenter: consolePresenter,
		jsonPresenter:    jsonPresenter,
	}
}

// SetSkipCCMetrics sets whether to skip Claude Code and Cursor metrics
func (c *CLIController) SetSkipCCMetrics(skip bool) {
	c.skipCCMetrics = skip
}

// SetBedrockService sets the Bedrock service
func (c *CLIController) SetBedrockService(service usecase.BedrockService) {
	c.bedrockService = service
}

// SetVertexAIService sets the Vertex AI service
func (c *CLIController) SetVertexAIService(service usecase.VertexAIService) {
	c.vertexAIService = service
}

// Run executes the CLI controller - always shows today's tokens in JST
func (c *CLIController) Run() error {
	// If skip CC metrics is enabled, try to show Bedrock/Vertex AI metrics instead
	if c.skipCCMetrics {
		// Try to get and display Bedrock metrics
		if c.bedrockService != nil && c.bedrockService.IsEnabled() {
			jst, _ := time.LoadLocation("Asia/Tokyo")
			today := time.Now().In(jst)
			usage, err := c.bedrockService.GetDailyUsage(today)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to get Bedrock usage: %v\n", err)
			} else if usage != nil {
				fmt.Printf("Bedrock tokens today: %d\n", usage.TotalTokens())
			}
		}

		// Try to get and display Vertex AI metrics
		if c.vertexAIService != nil && c.vertexAIService.IsEnabled() {
			jst, _ := time.LoadLocation("Asia/Tokyo")
			today := time.Now().In(jst)
			usage, err := c.vertexAIService.GetDailyUsage(today)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to get Vertex AI usage: %v\n", err)
			} else if usage != nil {
				fmt.Printf("Vertex AI tokens today: %d\n", usage.TotalTokens())
			}
		}

		return nil
	}

	// Original CC metrics logic
	if c.ccService == nil {
		return nil
	}

	// Get JST timezone
	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		return fmt.Errorf("failed to load JST timezone: %w", err)
	}

	// Get current time in JST
	now := time.Now().In(jst)

	// Calculate today's start time (00:00 JST)
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, jst)

	// Get cc entries from start of day to current time
	entries, err := c.ccService.LoadCcData(usecase.CcDataFilter{
		StartDate: &startOfDay,
		EndDate:   &now,
	})
	if err != nil {
		return fmt.Errorf("failed to load cc data: %w", err)
	}

	// Calculate total tokens
	totalTokens := 0
	for _, entry := range entries.Entries {
		totalTokens += entry.TotalTokens
	}

	// Just print the number
	fmt.Println(totalTokens)
	return nil
}
