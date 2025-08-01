package repository

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ca-srg/tosage/domain"
	"github.com/ca-srg/tosage/domain/entity"
	"github.com/ca-srg/tosage/domain/repository"
	"github.com/ca-srg/tosage/domain/valueobject"
)

// CursorAPIRepository implements the repository.CursorAPIRepository interface
type CursorAPIRepository struct {
	httpClient *http.Client
	baseURL    string
}

// NewCursorAPIRepository creates a new CursorAPIRepository instance
func NewCursorAPIRepository(timeout time.Duration) repository.CursorAPIRepository {
	return &CursorAPIRepository{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: "https://cursor.com",
	}
}

// API response structures

type usageResponse struct {
	GPT4 struct {
		NumRequests     int `json:"numRequests"`
		MaxRequestUsage int `json:"maxRequestUsage"`
		NumTokens       int `json:"numTokens"`
	} `json:"gpt-4"`
	GPT432K struct {
		NumRequests     int  `json:"numRequests"`
		MaxRequestUsage *int `json:"maxRequestUsage"`
		NumTokens       int  `json:"numTokens"`
	} `json:"gpt-4-32k"`
	StartOfMonth string `json:"startOfMonth"`
}

type teamResponse struct {
	Teams []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Role string `json:"role"`
	} `json:"teams"`
}

type teamMemberResponse struct {
	UserID      int `json:"userId"`
	TeamMembers []struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
		Role  string `json:"role"`
	} `json:"teamMembers"`
}

type teamSpendResponse struct {
	TeamMemberSpend []struct {
		UserID                   int     `json:"userId"`
		Name                     string  `json:"name"`
		Email                    string  `json:"email"`
		Role                     string  `json:"role"`
		HardLimitOverrideDollars float64 `json:"hardLimitOverrideDollars"`
		FastPremiumRequests      int     `json:"fastPremiumRequests"`
	} `json:"teamMemberSpend"`
}

type monthlyInvoiceResponse struct {
	Items []struct {
		Description string `json:"description"`
		Cents       *int   `json:"cents"`
	} `json:"items"`
	HasUnpaidMidMonthInvoice bool `json:"hasUnpaidMidMonthInvoice"`
}

type filteredUsageEventsResponse struct {
	TotalUsageEventsCount int `json:"totalUsageEventsCount"`
	UsageEventsDisplay    []struct {
		Timestamp        string  `json:"timestamp"`
		Model            string  `json:"model"`
		Kind             string  `json:"kind"`
		MaxMode          bool    `json:"maxMode"`
		RequestsCosts    float64 `json:"requestsCosts"`
		UsageBasedCosts  string  `json:"usageBasedCosts"`
		IsTokenBasedCall bool    `json:"isTokenBasedCall"`
		TokenUsage       struct {
			InputTokens      int     `json:"inputTokens"`
			OutputTokens     int     `json:"outputTokens"`
			CacheWriteTokens int     `json:"cacheWriteTokens"`
			CacheReadTokens  int     `json:"cacheReadTokens"`
			TotalCents       float64 `json:"totalCents"`
		} `json:"tokenUsage"`
		OwningUser string `json:"owningUser"`
		OwningTeam string `json:"owningTeam"`
	} `json:"usageEventsDisplay"`
}

type hardLimitResponse struct {
	HardLimit           *float64 `json:"hardLimit"`
	HardLimitPerUser    *float64 `json:"hardLimitPerUser"`
	NoUsageBasedAllowed bool     `json:"noUsageBasedAllowed"`
}

type usageBasedStatusResponse struct {
	UsageBasedPremiumRequests bool `json:"usageBasedPremiumRequests"`
}

