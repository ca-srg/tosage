package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Netflix/go-env"
)

// PrometheusConfig holds Prometheus integration configuration
type PrometheusConfig struct {
	// Remote Write configuration
	// RemoteWriteURL is the Prometheus Remote Write endpoint URL
	RemoteWriteURL string `json:"remote_write_url" env:"TOSAGE_PROMETHEUS_REMOTE_WRITE_URL"`

	// RemoteWriteUsername is the username for Remote Write authentication
	RemoteWriteUsername string `json:"remote_write_username" env:"TOSAGE_PROMETHEUS_REMOTE_WRITE_USERNAME"`

	// RemoteWritePassword is the password for Remote Write authentication
	RemoteWritePassword string `json:"remote_write_password" env:"TOSAGE_PROMETHEUS_REMOTE_WRITE_PASSWORD"`

	// Query configuration (new fields)
	// URL is the Prometheus query endpoint URL
	URL string `json:"url" env:"TOSAGE_PROMETHEUS_URL"`

	// Username is the username for query authentication
	Username string `json:"username" env:"TOSAGE_PROMETHEUS_USERNAME"`

	// Password is the password for query authentication
	Password string `json:"password" env:"TOSAGE_PROMETHEUS_PASSWORD"`

	// Common configuration
	// HostLabel is the host label value for metrics
	HostLabel string `json:"host_label,omitempty" env:"TOSAGE_PROMETHEUS_HOST_LABEL"`

	// IntervalSec is the interval in seconds between metric pushes
	IntervalSec int `json:"interval_seconds,omitempty" env:"TOSAGE_PROMETHEUS_INTERVAL_SECONDS,default=600"`

	// TimeoutSec is the timeout in seconds for metric pushes
	TimeoutSec int `json:"timeout_seconds,omitempty" env:"TOSAGE_PROMETHEUS_TIMEOUT_SECONDS,default=30"`
}

// CursorConfig holds Cursor integration configuration
type CursorConfig struct {
	// DatabasePath is the custom path to Cursor SQLite database
	DatabasePath string `json:"database_path,omitempty" env:"TOSAGE_CURSOR_DB_PATH,default="`

	// APITimeout is the timeout in seconds for Cursor API requests
	APITimeout int `json:"api_timeout,omitempty" env:"TOSAGE_CURSOR_API_TIMEOUT,default=30"`

	// CacheTimeout is the cache timeout in seconds for API responses
	CacheTimeout int `json:"cache_timeout,omitempty" env:"TOSAGE_CURSOR_CACHE_TIMEOUT,default=300"`
}

// BedrockConfig holds AWS Bedrock integration configuration
type BedrockConfig struct {
	// Enabled indicates if Bedrock tracking is enabled
	Enabled bool `json:"enabled,omitempty" env:"TOSAGE_BEDROCK_ENABLED,default=false"`

	// Regions is the list of AWS regions to monitor
	// Environment variable: TOSAGE_BEDROCK_REGIONS (comma-separated, e.g., "us-east-1,us-west-2,eu-west-1")
	Regions []string `json:"regions,omitempty" env:"TOSAGE_BEDROCK_REGIONS"`

	// AWSProfile is the AWS profile to use (optional)
	AWSProfile string `json:"aws_profile,omitempty" env:"TOSAGE_BEDROCK_AWS_PROFILE,default="`

	// AssumeRoleARN is the ARN of the role to assume (optional)
	AssumeRoleARN string `json:"assume_role_arn,omitempty" env:"TOSAGE_BEDROCK_ASSUME_ROLE_ARN,default="`

	// CollectionIntervalSec is how often to collect metrics in seconds
	CollectionIntervalSec int `json:"collection_interval_seconds,omitempty" env:"TOSAGE_BEDROCK_COLLECTION_INTERVAL_SECONDS,default=600"`
}

// VertexAIConfig holds Google Cloud Vertex AI integration configuration
type VertexAIConfig struct {
	// Enabled indicates if Vertex AI tracking is enabled
	Enabled bool `json:"enabled,omitempty" env:"TOSAGE_VERTEX_AI_ENABLED,default=false"`

	// ProjectID is the Google Cloud Project ID
	ProjectID string `json:"project_id,omitempty" env:"TOSAGE_VERTEX_AI_PROJECT_ID,default="`

	// ServiceAccountKeyPath is the path to the service account key file (optional)
	ServiceAccountKeyPath string `json:"service_account_key_path,omitempty" env:"TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY_PATH,default="`

	// ServiceAccountKey is the service account key JSON content (optional)
	ServiceAccountKey string `json:"service_account_key,omitempty" env:"TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY,default="`

	// CollectionIntervalSec is how often to collect metrics in seconds
	CollectionIntervalSec int `json:"collection_interval_seconds,omitempty" env:"TOSAGE_VERTEX_AI_COLLECTION_INTERVAL_SECONDS,default=600"`
}

// DaemonConfig holds daemon mode configuration
type DaemonConfig struct {
	// Enabled indicates whether daemon mode is enabled
	Enabled bool `json:"enabled,omitempty" env:"TOSAGE_DAEMON_ENABLED"`

	// StartAtLogin indicates whether to start at system login
	StartAtLogin bool `json:"start_at_login,omitempty" env:"TOSAGE_DAEMON_START_AT_LOGIN"`

	// HideFromDock indicates whether to hide the app from the Dock (macOS only)
	HideFromDock bool `json:"hide_from_dock,omitempty" env:"TOSAGE_DAEMON_HIDE_FROM_DOCK,default=true"`

	// LogPath is the path for daemon log files
	LogPath string `json:"log_path,omitempty" env:"TOSAGE_DAEMON_LOG_PATH"`

	// PidFile is the path for the daemon PID file
	PidFile string `json:"pid_file,omitempty" env:"TOSAGE_DAEMON_PID_FILE"`
}

