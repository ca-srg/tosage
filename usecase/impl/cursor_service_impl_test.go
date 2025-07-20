package impl

import (
	"fmt"
	"testing"
	"time"

	"github.com/ca-srg/tosage/domain"
	"github.com/ca-srg/tosage/domain/entity"
	"github.com/ca-srg/tosage/domain/repository"
	"github.com/ca-srg/tosage/domain/valueobject"
	"github.com/ca-srg/tosage/infrastructure/config"
)

// Mock repositories for testing

type mockCursorTokenRepository struct {
	token *valueobject.CursorToken
	err   error
}

func (m *mockCursorTokenRepository) GetToken() (*valueobject.CursorToken, error) {
	return m.token, m.err
}

type mockCursorAPIRepository struct {
	usage     *entity.CursorUsage
	limit     *repository.UsageLimitInfo
	status    *repository.UsageBasedStatus
	usageErr  error
	limitErr  error
	statusErr error
	callCount map[string]int
}

func newMockCursorAPIRepository() *mockCursorAPIRepository {
	return &mockCursorAPIRepository{
		callCount: make(map[string]int),
	}
}

func (m *mockCursorAPIRepository) GetUsageStats(token *valueobject.CursorToken) (*entity.CursorUsage, error) {
	m.callCount["GetUsageStats"]++
	return m.usage, m.usageErr
}

func (m *mockCursorAPIRepository) GetUsageLimit(token *valueobject.CursorToken, teamID *int) (*repository.UsageLimitInfo, error) {
	m.callCount["GetUsageLimit"]++
	return m.limit, m.limitErr
}

func (m *mockCursorAPIRepository) CheckUsageBasedStatus(token *valueobject.CursorToken, teamID *int) (*repository.UsageBasedStatus, error) {
	m.callCount["CheckUsageBasedStatus"]++
	return m.status, m.statusErr
}

func (m *mockCursorAPIRepository) GetAggregatedTokenUsage(token *valueobject.CursorToken) (int64, error) {
	m.callCount["GetAggregatedTokenUsage"]++
	return 0, nil
}

// Test helper functions

func createTestToken(expired bool) *valueobject.CursorToken {
	// Create a mock JWT token for testing
	expTime := time.Now().Add(time.Hour)
	if expired {
		expTime = time.Now().Add(-time.Hour)
	}

	// This is a simplified mock - in real tests you'd use a proper JWT
	token, _ := valueobject.NewCursorToken(createMockJWT("auth0|testuser", expTime.Unix()))
	return token
}

func createMockJWT(sub string, exp int64) string {
	// Simplified JWT creation for testing
	header := `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9` // {"alg":"HS256","typ":"JWT"}
	payload := fmt.Sprintf(`{"sub":"%s","exp":%d}`, sub, exp)
	encodedPayload := encodeBase64([]byte(payload))
	signature := `SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c`

	return header + "." + encodedPayload + "." + signature
}

func encodeBase64(data []byte) string {
	// Simple base64 encoding for testing
	const base64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	result := ""

	for i := 0; i < len(data); i += 3 {
		b1, b2, b3 := 0, 0, 0

		if i < len(data) {
			b1 = int(data[i])
		}
		if i+1 < len(data) {
			b2 = int(data[i+1])
		}
		if i+2 < len(data) {
			b3 = int(data[i+2])
		}

		result += string(base64[(b1>>2)&0x3F])
		result += string(base64[((b1<<4)|(b2>>4))&0x3F])
		if i+1 < len(data) {
			result += string(base64[((b2<<2)|(b3>>6))&0x3F])
		} else {
			result += "="
		}
		if i+2 < len(data) {
			result += string(base64[b3&0x3F])
		} else {
			result += "="
		}
	}

	return result
}

func createTestUsage() *entity.CursorUsage {
	return entity.NewCursorUsage(
		entity.PremiumRequestsInfo{
			Current:      100,
			Limit:        500,
			StartOfMonth: "2024-01-01",
		},
		entity.UsageBasedPricingInfo{
			CurrentMonth: entity.MonthlyUsage{
				Month: 1,
				Year:  2024,
			},
			LastMonth: entity.MonthlyUsage{
				Month: 12,
				Year:  2023,
			},
		},
		&entity.TeamInfo{
			TeamID: 123,
			UserID: 456,
		},
	)
}

// Tests

