package valueobject

// TokenStats represents token usage statistics
type TokenStats struct {
	inputTokens         int
	outputTokens        int
	cacheCreationTokens int
	cacheReadTokens     int
}

// NewTokenStats creates a new TokenStats value object
func NewTokenStats(
	inputTokens int,
	outputTokens int,
	cacheCreationTokens int,
	cacheReadTokens int,
) TokenStats {
	return TokenStats{
		inputTokens:         inputTokens,
		outputTokens:        outputTokens,
		cacheCreationTokens: cacheCreationTokens,
		cacheReadTokens:     cacheReadTokens,
	}
}

// NewEmptyTokenStats creates an empty TokenStats
func NewEmptyTokenStats() TokenStats {
	return TokenStats{}
}

// InputTokens returns the number of input tokens
func (ts TokenStats) InputTokens() int {
	return ts.inputTokens
}

// OutputTokens returns the number of output tokens
func (ts TokenStats) OutputTokens() int {
	return ts.outputTokens
}

// CacheCreationTokens returns the number of cache creation tokens
func (ts TokenStats) CacheCreationTokens() int {
	return ts.cacheCreationTokens
}

// CacheReadTokens returns the number of cache read tokens
func (ts TokenStats) CacheReadTokens() int {
	return ts.cacheReadTokens
}

// TotalTokens returns the total number of tokens
func (ts TokenStats) TotalTokens() int {
	return ts.inputTokens + ts.outputTokens + ts.cacheCreationTokens + ts.cacheReadTokens
}

// IsEmpty checks if all token counts are zero
func (ts TokenStats) IsEmpty() bool {
	return ts.TotalTokens() == 0
}

// Equals checks if two TokenStats are equal
func (ts TokenStats) Equals(other TokenStats) bool {
	return ts.inputTokens == other.inputTokens &&
		ts.outputTokens == other.outputTokens &&
		ts.cacheCreationTokens == other.cacheCreationTokens &&
		ts.cacheReadTokens == other.cacheReadTokens
}

// GetTokenPercentages calculates the percentage of each token type
func (ts TokenStats) GetTokenPercentages() TokenPercentages {
	total := float64(ts.TotalTokens())
	if total == 0 {
		return TokenPercentages{}
	}

	return TokenPercentages{
		Input:         (float64(ts.inputTokens) / total) * 100,
		Output:        (float64(ts.outputTokens) / total) * 100,
		CacheCreation: (float64(ts.cacheCreationTokens) / total) * 100,
		CacheRead:     (float64(ts.cacheReadTokens) / total) * 100,
	}
}

// TokenPercentages represents the percentage distribution of token types
type TokenPercentages struct {
	Input         float64
	Output        float64
	CacheCreation float64
	CacheRead     float64
}

// MergeMultipleTokenStats merges multiple TokenStats into one
func MergeMultipleTokenStats(stats []TokenStats) TokenStats {
	if len(stats) == 0 {
		return NewEmptyTokenStats()
	}

	totalInput := 0
	totalOutput := 0
	totalCacheCreation := 0
	totalCacheRead := 0

	for _, stat := range stats {
		totalInput += stat.inputTokens
		totalOutput += stat.outputTokens
		totalCacheCreation += stat.cacheCreationTokens
		totalCacheRead += stat.cacheReadTokens
	}

	return NewTokenStats(totalInput, totalOutput, totalCacheCreation, totalCacheRead)
}
