package presenter

import (
	"encoding/json"
	"io"
	"os"
	"time"

	usecase "github.com/ca-srg/tosage/usecase/interface"
)

// JSONPresenterImpl implements JSONPresenter for JSON output
type JSONPresenterImpl struct {
	writer  io.Writer
	encoder *json.Encoder
}

// NewJSONPresenter creates a new JSON presenter
func NewJSONPresenter() *JSONPresenterImpl {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	return &JSONPresenterImpl{
		writer:  os.Stdout,
		encoder: encoder,
	}
}

// PrintDailyTokens prints daily token count as JSON
func (p *JSONPresenterImpl) PrintDailyTokens(date time.Time, tokens int) error {
	data := map[string]interface{}{
		"date":   date.Format("2006-01-02"),
		"tokens": tokens,
	}
	return p.encoder.Encode(data)
}

// PrintTokenStats prints token statistics as JSON
func (p *JSONPresenterImpl) PrintTokenStats(stats *usecase.TokenStatsResult) error {
	data := map[string]interface{}{
		"tokens": map[string]int{
			"input":         stats.InputTokens,
			"output":        stats.OutputTokens,
			"cacheCreation": stats.CacheCreationTokens,
			"cacheRead":     stats.CacheReadTokens,
			"total":         stats.TotalTokens,
		},
		"cost": map[string]interface{}{
			"amount":   stats.Cost,
			"currency": stats.Currency,
		},
		"entryCount": stats.EntryCount,
	}

	if stats.DateRange.Days > 0 {
		data["dateRange"] = map[string]interface{}{
			"start": stats.DateRange.Start.Format("2006-01-02"),
			"end":   stats.DateRange.End.Format("2006-01-02"),
			"days":  stats.DateRange.Days,
		}
	}

	return p.encoder.Encode(data)
}

// PrintCostBreakdown prints cost breakdown as JSON
func (p *JSONPresenterImpl) PrintCostBreakdown(result *usecase.CostBreakdownResult) error {
	breakdowns := make([]map[string]interface{}, len(result.Breakdowns))

	for i, item := range result.Breakdowns {
		breakdowns[i] = map[string]interface{}{
			"key": item.Key,
			"tokens": map[string]int{
				"input":         item.InputTokens,
				"output":        item.OutputTokens,
				"cacheCreation": item.CacheCreationTokens,
				"cacheRead":     item.CacheReadTokens,
				"total":         item.TotalTokens,
			},
			"cost": map[string]interface{}{
				"amount":   item.Cost,
				"currency": item.Currency,
			},
			"percentage": item.Percentage,
			"entryCount": item.EntryCount,
		}
	}

	data := map[string]interface{}{
		"breakdowns": breakdowns,
		"total": map[string]interface{}{
			"tokens": map[string]int{
				"input":         result.Total.InputTokens,
				"output":        result.Total.OutputTokens,
				"cacheCreation": result.Total.CacheCreationTokens,
				"cacheRead":     result.Total.CacheReadTokens,
				"total":         result.Total.TotalTokens,
			},
			"cost": map[string]interface{}{
				"amount":   result.Total.Cost,
				"currency": result.Total.Currency,
			},
			"entryCount": result.Total.EntryCount,
		},
	}

	return p.encoder.Encode(data)
}

// PrintModelBreakdown prints model breakdown as JSON
func (p *JSONPresenterImpl) PrintModelBreakdown(result *usecase.ModelBreakdownResult) error {
	models := make([]map[string]interface{}, len(result.Models))

	for i, model := range result.Models {
		models[i] = map[string]interface{}{
			"modelName": model.ModelName,
			"tokens": map[string]int{
				"input":         model.InputTokens,
				"output":        model.OutputTokens,
				"cacheCreation": model.CacheCreationTokens,
				"cacheRead":     model.CacheReadTokens,
				"total":         model.TotalTokens,
			},
			"cost": map[string]interface{}{
				"amount":   model.Cost,
				"currency": model.Currency,
			},
			"entryCount":      model.EntryCount,
			"tokenPercentage": model.TokenPercentage,
			"costPercentage":  model.CostPercentage,
		}
	}

	data := map[string]interface{}{
		"models": models,
		"total": map[string]interface{}{
			"tokens": map[string]int{
				"input":         result.Total.InputTokens,
				"output":        result.Total.OutputTokens,
				"cacheCreation": result.Total.CacheCreationTokens,
				"cacheRead":     result.Total.CacheReadTokens,
				"total":         result.Total.TotalTokens,
			},
			"cost": map[string]interface{}{
				"amount":   result.Total.Cost,
				"currency": result.Total.Currency,
			},
			"entryCount": result.Total.EntryCount,
		},
	}

	return p.encoder.Encode(data)
}

