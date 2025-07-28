package impl

import (
	"context"
	"fmt"
	"sync"

	"github.com/ca-srg/tosage/domain"
	"github.com/ca-srg/tosage/domain/repository"
	"github.com/ca-srg/tosage/infrastructure/config"
	usecase "github.com/ca-srg/tosage/usecase/interface"
)

// ConfigServiceImpl は ConfigService の実装
type ConfigServiceImpl struct {
	configRepo       repository.ConfigRepository
	migrationService usecase.ConfigMigrationService
	config           *config.AppConfig
	logger           domain.Logger
	mu               sync.RWMutex
}

// NewConfigService は新しい ConfigService を作成する
func NewConfigService(configRepo repository.ConfigRepository, migrationService usecase.ConfigMigrationService, logger domain.Logger) (usecase.ConfigService, error) {
	// 設定を読み込む（ロガーとマイグレーションサービスを渡す）
	cfg, err := loadConfigWithMigration(configRepo, migrationService, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return &ConfigServiceImpl{
		configRepo:       configRepo,
		migrationService: migrationService,
		config:           cfg,
		logger:           logger,
	}, nil
}

// loadConfigWithJSON loads configuration from JSON file and environment variables
func loadConfigWithJSON(configRepo repository.ConfigRepository, logger domain.Logger) (*config.AppConfig, error) {
	// エラー耐性のある設定読み込みを使用
	return loadConfigWithFallback(configRepo, logger)
}

// loadConfigWithMigration loads configuration with migration support
func loadConfigWithMigration(configRepo repository.ConfigRepository, migrationService usecase.ConfigMigrationService, logger domain.Logger) (*config.AppConfig, error) {
	ctx := context.Background()

	// まず通常の設定読み込みを実行
	cfg, err := loadConfigWithFallback(configRepo, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// マイグレーションが必要かチェック
	if migrationService.NeedsMigration(cfg) {
		logger.Info(ctx, "Configuration migration required",
			domain.NewField("current_version", cfg.Version),
			domain.NewField("target_version", migrationService.GetCurrentVersion()))

		// マイグレーションを実行
		migratedCfg, err := migrationService.Migrate(cfg)
		if err != nil {
			logger.Error(ctx, "Configuration migration failed, using original configuration",
				domain.NewField("error", err.Error()))
			// マイグレーション失敗時は元の設定を使用（フォールバック）
			return cfg, nil
		}

		// マイグレーション成功時は新しい設定を保存
		if err := configRepo.Save(migratedCfg); err != nil {
			logger.Error(ctx, "Failed to save migrated configuration",
				domain.NewField("error", err.Error()))
			// 保存失敗してもマイグレーション済み設定を使用
		} else {
			logger.Info(ctx, "Migrated configuration saved successfully",
				domain.NewField("config_path", configRepo.GetConfigPath()))
		}

		return migratedCfg, nil
	}

	return cfg, nil
}

// loadConfigWithFallback loads configuration with fallback to defaults on errors
func loadConfigWithFallback(configRepo repository.ConfigRepository, logger domain.Logger) (*config.AppConfig, error) {
	ctx := context.Background()

	// Start with default configuration
	cfg := config.DefaultConfig()
	logger.Info(ctx, "Loading configuration with fallback", domain.NewField("config_path", configRepo.GetConfigPath()))

	// Mark all defaults
	cfg.MarkDefaults()
	logger.Debug(ctx, "Marked default configuration values")

	// Load from JSON file if it exists
	jsonConfig, err := configRepo.Load()
	if err != nil {
		// JSON読み込みエラーは無視してデフォルト設定で継続
		logger.Warn(ctx, "Failed to load JSON configuration, using defaults",
			domain.NewField("error", err.Error()),
			domain.NewField("config_path", configRepo.GetConfigPath()))
	} else if jsonConfig != nil {
		// Merge JSON configuration
		cfg.MergeJSONConfig(jsonConfig)
		logger.Info(ctx, "Successfully loaded JSON configuration",
			domain.NewField("config_path", configRepo.GetConfigPath()))
	} else {
		logger.Info(ctx, "No JSON configuration file found, using defaults",
			domain.NewField("config_path", configRepo.GetConfigPath()))
	}

	// Load environment variables (they override JSON values)
	if err := cfg.LoadFromEnv(); err != nil {
		// 環境変数のエラーは無視してデフォルト値で継続
		logger.Warn(ctx, "Failed to load environment variables, using fallback values",
			domain.NewField("error", err.Error()))
	} else {
		logger.Debug(ctx, "Successfully loaded environment variables")
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		// 検証エラーは無視してデフォルト設定で継続
		logger.Warn(ctx, "Configuration validation failed, using default values",
			domain.NewField("error", err.Error()))
	} else {
		logger.Info(ctx, "Configuration validation successful")
	}

	return cfg, nil
}

// GetConfig は現在の設定を取得する
func (s *ConfigServiceImpl) GetConfig() *config.AppConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 設定のコピーを返す（直接変更を防ぐため）
	return s.config
}

// UpdateConfig は設定を更新する
func (s *ConfigServiceImpl) UpdateConfig(newConfig *config.AppConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 新しい設定を検証
	if err := newConfig.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// 設定をファイルに保存
	if err := s.configRepo.Save(newConfig); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// メモリ内の設定を更新
	s.config = newConfig

	return nil
}

// GetConfigWithSources は設定とそのソース情報を取得する
func (s *ConfigServiceImpl) GetConfigWithSources() (*config.AppConfig, config.ConfigSourceMap) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.config, s.config.ConfigSources
}

