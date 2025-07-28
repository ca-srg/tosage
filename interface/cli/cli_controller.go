package cli

import (
	"fmt"
	"time"

	"github.com/ca-srg/tosage/interface/presenter"
	usecase "github.com/ca-srg/tosage/usecase/interface"
)

// CLIController handles command-line interface operations
type CLIController struct {
	ccService        usecase.CcService
	consolePresenter presenter.ConsolePresenter
	jsonPresenter    presenter.JSONPresenter
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

// Run executes the CLI controller - always shows today's tokens in JST
func (c *CLIController) Run() error {
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