// PromtailConfig holds Promtail logging configuration
type PromtailConfig struct {
	// URL is the Promtail push endpoint URL
	URL string `json:"url" env:"TOSAGE_LOKI_URL,required"`

	// Username is the username for basic authentication
	Username string `json:"username" env:"TOSAGE_LOKI_USERNAME,required"`

	// Password is the password for basic authentication
	Password string `json:"password" env:"TOSAGE_LOKI_PASSWORD,required"`

	// BatchWaitSeconds is the time to wait before sending a batch
	BatchWaitSeconds int `json:"batch_wait_seconds,omitempty" env:"TOSAGE_LOKI_BATCH_WAIT_SECONDS,default=1"`

	// BatchCapacity is the maximum number of log entries in a batch
	BatchCapacity int `json:"batch_capacity,omitempty" env:"TOSAGE_LOKI_BATCH_CAPACITY,default=100"`

	// TimeoutSeconds is the timeout for sending logs
	TimeoutSeconds int `json:"timeout_seconds,omitempty" env:"TOSAGE_LOKI_TIMEOUT_SECONDS,default=5"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	// Level is the minimum log level (debug, info, warn, error)
	Level string `json:"level,omitempty" env:"TOSAGE_LOG_LEVEL,default=info"`

	// Debug enables debug mode with stdout logging
	Debug bool `json:"debug,omitempty" env:"TOSAGE_LOG_DEBUG,default=false"`

	// Promtail holds Promtail configuration
	Promtail *PromtailConfig `json:"promtail,omitempty"`
}

// CSVExportConfig holds CSV export configuration
type CSVExportConfig struct {
	// DefaultOutputPath is the default output directory for CSV files
	DefaultOutputPath string `json:"default_output_path,omitempty" env:"TOSAGE_CSV_EXPORT_DEFAULT_OUTPUT_PATH,default=."`

	// DefaultStartDays is the default number of days to look back for data
	DefaultStartDays int `json:"default_start_days,omitempty" env:"TOSAGE_CSV_EXPORT_DEFAULT_START_DAYS,default=30"`

	// DefaultMetricTypes is the default comma-separated list of metric types to export
	DefaultMetricTypes string `json:"default_metric_types,omitempty" env:"TOSAGE_CSV_EXPORT_DEFAULT_METRIC_TYPES,default=claude_code,cursor,bedrock,vertex_ai"`

	// MaxExportDays is the maximum number of days allowed for export range
	MaxExportDays int `json:"max_export_days,omitempty" env:"TOSAGE_CSV_EXPORT_MAX_EXPORT_DAYS,default=365"`

	// TimeZone is the timezone to use for CSV export (IANA timezone)
	TimeZone string `json:"timezone,omitempty" env:"TOSAGE_CSV_EXPORT_TIMEZONE,default=Asia/Tokyo"`
}

// ConfigSource represents the source of a configuration value
type ConfigSource string

const (
	SourceDefault     ConfigSource = "default"
	SourceJSONFile    ConfigSource = "json"
	SourceEnvironment ConfigSource = "env"
)

// ConfigSourceMap tracks the source of each configuration field
type ConfigSourceMap map[string]ConfigSource

// AppConfig holds application configuration
type AppConfig struct {
	// Version is the configuration schema version
	Version int `json:"version,omitempty"`

	// ClaudePath is the custom path to Claude data directory
	ClaudePath string `json:"claude_path,omitempty" env:"TOSAGE_CLAUDE_PATH"`

	// Prometheus holds Prometheus integration configuration
	Prometheus *PrometheusConfig `json:"prometheus,omitempty"`

	// Cursor holds Cursor integration configuration
	Cursor *CursorConfig `json:"cursor,omitempty"`

	// Bedrock holds AWS Bedrock integration configuration
	Bedrock *BedrockConfig `json:"bedrock,omitempty"`

	// VertexAI holds Google Cloud Vertex AI integration configuration
	VertexAI *VertexAIConfig `json:"vertex_ai,omitempty"`

	// Daemon holds daemon mode configuration
	Daemon *DaemonConfig `json:"daemon,omitempty"`

	// Logging holds logging configuration
	Logging *LoggingConfig `json:"logging,omitempty"`

	// CSVExport holds CSV export configuration
	CSVExport *CSVExportConfig `json:"csv_export,omitempty"`

	// ConfigSources tracks the source of each configuration field
	ConfigSources ConfigSourceMap `json:"-"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *AppConfig {
	return &AppConfig{
		Version:    1, // Current configuration version
		ClaudePath: "",
		Prometheus: &PrometheusConfig{
			RemoteWriteURL:      "", // Empty by default, must be set via environment variable or config.json
			RemoteWriteUsername: "",
			RemoteWritePassword: "",
			URL:                 "",
			Username:            "",
			Password:            "",
			HostLabel:           "",
			IntervalSec:         600, // 10 minutes
			TimeoutSec:          30,
		},
		Cursor: &CursorConfig{
			DatabasePath: "",
			APITimeout:   30,  // 30 seconds
			CacheTimeout: 300, // 5 minutes
		},
		Bedrock: &BedrockConfig{
			Enabled:               false, // Disabled by default for security
			Regions:               []string{"us-east-1", "us-west-2"},
			AWSProfile:            "",
			AssumeRoleARN:         "",
			CollectionIntervalSec: 600, // 10 minutes
		},
		VertexAI: &VertexAIConfig{
			Enabled:               false, // Disabled by default for security
			ProjectID:             "",
			ServiceAccountKeyPath: "",
			ServiceAccountKey:     "",
			CollectionIntervalSec: 600, // 10 minutes
		},
		Daemon: &DaemonConfig{
			Enabled:      false,
			StartAtLogin: false,
			HideFromDock: false,
			LogPath:      "/tmp/tosage.log",
			PidFile:      "/tmp/tosage.pid",
		},
		Logging: &LoggingConfig{
			Level: "info",
			Debug: false,
			Promtail: &PromtailConfig{
				URL:              "http://localhost:3100/loki/api/v1/push",
				BatchWaitSeconds: 1,
				BatchCapacity:    100,
				TimeoutSeconds:   5,
			},
		},
		CSVExport: &CSVExportConfig{
			DefaultOutputPath:  ".",
			DefaultStartDays:   30,
			DefaultMetricTypes: "claude_code,cursor,bedrock,vertex_ai",
			MaxExportDays:      365,
			TimeZone:           "Asia/Tokyo",
		},
		ConfigSources: make(ConfigSourceMap),
	}
}

// MinimalDefaultConfig returns the minimal configuration template for initial setup
func MinimalDefaultConfig() *AppConfig {
	return &AppConfig{
		Version: 1, // Current configuration version
		Prometheus: &PrometheusConfig{
			RemoteWriteURL:      "",
			RemoteWriteUsername: "",
			RemoteWritePassword: "",
			URL:                 "",
			Username:            "",
			Password:            "",
			HostLabel:           "",
			IntervalSec:         600, // 10 minutes
			TimeoutSec:          30,
		},
		Logging: &LoggingConfig{
			Level: "info",
			Debug: false,
			Promtail: &PromtailConfig{
				URL:              "",
				Username:         "",
				Password:         "",
				BatchWaitSeconds: 1,
				BatchCapacity:    100,
				TimeoutSeconds:   5,
			},
		},
		CSVExport: &CSVExportConfig{
			DefaultOutputPath:  ".",
			DefaultStartDays:   30,
			DefaultMetricTypes: "claude_code,cursor",
			MaxExportDays:      365,
			TimeZone:           "Asia/Tokyo",
		},
		ConfigSources: make(ConfigSourceMap),
	}
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*AppConfig, error) {
	config := DefaultConfig()

	// Load environment variables using Netflix/go-env
	if err := config.LoadFromEnv(); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// LoadFromEnv loads configuration from environment variables using Netflix/go-env
func (c *AppConfig) LoadFromEnv() error {
	// Store original values to detect changes
	original := &AppConfig{
		ClaudePath: c.ClaudePath,
	}
	if c.Prometheus != nil {
		original.Prometheus = &PrometheusConfig{
			RemoteWriteURL:      c.Prometheus.RemoteWriteURL,
			RemoteWriteUsername: c.Prometheus.RemoteWriteUsername,
			RemoteWritePassword: c.Prometheus.RemoteWritePassword,
			URL:                 c.Prometheus.URL,
			Username:            c.Prometheus.Username,
			Password:            c.Prometheus.Password,
			HostLabel:           c.Prometheus.HostLabel,
			IntervalSec:         c.Prometheus.IntervalSec,
			TimeoutSec:          c.Prometheus.TimeoutSec,
		}
	}
	if c.Cursor != nil {
		original.Cursor = &CursorConfig{
			DatabasePath: c.Cursor.DatabasePath,
			APITimeout:   c.Cursor.APITimeout,
			CacheTimeout: c.Cursor.CacheTimeout,
		}
	}
	if c.Bedrock != nil {
		original.Bedrock = &BedrockConfig{
			Enabled:               c.Bedrock.Enabled,
			Regions:               c.Bedrock.Regions,
			AWSProfile:            c.Bedrock.AWSProfile,
			AssumeRoleARN:         c.Bedrock.AssumeRoleARN,
			CollectionIntervalSec: c.Bedrock.CollectionIntervalSec,
		}
	}
	if c.VertexAI != nil {
		original.VertexAI = &VertexAIConfig{
			Enabled:               c.VertexAI.Enabled,
			ProjectID:             c.VertexAI.ProjectID,
			ServiceAccountKeyPath: c.VertexAI.ServiceAccountKeyPath,
			ServiceAccountKey:     c.VertexAI.ServiceAccountKey,
			CollectionIntervalSec: c.VertexAI.CollectionIntervalSec,
		}
	}
	if c.Daemon != nil {
		original.Daemon = &DaemonConfig{
			Enabled:      c.Daemon.Enabled,
			StartAtLogin: c.Daemon.StartAtLogin,
			HideFromDock: c.Daemon.HideFromDock,
			LogPath:      c.Daemon.LogPath,
			PidFile:      c.Daemon.PidFile,
		}
	}
	if c.Logging != nil {
		original.Logging = &LoggingConfig{
			Level: c.Logging.Level,
			Debug: c.Logging.Debug,
		}
		if c.Logging.Promtail != nil {
			original.Logging.Promtail = &PromtailConfig{
				URL:              c.Logging.Promtail.URL,
				Username:         c.Logging.Promtail.Username,
				Password:         c.Logging.Promtail.Password,
				BatchWaitSeconds: c.Logging.Promtail.BatchWaitSeconds,
				BatchCapacity:    c.Logging.Promtail.BatchCapacity,
				TimeoutSeconds:   c.Logging.Promtail.TimeoutSeconds,
			}
		}
	}
	if c.CSVExport != nil {
		original.CSVExport = &CSVExportConfig{
			DefaultOutputPath:  c.CSVExport.DefaultOutputPath,
			DefaultStartDays:   c.CSVExport.DefaultStartDays,
			DefaultMetricTypes: c.CSVExport.DefaultMetricTypes,
			MaxExportDays:      c.CSVExport.MaxExportDays,
			TimeZone:           c.CSVExport.TimeZone,
		}
	}

	// Use Netflix/go-env to unmarshal environment variables into the config struct
	_, err := env.UnmarshalFromEnviron(c)
	if err != nil {
		return fmt.Errorf("failed to unmarshal environment variables: %w", err)
	}

	// Track environment variable overrides
	if c.ClaudePath != original.ClaudePath && os.Getenv("TOSAGE_CLAUDE_PATH") != "" {
		c.ConfigSources["ClaudePath"] = SourceEnvironment
	}

	// Special handling for Prometheus nested struct
	if c.Prometheus != nil {
		_, err = env.UnmarshalFromEnviron(c.Prometheus)
		if err != nil {
			return fmt.Errorf("failed to unmarshal Prometheus environment variables: %w", err)
		}
		c.trackPrometheusEnvOverrides(original.Prometheus)
	}

	// Special handling for Cursor nested struct
	if c.Cursor != nil {
		_, err = env.UnmarshalFromEnviron(c.Cursor)
		if err != nil {
			return fmt.Errorf("failed to unmarshal Cursor environment variables: %w", err)
		}
		c.trackCursorEnvOverrides(original.Cursor)
	}

	// Special handling for Bedrock nested struct
	if c.Bedrock != nil {
		_, err = env.UnmarshalFromEnviron(c.Bedrock)
		if err != nil {
			return fmt.Errorf("failed to unmarshal Bedrock environment variables: %w", err)
		}
		// Custom handling for Regions slice
		if regionsEnv := os.Getenv("TOSAGE_BEDROCK_REGIONS"); regionsEnv != "" {
			c.Bedrock.Regions = splitCommaSeparated(regionsEnv)
		}
		c.trackBedrockEnvOverrides(original.Bedrock)
	}

	// Special handling for VertexAI nested struct
	if c.VertexAI != nil {
		_, err = env.UnmarshalFromEnviron(c.VertexAI)
		if err != nil {
			return fmt.Errorf("failed to unmarshal VertexAI environment variables: %w", err)
		}
		// Custom handling for base64-encoded ServiceAccountKey
		if base64Key := os.Getenv("TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY"); base64Key != "" {
			decodedKey, err := base64.StdEncoding.DecodeString(base64Key)
			if err != nil {
				return fmt.Errorf("failed to decode base64 service account key: %w", err)
			}
			c.VertexAI.ServiceAccountKey = string(decodedKey)
		}
		c.trackVertexAIEnvOverrides(original.VertexAI)
	}

	// Special handling for Daemon nested struct
	if c.Daemon != nil {
		_, err = env.UnmarshalFromEnviron(c.Daemon)
		if err != nil {
			return fmt.Errorf("failed to unmarshal Daemon environment variables: %w", err)
		}
		c.trackDaemonEnvOverrides(original.Daemon)
	}

	// Special handling for Logging nested struct
	if c.Logging != nil {
		_, err = env.UnmarshalFromEnviron(c.Logging)
		if err != nil {
			return fmt.Errorf("failed to unmarshal Logging environment variables: %w", err)
		}
		c.trackLoggingEnvOverrides(original.Logging)

		// Handle Promtail nested struct
		if c.Logging.Promtail != nil {
			_, err = env.UnmarshalFromEnviron(c.Logging.Promtail)
			if err != nil {
				return fmt.Errorf("failed to unmarshal Promtail environment variables: %w", err)
			}
			if original.Logging != nil && original.Logging.Promtail != nil {
				c.trackPromtailEnvOverrides(original.Logging.Promtail)
			}
		}
	}

	// Special handling for CSVExport nested struct
	if c.CSVExport != nil {
		_, err = env.UnmarshalFromEnviron(c.CSVExport)
		if err != nil {
			return fmt.Errorf("failed to unmarshal CSVExport environment variables: %w", err)
		}
		c.trackCSVExportEnvOverrides(original.CSVExport)
	}

	return nil
}

// trackPrometheusEnvOverrides tracks environment variable overrides for Prometheus config
func (c *AppConfig) trackPrometheusEnvOverrides(original *PrometheusConfig) {
	// Debug: Log what we're tracking
	if os.Getenv("TOSAGE_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "Debug: trackPrometheusEnvOverrides called\n")
		fmt.Fprintf(os.Stderr, "Debug: ENV TOSAGE_PROMETHEUS_REMOTE_WRITE_URL='%s'\n", os.Getenv("TOSAGE_PROMETHEUS_REMOTE_WRITE_URL"))
		fmt.Fprintf(os.Stderr, "Debug: Config RemoteWriteURL='%s'\n", c.Prometheus.RemoteWriteURL)
		fmt.Fprintf(os.Stderr, "Debug: Original RemoteWriteURL='%s'\n", original.RemoteWriteURL)
	}
	if original == nil {
		return
	}
	if c.Prometheus.RemoteWriteURL != original.RemoteWriteURL && os.Getenv("TOSAGE_PROMETHEUS_REMOTE_WRITE_URL") != "" {
		c.ConfigSources["Prometheus.RemoteWriteURL"] = SourceEnvironment
	}
	if c.Prometheus.RemoteWriteUsername != original.RemoteWriteUsername && os.Getenv("TOSAGE_PROMETHEUS_REMOTE_WRITE_USERNAME") != "" {
		c.ConfigSources["Prometheus.RemoteWriteUsername"] = SourceEnvironment
	}
	if c.Prometheus.RemoteWritePassword != original.RemoteWritePassword && os.Getenv("TOSAGE_PROMETHEUS_REMOTE_WRITE_PASSWORD") != "" {
		c.ConfigSources["Prometheus.RemoteWritePassword"] = SourceEnvironment
	}
	if c.Prometheus.URL != original.URL && os.Getenv("TOSAGE_PROMETHEUS_URL") != "" {
		c.ConfigSources["Prometheus.URL"] = SourceEnvironment
	}
	if c.Prometheus.Username != original.Username && os.Getenv("TOSAGE_PROMETHEUS_USERNAME") != "" {
		c.ConfigSources["Prometheus.Username"] = SourceEnvironment
	}
	if c.Prometheus.Password != original.Password && os.Getenv("TOSAGE_PROMETHEUS_PASSWORD") != "" {
		c.ConfigSources["Prometheus.Password"] = SourceEnvironment
	}
	if c.Prometheus.HostLabel != original.HostLabel && os.Getenv("TOSAGE_PROMETHEUS_HOST_LABEL") != "" {
		c.ConfigSources["Prometheus.HostLabel"] = SourceEnvironment
	}
	if c.Prometheus.IntervalSec != original.IntervalSec && os.Getenv("TOSAGE_PROMETHEUS_INTERVAL_SECONDS") != "" {
		c.ConfigSources["Prometheus.IntervalSec"] = SourceEnvironment
	}
	if c.Prometheus.TimeoutSec != original.TimeoutSec && os.Getenv("TOSAGE_PROMETHEUS_TIMEOUT_SECONDS") != "" {
		c.ConfigSources["Prometheus.TimeoutSec"] = SourceEnvironment
	}
}

// trackCursorEnvOverrides tracks environment variable overrides for Cursor config
func (c *AppConfig) trackCursorEnvOverrides(original *CursorConfig) {
	if original == nil {
		return
	}
	if c.Cursor.DatabasePath != original.DatabasePath && os.Getenv("TOSAGE_CURSOR_DB_PATH") != "" {
		c.ConfigSources["Cursor.DatabasePath"] = SourceEnvironment
	}
	if c.Cursor.APITimeout != original.APITimeout && os.Getenv("TOSAGE_CURSOR_API_TIMEOUT") != "" {
		c.ConfigSources["Cursor.APITimeout"] = SourceEnvironment
	}
	if c.Cursor.CacheTimeout != original.CacheTimeout && os.Getenv("TOSAGE_CURSOR_CACHE_TIMEOUT") != "" {
		c.ConfigSources["Cursor.CacheTimeout"] = SourceEnvironment
	}
}

// trackBedrockEnvOverrides tracks environment variable overrides for Bedrock config
func (c *AppConfig) trackBedrockEnvOverrides(original *BedrockConfig) {
	if original == nil {
		return
	}
	if c.Bedrock.Enabled != original.Enabled && os.Getenv("TOSAGE_BEDROCK_ENABLED") != "" {
		c.ConfigSources["Bedrock.Enabled"] = SourceEnvironment
	}
	if c.Bedrock.AWSProfile != original.AWSProfile && os.Getenv("TOSAGE_BEDROCK_AWS_PROFILE") != "" {
		c.ConfigSources["Bedrock.AWSProfile"] = SourceEnvironment
	}
	if c.Bedrock.AssumeRoleARN != original.AssumeRoleARN && os.Getenv("TOSAGE_BEDROCK_ASSUME_ROLE_ARN") != "" {
		c.ConfigSources["Bedrock.AssumeRoleARN"] = SourceEnvironment
	}
	if c.Bedrock.CollectionIntervalSec != original.CollectionIntervalSec && os.Getenv("TOSAGE_BEDROCK_COLLECTION_INTERVAL_SECONDS") != "" {
		c.ConfigSources["Bedrock.CollectionIntervalSec"] = SourceEnvironment
	}
	// Track Regions if changed from environment
	if !slicesEqual(c.Bedrock.Regions, original.Regions) && os.Getenv("TOSAGE_BEDROCK_REGIONS") != "" {
		c.ConfigSources["Bedrock.Regions"] = SourceEnvironment
	}
}

// trackVertexAIEnvOverrides tracks environment variable overrides for VertexAI config
func (c *AppConfig) trackVertexAIEnvOverrides(original *VertexAIConfig) {
	if original == nil {
		return
	}
	if c.VertexAI.Enabled != original.Enabled && os.Getenv("TOSAGE_VERTEX_AI_ENABLED") != "" {
		c.ConfigSources["VertexAI.Enabled"] = SourceEnvironment
	}
	if c.VertexAI.ProjectID != original.ProjectID && os.Getenv("TOSAGE_VERTEX_AI_PROJECT_ID") != "" {
		c.ConfigSources["VertexAI.ProjectID"] = SourceEnvironment
	}
	if c.VertexAI.ServiceAccountKeyPath != original.ServiceAccountKeyPath && os.Getenv("TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY_PATH") != "" {
		c.ConfigSources["VertexAI.ServiceAccountKeyPath"] = SourceEnvironment
	}
	if c.VertexAI.ServiceAccountKey != original.ServiceAccountKey && os.Getenv("TOSAGE_VERTEX_AI_SERVICE_ACCOUNT_KEY") != "" {
		c.ConfigSources["VertexAI.ServiceAccountKey"] = SourceEnvironment
	}
	if c.VertexAI.CollectionIntervalSec != original.CollectionIntervalSec && os.Getenv("TOSAGE_VERTEX_AI_COLLECTION_INTERVAL_SECONDS") != "" {
		c.ConfigSources["VertexAI.CollectionIntervalSec"] = SourceEnvironment
	}
	// Track Locations if changed from environment
}

// trackDaemonEnvOverrides tracks environment variable overrides for Daemon config
func (c *AppConfig) trackDaemonEnvOverrides(original *DaemonConfig) {
	if original == nil {
		return
	}
	if c.Daemon.Enabled != original.Enabled && os.Getenv("TOSAGE_DAEMON_ENABLED") != "" {
		c.ConfigSources["Daemon.Enabled"] = SourceEnvironment
	}
	if c.Daemon.StartAtLogin != original.StartAtLogin && os.Getenv("TOSAGE_DAEMON_START_AT_LOGIN") != "" {
		c.ConfigSources["Daemon.StartAtLogin"] = SourceEnvironment
	}
	if c.Daemon.HideFromDock != original.HideFromDock && os.Getenv("TOSAGE_DAEMON_HIDE_FROM_DOCK") != "" {
		c.ConfigSources["Daemon.HideFromDock"] = SourceEnvironment
	}
	if c.Daemon.LogPath != original.LogPath && os.Getenv("TOSAGE_DAEMON_LOG_PATH") != "" {
		c.ConfigSources["Daemon.LogPath"] = SourceEnvironment
	}
	if c.Daemon.PidFile != original.PidFile && os.Getenv("TOSAGE_DAEMON_PID_FILE") != "" {
		c.ConfigSources["Daemon.PidFile"] = SourceEnvironment
	}
}

// trackLoggingEnvOverrides tracks environment variable overrides for Logging config
func (c *AppConfig) trackLoggingEnvOverrides(original *LoggingConfig) {
	if original == nil {
		return
	}
	if c.Logging.Level != original.Level && os.Getenv("TOSAGE_LOG_LEVEL") != "" {
		c.ConfigSources["Logging.Level"] = SourceEnvironment
	}
	if c.Logging.Debug != original.Debug && os.Getenv("TOSAGE_LOG_DEBUG") != "" {
		c.ConfigSources["Logging.Debug"] = SourceEnvironment
	}
}

// trackPromtailEnvOverrides tracks environment variable overrides for Promtail config
func (c *AppConfig) trackPromtailEnvOverrides(original *PromtailConfig) {
	if original == nil {
		return
	}
	if c.Logging.Promtail.URL != original.URL && os.Getenv("TOSAGE_LOKI_URL") != "" {
		c.ConfigSources["Promtail.URL"] = SourceEnvironment
	}
	if c.Logging.Promtail.Username != original.Username && os.Getenv("TOSAGE_LOKI_USERNAME") != "" {
		c.ConfigSources["Promtail.Username"] = SourceEnvironment
	}
	if c.Logging.Promtail.Password != original.Password && os.Getenv("TOSAGE_LOKI_PASSWORD") != "" {
		c.ConfigSources["Promtail.Password"] = SourceEnvironment
	}
	if c.Logging.Promtail.BatchWaitSeconds != original.BatchWaitSeconds && os.Getenv("TOSAGE_LOKI_BATCH_WAIT_SECONDS") != "" {
		c.ConfigSources["Promtail.BatchWaitSeconds"] = SourceEnvironment
	}
	if c.Logging.Promtail.BatchCapacity != original.BatchCapacity && os.Getenv("TOSAGE_LOKI_BATCH_CAPACITY") != "" {
		c.ConfigSources["Promtail.BatchCapacity"] = SourceEnvironment
	}
	if c.Logging.Promtail.TimeoutSeconds != original.TimeoutSeconds && os.Getenv("TOSAGE_LOKI_TIMEOUT_SECONDS") != "" {
		c.ConfigSources["Promtail.TimeoutSeconds"] = SourceEnvironment
	}
}

// trackCSVExportEnvOverrides tracks environment variable overrides for CSVExport config
func (c *AppConfig) trackCSVExportEnvOverrides(original *CSVExportConfig) {
	if original == nil {
		return
	}
	if c.CSVExport.DefaultOutputPath != original.DefaultOutputPath && os.Getenv("TOSAGE_CSV_EXPORT_DEFAULT_OUTPUT_PATH") != "" {
		c.ConfigSources["CSVExport.DefaultOutputPath"] = SourceEnvironment
	}
	if c.CSVExport.DefaultStartDays != original.DefaultStartDays && os.Getenv("TOSAGE_CSV_EXPORT_DEFAULT_START_DAYS") != "" {
		c.ConfigSources["CSVExport.DefaultStartDays"] = SourceEnvironment
	}
	if c.CSVExport.DefaultMetricTypes != original.DefaultMetricTypes && os.Getenv("TOSAGE_CSV_EXPORT_DEFAULT_METRIC_TYPES") != "" {
		c.ConfigSources["CSVExport.DefaultMetricTypes"] = SourceEnvironment
	}
	if c.CSVExport.MaxExportDays != original.MaxExportDays && os.Getenv("TOSAGE_CSV_EXPORT_MAX_EXPORT_DAYS") != "" {
		c.ConfigSources["CSVExport.MaxExportDays"] = SourceEnvironment
	}
	if c.CSVExport.TimeZone != original.TimeZone && os.Getenv("TOSAGE_CSV_EXPORT_TIMEZONE") != "" {
		c.ConfigSources["CSVExport.TimeZone"] = SourceEnvironment
	}
}

// Validate validates the configuration
func (c *AppConfig) Validate() error {
	// Validate Prometheus configuration
	if c.Prometheus != nil {
		if err := c.validatePrometheus(); err != nil {
			return err
		}
	}

	// Validate Cursor configuration
	if c.Cursor != nil {
		if err := c.validateCursor(); err != nil {
			return err
		}
	}

	// Validate Bedrock configuration
	if c.Bedrock != nil {
		if err := c.validateBedrock(); err != nil {
			return err
		}
	}

	// Validate VertexAI configuration
	if c.VertexAI != nil {
		if err := c.validateVertexAI(); err != nil {
			return err
		}
	}

	// Validate Daemon configuration
	if c.Daemon != nil {
		if err := c.validateDaemon(); err != nil {
			return err
		}
	}

	// Validate Logging configuration
	if c.Logging != nil {
		if err := c.validateLogging(); err != nil {
			return err
		}
	}

	// Validate CSVExport configuration
	if c.CSVExport != nil {
		if err := c.validateCSVExport(); err != nil {
			return err
		}
	}

	return nil
}

// validatePrometheus validates Prometheus configuration
func (c *AppConfig) validatePrometheus() error {
	if c.Prometheus == nil {
		return nil
	}

	// Skip validation if RemoteWriteURL is empty (initial configuration)
	if c.Prometheus.RemoteWriteURL == "" {
		return nil
	}

	// Validate interval is reasonable
	if c.Prometheus.IntervalSec < 60 {
		return fmt.Errorf("prometheus interval must be at least 60 seconds")
	}

	// Validate timeout is reasonable
	if c.Prometheus.TimeoutSec < 1 {
		return fmt.Errorf("prometheus timeout must be at least 1 second")
	}

	if c.Prometheus.TimeoutSec >= c.Prometheus.IntervalSec {
		return fmt.Errorf("prometheus timeout must be less than interval")
	}

	// Validate basic authentication is provided for remote write
	if c.Prometheus.RemoteWriteUsername == "" || c.Prometheus.RemoteWritePassword == "" {
		return fmt.Errorf("remote write username and password are required when remote write URL is set")
	}

	// Validate query configuration if URL is provided
	if c.Prometheus.URL != "" {
		// Validate basic authentication is provided for query
		if c.Prometheus.Username == "" || c.Prometheus.Password == "" {
			return fmt.Errorf("query username and password are required when query URL is set")
		}
	}

	return nil
}

// validateCursor validates Cursor configuration
func (c *AppConfig) validateCursor() error {
	if c.Cursor == nil {
		return nil
	}

	// Validate API timeout is reasonable
	if c.Cursor.APITimeout < 1 {
		return fmt.Errorf("cursor API timeout must be at least 1 second")
	}

	// Validate cache timeout is reasonable
	if c.Cursor.CacheTimeout < 0 {
		return fmt.Errorf("cursor cache timeout cannot be negative")
	}

	return nil
}

// validateBedrock validates Bedrock configuration
func (c *AppConfig) validateBedrock() error {
	if c.Bedrock == nil {
		return nil
	}

	// Validate collection interval is reasonable when enabled
	if c.Bedrock.Enabled && c.Bedrock.CollectionIntervalSec < 60 {
		return fmt.Errorf("bedrock collection interval must be at least 60 seconds")
	}

	// Validate regions are provided when enabled
	if c.Bedrock.Enabled && len(c.Bedrock.Regions) == 0 {
		return fmt.Errorf("bedrock regions cannot be empty when bedrock is enabled")
	}

	return nil
}

// validateVertexAI validates VertexAI configuration
func (c *AppConfig) validateVertexAI() error {
	if c.VertexAI == nil {
		return nil
	}

	// Validate collection interval is reasonable when enabled
	if c.VertexAI.Enabled && c.VertexAI.CollectionIntervalSec < 60 {
		return fmt.Errorf("vertex ai collection interval must be at least 60 seconds")
	}

	// Validate project ID is provided when enabled
	if c.VertexAI.Enabled && c.VertexAI.ProjectID == "" {
		return fmt.Errorf("vertex ai project ID cannot be empty when vertex ai is enabled")
	}

	// Validate service account key JSON if provided
	if c.VertexAI.ServiceAccountKey != "" {
		var keyData map[string]interface{}
		if err := json.Unmarshal([]byte(c.VertexAI.ServiceAccountKey), &keyData); err != nil {
			return fmt.Errorf("invalid service account key JSON: %w", err)
		}

		// Check required fields in service account key
		requiredFields := []string{"type", "project_id", "private_key_id", "private_key", "client_email"}
		for _, field := range requiredFields {
			if _, ok := keyData[field]; !ok {
				return fmt.Errorf("service account key missing required field: %s", field)
			}
		}

		// Validate type field
		if keyType, ok := keyData["type"].(string); !ok || keyType != "service_account" {
			return fmt.Errorf("service account key must have type 'service_account'")
		}
	}

	return nil
}

// validateDaemon validates Daemon configuration
func (c *AppConfig) validateDaemon() error {
	if c.Daemon == nil {
		return nil
	}

	// Validate log path is not empty when daemon is enabled
	if c.Daemon.Enabled && c.Daemon.LogPath == "" {
		return fmt.Errorf("daemon log path cannot be empty when daemon is enabled")
	}

	// Validate PID file path is not empty when daemon is enabled
	if c.Daemon.Enabled && c.Daemon.PidFile == "" {
		return fmt.Errorf("daemon PID file path cannot be empty when daemon is enabled")
	}

	return nil
}

// validateLogging validates Logging configuration
func (c *AppConfig) validateLogging() error {
	if c.Logging == nil {
		return nil
	}

	// Validate log level only if specified
	if c.Logging.Level != "" {
		validLevels := map[string]bool{
			"debug": true,
			"info":  true,
			"warn":  true,
			"error": true,
		}
		if !validLevels[c.Logging.Level] {
			return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.Logging.Level)
		}
	}

	// Validate Promtail configuration
	if c.Logging.Promtail != nil {
		// Skip validation if Promtail URL is empty (initial configuration)
		if c.Logging.Promtail.URL == "" {
			return nil
		}

		if c.Logging.Promtail.BatchWaitSeconds < 1 {
			return fmt.Errorf("promtail batch wait must be at least 1 second")
		}

		if c.Logging.Promtail.BatchCapacity < 1 {
			return fmt.Errorf("promtail batch capacity must be at least 1")
		}

		if c.Logging.Promtail.TimeoutSeconds < 1 {
			return fmt.Errorf("promtail timeout must be at least 1 second")
		}
	}

	return nil
}

// validateCSVExport validates CSVExport configuration
func (c *AppConfig) validateCSVExport() error {
	if c.CSVExport == nil {
		return nil
	}

	// Validate default start days is reasonable
	if c.CSVExport.DefaultStartDays < 0 {
		return fmt.Errorf("csv export default start days cannot be negative")
	}

	// Validate max export days is reasonable
	if c.CSVExport.MaxExportDays < 1 {
		return fmt.Errorf("csv export max export days must be at least 1")
	}

	// Validate that default start days doesn't exceed max export days
	if c.CSVExport.DefaultStartDays > c.CSVExport.MaxExportDays {
		return fmt.Errorf("csv export default start days cannot exceed max export days")
	}

	// Validate timezone format
	if c.CSVExport.TimeZone != "" {
		if _, err := time.LoadLocation(c.CSVExport.TimeZone); err != nil {
			return fmt.Errorf("csv export timezone is invalid: %w", err)
		}
	}

	return nil
}

// MarkDefaults marks all configuration fields as coming from defaults
func (c *AppConfig) MarkDefaults() {
	c.ConfigSources["Version"] = SourceDefault
	c.ConfigSources["ClaudePath"] = SourceDefault
	c.ConfigSources["Prometheus.RemoteWriteURL"] = SourceDefault
	c.ConfigSources["Prometheus.RemoteWriteUsername"] = SourceDefault
	c.ConfigSources["Prometheus.RemoteWritePassword"] = SourceDefault
	c.ConfigSources["Prometheus.URL"] = SourceDefault
	c.ConfigSources["Prometheus.Username"] = SourceDefault
	c.ConfigSources["Prometheus.Password"] = SourceDefault
	c.ConfigSources["Prometheus.HostLabel"] = SourceDefault
	c.ConfigSources["Prometheus.IntervalSec"] = SourceDefault
	c.ConfigSources["Prometheus.TimeoutSec"] = SourceDefault
	c.ConfigSources["Cursor.DatabasePath"] = SourceDefault
	c.ConfigSources["Cursor.APITimeout"] = SourceDefault
	c.ConfigSources["Cursor.CacheTimeout"] = SourceDefault
	c.ConfigSources["Bedrock.Enabled"] = SourceDefault
	c.ConfigSources["Bedrock.AWSProfile"] = SourceDefault
	c.ConfigSources["Bedrock.AssumeRoleARN"] = SourceDefault
	c.ConfigSources["Bedrock.CollectionIntervalSec"] = SourceDefault
	c.ConfigSources["VertexAI.Enabled"] = SourceDefault
	c.ConfigSources["VertexAI.ProjectID"] = SourceDefault
	c.ConfigSources["VertexAI.ServiceAccountKeyPath"] = SourceDefault
	c.ConfigSources["VertexAI.ServiceAccountKey"] = SourceDefault
	c.ConfigSources["VertexAI.CollectionIntervalSec"] = SourceDefault
	c.ConfigSources["Daemon.Enabled"] = SourceDefault
	c.ConfigSources["Daemon.StartAtLogin"] = SourceDefault
	c.ConfigSources["Daemon.HideFromDock"] = SourceDefault
	c.ConfigSources["Daemon.LogPath"] = SourceDefault
	c.ConfigSources["Daemon.PidFile"] = SourceDefault
	c.ConfigSources["Logging.Level"] = SourceDefault
	c.ConfigSources["Logging.Debug"] = SourceDefault
	c.ConfigSources["Promtail.URL"] = SourceDefault
	c.ConfigSources["Promtail.Username"] = SourceDefault
	c.ConfigSources["Promtail.Password"] = SourceDefault
	c.ConfigSources["Promtail.BatchWaitSeconds"] = SourceDefault
	c.ConfigSources["Promtail.BatchCapacity"] = SourceDefault
	c.ConfigSources["Promtail.TimeoutSeconds"] = SourceDefault
	c.ConfigSources["CSVExport.DefaultOutputPath"] = SourceDefault
	c.ConfigSources["CSVExport.DefaultStartDays"] = SourceDefault
	c.ConfigSources["CSVExport.DefaultMetricTypes"] = SourceDefault
	c.ConfigSources["CSVExport.MaxExportDays"] = SourceDefault
	c.ConfigSources["CSVExport.TimeZone"] = SourceDefault
}

// MergeJSONConfig merges JSON configuration into the current configuration
func (c *AppConfig) MergeJSONConfig(jsonConfig *AppConfig) {
	// Merge top-level fields
	// Always merge version from JSON, even if it's 0 (legacy config)
	c.Version = jsonConfig.Version
	c.ConfigSources["Version"] = SourceJSONFile
	if jsonConfig.ClaudePath != "" {
		c.ClaudePath = jsonConfig.ClaudePath
		c.ConfigSources["ClaudePath"] = SourceJSONFile
	}

	// Merge Prometheus configuration
	if jsonConfig.Prometheus != nil {
		if c.Prometheus == nil {
			c.Prometheus = &PrometheusConfig{}
		}
		c.mergePrometheusConfig(jsonConfig.Prometheus)
	}

	// Merge Cursor configuration
	if jsonConfig.Cursor != nil {
		if c.Cursor == nil {
			c.Cursor = &CursorConfig{}
		}
		c.mergeCursorConfig(jsonConfig.Cursor)
	}

	// Merge Bedrock configuration
	if jsonConfig.Bedrock != nil {
		if c.Bedrock == nil {
			c.Bedrock = &BedrockConfig{}
		}
		c.mergeBedrockConfig(jsonConfig.Bedrock)
	}

	// Merge VertexAI configuration
	if jsonConfig.VertexAI != nil {
		if c.VertexAI == nil {
			c.VertexAI = &VertexAIConfig{}
		}
		c.mergeVertexAIConfig(jsonConfig.VertexAI)
	}

	// Merge Daemon configuration
	if jsonConfig.Daemon != nil {
		if c.Daemon == nil {
			c.Daemon = &DaemonConfig{}
		}
		c.mergeDaemonConfig(jsonConfig.Daemon)
	}

	// Merge Logging configuration
	if jsonConfig.Logging != nil {
		if c.Logging == nil {
			c.Logging = &LoggingConfig{}
		}
		c.mergeLoggingConfig(jsonConfig.Logging)
	}

	// Merge CSVExport configuration
	if jsonConfig.CSVExport != nil {
		if c.CSVExport == nil {
			c.CSVExport = &CSVExportConfig{}
		}
		c.mergeCSVExportConfig(jsonConfig.CSVExport)
	}
}

// mergePrometheusConfig merges Prometheus configuration from JSON
func (c *AppConfig) mergePrometheusConfig(jsonConfig *PrometheusConfig) {
	if jsonConfig.RemoteWriteURL != "" {
		c.Prometheus.RemoteWriteURL = jsonConfig.RemoteWriteURL
		c.ConfigSources["Prometheus.RemoteWriteURL"] = SourceJSONFile
	}
	if jsonConfig.RemoteWriteUsername != "" {
		c.Prometheus.RemoteWriteUsername = jsonConfig.RemoteWriteUsername
		c.ConfigSources["Prometheus.RemoteWriteUsername"] = SourceJSONFile
	}
	if jsonConfig.RemoteWritePassword != "" {
		c.Prometheus.RemoteWritePassword = jsonConfig.RemoteWritePassword
		c.ConfigSources["Prometheus.RemoteWritePassword"] = SourceJSONFile
	}
	if jsonConfig.URL != "" {
		c.Prometheus.URL = jsonConfig.URL
		c.ConfigSources["Prometheus.URL"] = SourceJSONFile
	}
	if jsonConfig.Username != "" {
		c.Prometheus.Username = jsonConfig.Username
		c.ConfigSources["Prometheus.Username"] = SourceJSONFile
	}
	if jsonConfig.Password != "" {
		c.Prometheus.Password = jsonConfig.Password
		c.ConfigSources["Prometheus.Password"] = SourceJSONFile
	}
	if jsonConfig.HostLabel != "" {
		c.Prometheus.HostLabel = jsonConfig.HostLabel
		c.ConfigSources["Prometheus.HostLabel"] = SourceJSONFile
	}
	if jsonConfig.IntervalSec != 0 {
		c.Prometheus.IntervalSec = jsonConfig.IntervalSec
		c.ConfigSources["Prometheus.IntervalSec"] = SourceJSONFile
	}
	if jsonConfig.TimeoutSec != 0 {
		c.Prometheus.TimeoutSec = jsonConfig.TimeoutSec
		c.ConfigSources["Prometheus.TimeoutSec"] = SourceJSONFile
	}
}

// mergeCursorConfig merges Cursor configuration from JSON
func (c *AppConfig) mergeCursorConfig(jsonConfig *CursorConfig) {
	if jsonConfig.DatabasePath != "" {
		c.Cursor.DatabasePath = jsonConfig.DatabasePath
		c.ConfigSources["Cursor.DatabasePath"] = SourceJSONFile
	}
	if jsonConfig.APITimeout != 0 {
		c.Cursor.APITimeout = jsonConfig.APITimeout
		c.ConfigSources["Cursor.APITimeout"] = SourceJSONFile
	}
	if jsonConfig.CacheTimeout != 0 {
		c.Cursor.CacheTimeout = jsonConfig.CacheTimeout
		c.ConfigSources["Cursor.CacheTimeout"] = SourceJSONFile
	}
}

// mergeDaemonConfig merges Daemon configuration from JSON
func (c *AppConfig) mergeDaemonConfig(jsonConfig *DaemonConfig) {
	// Note: bool fields need special handling because zero value is false
	c.Daemon.Enabled = jsonConfig.Enabled
	c.ConfigSources["Daemon.Enabled"] = SourceJSONFile

	c.Daemon.StartAtLogin = jsonConfig.StartAtLogin
	c.ConfigSources["Daemon.StartAtLogin"] = SourceJSONFile

	c.Daemon.HideFromDock = jsonConfig.HideFromDock
	c.ConfigSources["Daemon.HideFromDock"] = SourceJSONFile

	if jsonConfig.LogPath != "" {
		c.Daemon.LogPath = jsonConfig.LogPath
		c.ConfigSources["Daemon.LogPath"] = SourceJSONFile
	}
	if jsonConfig.PidFile != "" {
		c.Daemon.PidFile = jsonConfig.PidFile
		c.ConfigSources["Daemon.PidFile"] = SourceJSONFile
	}
}

// mergeLoggingConfig merges Logging configuration from JSON
func (c *AppConfig) mergeLoggingConfig(jsonConfig *LoggingConfig) {
	if jsonConfig.Level != "" {
		c.Logging.Level = jsonConfig.Level
		c.ConfigSources["Logging.Level"] = SourceJSONFile
	}

	// Note: bool field
	c.Logging.Debug = jsonConfig.Debug
	c.ConfigSources["Logging.Debug"] = SourceJSONFile

	// Merge Promtail configuration
	if jsonConfig.Promtail != nil {
		if c.Logging.Promtail == nil {
			c.Logging.Promtail = &PromtailConfig{}
		}
		c.mergePromtailConfig(jsonConfig.Promtail)
	}
}

// mergePromtailConfig merges Promtail configuration from JSON
func (c *AppConfig) mergePromtailConfig(jsonConfig *PromtailConfig) {
	if jsonConfig.URL != "" {
		c.Logging.Promtail.URL = jsonConfig.URL
		c.ConfigSources["Promtail.URL"] = SourceJSONFile
	}
	if jsonConfig.Username != "" {
		c.Logging.Promtail.Username = jsonConfig.Username
		c.ConfigSources["Promtail.Username"] = SourceJSONFile
	}
	if jsonConfig.Password != "" {
		c.Logging.Promtail.Password = jsonConfig.Password
		c.ConfigSources["Promtail.Password"] = SourceJSONFile
	}
	if jsonConfig.BatchWaitSeconds != 0 {
		c.Logging.Promtail.BatchWaitSeconds = jsonConfig.BatchWaitSeconds
		c.ConfigSources["Promtail.BatchWaitSeconds"] = SourceJSONFile
	}
	if jsonConfig.BatchCapacity != 0 {
		c.Logging.Promtail.BatchCapacity = jsonConfig.BatchCapacity
		c.ConfigSources["Promtail.BatchCapacity"] = SourceJSONFile
	}
	if jsonConfig.TimeoutSeconds != 0 {
		c.Logging.Promtail.TimeoutSeconds = jsonConfig.TimeoutSeconds
		c.ConfigSources["Promtail.TimeoutSeconds"] = SourceJSONFile
	}
}

// mergeBedrockConfig merges Bedrock configuration from JSON
func (c *AppConfig) mergeBedrockConfig(jsonConfig *BedrockConfig) {
	// Note: bool fields need special handling because zero value is false
	c.Bedrock.Enabled = jsonConfig.Enabled
	c.ConfigSources["Bedrock.Enabled"] = SourceJSONFile

	if jsonConfig.AWSProfile != "" {
		c.Bedrock.AWSProfile = jsonConfig.AWSProfile
		c.ConfigSources["Bedrock.AWSProfile"] = SourceJSONFile
	}
	if jsonConfig.AssumeRoleARN != "" {
		c.Bedrock.AssumeRoleARN = jsonConfig.AssumeRoleARN
		c.ConfigSources["Bedrock.AssumeRoleARN"] = SourceJSONFile
	}
	if jsonConfig.CollectionIntervalSec != 0 {
		c.Bedrock.CollectionIntervalSec = jsonConfig.CollectionIntervalSec
		c.ConfigSources["Bedrock.CollectionIntervalSec"] = SourceJSONFile
	}
	if len(jsonConfig.Regions) > 0 {
		c.Bedrock.Regions = jsonConfig.Regions
		c.ConfigSources["Bedrock.Regions"] = SourceJSONFile
	}
}

// mergeVertexAIConfig merges VertexAI configuration from JSON
func (c *AppConfig) mergeVertexAIConfig(jsonConfig *VertexAIConfig) {
	// Note: bool fields need special handling because zero value is false
	c.VertexAI.Enabled = jsonConfig.Enabled
	c.ConfigSources["VertexAI.Enabled"] = SourceJSONFile

	if jsonConfig.ProjectID != "" {
		c.VertexAI.ProjectID = jsonConfig.ProjectID
		c.ConfigSources["VertexAI.ProjectID"] = SourceJSONFile
	}
	if jsonConfig.ServiceAccountKeyPath != "" {
		c.VertexAI.ServiceAccountKeyPath = jsonConfig.ServiceAccountKeyPath
		c.ConfigSources["VertexAI.ServiceAccountKeyPath"] = SourceJSONFile
	}
	if jsonConfig.ServiceAccountKey != "" {
		c.VertexAI.ServiceAccountKey = jsonConfig.ServiceAccountKey
		c.ConfigSources["VertexAI.ServiceAccountKey"] = SourceJSONFile
	}
	if jsonConfig.CollectionIntervalSec != 0 {
		c.VertexAI.CollectionIntervalSec = jsonConfig.CollectionIntervalSec
		c.ConfigSources["VertexAI.CollectionIntervalSec"] = SourceJSONFile
	}
}

// mergeCSVExportConfig merges CSVExport configuration from JSON
func (c *AppConfig) mergeCSVExportConfig(jsonConfig *CSVExportConfig) {
	if jsonConfig.DefaultOutputPath != "" {
		c.CSVExport.DefaultOutputPath = jsonConfig.DefaultOutputPath
		c.ConfigSources["CSVExport.DefaultOutputPath"] = SourceJSONFile
	}
	if jsonConfig.DefaultStartDays != 0 {
		c.CSVExport.DefaultStartDays = jsonConfig.DefaultStartDays
		c.ConfigSources["CSVExport.DefaultStartDays"] = SourceJSONFile
	}
	if jsonConfig.DefaultMetricTypes != "" {
		c.CSVExport.DefaultMetricTypes = jsonConfig.DefaultMetricTypes
		c.ConfigSources["CSVExport.DefaultMetricTypes"] = SourceJSONFile
	}
	if jsonConfig.MaxExportDays != 0 {
		c.CSVExport.MaxExportDays = jsonConfig.MaxExportDays
		c.ConfigSources["CSVExport.MaxExportDays"] = SourceJSONFile
	}
	if jsonConfig.TimeZone != "" {
		c.CSVExport.TimeZone = jsonConfig.TimeZone
		c.ConfigSources["CSVExport.TimeZone"] = SourceJSONFile
	}
}

// splitCommaSeparated splits a comma-separated string into a slice of strings
// It also trims whitespace from each element
func splitCommaSeparated(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// slicesEqual compares two string slices for equality
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
