package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/ca-srg/tosage/domain/entity"
	"github.com/ca-srg/tosage/domain/repository"
	"github.com/ca-srg/tosage/infrastructure/config"
)

// PrometheusQueryRepository implements repository.PrometheusQueryRepository
type PrometheusQueryRepository struct {
	config     *config.PrometheusConfig
	httpClient *http.Client
	baseURL    string
}

// NewPrometheusQueryRepository creates a new Prometheus query repository
func NewPrometheusQueryRepository(cfg *config.PrometheusConfig) (repository.PrometheusQueryRepository, error) {
	if cfg == nil {
		return nil, repository.NewPrometheusQueryError("initialize", fmt.Errorf("prometheus config is nil"))
	}

	// Parse base URL from remote write URL
	u, err := url.Parse(cfg.RemoteWriteURL)
	if err != nil {
		return nil, repository.NewPrometheusQueryError("initialize", fmt.Errorf("invalid remote write URL: %w", err))
	}

	// Convert remote write URL to query API URL
	// e.g., http://localhost:9090/api/v1/write -> http://localhost:9090
	baseURL := fmt.Sprintf("%s://%s", u.Scheme, u.Host)

	return &PrometheusQueryRepository{
		config: cfg,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSec) * time.Second,
		},
		baseURL: baseURL,
	}, nil
}

// PrometheusResponse represents the response from Prometheus API
type PrometheusResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string                   `json:"resultType"`
		Result     []PrometheusResultSeries `json:"result"`
	} `json:"data"`
	ErrorType string `json:"errorType,omitempty"`
	Error     string `json:"error,omitempty"`
}

// PrometheusResultSeries represents a single time series result
type PrometheusResultSeries struct {
	Metric map[string]string `json:"metric"`
	Values [][]interface{}   `json:"values"` // For range queries: [timestamp, value]
	Value  []interface{}     `json:"value"`  // For instant queries: [timestamp, value]
}

// QueryRange queries Prometheus for time series data within a time range
func (r *PrometheusQueryRepository) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) ([]*entity.MetricDataPoint, error) {
	// Construct query parameters
	params := url.Values{}
	params.Set("query", query)
	params.Set("start", strconv.FormatInt(start.Unix(), 10))
	params.Set("end", strconv.FormatInt(end.Unix(), 10))
	params.Set("step", fmt.Sprintf("%.0f", step.Seconds()))

	// Build request URL
	reqURL := fmt.Sprintf("%s/api/v1/query_range?%s", r.baseURL, params.Encode())

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, repository.NewPrometheusQueryError("query_range", err)
	}

	// Add authentication if configured
	if r.config.Username != "" && r.config.Password != "" {
		req.SetBasicAuth(r.config.Username, r.config.Password)
	}

	// Execute request
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, repository.NewPrometheusQueryError("query_range", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			// Log error but don't fail the operation
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
		}
	}()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, repository.NewPrometheusQueryError("query_range", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, repository.NewPrometheusQueryError("query_range", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body)))
	}

	// Parse response
	var promResp PrometheusResponse
	if err := json.Unmarshal(body, &promResp); err != nil {
		return nil, repository.NewPrometheusQueryError("query_range", err)
	}

	// Check Prometheus response status
	if promResp.Status != "success" {
		return nil, repository.NewPrometheusQueryError("query_range", fmt.Errorf("%s: %s", promResp.ErrorType, promResp.Error))
	}

	// Convert to MetricDataPoints
	return r.convertToDataPoints(promResp, true)
}

