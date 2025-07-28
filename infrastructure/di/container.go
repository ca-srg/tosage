package di

import (
	"fmt"
	"os"
	"time"

	"github.com/ca-srg/tosage/domain"
	"github.com/ca-srg/tosage/domain/repository"
	"github.com/ca-srg/tosage/infrastructure/config"
	"github.com/ca-srg/tosage/infrastructure/logging"
	infraRepo "github.com/ca-srg/tosage/infrastructure/repository"
	"github.com/ca-srg/tosage/infrastructure/service"
	"github.com/ca-srg/tosage/interface/controller"
	"github.com/ca-srg/tosage/interface/presenter"
	"github.com/ca-srg/tosage/usecase/impl"
	usecase "github.com/ca-srg/tosage/usecase/interface"
)

// Container is the dependency injection container
type Container struct {
	// Configuration
	config        *config.AppConfig
	configRepo    repository.ConfigRepository
	configService usecase.ConfigService

	// Repositories
	ccRepo          repository.CcRepository
	metricsRepo     repository.MetricsRepository
	cursorTokenRepo repository.CursorTokenRepository
	cursorAPIRepo   repository.CursorAPIRepository

	// Services
	timezoneService repository.TimezoneService

	// Use Cases
	ccService      usecase.CcService
	metricsService usecase.MetricsService
	cursorService  usecase.CursorService
	statusService  usecase.StatusService
	restartManager usecase.RestartManager

	// Presenters
	consolePresenter presenter.ConsolePresenter
	jsonPresenter    presenter.JSONPresenter

	// Controllers
	cliController     *controller.CLIController
	systrayController *controller.SystrayController
	daemonController  *controller.DaemonController

	// Logging
	loggerFactory domain.LoggerFactory
	logger        domain.Logger

	// Options
	debugMode bool
}

// ContainerOption is a function that configures the container
type ContainerOption func(*Container)

// WithDebugMode sets the debug mode
func WithDebugMode(debug bool) ContainerOption {
	return func(c *Container) {
		c.debugMode = debug
	}
}

// NewContainer creates a new DI container
func NewContainer(opts ...ContainerOption) (*Container, error) {
	container := &Container{}

	// Apply options
	for _, opt := range opts {
		opt(container)
	}

	// Load configuration
	if err := container.initConfig(); err != nil {
		return nil, fmt.Errorf("failed to initialize config: %w", err)
	}

	// Initialize logging
	if err := container.initLogging(); err != nil {
		return nil, fmt.Errorf("failed to initialize logging: %w", err)
	}

	// Initialize repositories
	if err := container.initRepositories(); err != nil {
		return nil, fmt.Errorf("failed to initialize repositories: %w", err)
	}

	// Initialize domain services
	if err := container.initDomainServices(); err != nil {
		return nil, fmt.Errorf("failed to initialize domain services: %w", err)
	}

	// Initialize use cases
	if err := container.initUseCases(); err != nil {
		return nil, fmt.Errorf("failed to initialize use cases: %w", err)
	}

	// Initialize presenters
	if err := container.initPresenters(); err != nil {
		return nil, fmt.Errorf("failed to initialize presenters: %w", err)
	}

	// Initialize controllers
	if err := container.initControllers(); err != nil {
		return nil, fmt.Errorf("failed to initialize controllers: %w", err)
	}

	// Initialize Prometheus components if enabled
	if err := container.initPrometheus(); err != nil {
		return nil, fmt.Errorf("failed to initialize prometheus: %w", err)
	}

	// Initialize Daemon components if enabled
	if err := container.initDaemon(); err != nil {
		return nil, fmt.Errorf("failed to initialize daemon: %w", err)
	}

	return container, nil
}

// initConfig initializes configuration
func (c *Container) initConfig() error {
	// Create config repository
	c.configRepo = infraRepo.NewJSONConfigRepository()

	// Create temporary NoOpLogger for initial configuration loading
	tempLogger := &logging.NoOpLogger{}

	// Create config service with temporary logger
	configService, err := impl.NewConfigService(c.configRepo, tempLogger)
	if err != nil {
		// エラー耐性: 設定サービスの作成に失敗してもデフォルト設定で継続
		c.config = config.DefaultConfig()
		// ConfigServiceがないとシステムが動作しないので、エラーを返す
		return fmt.Errorf("failed to create config service: %w", err)
	}
	c.configService = configService

	// Ensure config file exists (create template if needed)
	if err := configService.EnsureConfigExists(); err != nil {
		// エラーメッセージを標準エラー出力に表示
		fmt.Fprintf(os.Stderr, "Warning: Failed to create config file: %v\n", err)
		// デフォルト設定で継続
	}

	// Get configuration from service (with fallback to defaults)
	cfg := configService.GetConfig()

	// Override debug mode if set via command line
	if c.debugMode {
		if cfg.Logging == nil {
			cfg.Logging = &config.LoggingConfig{
				Level: "info",
				Debug: true,
				Promtail: &config.PromtailConfig{
					URL:              "http://localhost:3100/loki/api/v1/push",
					BatchWaitSeconds: 1,
					BatchCapacity:    100,
					TimeoutSeconds:   5,
				},
			}
		} else {
			cfg.Logging.Debug = true
		}
	}

	// Ensure Daemon config exists if daemon mode is enabled via environment
	if os.Getenv("TOSAGE_DAEMON_ENABLED") == "true" && cfg.Daemon == nil {
		cfg.Daemon = &config.DaemonConfig{
			Enabled:      true,
			StartAtLogin: false,
			LogPath:      "/tmp/tosage.log",
			PidFile:      "/tmp/tosage.pid",
		}
	}

	c.config = cfg
	return nil
}

