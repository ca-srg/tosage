package presenter

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	usecase "github.com/ca-srg/tosage/usecase/interface"
)

// ConsolePresenterImpl implements ConsolePresenter for terminal output
type ConsolePresenterImpl struct {
	writer io.Writer
}

// NewConsolePresenter creates a new console presenter
func NewConsolePresenter() *ConsolePresenterImpl {
	return &ConsolePresenterImpl{
		writer: os.Stdout,
	}
}

// PrintVersion prints version information
func (p *ConsolePresenterImpl) PrintVersion() {
	_, _ = fmt.Fprintln(p.writer, "tosage version 1.0.0")
}

// PrintError prints an error message
func (p *ConsolePresenterImpl) PrintError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}

// PrintStringList prints a list of strings with a title
func (p *ConsolePresenterImpl) PrintStringList(title string, items []string) error {
	_, _ = fmt.Fprintf(p.writer, "%s:\n", title)
	for _, item := range items {
		_, _ = fmt.Fprintf(p.writer, "  - %s\n", item)
	}
	return nil
}

// PrintDailyTokens prints daily token count (simple format)
func (p *ConsolePresenterImpl) PrintDailyTokens(date time.Time, tokens int) error {
	_, _ = fmt.Fprintln(p.writer, tokens)
	return nil
}

// PrintDailyTokensVerbose prints daily token count with date
func (p *ConsolePresenterImpl) PrintDailyTokensVerbose(date time.Time, tokens int) error {
	_, _ = fmt.Fprintf(p.writer, "Date: %s\n", date.Format("2006-01-02"))
	_, _ = fmt.Fprintf(p.writer, "Total Tokens: %s\n", p.formatNumber(tokens))
	return nil
}

// PrintTokenStats prints token statistics
func (p *ConsolePresenterImpl) PrintTokenStats(stats *usecase.TokenStatsResult) error {
	_, _ = fmt.Fprintln(p.writer, "Token Cc Statistics")
	_, _ = fmt.Fprintln(p.writer, strings.Repeat("=", 50))

	if stats.DateRange.Days > 0 {
		_, _ = fmt.Fprintf(p.writer, "Period: %s to %s (%d days)\n",
			stats.DateRange.Start.Format("2006-01-02"),
			stats.DateRange.End.Format("2006-01-02"),
			stats.DateRange.Days)
	}

	_, _ = fmt.Fprintf(p.writer, "Entries: %s\n", p.formatNumber(stats.EntryCount))
	_, _ = fmt.Fprintln(p.writer)

	// Token breakdown
	_, _ = fmt.Fprintln(p.writer, "Token Breakdown:")
	_, _ = fmt.Fprintf(p.writer, "  Input Tokens:         %s\n", p.formatNumber(stats.InputTokens))
	_, _ = fmt.Fprintf(p.writer, "  Output Tokens:        %s\n", p.formatNumber(stats.OutputTokens))
	_, _ = fmt.Fprintf(p.writer, "  Cache Creation:       %s\n", p.formatNumber(stats.CacheCreationTokens))
	_, _ = fmt.Fprintf(p.writer, "  Cache Read:           %s\n", p.formatNumber(stats.CacheReadTokens))
	_, _ = fmt.Fprintf(p.writer, "  Total Tokens:         %s\n", p.formatNumber(stats.TotalTokens))
	_, _ = fmt.Fprintln(p.writer)

	// Cost
	_, _ = fmt.Fprintf(p.writer, "Total Cost: %s %.2f\n", p.getCurrencySymbol(stats.Currency), stats.Cost)

	return nil
}

