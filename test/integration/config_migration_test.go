package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ca-srg/tosage/infrastructure/config"
	"github.com/ca-srg/tosage/infrastructure/di"
	"github.com/ca-srg/tosage/infrastructure/repository"
)

func TestConfigMigration_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	tests := []struct {
		name           string
		initialConfig  map[string]interface{}
		expectedConfig *config.AppConfig
		expectError    bool
	}{
		{
			name: "migrate legacy config file",
			initialConfig: map[string]interface{}{
				"claude_path": "/test/claude",
				"prometheus": map[string]interface{}{
					"remote_write_url": "http://prometheus:9090/api/v1/write",
					"username":         "olduser",
					"password":         "oldpass",
					"host_label":       "testhost",
					"interval_seconds": 600,
					"timeout_seconds":  30,
				},
				"daemon": map[string]interface{}{
					"enabled": true,
				},
			},
			expectedConfig: &config.AppConfig{
				Version:    1,
				ClaudePath: "/test/claude",
				Prometheus: &config.PrometheusConfig{
					RemoteWriteURL:      "http://prometheus:9090/api/v1/write",
					RemoteWriteUsername: "olduser",
					RemoteWritePassword: "oldpass",
					HostLabel:           "testhost",
					IntervalSec:         600,
					TimeoutSec:          30,
				},
			},
			expectError: false,
		},
		{
			name: "already migrated config",
			initialConfig: map[string]interface{}{
				"version":     1,
				"claude_path": "/test/claude",
				"prometheus": map[string]interface{}{
					"remote_write_url":      "http://prometheus:9090/api/v1/write",
					"remote_write_username": "newuser",
					"remote_write_password": "newpass",
					"url":                   "http://prometheus:9090",
					"username":              "queryuser",
					"password":              "querypass",
				},
			},
			expectedConfig: &config.AppConfig{
				Version:    1,
				ClaudePath: "/test/claude",
				Prometheus: &config.PrometheusConfig{
					RemoteWriteURL:      "http://prometheus:9090/api/v1/write",
					RemoteWriteUsername: "newuser",
					RemoteWritePassword: "newpass",
					URL:                 "http://prometheus:9090",
					Username:            "queryuser",
					Password:            "querypass",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for test
			tempDir, err := os.MkdirTemp("", "tosage-migration-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() {
				if err := os.RemoveAll(tempDir); err != nil {
					t.Logf("Failed to remove temp dir: %v", err)
				}
			}()

			// Write initial config file
			configPath := filepath.Join(tempDir, "config.json")
			configData, err := json.MarshalIndent(tt.initialConfig, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal initial config: %v", err)
			}
			if err := os.WriteFile(configPath, configData, 0600); err != nil {
				t.Fatalf("Failed to write initial config: %v", err)
			}
			// Create custom config repository
			configRepo := repository.NewJSONConfigRepository()
			repo := configRepo.(*repository.JSONConfigRepository)
			repo.SetConfigDir(tempDir)
			repo.SetConfigFile(configPath)

			// Create container with custom config repository
			builder := di.NewContainerBuilder().
				WithConfigRepository(configRepo)

			container, err := builder.Build()
			if err != nil {
				if tt.expectError {
					return // Expected error
				}
				t.Fatalf("Failed to build container: %v", err)
			}

			// Get the loaded config
			configService := container.GetConfigService()
			loadedConfig := configService.GetConfig()

			// Verify migration happened correctly
			if loadedConfig.Version != tt.expectedConfig.Version {
				t.Errorf("Version mismatch: got %d, want %d",
					loadedConfig.Version, tt.expectedConfig.Version)
			}

			// Wait a moment to ensure file write is complete
			time.Sleep(100 * time.Millisecond)

			// Verify the config was saved with migration
			savedConfig, err := configRepo.Load()
			if err != nil {
				t.Fatalf("Failed to load saved config: %v", err)
			}

			if savedConfig.Version != tt.expectedConfig.Version {
				t.Errorf("Saved config version mismatch: got %d, want %d",
					savedConfig.Version, tt.expectedConfig.Version)
			}

			// Check Prometheus fields if present
			if tt.expectedConfig.Prometheus != nil && savedConfig.Prometheus != nil {
				verifyPrometheusConfig(t, savedConfig.Prometheus, tt.expectedConfig.Prometheus)
			}
		})
	}
}

