package impl

import (
	"context"
	"fmt"

	"github.com/ca-srg/tosage/domain"
	"github.com/ca-srg/tosage/infrastructure/config"
	usecase "github.com/ca-srg/tosage/usecase/interface"
)

// ConfigMigrationServiceImpl は ConfigMigrationService の実装
type ConfigMigrationServiceImpl struct {
	logger domain.Logger
}

// NewConfigMigrationService は新しい ConfigMigrationService を作成する
func NewConfigMigrationService(logger domain.Logger) usecase.ConfigMigrationService {
	return &ConfigMigrationServiceImpl{
		logger: logger,
	}
}

// NeedsMigration は設定がマイグレーションを必要とするかチェックする
func (s *ConfigMigrationServiceImpl) NeedsMigration(cfg *config.AppConfig) bool {
	// バージョンフィールドが存在しない、または0の場合はマイグレーションが必要
	return cfg.Version == 0
}

// GetCurrentVersion は現在の設定バージョンを返す
func (s *ConfigMigrationServiceImpl) GetCurrentVersion() int {
	return 1
}

// Migrate はレガシー形式から現在の形式への移行を実行する
func (s *ConfigMigrationServiceImpl) Migrate(cfg *config.AppConfig) (*config.AppConfig, error) {
	ctx := context.Background()

	// すでに最新バージョンの場合はそのまま返す
	if !s.NeedsMigration(cfg) {
		s.logger.Debug(ctx, "Configuration is already at current version",
			domain.NewField("version", cfg.Version))
		return cfg, nil
	}

	s.logger.Info(ctx, "Starting configuration migration",
		domain.NewField("from_version", cfg.Version),
		domain.NewField("to_version", s.GetCurrentVersion()))

	// 設定のコピーを作成（元の設定を変更しないため）
	migratedCfg := s.copyConfig(cfg)

	// マイグレーションを実行
	if err := s.migrateV0ToV1(migratedCfg); err != nil {
		s.logger.Error(ctx, "Failed to migrate configuration",
			domain.NewField("error", err.Error()))
		return nil, fmt.Errorf("failed to migrate configuration: %w", err)
	}

	// マイグレーション後の検証
	if err := s.validateMigratedConfig(migratedCfg); err != nil {
		s.logger.Error(ctx, "Migrated configuration validation failed",
			domain.NewField("error", err.Error()))
		return nil, fmt.Errorf("migrated configuration validation failed: %w", err)
	}

	s.logger.Info(ctx, "Configuration migration completed successfully",
		domain.NewField("new_version", migratedCfg.Version))

	return migratedCfg, nil
}

// migrateV0ToV1 はバージョン0から1へのマイグレーションを実行する
func (s *ConfigMigrationServiceImpl) migrateV0ToV1(cfg *config.AppConfig) error {
	ctx := context.Background()

	// Prometheusフィールドの移行
	if cfg.Prometheus != nil {
		s.logger.Debug(ctx, "Migrating Prometheus configuration fields")

		// 既存のusername/passwordフィールドをremote_write_username/remote_write_passwordに移行
		// ただし、新しいフィールドがまだ設定されていない場合のみ（環境変数を優先）
		if cfg.Prometheus.Username != "" && cfg.Prometheus.RemoteWriteUsername == "" {
			cfg.Prometheus.RemoteWriteUsername = cfg.Prometheus.Username
			s.logger.Debug(ctx, "Migrated prometheus.username to prometheus.remote_write_username")
		}
		// 古いフィールドは常にクリア
		cfg.Prometheus.Username = ""

		if cfg.Prometheus.Password != "" && cfg.Prometheus.RemoteWritePassword == "" {
			cfg.Prometheus.RemoteWritePassword = cfg.Prometheus.Password
			s.logger.Debug(ctx, "Migrated prometheus.password to prometheus.remote_write_password")
		}
		// 古いフィールドは常にクリア
		cfg.Prometheus.Password = ""
	}

	// バージョンフィールドを設定
	cfg.Version = 1
	s.logger.Debug(ctx, "Set configuration version to 1")

	return nil
}

// validateMigratedConfig はマイグレーション後の設定を検証する
func (s *ConfigMigrationServiceImpl) validateMigratedConfig(cfg *config.AppConfig) error {
	// バージョンが正しく設定されているか確認
	if cfg.Version != s.GetCurrentVersion() {
		return fmt.Errorf("invalid version after migration: expected %d, got %d",
			s.GetCurrentVersion(), cfg.Version)
	}

	// Prometheusの設定が正しく移行されているか確認
	if cfg.Prometheus != nil && cfg.Prometheus.RemoteWriteURL != "" {
		// RemoteWriteURLが設定されている場合、認証情報も必要
		if cfg.Prometheus.RemoteWriteUsername == "" || cfg.Prometheus.RemoteWritePassword == "" {
			return fmt.Errorf("remote write authentication is required when remote write URL is set")
		}
	}

	return nil
}

