package integration

import (
	"context"
	"encoding/base64"
	"os"
	"testing"

	"github.com/ca-srg/tosage/infrastructure/auth"
	"github.com/ca-srg/tosage/infrastructure/di"
	"github.com/ca-srg/tosage/infrastructure/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVertexAIAuthentication_ServiceAccountKey(t *testing.T) {
	// Skip if no service account key is provided
	serviceAccountKey := os.Getenv("TEST_VERTEX_AI_SERVICE_ACCOUNT_KEY")
	if serviceAccountKey == "" {
		t.Skip("TEST_VERTEX_AI_SERVICE_ACCOUNT_KEY not set, skipping integration test")
	}

	projectID := os.Getenv("TEST_VERTEX_AI_PROJECT_ID")
	if projectID == "" {
		t.Skip("TEST_VERTEX_AI_PROJECT_ID not set, skipping integration test")
	}

	// Test with direct service account key
	t.Run("direct service account key authentication", func(t *testing.T) {
		authenticator, err := auth.NewVertexAIAuthenticator(serviceAccountKey, "")
		require.NoError(t, err)
		require.NotNil(t, authenticator)

		// Validate credentials
		err = authenticator.ValidateCredentials()
		assert.NoError(t, err)

		// Get access token
		ctx := context.Background()
		token, err := authenticator.GetAccessToken(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		// Create repository with authenticator
		repo, err := repository.NewVertexAIRESTRepository(projectID, authenticator)
		assert.NoError(t, err)
		assert.NotNil(t, repo)

		// Test connection
		err = repo.CheckConnection()
		assert.NoError(t, err)
	})
}

func TestVertexAIAuthentication_ServiceAccountKeyFile(t *testing.T) {
	// Skip if no service account key file is provided
	keyPath := os.Getenv("TEST_VERTEX_AI_SERVICE_ACCOUNT_KEY_PATH")
	if keyPath == "" {
		t.Skip("TEST_VERTEX_AI_SERVICE_ACCOUNT_KEY_PATH not set, skipping integration test")
	}

	projectID := os.Getenv("TEST_VERTEX_AI_PROJECT_ID")
	if projectID == "" {
		t.Skip("TEST_VERTEX_AI_PROJECT_ID not set, skipping integration test")
	}

	// Test with service account key file
	t.Run("service account key file authentication", func(t *testing.T) {
		authenticator, err := auth.NewVertexAIAuthenticator("", keyPath)
		require.NoError(t, err)
		require.NotNil(t, authenticator)

		// Get access token
		ctx := context.Background()
		token, err := authenticator.GetAccessToken(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		// Create repository with authenticator
		repo, err := repository.NewVertexAIRESTRepository(projectID, authenticator)
		assert.NoError(t, err)
		assert.NotNil(t, repo)

		// Test connection
		err = repo.CheckConnection()
		assert.NoError(t, err)
	})
}

func TestVertexAIAuthentication_DefaultCredentials(t *testing.T) {
	// Skip if no default credentials are available
	projectID := os.Getenv("TEST_VERTEX_AI_PROJECT_ID")
	if projectID == "" {
		t.Skip("TEST_VERTEX_AI_PROJECT_ID not set, skipping integration test")
	}

	// Check if application default credentials are available
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		gcloudConfigPath := os.ExpandEnv("$HOME/.config/gcloud/application_default_credentials.json")
		if _, err := os.Stat(gcloudConfigPath); os.IsNotExist(err) {
			t.Skip("No default credentials available, skipping integration test")
		}
	}

	// Test with application default credentials
	t.Run("application default credentials authentication", func(t *testing.T) {
		authenticator, err := auth.NewVertexAIAuthenticator("", "")
		require.NoError(t, err)
		require.NotNil(t, authenticator)

		// Get access token
		ctx := context.Background()
		token, err := authenticator.GetAccessToken(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		// Create repository with authenticator
		repo, err := repository.NewVertexAIRESTRepository(projectID, authenticator)
		assert.NoError(t, err)
		assert.NotNil(t, repo)

		// Test connection
		err = repo.CheckConnection()
		assert.NoError(t, err)
	})
}

func TestVertexAIAuthentication_Priority(t *testing.T) {
	// This test verifies the three-tier priority system
	serviceAccountKey := os.Getenv("TEST_VERTEX_AI_SERVICE_ACCOUNT_KEY")
	keyPath := os.Getenv("TEST_VERTEX_AI_SERVICE_ACCOUNT_KEY_PATH")
	projectID := os.Getenv("TEST_VERTEX_AI_PROJECT_ID")

	if projectID == "" {
		t.Skip("TEST_VERTEX_AI_PROJECT_ID not set, skipping integration test")
	}

	// Test priority 1: Direct key takes precedence over file path
	if serviceAccountKey != "" && keyPath != "" {
		t.Run("priority 1 - direct key over file path", func(t *testing.T) {
			authenticator, err := auth.NewVertexAIAuthenticator(serviceAccountKey, keyPath)
			require.NoError(t, err)
			require.NotNil(t, authenticator)

			// Should use direct key, not file
			ctx := context.Background()
			token, err := authenticator.GetAccessToken(ctx)
			assert.NoError(t, err)
			assert.NotEmpty(t, token)
		})
	}

	// Test priority 2: File path when no direct key
	if keyPath != "" {
		t.Run("priority 2 - file path when no direct key", func(t *testing.T) {
			authenticator, err := auth.NewVertexAIAuthenticator("", keyPath)
			require.NoError(t, err)
			require.NotNil(t, authenticator)

			ctx := context.Background()
			token, err := authenticator.GetAccessToken(ctx)
			assert.NoError(t, err)
			assert.NotEmpty(t, token)
		})
	}

	// Test priority 3: Default credentials when neither key nor path provided
	t.Run("priority 3 - default credentials", func(t *testing.T) {
		// Skip if no default credentials
		if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
			gcloudConfigPath := os.ExpandEnv("$HOME/.config/gcloud/application_default_credentials.json")
			if _, err := os.Stat(gcloudConfigPath); os.IsNotExist(err) {
				t.Skip("No default credentials available")
			}
		}

		authenticator, err := auth.NewVertexAIAuthenticator("", "")
		require.NoError(t, err)
		require.NotNil(t, authenticator)

		ctx := context.Background()
		token, err := authenticator.GetAccessToken(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
	})
}

func TestVertexAIAuthentication_EndToEnd(t *testing.T) {
	// This test verifies the complete integration from config to API call
	serviceAccountKey := os.Getenv("TEST_VERTEX_AI_SERVICE_ACCOUNT_KEY")
	projectID := os.Getenv("TEST_VERTEX_AI_PROJECT_ID")

	if serviceAccountKey == "" || projectID == "" {
		t.Skip("TEST_VERTEX_AI_SERVICE_ACCOUNT_KEY or TEST_VERTEX_AI_PROJECT_ID not set, skipping integration test")
	}

	// Set up environment variables (with base64 encoded key)
	_ = os.Setenv("TOSAGE_VERTEX_AI_ENABLED", "true")
	_ = os.Setenv("TOSAGE_VERTEX_AI_PROJECT_ID", projectID)
	base64Key := base64.StdEncoding.EncodeToString([]byte(serviceAccountKey))
	_ = os.Setenv("TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY", base64Key)
	defer func() {
		_ = os.Unsetenv("TOSAGE_VERTEX_AI_ENABLED")
		_ = os.Unsetenv("TOSAGE_VERTEX_AI_PROJECT_ID")
		_ = os.Unsetenv("TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY")
	}()

	// Create container with Vertex AI enabled
	container, err := di.NewContainer(di.WithVertexAIEnabled(true))
	require.NoError(t, err)
	require.NotNil(t, container)

	// Get Vertex AI service
	vertexAIService := container.GetVertexAIService()
	assert.NotNil(t, vertexAIService) // Should be initialized when properly configured

	// Check configuration
	cfg := container.GetConfig()
	assert.NotNil(t, cfg.VertexAI)
	assert.True(t, cfg.VertexAI.Enabled)
	assert.Equal(t, projectID, cfg.VertexAI.ProjectID)
	assert.Equal(t, serviceAccountKey, cfg.VertexAI.ServiceAccountKey)
}

func TestVertexAIAuthentication_InvalidCredentials(t *testing.T) {
	// Test with invalid service account key
	invalidKey := `{
		"type": "service_account",
		"project_id": "invalid-project",
		"private_key_id": "invalid",
		"private_key": "-----BEGIN RSA PRIVATE KEY-----\ninvalid-key\n-----END RSA PRIVATE KEY-----",
		"client_email": "invalid@invalid-project.iam.gserviceaccount.com"
	}`

	authenticator, err := auth.NewVertexAIAuthenticator(invalidKey, "")
	// Should create authenticator successfully (validation happens on token retrieval)
	assert.NoError(t, err)
	assert.NotNil(t, authenticator)

	// But getting token should fail
	ctx := context.Background()
	token, err := authenticator.GetAccessToken(ctx)
	assert.Error(t, err)
	assert.Empty(t, token)
}

func TestVertexAIAuthentication_MissingProjectID(t *testing.T) {
	// Test that proper error is returned when project ID is missing
	_ = os.Setenv("TOSAGE_VERTEX_AI_ENABLED", "true")
	_ = os.Setenv("TOSAGE_VERTEX_AI_PROJECT_ID", "") // Empty project ID
	defer func() {
		_ = os.Unsetenv("TOSAGE_VERTEX_AI_ENABLED")
		_ = os.Unsetenv("TOSAGE_VERTEX_AI_PROJECT_ID")
	}()

	// Create container
	container, err := di.NewContainer(di.WithVertexAIEnabled(true))
	require.NoError(t, err) // Container creation should succeed

	// But Vertex AI service should not be initialized
	vertexAIService := container.GetVertexAIService()
	assert.Nil(t, vertexAIService) // Should be nil when project ID is missing
}
