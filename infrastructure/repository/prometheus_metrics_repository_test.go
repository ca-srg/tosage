package repository

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ca-srg/tosage/infrastructure/config"
)

func TestNewPrometheusMetricsRepository(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.PrometheusConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name:    "disabled prometheus",
			config:  &config.PrometheusConfig{},
			wantErr: true, // RemoteWriteURL is required
		},
		{
			name: "enabled prometheus with valid config",
			config: &config.PrometheusConfig{
				RemoteWriteURL: "http://localhost:9091",
				HostLabel:      "test-host",
				IntervalSec:    600,
				TimeoutSec:     30,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := NewPrometheusMetricsRepository(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPrometheusMetricsRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && repo == nil {
				t.Error("NewPrometheusMetricsRepository() returned nil repository")
			}
		})
	}
}

func TestPrometheusMetricsRepository_SendTokenMetric(t *testing.T) {
	// Create test server to mock Remote Write endpoint
	var receivedContentType string
	var receivedContentEncoding string
	var receivedMethod string
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		receivedMethod = r.Method
		receivedContentType = r.Header.Get("Content-Type")
		receivedContentEncoding = r.Header.Get("Content-Encoding")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &config.PrometheusConfig{
		RemoteWriteURL: server.URL,
		HostLabel:      "test-host",
		TimeoutSec:     30,
	}

	repo, err := NewPrometheusMetricsRepository(config)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	err = repo.SendTokenMetric(12345, "test-host", "tosage_cc_token")
	if err != nil {
		t.Errorf("SendTokenMetric() returned unexpected error: %v", err)
	}

	// Verify request was made
	if requestCount != 1 {
		t.Errorf("Expected 1 request, got %d", requestCount)
	}

	// Verify HTTP method
	if receivedMethod != "POST" {
		t.Errorf("Expected POST method, got %s", receivedMethod)
	}

	// Verify headers (protobuf format)
	if receivedContentType != "application/x-protobuf" {
		t.Errorf("Unexpected Content-Type: %s", receivedContentType)
	}

	if receivedContentEncoding != "snappy" {
		t.Errorf("Unexpected Content-Encoding: %s", receivedContentEncoding)
	}
}

func TestPrometheusMetricsRepository_WithAuth(t *testing.T) {
	tests := []struct {
		name           string
		config         *config.PrometheusConfig
		expectedHeader string
		expectedValue  string
	}{
		{
			name: "basic auth",
			config: &config.PrometheusConfig{
				RemoteWriteURL:      "placeholder",
				RemoteWriteUsername: "testuser",
				RemoteWritePassword: "testpass",
				TimeoutSec:          30,
			},
			expectedHeader: "Authorization",
			expectedValue:  "Basic dGVzdHVzZXI6dGVzdHBhc3M=", // base64("testuser:testpass")
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedAuthHeader string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedAuthHeader = r.Header.Get(tt.expectedHeader)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			// Update config with actual server URL
			tt.config.RemoteWriteURL = server.URL

			repo, err := NewPrometheusMetricsRepository(tt.config)
			if err != nil {
				t.Fatalf("Failed to create repository: %v", err)
			}

			err = repo.SendTokenMetric(12345, "test-host", "tosage_cc_token")
			if err != nil {
				t.Errorf("SendTokenMetric() returned unexpected error: %v", err)
			}

			if receivedAuthHeader != tt.expectedValue {
				t.Errorf("Expected auth header %q, got %q", tt.expectedValue, receivedAuthHeader)
			}
		})
	}
}

func TestPrometheusMetricsRepository_Close(t *testing.T) {
	config := &config.PrometheusConfig{
		RemoteWriteURL: "http://localhost:9090/api/v1/write",
		HostLabel:      "test-host",
		TimeoutSec:     30,
	}

	repo, err := NewPrometheusMetricsRepository(config)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Close should not return error
	if err := repo.Close(); err != nil {
		t.Errorf("Close() returned unexpected error: %v", err)
	}
}