// PrintCostBreakdown prints cost breakdown
func (p *ConsolePresenterImpl) PrintCostBreakdown(result *usecase.CostBreakdownResult, groupBy string) error {
	_, _ = fmt.Fprintf(p.writer, "Cost Breakdown by %s\n", groupBy)
	_, _ = fmt.Fprintln(p.writer, strings.Repeat("=", 80))

	// Create table
	w := tabwriter.NewWriter(p.writer, 0, 0, 2, ' ', 0)

	// Header
	_, _ = fmt.Fprintf(w, "%s\tTokens\tCost\tPercentage\tEntries\n", groupBy)
	_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
		strings.Repeat("-", 20),
		strings.Repeat("-", 15),
		strings.Repeat("-", 12),
		strings.Repeat("-", 10),
		strings.Repeat("-", 7))

	// Data rows
	for _, item := range result.Breakdowns {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s %.2f\t%.1f%%\t%d\n",
			p.truncateString(item.Key, 20),
			p.formatNumber(item.TotalTokens),
			p.getCurrencySymbol(item.Currency),
			item.Cost,
			item.Percentage,
			item.EntryCount)
	}

	// Total row
	_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
		strings.Repeat("-", 20),
		strings.Repeat("-", 15),
		strings.Repeat("-", 12),
		strings.Repeat("-", 10),
		strings.Repeat("-", 7))
	_, _ = fmt.Fprintf(w, "Total\t%s\t%s %.2f\t100.0%%\t%d\n",
		p.formatNumber(result.Total.TotalTokens),
		p.getCurrencySymbol(result.Total.Currency),
		result.Total.Cost,
		result.Total.EntryCount)

	_ = w.Flush()
	return nil
}

// PrintModelBreakdown prints model breakdown
func (p *ConsolePresenterImpl) PrintModelBreakdown(result *usecase.ModelBreakdownResult) error {
	_, _ = fmt.Fprintln(p.writer, "Model Cc Breakdown")
	_, _ = fmt.Fprintln(p.writer, strings.Repeat("=", 100))

	// Create table
	w := tabwriter.NewWriter(p.writer, 0, 0, 2, ' ', 0)

	// Header
	_, _ = fmt.Fprintf(w, "Model\tInput\tOutput\tCache\tTotal Tokens\tCost\tToken %%\tCost %%\n")
	_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		strings.Repeat("-", 25),
		strings.Repeat("-", 10),
		strings.Repeat("-", 10),
		strings.Repeat("-", 10),
		strings.Repeat("-", 12),
		strings.Repeat("-", 10),
		strings.Repeat("-", 8),
		strings.Repeat("-", 8))

	// Data rows
	for _, model := range result.Models {
		cacheTokens := model.CacheCreationTokens + model.CacheReadTokens
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s %.2f\t%.1f%%\t%.1f%%\n",
			p.truncateString(model.ModelName, 25),
			p.formatNumber(model.InputTokens),
			p.formatNumber(model.OutputTokens),
			p.formatNumber(cacheTokens),
			p.formatNumber(model.TotalTokens),
			p.getCurrencySymbol(model.Currency),
			model.Cost,
			model.TokenPercentage,
			model.CostPercentage)
	}

	// Total row
	_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		strings.Repeat("-", 25),
		strings.Repeat("-", 10),
		strings.Repeat("-", 10),
		strings.Repeat("-", 10),
		strings.Repeat("-", 12),
		strings.Repeat("-", 10),
		strings.Repeat("-", 8),
		strings.Repeat("-", 8))

	totalCache := result.Total.CacheCreationTokens + result.Total.CacheReadTokens
	_, _ = fmt.Fprintf(w, "Total\t%s\t%s\t%s\t%s\t%s %.2f\t100.0%%\t100.0%%\n",
		p.formatNumber(result.Total.InputTokens),
		p.formatNumber(result.Total.OutputTokens),
		p.formatNumber(totalCache),
		p.formatNumber(result.Total.TotalTokens),
		p.getCurrencySymbol(result.Total.Currency),
		result.Total.Cost)

	_ = w.Flush()
	return nil
}

