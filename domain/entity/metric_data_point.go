package entity

import (
	"time"
)

// MetricDataPoint represents a single metric data point from Prometheus
type MetricDataPoint struct {
	Timestamp        time.Time
	ClaudeCodeTokens int
	CursorTokens     int
	TotalTokens      int
	Host             string
	Timezone         string
	TimezoneOffset   string
	DetectionMethod  string
}

// NewMetricDataPoint creates a new metric data point
func NewMetricDataPoint(
	timestamp time.Time,
	claudeCodeTokens int,
	cursorTokens int,
	host string,
) *MetricDataPoint {
	return &MetricDataPoint{
		Timestamp:        timestamp,
		ClaudeCodeTokens: claudeCodeTokens,
		CursorTokens:     cursorTokens,
		TotalTokens:      claudeCodeTokens + cursorTokens,
		Host:             host,
	}
}

// WithTimezone sets timezone information
func (m *MetricDataPoint) WithTimezone(timezone, offset, detectionMethod string) *MetricDataPoint {
	m.Timezone = timezone
	m.TimezoneOffset = offset
	m.DetectionMethod = detectionMethod
	return m
}
