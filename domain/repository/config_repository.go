package repository

import (
	"github.com/ca-srg/tosage/infrastructure/config"
)

// ConfigRepository は設定ファイルの読み書きを管理するリポジトリインターフェース
type ConfigRepository interface {
	// Exists は設定ファイルが存在するかどうかを確認する
	Exists() (bool, error)

	// Load は設定ファイルから設定を読み込む
	Load() (*config.AppConfig, error)

	// Save は設定をファイルに保存する
	Save(config *config.AppConfig) error

	// GetConfigPath は設定ファイルのパスを返す
	GetConfigPath() string

	// EnsureConfigDir は設定ディレクトリが存在することを保証する
	EnsureConfigDir() error

	// Validate は設定内容の妥当性を検証する
	Validate(config *config.AppConfig) error
}
