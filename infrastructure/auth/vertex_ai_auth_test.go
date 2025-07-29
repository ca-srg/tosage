package auth

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateServiceAccountKey(t *testing.T) {
	tests := []struct {
		name    string
		key     *ServiceAccountKey
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid service account key",
			key: &ServiceAccountKey{
				Type:         "service_account",
				ProjectID:    "test-project",
				PrivateKeyID: "key-id",
				PrivateKey:   "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
				ClientEmail:  "test@test-project.iam.gserviceaccount.com",
			},
			wantErr: false,
		},
		{
			name: "invalid type",
			key: &ServiceAccountKey{
				Type:         "user",
				ProjectID:    "test-project",
				PrivateKeyID: "key-id",
				PrivateKey:   "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
				ClientEmail:  "test@test-project.iam.gserviceaccount.com",
			},
			wantErr: true,
			errMsg:  "invalid service account type: user (expected 'service_account')",
		},
		{
			name: "missing project_id",
			key: &ServiceAccountKey{
				Type:         "service_account",
				ProjectID:    "",
				PrivateKeyID: "key-id",
				PrivateKey:   "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
				ClientEmail:  "test@test-project.iam.gserviceaccount.com",
			},
			wantErr: true,
			errMsg:  "service account key missing required field: project_id",
		},
		{
			name: "missing private_key_id",
			key: &ServiceAccountKey{
				Type:         "service_account",
				ProjectID:    "test-project",
				PrivateKeyID: "",
				PrivateKey:   "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
				ClientEmail:  "test@test-project.iam.gserviceaccount.com",
			},
			wantErr: true,
			errMsg:  "service account key missing required field: private_key_id",
		},
		{
			name: "missing private_key",
			key: &ServiceAccountKey{
				Type:         "service_account",
				ProjectID:    "test-project",
				PrivateKeyID: "key-id",
				PrivateKey:   "",
				ClientEmail:  "test@test-project.iam.gserviceaccount.com",
			},
			wantErr: true,
			errMsg:  "service account key missing required field: private_key",
		},
		{
			name: "missing client_email",
			key: &ServiceAccountKey{
				Type:         "service_account",
				ProjectID:    "test-project",
				PrivateKeyID: "key-id",
				PrivateKey:   "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
				ClientEmail:  "",
			},
			wantErr: true,
			errMsg:  "service account key missing required field: client_email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateServiceAccountKey(tt.key)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCredentials(t *testing.T) {
	tests := []struct {
		name              string
		serviceAccountKey string
		wantErr           bool
		errMsg            string
	}{
		{
			name: "valid JSON key",
			serviceAccountKey: `{
				"type": "service_account",
				"project_id": "test-project",
				"private_key_id": "key-id",
				"private_key": "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
				"client_email": "test@test-project.iam.gserviceaccount.com"
			}`,
			wantErr: false,
		},
		{
			name:              "empty key",
			serviceAccountKey: "",
			wantErr:           false, // Empty key is valid (will use default credentials)
		},
		{
			name:              "invalid JSON",
			serviceAccountKey: "not-json",
			wantErr:           true,
			errMsg:            "failed to parse service account key",
		},
		{
			name: "missing required fields",
			serviceAccountKey: `{
				"type": "service_account",
				"project_id": "test-project"
			}`,
			wantErr: true,
			errMsg:  "service account key missing required field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := &vertexAIAuthenticatorImpl{
				serviceAccountKey: tt.serviceAccountKey,
			}
			err := auth.ValidateCredentials()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewVertexAIAuthenticator(t *testing.T) {
	// Note: This test only verifies the authenticator creation logic.
	// It doesn't test actual Google Cloud authentication which would require
	// valid credentials and network access.

	tests := []struct {
		name                  string
		serviceAccountKey     string
		serviceAccountKeyPath string
		wantErr               bool
		errMsg                string
	}{
		{
			name:                  "no credentials provided",
			serviceAccountKey:     "",
			serviceAccountKeyPath: "",
			wantErr:               false, // Should succeed and use default credentials
		},
		{
			name: "valid service account key JSON",
			serviceAccountKey: `{
				"type": "service_account",
				"project_id": "test-project",
				"private_key_id": "key-id",
				"private_key": "-----BEGIN RSA PRIVATE KEY-----\nMIIBOwIBAAJBAOLr5vIzVJZQaudJJcVh8fFUvBT9gkH09jtpfwwhhp1V3k5rqeC8\n3zYLRXJL5Q6p3iqVrWtAKGrE4Y6ggDuMnEMCAwEAAQJBALu0tPVFGzaJS6L/AT1g\n3NrBmXNmGj6AqPfJY3tReWe9E04qmDz2HLMssO2fNwV5bxLLDd5iwTKlpE5vcr5E\nu5kCIQD1b5M+BvKLPhKBGc7f8h2oXETnogU+w8R5P2oLP1dG1QIhAOzfPnRQQypL\nK0OccJiXUr0i5DeVTN8TGpWa6XimFk73AiAbLuNwKUhrkwWh4ThaMc0w7kR1qZ3X\nvZrHBXyWLddd7QIgNa/+lVGGO2F5pXpdNykJZeeqc6qv7X8qOEIxt5BnggMCIDvG\n7y1Mr+hFPepFOi1qzHkhjnnFh8vMMKj8MgMt+OKM\n-----END RSA PRIVATE KEY-----",
				"client_email": "test@test-project.iam.gserviceaccount.com",
				"client_id": "123456789",
				"auth_uri": "https://accounts.google.com/o/oauth2/auth",
				"token_uri": "https://oauth2.googleapis.com/token",
				"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
				"client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/test%40test-project.iam.gserviceaccount.com"
			}`,
			serviceAccountKeyPath: "",
			wantErr:               false,
		},
		{
			name:                  "invalid service account key JSON",
			serviceAccountKey:     "invalid-json",
			serviceAccountKeyPath: "",
			wantErr:               true,
			errMsg:                "failed to create token source",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth, err := NewVertexAIAuthenticator(tt.serviceAccountKey, tt.serviceAccountKeyPath)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, auth)
			}
		})
	}
}