// GetUsageStats retrieves current usage statistics from the Cursor API
func (r *CursorAPIRepository) GetUsageStats(token *valueobject.CursorToken) (*entity.CursorUsage, error) {

	// Check team membership
	teamInfo, err := r.checkTeamMembership(token)
	if err != nil {
		// Continue without team info
		teamInfo = nil
	}

	// Get premium requests data
	premiumRequests, err := r.getPremiumRequests(token, teamInfo)
	if err != nil {
		return nil, err
	}

	// Get usage-based pricing data
	usageBasedPricing, err := r.getUsageBasedPricing(token)
	if err != nil {
		return nil, err
	}

	// Create and return CursorUsage entity
	return entity.NewCursorUsage(premiumRequests, usageBasedPricing, teamInfo), nil
}

// GetUsageLimit retrieves the current usage limit settings
func (r *CursorAPIRepository) GetUsageLimit(token *valueobject.CursorToken, teamID *int) (*repository.UsageLimitInfo, error) {
	payload := make(map[string]interface{})
	if teamID != nil {
		payload["teamId"] = *teamID
	}

	resp, err := r.makeAPIRequest(token, "POST", "/api/dashboard/get-hard-limit", payload)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var result hardLimitResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, domain.ErrCursorAPIWithCause("decode usage limit response", err)
	}

	return &repository.UsageLimitInfo{
		HardLimit:           result.HardLimit,
		HardLimitPerUser:    result.HardLimitPerUser,
		NoUsageBasedAllowed: result.NoUsageBasedAllowed,
	}, nil
}

// CheckUsageBasedStatus checks if usage-based pricing is enabled
func (r *CursorAPIRepository) CheckUsageBasedStatus(token *valueobject.CursorToken, teamID *int) (*repository.UsageBasedStatus, error) {
	payload := make(map[string]interface{})
	if teamID != nil {
		payload["teamId"] = *teamID
	}

	resp, err := r.makeAPIRequest(token, "POST", "/api/dashboard/get-usage-based-premium-requests", payload)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var statusResp usageBasedStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return nil, domain.ErrCursorAPIWithCause("decode usage-based status", err)
	}

	// Get hard limit to determine spending limit
	limitInfo, err := r.GetUsageLimit(token, teamID)
	if err != nil {
		return nil, err
	}

	return &repository.UsageBasedStatus{
		IsEnabled: statusResp.UsageBasedPremiumRequests,
		Limit:     limitInfo.HardLimit,
	}, nil
}

