package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ca-srg/tosage/domain/repository"
	"github.com/ca-srg/tosage/infrastructure/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// MockVertexAIAuthenticator is a mock implementation of VertexAIAuthenticator
type MockVertexAIAuthenticator struct {
	mock.Mock
}

func (m *MockVertexAIAuthenticator) GetAccessToken(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *MockVertexAIAuthenticator) ValidateCredentials() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockVertexAIAuthenticator) IsUsingADC() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockVertexAIAuthenticator) GetTokenSource() oauth2.TokenSource {
	args := m.Called()
	if tokenSource := args.Get(0); tokenSource != nil {
		return tokenSource.(oauth2.TokenSource)
	}
	return nil
}

func TestVertexAIRESTRepository_NewRepository(t *testing.T) {
	mockAuth := new(MockVertexAIAuthenticator)

	// Test successful creation
	repo, err := NewVertexAIRESTRepository("test-project", mockAuth)
	assert.NoError(t, err)
	assert.NotNil(t, repo)
	assert.Equal(t, "test-project", repo.projectID)
	assert.Equal(t, mockAuth, repo.authenticator)
	assert.Equal(t, 10, repo.maxRetries)
	assert.Equal(t, 2*time.Second, repo.retryDelay)
}

func TestVertexAIRESTRepository_NewRepository_NilAuthenticator(t *testing.T) {
	// Test with nil authenticator
	repo, err := NewVertexAIRESTRepository("test-project", nil)
	assert.Error(t, err)
	assert.Nil(t, repo)
	assert.Contains(t, err.Error(), "authenticator cannot be nil")
}

func TestVertexAIRESTRepository_SetRetryParameters(t *testing.T) {
	mockAuth := new(MockVertexAIAuthenticator)
	repo, err := NewVertexAIRESTRepository("test-project", mockAuth)
	require.NoError(t, err)

	repo.SetMaxRetries(5)
	assert.Equal(t, 5, repo.maxRetries)

	repo.SetRetryDelay(5 * time.Second)
	assert.Equal(t, 5*time.Second, repo.retryDelay)
}

func TestVertexAIRESTRepository_Close(t *testing.T) {
	mockAuth := new(MockVertexAIAuthenticator)
	repo, err := NewVertexAIRESTRepository("test-project", mockAuth)
	require.NoError(t, err)

	err = repo.Close()
	assert.NoError(t, err)
}

func TestVertexAIRESTRepository_GetAccessToken(t *testing.T) {
	mockAuth := new(MockVertexAIAuthenticator)
	repo, err := NewVertexAIRESTRepository("test-project", mockAuth)
	require.NoError(t, err)

	ctx := context.Background()
	expectedToken := "test-access-token"

	// Mock successful token retrieval
	mockAuth.On("GetAccessToken", ctx).Return(expectedToken, nil)

	token, err := repo.getAccessToken(ctx)
	assert.NoError(t, err)
	assert.Equal(t, expectedToken, token)
	mockAuth.AssertExpectations(t)
}

func TestVertexAIRESTRepository_GetAccessToken_Error(t *testing.T) {
	mockAuth := new(MockVertexAIAuthenticator)
	repo, err := NewVertexAIRESTRepository("test-project", mockAuth)
	require.NoError(t, err)

	ctx := context.Background()
	expectedError := errors.New("authentication failed")

	// Mock failed token retrieval
	mockAuth.On("GetAccessToken", ctx).Return("", expectedError)

	token, err := repo.getAccessToken(ctx)
	assert.Error(t, err)
	assert.Equal(t, "", token)
	assert.Equal(t, expectedError, err)
	mockAuth.AssertExpectations(t)
}

// Integration test with real authenticator (requires actual GCP credentials)
func TestVertexAIRESTRepository_Integration(t *testing.T) {
	// Skip if not in integration test mode
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Skip if no credentials are available
	projectID := "test-project"

	// Try to create authenticator with default credentials
	authenticator, err := auth.NewVertexAIAuthenticator("", "")
	if err != nil {
		t.Skip("No valid credentials available for integration test")
	}

	repo, err := NewVertexAIRESTRepository(projectID, authenticator)
	require.NoError(t, err)

	// Set shorter retry parameters for testing
	repo.SetMaxRetries(3)
	repo.SetRetryDelay(1 * time.Second)

	t.Run("CheckConnection", func(t *testing.T) {
		err := repo.CheckConnection()
		// We expect this to fail if models are not available
		// but the function should handle it gracefully
		if err != nil {
			t.Logf("CheckConnection returned error (expected if no models available): %v", err)
		}
	})

	t.Run("GetUsageMetrics", func(t *testing.T) {
		now := time.Now()
		start := now.Add(-1 * time.Hour)

		usage, err := repo.GetUsageMetrics(projectID, start, now)

		// If we get an error (e.g., authentication issues), it's expected
		if err != nil {
			t.Logf("GetUsageMetrics returned error (expected if models unavailable): %v", err)
			return
		}

		// If no error, we should get valid usage
		assert.NotNil(t, usage)
		assert.Equal(t, projectID, usage.ProjectID())
		// Location is now empty since we don't filter by location
		assert.GreaterOrEqual(t, usage.InputTokens(), int64(0))
		assert.GreaterOrEqual(t, usage.OutputTokens(), int64(0))
	})

	// ListAvailableLocations has been removed since location filtering is no longer needed
}

