package usecase

import (
	"github.com/ca-srg/tosage/infrastructure/config"
)

// ConfigMigrationService は設定マイグレーションサービスのインターフェース
type ConfigMigrationService interface {
	// NeedsMigration は設定がマイグレーションを必要とするかチェックする
	NeedsMigration(config *config.AppConfig) bool

	// Migrate はレガシー形式から現在の形式への移行を実行する
	Migrate(config *config.AppConfig) (*config.AppConfig, error)

	// GetCurrentVersion は現在の設定バージョンを返す
	GetCurrentVersion() int
}
