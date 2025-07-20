package entity

import (
	"testing"
)

func TestNewCursorUsage(t *testing.T) {
	premiumRequests := PremiumRequestsInfo{
		Current:      100,
		Limit:        500,
		StartOfMonth: "2024-01-01",
	}

	usageBasedPricing := UsageBasedPricingInfo{
		CurrentMonth: MonthlyUsage{
			Month: 1,
			Year:  2024,
			Items: []UsageItem{
				{
					RequestCount:   10,
					Model:          "gpt-4",
					CostPerRequest: 0.03,
					TotalCost:      0.30,
					Description:    "10 gpt-4 requests",
				},
			},
			MidMonthPayment:  5.00,
			HasUnpaidInvoice: false,
		},
		LastMonth: MonthlyUsage{
			Month: 12,
			Year:  2023,
			Items: []UsageItem{
				{
					RequestCount:   20,
					Model:          "claude-3-opus",
					CostPerRequest: 0.015,
					TotalCost:      0.30,
					Description:    "20 claude-3-opus requests",
				},
			},
		},
	}

	teamInfo := &TeamInfo{
		TeamID:   123,
		UserID:   456,
		TeamName: "Test Team",
		Role:     "member",
	}

	usage := NewCursorUsage(premiumRequests, usageBasedPricing, teamInfo)

	if usage == nil {
		t.Fatal("expected usage to be created, got nil")
	}

	if usage.PremiumRequests().Current != premiumRequests.Current {
		t.Errorf("expected premium requests current %d, got %d",
			premiumRequests.Current, usage.PremiumRequests().Current)
	}

	if !usage.IsTeamMember() {
		t.Error("expected user to be team member")
	}

	if usage.TeamInfo().TeamID != teamInfo.TeamID {
		t.Errorf("expected team ID %d, got %d", teamInfo.TeamID, usage.TeamInfo().TeamID)
	}

	if usage.Timestamp().IsZero() {
		t.Error("expected timestamp to be set")
	}
}

func TestCursorUsage_PremiumRequestsPercentage(t *testing.T) {
	tests := []struct {
		name     string
		current  int
		limit    int
		expected float64
	}{
		{
			name:     "normal usage",
			current:  100,
			limit:    500,
			expected: 20.0,
		},
		{
			name:     "zero usage",
			current:  0,
			limit:    500,
			expected: 0.0,
		},
		{
			name:     "full usage",
			current:  500,
			limit:    500,
			expected: 100.0,
		},
		{
			name:     "over limit",
			current:  600,
			limit:    500,
			expected: 120.0,
		},
		{
			name:     "zero limit",
			current:  100,
			limit:    0,
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage := NewCursorUsage(
				PremiumRequestsInfo{
					Current: tt.current,
					Limit:   tt.limit,
				},
				UsageBasedPricingInfo{},
				nil,
			)

			got := usage.PremiumRequestsPercentage()
			if got != tt.expected {
				t.Errorf("expected percentage %.2f, got %.2f", tt.expected, got)
			}
		})
	}
}

func TestCursorUsage_CurrentMonthTotalCost(t *testing.T) {
	usage := NewCursorUsage(
		PremiumRequestsInfo{},
		UsageBasedPricingInfo{
			CurrentMonth: MonthlyUsage{
				Month: 1,
				Year:  2024,
				Items: []UsageItem{
					{TotalCost: 10.50},
					{TotalCost: 5.25},
					{TotalCost: 3.75},
				},
				MidMonthPayment: 5.00,
			},
		},
		nil,
	)

	expected := 10.50 + 5.25 + 3.75 - 5.00 // Total minus mid-month payment
	got := usage.CurrentMonthTotalCost()

	if got != expected {
		t.Errorf("expected total cost %.2f, got %.2f", expected, got)
	}
}

func TestCursorUsage_Validate(t *testing.T) {
	tests := []struct {
		name      string
		usage     *CursorUsage
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid usage",
			usage: NewCursorUsage(
				PremiumRequestsInfo{
					Current:      100,
					Limit:        500,
					StartOfMonth: "2024-01-01",
				},
				UsageBasedPricingInfo{
					CurrentMonth: MonthlyUsage{
						Month: 1,
						Year:  2024,
						Items: []UsageItem{
							{
								RequestCount:   10,
								Model:          "gpt-4",
								CostPerRequest: 0.03,
								TotalCost:      0.30,
							},
						},
					},
					LastMonth: MonthlyUsage{
						Month: 12,
						Year:  2023,
					},
				},
				nil,
			),
			wantError: false,
		},
		{
			name: "negative premium requests current",
			usage: NewCursorUsage(
				PremiumRequestsInfo{
					Current: -1,
					Limit:   500,
				},
				UsageBasedPricingInfo{
					CurrentMonth: MonthlyUsage{Month: 1, Year: 2024},
					LastMonth:    MonthlyUsage{Month: 12, Year: 2023},
				},
				nil,
			),
			wantError: true,
			errorMsg:  "premium requests current cannot be negative",
		},
		{
			name: "current exceeds limit",
			usage: NewCursorUsage(
				PremiumRequestsInfo{
					Current: 600,
					Limit:   500,
				},
				UsageBasedPricingInfo{
					CurrentMonth: MonthlyUsage{Month: 1, Year: 2024},
					LastMonth:    MonthlyUsage{Month: 12, Year: 2023},
				},
				nil,
			),
			wantError: true,
			errorMsg:  "premium requests current (600) exceeds limit (500)",
		},
		{
			name: "invalid month",
			usage: NewCursorUsage(
				PremiumRequestsInfo{},
				UsageBasedPricingInfo{
					CurrentMonth: MonthlyUsage{Month: 13, Year: 2024},
					LastMonth:    MonthlyUsage{Month: 12, Year: 2023},
				},
				nil,
			),
			wantError: true,
			errorMsg:  "invalid month: 13",
		},
		{
			name: "invalid year",
			usage: NewCursorUsage(
				PremiumRequestsInfo{},
				UsageBasedPricingInfo{
					CurrentMonth: MonthlyUsage{Month: 1, Year: 2019},
					LastMonth:    MonthlyUsage{Month: 12, Year: 2023},
				},
				nil,
			),
			wantError: true,
			errorMsg:  "invalid year: 2019",
		},
		{
			name: "negative mid-month payment",
			usage: NewCursorUsage(
				PremiumRequestsInfo{},
				UsageBasedPricingInfo{
					CurrentMonth: MonthlyUsage{
						Month:           1,
						Year:            2024,
						MidMonthPayment: -10.0,
					},
					LastMonth: MonthlyUsage{Month: 12, Year: 2023},
				},
				nil,
			),
			wantError: true,
			errorMsg:  "mid-month payment cannot be negative",
		},
		{
			name: "negative request count in item",
			usage: NewCursorUsage(
				PremiumRequestsInfo{},
				UsageBasedPricingInfo{
					CurrentMonth: MonthlyUsage{
						Month: 1,
						Year:  2024,
						Items: []UsageItem{
							{RequestCount: -1},
						},
					},
					LastMonth: MonthlyUsage{Month: 12, Year: 2023},
				},
				nil,
			),
			wantError: true,
			errorMsg:  "request count cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.usage.Validate()

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
			}
		})
	}
}

func TestCursorUsage_NonTeamMember(t *testing.T) {
	usage := NewCursorUsage(
		PremiumRequestsInfo{},
		UsageBasedPricingInfo{},
		nil, // No team info
	)

	if usage.IsTeamMember() {
		t.Error("expected user not to be team member")
	}

	if usage.TeamInfo() != nil {
		t.Error("expected team info to be nil")
	}
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
