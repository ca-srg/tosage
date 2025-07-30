package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// VertexAIAuthenticator provides authentication for Vertex AI services
type VertexAIAuthenticator interface {
	// GetAccessToken returns a valid access token for Vertex AI
	GetAccessToken(ctx context.Context) (string, error)
	// ValidateCredentials validates the service account key structure
	ValidateCredentials() error
	// IsUsingADC returns true if the authenticator is using Application Default Credentials
	IsUsingADC() bool
}

// vertexAIAuthenticatorImpl implements VertexAIAuthenticator
type vertexAIAuthenticatorImpl struct {
	serviceAccountKey     string
	serviceAccountKeyPath string
	tokenSource           oauth2.TokenSource
}

// ServiceAccountKey represents the structure of a Google Cloud service account key
type ServiceAccountKey struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
}

// NewVertexAIAuthenticator creates a new Vertex AI authenticator
func NewVertexAIAuthenticator(serviceAccountKey, serviceAccountKeyPath string) (VertexAIAuthenticator, error) {
	auth := &vertexAIAuthenticatorImpl{
		serviceAccountKey:     serviceAccountKey,
		serviceAccountKeyPath: serviceAccountKeyPath,
	}

	// Initialize token source with three-tier priority system
	tokenSource, err := auth.createTokenSource(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to create token source: %w", err)
	}

	auth.tokenSource = tokenSource
	return auth, nil
}

// GetAccessToken returns a valid access token for Vertex AI
func (a *vertexAIAuthenticatorImpl) GetAccessToken(ctx context.Context) (string, error) {
	token, err := a.tokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	if !token.Valid() {
		return "", fmt.Errorf("token is invalid or expired")
	}

	return token.AccessToken, nil
}

// IsUsingADC returns true if the authenticator is using Application Default Credentials
func (a *vertexAIAuthenticatorImpl) IsUsingADC() bool {
	return a.serviceAccountKey == "" && a.serviceAccountKeyPath == ""
}

// ValidateCredentials validates the service account key structure
func (a *vertexAIAuthenticatorImpl) ValidateCredentials() error {
	if a.serviceAccountKey != "" {
		var key ServiceAccountKey
		if err := json.Unmarshal([]byte(a.serviceAccountKey), &key); err != nil {
			return fmt.Errorf("failed to parse service account key: %w", err)
		}

		if err := validateServiceAccountKey(&key); err != nil {
			return err
		}
	}

	return nil
}

// createTokenSource creates token source with three-tier priority system
func (a *vertexAIAuthenticatorImpl) createTokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	// Priority 1: Direct service account key JSON
	if a.serviceAccountKey != "" {
		return a.createTokenSourceFromJSON(ctx, a.serviceAccountKey)
	}

	// Priority 2: Service account key file path
	if a.serviceAccountKeyPath != "" {
		return a.createTokenSourceFromFile(ctx, a.serviceAccountKeyPath)
	}

	// Priority 3: Application default credentials
	return a.createDefaultTokenSource(ctx)
}

// createTokenSourceFromJSON creates token source from service account key JSON
func (a *vertexAIAuthenticatorImpl) createTokenSourceFromJSON(ctx context.Context, keyJSON string) (oauth2.TokenSource, error) {
	// Validate JSON structure first
	var key ServiceAccountKey
	if err := json.Unmarshal([]byte(keyJSON), &key); err != nil {
		return nil, fmt.Errorf("invalid service account key JSON: %w", err)
	}

	if err := validateServiceAccountKey(&key); err != nil {
		return nil, err
	}

	// Create credentials from JSON
	creds, err := google.CredentialsFromJSON(ctx, []byte(keyJSON), "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("failed to create credentials from JSON: %w", err)
	}

	return creds.TokenSource, nil
}

// createTokenSourceFromFile creates token source from service account key file
func (a *vertexAIAuthenticatorImpl) createTokenSourceFromFile(ctx context.Context, keyPath string) (oauth2.TokenSource, error) {
	// Read and validate file
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read service account key file: %w", err)
	}

	// Validate JSON structure
	var key ServiceAccountKey
	if err := json.Unmarshal(keyData, &key); err != nil {
		return nil, fmt.Errorf("invalid service account key file format: %w", err)
	}

	if err := validateServiceAccountKey(&key); err != nil {
		return nil, err
	}

	// Create credentials from file
	creds, err := google.CredentialsFromJSON(ctx, keyData, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("failed to create credentials from file: %w", err)
	}

	return creds.TokenSource, nil
}

// createDefaultTokenSource creates token source using application default credentials
func (a *vertexAIAuthenticatorImpl) createDefaultTokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	// Try to use application default credentials
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("failed to find default credentials: %w", err)
	}

	return creds.TokenSource, nil
}

// validateServiceAccountKey validates required fields in service account key
func validateServiceAccountKey(key *ServiceAccountKey) error {
	if key.Type != "service_account" {
		return fmt.Errorf("invalid service account type: %s (expected 'service_account')", key.Type)
	}

	if key.ProjectID == "" {
		return fmt.Errorf("service account key missing required field: project_id")
	}

	if key.PrivateKeyID == "" {
		return fmt.Errorf("service account key missing required field: private_key_id")
	}

	if key.PrivateKey == "" {
		return fmt.Errorf("service account key missing required field: private_key")
	}

	if key.ClientEmail == "" {
		return fmt.Errorf("service account key missing required field: client_email")
	}

	return nil
}
