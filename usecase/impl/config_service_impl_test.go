package impl

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ca-srg/tosage/domain"
	"github.com/ca-srg/tosage/infrastructure/config"
	"github.com/ca-srg/tosage/infrastructure/repository"
)

// MockLogger is a test mock for domain.Logger
type MockLogger struct{}

func (m *MockLogger) Debug(ctx context.Context, msg string, fields ...domain.Field) {}
func (m *MockLogger) Info(ctx context.Context, msg string, fields ...domain.Field)  {}
func (m *MockLogger) Warn(ctx context.Context, msg string, fields ...domain.Field)  {}
func (m *MockLogger) Error(ctx context.Context, msg string, fields ...domain.Field) {}
func (m *MockLogger) WithFields(fields ...domain.Field) domain.Logger {
	return m
}

func TestConfigServiceImpl_GetConfig(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "tosage-config-service-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// テスト用のリポジトリを作成
	configRepo := &repository.JSONConfigRepository{}
	// テスト用にパスを設定
	configRepo.SetConfigDir(tempDir)
	configRepo.SetConfigFile(filepath.Join(tempDir, "config.json"))

	// ConfigService を作成
	mockLogger := &MockLogger{}
	migrationService := NewConfigMigrationService(mockLogger)
	service, err := NewConfigService(configRepo, migrationService, mockLogger)
	if err != nil {
		t.Fatalf("Failed to create config service: %v", err)
	}

	// 設定を取得
	cfg := service.GetConfig()
	if cfg == nil {
		t.Fatal("GetConfig returned nil")
	}

	// デフォルト値を確認
}

func TestConfigServiceImpl_UpdateConfig(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "tosage-config-service-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// テスト用のリポジトリを作成
	configRepo := repository.NewJSONConfigRepository()
	// テスト用にパスを上書き
	repo := configRepo.(*repository.JSONConfigRepository)
	repo.SetConfigDir(tempDir)
	repo.SetConfigFile(filepath.Join(tempDir, "config.json"))

	// ConfigService を作成
	mockLogger := &MockLogger{}
	migrationService := NewConfigMigrationService(mockLogger)
	service, err := NewConfigService(configRepo, migrationService, mockLogger)
	if err != nil {
		t.Fatalf("Failed to create config service: %v", err)
	}

	// 新しい設定を作成
	newConfig := config.DefaultConfig()
	newConfig.ClaudePath = "/new/path"
	newConfig.Prometheus.RemoteWriteUsername = "testuser"
	newConfig.Prometheus.RemoteWritePassword = "testpass"

	// 設定を更新
	err = service.UpdateConfig(newConfig)
	if err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	// 更新された設定を確認
	updatedConfig := service.GetConfig()
	if updatedConfig.ClaudePath != "/new/path" {
		t.Errorf("Expected ClaudePath '/new/path', got '%s'", updatedConfig.ClaudePath)
	}

	// ファイルに保存されたことを確認
	savedConfig, err := configRepo.Load()
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}
	if savedConfig.ClaudePath != "/new/path" {
		t.Errorf("Saved config has wrong ClaudePath: %s", savedConfig.ClaudePath)
	}
}