// PrintDateBreakdown prints date breakdown as JSON
func (p *JSONPresenterImpl) PrintDateBreakdown(result *usecase.DateBreakdownResult) error {
	dates := make([]map[string]interface{}, len(result.Dates))

	for i, date := range result.Dates {
		dates[i] = map[string]interface{}{
			"date": date.Date,
			"tokens": map[string]int{
				"input":         date.InputTokens,
				"output":        date.OutputTokens,
				"cacheCreation": date.CacheCreationTokens,
				"cacheRead":     date.CacheReadTokens,
				"total":         date.TotalTokens,
			},
			"cost": map[string]interface{}{
				"amount":   date.Cost,
				"currency": date.Currency,
			},
			"entryCount": date.EntryCount,
		}
	}

	data := map[string]interface{}{
		"dates": dates,
		"total": map[string]interface{}{
			"tokens": map[string]int{
				"input":         result.Total.InputTokens,
				"output":        result.Total.OutputTokens,
				"cacheCreation": result.Total.CacheCreationTokens,
				"cacheRead":     result.Total.CacheReadTokens,
				"total":         result.Total.TotalTokens,
			},
			"cost": map[string]interface{}{
				"amount":   result.Total.Cost,
				"currency": result.Total.Currency,
			},
			"entryCount": result.Total.EntryCount,
		},
	}

	return p.encoder.Encode(data)
}

// PrintCcSummary prints cc summary as JSON
func (p *JSONPresenterImpl) PrintCcSummary(summary *usecase.CcSummaryResult) error {
	data := map[string]interface{}{
		"totalTokens": summary.TotalTokens,
		"totalCost": map[string]interface{}{
			"amount":   summary.TotalCost,
			"currency": summary.Currency,
		},
		"entryCount":     summary.EntryCount,
		"uniqueProjects": summary.UniqueProjects,
		"uniqueModels":   summary.UniqueModels,
		"uniqueSessions": summary.UniqueSessions,
		"dateRange": map[string]interface{}{
			"start": summary.DateRange.Start.Format("2006-01-02"),
			"end":   summary.DateRange.End.Format("2006-01-02"),
			"days":  summary.DateRange.Days,
		},
		"averages": map[string]interface{}{
			"dailyTokens": summary.AverageDailyTokens,
			"dailyCost":   summary.AverageDailyCost,
		},
		"tokenDistribution": map[string]float64{
			"inputPercentage":         summary.TokenDistribution.InputPercentage,
			"outputPercentage":        summary.TokenDistribution.OutputPercentage,
			"cacheCreationPercentage": summary.TokenDistribution.CacheCreationPercentage,
			"cacheReadPercentage":     summary.TokenDistribution.CacheReadPercentage,
		},
	}

	if summary.MostUsedModel != "" {
		data["mostUsedModel"] = summary.MostUsedModel
	}
	if summary.MostActiveProject != "" {
		data["mostActiveProject"] = summary.MostActiveProject
	}

	return p.encoder.Encode(data)
}

// PrintCostEstimate prints monthly cost estimate as JSON
func (p *JSONPresenterImpl) PrintCostEstimate(estimate *usecase.CostEstimateResult) error {
	data := map[string]interface{}{
		"estimatedMonthlyCost": map[string]interface{}{
			"amount":   estimate.EstimatedMonthlyCost,
			"currency": estimate.Currency,
		},
		"basedOnDays":      estimate.BasedOnDays,
		"averageDailyCost": estimate.AverageDailyCost,
		"confidence":       estimate.Confidence,
	}

	return p.encoder.Encode(data)
}

// PrintCcData prints cc data entries as JSON
func (p *JSONPresenterImpl) PrintCcData(data *usecase.CcDataResult) error {
	entries := make([]map[string]interface{}, len(data.Entries))

	for i, entry := range data.Entries {
		entries[i] = map[string]interface{}{
			"id":          entry.ID,
			"timestamp":   entry.Timestamp.Format(time.RFC3339),
			"date":        entry.Date,
			"sessionId":   entry.SessionID,
			"projectPath": entry.ProjectPath,
			"model":       entry.Model,
			"tokens": map[string]int{
				"input":         entry.InputTokens,
				"output":        entry.OutputTokens,
				"cacheCreation": entry.CacheCreationTokens,
				"cacheRead":     entry.CacheReadTokens,
				"total":         entry.TotalTokens,
			},
			"cost": map[string]interface{}{
				"amount":   entry.Cost,
				"currency": entry.Currency,
			},
			"version": entry.Version,
		}

		if entry.MessageID != "" {
			entries[i]["messageId"] = entry.MessageID
		}
		if entry.RequestID != "" {
			entries[i]["requestId"] = entry.RequestID
		}
	}

	result := map[string]interface{}{
		"entries":    entries,
		"totalCount": data.TotalCount,
		"hasMore":    data.HasMore,
	}

	return p.encoder.Encode(result)
}

// PrintError prints an error as JSON
func (p *JSONPresenterImpl) PrintError(err error) error {
	data := map[string]interface{}{
		"error": map[string]string{
			"message": err.Error(),
		},
	}

	// Use stderr for errors
	encoder := json.NewEncoder(os.Stderr)
	encoder.SetIndent("", "  ")

	return encoder.Encode(data)
}

// SetWriter sets the output writer (mainly for testing)
func (p *JSONPresenterImpl) SetWriter(w io.Writer) {
	p.writer = w
	p.encoder = json.NewEncoder(w)
	p.encoder.SetIndent("", "  ")
}