// PrintDateBreakdown prints date breakdown
func (p *ConsolePresenterImpl) PrintDateBreakdown(result *usecase.DateBreakdownResult) error {
	_, _ = fmt.Fprintln(p.writer, "Daily Cc Breakdown")
	_, _ = fmt.Fprintln(p.writer, strings.Repeat("=", 80))

	// Create table
	w := tabwriter.NewWriter(p.writer, 0, 0, 2, ' ', 0)

	// Header
	_, _ = fmt.Fprintf(w, "Date\tInput\tOutput\tCache\tTotal Tokens\tCost\tEntries\n")
	_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		strings.Repeat("-", 10),
		strings.Repeat("-", 10),
		strings.Repeat("-", 10),
		strings.Repeat("-", 10),
		strings.Repeat("-", 12),
		strings.Repeat("-", 10),
		strings.Repeat("-", 7))

	// Data rows
	for _, date := range result.Dates {
		cacheTokens := date.CacheCreationTokens + date.CacheReadTokens
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s %.2f\t%d\n",
			date.Date,
			p.formatNumber(date.InputTokens),
			p.formatNumber(date.OutputTokens),
			p.formatNumber(cacheTokens),
			p.formatNumber(date.TotalTokens),
			p.getCurrencySymbol(date.Currency),
			date.Cost,
			date.EntryCount)
	}

	// Total row
	_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		strings.Repeat("-", 10),
		strings.Repeat("-", 10),
		strings.Repeat("-", 10),
		strings.Repeat("-", 10),
		strings.Repeat("-", 12),
		strings.Repeat("-", 10),
		strings.Repeat("-", 7))

	totalCache := result.Total.CacheCreationTokens + result.Total.CacheReadTokens
	_, _ = fmt.Fprintf(w, "Total\t%s\t%s\t%s\t%s\t%s %.2f\t%d\n",
		p.formatNumber(result.Total.InputTokens),
		p.formatNumber(result.Total.OutputTokens),
		p.formatNumber(totalCache),
		p.formatNumber(result.Total.TotalTokens),
		p.getCurrencySymbol(result.Total.Currency),
		result.Total.Cost,
		result.Total.EntryCount)

	_ = w.Flush()
	return nil
}

// PrintCcSummary prints usage summary
func (p *ConsolePresenterImpl) PrintCcSummary(summary *usecase.CcSummaryResult) error {
	_, _ = fmt.Fprintln(p.writer, "Cc Summary")
	_, _ = fmt.Fprintln(p.writer, strings.Repeat("=", 60))

	// Period
	_, _ = fmt.Fprintf(p.writer, "Period: %s to %s (%d days)\n",
		summary.DateRange.Start.Format("2006-01-02"),
		summary.DateRange.End.Format("2006-01-02"),
		summary.DateRange.Days)
	_, _ = fmt.Fprintln(p.writer)

	// Overview
	_, _ = fmt.Fprintln(p.writer, "Overview:")
	_, _ = fmt.Fprintf(p.writer, "  Total Tokens:       %s\n", p.formatNumber(summary.TotalTokens))
	_, _ = fmt.Fprintf(p.writer, "  Total Cost:         %s %.2f\n", p.getCurrencySymbol(summary.Currency), summary.TotalCost)
	_, _ = fmt.Fprintf(p.writer, "  Total Entries:      %s\n", p.formatNumber(summary.EntryCount))
	_, _ = fmt.Fprintln(p.writer)

	// Daily averages
	_, _ = fmt.Fprintln(p.writer, "Daily Averages:")
	_, _ = fmt.Fprintf(p.writer, "  Tokens per Day:     %s\n", p.formatNumber(summary.AverageDailyTokens))
	_, _ = fmt.Fprintf(p.writer, "  Cost per Day:       %s %.2f\n", p.getCurrencySymbol(summary.Currency), summary.AverageDailyCost)
	_, _ = fmt.Fprintln(p.writer)

	// Cc patterns
	_, _ = fmt.Fprintln(p.writer, "Cc Patterns:")
	_, _ = fmt.Fprintf(p.writer, "  Unique Projects:    %d\n", summary.UniqueProjects)
	_, _ = fmt.Fprintf(p.writer, "  Unique Models:      %d\n", summary.UniqueModels)
	_, _ = fmt.Fprintf(p.writer, "  Unique Sessions:    %d\n", summary.UniqueSessions)
	if summary.MostUsedModel != "" {
		_, _ = fmt.Fprintf(p.writer, "  Most Used Model:    %s\n", summary.MostUsedModel)
	}
	if summary.MostActiveProject != "" {
		_, _ = fmt.Fprintf(p.writer, "  Most Active Project: %s\n", summary.MostActiveProject)
	}
	_, _ = fmt.Fprintln(p.writer)

	// Token distribution
	_, _ = fmt.Fprintln(p.writer, "Token Distribution:")
	_, _ = fmt.Fprintf(p.writer, "  Input:              %.1f%%\n", summary.TokenDistribution.InputPercentage)
	_, _ = fmt.Fprintf(p.writer, "  Output:             %.1f%%\n", summary.TokenDistribution.OutputPercentage)
	_, _ = fmt.Fprintf(p.writer, "  Cache Creation:     %.1f%%\n", summary.TokenDistribution.CacheCreationPercentage)
	_, _ = fmt.Fprintf(p.writer, "  Cache Read:         %.1f%%\n", summary.TokenDistribution.CacheReadPercentage)

	return nil
}

