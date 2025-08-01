package repository

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ca-srg/tosage/domain/entity"
	"github.com/ca-srg/tosage/domain/repository"
	"github.com/ca-srg/tosage/infrastructure/auth"
)

// VertexAIRESTRepository implements VertexAIRepository using REST API with retry logic
type VertexAIRESTRepository struct {
	projectID      string
	authenticator  auth.VertexAIAuthenticator
	client         *http.Client
	maxRetries     int
	retryDelay     time.Duration
	serviceAccount string
}

// NewVertexAIRESTRepository creates a new Vertex AI REST repository
func NewVertexAIRESTRepository(projectID string, authenticator auth.VertexAIAuthenticator) (*VertexAIRESTRepository, error) {
	if authenticator == nil {
		return nil, fmt.Errorf("authenticator cannot be nil")
	}

	// Try to extract service account email from environment
	serviceAccount := "YOUR_SERVICE_ACCOUNT_EMAIL"
	if keyBase64 := os.Getenv("TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY"); keyBase64 != "" {
		// Decode base64
		keyData, err := base64.StdEncoding.DecodeString(keyBase64)
		if err == nil {
			// Try to parse as JSON
			var key struct {
				ClientEmail string `json:"client_email"`
			}
			if err := json.Unmarshal(keyData, &key); err == nil && key.ClientEmail != "" {
				serviceAccount = key.ClientEmail
			}
		}
	}

	return &VertexAIRESTRepository{
		projectID:      projectID,
		authenticator:  authenticator,
		client:         &http.Client{Timeout: 30 * time.Second},
		maxRetries:     10,
		retryDelay:     2 * time.Second,
		serviceAccount: serviceAccount,
	}, nil
}

// TokenCountRequest represents the request structure for token counting
type TokenCountRequest struct {
	Contents []Content `json:"contents"`
}

// Content represents a content item in the request
type Content struct {
	Role  string `json:"role"`
	Parts []Part `json:"parts"`
}

// Part represents a part of the content
type Part struct {
	Text string `json:"text"`
}