func TestCursorServiceImpl_GetCurrentUsage(t *testing.T) {
	tests := []struct {
		name          string
		tokenRepo     *mockCursorTokenRepository
		apiRepo       *mockCursorAPIRepository
		config        *config.CursorConfig
		wantError     bool
		errorContains string
		validateCache bool
	}{
		{
			name: "successful retrieval",
			tokenRepo: &mockCursorTokenRepository{
				token: createTestToken(false),
			},
			apiRepo: func() *mockCursorAPIRepository {
				repo := newMockCursorAPIRepository()
				repo.usage = createTestUsage()
				return repo
			}(),
			config: &config.CursorConfig{
				CacheTimeout: 300,
			},
			wantError: false,
		},
		{
			name: "token repository error",
			tokenRepo: &mockCursorTokenRepository{
				err: domain.ErrCursorDatabase("GetToken", "/path/to/db"),
			},
			apiRepo: newMockCursorAPIRepository(),
			config: &config.CursorConfig{
				CacheTimeout: 300,
			},
			wantError:     true,
			errorContains: "failed to retrieve Cursor token",
		},
		{
			name: "expired token",
			tokenRepo: &mockCursorTokenRepository{
				token: createTestToken(true),
			},
			apiRepo: newMockCursorAPIRepository(),
			config: &config.CursorConfig{
				CacheTimeout: 300,
			},
			wantError:     true,
			errorContains: "token has expired",
		},
		{
			name: "api error",
			tokenRepo: &mockCursorTokenRepository{
				token: createTestToken(false),
			},
			apiRepo: func() *mockCursorAPIRepository {
				repo := newMockCursorAPIRepository()
				repo.usageErr = domain.ErrCursorAPI("GetUsageStats", 401, "Unauthorized")
				return repo
			}(),
			config: &config.CursorConfig{
				CacheTimeout: 300,
			},
			wantError:     true,
			errorContains: "failed to fetch usage statistics",
		},
		{
			name: "cached response",
			tokenRepo: &mockCursorTokenRepository{
				token: createTestToken(false),
			},
			apiRepo: func() *mockCursorAPIRepository {
				repo := newMockCursorAPIRepository()
				repo.usage = createTestUsage()
				return repo
			}(),
			config: &config.CursorConfig{
				CacheTimeout: 300,
			},
			wantError:     false,
			validateCache: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewCursorService(tt.tokenRepo, tt.apiRepo, tt.config)

			// First call
			usage, err := service.GetCurrentUsage()

			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if usage == nil {
					t.Error("expected usage, got nil")
				}
			}

			// Test cache behavior
			if tt.validateCache && !tt.wantError {
				// Reset call count
				initialCallCount := tt.apiRepo.callCount["GetUsageStats"]

				// Second call should use cache
				usage2, err2 := service.GetCurrentUsage()
				if err2 != nil {
					t.Errorf("unexpected error on cached call: %v", err2)
				}
				if usage2 == nil {
					t.Error("expected cached usage, got nil")
				}

				// Verify API was not called again
				if tt.apiRepo.callCount["GetUsageStats"] != initialCallCount {
					t.Error("expected cached response, but API was called again")
				}
			}
		})
	}
}

func TestCursorServiceImpl_GetUsageLimit(t *testing.T) {
	tests := []struct {
		name          string
		tokenRepo     *mockCursorTokenRepository
		apiRepo       *mockCursorAPIRepository
		config        *config.CursorConfig
		wantError     bool
		errorContains string
	}{
		{
			name: "successful retrieval - individual user",
			tokenRepo: &mockCursorTokenRepository{
				token: createTestToken(false),
			},
			apiRepo: func() *mockCursorAPIRepository {
				repo := newMockCursorAPIRepository()
				repo.usage = entity.NewCursorUsage(
					entity.PremiumRequestsInfo{},
					entity.UsageBasedPricingInfo{},
					nil, // No team info
				)
				repo.limit = &repository.UsageLimitInfo{
					HardLimit:           floatPtr(100.0),
					NoUsageBasedAllowed: false,
				}
				return repo
			}(),
			config: &config.CursorConfig{
				CacheTimeout: 300,
			},
			wantError: false,
		},
		{
			name: "successful retrieval - team member",
			tokenRepo: &mockCursorTokenRepository{
				token: createTestToken(false),
			},
			apiRepo: func() *mockCursorAPIRepository {
				repo := newMockCursorAPIRepository()
				repo.usage = createTestUsage() // Has team info
				repo.limit = &repository.UsageLimitInfo{
					HardLimit:           floatPtr(500.0),
					HardLimitPerUser:    floatPtr(100.0),
					NoUsageBasedAllowed: false,
				}
				return repo
			}(),
			config: &config.CursorConfig{
				CacheTimeout: 300,
			},
			wantError: false,
		},
		{
			name: "api error",
			tokenRepo: &mockCursorTokenRepository{
				token: createTestToken(false),
			},
			apiRepo: func() *mockCursorAPIRepository {
				repo := newMockCursorAPIRepository()
				repo.usage = createTestUsage()
				repo.limitErr = fmt.Errorf("network error")
				return repo
			}(),
			config: &config.CursorConfig{
				CacheTimeout: 300,
			},
			wantError:     true,
			errorContains: "failed to fetch usage limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewCursorService(tt.tokenRepo, tt.apiRepo, tt.config)

			limit, err := service.GetUsageLimit()

			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if limit == nil {
					t.Error("expected limit, got nil")
				}
			}
		})
	}
}