// initLogging initializes logging components
func (c *Container) initLogging() error {
	// Ensure logging configuration exists
	if c.config.Logging == nil {
		c.config.Logging = &config.LoggingConfig{
			Level: "info",
			Debug: false,
			Promtail: &config.PromtailConfig{
				URL:              "http://localhost:3100/loki/api/v1/push",
				BatchWaitSeconds: 1,
				BatchCapacity:    100,
				TimeoutSeconds:   5,
			},
		}
	}

	// Create logger factory
	c.loggerFactory = logging.NewLoggerFactory(c.config.Logging)

	// Create main logger for the container
	c.logger = c.loggerFactory.CreateLogger("tosage")

	return nil
}

// initRepositories initializes repository implementations
func (c *Container) initRepositories() error {
	// Initialize usage repository
	c.ccRepo = infraRepo.NewJSONLCcRepository(c.config.ClaudePath)

	// Initialize Cursor repositories only if Cursor config exists
	if c.config.Cursor != nil {
		c.cursorTokenRepo = infraRepo.NewCursorDBRepository(c.config.Cursor.DatabasePath)
		c.cursorAPIRepo = infraRepo.NewCursorAPIRepository(time.Duration(c.config.Cursor.APITimeout) * time.Second)
	} else {
		// Create default Cursor config if not exists
		c.config.Cursor = &config.CursorConfig{
			DatabasePath: "",
			APITimeout:   30,
			CacheTimeout: 300,
		}
		c.cursorTokenRepo = infraRepo.NewCursorDBRepository(c.config.Cursor.DatabasePath)
		c.cursorAPIRepo = infraRepo.NewCursorAPIRepository(time.Duration(c.config.Cursor.APITimeout) * time.Second)
	}

	return nil
}

// initDomainServices initializes domain services
func (c *Container) initDomainServices() error {
	// Initialize timezone service
	c.timezoneService = service.NewTimezoneServiceImpl(c.config, c.logger)
	return nil
}

// initUseCases initializes use case implementations
func (c *Container) initUseCases() error {
	c.ccService = impl.NewCcServiceImpl(c.ccRepo, c.timezoneService)

	// Initialize Status service
	c.statusService = impl.NewStatusService()

	// Initialize Cursor service if configured
	if c.config.Cursor != nil && c.cursorTokenRepo != nil && c.cursorAPIRepo != nil {
		c.cursorService = impl.NewCursorService(c.cursorTokenRepo, c.cursorAPIRepo, c.config.Cursor)
	}

	// Initialize Restart manager
	restartManager, err := impl.NewRestartManager()
	if err != nil {
		return fmt.Errorf("failed to create restart manager: %w", err)
	}
	c.restartManager = restartManager

	return nil
}

// initPresenters initializes presenter implementations
func (c *Container) initPresenters() error {
	c.consolePresenter = presenter.NewConsolePresenter()
	c.jsonPresenter = presenter.NewJSONPresenter()
	return nil
}

// initControllers initializes controller implementations
func (c *Container) initControllers() error {
	c.cliController = controller.NewCLIController(
		c.ccService,
		c.consolePresenter,
		c.jsonPresenter,
	)
	return nil
}

// initPrometheus initializes Prometheus components
func (c *Container) initPrometheus() error {
	// Ensure Prometheus config exists with defaults
	if c.config.Prometheus == nil {
		c.config.Prometheus = &config.PrometheusConfig{
			RemoteWriteURL: "http://localhost:9090/api/v1/write",
			HostLabel:      "",
			IntervalSec:    600,
			TimeoutSec:     30,
		}
	}

	// Initialize metrics repository
	metricsRepo, err := infraRepo.NewPrometheusMetricsRepository(c.config.Prometheus)
	if err != nil {
		return fmt.Errorf("failed to create metrics repository: %w", err)
	}
	c.metricsRepo = metricsRepo

	// Initialize metrics service
	c.metricsService = impl.NewMetricsServiceImpl(
		c.ccService,
		c.cursorService,
		c.metricsRepo,
		c.config.Prometheus,
		c.CreateLogger("metrics"),
		c.timezoneService,
	)

	return nil
}

