package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ca-srg/tosage/domain/repository"
	"github.com/ca-srg/tosage/infrastructure/config"
)

// JSONConfigRepository は JSON形式で設定を管理するリポジトリ実装
type JSONConfigRepository struct {
	configDir  string
	configFile string
}

// NewJSONConfigRepository は新しい JSONConfigRepository を作成する
func NewJSONConfigRepository() repository.ConfigRepository {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "tosage")
	return &JSONConfigRepository{
		configDir:  configDir,
		configFile: filepath.Join(configDir, "config.json"),
	}
}

// SetConfigDir はテスト用に設定ディレクトリを設定する
func (r *JSONConfigRepository) SetConfigDir(dir string) {
	r.configDir = dir
}

// SetConfigFile はテスト用に設定ファイルパスを設定する
func (r *JSONConfigRepository) SetConfigFile(file string) {
	r.configFile = file
}

// Exists は設定ファイルが存在するかどうかを確認する
func (r *JSONConfigRepository) Exists() (bool, error) {
	_, err := os.Stat(r.configFile)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("failed to check config file existence: %w", err)
}

// Load は設定ファイルから設定を読み込む
func (r *JSONConfigRepository) Load() (*config.AppConfig, error) {
	exists, err := r.Exists()
	if err != nil {
		return nil, err
	}
	if !exists {
		// ファイルが存在しない場合はnilを返す（エラーではない）
		return nil, nil
	}

	// ファイルのセキュリティチェック
	if err := r.ensureSecurePermissions(r.configFile, false); err != nil {
		return nil, fmt.Errorf("config file security check failed: %w", err)
	}

	data, err := os.ReadFile(r.configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg config.AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// Save は設定をファイルに保存する
func (r *JSONConfigRepository) Save(cfg *config.AppConfig) error {
	if err := r.EnsureConfigDir(); err != nil {
		return err
	}

	if err := r.Validate(cfg); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// 既存ファイルがある場合はバックアップを作成
	exists, err := r.Exists()
	if err != nil {
		return err
	}
	if exists {
		if err := r.Backup(); err != nil {
			// バックアップ失敗はログに記録するが、保存は続行
			fmt.Fprintf(os.Stderr, "Warning: failed to create backup: %v\n", err)
		}
	}

	// JSONにマーシャル（インデント付き）
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// 一時ファイルに書き込んでからアトミックに置き換え
	tmpFile := r.configFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp config file: %w", err)
	}

	if err := os.Rename(tmpFile, r.configFile); err != nil {
		_ = os.Remove(tmpFile) // クリーンアップ
		return fmt.Errorf("failed to save config file: %w", err)
	}

	// 設定ファイルのパーミッションを確認・修正
	if err := r.ensureSecurePermissions(r.configFile, false); err != nil {
		return fmt.Errorf("failed to secure config file: %w", err)
	}

	// 設定の整合性チェック
	if err := r.validateConfigIntegrity(cfg); err != nil {
		// 整合性チェックのエラーは警告として扱う
		fmt.Fprintf(os.Stderr, "Warning: config integrity check failed: %v\n", err)
	}

	return nil
}

// GetConfigPath は設定ファイルのパスを返す
func (r *JSONConfigRepository) GetConfigPath() string {
	return r.configFile
}

// EnsureConfigDir は設定ディレクトリが存在することを保証する
func (r *JSONConfigRepository) EnsureConfigDir() error {
	if err := os.MkdirAll(r.configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// ディレクトリのパーミッションを確認・修正
	if err := r.ensureSecurePermissions(r.configDir, true); err != nil {
		return fmt.Errorf("failed to secure config directory: %w", err)
	}

	return nil
}

// Backup は現在の設定ファイルのバックアップを作成する
func (r *JSONConfigRepository) Backup() error {
	exists, err := r.Exists()
	if err != nil {
		return err
	}
	if !exists {
		return nil // バックアップするものがない
	}

	timestamp := time.Now().Format("20060102-150405")
	backupFile := fmt.Sprintf("%s.backup.%s", r.configFile, timestamp)

	data, err := os.ReadFile(r.configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file for backup: %w", err)
	}

	if err := os.WriteFile(backupFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	// 古いバックアップファイルの削除（最新5個を保持）
	if err := r.cleanupOldBackups(); err != nil {
		// クリーンアップの失敗は警告として扱う
		fmt.Fprintf(os.Stderr, "Warning: failed to cleanup old backups: %v\n", err)
	}

	return nil
}

// Validate は設定内容の妥当性を検証する
func (r *JSONConfigRepository) Validate(cfg *config.AppConfig) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	// AppConfig の既存の Validate メソッドを使用
	return cfg.Validate()
}

// cleanupOldBackups は古いバックアップファイルを削除する
func (r *JSONConfigRepository) cleanupOldBackups() error {
	pattern := r.configFile + ".backup.*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	if len(matches) <= 5 {
		return nil // 削除するものがない
	}

	// 古い順にソート（ファイル名でソート）
	for i := 0; i < len(matches)-5; i++ {
		if err := os.Remove(matches[i]); err != nil {
			// 個別のファイル削除失敗は続行
			fmt.Fprintf(os.Stderr, "Warning: failed to remove old backup %s: %v\n", matches[i], err)
		}
	}

	return nil
}

// ensureSecurePermissions はファイルまたはディレクトリの権限を確保する
func (r *JSONConfigRepository) ensureSecurePermissions(path string, isDir bool) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	}

	currentMode := info.Mode().Perm()
	var expectedMode os.FileMode
	if isDir {
		expectedMode = 0700 // rwx------
	} else {
		expectedMode = 0600 // rw-------
	}

	// パーミッションが期待値と異なる場合は修正
	if currentMode != expectedMode {
		if err := os.Chmod(path, expectedMode); err != nil {
			return fmt.Errorf("failed to set permissions: %w", err)
		}
	}

	// 所有者の確認（Unix系OSのみ）
	if err := r.checkOwnership(path); err != nil {
		return fmt.Errorf("ownership check failed: %w", err)
	}

	return nil
}

// checkOwnership はファイルの所有者が現在のユーザーであることを確認する
func (r *JSONConfigRepository) checkOwnership(path string) error {
	// Unix系OSでの所有者チェック
	// Windowsでは常に成功を返す
	fileInfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	// システム固有の情報を取得
	stat, ok := fileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		// Windows等、Unix系でないOSの場合はスキップ
		return nil
	}

	// 現在のユーザーIDを取得
	currentUID := uint32(os.Getuid())

	// ファイルの所有者が現在のユーザーでない場合はエラー
	if stat.Uid != currentUID {
		return fmt.Errorf("file is not owned by current user (uid: %d, expected: %d)", stat.Uid, currentUID)
	}

	return nil
}

// validateConfigIntegrity は設定ファイルの整合性を検証する
func (r *JSONConfigRepository) validateConfigIntegrity(cfg *config.AppConfig) error {
	// パスワードフィールドが平文で保存されていないか確認
	if cfg.Prometheus != nil {
		// パスワードフィールドに特定のパターンがないか確認
		if cfg.Prometheus.Password != "" && len(cfg.Prometheus.Password) < 8 {
			// 短すぎるパスワードは警告
			fmt.Fprintf(os.Stderr, "Warning: Prometheus password appears to be weak\n")
		}
	}

	if cfg.Logging != nil && cfg.Logging.Promtail != nil {
		if cfg.Logging.Promtail.Password != "" && len(cfg.Logging.Promtail.Password) < 8 {
			fmt.Fprintf(os.Stderr, "Warning: Promtail password appears to be weak\n")
		}
	}

	return nil
}
