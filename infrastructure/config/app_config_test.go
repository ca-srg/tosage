package config

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVertexAIConfig_ServiceAccountKey(t *testing.T) {
	// Test that ServiceAccountKey field is properly initialized
	config := DefaultConfig()
	assert.NotNil(t, config.VertexAI)
	assert.Equal(t, "", config.VertexAI.ServiceAccountKey)
	assert.Equal(t, "", config.VertexAI.ServiceAccountKeyPath)
}

func TestVertexAIConfig_EnvironmentVariable(t *testing.T) {
	// Save original env var
	originalKey := os.Getenv("TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY")
	originalPath := os.Getenv("TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY_PATH")
	defer func() {
		_ = os.Setenv("TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY", originalKey)
		_ = os.Setenv("TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY_PATH", originalPath)
	}()

	// Test service account key from environment (base64 encoded)
	testKey := `{"type":"service_account","project_id":"test-project"}`
	base64Key := base64.StdEncoding.EncodeToString([]byte(testKey))
	_ = os.Setenv("TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY", base64Key)
	_ = os.Setenv("TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY_PATH", "/path/to/key.json")

	config := DefaultConfig()
	err := config.LoadFromEnv()
	require.NoError(t, err)

	assert.Equal(t, testKey, config.VertexAI.ServiceAccountKey)
	assert.Equal(t, "/path/to/key.json", config.VertexAI.ServiceAccountKeyPath)
	assert.Equal(t, SourceEnvironment, config.ConfigSources["VertexAI.ServiceAccountKey"])
	assert.Equal(t, SourceEnvironment, config.ConfigSources["VertexAI.ServiceAccountKeyPath"])
}

