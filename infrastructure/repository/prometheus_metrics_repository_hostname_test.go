package repository

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ca-srg/tosage/infrastructure/config"
)

func TestPrometheusMetricsRepository_HostnameDefault(t *testing.T) {
	// Get the expected hostname
	expectedHostname, err := os.Hostname()
	if err != nil {
		expectedHostname = "unknown"
	}

	tests := []struct {
		name             string
		hostLabel        string
		expectedHostname string
	}{
		{
			name:             "empty host label uses hostname",
			hostLabel:        "",
			expectedHostname: expectedHostname,
		},
		{
			name:             "specified host label is used",
			hostLabel:        "custom-host",
			expectedHostname: "custom-host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server to capture metrics
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// For Remote Write, we just need to verify the request was made
				// The actual content is Snappy-compressed and would need decompression
				// to verify, which is covered in other tests
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			config := &config.PrometheusConfig{
				RemoteWriteURL: server.URL,
				HostLabel:      tt.hostLabel,
				TimeoutSec:     30,
			}

			repo, err := NewPrometheusMetricsRepository(config)
			if err != nil {
				t.Fatalf("Failed to create repository: %v", err)
			}

			// Send a metric
			err = repo.SendTokenMetric(12345, "ignored-param", "tosage_cc_token")
			if err != nil {
				t.Fatalf("Failed to send metric: %v", err)
			}

			// Verify that the repository was created with the expected hostname
			// The actual hostname verification is internal to the repository
			// and is properly tested in integration tests
			promRepo := repo.(*PrometheusMetricsRepository)
			if promRepo.hostLabel != tt.expectedHostname {
				t.Errorf("Expected hostname '%s', got '%s'", tt.expectedHostname, promRepo.hostLabel)
			}
		})
	}
}

func TestPrometheusMetricsRepository_HostnameErrorHandling(t *testing.T) {
	// This test verifies that when os.Hostname() fails (which is rare),
	// the system falls back to "unknown"
	// Since we can't easily mock os.Hostname(), we'll just verify
	// that the repository can be created with an empty host label

	config := &config.PrometheusConfig{
		RemoteWriteURL: "http://localhost:9091",
		HostLabel:      "",
		TimeoutSec:     30,
	}

	repo, err := NewPrometheusMetricsRepository(config)
	if err != nil {
		t.Fatalf("Failed to create repository with empty host label: %v", err)
	}

	// Verify it's not nil and is functional
	if repo == nil {
		t.Error("Repository should not be nil")
	}
}