func TestCursorServiceImpl_IsUsageBasedPricingEnabled(t *testing.T) {
	tests := []struct {
		name          string
		tokenRepo     *mockCursorTokenRepository
		apiRepo       *mockCursorAPIRepository
		config        *config.CursorConfig
		wantEnabled   bool
		wantError     bool
		errorContains string
	}{
		{
			name: "usage-based pricing enabled",
			tokenRepo: &mockCursorTokenRepository{
				token: createTestToken(false),
			},
			apiRepo: func() *mockCursorAPIRepository {
				repo := newMockCursorAPIRepository()
				repo.usage = createTestUsage()
				repo.status = &repository.UsageBasedStatus{
					IsEnabled: true,
					Limit:     floatPtr(100.0),
				}
				return repo
			}(),
			config: &config.CursorConfig{
				CacheTimeout: 300,
			},
			wantEnabled: true,
			wantError:   false,
		},
		{
			name: "usage-based pricing disabled",
			tokenRepo: &mockCursorTokenRepository{
				token: createTestToken(false),
			},
			apiRepo: func() *mockCursorAPIRepository {
				repo := newMockCursorAPIRepository()
				repo.usage = createTestUsage()
				repo.status = &repository.UsageBasedStatus{
					IsEnabled: false,
				}
				return repo
			}(),
			config: &config.CursorConfig{
				CacheTimeout: 300,
			},
			wantEnabled: false,
			wantError:   false,
		},
		{
			name: "api error",
			tokenRepo: &mockCursorTokenRepository{
				token: createTestToken(false),
			},
			apiRepo: func() *mockCursorAPIRepository {
				repo := newMockCursorAPIRepository()
				repo.usage = createTestUsage()
				repo.statusErr = fmt.Errorf("api error")
				return repo
			}(),
			config: &config.CursorConfig{
				CacheTimeout: 300,
			},
			wantError:     true,
			errorContains: "failed to check usage-based pricing status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewCursorService(tt.tokenRepo, tt.apiRepo, tt.config)

			enabled, err := service.IsUsageBasedPricingEnabled()

			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if enabled != tt.wantEnabled {
					t.Errorf("expected enabled=%v, got %v", tt.wantEnabled, enabled)
				}
			}
		})
	}
}

func TestCursorServiceImpl_ClearCache(t *testing.T) {
	tokenRepo := &mockCursorTokenRepository{
		token: createTestToken(false),
	}
	apiRepo := newMockCursorAPIRepository()
	apiRepo.usage = createTestUsage()

	config := &config.CursorConfig{
		CacheTimeout: 300,
	}

	service := NewCursorService(tokenRepo, apiRepo, config).(*CursorServiceImpl)

	// First call to populate cache
	_, err := service.GetCurrentUsage()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	initialCallCount := apiRepo.callCount["GetUsageStats"]

	// Clear cache
	service.ClearCache()

	// Next call should hit API again
	_, err = service.GetCurrentUsage()
	if err != nil {
		t.Fatalf("unexpected error after cache clear: %v", err)
	}

	if apiRepo.callCount["GetUsageStats"] != initialCallCount+1 {
		t.Error("expected API call after cache clear")
	}
}

// Helper functions

func floatPtr(f float64) *float64 {
	return &f
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