// SaveConfig は現在の設定をファイルに保存する
func (s *ConfigServiceImpl) SaveConfig() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.configRepo.Save(s.config)
}

// ReloadConfig は設定を再読み込みする
func (s *ConfigServiceImpl) ReloadConfig() error {
	ctx := context.Background()
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info(ctx, "Reloading configuration")

	// 設定を再読み込み（マイグレーション対応）
	newConfig, err := loadConfigWithMigration(s.configRepo, s.migrationService, s.logger)
	if err != nil {
		s.logger.Error(ctx, "Failed to reload configuration",
			domain.NewField("error", err.Error()))
		return fmt.Errorf("failed to reload config: %w", err)
	}

	s.config = newConfig
	s.logger.Info(ctx, "Configuration reloaded successfully")
	return nil
}

// GetConfigPath は設定ファイルのパスを返す
func (s *ConfigServiceImpl) GetConfigPath() string {
	return s.configRepo.GetConfigPath()
}

// CreateDefaultConfig はデフォルト設定ファイルを作成する
func (s *ConfigServiceImpl) CreateDefaultConfig() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 設定ファイルが既に存在する場合はエラー
	exists, err := s.configRepo.Exists()
	if err != nil {
		return fmt.Errorf("failed to check config existence: %w", err)
	}
	if exists {
		return fmt.Errorf("config file already exists at %s", s.configRepo.GetConfigPath())
	}

	// デフォルト設定を作成
	defaultConfig := config.MinimalDefaultConfig()

	// デフォルト設定を保存
	if err := s.configRepo.Save(defaultConfig); err != nil {
		return fmt.Errorf("failed to save default config: %w", err)
	}

	// メモリ内の設定も更新
	s.config = defaultConfig

	return nil
}

// ExportConfig は現在の設定をエクスポート用に整形する（パスワードなどをマスク）
func (s *ConfigServiceImpl) ExportConfig() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 設定をマップに変換
	exportMap := make(map[string]interface{})

	// 基本設定
	exportMap["claude_path"] = s.config.ClaudePath

	// Prometheus設定
	if s.config.Prometheus != nil {
		prometheusMap := make(map[string]interface{})
		prometheusMap["remote_write_url"] = s.config.Prometheus.RemoteWriteURL
		prometheusMap["host_label"] = s.config.Prometheus.HostLabel
		prometheusMap["interval_seconds"] = s.config.Prometheus.IntervalSec
		prometheusMap["timeout_seconds"] = s.config.Prometheus.TimeoutSec
		// Remote Write認証情報
		prometheusMap["remote_write_username"] = s.config.Prometheus.RemoteWriteUsername
		// パスワードはマスク
		if s.config.Prometheus.RemoteWritePassword != "" {
			prometheusMap["remote_write_password"] = "****"
		}
		// Query認証情報
		prometheusMap["url"] = s.config.Prometheus.URL
		prometheusMap["username"] = s.config.Prometheus.Username
		// パスワードはマスク
		if s.config.Prometheus.Password != "" {
			prometheusMap["password"] = "****"
		}
		exportMap["prometheus"] = prometheusMap
	}

	// Cursor設定
	if s.config.Cursor != nil {
		cursorMap := make(map[string]interface{})
		cursorMap["database_path"] = s.config.Cursor.DatabasePath
		cursorMap["api_timeout"] = s.config.Cursor.APITimeout
		cursorMap["cache_timeout"] = s.config.Cursor.CacheTimeout
		exportMap["cursor"] = cursorMap
	}

	// Daemon設定
	if s.config.Daemon != nil {
		daemonMap := make(map[string]interface{})
		daemonMap["enabled"] = s.config.Daemon.Enabled
		daemonMap["start_at_login"] = s.config.Daemon.StartAtLogin
		daemonMap["log_path"] = s.config.Daemon.LogPath
		daemonMap["pid_file"] = s.config.Daemon.PidFile
		exportMap["daemon"] = daemonMap
	}

	// Logging設定
	if s.config.Logging != nil {
		loggingMap := make(map[string]interface{})
		loggingMap["level"] = s.config.Logging.Level
		loggingMap["debug"] = s.config.Logging.Debug

		// Promtail設定
		if s.config.Logging.Promtail != nil {
			promtailMap := make(map[string]interface{})
			promtailMap["url"] = s.config.Logging.Promtail.URL
			promtailMap["username"] = s.config.Logging.Promtail.Username
			// パスワードはマスク
			if s.config.Logging.Promtail.Password != "" {
				promtailMap["password"] = "****"
			}
			promtailMap["batch_wait_seconds"] = s.config.Logging.Promtail.BatchWaitSeconds
			promtailMap["batch_capacity"] = s.config.Logging.Promtail.BatchCapacity
			promtailMap["timeout_seconds"] = s.config.Logging.Promtail.TimeoutSeconds
			loggingMap["promtail"] = promtailMap
		}
		exportMap["logging"] = loggingMap
	}

	// ソース情報を追加
	sourcesMap := make(map[string]string)
	for key, source := range s.config.ConfigSources {
		sourcesMap[key] = string(source)
	}
	exportMap["_sources"] = sourcesMap

	return exportMap
}

