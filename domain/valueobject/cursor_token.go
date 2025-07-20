package valueobject

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// CursorToken represents a Cursor authentication token
type CursorToken struct {
	jwtToken     string
	sessionToken string
	userID       string
	expiresAt    time.Time
}

// JWTPayload represents the decoded JWT payload
type JWTPayload struct {
	Sub string `json:"sub"`
	Exp int64  `json:"exp"`
}

// NewCursorToken creates a new CursorToken from a JWT token string
func NewCursorToken(jwtToken string) (*CursorToken, error) {
	if jwtToken == "" {
		return nil, fmt.Errorf("JWT token cannot be empty")
	}

	// Decode JWT token (simplified version without signature verification)
	parts := strings.Split(jwtToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	// Decode payload (parts[1])
	payloadBytes, err := decodeBase64URL(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var payload JWTPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse JWT payload: %w", err)
	}

	if payload.Sub == "" {
		return nil, fmt.Errorf("JWT payload missing 'sub' field")
	}

	// Extract user ID from sub field (format: "auth0|123456")
	subParts := strings.Split(payload.Sub, "|")
	if len(subParts) != 2 {
		return nil, fmt.Errorf("invalid 'sub' format in JWT")
	}
	userID := subParts[1]

	// Create session token in the format: {userId}%3A%3A{jwt_token}
	sessionToken := fmt.Sprintf("%s%%3A%%3A%s", userID, jwtToken)

	// Parse expiration time
	expiresAt := time.Unix(payload.Exp, 0)

	return &CursorToken{
		jwtToken:     jwtToken,
		sessionToken: sessionToken,
		userID:       userID,
		expiresAt:    expiresAt,
	}, nil
}

// SessionToken returns the formatted session token for API requests
func (t *CursorToken) SessionToken() string {
	return t.sessionToken
}

// UserID returns the user ID extracted from the JWT
func (t *CursorToken) UserID() string {
	return t.userID
}

// IsExpired checks if the token has expired
func (t *CursorToken) IsExpired() bool {
	return time.Now().After(t.expiresAt)
}

// ExpiresAt returns the token expiration time
func (t *CursorToken) ExpiresAt() time.Time {
	return t.expiresAt
}

// decodeBase64URL decodes a base64url encoded string
func decodeBase64URL(s string) ([]byte, error) {
	// Add padding if necessary
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}

	// Replace URL-safe characters
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")

	// Use manual base64 decoding
	return manualBase64Decode(s)
}

// manualBase64Decode performs manual base64 decoding
func manualBase64Decode(s string) ([]byte, error) {
	const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	// Remove padding
	s = strings.TrimRight(s, "=")

	// Create decode map
	decodeMap := make(map[rune]int)
	for i, c := range base64Chars {
		decodeMap[c] = i
	}

	var result []byte
	var buffer int
	var bits int

	for _, c := range s {
		val, ok := decodeMap[c]
		if !ok {
			return nil, fmt.Errorf("invalid base64 character: %c", c)
		}

		buffer = (buffer << 6) | val
		bits += 6

		if bits >= 8 {
			bits -= 8
			result = append(result, byte(buffer>>bits))
			buffer &= (1 << bits) - 1
		}
	}

	return result, nil
}
