package repository

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ca-srg/tosage/domain/repository"
	"github.com/ca-srg/tosage/infrastructure/config"
)

// PrometheusMetricsRepository implements MetricsRepository using Prometheus Remote Write
type PrometheusMetricsRepository struct {
	config    *config.PrometheusConfig
	rwClient  *RemoteWriteClient
	hostLabel string
}

// NewPrometheusMetricsRepository creates a new Prometheus metrics repository
func NewPrometheusMetricsRepository(cfg *config.PrometheusConfig) (repository.MetricsRepository, error) {
	if cfg == nil {
		return nil, repository.NewMetricsRepositoryError("initialize", fmt.Errorf("prometheus config is nil"))
	}

	// Use hostname if HostLabel is not specified
	hostLabel := cfg.HostLabel
	if hostLabel == "" {
		hostname, err := os.Hostname()
		if err != nil {
			// Fall back to "unknown" if hostname cannot be determined
			hostLabel = "unknown"
		} else {
			hostLabel = hostname
		}
	}

	// Create authentication config (always use basic auth if credentials are provided)
	var authConfig *AuthConfig
	if cfg.Username != "" && cfg.Password != "" {
		authConfig = &AuthConfig{
			Username: cfg.Username,
			Password: cfg.Password,
		}
	}

	// Determine URL to use
	url := cfg.RemoteWriteURL
	if url == "" {
		return nil, repository.NewMetricsRepositoryError("initialize", fmt.Errorf("remote write url is empty"))
	}

	// Create Remote Write client
	rwClient, err := NewRemoteWriteClient(
		url,
		time.Duration(cfg.TimeoutSec)*time.Second,
		authConfig,
	)
	if err != nil {
		return nil, repository.NewMetricsRepositoryError("initialize", err)
	}

	return &PrometheusMetricsRepository{
		config:    cfg,
		rwClient:  rwClient,
		hostLabel: hostLabel,
	}, nil
}

// SendTokenMetric sends the total token count metric to Prometheus
func (r *PrometheusMetricsRepository) SendTokenMetric(totalTokens int, hostLabel string, metricName string) error {
	// Use provided hostLabel or fall back to configured one
	if hostLabel == "" {
		hostLabel = r.hostLabel
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.config.TimeoutSec)*time.Second)
	defer cancel()

	// Create labels for the metric
	labels := map[string]string{
		"host": hostLabel,
	}

	// Send metric via Remote Write
	err := r.rwClient.SendGaugeMetric(ctx, metricName, float64(totalTokens), labels)
	if err != nil {
		if ctx.Err() != nil {
			return repository.NewMetricsRepositoryError("send", fmt.Errorf("timeout: %w", err))
		}
		return repository.NewMetricsRepositoryError("send", err)
	}

	return nil
}

// Close cleans up resources
func (r *PrometheusMetricsRepository) Close() error {
	// Remote Write client doesn't require explicit cleanup
	return nil
}