func TestConfigMigration_WithEnvironmentVariables(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "tosage-migration-env-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create legacy config file
	configPath := filepath.Join(tempDir, "config.json")
	initialConfig := map[string]interface{}{
		"prometheus": map[string]interface{}{
			"remote_write_url": "http://prometheus:9090/api/v1/write",
			"username":         "fileuser",
			"password":         "filepass",
		},
	}
	configData, err := json.MarshalIndent(initialConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal initial config: %v", err)
	}
	if err := os.WriteFile(configPath, configData, 0600); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	// Set environment variables that should override file values
	if err := os.Setenv("TOSAGE_PROMETHEUS_REMOTE_WRITE_USERNAME", "envuser"); err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
	}
	if err := os.Setenv("TOSAGE_PROMETHEUS_REMOTE_WRITE_PASSWORD", "envpass"); err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("TOSAGE_PROMETHEUS_REMOTE_WRITE_USERNAME"); err != nil {
			t.Logf("Failed to unset environment variable: %v", err)
		}
		if err := os.Unsetenv("TOSAGE_PROMETHEUS_REMOTE_WRITE_PASSWORD"); err != nil {
			t.Logf("Failed to unset environment variable: %v", err)
		}
	}()

	// Create custom config repository
	configRepo := repository.NewJSONConfigRepository()
	repo := configRepo.(*repository.JSONConfigRepository)
	repo.SetConfigDir(tempDir)
	repo.SetConfigFile(configPath)

	// Create container
	builder := di.NewContainerBuilder().
		WithConfigRepository(configRepo)

	container, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build container: %v", err)
	}

	// Get the loaded config
	configService := container.GetConfigService()
	loadedConfig := configService.GetConfig()

	// Verify migration happened
	if loadedConfig.Version != 1 {
		t.Errorf("Expected version 1 after migration, got %d", loadedConfig.Version)
	}

	// Verify environment variables took precedence
	if loadedConfig.Prometheus.RemoteWriteUsername != "envuser" {
		t.Errorf("Expected RemoteWriteUsername from env 'envuser', got %s",
			loadedConfig.Prometheus.RemoteWriteUsername)
	}
	if loadedConfig.Prometheus.RemoteWritePassword != "envpass" {
		t.Errorf("Expected RemoteWritePassword from env 'envpass', got %s",
			loadedConfig.Prometheus.RemoteWritePassword)
	}

	// Verify source tracking
	sources := loadedConfig.ConfigSources
	if sources["Prometheus.RemoteWriteUsername"] != config.SourceEnvironment {
		t.Error("RemoteWriteUsername source should be environment")
	}
	if sources["Prometheus.RemoteWritePassword"] != config.SourceEnvironment {
		t.Error("RemoteWritePassword source should be environment")
	}
}

func TestConfigMigration_InvalidConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "tosage-migration-invalid-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create invalid legacy config (RemoteWriteURL without credentials)
	configPath := filepath.Join(tempDir, "config.json")
	initialConfig := map[string]interface{}{
		"prometheus": map[string]interface{}{
			"remote_write_url": "http://prometheus:9090/api/v1/write",
			// Missing username and password - should fail validation after migration
		},
	}
	configData, err := json.MarshalIndent(initialConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal initial config: %v", err)
	}
	if err := os.WriteFile(configPath, configData, 0600); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	// Create custom config repository
	configRepo := repository.NewJSONConfigRepository()
	repo := configRepo.(*repository.JSONConfigRepository)
	repo.SetConfigDir(tempDir)
	repo.SetConfigFile(configPath)

	// Create container
	builder := di.NewContainerBuilder().
		WithConfigRepository(configRepo)

	container, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build container: %v", err)
	}

	// Even with invalid config, container should build with fallback
	configService := container.GetConfigService()
	loadedConfig := configService.GetConfig()

	// The config should fall back to defaults due to validation failure
	if loadedConfig == nil {
		t.Error("Expected fallback config, got nil")
	}
}

// Helper function to verify Prometheus configuration
func verifyPrometheusConfig(t *testing.T, actual, expected *config.PrometheusConfig) {
	if actual.RemoteWriteURL != expected.RemoteWriteURL {
		t.Errorf("RemoteWriteURL mismatch: got %s, want %s",
			actual.RemoteWriteURL, expected.RemoteWriteURL)
	}
	if actual.RemoteWriteUsername != expected.RemoteWriteUsername {
		t.Errorf("RemoteWriteUsername mismatch: got %s, want %s",
			actual.RemoteWriteUsername, expected.RemoteWriteUsername)
	}
	if actual.RemoteWritePassword != expected.RemoteWritePassword {
		t.Errorf("RemoteWritePassword mismatch: got %s, want %s",
			actual.RemoteWritePassword, expected.RemoteWritePassword)
	}
	if actual.URL != expected.URL {
		t.Errorf("URL mismatch: got %s, want %s",
			actual.URL, expected.URL)
	}
	if actual.Username != expected.Username {
		t.Errorf("Username mismatch: got %s, want %s",
			actual.Username, expected.Username)
	}
	if actual.Password != expected.Password {
		t.Errorf("Password mismatch: got %s, want %s",
			actual.Password, expected.Password)
	}
	if actual.HostLabel != expected.HostLabel {
		t.Errorf("HostLabel mismatch: got %s, want %s",
			actual.HostLabel, expected.HostLabel)
	}
	if actual.IntervalSec != expected.IntervalSec {
		t.Errorf("IntervalSec mismatch: got %d, want %d",
			actual.IntervalSec, expected.IntervalSec)
	}
	if actual.TimeoutSec != expected.TimeoutSec {
		t.Errorf("TimeoutSec mismatch: got %d, want %d",
			actual.TimeoutSec, expected.TimeoutSec)
	}
}
