package repository

//func TestVertexAIRESTRepository_NewRepository(t *testing.T) {
//	repo, err := NewVertexAIRESTRepository("test-project", "")
//	assert.NoError(t, err)
//	assert.NotNil(t, repo)
//	assert.Equal(t, "test-project", repo.projectID)
//	assert.Equal(t, 10, repo.maxRetries)
//	assert.Equal(t, 2*time.Second, repo.retryDelay)
//}
//
//func TestVertexAIRESTRepository_SetRetryParameters(t *testing.T) {
//	repo, err := NewVertexAIRESTRepository("test-project", "")
//	require.NoError(t, err)
//
//	repo.SetMaxRetries(5)
//	assert.Equal(t, 5, repo.maxRetries)
//
//	repo.SetRetryDelay(5 * time.Second)
//	assert.Equal(t, 5*time.Second, repo.retryDelay)
//}
//
//func TestVertexAIRESTRepository_Close(t *testing.T) {
//	repo, err := NewVertexAIRESTRepository("test-project", "")
//	require.NoError(t, err)
//
//	err = repo.Close()
//	assert.NoError(t, err)
//}
//
//// Integration test (requires actual GCP credentials)
//func TestVertexAIRESTRepository_Integration(t *testing.T) {
//	// Skip if not in integration test mode
//	if testing.Short() {
//		t.Skip("Skipping integration test")
//	}
//
//	projectID := "infra-dev-1"
//	repo, err := NewVertexAIRESTRepository(projectID, "")
//	require.NoError(t, err)
//
//	// Set shorter retry parameters for testing
//	repo.SetMaxRetries(3)
//	repo.SetRetryDelay(1 * time.Second)
//
//	t.Run("CheckConnection", func(t *testing.T) {
//		err := repo.CheckConnection()
//		// We expect this to fail if models are not available
//		// but the function should handle it gracefully
//		if err != nil {
//			t.Logf("CheckConnection returned error (expected if no models available): %v", err)
//		}
//	})
//
//	t.Run("GetUsageMetrics", func(t *testing.T) {
//		now := time.Now()
//		start := now.Add(-1 * time.Hour)
//
//		usage, err := repo.GetUsageMetrics(projectID, "us-central1", start, now)
//		assert.NoError(t, err)
//		assert.NotNil(t, usage)
//
//		// Even if no models are available, we should get valid empty usage
//		assert.Equal(t, projectID, usage.ProjectID())
//		assert.Equal(t, "us-central1", usage.Location())
//		assert.GreaterOrEqual(t, usage.InputTokens(), int64(0))
//		assert.GreaterOrEqual(t, usage.OutputTokens(), int64(0))
//	})
//
//	t.Run("ListAvailableLocations", func(t *testing.T) {
//		locations, err := repo.ListAvailableLocations(projectID)
//		assert.NoError(t, err)
//		// Locations might be empty if no models are available
//		t.Logf("Available locations: %v", locations)
//	})
//}