// checkTeamMembership checks if the user is a team member
func (r *CursorAPIRepository) checkTeamMembership(token *valueobject.CursorToken) (*entity.TeamInfo, error) {
	
	// Get team list - send empty JSON object
	resp, err := r.makeAPIRequest(token, "POST", "/api/dashboard/teams", map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var teams teamResponse
	if err := json.NewDecoder(resp.Body).Decode(&teams); err != nil {
		return nil, domain.ErrCursorAPIWithCause("decode teams response", err)
	}
	

	if len(teams.Teams) == 0 {
		return nil, nil // Not a team member
	}

	// Get team details
	teamID := teams.Teams[0].ID
	resp, err = r.makeAPIRequest(token, "POST", "/api/dashboard/team", map[string]interface{}{
		"teamId": teamID,
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var teamDetails teamMemberResponse
	if err := json.NewDecoder(resp.Body).Decode(&teamDetails); err != nil {
		return nil, domain.ErrCursorAPIWithCause("decode team details", err)
	}
	

	return &entity.TeamInfo{
		TeamID:   teamID,
		UserID:   teamDetails.UserID,
		TeamName: teams.Teams[0].Name,
		Role:     teams.Teams[0].Role,
	}, nil
}

// getPremiumRequests gets premium request usage data
func (r *CursorAPIRepository) getPremiumRequests(token *valueobject.CursorToken, teamInfo *entity.TeamInfo) (entity.PremiumRequestsInfo, error) {
	userID := token.UserID()

	// Always get individual usage data first (for both team members and individual users)
	individualUsage, err := r.getIndividualUsage(token, userID)
	if err != nil {
		return entity.PremiumRequestsInfo{}, err
	}

	// Use default limit of 500 if maxRequestUsage is 0 (null in JSON)
	limit := individualUsage.GPT4.MaxRequestUsage
	if limit == 0 {
		limit = 500 // Default premium request limit
	}

	// For team members, try to get additional team data for validation
	// but always use the individual usage data for current request count
	if teamInfo != nil && teamInfo.TeamID > 0 {
		resp, err := r.makeAPIRequest(token, "POST", "/api/dashboard/get-team-spend", map[string]interface{}{
			"teamId": teamInfo.TeamID,
		})
		if err == nil {
			defer func() {
				_ = resp.Body.Close()
			}()

			var teamSpend teamSpendResponse
			if err := json.NewDecoder(resp.Body).Decode(&teamSpend); err == nil {
				// Log team spend data for debugging but use individual usage
				for _, member := range teamSpend.TeamMemberSpend {
					if member.UserID == teamInfo.UserID {
						// Team spend provides fastPremiumRequests but individual API
						// provides more accurate real-time GPT-4 usage
						break
					}
				}
			}
		}
	}

	return entity.PremiumRequestsInfo{
		Current:      individualUsage.GPT4.NumRequests,
		Limit:        limit,
		StartOfMonth: individualUsage.StartOfMonth,
	}, nil
}

// getIndividualUsage gets individual usage data
func (r *CursorAPIRepository) getIndividualUsage(token *valueobject.CursorToken, userID string) (*usageResponse, error) {
	req, err := http.NewRequest("GET", r.baseURL+"/api/usage?user="+userID, nil)
	if err != nil {
		return nil, domain.ErrCursorAPIWithCause("create usage request", err)
	}

	req.Header.Set("Cookie", fmt.Sprintf("WorkosCursorSessionToken=%s", token.SessionToken()))

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, domain.ErrCursorAPIWithCause("execute usage request", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, domain.ErrCursorAPI("get individual usage", resp.StatusCode, string(body))
	}

	var usage usageResponse
	if err := json.NewDecoder(resp.Body).Decode(&usage); err != nil {
		return nil, domain.ErrCursorAPIWithCause("decode usage response", err)
	}

	return &usage, nil
}

// getUsageBasedPricing gets usage-based pricing data for current and last month
func (r *CursorAPIRepository) getUsageBasedPricing(token *valueobject.CursorToken) (entity.UsageBasedPricingInfo, error) {
	now := time.Now()
	billingDay := 3

	// Calculate current billing month
	currentMonth := int(now.Month())
	currentYear := now.Year()
	if now.Day() < billingDay {
		currentMonth--
		if currentMonth == 0 {
			currentMonth = 12
			currentYear--
		}
	}

	// Calculate last billing month
	lastMonth := currentMonth - 1
	lastYear := currentYear
	if lastMonth == 0 {
		lastMonth = 12
		lastYear--
	}

	// Fetch current month data
	currentMonthData, err := r.fetchMonthlyInvoice(token, currentMonth, currentYear)
	if err != nil {
		return entity.UsageBasedPricingInfo{}, err
	}

	// Fetch last month data
	lastMonthData, err := r.fetchMonthlyInvoice(token, lastMonth, lastYear)
	if err != nil {
		return entity.UsageBasedPricingInfo{}, err
	}

	return entity.UsageBasedPricingInfo{
		CurrentMonth: currentMonthData,
		LastMonth:    lastMonthData,
	}, nil
}

// fetchMonthlyInvoice fetches invoice data for a specific month
func (r *CursorAPIRepository) fetchMonthlyInvoice(token *valueobject.CursorToken, month, year int) (entity.MonthlyUsage, error) {
	resp, err := r.makeAPIRequest(token, "POST", "/api/dashboard/get-monthly-invoice", map[string]interface{}{
		"month":              month,
		"year":               year,
		"includeUsageEvents": false,
	})
	if err != nil {
		return entity.MonthlyUsage{}, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var invoice monthlyInvoiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&invoice); err != nil {
		return entity.MonthlyUsage{}, domain.ErrCursorAPIWithCause("decode monthly invoice", err)
	}

	// Parse invoice items
	var usageItems []entity.UsageItem
	var midMonthPayment float64

	for _, item := range invoice.Items {
		if item.Cents == nil {
			continue
		}

		// Check for mid-month payment
		if strings.Contains(item.Description, "Mid-month usage paid") {
			midMonthPayment += float64(*item.Cents) / 100.0
			continue
		}

		// Parse regular usage items
		usageItem := r.parseInvoiceItem(item)
		if usageItem != nil {
			usageItems = append(usageItems, *usageItem)
		}
	}

	return entity.MonthlyUsage{
		Month:            month,
		Year:             year,
		Items:            usageItems,
		MidMonthPayment:  midMonthPayment,
		HasUnpaidInvoice: invoice.HasUnpaidMidMonthInvoice,
	}, nil
}

// parseInvoiceItem parses a single invoice item
func (r *CursorAPIRepository) parseInvoiceItem(item struct {
	Description string `json:"description"`
	Cents       *int   `json:"cents"`
}) *entity.UsageItem {
	if item.Cents == nil || *item.Cents == 0 {
		return nil
	}

	// Extract request count and model from description
	var requestCount int
	var model string
	var isToolCall bool
	var isDiscounted bool

	// Check for different description patterns
	if strings.Contains(item.Description, "token-based usage calls to") {
		// Pattern: "123 token-based usage calls to claude-3-opus, totalling: $12.34"
		_, _ = fmt.Sscanf(item.Description, "%d token-based usage calls to %s", &requestCount, &model)
		model = strings.TrimSuffix(model, ",")
	} else if strings.Contains(item.Description, "tool calls") {
		// Pattern: "123 tool calls"
		_, _ = fmt.Sscanf(item.Description, "%d tool calls", &requestCount)
		model = "Tool Calls"
		isToolCall = true
	} else if strings.Contains(item.Description, "extra fast premium request") {
		// Pattern: "123 extra fast premium requests (Haiku)"
		_, _ = fmt.Sscanf(item.Description, "%d extra fast premium request", &requestCount)
		if match := strings.Index(item.Description, "("); match != -1 {
			end := strings.Index(item.Description[match:], ")")
			if end != -1 {
				model = item.Description[match+1 : match+end]
			}
		}
		if model == "" {
			model = "Fast Premium"
		}
	} else {
		// Try to extract number and model from generic pattern
		parts := strings.Fields(item.Description)
		if len(parts) > 0 {
			requestCount, _ = strconv.Atoi(parts[0])
		}

		// Extract model name using regex-like pattern matching
		modelPatterns := []string{
			"claude-", "gpt-", "gemini-", "o1", "o3", "o4",
		}
		for _, pattern := range modelPatterns {
			if idx := strings.Index(strings.ToLower(item.Description), pattern); idx != -1 {
				// Find the end of the model name
				endIdx := idx + len(pattern)
				for endIdx < len(item.Description) && (isAlphaNumeric(item.Description[endIdx]) || item.Description[endIdx] == '-' || item.Description[endIdx] == '.') {
					endIdx++
				}
				model = item.Description[idx:endIdx]
				break
			}
		}

		if model == "" {
			model = "Unknown Model"
		}
	}

	// Check if discounted
	if strings.Contains(strings.ToLower(item.Description), "discounted") {
		isDiscounted = true
	}

	if requestCount == 0 {
		return nil
	}

	totalCost := float64(*item.Cents) / 100.0
	costPerRequest := totalCost / float64(requestCount)

	return &entity.UsageItem{
		RequestCount:   requestCount,
		Model:          model,
		CostPerRequest: costPerRequest,
		TotalCost:      totalCost,
		Description:    item.Description,
		IsDiscounted:   isDiscounted,
		IsToolCall:     isToolCall,
	}
}

// makeAPIRequest makes a request to the Cursor API
func (r *CursorAPIRepository) makeAPIRequest(token *valueobject.CursorToken, method, path string, payload interface{}) (*http.Response, error) {
	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, domain.ErrCursorAPIWithCause("marshal request payload", err)
		}
		body = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, r.baseURL+path, body)
	if err != nil {
		return nil, domain.ErrCursorAPIWithCause("create request", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Cookie", fmt.Sprintf("WorkosCursorSessionToken=%s", token.SessionToken()))
	
	// Add Origin and Referer headers to pass CSRF check
	req.Header.Set("Origin", "https://cursor.com")
	req.Header.Set("Referer", "https://cursor.com/")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, domain.ErrCursorAPIWithCause("execute request", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, domain.ErrCursorAPI(path, resp.StatusCode, string(body))
	}

	return resp, nil
}

// isAlphaNumeric checks if a byte is alphanumeric
func isAlphaNumeric(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// GetAggregatedTokenUsage retrieves aggregated token usage from 00:00 to current time in the machine's timezone
func (r *CursorAPIRepository) GetAggregatedTokenUsage(token *valueobject.CursorToken) (int64, error) {
	// Get current time in the machine's local timezone
	now := time.Now()

	// Calculate 00:00 today in the local timezone
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Convert to milliseconds for API
	startDate := startOfDay.UnixMilli()
	endDate := now.UnixMilli()
	

	// Check if user is a team member
	teamInfo, err := r.checkTeamMembership(token)
	if err != nil {
		// If team check fails, return 0 (not an error)
		return 0, nil
	}

	// If not a team member, return 0
	if teamInfo == nil || teamInfo.TeamID == 0 {
		return 0, nil
	}
	

	// Create request payload
	payload := map[string]interface{}{
		"teamId":    teamInfo.TeamID,
		"startDate": strconv.FormatInt(startDate, 10),
		"endDate":   strconv.FormatInt(endDate, 10),
		"userId":    teamInfo.UserID,
		"page":      1,
		"pageSize":  100,
	}
	

	totalTokens := int64(0)
	page := 1
	totalEvents := 0
	eventsWithTokens := 0

	// Paginate through all results
	for {
		payload["page"] = page

		// Make API request
		resp, err := r.makeAPIRequest(token, "POST", "/api/dashboard/get-filtered-usage-events", payload)
		if err != nil {
			// If API fails, return 0 (not an error)
			return 0, nil
		}

		// Decode response
		var usageResp filteredUsageEventsResponse
		if err := json.NewDecoder(resp.Body).Decode(&usageResp); err != nil {
			_ = resp.Body.Close()
			return 0, domain.ErrCursorAPIWithCause("decode filtered usage events", err)
		}
		_ = resp.Body.Close()
		

		// Process each usage event
		for _, event := range usageResp.UsageEventsDisplay {
			// Parse timestamp
			timestamp, err := strconv.ParseInt(event.Timestamp, 10, 64)
			if err != nil {
				continue
			}

			// Convert to time
			eventTime := time.UnixMilli(timestamp)

			// Check if event is within today's range
			if eventTime.Before(startOfDay) || eventTime.After(now) {
				continue
			}
			
			totalEvents++

			// Sum all token types from tokenUsage
			if event.IsTokenBasedCall {
				eventTokens := int64(event.TokenUsage.InputTokens) + 
					int64(event.TokenUsage.OutputTokens) + 
					int64(event.TokenUsage.CacheWriteTokens) + 
					int64(event.TokenUsage.CacheReadTokens)
				
				if eventTokens > 0 {
					eventsWithTokens++
					totalTokens += eventTokens
				}
			}
		}

		// Check if we need to fetch more pages
		if len(usageResp.UsageEventsDisplay) < 100 {
			break
		}

		page++
	}
	

	return totalTokens, nil
}