// EnsureConfigExists は設定ファイルが存在することを確認し、存在しない場合はテンプレートを作成する
func (s *ConfigServiceImpl) EnsureConfigExists() error {
	ctx := context.Background()
	s.mu.Lock()
	defer s.mu.Unlock()

	// 設定ファイルの存在確認
	configPath := s.configRepo.GetConfigPath()
	s.logger.Debug(ctx, "Checking if configuration file exists",
		domain.NewField("config_path", configPath))

	exists, err := s.configRepo.Exists()
	if err != nil {
		s.logger.Error(ctx, "Failed to check config existence",
			domain.NewField("error", err.Error()),
			domain.NewField("config_path", configPath))
		return fmt.Errorf("failed to check config existence: %w", err)
	}

	// 設定ファイルが既に存在する場合は何もしない
	if exists {
		s.logger.Debug(ctx, "Configuration file already exists",
			domain.NewField("config_path", configPath))
		return nil
	}

	// 設定ファイルが存在しない場合はテンプレートを作成
	s.logger.Info(ctx, "Configuration file not found, creating template",
		domain.NewField("config_path", configPath))

	defaultConfig := config.MinimalDefaultConfig()
	if err := s.configRepo.Save(defaultConfig); err != nil {
		s.logger.Error(ctx, "Failed to create template configuration",
			domain.NewField("error", err.Error()),
			domain.NewField("config_path", configPath))
		return fmt.Errorf("failed to create template config: %w", err)
	}

	// メモリ内の設定も更新
	s.config = defaultConfig
	s.logger.Info(ctx, "Template configuration created successfully",
		domain.NewField("config_path", configPath))

	return nil
}

// CreateTemplateConfig はテンプレート設定ファイルを作成する
func (s *ConfigServiceImpl) CreateTemplateConfig() error {
	ctx := context.Background()
	s.mu.Lock()
	defer s.mu.Unlock()

	configPath := s.configRepo.GetConfigPath()
	s.logger.Info(ctx, "Creating template configuration file",
		domain.NewField("config_path", configPath))

	// デフォルト設定を作成
	defaultConfig := config.MinimalDefaultConfig()

	// テンプレート設定を保存
	if err := s.configRepo.Save(defaultConfig); err != nil {
		s.logger.Error(ctx, "Failed to save template configuration",
			domain.NewField("error", err.Error()),
			domain.NewField("config_path", configPath))
		return fmt.Errorf("failed to save template config: %w", err)
	}

	s.logger.Info(ctx, "Template configuration file created successfully",
		domain.NewField("config_path", configPath))
	return nil
}

// LoadConfigWithFallback はエラー耐性のある設定読み込みを行う
func (s *ConfigServiceImpl) LoadConfigWithFallback() (*config.AppConfig, error) {
	// エラー耐性のある設定読み込みを使用
	return loadConfigWithFallback(s.configRepo, s.logger)
}

// LoadConfigWithMigration はマイグレーション対応の設定読み込みを行う
func (s *ConfigServiceImpl) LoadConfigWithMigration() (*config.AppConfig, error) {
	// マイグレーション対応の設定読み込みを使用
	return loadConfigWithMigration(s.configRepo, s.migrationService, s.logger)
}
