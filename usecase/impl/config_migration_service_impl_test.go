package impl

import (
	"testing"

	"github.com/ca-srg/tosage/infrastructure/config"
)

func TestConfigMigrationServiceImpl_NeedsMigration(t *testing.T) {
	mockLogger := &MockLogger{}
	service := NewConfigMigrationService(mockLogger)

	tests := []struct {
		name     string
		config   *config.AppConfig
		expected bool
	}{
		{
			name: "version 0 needs migration",
			config: &config.AppConfig{
				Version: 0,
			},
			expected: true,
		},
		{
			name:   "no version field needs migration",
			config: &config.AppConfig{
				// Version field is not set, defaults to 0
			},
			expected: true,
		},
		{
			name: "version 1 does not need migration",
			config: &config.AppConfig{
				Version: 1,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.NeedsMigration(tt.config)
			if result != tt.expected {
				t.Errorf("NeedsMigration() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfigMigrationServiceImpl_GetCurrentVersion(t *testing.T) {
	mockLogger := &MockLogger{}
	service := NewConfigMigrationService(mockLogger)

	currentVersion := service.GetCurrentVersion()
	if currentVersion != 1 {
		t.Errorf("GetCurrentVersion() = %v, want %v", currentVersion, 1)
	}
}

func TestConfigMigrationServiceImpl_Migrate(t *testing.T) {
	mockLogger := &MockLogger{}
	service := NewConfigMigrationService(mockLogger)

	tests := []struct {
		name           string
		inputConfig    *config.AppConfig
		expectError    bool
		expectedFields map[string]string
	}{
		{
			name: "migrate legacy prometheus config",
			inputConfig: &config.AppConfig{
				Version: 0,
				Prometheus: &config.PrometheusConfig{
					RemoteWriteURL: "http://localhost:9090",
					Username:       "legacyuser",
					Password:       "legacypass",
				},
			},
			expectError: false,
			expectedFields: map[string]string{
				"Version":             "1",
				"RemoteWriteURL":      "http://localhost:9090",
				"RemoteWriteUsername": "legacyuser",
				"RemoteWritePassword": "legacypass",
				"Username":            "",
				"Password":            "",
			},
		},
		{
			name: "already migrated config",
			inputConfig: &config.AppConfig{
				Version: 1,
				Prometheus: &config.PrometheusConfig{
					RemoteWriteURL:      "http://localhost:9090",
					RemoteWriteUsername: "newuser",
					RemoteWritePassword: "newpass",
				},
			},
			expectError: false,
			expectedFields: map[string]string{
				"Version":             "1",
				"RemoteWriteURL":      "http://localhost:9090",
				"RemoteWriteUsername": "newuser",
				"RemoteWritePassword": "newpass",
			},
		},
		{
			name: "config without prometheus section",
			inputConfig: &config.AppConfig{
				Version: 0,
			},
			expectError: false,
			expectedFields: map[string]string{
				"Version": "1",
			},
		},
		{
			name: "partial prometheus config",
			inputConfig: &config.AppConfig{
				Version: 0,
				Prometheus: &config.PrometheusConfig{
					RemoteWriteURL: "http://localhost:9090",
					Username:       "onlyuser",
					// Password is empty - this should fail validation
				},
			},
			expectError: true, // Changed to expect error due to missing password
			expectedFields: map[string]string{
				"Version":             "1",
				"RemoteWriteURL":      "http://localhost:9090",
				"RemoteWriteUsername": "onlyuser",
				"RemoteWritePassword": "",
				"Username":            "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.Migrate(tt.inputConfig)

			if tt.expectError && err == nil {
				t.Error("Migrate() expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Migrate() unexpected error: %v", err)
			}

			if !tt.expectError && result != nil {
				// Check version
				if v, ok := tt.expectedFields["Version"]; ok {
					if result.Version != parseVersion(v) {
						t.Errorf("Version = %v, want %v", result.Version, v)
					}
				}

				// Check Prometheus fields if present
				if result.Prometheus != nil {
					if v, ok := tt.expectedFields["RemoteWriteURL"]; ok {
						if result.Prometheus.RemoteWriteURL != v {
							t.Errorf("RemoteWriteURL = %v, want %v", result.Prometheus.RemoteWriteURL, v)
						}
					}
					if v, ok := tt.expectedFields["RemoteWriteUsername"]; ok {
						if result.Prometheus.RemoteWriteUsername != v {
							t.Errorf("RemoteWriteUsername = %v, want %v", result.Prometheus.RemoteWriteUsername, v)
						}
					}
					if v, ok := tt.expectedFields["RemoteWritePassword"]; ok {
						if result.Prometheus.RemoteWritePassword != v {
							t.Errorf("RemoteWritePassword = %v, want %v", result.Prometheus.RemoteWritePassword, v)
						}
					}
					if v, ok := tt.expectedFields["Username"]; ok {
						if result.Prometheus.Username != v {
							t.Errorf("Username = %v, want %v", result.Prometheus.Username, v)
						}
					}
					if v, ok := tt.expectedFields["Password"]; ok {
						if result.Prometheus.Password != v {
							t.Errorf("Password = %v, want %v", result.Prometheus.Password, v)
						}
					}
				}
			}
		})
	}
}

func TestConfigMigrationServiceImpl_Migrate_ValidationError(t *testing.T) {
	mockLogger := &MockLogger{}
	service := NewConfigMigrationService(mockLogger)

	// Test case where RemoteWriteURL is set but credentials are missing after migration
	inputConfig := &config.AppConfig{
		Version: 0,
		Prometheus: &config.PrometheusConfig{
			RemoteWriteURL: "http://localhost:9090",
			// No username/password - should fail validation
		},
	}

	result, err := service.Migrate(inputConfig)
	if err == nil {
		t.Error("Migrate() should have returned validation error")
	}
	if result != nil {
		t.Error("Migrate() should return nil on validation error")
	}
}

func TestConfigMigrationServiceImpl_CopyConfig(t *testing.T) {
	mockLogger := &MockLogger{}
	service := NewConfigMigrationService(mockLogger).(*ConfigMigrationServiceImpl)

	// Create a complex config to test deep copy
	original := &config.AppConfig{
		Version:    1,
		ClaudePath: "/path/to/claude",
		Prometheus: &config.PrometheusConfig{
			RemoteWriteURL:      "http://localhost:9090",
			RemoteWriteUsername: "user",
			RemoteWritePassword: "pass",
			HostLabel:           "testhost",
			IntervalSec:         600,
			TimeoutSec:          30,
		},
		Cursor: &config.CursorConfig{
			DatabasePath: "/path/to/db",
			APITimeout:   30,
			CacheTimeout: 300,
		},
		Bedrock: &config.BedrockConfig{
			Enabled:               true,
			Regions:               []string{"us-east-1", "us-west-2"},
			AWSProfile:            "default",
			AssumeRoleARN:         "arn:aws:iam::123456789012:role/test",
			CollectionIntervalSec: 900,
		},
		VertexAI: &config.VertexAIConfig{
			Enabled:               true,
			ProjectID:             "test-project",
			Locations:             []string{"us-central1"},
			ServiceAccountKeyPath: "/path/to/key.json",
			CollectionIntervalSec: 900,
		},
		Daemon: &config.DaemonConfig{
			Enabled:      true,
			StartAtLogin: true,
			HideFromDock: true,
			LogPath:      "/tmp/tosage.log",
			PidFile:      "/tmp/tosage.pid",
		},
		Logging: &config.LoggingConfig{
			Level: "info",
			Debug: false,
			Promtail: &config.PromtailConfig{
				URL:              "http://localhost:3100",
				Username:         "promuser",
				Password:         "prompass",
				BatchWaitSeconds: 1,
				BatchCapacity:    100,
				TimeoutSeconds:   5,
			},
		},
		CSVExport: &config.CSVExportConfig{
			DefaultOutputPath:  ".",
			DefaultStartDays:   30,
			DefaultMetricTypes: "claude_code,cursor",
			MaxExportDays:      365,
			TimeZone:           "Asia/Tokyo",
		},
		ConfigSources: config.ConfigSourceMap{
			"ClaudePath": config.SourceJSONFile,
			"Version":    config.SourceDefault,
		},
	}

	// Perform copy
	copied := service.copyConfig(original)

	// Verify it's a different instance
	if copied == original {
		t.Error("copyConfig() returned same instance, not a copy")
	}

	// Verify all fields are copied correctly
	if copied.Version != original.Version {
		t.Errorf("Version not copied correctly: got %v, want %v", copied.Version, original.Version)
	}
	if copied.ClaudePath != original.ClaudePath {
		t.Errorf("ClaudePath not copied correctly: got %v, want %v", copied.ClaudePath, original.ClaudePath)
	}

	// Verify nested objects are deep copied
	if copied.Prometheus == original.Prometheus {
		t.Error("Prometheus config is not deep copied")
	}
	if copied.Prometheus.RemoteWriteURL != original.Prometheus.RemoteWriteURL {
		t.Errorf("Prometheus.RemoteWriteURL not copied correctly")
	}

	// Verify slices are copied
	if &copied.Bedrock.Regions == &original.Bedrock.Regions {
		t.Error("Bedrock.Regions slice is not copied")
	}
	if len(copied.Bedrock.Regions) != len(original.Bedrock.Regions) {
		t.Error("Bedrock.Regions slice length mismatch")
	}

	// Verify ConfigSources map is copied
	if len(copied.ConfigSources) != len(original.ConfigSources) {
		t.Error("ConfigSources map not copied correctly")
	}
	for k, v := range original.ConfigSources {
		if copied.ConfigSources[k] != v {
			t.Errorf("ConfigSources[%s] not copied correctly", k)
		}
	}
}

// Helper function to parse version string to int
func parseVersion(v string) int {
	switch v {
	case "1":
		return 1
	default:
		return 0
	}
}