// QueryInstant queries Prometheus for instant vector at a specific time
func (r *PrometheusQueryRepository) QueryInstant(ctx context.Context, query string, timestamp time.Time) ([]*entity.MetricDataPoint, error) {
	// Construct query parameters
	params := url.Values{}
	params.Set("query", query)
	params.Set("time", strconv.FormatInt(timestamp.Unix(), 10))

	// Build request URL
	reqURL := fmt.Sprintf("%s/api/v1/query?%s", r.baseURL, params.Encode())

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, repository.NewPrometheusQueryError("query", err)
	}

	// Add authentication if configured
	if r.config.Username != "" && r.config.Password != "" {
		req.SetBasicAuth(r.config.Username, r.config.Password)
	}

	// Execute request
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, repository.NewPrometheusQueryError("query", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			// Log error but don't fail the operation
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
		}
	}()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, repository.NewPrometheusQueryError("query", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, repository.NewPrometheusQueryError("query", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body)))
	}

	// Parse response
	var promResp PrometheusResponse
	if err := json.Unmarshal(body, &promResp); err != nil {
		return nil, repository.NewPrometheusQueryError("query", err)
	}

	// Check Prometheus response status
	if promResp.Status != "success" {
		return nil, repository.NewPrometheusQueryError("query", fmt.Errorf("%s: %s", promResp.ErrorType, promResp.Error))
	}

	// Convert to MetricDataPoints
	return r.convertToDataPoints(promResp, false)
}

// convertToDataPoints converts Prometheus response to MetricDataPoints
func (r *PrometheusQueryRepository) convertToDataPoints(resp PrometheusResponse, isRangeQuery bool) ([]*entity.MetricDataPoint, error) {
	var dataPoints []*entity.MetricDataPoint

	for _, series := range resp.Data.Result {
		// Extract labels
		host := series.Metric["host"]
		if host == "" {
			host = "unknown"
		}

		// Process values based on query type
		if isRangeQuery {
			// Range query: multiple values
			for _, valuePair := range series.Values {
				if len(valuePair) != 2 {
					continue
				}

				// Extract timestamp and value
				timestampFloat, ok := valuePair[0].(float64)
				if !ok {
					continue
				}
				timestamp := time.Unix(int64(timestampFloat), 0)

				valueStr, ok := valuePair[1].(string)
				if !ok {
					continue
				}
				value, err := strconv.ParseFloat(valueStr, 64)
				if err != nil {
					continue
				}

				// Create data point based on metric name
				dp := r.createDataPoint(series.Metric, timestamp, int(value), host)
				if dp != nil {
					dataPoints = append(dataPoints, dp)
				}
			}
		} else {
			// Instant query: single value
			if len(series.Value) != 2 {
				continue
			}

			// Extract timestamp and value
			timestampFloat, ok := series.Value[0].(float64)
			if !ok {
				continue
			}
			timestamp := time.Unix(int64(timestampFloat), 0)

			valueStr, ok := series.Value[1].(string)
			if !ok {
				continue
			}
			value, err := strconv.ParseFloat(valueStr, 64)
			if err != nil {
				continue
			}

			// Create data point based on metric name
			dp := r.createDataPoint(series.Metric, timestamp, int(value), host)
			if dp != nil {
				dataPoints = append(dataPoints, dp)
			}
		}
	}

	return dataPoints, nil
}

// createDataPoint creates a MetricDataPoint from metric labels and value
func (r *PrometheusQueryRepository) createDataPoint(metric map[string]string, timestamp time.Time, value int, host string) *entity.MetricDataPoint {
	// Determine metric type from __name__ label
	metricName := metric["__name__"]

	// Initialize token counts
	claudeCodeTokens := 0
	cursorTokens := 0

	// Set token count based on metric name
	switch metricName {
	case "tosage_cc_token":
		claudeCodeTokens = value
	case "tosage_cursor_token":
		cursorTokens = value
	default:
		// Unknown metric, skip
		return nil
	}

	// Create data point
	dp := entity.NewMetricDataPoint(
		timestamp,
		claudeCodeTokens,
		cursorTokens,
		host,
	)

	// Add timezone information if available
	if tz := metric["timezone"]; tz != "" {
		dp.WithTimezone(
			tz,
			metric["timezone_offset"],
			metric["detection_method"],
		)
	}

	return dp
}