// initDaemon initializes daemon components
func (c *Container) initDaemon() error {
	// Only initialize if daemon mode is configured
	if c.config.Daemon == nil || !c.config.Daemon.Enabled {
		return nil
	}

	// Initialize systray controller
	c.systrayController = controller.NewSystrayController(
		c.ccService,
		c.statusService,
		c.metricsService,
		c.configService,
	)

	// Initialize daemon controller
	c.daemonController = controller.NewDaemonController(
		c.config,
		c.configService,
		c.ccService,
		c.statusService,
		c.metricsService,
		c.systrayController,
		c.CreateLogger("daemon"),
	)

	return nil
}

// GetConfig returns the application configuration
func (c *Container) GetConfig() *config.AppConfig {
	return c.config
}

// GetCcRepository returns the usage repository
func (c *Container) GetCcRepository() repository.CcRepository {
	return c.ccRepo
}

// GetCcService returns the usage service
func (c *Container) GetCcService() usecase.CcService {
	return c.ccService
}

// GetConsolePresenter returns the console presenter
func (c *Container) GetConsolePresenter() presenter.ConsolePresenter {
	return c.consolePresenter
}

// GetJSONPresenter returns the JSON presenter
func (c *Container) GetJSONPresenter() presenter.JSONPresenter {
	return c.jsonPresenter
}

// GetCLIController returns the CLI controller
func (c *Container) GetCLIController() *controller.CLIController {
	return c.cliController
}

// GetMetricsRepository returns the metrics repository
func (c *Container) GetMetricsRepository() repository.MetricsRepository {
	return c.metricsRepo
}

// GetMetricsService returns the metrics service
func (c *Container) GetMetricsService() usecase.MetricsService {
	return c.metricsService
}

// GetCursorTokenRepository returns the Cursor token repository
func (c *Container) GetCursorTokenRepository() repository.CursorTokenRepository {
	return c.cursorTokenRepo
}

// GetCursorAPIRepository returns the Cursor API repository
func (c *Container) GetCursorAPIRepository() repository.CursorAPIRepository {
	return c.cursorAPIRepo
}

// GetCursorService returns the Cursor service
func (c *Container) GetCursorService() usecase.CursorService {
	return c.cursorService
}

// GetStatusService returns the status service
func (c *Container) GetStatusService() usecase.StatusService {
	return c.statusService
}

// GetSystrayController returns the systray controller
func (c *Container) GetSystrayController() *controller.SystrayController {
	return c.systrayController
}

// GetDaemonController returns the daemon controller
func (c *Container) GetDaemonController() *controller.DaemonController {
	return c.daemonController
}

// GetLoggerFactory returns the logger factory
func (c *Container) GetLoggerFactory() domain.LoggerFactory {
	return c.loggerFactory
}

// GetLogger returns the main logger
func (c *Container) GetLogger() domain.Logger {
	return c.logger
}

// CreateLogger creates a new logger for a specific component
func (c *Container) CreateLogger(component string) domain.Logger {
	if c.loggerFactory == nil {
		return &logging.NoOpLogger{}
	}
	return c.loggerFactory.CreateLogger(component)
}

// GetConfigRepository returns the config repository
func (c *Container) GetConfigRepository() repository.ConfigRepository {
	return c.configRepo
}

// GetConfigService returns the config service
func (c *Container) GetConfigService() usecase.ConfigService {
	return c.configService
}

// GetRestartManager returns the restart manager
func (c *Container) GetRestartManager() usecase.RestartManager {
	return c.restartManager
}

// GetTimezoneService returns the timezone service
func (c *Container) GetTimezoneService() repository.TimezoneService {
	return c.timezoneService
}

// InitDaemonComponents initializes daemon components on demand
func (c *Container) InitDaemonComponents() error {
	return c.initDaemon()
}

// Builder pattern for custom container configuration

// ContainerBuilder builds a custom container
type ContainerBuilder struct {
	config          *config.AppConfig
	configRepo      repository.ConfigRepository
	ccRepo          repository.CcRepository
	metricsRepo     repository.MetricsRepository
	cursorTokenRepo repository.CursorTokenRepository
	cursorAPIRepo   repository.CursorAPIRepository
	useCustom       bool
}

// NewContainerBuilder creates a new container builder
func NewContainerBuilder() *ContainerBuilder {
	return &ContainerBuilder{}
}

// WithConfig sets a custom configuration
func (b *ContainerBuilder) WithConfig(cfg *config.AppConfig) *ContainerBuilder {
	b.config = cfg
	b.useCustom = true
	return b
}