// copyConfig は設定のディープコピーを作成する
func (s *ConfigMigrationServiceImpl) copyConfig(src *config.AppConfig) *config.AppConfig {
	// 新しいAppConfigインスタンスを作成
	dst := &config.AppConfig{
		Version:       src.Version,
		ClaudePath:    src.ClaudePath,
		ConfigSources: make(config.ConfigSourceMap),
	}

	// ConfigSourcesをコピー
	for k, v := range src.ConfigSources {
		dst.ConfigSources[k] = v
	}

	// Prometheus設定をコピー
	if src.Prometheus != nil {
		dst.Prometheus = &config.PrometheusConfig{
			RemoteWriteURL:      src.Prometheus.RemoteWriteURL,
			RemoteWriteUsername: src.Prometheus.RemoteWriteUsername,
			RemoteWritePassword: src.Prometheus.RemoteWritePassword,
			URL:                 src.Prometheus.URL,
			Username:            src.Prometheus.Username,
			Password:            src.Prometheus.Password,
			HostLabel:           src.Prometheus.HostLabel,
			IntervalSec:         src.Prometheus.IntervalSec,
			TimeoutSec:          src.Prometheus.TimeoutSec,
		}
	}

	// Cursor設定をコピー
	if src.Cursor != nil {
		dst.Cursor = &config.CursorConfig{
			DatabasePath: src.Cursor.DatabasePath,
			APITimeout:   src.Cursor.APITimeout,
			CacheTimeout: src.Cursor.CacheTimeout,
		}
	}

	// Bedrock設定をコピー
	if src.Bedrock != nil {
		dst.Bedrock = &config.BedrockConfig{
			Enabled:               src.Bedrock.Enabled,
			Regions:               append([]string{}, src.Bedrock.Regions...),
			AWSProfile:            src.Bedrock.AWSProfile,
			AssumeRoleARN:         src.Bedrock.AssumeRoleARN,
			CollectionIntervalSec: src.Bedrock.CollectionIntervalSec,
		}
	}

	// VertexAI設定をコピー
	if src.VertexAI != nil {
		dst.VertexAI = &config.VertexAIConfig{
			Enabled:               src.VertexAI.Enabled,
			ProjectID:             src.VertexAI.ProjectID,
			ServiceAccountKeyPath: src.VertexAI.ServiceAccountKeyPath,
			CollectionIntervalSec: src.VertexAI.CollectionIntervalSec,
		}
	}

	// Daemon設定をコピー
	if src.Daemon != nil {
		dst.Daemon = &config.DaemonConfig{
			Enabled:      src.Daemon.Enabled,
			StartAtLogin: src.Daemon.StartAtLogin,
			HideFromDock: src.Daemon.HideFromDock,
			LogPath:      src.Daemon.LogPath,
			PidFile:      src.Daemon.PidFile,
		}
	}

	// Logging設定をコピー
	if src.Logging != nil {
		dst.Logging = &config.LoggingConfig{
			Level: src.Logging.Level,
			Debug: src.Logging.Debug,
		}
		if src.Logging.Promtail != nil {
			dst.Logging.Promtail = &config.PromtailConfig{
				URL:              src.Logging.Promtail.URL,
				Username:         src.Logging.Promtail.Username,
				Password:         src.Logging.Promtail.Password,
				BatchWaitSeconds: src.Logging.Promtail.BatchWaitSeconds,
				BatchCapacity:    src.Logging.Promtail.BatchCapacity,
				TimeoutSeconds:   src.Logging.Promtail.TimeoutSeconds,
			}
		}
	}

	// CSVExport設定をコピー
	if src.CSVExport != nil {
		dst.CSVExport = &config.CSVExportConfig{
			DefaultOutputPath:  src.CSVExport.DefaultOutputPath,
			DefaultStartDays:   src.CSVExport.DefaultStartDays,
			DefaultMetricTypes: src.CSVExport.DefaultMetricTypes,
			MaxExportDays:      src.CSVExport.MaxExportDays,
			TimeZone:           src.CSVExport.TimeZone,
		}
	}

	return dst
}