// TokenCountResponse represents the response from the token count API
type TokenCountResponse struct {
	TotalTokens             int64 `json:"totalTokens"`
	CachedContentTokenCount int64 `json:"cachedContentTokenCount,omitempty"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// getAccessToken retrieves a valid access token
func (r *VertexAIRESTRepository) getAccessToken(ctx context.Context) (string, error) {
	return r.authenticator.GetAccessToken(ctx)
}

// ListPublisherModels returns a hardcoded list of supported Gemini models for the specific location
func (r *VertexAIRESTRepository) ListPublisherModels(ctx context.Context, location string) ([]string, error) {
	// Return models based on location-specific availability
	// This helps avoid unnecessary 404 errors for models not available in certain regions

	// Get location-specific models
	geminiModels := GetAvailableModelsForLocation(location)

	return geminiModels, nil
}

// callTokenCountAPI calls the Vertex AI token count API with retry logic
func (r *VertexAIRESTRepository) callTokenCountAPI(ctx context.Context, location, model, text string) (int64, error) {
	url := fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:countTokens",
		location, r.projectID, location, model)

	reqBody := TokenCountRequest{
		Contents: []Content{
			{
				Role: "user",
				Parts: []Part{
					{Text: text},
				},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= r.maxRetries; attempt++ {

		// Get fresh token for each attempt
		token, err := r.getAccessToken(ctx)
		if err != nil {
			lastErr = fmt.Errorf("authentication failed: %w", err)
			// Use exponential backoff for retries
			backoffDelay := r.retryDelay * time.Duration(1<<uint(attempt-1))
			if backoffDelay > 30*time.Second {
				backoffDelay = 30 * time.Second
			}
			time.Sleep(backoffDelay)
			continue
		}

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return 0, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := r.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("network error: %w", err)
			// Use exponential backoff for retries
			backoffDelay := r.retryDelay * time.Duration(1<<uint(attempt-1))
			if backoffDelay > 30*time.Second {
				backoffDelay = 30 * time.Second
			}
			time.Sleep(backoffDelay)
			continue
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
			}
		}()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			// Use exponential backoff for retries
			backoffDelay := r.retryDelay * time.Duration(1<<uint(attempt-1))
			if backoffDelay > 30*time.Second {
				backoffDelay = 30 * time.Second
			}
			time.Sleep(backoffDelay)
			continue
		}

		// Success case
		if resp.StatusCode == http.StatusOK {
			var tokenResp TokenCountResponse
			if err := json.Unmarshal(body, &tokenResp); err != nil {
				lastErr = fmt.Errorf("failed to parse response: %w", err)
				// Use exponential backoff for retries
				backoffDelay := r.retryDelay * time.Duration(1<<uint(attempt-1))
				if backoffDelay > 30*time.Second {
					backoffDelay = 30 * time.Second
				}
				time.Sleep(backoffDelay)
				continue
			}
			return tokenResp.TotalTokens, nil
		}

		// Parse error response
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			lastErr = fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		} else {
			lastErr = fmt.Errorf("API error: %s (code: %d)", errResp.Error.Message, errResp.Error.Code)
		}

		// Handle non-retryable errors
		switch resp.StatusCode {
		case http.StatusNotFound:
			// Model not found - non-retryable
			return 0, fmt.Errorf("model '%s' not found: Publisher Model `projects/%s/locations/%s/publishers/google/models/%s` was not found or your project does not have access to it",
				model, r.projectID, location, model)

		case http.StatusForbidden:
			// Permission denied - non-retryable
			permissionErr := fmt.Errorf("permission denied: Missing permission 'aiplatform.endpoints.predict' for model '%s'. "+
				"Grant the service account '%s' the 'Vertex AI User' role using: "+
				"gcloud projects add-iam-policy-binding %s --member='serviceAccount:%s' --role='roles/aiplatform.user'",
				model, r.serviceAccount, r.projectID, r.serviceAccount)

			// Log detailed remediation steps without exiting
			log.Printf("[ERROR] %v", permissionErr)
			log.Printf("[ERROR] Additional steps to fix:")
			log.Printf("[ERROR] 1. Ensure Vertex AI API is enabled: gcloud services enable aiplatform.googleapis.com --project=%s", r.projectID)
			log.Printf("[ERROR] 2. Check service account credentials in TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY")

			return 0, permissionErr

		case http.StatusTooManyRequests:
			// Rate limit - retryable with longer backoff
			backoffDelay := r.retryDelay * time.Duration(2<<uint(attempt))
			if backoffDelay > 60*time.Second {
				backoffDelay = 60 * time.Second
			}
			time.Sleep(backoffDelay)
			continue

		default:
			// Other errors - retryable with exponential backoff
			if resp.StatusCode >= 500 {
			} else {
			}
			backoffDelay := r.retryDelay * time.Duration(1<<uint(attempt-1))
			if backoffDelay > 30*time.Second {
				backoffDelay = 30 * time.Second
			}
			time.Sleep(backoffDelay)
		}
	}

	return 0, fmt.Errorf("failed after %d attempts: %w", r.maxRetries, lastErr)
}

// GetUsageMetrics retrieves Vertex AI usage metrics (placeholder implementation)
func (r *VertexAIRESTRepository) GetUsageMetrics(projectID string, start, end time.Time) (*entity.VertexAIUsage, error) {

	// For now, we'll use a simple test to demonstrate the retry logic
	// In a real implementation, this would aggregate actual usage data
	ctx := context.Background()

	// Use a default location since we're not filtering by location anymore
	location := "us-central1"
	
	// Get available models from API
	models, err := r.ListPublisherModels(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("failed to list publisher models: %w", err)
	}

	// Check if we got any models
	if len(models) == 0 {
		return nil, fmt.Errorf("no models found")
	}


	// Try to count tokens for a test message
	testText := "This is a test message to verify Vertex AI connectivity and count tokens."

	var totalTokens int64
	var successfulModel string
	var successfulModels []string
	var failureReasons = make(map[string]string)

	for _, model := range models {
		tokens, err := r.callTokenCountAPI(ctx, location, model, testText)
		if err == nil {
			totalTokens = tokens
			successfulModel = model
			successfulModels = append(successfulModels, fmt.Sprintf("%s (tokens: %d)", model, tokens))
			break
		}
		// Store failure reason for debugging
		failureReasons[model] = err.Error()
	}

	// Log summary of model availability
	if len(successfulModels) > 0 {
		log.Printf("[INFO] Successfully connected to models in %s: %v", location, successfulModels)
	}
	if len(failureReasons) > 0 {
		log.Printf("[INFO] Model availability issues in %s:", location)
		for model, reason := range failureReasons {
			log.Printf("[INFO]   - %s: %s", model, reason)
		}
	}

	if successfulModel == "" {
		return nil, fmt.Errorf("could not connect to any Vertex AI model")
	}

	// Create a simple metric for demonstration
	modelMetrics := []entity.VertexAIModelMetric{
		{
			ModelID:      successfulModel,
			InputTokens:  totalTokens,
			OutputTokens: 0,
			RequestCount: 1,
			LatencyMs:    100,
			Cost:         0.0,
		},
	}

	return entity.NewVertexAIUsage(
		totalTokens,
		0,
		0.0,
		modelMetrics,
		projectID,
		"", // Empty location since we're not filtering by location
	)
}

// GetDailyUsage retrieves aggregated usage for a specific date
func (r *VertexAIRESTRepository) GetDailyUsage(projectID string, date time.Time) (*entity.VertexAIUsage, error) {
	// Convert to JST for consistent date boundaries
	jst, _ := time.LoadLocation("Asia/Tokyo")
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, jst)
	endOfDay := startOfDay.Add(24 * time.Hour)

	return r.GetUsageMetrics(projectID, startOfDay, endOfDay)
}

// GetCurrentMonthUsage retrieves usage for the current month
func (r *VertexAIRESTRepository) GetCurrentMonthUsage(projectID string) (*entity.VertexAIUsage, error) {
	jst, _ := time.LoadLocation("Asia/Tokyo")
	now := time.Now().In(jst)
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, jst)

	return r.GetUsageMetrics(projectID, startOfMonth, now)
}

// CheckConnection verifies Vertex AI API connectivity
func (r *VertexAIRESTRepository) CheckConnection() error {
	ctx := context.Background()

	// Try to connect to at least one model
	locations := []string{"us-central1", "asia-northeast1"}

	for _, location := range locations {
		// Get available models from API
		models, err := r.ListPublisherModels(ctx, location)
		if err != nil {
			continue
		}

		// Try each model
		for _, model := range models {
			_, err := r.callTokenCountAPI(ctx, location, model, "test")
			if err == nil {
				return nil // Connection successful
			}
		}
	}

	return fmt.Errorf("could not connect to any Vertex AI model")
}


// SetMaxRetries sets the maximum number of retries
func (r *VertexAIRESTRepository) SetMaxRetries(retries int) {
	r.maxRetries = retries
}

// SetRetryDelay sets the delay between retries
func (r *VertexAIRESTRepository) SetRetryDelay(delay time.Duration) {
	r.retryDelay = delay
}

// Close closes any resources (no-op for REST client)
func (r *VertexAIRESTRepository) Close() error {
	return nil
}

// Ensure VertexAIRESTRepository implements VertexAIRepository
var _ repository.VertexAIRepository = (*VertexAIRESTRepository)(nil)