// Test callTokenCountAPI with mock authenticator
func TestVertexAIRESTRepository_CallTokenCountAPI(t *testing.T) {
	// This test would require mocking HTTP client which is more complex
	// For now, we just ensure the authenticator is called correctly
	mockAuth := new(MockVertexAIAuthenticator)
	repo, err := NewVertexAIRESTRepository("test-project", mockAuth)
	require.NoError(t, err)

	// Set minimal retry parameters for faster test
	repo.SetMaxRetries(1)
	repo.SetRetryDelay(100 * time.Millisecond)

	ctx := context.Background()
	mockAuth.On("GetAccessToken", ctx).Return("", errors.New("auth failed"))

	// This will fail due to auth error, but that's what we're testing
	_, err = repo.callTokenCountAPI(ctx, "us-central1", "gemini-pro", "test text")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed after 1 attempts")
	mockAuth.AssertExpectations(t)
}

func TestVertexAIRESTRepository_ListPublisherModels(t *testing.T) {
	mockAuth := new(MockVertexAIAuthenticator)
	repo, err := NewVertexAIRESTRepository("test-project", mockAuth)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("Returns location-specific models", func(t *testing.T) {
		// Test US region (more models available)
		models, err := repo.ListPublisherModels(ctx, "us-central1")
		assert.NoError(t, err)
		assert.NotEmpty(t, models)
		assert.Contains(t, models, "gemini-2.5-flash")
		
		// Test Asia region (limited models)
		models, err = repo.ListPublisherModels(ctx, "asia-northeast1")
		assert.NoError(t, err)
		assert.NotEmpty(t, models)
		assert.Contains(t, models, "gemini-2.5-flash")
		
		// Test region with no specific models (should return defaults)
		models, err = repo.ListPublisherModels(ctx, "unknown-region")
		assert.NoError(t, err)
		assert.NotEmpty(t, models)
		assert.Contains(t, models, "gemini-pro")
	})

	t.Run("Returns different models for different locations", func(t *testing.T) {
		models1, err1 := repo.ListPublisherModels(ctx, "us-central1")
		assert.NoError(t, err1)
		
		models2, err2 := repo.ListPublisherModels(ctx, "asia-northeast2")
		assert.NoError(t, err2)
		
		// Models should be different based on location availability
		assert.NotEqual(t, models1, models2)
		
		// US should have more models
		assert.Greater(t, len(models1), len(models2))
	})

	t.Run("Does not make API calls", func(t *testing.T) {
		// This test verifies that no authenticator methods are called
		// since we're not making API calls anymore
		mockAuthClean := new(MockVertexAIAuthenticator)
		repoClean, err := NewVertexAIRESTRepository("test-project", mockAuthClean)
		require.NoError(t, err)
		
		models, err := repoClean.ListPublisherModels(ctx, "us-central1")
		assert.NoError(t, err)
		assert.NotEmpty(t, models)
		
		// Verify no calls were made to the authenticator
		mockAuthClean.AssertNotCalled(t, "GetAccessToken", mock.Anything)
		mockAuthClean.AssertNotCalled(t, "IsUsingADC")
	})
}

// Test error handling improvements
func TestVertexAIRESTRepository_CallTokenCountAPI_ErrorHandling(t *testing.T) {
	// This test would require mocking HTTP responses which is complex
	// For now, we test that the method handles authentication errors correctly
	mockAuth := new(MockVertexAIAuthenticator)
	repo, err := NewVertexAIRESTRepository("test-project", mockAuth)
	require.NoError(t, err)

	// Set minimal retry parameters for faster test
	repo.SetMaxRetries(2)
	repo.SetRetryDelay(100 * time.Millisecond)

	ctx := context.Background()

	t.Run("Authentication error with retry", func(t *testing.T) {
		// First call fails, second succeeds (but we'll fail at HTTP level)
		mockAuth.On("GetAccessToken", ctx).Return("", errors.New("auth failed")).Once()
		mockAuth.On("GetAccessToken", ctx).Return("test-token", nil).Once()

		// This will fail due to network error (no actual HTTP server)
		_, err := repo.callTokenCountAPI(ctx, "us-central1", "gemini-pro", "test text")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed after 2 attempts")
		
		// Verify that authentication was retried
		mockAuth.AssertNumberOfCalls(t, "GetAccessToken", 2)
	})
}

// Ensure VertexAIRESTRepository implements VertexAIRepository
var _ repository.VertexAIRepository = (*VertexAIRESTRepository)(nil)
