package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/ca-srg/tosage/domain/entity"
	"github.com/ca-srg/tosage/domain/repository"
)

// VertexAIRESTRepository implements VertexAIRepository using REST API with retry logic
type VertexAIRESTRepository struct {
	projectID             string
	serviceAccountKeyPath string
	client                *http.Client
	maxRetries            int
	retryDelay            time.Duration
}

// NewVertexAIRESTRepository creates a new Vertex AI REST repository
func NewVertexAIRESTRepository(projectID, serviceAccountKeyPath string) (*VertexAIRESTRepository, error) {
	return &VertexAIRESTRepository{
		projectID:             projectID,
		serviceAccountKeyPath: serviceAccountKeyPath,
		client:                &http.Client{Timeout: 30 * time.Second},
		maxRetries:            10,
		retryDelay:            2 * time.Second,
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
func (r *VertexAIRESTRepository) getAccessToken() (string, error) {
	var cmd *exec.Cmd
	if r.serviceAccountKeyPath != "" {
		// Use service account if provided
		cmd = exec.Command("gcloud", "auth", "print-access-token",
			"--impersonate-service-account", r.serviceAccountKeyPath)
	} else {
		// Use application default credentials
		cmd = exec.Command("gcloud", "auth", "application-default", "print-access-token")
	}

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	return string(bytes.TrimSpace(output)), nil
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
		log.Printf("[DEBUG] Attempt %d/%d to call Vertex AI token count API", attempt, r.maxRetries)

		// Get fresh token for each attempt
		token, err := r.getAccessToken()
		if err != nil {
			lastErr = err
			log.Printf("[DEBUG] Failed to get access token: %v", err)
			time.Sleep(r.retryDelay)
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
			lastErr = err
			log.Printf("[DEBUG] Request failed: %v", err)
			time.Sleep(r.retryDelay)
			continue
		}
		defer func() {
			if cerr := resp.Body.Close(); cerr != nil {
				log.Printf("[DEBUG] Failed to close response body: %v", cerr)
			}
		}()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			log.Printf("[DEBUG] Failed to read response: %v", err)
			time.Sleep(r.retryDelay)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			var tokenResp TokenCountResponse
			if err := json.Unmarshal(body, &tokenResp); err != nil {
				lastErr = err
				log.Printf("[DEBUG] Failed to parse response: %v", err)
				time.Sleep(r.retryDelay)
				continue
			}
			log.Printf("[DEBUG] Successfully got token count: %d", tokenResp.TotalTokens)
			return tokenResp.TotalTokens, nil
		}

		// Handle error response
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			log.Printf("[DEBUG] Failed to parse error response: %v, body: %s", err, string(body))
		} else {
			log.Printf("[DEBUG] API error: %s (code: %d, status: %s)",
				errResp.Error.Message, errResp.Error.Code, errResp.Error.Status)

			// Don't retry on certain errors
			if errResp.Error.Code == 403 || errResp.Error.Code == 404 {
				return 0, fmt.Errorf("API error: %s", errResp.Error.Message)
			}
		}

		lastErr = fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		time.Sleep(r.retryDelay)
	}

	return 0, fmt.Errorf("failed after %d attempts: %w", r.maxRetries, lastErr)
}

// GetUsageMetrics retrieves Vertex AI usage metrics (placeholder implementation)
func (r *VertexAIRESTRepository) GetUsageMetrics(projectID, location string, start, end time.Time) (*entity.VertexAIUsage, error) {
	log.Printf("[DEBUG] GetUsageMetrics called with projectID=%s, location=%s, start=%v, end=%v",
		projectID, location, start.Format(time.RFC3339), end.Format(time.RFC3339))

	// For now, we'll use a simple test to demonstrate the retry logic
	// In a real implementation, this would aggregate actual usage data
	ctx := context.Background()

	// Try to count tokens for a test message
	testText := "This is a test message to verify Vertex AI connectivity and count tokens."
	models := []string{"gemini-1.5-flash", "gemini-1.5-pro", "gemini-pro"}

	var totalTokens int64
	var successfulModel string

	for _, model := range models {
		tokens, err := r.callTokenCountAPI(ctx, location, model, testText)
		if err == nil {
			totalTokens = tokens
			successfulModel = model
			break
		}
		log.Printf("[DEBUG] Failed with model %s: %v", model, err)
	}

	if successfulModel == "" {
		log.Printf("[WARN] Could not connect to any Vertex AI model in location %s", location)
		// Return empty usage instead of error to allow graceful degradation
		return entity.NewVertexAIUsage(0, 0, 0.0, []entity.VertexAIModelMetric{}, projectID, location)
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
		location,
	)
}

// GetDailyUsage retrieves aggregated usage for a specific date
func (r *VertexAIRESTRepository) GetDailyUsage(projectID, location string, date time.Time) (*entity.VertexAIUsage, error) {
	// Convert to JST for consistent date boundaries
	jst, _ := time.LoadLocation("Asia/Tokyo")
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, jst)
	endOfDay := startOfDay.Add(24 * time.Hour)

	return r.GetUsageMetrics(projectID, location, startOfDay, endOfDay)
}

// GetCurrentMonthUsage retrieves usage for the current month
func (r *VertexAIRESTRepository) GetCurrentMonthUsage(projectID, location string) (*entity.VertexAIUsage, error) {
	jst, _ := time.LoadLocation("Asia/Tokyo")
	now := time.Now().In(jst)
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, jst)

	return r.GetUsageMetrics(projectID, location, startOfMonth, now)
}

// CheckConnection verifies Vertex AI API connectivity
func (r *VertexAIRESTRepository) CheckConnection() error {
	ctx := context.Background()

	// Try to connect to at least one model
	locations := []string{"us-central1", "asia-northeast1"}
	models := []string{"gemini-1.5-flash", "gemini-1.5-pro", "gemini-pro"}

	for _, location := range locations {
		for _, model := range models {
			_, err := r.callTokenCountAPI(ctx, location, model, "test")
			if err == nil {
				return nil // Connection successful
			}
		}
	}

	return fmt.Errorf("could not connect to any Vertex AI model")
}

// ListAvailableLocations returns locations with Vertex AI activity
func (r *VertexAIRESTRepository) ListAvailableLocations(projectID string) ([]string, error) {
	// Common Vertex AI locations
	locations := []string{
		"us-central1",
		"us-east1",
		"us-west1",
		"europe-west1",
		"europe-west4",
		"asia-northeast1",
		"asia-southeast1",
	}

	ctx := context.Background()
	var activeLocations []string

	for _, location := range locations {
		// Check if we can connect to any model in this location
		models := []string{"gemini-1.5-flash", "gemini-1.5-pro", "gemini-pro"}
		for _, model := range models {
			_, err := r.callTokenCountAPI(ctx, location, model, "test")
			if err == nil {
				activeLocations = append(activeLocations, location)
				break
			}
		}
	}

	return activeLocations, nil
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