func TestVertexAIConfig_Validation(t *testing.T) {
	tests := []struct {
		name              string
		serviceAccountKey string
		enabled           bool
		projectID         string
		wantErr           bool
		errMsg            string
	}{
		{
			name:              "valid empty key",
			serviceAccountKey: "",
			enabled:           true,
			projectID:         "test-project",
			wantErr:           false,
		},
		{
			name: "valid service account key",
			serviceAccountKey: `{
				"type": "service_account",
				"project_id": "test-project",
				"private_key_id": "key-id",
				"private_key": "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
				"client_email": "test@test-project.iam.gserviceaccount.com"
			}`,
			enabled:   true,
			projectID: "test-project",
			wantErr:   false,
		},
		{
			name:              "invalid JSON",
			serviceAccountKey: "not-json",
			enabled:           true,
			projectID:         "test-project",
			wantErr:           true,
			errMsg:            "invalid service account key JSON",
		},
		{
			name: "missing required field",
			serviceAccountKey: `{
				"type": "service_account",
				"project_id": "test-project"
			}`,
			enabled:   true,
			projectID: "test-project",
			wantErr:   true,
			errMsg:    "service account key missing required field",
		},
		{
			name: "wrong type",
			serviceAccountKey: `{
				"type": "user",
				"project_id": "test-project",
				"private_key_id": "key-id",
				"private_key": "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
				"client_email": "test@test-project.iam.gserviceaccount.com"
			}`,
			enabled:   true,
			projectID: "test-project",
			wantErr:   true,
			errMsg:    "service account key must have type 'service_account'",
		},
		{
			name:              "disabled vertex ai with invalid key",
			serviceAccountKey: "invalid-json", // Service account key is still validated even when disabled
			enabled:           false,
			projectID:         "test-project",
			wantErr:           true,
			errMsg:            "invalid service account key JSON",
		},
		{
			name:              "disabled vertex ai without key",
			serviceAccountKey: "", // No validation when no key is provided
			enabled:           false,
			projectID:         "",
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &AppConfig{
				VertexAI: &VertexAIConfig{
					Enabled:               tt.enabled,
					ProjectID:             tt.projectID,
					ServiceAccountKey:     tt.serviceAccountKey,
					CollectionIntervalSec: 600,
				},
			}

			err := config.validateVertexAI()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVertexAIConfig_EnvironmentTracking(t *testing.T) {
	// Save original env vars
	originalVars := map[string]string{
		"TOSAGE_VERTEX_AI_ENABLED":                     os.Getenv("TOSAGE_VERTEX_AI_ENABLED"),
		"TOSAGE_VERTEX_AI_PROJECT_ID":                  os.Getenv("TOSAGE_VERTEX_AI_PROJECT_ID"),
		"TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY_PATH":    os.Getenv("TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY_PATH"),
		"TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY":         os.Getenv("TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY"),
		"TOSAGE_VERTEX_AI_COLLECTION_INTERVAL_SECONDS": os.Getenv("TOSAGE_VERTEX_AI_COLLECTION_INTERVAL_SECONDS"),
	}
	defer func() {
		for k, v := range originalVars {
			_ = os.Setenv(k, v)
		}
	}()

	// Set all environment variables
	_ = os.Setenv("TOSAGE_VERTEX_AI_ENABLED", "true")
	_ = os.Setenv("TOSAGE_VERTEX_AI_PROJECT_ID", "my-project")
	_ = os.Setenv("TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY_PATH", "/path/to/key.json")
	// Base64 encode the service account key
	encodedKey := base64.StdEncoding.EncodeToString([]byte(`{"type":"service_account"}`))
	_ = os.Setenv("TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY", encodedKey)
	_ = os.Setenv("TOSAGE_VERTEX_AI_COLLECTION_INTERVAL_SECONDS", "900")

	config := DefaultConfig()
	err := config.LoadFromEnv()
	require.NoError(t, err)

	// Check that all fields were loaded from environment
	assert.True(t, config.VertexAI.Enabled)
	assert.Equal(t, "my-project", config.VertexAI.ProjectID)
	assert.Equal(t, "/path/to/key.json", config.VertexAI.ServiceAccountKeyPath)
	assert.Equal(t, `{"type":"service_account"}`, config.VertexAI.ServiceAccountKey)
	assert.Equal(t, 900, config.VertexAI.CollectionIntervalSec)

	// Check that sources were tracked correctly
	assert.Equal(t, SourceEnvironment, config.ConfigSources["VertexAI.Enabled"])
	assert.Equal(t, SourceEnvironment, config.ConfigSources["VertexAI.ProjectID"])
	assert.Equal(t, SourceEnvironment, config.ConfigSources["VertexAI.ServiceAccountKeyPath"])
	assert.Equal(t, SourceEnvironment, config.ConfigSources["VertexAI.ServiceAccountKey"])
	assert.Equal(t, SourceEnvironment, config.ConfigSources["VertexAI.CollectionIntervalSec"])
}

func TestVertexAIConfig_JSONMerge(t *testing.T) {
	baseConfig := DefaultConfig()
	baseConfig.MarkDefaults()

	jsonConfig := &AppConfig{
		VertexAI: &VertexAIConfig{
			Enabled:               true,
			ProjectID:             "json-project",
			ServiceAccountKeyPath: "/json/path/key.json",
			ServiceAccountKey:     `{"type":"service_account","project_id":"json-project"}`,
			CollectionIntervalSec: 1200,
		},
	}

	baseConfig.MergeJSONConfig(jsonConfig)

	// Check that values were merged
	assert.True(t, baseConfig.VertexAI.Enabled)
	assert.Equal(t, "json-project", baseConfig.VertexAI.ProjectID)
	assert.Equal(t, "/json/path/key.json", baseConfig.VertexAI.ServiceAccountKeyPath)
	assert.Equal(t, `{"type":"service_account","project_id":"json-project"}`, baseConfig.VertexAI.ServiceAccountKey)
	assert.Equal(t, 1200, baseConfig.VertexAI.CollectionIntervalSec)

	// Check that sources were updated
	assert.Equal(t, SourceJSONFile, baseConfig.ConfigSources["VertexAI.Enabled"])
	assert.Equal(t, SourceJSONFile, baseConfig.ConfigSources["VertexAI.ProjectID"])
	assert.Equal(t, SourceJSONFile, baseConfig.ConfigSources["VertexAI.ServiceAccountKeyPath"])
	assert.Equal(t, SourceJSONFile, baseConfig.ConfigSources["VertexAI.ServiceAccountKey"])
	assert.Equal(t, SourceJSONFile, baseConfig.ConfigSources["VertexAI.CollectionIntervalSec"])
}

func TestVertexAIConfig_BackwardCompatibility(t *testing.T) {
	// Test that old configs without ServiceAccountKey still work
	oldConfigJSON := `{
		"version": 1,
		"vertex_ai": {
			"enabled": true,
			"project_id": "old-project",
			"locations": ["us-central1"],
			"service_account_key_path": "/old/path/key.json",
			"collection_interval_seconds": 600
		}
	}`

	var config AppConfig
	err := json.Unmarshal([]byte(oldConfigJSON), &config)
	require.NoError(t, err)

	assert.True(t, config.VertexAI.Enabled)
	assert.Equal(t, "old-project", config.VertexAI.ProjectID)
	assert.Equal(t, "/old/path/key.json", config.VertexAI.ServiceAccountKeyPath)
	assert.Equal(t, "", config.VertexAI.ServiceAccountKey) // Should be empty for old configs
}

func TestVertexAIConfig_CompleteValidation(t *testing.T) {
	// Test complete validation flow
	validKey := `{
		"type": "service_account",
		"project_id": "my-test-project",
		"private_key_id": "abc123def456",
		"private_key": "-----BEGIN RSA PRIVATE KEY-----\nMIIBOwIBAAJBAOLr5vIzVJZQaudJJcVh8fFUvBT9gkH09jtpfwwhhp1V3k5rqeC8\n3zYLRXJL5Q6p3iqVrWtAKGrE4Y6ggDuMnEMCAwEAAQJBALu0tPVFGzaJS6L/AT1g\n3NrBmXNmGj6AqPfJY3tReWe9E04qmDz2HLMssO2fNwV5bxLLDd5iwTKlpE5vcr5E\nu5kCIQD1b5M+BvKLPhKBGc7f8h2oXETnogU+w8R5P2oLP1dG1QIhAOzfPnRQQypL\nK0OccJiXUr0i5DeVTN8TGpWa6XimFk73AiAbLuNwKUhrkwWh4ThaMc0w7kR1qZ3X\nvZrHBXyWLddd7QIgNa/+lVGGO2F5pXpdNykJZeeqc6qv7X8qOEIxt5BnggMCIDvG\n7y1Mr+hFPepFOi1qzHkhjnnFh8vMMKj8MgMt+OKM\n-----END RSA PRIVATE KEY-----",
		"client_email": "my-service-account@my-test-project.iam.gserviceaccount.com",
		"client_id": "1234567890",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "https://oauth2.googleapis.com/token"
	}`

	config := &AppConfig{
		VertexAI: &VertexAIConfig{
			Enabled:               true,
			ProjectID:             "my-test-project",
			ServiceAccountKeyPath: "",
			ServiceAccountKey:     validKey,
			CollectionIntervalSec: 600,
		},
		Prometheus: &PrometheusConfig{
			RemoteWriteURL:      "https://prometheus.example.com/write",
			RemoteWriteUsername: "user",
			RemoteWritePassword: "pass",
			IntervalSec:         600,
			TimeoutSec:          30,
		},
	}

	err := config.Validate()
	assert.NoError(t, err)
}
