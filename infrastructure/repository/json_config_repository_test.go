package repository

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ca-srg/tosage/infrastructure/config"
)

func TestJSONConfigRepository_SaveAndLoad(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "tosage-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// テスト用のリポジトリを作成
	repo := &JSONConfigRepository{
		configDir:  tempDir,
		configFile: filepath.Join(tempDir, "config.json"),
	}

	// テスト用の設定
	testConfig := &config.AppConfig{
		ClaudePath: "/test/path",
		Prometheus: &config.PrometheusConfig{
			RemoteWriteURL: "http://test-prometheus:9090/api/v1/write",
			IntervalSec:    60,
			TimeoutSec:     10,
			Username:       "testuser",
			Password:       "testpass",
		},
		Logging: &config.LoggingConfig{
			Level: "info",
			Debug: false,
		},
	}

	// ファイルが存在しないことを確認
	exists, err := repo.Exists()
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if exists {
		t.Error("Config file should not exist initially")
	}

	// 設定を保存
	if err := repo.Save(testConfig); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// ファイルが存在することを確認
	exists, err = repo.Exists()
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if !exists {
		t.Error("Config file should exist after save")
	}

	// 設定を読み込み
	loadedConfig, err := repo.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 読み込んだ設定を検証
	if loadedConfig.ClaudePath != testConfig.ClaudePath {
		t.Errorf("ClaudePath mismatch: got %s, want %s", loadedConfig.ClaudePath, testConfig.ClaudePath)
	}
	if loadedConfig.Prometheus.RemoteWriteURL != testConfig.Prometheus.RemoteWriteURL {
		t.Errorf("Prometheus.RemoteWriteURL mismatch: got %s, want %s",
			loadedConfig.Prometheus.RemoteWriteURL, testConfig.Prometheus.RemoteWriteURL)
	}
}

func TestJSONConfigRepository_Backup(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "tosage-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	repo := &JSONConfigRepository{
		configDir:  tempDir,
		configFile: filepath.Join(tempDir, "config.json"),
	}

	// 初期設定を保存
	initialConfig := &config.AppConfig{
		ClaudePath: "/initial/path",
		Prometheus: &config.PrometheusConfig{
			RemoteWriteURL: "http://initial:9090",
			IntervalSec:    60,
			TimeoutSec:     10,
			Username:       "testuser",
			Password:       "testpass",
		},
	}
	if err := repo.Save(initialConfig); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	// 更新された設定を保存（これによりバックアップが作成されるはず）
	updatedConfig := &config.AppConfig{
		ClaudePath: "/updated/path",
		Prometheus: &config.PrometheusConfig{
			RemoteWriteURL: "http://updated:9090",
			IntervalSec:    120,
			TimeoutSec:     5,
			Username:       "testuser",
			Password:       "testpass",
		},
	}
	if err := repo.Save(updatedConfig); err != nil {
		t.Fatalf("Failed to save updated config: %v", err)
	}

	// バックアップファイルが存在することを確認
	pattern := repo.configFile + ".backup.*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("Failed to find backup files: %v", err)
	}
	if len(matches) == 0 {
		t.Error("No backup files found")
	}
}

func TestJSONConfigRepository_LoadNonExistent(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "tosage-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	repo := &JSONConfigRepository{
		configDir:  tempDir,
		configFile: filepath.Join(tempDir, "config.json"),
	}

	// 存在しないファイルを読み込む
	cfg, err := repo.Load()
	if err != nil {
		t.Fatalf("Load should not error for non-existent file: %v", err)
	}
	if cfg != nil {
		t.Error("Load should return nil for non-existent file")
	}
}

func TestJSONConfigRepository_Validate(t *testing.T) {
	repo := NewJSONConfigRepository()

	// nilの設定を検証
	err := repo.Validate(nil)
	if err == nil {
		t.Error("Validate should error for nil config")
	}

	// 有効な設定を検証
	validConfig := &config.AppConfig{
		Prometheus: &config.PrometheusConfig{
			RemoteWriteURL: "http://prometheus:9090",
			IntervalSec:    60,
			TimeoutSec:     10,
			Username:       "testuser",
			Password:       "testpass",
		},
	}
	err = repo.Validate(validConfig)
	if err != nil {
		t.Errorf("Validate should not error for valid config: %v", err)
	}

	// 無効な設定を検証（タイムアウトが0）
	invalidConfig := &config.AppConfig{
		Prometheus: &config.PrometheusConfig{
			RemoteWriteURL: "http://prometheus:9090",
			IntervalSec:    60,
			TimeoutSec:     0,
			Username:       "testuser",
			Password:       "testpass",
		},
	}
	err = repo.Validate(invalidConfig)
	if err == nil {
		t.Error("Validate should error for invalid config")
	}
}
