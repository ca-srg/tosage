package usecase

import (
	"github.com/ca-srg/tosage/infrastructure/config"
)

// ConfigService は設定管理のサービスインターフェース
type ConfigService interface {
	// GetConfig は現在の設定を取得する
	GetConfig() *config.AppConfig

	// UpdateConfig は設定を更新する
	UpdateConfig(newConfig *config.AppConfig) error

	// GetConfigWithSources は設定とそのソース情報を取得する
	GetConfigWithSources() (*config.AppConfig, config.ConfigSourceMap)

	// SaveConfig は現在の設定をファイルに保存する
	SaveConfig() error

	// ReloadConfig は設定を再読み込みする
	ReloadConfig() error

	// GetConfigPath は設定ファイルのパスを返す
	GetConfigPath() string

	// CreateDefaultConfig はデフォルト設定ファイルを作成する
	CreateDefaultConfig() error

	// ExportConfig は現在の設定をエクスポート用に整形する（パスワードなどをマスク）
	ExportConfig() map[string]interface{}

	// EnsureConfigExists は設定ファイルが存在することを確認し、存在しない場合はテンプレートを作成する
	EnsureConfigExists() error

	// CreateTemplateConfig はテンプレート設定ファイルを作成する
	CreateTemplateConfig() error

	// LoadConfigWithFallback はエラー耐性のある設定読み込みを行う
	LoadConfigWithFallback() (*config.AppConfig, error)
}
