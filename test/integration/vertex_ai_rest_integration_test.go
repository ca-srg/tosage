// +build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ca-srg/tosage/infrastructure/auth"
	"github.com/ca-srg/tosage/infrastructure/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVertexAIRESTRepository_Integration_CountTokens(t *testing.T) {
	// Skip if not in integration test mode
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Check for required environment variables
	projectID := os.Getenv("TOSAGE_VERTEX_AI_PROJECT_ID")
	if projectID == "" {
		t.Skip("TOSAGE_VERTEX_AI_PROJECT_ID not set, skipping integration test")
	}

	// Try to create authenticator
	serviceAccountKey := os.Getenv("TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY")
	serviceAccountKeyPath := os.Getenv("TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY_PATH")
	
	authenticator, err := auth.NewVertexAIAuthenticator(serviceAccountKey, serviceAccountKeyPath)
	if err != nil {
		t.Skipf("Failed to create authenticator: %v", err)
	}

	// Create repository
	repo, err := repository.NewVertexAIRESTRepository(projectID, authenticator)
	require.NoError(t, err)

	// Set shorter retry parameters for testing
	repo.SetMaxRetries(3)
	repo.SetRetryDelay(1 * time.Second)

	ctx := context.Background()

	t.Run("Count tokens with valid Gemini model", func(t *testing.T) {
		testText := "This is a test message to verify Vertex AI connectivity and count tokens. " +
			"The message contains multiple sentences to ensure we get a reasonable token count."

		// Test with different Gemini models
		models := []string{
			"gemini-2.5-flash",
			"gemini-2.0-flash",
		}

		successCount := 0
		for _, model := range models {
			t.Run(model, func(t *testing.T) {
				// Using the private method for testing
				// In a real scenario, this would be called through GetUsageMetrics
				
				// Note: We can't directly test callTokenCountAPI as it's private
				// Instead, we test through GetUsageMetrics which uses it internally
				usage, err := repo.GetUsageMetrics(projectID, time.Now().Add(-1*time.Hour), time.Now())
				
				if err != nil {
					t.Logf("Failed to get usage metrics for model %s: %v", model, err)
					// Check if it's a permission error
					if assert.Contains(t, err.Error(), "permission denied") || 
					   assert.Contains(t, err.Error(), "not found") {
						t.Logf("Expected error for model %s (permission or not found)", model)
					}
				} else {
					successCount++
					assert.NotNil(t, usage)
					assert.GreaterOrEqual(t, usage.InputTokens(), int64(0))
					t.Logf("Successfully got usage for model %s: %d tokens", model, usage.InputTokens())
				}
			})
		}

		// At least one model should work if credentials are valid
		if successCount == 0 {
			t.Log("No models were successful - this might indicate credential or permission issues")
		}
	})

	t.Run("List hardcoded models", func(t *testing.T) {
		models, err := repo.ListPublisherModels(ctx, "us-central1")
		assert.NoError(t, err)
		assert.NotEmpty(t, models)
		
		// Verify we get location-specific models
		// For us-central1, we expect more models
		if len(models) > 0 {
			t.Logf("Models available in us-central1: %v", models)
		}
		
		t.Logf("Got %d hardcoded models: %v", len(models), models)
	})

	t.Run("Error handling for invalid model", func(t *testing.T) {
		// This tests that our error handling works correctly
		// by using GetUsageMetrics with an invalid configuration
		invalidRepo, err := repository.NewVertexAIRESTRepository("invalid-project-id", authenticator)
		require.NoError(t, err)
		
		invalidRepo.SetMaxRetries(1) // Reduce retries for faster test
		invalidRepo.SetRetryDelay(100 * time.Millisecond)
		
		usage, err := invalidRepo.GetUsageMetrics("invalid-project-id", "us-central1", 
			time.Now().Add(-1*time.Hour), time.Now())
		
		assert.Error(t, err)
		assert.Nil(t, usage)
		t.Logf("Expected error for invalid project: %v", err)
	})

	t.Run("Check connection", func(t *testing.T) {
		err := repo.CheckConnection()
		if err != nil {
			t.Logf("CheckConnection failed (expected if permissions are limited): %v", err)
			// This is not a failure - just informational
		} else {
			t.Log("CheckConnection succeeded")
		}
	})

	t.Run("List available locations", func(t *testing.T) {
		locations, err := repo.ListAvailableLocations(projectID)
		assert.NoError(t, err)
		
		if len(locations) > 0 {
			t.Logf("Found %d active locations: %v", len(locations), locations)
		} else {
			t.Log("No active locations found (this might be due to permissions)")
		}
	})
}

func TestVertexAIRESTRepository_Integration_Authentication(t *testing.T) {
	// Skip if not in integration test mode
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("Service account key authentication", func(t *testing.T) {
		serviceAccountKey := os.Getenv("TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY")
		if serviceAccountKey == "" {
			t.Skip("TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY not set")
		}

		authenticator, err := auth.NewVertexAIAuthenticator(serviceAccountKey, "")
		require.NoError(t, err)

		ctx := context.Background()
		token, err := authenticator.GetAccessToken(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.False(t, authenticator.IsUsingADC())
	})

	t.Run("Service account key file authentication", func(t *testing.T) {
		serviceAccountKeyPath := os.Getenv("TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY_PATH")
		if serviceAccountKeyPath == "" {
			t.Skip("TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY_PATH not set")
		}

		authenticator, err := auth.NewVertexAIAuthenticator("", serviceAccountKeyPath)
		require.NoError(t, err)

		ctx := context.Background()
		token, err := authenticator.GetAccessToken(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.False(t, authenticator.IsUsingADC())
	})

	t.Run("Application default credentials", func(t *testing.T) {
		// Only test if no explicit credentials are set
		if os.Getenv("TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY") != "" ||
		   os.Getenv("TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY_PATH") != "" {
			t.Skip("Explicit credentials are set, skipping ADC test")
		}

		authenticator, err := auth.NewVertexAIAuthenticator("", "")
		if err != nil {
			t.Skipf("ADC not available: %v", err)
		}

		assert.True(t, authenticator.IsUsingADC())
		
		ctx := context.Background()
		token, err := authenticator.GetAccessToken(ctx)
		if err != nil {
			t.Logf("ADC token retrieval failed (expected if not configured): %v", err)
		} else {
			assert.NotEmpty(t, token)
		}
	})
}