func TestCreateTokenSourceFromJSON(t *testing.T) {
	auth := &vertexAIAuthenticatorImpl{}
	ctx := context.Background()

	tests := []struct {
		name    string
		keyJSON string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid service account key",
			keyJSON: `{
				"type": "service_account",
				"project_id": "test-project",
				"private_key_id": "key-id",
				"private_key": "-----BEGIN RSA PRIVATE KEY-----\nMIIBOwIBAAJBAOLr5vIzVJZQaudJJcVh8fFUvBT9gkH09jtpfwwhhp1V3k5rqeC8\n3zYLRXJL5Q6p3iqVrWtAKGrE4Y6ggDuMnEMCAwEAAQJBALu0tPVFGzaJS6L/AT1g\n3NrBmXNmGj6AqPfJY3tReWe9E04qmDz2HLMssO2fNwV5bxLLDd5iwTKlpE5vcr5E\nu5kCIQD1b5M+BvKLPhKBGc7f8h2oXETnogU+w8R5P2oLP1dG1QIhAOzfPnRQQypL\nK0OccJiXUr0i5DeVTN8TGpWa6XimFk73AiAbLuNwKUhrkwWh4ThaMc0w7kR1qZ3X\nvZrHBXyWLddd7QIgNa/+lVGGO2F5pXpdNykJZeeqc6qv7X8qOEIxt5BnggMCIDvG\n7y1Mr+hFPepFOi1qzHkhjnnFh8vMMKj8MgMt+OKM\n-----END RSA PRIVATE KEY-----",
				"client_email": "test@test-project.iam.gserviceaccount.com",
				"client_id": "123456789",
				"auth_uri": "https://accounts.google.com/o/oauth2/auth",
				"token_uri": "https://oauth2.googleapis.com/token",
				"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
				"client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/test%40test-project.iam.gserviceaccount.com"
			}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			keyJSON: "not-json",
			wantErr: true,
			errMsg:  "invalid service account key JSON",
		},
		{
			name: "invalid key type",
			keyJSON: `{
				"type": "user",
				"project_id": "test-project",
				"private_key_id": "key-id",
				"private_key": "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
				"client_email": "test@test-project.iam.gserviceaccount.com"
			}`,
			wantErr: true,
			errMsg:  "invalid service account type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenSource, err := auth.createTokenSourceFromJSON(ctx, tt.keyJSON)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, tokenSource)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tokenSource)
			}
		})
	}
}

func TestServiceAccountKeyJSONParsing(t *testing.T) {
	// Test that JSON parsing works correctly for service account keys
	validKey := `{
		"type": "service_account",
		"project_id": "my-project-123",
		"private_key_id": "abc123",
		"private_key": "-----BEGIN RSA PRIVATE KEY-----\nMIIBOwIBAAJBAOLr5vIzVJZQaudJJcVh8fFUvBT9gkH09jtpfwwhhp1V3k5rqeC8\n3zYLRXJL5Q6p3iqVrWtAKGrE4Y6ggDuMnEMCAwEAAQJBALu0tPVFGzaJS6L/AT1g\n3NrBmXNmGj6AqPfJY3tReWe9E04qmDz2HLMssO2fNwV5bxLLDd5iwTKlpE5vcr5E\nu5kCIQD1b5M+BvKLPhKBGc7f8h2oXETnogU+w8R5P2oLP1dG1QIhAOzfPnRQQypL\nK0OccJiXUr0i5DeVTN8TGpWa6XimFk73AiAbLuNwKUhrkwWh4ThaMc0w7kR1qZ3X\nvZrHBXyWLddd7QIgNa/+lVGGO2F5pXpdNykJZeeqc6qv7X8qOEIxt5BnggMCIDvG\n7y1Mr+hFPepFOi1qzHkhjnnFh8vMMKj8MgMt+OKM\n-----END RSA PRIVATE KEY-----",
		"client_email": "service-account@my-project-123.iam.gserviceaccount.com",
		"client_id": "1234567890",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "https://oauth2.googleapis.com/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/service-account%40my-project-123.iam.gserviceaccount.com"
	}`

	var key ServiceAccountKey
	err := json.Unmarshal([]byte(validKey), &key)
	require.NoError(t, err)

	assert.Equal(t, "service_account", key.Type)
	assert.Equal(t, "my-project-123", key.ProjectID)
	assert.Equal(t, "abc123", key.PrivateKeyID)
	assert.Contains(t, key.PrivateKey, "BEGIN RSA PRIVATE KEY")
	assert.Equal(t, "service-account@my-project-123.iam.gserviceaccount.com", key.ClientEmail)
	assert.Equal(t, "1234567890", key.ClientID)
}

func TestAuthenticationPriority(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name                  string
		serviceAccountKey     string
		serviceAccountKeyPath string
		expectation           string
	}{
		{
			name: "priority 1: direct service account key",
			serviceAccountKey: `{
				"type": "service_account",
				"project_id": "test-project",
				"private_key_id": "key-id",
				"private_key": "-----BEGIN RSA PRIVATE KEY-----\nMIIBOwIBAAJBAOLr5vIzVJZQaudJJcVh8fFUvBT9gkH09jtpfwwhhp1V3k5rqeC8\n3zYLRXJL5Q6p3iqVrWtAKGrE4Y6ggDuMnEMCAwEAAQJBALu0tPVFGzaJS6L/AT1g\n3NrBmXNmGj6AqPfJY3tReWe9E04qmDz2HLMssO2fNwV5bxLLDd5iwTKlpE5vcr5E\nu5kCIQD1b5M+BvKLPhKBGc7f8h2oXETnogU+w8R5P2oLP1dG1QIhAOzfPnRQQypL\nK0OccJiXUr0i5DeVTN8TGpWa6XimFk73AiAbLuNwKUhrkwWh4ThaMc0w7kR1qZ3X\nvZrHBXyWLddd7QIgNa/+lVGGO2F5pXpdNykJZeeqc6qv7X8qOEIxt5BnggMCIDvG\n7y1Mr+hFPepFOi1qzHkhjnnFh8vMMKj8MgMt+OKM\n-----END RSA PRIVATE KEY-----",
				"client_email": "test@test-project.iam.gserviceaccount.com",
				"client_id": "123456789",
				"auth_uri": "https://accounts.google.com/o/oauth2/auth",
				"token_uri": "https://oauth2.googleapis.com/token",
				"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
				"client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/test%40test-project.iam.gserviceaccount.com"
			}`,
			serviceAccountKeyPath: "/path/to/key.json", // This should be ignored
			expectation:           "should use direct key",
		},
		{
			name:                  "priority 2: service account key file path",
			serviceAccountKey:     "",
			serviceAccountKeyPath: "/path/to/key.json",
			expectation:           "should attempt to use file path",
		},
		{
			name:                  "priority 3: application default credentials",
			serviceAccountKey:     "",
			serviceAccountKeyPath: "",
			expectation:           "should use default credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := &vertexAIAuthenticatorImpl{
				serviceAccountKey:     tt.serviceAccountKey,
				serviceAccountKeyPath: tt.serviceAccountKeyPath,
			}

			// Note: createTokenSource will fail for file path case since the file doesn't exist
			// and may fail for default credentials if not configured.
			// This test mainly verifies the priority logic.
			tokenSource, err := auth.createTokenSource(ctx)

			if tt.serviceAccountKey != "" {
				// Should succeed with valid JSON key
				assert.NoError(t, err)
				assert.NotNil(t, tokenSource)
			} else if tt.serviceAccountKeyPath != "" {
				// Should fail because file doesn't exist
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to read service account key file")
			} else {
				// May succeed or fail depending on environment
				// This is OK for unit test
				_ = tokenSource
			}
		})
	}
}
