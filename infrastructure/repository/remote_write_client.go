package repository

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang/snappy"
)

// RemoteWriteClient handles sending metrics to Prometheus Remote Write endpoint
type RemoteWriteClient struct {
	url        string
	client     *http.Client
	authConfig *AuthConfig
}

// AuthConfig holds authentication configuration (basic auth only)
type AuthConfig struct {
	Username string
	Password string
}

// NewRemoteWriteClient creates a new Remote Write client
func NewRemoteWriteClient(url string, timeout time.Duration, authConfig *AuthConfig) (*RemoteWriteClient, error) {
	if url == "" {
		return nil, fmt.Errorf("remote write URL is required")
	}

	client := &http.Client{
		Timeout: timeout,
	}

	return &RemoteWriteClient{
		url:        url,
		client:     client,
		authConfig: authConfig,
	}, nil
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries: 3,
		BaseDelay:  time.Second,
		MaxDelay:   30 * time.Second,
	}
}

// SendGaugeMetric sends a gauge metric to the Remote Write endpoint with retry logic
// This implementation uses text format instead of protobuf for simplicity
func (c *RemoteWriteClient) SendGaugeMetric(ctx context.Context, metricName string, value float64, labels map[string]string) error {
	retryConfig := DefaultRetryConfig()

	var lastErr error
	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate exponential backoff delay
			multiplier := 1 << uint(attempt-1)
			delay := time.Duration(float64(retryConfig.BaseDelay) * float64(multiplier))
			if delay > retryConfig.MaxDelay {
				delay = retryConfig.MaxDelay
			}

			// Wait with context cancellation support
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
			}
		}

		err := c.sendGaugeMetricOnce(ctx, metricName, value, labels)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return err
		}
	}

	return fmt.Errorf("failed after %d retries: %w", retryConfig.MaxRetries, lastErr)
}

// sendGaugeMetricOnce sends a gauge metric once (without retry)
func (c *RemoteWriteClient) sendGaugeMetricOnce(ctx context.Context, metricName string, value float64, labels map[string]string) error {
	// Encode the write request using our custom protobuf encoder
	timestamp := time.Now().UnixMilli()
	data, err := encodeWriteRequest(metricName, value, labels, timestamp)
	if err != nil {
		return fmt.Errorf("failed to encode write request: %w", err)
	}

	// Compress with Snappy
	compressed := snappy.Encode(nil, data)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewReader(compressed))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers for protobuf format
	httpReq.Header.Set("Content-Type", "application/x-protobuf")
	httpReq.Header.Set("Content-Encoding", "snappy")
	httpReq.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

	// Add authentication
	if err := c.addAuthentication(httpReq); err != nil {
		return fmt.Errorf("failed to add authentication: %w", err)
	}

	// Send request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Read response body for error messages
	body, _ := io.ReadAll(resp.Body)

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		// Log password on 401 error for debugging
		if resp.StatusCode == http.StatusUnauthorized {
			password := os.Getenv("TOSAGE_PROMETHEUS_REMOTE_WRITE_PASSWORD")
			fmt.Fprintf(os.Stderr, "[AUTH DEBUG] 401 error occurred. TOSAGE_PROMETHEUS_REMOTE_WRITE_PASSWORD=%q\n", password)
		}
		return fmt.Errorf("remote write failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// addAuthentication adds authentication headers to the request
func (c *RemoteWriteClient) addAuthentication(req *http.Request) error {
	if c.authConfig == nil {
		return nil
	}

	// Always use basic authentication
	if c.authConfig.Username == "" || c.authConfig.Password == "" {
		return fmt.Errorf("basic auth requires username and password")
	}
	auth := base64.StdEncoding.EncodeToString([]byte(c.authConfig.Username + ":" + c.authConfig.Password))
	req.Header.Set("Authorization", "Basic "+auth)

	return nil
}

// isRetryableError determines if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for timeout errors
	if os.IsTimeout(err) {
		return true
	}

	// Check for specific error messages indicating server errors
	errMsg := err.Error()

	// Retry on 5xx status codes
	if strings.Contains(errMsg, "status 50") ||
		strings.Contains(errMsg, "status 502") ||
		strings.Contains(errMsg, "status 503") ||
		strings.Contains(errMsg, "status 504") {
		return true
	}

	// Don't retry on client errors (4xx)
	if strings.Contains(errMsg, "status 40") ||
		strings.Contains(errMsg, "status 401") ||
		strings.Contains(errMsg, "status 403") ||
		strings.Contains(errMsg, "status 404") {
		return false
	}

	// Retry on network errors
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "no such host") ||
		strings.Contains(errMsg, "timeout") {
		return true
	}

	// Default to not retrying
	return false
}
