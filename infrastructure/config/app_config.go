package config

import (
	"fmt"
	"os"

	"github.com/Netflix/go-env"
)

// PrometheusConfig holds Prometheus integration configuration
type PrometheusConfig struct {
	// RemoteWriteURL is the Prometheus Remote Write endpoint URL
	RemoteWriteURL string `json:"remote_write_url" env:"TOSAGE_PROMETHEUS_REMOTE_WRITE_URL"`

	// HostLabel is the host label value for metrics
	HostLabel string `json:"host_label,omitempty" env:"TOSAGE_PROMETHEUS_HOST_LABEL"`

	// IntervalSec is the interval in seconds between metric pushes
	IntervalSec int `json:"interval_seconds,omitempty" env:"TOSAGE_PROMETHEUS_INTERVAL_SECONDS,default=600"`

	// TimeoutSec is the timeout in seconds for metric pushes
	TimeoutSec int `json:"timeout_seconds,omitempty" env:"TOSAGE_PROMETHEUS_TIMEOUT_SECONDS,default=30"`

	// Basic authentication (required when RemoteWriteURL is set)
	Username string `json:"username" env:"TOSAGE_PROMETHEUS_USERNAME"`
	Password string `json:"password" env:"TOSAGE_PROMETHEUS_PASSWORD"`
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

// DaemonConfig holds daemon mode configuration
type DaemonConfig struct {
	// Enabled indicates whether daemon mode is enabled
	Enabled bool `json:"enabled,omitempty" env:"TOSAGE_DAEMON_ENABLED,default=false"`

	// StartAtLogin indicates whether to start at system login
	StartAtLogin bool `json:"start_at_login,omitempty" env:"TOSAGE_DAEMON_START_AT_LOGIN,default=false"`

	// LogPath is the path for daemon log files
	LogPath string `json:"log_path,omitempty" env:"TOSAGE_DAEMON_LOG_PATH,default=/tmp/tosage.log"`

	// PidFile is the path for the daemon PID file
	PidFile string `json:"pid_file,omitempty" env:"TOSAGE_DAEMON_PID_FILE,default=/tmp/tosage.pid"`
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
	// ClaudePath is the custom path to Claude data directory
	ClaudePath string `json:"claude_path,omitempty" env:"TOSAGE_CLAUDE_PATH"`

	// Prometheus holds Prometheus integration configuration
	Prometheus *PrometheusConfig `json:"prometheus,omitempty"`

	// Cursor holds Cursor integration configuration
	Cursor *CursorConfig `json:"cursor,omitempty"`

	// Daemon holds daemon mode configuration
	Daemon *DaemonConfig `json:"daemon,omitempty"`

	// Logging holds logging configuration
	Logging *LoggingConfig `json:"logging,omitempty"`

	// ConfigSources tracks the source of each configuration field
	ConfigSources ConfigSourceMap `json:"-"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *AppConfig {
	return &AppConfig{
		ClaudePath: "",
		Prometheus: &PrometheusConfig{
			RemoteWriteURL: "http://localhost:9090/api/v1/write", // デフォルトのPrometheus URL
			HostLabel:      "",
			IntervalSec:    600, // 10 minutes
			TimeoutSec:     30,
		},
		Cursor: &CursorConfig{
			DatabasePath: "",
			APITimeout:   30,  // 30 seconds
			CacheTimeout: 300, // 5 minutes
		},
		Daemon: &DaemonConfig{
			Enabled:      false,
			StartAtLogin: false,
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
		ConfigSources: make(ConfigSourceMap),
	}
}

// MinimalDefaultConfig returns the minimal configuration template for initial setup
func MinimalDefaultConfig() *AppConfig {
	return &AppConfig{
		Prometheus: &PrometheusConfig{
			RemoteWriteURL: "",
			Username:       "",
			Password:       "",
		},
		Logging: &LoggingConfig{
			Promtail: &PromtailConfig{
				URL:      "",
				Username: "",
				Password: "",
			},
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
			RemoteWriteURL: c.Prometheus.RemoteWriteURL,
			HostLabel:      c.Prometheus.HostLabel,
			IntervalSec:    c.Prometheus.IntervalSec,
			TimeoutSec:     c.Prometheus.TimeoutSec,
			Username:       c.Prometheus.Username,
			Password:       c.Prometheus.Password,
		}
	}
	if c.Cursor != nil {
		original.Cursor = &CursorConfig{
			DatabasePath: c.Cursor.DatabasePath,
			APITimeout:   c.Cursor.APITimeout,
			CacheTimeout: c.Cursor.CacheTimeout,
		}
	}
	if c.Daemon != nil {
		original.Daemon = &DaemonConfig{
			Enabled:      c.Daemon.Enabled,
			StartAtLogin: c.Daemon.StartAtLogin,
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

	return nil
}

// trackPrometheusEnvOverrides tracks environment variable overrides for Prometheus config
func (c *AppConfig) trackPrometheusEnvOverrides(original *PrometheusConfig) {
	if original == nil {
		return
	}
	if c.Prometheus.RemoteWriteURL != original.RemoteWriteURL && os.Getenv("TOSAGE_PROMETHEUS_REMOTE_WRITE_URL") != "" {
		c.ConfigSources["Prometheus.RemoteWriteURL"] = SourceEnvironment
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
	if c.Prometheus.Username != original.Username && os.Getenv("TOSAGE_PROMETHEUS_USERNAME") != "" {
		c.ConfigSources["Prometheus.Username"] = SourceEnvironment
	}
	if c.Prometheus.Password != original.Password && os.Getenv("TOSAGE_PROMETHEUS_PASSWORD") != "" {
		c.ConfigSources["Prometheus.Password"] = SourceEnvironment
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

	// Validate basic authentication is provided
	if c.Prometheus.Username == "" || c.Prometheus.Password == "" {
		return fmt.Errorf("basic auth username and password are required when remote write URL is set")
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

// MarkDefaults marks all configuration fields as coming from defaults
func (c *AppConfig) MarkDefaults() {
	c.ConfigSources["ClaudePath"] = SourceDefault
	c.ConfigSources["Prometheus.RemoteWriteURL"] = SourceDefault
	c.ConfigSources["Prometheus.HostLabel"] = SourceDefault
	c.ConfigSources["Prometheus.IntervalSec"] = SourceDefault
	c.ConfigSources["Prometheus.TimeoutSec"] = SourceDefault
	c.ConfigSources["Prometheus.Username"] = SourceDefault
	c.ConfigSources["Prometheus.Password"] = SourceDefault
	c.ConfigSources["Cursor.DatabasePath"] = SourceDefault
	c.ConfigSources["Cursor.APITimeout"] = SourceDefault
	c.ConfigSources["Cursor.CacheTimeout"] = SourceDefault
	c.ConfigSources["Daemon.Enabled"] = SourceDefault
	c.ConfigSources["Daemon.StartAtLogin"] = SourceDefault
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
}

// MergeJSONConfig merges JSON configuration into the current configuration
func (c *AppConfig) MergeJSONConfig(jsonConfig *AppConfig) {
	// Merge top-level fields
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
}

// mergePrometheusConfig merges Prometheus configuration from JSON
func (c *AppConfig) mergePrometheusConfig(jsonConfig *PrometheusConfig) {
	if jsonConfig.RemoteWriteURL != "" {
		c.Prometheus.RemoteWriteURL = jsonConfig.RemoteWriteURL
		c.ConfigSources["Prometheus.RemoteWriteURL"] = SourceJSONFile
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
	if jsonConfig.Username != "" {
		c.Prometheus.Username = jsonConfig.Username
		c.ConfigSources["Prometheus.Username"] = SourceJSONFile
	}
	if jsonConfig.Password != "" {
		c.Prometheus.Password = jsonConfig.Password
		c.ConfigSources["Prometheus.Password"] = SourceJSONFile
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