// PrintCostEstimate prints monthly cost estimate
func (p *ConsolePresenterImpl) PrintCostEstimate(estimate *usecase.CostEstimateResult) error {
	_, _ = fmt.Fprintln(p.writer, "Monthly Cost Estimate")
	_, _ = fmt.Fprintln(p.writer, strings.Repeat("=", 40))

	_, _ = fmt.Fprintf(p.writer, "Estimated Monthly Cost: %s %.2f\n",
		p.getCurrencySymbol(estimate.Currency),
		estimate.EstimatedMonthlyCost)
	_, _ = fmt.Fprintln(p.writer)

	_, _ = fmt.Fprintf(p.writer, "Based on:         %d days of data\n", estimate.BasedOnDays)
	_, _ = fmt.Fprintf(p.writer, "Average Daily:    %s %.2f\n",
		p.getCurrencySymbol(estimate.Currency),
		estimate.AverageDailyCost)
	_, _ = fmt.Fprintf(p.writer, "Confidence:       %.0f%%\n", estimate.Confidence*100)

	if estimate.Confidence < 0.5 {
		_, _ = fmt.Fprintln(p.writer, "\nNote: Low confidence due to limited data.")
	}

	return nil
}

// PrintCcData prints usage data entries
func (p *ConsolePresenterImpl) PrintCcData(data *usecase.CcDataResult) error {
	_, _ = fmt.Fprintf(p.writer, "Cc Data (%d entries", data.TotalCount)
	if data.HasMore {
		_, _ = fmt.Fprintf(p.writer, ", showing %d", len(data.Entries))
	}
	_, _ = fmt.Fprintln(p.writer, ")")
	_, _ = fmt.Fprintln(p.writer, strings.Repeat("=", 100))

	if len(data.Entries) == 0 {
		_, _ = fmt.Fprintln(p.writer, "No entries found.")
		return nil
	}

	// Create table
	w := tabwriter.NewWriter(p.writer, 0, 0, 2, ' ', 0)

	// Header
	_, _ = fmt.Fprintf(w, "Timestamp\tProject\tModel\tTokens\tCost\n")
	_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
		strings.Repeat("-", 19),
		strings.Repeat("-", 20),
		strings.Repeat("-", 20),
		strings.Repeat("-", 10),
		strings.Repeat("-", 10))

	// Data rows
	for _, entry := range data.Entries {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s %.2f\n",
			entry.Timestamp.Format("2006-01-02 15:04:05"),
			p.truncateString(entry.ProjectPath, 20),
			p.truncateString(entry.Model, 20),
			p.formatNumber(entry.TotalTokens),
			p.getCurrencySymbol(entry.Currency),
			entry.Cost)
	}

	_ = w.Flush()

	if data.HasMore {
		_, _ = fmt.Fprintln(p.writer, "\n... more entries available")
	}

	return nil
}

// Helper methods

func (p *ConsolePresenterImpl) formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}

	// Format with commas
	str := fmt.Sprintf("%d", n)
	result := ""
	for i, digit := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result += ","
		}
		result += string(digit)
	}
	return result
}

func (p *ConsolePresenterImpl) getCurrencySymbol(currency string) string {
	switch currency {
	case "USD":
		return "$"
	case "EUR":
		return "€"
	case "JPY":
		return "¥"
	default:
		return currency + " "
	}
}

func (p *ConsolePresenterImpl) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