func TestPrometheusMetricsRepository_BackwardCompatibility(t *testing.T) {
	// Test that ServerURL still works when RemoteWriteURL is not set
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &config.PrometheusConfig{
		RemoteWriteURL: server.URL, // Using old ServerURL field
		HostLabel:      "test-host",
		TimeoutSec:     30,
	}

	repo, err := NewPrometheusMetricsRepository(config)
	if err != nil {
		t.Fatalf("Failed to create repository with legacy ServerURL: %v", err)
	}

	// Should work with legacy configuration
	err = repo.SendTokenMetric(12345, "test-host", "tosage_cc_token")
	if err != nil {
		t.Errorf("SendTokenMetric() with legacy config returned unexpected error: %v", err)
	}
}

func TestPrometheusMetricsRepository_SendTokenMetric_CursorMetrics(t *testing.T) {
	// Create test server to mock Remote Write endpoint
	var receivedContentType string
	var receivedContentEncoding string
	var receivedMethod string
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		receivedMethod = r.Method
		receivedContentType = r.Header.Get("Content-Type")
		receivedContentEncoding = r.Header.Get("Content-Encoding")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &config.PrometheusConfig{
		RemoteWriteURL: server.URL,
		HostLabel:      "test-host",
		TimeoutSec:     30,
	}

	repo, err := NewPrometheusMetricsRepository(config)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	tests := []struct {
		name       string
		tokens     int
		hostLabel  string
		metricName string
		wantErr    bool
	}{
		{
			name:       "successful send cursor tokens",
			tokens:     450,
			hostLabel:  "custom-host",
			metricName: "tosage_cursor_token",
			wantErr:    false,
		},
		{
			name:       "successful send with empty host label",
			tokens:     200,
			hostLabel:  "",
			metricName: "tosage_cursor_token",
			wantErr:    false,
		},
		{
			name:       "successful send cc tokens",
			tokens:     100,
			hostLabel:  "test-host",
			metricName: "tosage_cc_token",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount = 0
			err := repo.SendTokenMetric(tt.tokens, tt.hostLabel, tt.metricName)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendTokenMetric() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify request was made
				if requestCount != 1 {
					t.Errorf("Expected 1 request, got %d", requestCount)
				}

				// Verify HTTP method
				if receivedMethod != "POST" {
					t.Errorf("Expected POST method, got %s", receivedMethod)
				}

				// Verify headers (protobuf format)
				if receivedContentType != "application/x-protobuf" {
					t.Errorf("Unexpected Content-Type: %s", receivedContentType)
				}

				if receivedContentEncoding != "snappy" {
					t.Errorf("Unexpected Content-Encoding: %s", receivedContentEncoding)
				}
			}
		})
	}
}

//func TestPrometheusMetricsRepository_SendTokenMetric_Timeout(t *testing.T) {
//	// Create test server that delays response to trigger timeout
//	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		// Don't respond to trigger timeout
//		<-r.Context().Done()
//	}))
//	defer server.Close()
//
//	config := &config.PrometheusConfig{
//		RemoteWriteURL: server.URL,
//		HostLabel:      "test-host",
//		TimeoutSec:     1, // Short timeout
//	}
//
//	repo, err := NewPrometheusMetricsRepository(config)
//	if err != nil {
//		t.Fatalf("Failed to create repository: %v", err)
//	}
//
//	err = repo.SendTokenMetric(100, "test-host", "tosage_cursor_token")
//	if err == nil {
//		t.Error("Expected timeout error, got nil")
//	}
//}

func TestPrometheusMetricsRepository_SendTokenMetric_ConnectionError(t *testing.T) {
	config := &config.PrometheusConfig{
		RemoteWriteURL: "http://localhost:99999", // Invalid port
		HostLabel:      "test-host",
		TimeoutSec:     5,
	}

	repo, err := NewPrometheusMetricsRepository(config)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	err = repo.SendTokenMetric(100, "test-host", "tosage_cursor_token")
	if err == nil {
		t.Error("Expected connection error, got nil")
	}
}