func TestConfigServiceImpl_CreateDefaultConfig(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "tosage-config-service-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// テスト用のリポジトリを作成
	configRepo := repository.NewJSONConfigRepository()
	// テスト用にパスを上書き
	repo := configRepo.(*repository.JSONConfigRepository)
	repo.SetConfigDir(tempDir)
	repo.SetConfigFile(filepath.Join(tempDir, "config.json"))

	// ConfigService を作成
	mockLogger := &MockLogger{}
	migrationService := NewConfigMigrationService(mockLogger)
	service, err := NewConfigService(configRepo, migrationService, mockLogger)
	if err != nil {
		t.Fatalf("Failed to create config service: %v", err)
	}

	// デフォルト設定を作成（設定ファイルがまだ存在しないので成功するはず）
	err = service.CreateDefaultConfig()
	if err != nil {
		t.Errorf("CreateDefaultConfig failed: %v", err)
	}

	// 2回目の呼び出しは失敗するはず（既に存在するため）
	err = service.CreateDefaultConfig()
	if err == nil {
		t.Error("CreateDefaultConfig should fail when config already exists")
	}

	// 新しいサービスを作成（設定ファイルなし）
	tempDir2, err := os.MkdirTemp("", "tosage-config-service-test2-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir2)
	}()

	configRepo2 := repository.NewJSONConfigRepository()
	repo2 := configRepo2.(*repository.JSONConfigRepository)
	repo2.SetConfigDir(tempDir2)
	repo2.SetConfigFile(filepath.Join(tempDir2, "config.json"))

	mockLogger2 := &MockLogger{}
	migrationService2 := NewConfigMigrationService(mockLogger2)
	service2, err := NewConfigService(configRepo2, migrationService2, mockLogger2)
	if err != nil {
		t.Fatalf("Failed to create config service: %v", err)
	}

	// デフォルト設定を作成
	err = service2.CreateDefaultConfig()
	if err != nil {
		t.Fatalf("Failed to create default config: %v", err)
	}

	// ファイルが作成されたことを確認
	exists, err := configRepo2.Exists()
	if err != nil {
		t.Fatalf("Failed to check config existence: %v", err)
	}
	if !exists {
		t.Error("Config file should exist after CreateDefaultConfig")
	}
}

func TestConfigServiceImpl_ExportConfig(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "tosage-config-service-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// テスト用のリポジトリを作成
	configRepo := repository.NewJSONConfigRepository()
	repo := configRepo.(*repository.JSONConfigRepository)
	repo.SetConfigDir(tempDir)
	repo.SetConfigFile(filepath.Join(tempDir, "config.json"))

	// ConfigService を作成
	mockLogger := &MockLogger{}
	migrationService := NewConfigMigrationService(mockLogger)
	service, err := NewConfigService(configRepo, migrationService, mockLogger)
	if err != nil {
		t.Fatalf("Failed to create config service: %v", err)
	}

	// パスワードを含む設定を作成
	newConfig := config.DefaultConfig()
	newConfig.Prometheus.RemoteWriteUsername = "testuser"
	newConfig.Prometheus.RemoteWritePassword = "secret-password"
	newConfig.Logging.Promtail.Password = "promtail-secret"

	// 設定を更新
	err = service.UpdateConfig(newConfig)
	if err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	// エクスポート
	exported := service.ExportConfig()

	// パスワードがマスクされていることを確認
	prometheusMap := exported["prometheus"].(map[string]interface{})
	if prometheusMap["remote_write_password"] != "****" {
		t.Error("Prometheus remote write password should be masked")
	}

	loggingMap := exported["logging"].(map[string]interface{})
	promtailMap := loggingMap["promtail"].(map[string]interface{})
	if promtailMap["password"] != "****" {
		t.Error("Promtail password should be masked")
	}

	// ソース情報が含まれていることを確認
	sources := exported["_sources"].(map[string]string)
	if sources == nil {
		t.Error("Export should include source information")
	}
}

