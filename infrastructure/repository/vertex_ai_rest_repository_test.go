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

		usage, err := repo.GetUsageMetrics(projectID, "us-central1", start, now)
		assert.NoError(t, err)
		assert.NotNil(t, usage)

		// Even if no models are available, we should get valid empty usage
		assert.Equal(t, projectID, usage.ProjectID())
		assert.Equal(t, "us-central1", usage.Location())
		assert.GreaterOrEqual(t, usage.InputTokens(), int64(0))
		assert.GreaterOrEqual(t, usage.OutputTokens(), int64(0))
	})

	t.Run("ListAvailableLocations", func(t *testing.T) {
		locations, err := repo.ListAvailableLocations(projectID)
		assert.NoError(t, err)
		// Locations might be empty if no models are available
		t.Logf("Available locations: %v", locations)
	})
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

// Ensure VertexAIRESTRepository implements VertexAIRepository
var _ repository.VertexAIRepository = (*VertexAIRESTRepository)(nil)
