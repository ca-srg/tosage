package presenter

import (
	"time"

	usecase "github.com/ca-srg/tosage/usecase/interface"
)

// ConsolePresenter handles console output formatting
type ConsolePresenter interface {
	// Version and basic output
	PrintVersion()
	PrintError(err error)
	PrintStringList(title string, items []string) error

	// Token-related output
	PrintDailyTokens(date time.Time, tokens int) error
	PrintDailyTokensVerbose(date time.Time, tokens int) error
	PrintTokenStats(stats *usecase.TokenStatsResult) error

	// Breakdown output
	PrintCostBreakdown(result *usecase.CostBreakdownResult, groupBy string) error
	PrintModelBreakdown(result *usecase.ModelBreakdownResult) error
	PrintDateBreakdown(result *usecase.DateBreakdownResult) error

	// Summary and estimates
	PrintCcSummary(summary *usecase.CcSummaryResult) error
	PrintCostEstimate(estimate *usecase.CostEstimateResult) error

	// Data listing
	PrintCcData(data *usecase.CcDataResult) error
}

// JSONPresenter handles JSON output formatting
type JSONPresenter interface {
	// Token-related output
	PrintDailyTokens(date time.Time, tokens int) error
	PrintTokenStats(stats *usecase.TokenStatsResult) error

	// Breakdown output
	PrintCostBreakdown(result *usecase.CostBreakdownResult) error
	PrintModelBreakdown(result *usecase.ModelBreakdownResult) error
	PrintDateBreakdown(result *usecase.DateBreakdownResult) error

	// Summary and estimates
	PrintCcSummary(summary *usecase.CcSummaryResult) error
	PrintCostEstimate(estimate *usecase.CostEstimateResult) error

	// Data listing
	PrintCcData(data *usecase.CcDataResult) error
}