func TestConfigServiceImpl_EnsureConfigExists(t *testing.T) {
	t.Run("create template when config does not exist", func(t *testing.T) {
		// テスト用の一時ディレクトリを作成
		tempDir, err := os.MkdirTemp("", "tosage-config-service-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer func() {
			_ = os.RemoveAll(tempDir)
		}()

		// テスト用のリポジトリを作成
		configRepo := &repository.JSONConfigRepository{}
		configRepo.SetConfigDir(tempDir)
		configRepo.SetConfigFile(filepath.Join(tempDir, "config.json"))

		// ConfigService を作成
		mockLogger := &MockLogger{}
		migrationService := NewConfigMigrationService(mockLogger)
		service, err := NewConfigService(configRepo, migrationService, mockLogger)
		if err != nil {
			t.Fatalf("Failed to create config service: %v", err)
		}

		// 設定ファイルが存在しないことを確認
		exists, _ := configRepo.Exists()
		if exists {
			t.Fatal("Config file should not exist initially")
		}

		// EnsureConfigExists を実行
		err = service.EnsureConfigExists()
		if err != nil {
			t.Fatalf("EnsureConfigExists failed: %v", err)
		}

		// 設定ファイルが作成されたことを確認
		exists, _ = configRepo.Exists()
		if !exists {
			t.Fatal("Config file should be created")
		}

		// 作成された設定ファイルが正しいことを確認
		loadedConfig, err := configRepo.Load()
		if err != nil {
			t.Fatalf("Failed to load created config: %v", err)
		}
		if loadedConfig == nil {
			t.Fatal("Loaded config should not be nil")
		}
	})

	t.Run("do nothing when config already exists", func(t *testing.T) {
		// テスト用の一時ディレクトリを作成
		tempDir, err := os.MkdirTemp("", "tosage-config-service-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer func() {
			_ = os.RemoveAll(tempDir)
		}()

		// テスト用のリポジトリを作成
		configRepo := &repository.JSONConfigRepository{}
		configRepo.SetConfigDir(tempDir)
		configRepo.SetConfigFile(filepath.Join(tempDir, "config.json"))

		// ConfigService を作成
		mockLogger := &MockLogger{}
		migrationService := NewConfigMigrationService(mockLogger)
		service, err := NewConfigService(configRepo, migrationService, mockLogger)
		if err != nil {
			t.Fatalf("Failed to create config service: %v", err)
		}

		// 既存の設定ファイルを作成
		customConfig := config.DefaultConfig()
		customConfig.Prometheus.RemoteWriteUsername = "testuser"
		customConfig.Prometheus.RemoteWritePassword = "testpass"
		err = configRepo.Save(customConfig)
		if err != nil {
			t.Fatalf("Failed to save initial config: %v", err)
		}

		// EnsureConfigExists を実行
		err = service.EnsureConfigExists()
		if err != nil {
			t.Fatalf("EnsureConfigExists failed: %v", err)
		}

		// 設定ファイルが変更されていないことを確認
		loadedConfig, _ := configRepo.Load()
		if loadedConfig.Prometheus.RemoteWriteUsername != "testuser" {
			t.Error("Existing config should not be modified")
		}
	})
}

func TestConfigServiceImpl_CreateTemplateConfig(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "tosage-config-service-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// テスト用のリポジトリを作成
	configRepo := &repository.JSONConfigRepository{}
	configRepo.SetConfigDir(tempDir)
	configRepo.SetConfigFile(filepath.Join(tempDir, "config.json"))

	// ConfigService を作成
	mockLogger := &MockLogger{}
	migrationService := NewConfigMigrationService(mockLogger)
	service, err := NewConfigService(configRepo, migrationService, mockLogger)
	if err != nil {
		t.Fatalf("Failed to create config service: %v", err)
	}

	// CreateTemplateConfig を実行
	err = service.CreateTemplateConfig()
	if err != nil {
		t.Fatalf("CreateTemplateConfig failed: %v", err)
	}

	// 設定ファイルが作成されたことを確認
	exists, _ := configRepo.Exists()
	if !exists {
		t.Fatal("Template config file should be created")
	}

	// テンプレート設定が正しいことを確認
	loadedConfig, err := configRepo.Load()
	if err != nil {
		t.Fatalf("Failed to load template config: %v", err)
	}

	// MinimalDefaultConfigの内容を確認
	// Prometheusの基本設定のみ確認
	if loadedConfig.Prometheus == nil {
		t.Error("Prometheus config should exist")
	} else {
		if loadedConfig.Prometheus.RemoteWriteURL != "" {
			t.Errorf("Expected empty RemoteWriteURL, got '%s'", loadedConfig.Prometheus.RemoteWriteURL)
		}
	}
	// Daemonは設定されない
	if loadedConfig.Daemon != nil {
		t.Error("Daemon config should not exist in minimal template")
	}
	// Loggingの基本設定を確認
	if loadedConfig.Logging == nil {
		t.Error("Logging config should exist")
	} else if loadedConfig.Logging.Promtail == nil {
		t.Error("Promtail config should exist")
	}
}

