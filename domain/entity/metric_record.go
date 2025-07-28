package entity

import (
	"time"
)

// MetricRecord represents a unified structure for all metric data to be exported
type MetricRecord struct {
	Timestamp time.Time
	Source    string // claude_code, cursor, bedrock, vertex_ai
	Project   string
	Value     float64
	Unit      string // tokens, requests, etc.
	Metadata  map[string]string
}

// NewMetricRecord creates a new MetricRecord
func NewMetricRecord(timestamp time.Time, source, project string, value float64, unit string) *MetricRecord {
	return &MetricRecord{
		Timestamp: timestamp,
		Source:    source,
		Project:   project,
		Value:     value,
		Unit:      unit,
		Metadata:  make(map[string]string),
	}
}

// AddMetadata adds metadata to the record
func (m *MetricRecord) AddMetadata(key, value string) {
	if m.Metadata == nil {
		m.Metadata = make(map[string]string)
	}
	m.Metadata[key] = value
}

// GetMetadata retrieves metadata value
func (m *MetricRecord) GetMetadata(key string) (string, bool) {
	if m.Metadata == nil {
		return "", false
	}
	value, exists := m.Metadata[key]
	return value, exists
}
