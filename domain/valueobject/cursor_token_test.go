package valueobject

import (
	"fmt"
	"testing"
	"time"
)

func TestNewCursorToken(t *testing.T) {
	tests := []struct {
		name      string
		jwtToken  string
		wantError bool
		errorMsg  string
		validate  func(t *testing.T, token *CursorToken)
	}{
		{
			name:      "empty token",
			jwtToken:  "",
			wantError: true,
			errorMsg:  "JWT token cannot be empty",
		},
		{
			name:      "invalid format - not enough parts",
			jwtToken:  "invalid.token",
			wantError: true,
			errorMsg:  "invalid JWT format",
		},
		{
			name:      "invalid format - too many parts",
			jwtToken:  "too.many.parts.here",
			wantError: true,
			errorMsg:  "invalid JWT format",
		},
		{
			name:     "valid token",
			jwtToken: createMockJWT("auth0|123456", time.Now().Add(time.Hour).Unix()),
			validate: func(t *testing.T, token *CursorToken) {
				if token == nil {
					t.Fatal("expected token, got nil")
				}
				if token.UserID() != "123456" {
					t.Errorf("expected user ID 123456, got %s", token.UserID())
				}
				if token.IsExpired() {
					t.Error("token should not be expired")
				}
				expectedSession := "123456%3A%3A" + token.jwtToken
				if token.SessionToken() != expectedSession {
					t.Errorf("expected session token %s, got %s", expectedSession, token.SessionToken())
				}
			},
		},
		{
			name:     "expired token",
			jwtToken: createMockJWT("auth0|789012", time.Now().Add(-time.Hour).Unix()),
			validate: func(t *testing.T, token *CursorToken) {
				if token == nil {
					t.Fatal("expected token, got nil")
				}
				if !token.IsExpired() {
					t.Error("token should be expired")
				}
			},
		},
		{
			name:      "invalid payload - missing sub",
			jwtToken:  createMockJWTWithPayload(`{"exp":1234567890}`),
			wantError: true,
			errorMsg:  "JWT payload missing 'sub' field",
		},
		{
			name:      "invalid sub format",
			jwtToken:  createMockJWTWithPayload(`{"sub":"invalid-format","exp":1234567890}`),
			wantError: true,
			errorMsg:  "invalid 'sub' format in JWT",
		},
		{
			name:      "malformed base64",
			jwtToken:  "header.!!!invalid-base64!!!.signature",
			wantError: true,
			errorMsg:  "failed to decode JWT payload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := NewCursorToken(tt.jwtToken)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errorMsg)
				} else if !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.validate != nil {
					tt.validate(t, token)
				}
			}
		})
	}
}

func TestCursorToken_Methods(t *testing.T) {
	futureTime := time.Now().Add(time.Hour)
	token, err := NewCursorToken(createMockJWT("auth0|user123", futureTime.Unix()))
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	t.Run("UserID", func(t *testing.T) {
		if got := token.UserID(); got != "user123" {
			t.Errorf("UserID() = %v, want %v", got, "user123")
		}
	})

	t.Run("SessionToken", func(t *testing.T) {
		expected := "user123%3A%3A" + token.jwtToken
		if got := token.SessionToken(); got != expected {
			t.Errorf("SessionToken() = %v, want %v", got, expected)
		}
	})

	t.Run("IsExpired", func(t *testing.T) {
		if token.IsExpired() {
			t.Error("IsExpired() = true, want false")
		}
	})

	t.Run("ExpiresAt", func(t *testing.T) {
		// Allow for small time differences
		diff := token.ExpiresAt().Sub(futureTime).Abs()
		if diff > time.Second {
			t.Errorf("ExpiresAt() differs by %v, expected less than 1 second", diff)
		}
	})
}

func TestDecodeBase64URL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "standard base64url",
			input:    "SGVsbG8gV29ybGQ",
			expected: "Hello World",
		},
		{
			name:     "with padding needed (2)",
			input:    "SGVsbG8gV29ybGQh",
			expected: "Hello World!",
		},
		{
			name:     "with padding needed (1)",
			input:    "SGVsbG8gV29ybGQhIQ",
			expected: "Hello World!!",
		},
		{
			name:     "url safe characters",
			input:    "SGVsbG8tV29ybGRfXw",
			expected: "Hello-World__",
		},
		{
			name:    "invalid characters",
			input:   "Hello@World",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decodeBase64URL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if string(result) != tt.expected {
					t.Errorf("got %q, want %q", string(result), tt.expected)
				}
			}
		})
	}
}

// Helper functions

func createMockJWT(sub string, exp int64) string {
	header := `{"alg":"HS256","typ":"JWT"}`
	payload := fmt.Sprintf(`{"sub":"%s","exp":%d}`, sub, exp)
	signature := "mock-signature"

	return encodeBase64URL([]byte(header)) + "." +
		encodeBase64URL([]byte(payload)) + "." +
		signature
}

func createMockJWTWithPayload(payload string) string {
	header := `{"alg":"HS256","typ":"JWT"}`
	signature := "mock-signature"

	return encodeBase64URL([]byte(header)) + "." +
		encodeBase64URL([]byte(payload)) + "." +
		signature
}

func encodeBase64URL(data []byte) string {
	// Simple base64url encoding
	const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

	result := ""
	for i := 0; i < len(data); i += 3 {
		b1, b2, b3 := uint(0), uint(0), uint(0)

		b1 = uint(data[i])
		if i+1 < len(data) {
			b2 = uint(data[i+1])
		}
		if i+2 < len(data) {
			b3 = uint(data[i+2])
		}

		result += string(base64Chars[(b1>>2)&0x3F])
		result += string(base64Chars[((b1<<4)|(b2>>4))&0x3F])
		if i+1 < len(data) {
			result += string(base64Chars[((b2<<2)|(b3>>6))&0x3F])
		}
		if i+2 < len(data) {
			result += string(base64Chars[b3&0x3F])
		}
	}

	// Remove padding
	for len(result) > 0 && result[len(result)-1] == '=' {
		result = result[:len(result)-1]
	}

	return result
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && contains(s, substr)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