func TestConfigServiceImpl_LoadConfigWithFallback(t *testing.T) {
	t.Run("load with valid config file", func(t *testing.T) {
		// テスト用の一時ディレクトリを作成
		tempDir, err := os.MkdirTemp("", "tosage-config-service-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer func() {
			_ = os.RemoveAll(tempDir)
		}()

		// テスト用のリポジトリを作成
		configRepo := &repository.JSONConfigRepository{}
		configRepo.SetConfigDir(tempDir)
		configRepo.SetConfigFile(filepath.Join(tempDir, "config.json"))

		// ConfigService を作成
		mockLogger := &MockLogger{}
		migrationService := NewConfigMigrationService(mockLogger)
		service, err := NewConfigService(configRepo, migrationService, mockLogger)
		if err != nil {
			t.Fatalf("Failed to create config service: %v", err)
		}

		// 有効な設定ファイルを作成
		validConfig := config.DefaultConfig()
		validConfig.ClaudePath = "/custom/path"
		validConfig.Prometheus.RemoteWriteUsername = "testuser"
		validConfig.Prometheus.RemoteWritePassword = "testpass"
		err = configRepo.Save(validConfig)
		if err != nil {
			t.Fatalf("Failed to save config: %v", err)
		}

		// LoadConfigWithFallback を実行
		cfg, err := service.LoadConfigWithFallback()
		if err != nil {
			t.Fatalf("LoadConfigWithFallback failed: %v", err)
		}

		// カスタム値が読み込まれたことを確認
		if cfg.ClaudePath != "/custom/path" {
			t.Errorf("Expected custom path '/custom/path', got '%s'", cfg.ClaudePath)
		}
	})

	t.Run("fallback to defaults when config file is missing", func(t *testing.T) {
		// テスト用の一時ディレクトリを作成
		tempDir, err := os.MkdirTemp("", "tosage-config-service-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer func() {
			_ = os.RemoveAll(tempDir)
		}()

		// テスト用のリポジトリを作成
		configRepo := &repository.JSONConfigRepository{}
		configRepo.SetConfigDir(tempDir)
		configRepo.SetConfigFile(filepath.Join(tempDir, "config.json"))

		// ConfigService を作成
		mockLogger := &MockLogger{}
		migrationService := NewConfigMigrationService(mockLogger)
		service, err := NewConfigService(configRepo, migrationService, mockLogger)
		if err != nil {
			t.Fatalf("Failed to create config service: %v", err)
		}

		// 設定ファイルが存在しない状態で LoadConfigWithFallback を実行
		cfg, err := service.LoadConfigWithFallback()
		if err != nil {
			t.Fatalf("LoadConfigWithFallback should not fail: %v", err)
		}

		// デフォルト値が使用されることを確認
		if cfg.Prometheus.IntervalSec != 600 {
			t.Errorf("Expected default interval 600, got %d", cfg.Prometheus.IntervalSec)
		}
	})

	t.Run("fallback to defaults when config file is malformed", func(t *testing.T) {
		// テスト用の一時ディレクトリを作成
		tempDir, err := os.MkdirTemp("", "tosage-config-service-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer func() {
			_ = os.RemoveAll(tempDir)
		}()

		// 不正なJSONファイルを作成
		configPath := filepath.Join(tempDir, "config.json")
		err = os.WriteFile(configPath, []byte("{ invalid json"), 0600)
		if err != nil {
			t.Fatalf("Failed to write invalid config: %v", err)
		}

		// テスト用のリポジトリを作成
		configRepo := &repository.JSONConfigRepository{}
		configRepo.SetConfigDir(tempDir)
		configRepo.SetConfigFile(configPath)

		// ConfigService を作成（エラーを無視）
		mockLogger := &MockLogger{}
		migrationService := NewConfigMigrationService(mockLogger)
		service, _ := NewConfigService(configRepo, migrationService, mockLogger)

		// LoadConfigWithFallback を実行
		cfg, err := service.LoadConfigWithFallback()
		if err != nil {
			t.Fatalf("LoadConfigWithFallback should not fail with malformed JSON: %v", err)
		}

		// デフォルト値が使用されることを確認
		if cfg.Prometheus != nil && cfg.Prometheus.IntervalSec == 0 {
			t.Error("Expected Prometheus config to have default values")
		}
	})
}
