package repository

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewRemoteWriteClient(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		timeout     time.Duration
		authConfig  *AuthConfig
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid configuration",
			url:     "http://localhost:9090/api/v1/write",
			timeout: 30 * time.Second,
			wantErr: false,
		},
		{
			name:        "empty URL",
			url:         "",
			timeout:     30 * time.Second,
			wantErr:     true,
			errContains: "remote write URL is required",
		},
		{
			name:    "with basic auth",
			url:     "http://localhost:9090/api/v1/write",
			timeout: 30 * time.Second,
			authConfig: &AuthConfig{
				Username: "user",
				Password: "pass",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewRemoteWriteClient(tt.url, tt.timeout, tt.authConfig)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if client == nil {
					t.Errorf("expected client but got nil")
				}
			}
		})
	}
}

func TestSendGaugeMetric(t *testing.T) {
	tests := []struct {
		name           string
		metricName     string
		value          float64
		labels         map[string]string
		serverResponse int
		serverError    bool
		wantErr        bool
		errContains    string
		checkRetry     bool
		retryCount     int
	}{
		{
			name:           "successful send",
			metricName:     "test_metric",
			value:          42.0,
			labels:         map[string]string{"host": "test-host"},
			serverResponse: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "server error with retry",
			metricName:     "test_metric",
			value:          42.0,
			labels:         map[string]string{"host": "test-host"},
			serverResponse: http.StatusInternalServerError,
			wantErr:        true,
			errContains:    "status 500",
			checkRetry:     true,
			retryCount:     4, // initial + 3 retries
		},
		{
			name:           "client error no retry",
			metricName:     "test_metric",
			value:          42.0,
			labels:         map[string]string{"host": "test-host"},
			serverResponse: http.StatusBadRequest,
			wantErr:        true,
			errContains:    "status 400",
			checkRetry:     true,
			retryCount:     1, // no retry for client errors
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++

				// Check headers
				if r.Header.Get("Content-Type") != "application/x-protobuf" {
					t.Errorf("unexpected Content-Type: %s", r.Header.Get("Content-Type"))
				}
				if r.Header.Get("Content-Encoding") != "snappy" {
					t.Errorf("unexpected Content-Encoding: %s", r.Header.Get("Content-Encoding"))
				}

				if tt.serverError {
					http.Error(w, "server error", tt.serverResponse)
					return
				}

				w.WriteHeader(tt.serverResponse)
			}))
			defer server.Close()

			client, err := NewRemoteWriteClient(server.URL, 5*time.Second, nil)
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			ctx := context.Background()
			err = client.SendGaugeMetric(ctx, tt.metricName, tt.value, tt.labels)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			if tt.checkRetry && requestCount != tt.retryCount {
				t.Errorf("expected %d requests, got %d", tt.retryCount, requestCount)
			}
		})
	}
}

func TestAddAuthentication(t *testing.T) {
	tests := []struct {
		name        string
		authConfig  *AuthConfig
		wantHeader  string
		wantValue   string
		wantErr     bool
		errContains string
	}{
		{
			name:       "no authentication",
			authConfig: nil,
			wantErr:    false,
		},
		{
			name: "basic auth valid",
			authConfig: &AuthConfig{
				Username: "user",
				Password: "pass",
			},
			wantHeader: "Authorization",
			wantValue:  "Basic dXNlcjpwYXNz", // base64("user:pass")
			wantErr:    false,
		},
		{
			name: "basic auth missing username",
			authConfig: &AuthConfig{
				Password: "pass",
			},
			wantErr:     true,
			errContains: "basic auth requires username and password",
		},
		{
			name: "basic auth missing password",
			authConfig: &AuthConfig{
				Username: "user",
			},
			wantErr:     true,
			errContains: "basic auth requires username and password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &RemoteWriteClient{
				authConfig: tt.authConfig,
			}

			req, _ := http.NewRequest("POST", "http://example.com", nil)
			err := client.addAuthentication(req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.wantHeader != "" {
					got := req.Header.Get(tt.wantHeader)
					if got != tt.wantValue {
						t.Errorf("header %s = %v, want %v", tt.wantHeader, got, tt.wantValue)
					}
				}
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "500 server error",
			err:  fmt.Errorf("remote write failed with status 500: server error"),
			want: true,
		},
		{
			name: "502 bad gateway",
			err:  fmt.Errorf("remote write failed with status 502: bad gateway"),
			want: true,
		},
		{
			name: "503 service unavailable",
			err:  fmt.Errorf("remote write failed with status 503: service unavailable"),
			want: true,
		},
		{
			name: "504 gateway timeout",
			err:  fmt.Errorf("remote write failed with status 504: gateway timeout"),
			want: true,
		},
		{
			name: "400 bad request",
			err:  fmt.Errorf("remote write failed with status 400: bad request"),
			want: false,
		},
		{
			name: "401 unauthorized",
			err:  fmt.Errorf("remote write failed with status 401: unauthorized"),
			want: false,
		},
		{
			name: "403 forbidden",
			err:  fmt.Errorf("remote write failed with status 403: forbidden"),
			want: false,
		},
		{
			name: "404 not found",
			err:  fmt.Errorf("remote write failed with status 404: not found"),
			want: false,
		},
		{
			name: "connection refused",
			err:  fmt.Errorf("connection refused"),
			want: true,
		},
		{
			name: "no such host",
			err:  fmt.Errorf("no such host"),
			want: true,
		},
		{
			name: "timeout error",
			err:  fmt.Errorf("request timeout"),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRetryableError(tt.err)
			if got != tt.want {
				t.Errorf("isRetryableError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSendGaugeMetric_401Error(t *testing.T) {
	// Capture stderr to check the debug output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Set test password in environment
	testPassword := "test-password-123"
	os.Setenv("TOSAGE_PROMETHEUS_REMOTE_WRITE_PASSWORD", testPassword)
	defer os.Unsetenv("TOSAGE_PROMETHEUS_REMOTE_WRITE_PASSWORD")

	// Create test server that returns 401
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status":"error","error":"authentication error: invalid token"}`))
	}))
	defer server.Close()

	// Create client with auth
	client, _ := NewRemoteWriteClient(server.URL, 30*time.Second, &AuthConfig{
		Username: "user",
		Password: "wrong-password",
	})

	// Send metric (should fail with 401)
	ctx := context.Background()
	err := client.SendGaugeMetric(ctx, "test_metric", 100, map[string]string{"test": "label"})

	// Check error
	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should contain 401: %v", err)
	}

	// Close writer and restore stderr
	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check that password was logged
	if !strings.Contains(output, "[AUTH DEBUG]") {
		t.Error("expected [AUTH DEBUG] in stderr output")
	}
	if !strings.Contains(output, "401 error occurred") {
		t.Error("expected '401 error occurred' in stderr output")
	}
	if !strings.Contains(output, testPassword) {
		t.Errorf("expected password %q in stderr output, got: %s", testPassword, output)
	}
}