// WithConfigRepository sets a custom config repository
func (b *ContainerBuilder) WithConfigRepository(repo repository.ConfigRepository) *ContainerBuilder {
	b.configRepo = repo
	b.useCustom = true
	return b
}

// WithCcRepository sets a custom usage repository
func (b *ContainerBuilder) WithCcRepository(repo repository.CcRepository) *ContainerBuilder {
	b.ccRepo = repo
	b.useCustom = true
	return b
}

// WithMetricsRepository sets a custom metrics repository
func (b *ContainerBuilder) WithMetricsRepository(repo repository.MetricsRepository) *ContainerBuilder {
	b.metricsRepo = repo
	b.useCustom = true
	return b
}

// WithCursorTokenRepository sets a custom Cursor token repository
func (b *ContainerBuilder) WithCursorTokenRepository(repo repository.CursorTokenRepository) *ContainerBuilder {
	b.cursorTokenRepo = repo
	b.useCustom = true
	return b
}

// WithCursorAPIRepository sets a custom Cursor API repository
func (b *ContainerBuilder) WithCursorAPIRepository(repo repository.CursorAPIRepository) *ContainerBuilder {
	b.cursorAPIRepo = repo
	b.useCustom = true
	return b
}

// Build builds the container with custom components
func (b *ContainerBuilder) Build() (*Container, error) {
	container := &Container{}

	// Use custom config repository or create default
	if b.configRepo != nil {
		container.configRepo = b.configRepo
	} else {
		container.configRepo = infraRepo.NewJSONConfigRepository()
	}

	// Use custom config or load default
	if b.config != nil {
		container.config = b.config
		// Create config service with custom config using temporary logger
		tempLogger := &logging.NoOpLogger{}
		configService, err := impl.NewConfigService(container.configRepo, tempLogger)
		if err != nil {
			return nil, fmt.Errorf("failed to create config service: %w", err)
		}
		container.configService = configService
	} else {
		if err := container.initConfig(); err != nil {
			return nil, fmt.Errorf("failed to initialize config: %w", err)
		}
	}

	// Use custom repositories or create default
	if b.ccRepo != nil {
		container.ccRepo = b.ccRepo
	} else {
		container.ccRepo = infraRepo.NewJSONLCcRepository(container.config.ClaudePath)
	}

	if b.metricsRepo != nil {
		container.metricsRepo = b.metricsRepo
	} else {
		// Initialize metrics repository
		metricsRepo, err := infraRepo.NewPrometheusMetricsRepository(container.config.Prometheus)
		if err != nil {
			return nil, fmt.Errorf("failed to create metrics repository: %w", err)
		}
		container.metricsRepo = metricsRepo
	}

	// Use custom Cursor repositories or create default
	if b.cursorTokenRepo != nil {
		container.cursorTokenRepo = b.cursorTokenRepo
	} else if container.config.Cursor != nil {
		container.cursorTokenRepo = infraRepo.NewCursorDBRepository(container.config.Cursor.DatabasePath)
	}

	if b.cursorAPIRepo != nil {
		container.cursorAPIRepo = b.cursorAPIRepo
	} else if container.config.Cursor != nil {
		container.cursorAPIRepo = infraRepo.NewCursorAPIRepository(time.Duration(container.config.Cursor.APITimeout) * time.Second)
	}

	// Initialize remaining components
	if err := container.initDomainServices(); err != nil {
		return nil, fmt.Errorf("failed to initialize domain services: %w", err)
	}

	if err := container.initUseCases(); err != nil {
		return nil, fmt.Errorf("failed to initialize use cases: %w", err)
	}

	if err := container.initPresenters(); err != nil {
		return nil, fmt.Errorf("failed to initialize presenters: %w", err)
	}
	if err := container.initControllers(); err != nil {
		return nil, fmt.Errorf("failed to initialize controllers: %w", err)
	}

	// Initialize metrics service
	container.metricsService = impl.NewMetricsServiceImpl(
		container.ccService,
		container.cursorService,
		container.metricsRepo,
		container.config.Prometheus,
		container.CreateLogger("metrics"),
		container.timezoneService,
	)

	// Initialize daemon components if configured
	if err := container.initDaemon(); err != nil {
		return nil, fmt.Errorf("failed to initialize daemon: %w", err)
	}

	return container, nil
}

// ServiceLocator provides a global access point to services (use with caution)
var defaultContainer *Container

// InitializeDefault initializes the default container
func InitializeDefault() error {
	container, err := NewContainer()
	if err != nil {
		return err
	}
	defaultContainer = container
	return nil
}

// GetDefaultContainer returns the default container
func GetDefaultContainer() (*Container, error) {
	if defaultContainer == nil {
		if err := InitializeDefault(); err != nil {
			return nil, err
		}
	}
	return defaultContainer, nil
}
