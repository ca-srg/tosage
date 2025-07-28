package di

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ca-srg/tosage/domain"
	"github.com/ca-srg/tosage/domain/repository"
	"github.com/ca-srg/tosage/infrastructure/config"
	"github.com/ca-srg/tosage/infrastructure/logging"
	infraRepo "github.com/ca-srg/tosage/infrastructure/repository"
	"github.com/ca-srg/tosage/infrastructure/service"
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
	bedrockRepo     repository.BedrockRepository
	vertexAIRepo    repository.VertexAIRepository
	csvWriterRepo   repository.CSVWriterRepository

	// Services
	timezoneService repository.TimezoneService

	// Use Cases
	ccService            usecase.CcService
	metricsService       usecase.MetricsService
	cursorService        usecase.CursorService
	bedrockService       usecase.BedrockService
	vertexAIService      usecase.VertexAIService
	statusService        usecase.StatusService
	restartManager       usecase.RestartManager
	metricsDataCollector usecase.MetricsDataCollector
	csvExportService     usecase.CSVExportService

	// Presenters
	consolePresenter presenter.ConsolePresenter
	jsonPresenter    presenter.JSONPresenter

	// Controllers
	cliController interface{}

	// Platform-specific container
	darwinContainer *DarwinContainer

	// Logging
	loggerFactory domain.LoggerFactory
	logger        domain.Logger

	// Options
	debugMode       bool
	bedrockEnabled  bool
	vertexAIEnabled bool
}

// ContainerOption is a function that configures the container
type ContainerOption func(*Container)

// WithDebugMode sets the debug mode
func WithDebugMode(debug bool) ContainerOption {
	return func(c *Container) {
		c.debugMode = debug
	}
}

// WithBedrockEnabled sets the Bedrock enabled mode
func WithBedrockEnabled(enabled bool) ContainerOption {
	return func(c *Container) {
		c.bedrockEnabled = enabled
	}
}

// WithVertexAIEnabled sets the Vertex AI enabled mode
func WithVertexAIEnabled(enabled bool) ContainerOption {
	return func(c *Container) {
		c.vertexAIEnabled = enabled
	}
}

// NewContainer creates a new DI container
func NewContainer(opts ...ContainerOption) (*Container, error) {
	container := &Container{}

	// Apply options
	for _, opt := range opts {
		opt(container)
	}

	// Debug: Log container state
	if container.bedrockEnabled {
		fmt.Fprintf(os.Stderr, "Debug: Bedrock is enabled via command line flag\n")
	}
	if container.vertexAIEnabled {
		fmt.Fprintf(os.Stderr, "Debug: Vertex AI is enabled via command line flag\n")
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

	// Initialize Daemon components if enabled (platform-specific)
	if err := container.initDaemonPlatform(); err != nil {
		return nil, fmt.Errorf("failed to initialize daemon: %w", err)
	}

	return container, nil
}

// initConfig initializes configuration
func (c *Container) initConfig() error {
	// Create config repository if not already set
	if c.configRepo == nil {
		c.configRepo = infraRepo.NewJSONConfigRepository()
	}

	// Create temporary NoOpLogger for initial configuration loading
	tempLogger := &logging.NoOpLogger{}

	// Create migration service
	migrationService := impl.NewConfigMigrationService(tempLogger)

	// Create config service with temporary logger and migration service
	configService, err := impl.NewConfigService(c.configRepo, migrationService, tempLogger)
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

	// Override Bedrock enabled state if set via command line
	if c.bedrockEnabled {
		fmt.Fprintf(os.Stderr, "Debug: Setting up Bedrock configuration\n")
		if cfg.Bedrock == nil {
			cfg.Bedrock = &config.BedrockConfig{
				Enabled:               true,
				Regions:               []string{"us-east-1", "us-west-2"},
				AWSProfile:            "",
				AssumeRoleARN:         "",
				CollectionIntervalSec: 900,
			}
			fmt.Fprintf(os.Stderr, "Debug: Created new Bedrock config\n")
		} else {
			cfg.Bedrock.Enabled = true
			fmt.Fprintf(os.Stderr, "Debug: Updated existing Bedrock config\n")
		}
	}

	// Override Vertex AI enabled state if set via command line
	if c.vertexAIEnabled {
		if cfg.VertexAI == nil {
			cfg.VertexAI = &config.VertexAIConfig{
				Enabled:               true,
				ProjectID:             os.Getenv("GOOGLE_CLOUD_PROJECT"), // Try to get from environment
				Locations:             []string{"us-central1", "us-east1", "asia-northeast1"},
				ServiceAccountKeyPath: "",
				CollectionIntervalSec: 900,
			}
		} else {
			cfg.VertexAI.Enabled = true
			// If ProjectID is still empty, try to get from environment
			if cfg.VertexAI.ProjectID == "" {
				cfg.VertexAI.ProjectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
			}
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
	// Debug: Log repository initialization
	if c.debugMode {
		fmt.Fprintf(os.Stderr, "Debug: Starting repository initialization\n")
		fmt.Fprintf(os.Stderr, "Debug: bedrockEnabled=%v, vertexAIEnabled=%v\n", c.bedrockEnabled, c.vertexAIEnabled)
		if c.config.Bedrock != nil {
			fmt.Fprintf(os.Stderr, "Debug: Bedrock config exists, enabled=%v\n", c.config.Bedrock.Enabled)
		}
	}
	// Initialize usage repository only if Bedrock and Vertex AI are not enabled
	if !c.bedrockEnabled && !c.vertexAIEnabled {
		c.ccRepo = infraRepo.NewJSONLCcRepository(c.config.ClaudePath)
	}

	// Initialize Cursor repositories only if Bedrock and Vertex AI are not enabled and if Cursor config exists
	if !c.bedrockEnabled && !c.vertexAIEnabled {
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
	}

	// Initialize Bedrock repository if enabled
	if c.config.Bedrock != nil && c.config.Bedrock.Enabled {
		fmt.Fprintf(os.Stderr, "Debug: Attempting to initialize Bedrock repository\n")
		bedrockRepo, err := infraRepo.NewBedrockCloudWatchRepository(c.config.Bedrock.AWSProfile)
		if err != nil {
			// Log warning but don't fail initialization
			c.logger.Warn(context.TODO(), "Failed to initialize Bedrock repository", domain.NewField("error", err.Error()))
			// Also output to stderr for immediate visibility
			fmt.Fprintf(os.Stderr, "Warning: Failed to initialize Bedrock repository: %v\n", err)
			fmt.Fprintf(os.Stderr, "Please check your AWS credentials configuration.\n")

			// In debug mode, provide more detailed information
			if c.debugMode {
				c.logger.Debug(context.TODO(), "Bedrock initialization details",
					domain.NewField("aws_profile", c.config.Bedrock.AWSProfile),
					domain.NewField("regions", c.config.Bedrock.Regions),
					domain.NewField("error_type", fmt.Sprintf("%T", err)),
					domain.NewField("error_details", err.Error()))
				fmt.Fprintf(os.Stderr, "Debug: AWS Profile: %s\n", c.config.Bedrock.AWSProfile)
				fmt.Fprintf(os.Stderr, "Debug: Regions: %v\n", c.config.Bedrock.Regions)
			}
		} else {
			c.bedrockRepo = bedrockRepo
			fmt.Fprintf(os.Stderr, "Debug: Bedrock repository initialized successfully\n")
		}
	} else {
		if c.debugMode {
			fmt.Fprintf(os.Stderr, "Debug: Bedrock repository not initialized (config=%v, enabled=%v)\n",
				c.config.Bedrock != nil, c.config.Bedrock != nil && c.config.Bedrock.Enabled)
		}
	}

	// Initialize Vertex AI repository if enabled
	if c.config.VertexAI != nil && c.config.VertexAI.Enabled {
		if c.config.VertexAI.ProjectID == "" {
			c.logger.Warn(context.TODO(), "Vertex AI is enabled but project ID is not set",
				domain.NewField("hint", "Set TOSAGE_VERTEX_AI_PROJECT_ID or GOOGLE_CLOUD_PROJECT environment variable"))
			// Also output to stderr for immediate visibility
			fmt.Fprintf(os.Stderr, "Warning: Vertex AI is enabled but project ID is not set\n")
			fmt.Fprintf(os.Stderr, "Please set GOOGLE_CLOUD_PROJECT environment variable.\n")
		} else {
			// Use REST repository with retry logic
			vertexAIRepo, err := infraRepo.NewVertexAIRESTRepository(c.config.VertexAI.ProjectID, c.config.VertexAI.ServiceAccountKeyPath)
			if err != nil {
				// Log warning but don't fail initialization
				c.logger.Warn(context.TODO(), "Failed to initialize Vertex AI repository", domain.NewField("error", err.Error()))
				// Also output to stderr for immediate visibility
				fmt.Fprintf(os.Stderr, "Warning: Failed to initialize Vertex AI repository: %v\n", err)
				fmt.Fprintf(os.Stderr, "Please check your Google Cloud credentials configuration.\n")

				// In debug mode, provide more detailed information
				if c.debugMode {
					c.logger.Debug(context.TODO(), "Vertex AI initialization details",
						domain.NewField("project_id", c.config.VertexAI.ProjectID),
						domain.NewField("service_account_key_path", c.config.VertexAI.ServiceAccountKeyPath),
						domain.NewField("locations", c.config.VertexAI.Locations),
						domain.NewField("error_type", fmt.Sprintf("%T", err)),
						domain.NewField("error_details", err.Error()))
					fmt.Fprintf(os.Stderr, "Debug: Project ID: %s\n", c.config.VertexAI.ProjectID)
					fmt.Fprintf(os.Stderr, "Debug: Service Account Key Path: %s\n", c.config.VertexAI.ServiceAccountKeyPath)
					fmt.Fprintf(os.Stderr, "Debug: Locations: %v\n", c.config.VertexAI.Locations)
				}
			} else {
				c.vertexAIRepo = vertexAIRepo
				c.logger.Info(context.TODO(), "Vertex AI REST repository initialized with retry logic",
					domain.NewField("project_id", c.config.VertexAI.ProjectID),
					domain.NewField("locations", c.config.VertexAI.Locations))
			}
		}
	}

	// Initialize CSV writer repository
	c.csvWriterRepo = infraRepo.NewCSVWriterRepository(c.CreateLogger("csv-writer"))

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
	// Initialize CC service only if Bedrock and Vertex AI are not enabled
	if !c.bedrockEnabled && !c.vertexAIEnabled {
		c.ccService = impl.NewCcServiceImpl(c.ccRepo, c.timezoneService)
	}

	// Initialize Status service
	c.statusService = impl.NewStatusService()

	// Initialize Cursor service only if Bedrock and Vertex AI are not enabled and if configured
	if !c.bedrockEnabled && !c.vertexAIEnabled && c.config.Cursor != nil && c.cursorTokenRepo != nil && c.cursorAPIRepo != nil {
		c.cursorService = impl.NewCursorService(c.cursorTokenRepo, c.cursorAPIRepo, c.config.Cursor)
	}

	// Initialize Bedrock service if configured
	if c.config.Bedrock != nil && c.bedrockRepo != nil {
		bedrockConfig := &repository.BedrockConfig{
			Enabled:            c.config.Bedrock.Enabled,
			Regions:            c.config.Bedrock.Regions,
			AWSProfile:         c.config.Bedrock.AWSProfile,
			AssumeRoleARN:      c.config.Bedrock.AssumeRoleARN,
			CollectionInterval: time.Duration(c.config.Bedrock.CollectionIntervalSec) * time.Second,
		}
		c.bedrockService = impl.NewBedrockService(c.bedrockRepo, bedrockConfig, c.CreateLogger("bedrock"))
	}

	// Initialize Vertex AI service if configured
	if c.config.VertexAI != nil && c.vertexAIRepo != nil {
		vertexAIConfig := &repository.VertexAIConfig{
			Enabled:               c.config.VertexAI.Enabled,
			ProjectID:             c.config.VertexAI.ProjectID,
			Locations:             c.config.VertexAI.Locations,
			ServiceAccountKeyPath: c.config.VertexAI.ServiceAccountKeyPath,
			CollectionInterval:    time.Duration(c.config.VertexAI.CollectionIntervalSec) * time.Second,
		}
		c.vertexAIService = impl.NewVertexAIService(c.vertexAIRepo, vertexAIConfig)
	}

	// Initialize Restart manager
	restartManager, err := impl.NewRestartManager()
	if err != nil {
		return fmt.Errorf("failed to create restart manager: %w", err)
	}
	c.restartManager = restartManager

	// Initialize Metrics Data Collector
	c.metricsDataCollector = impl.NewMetricsDataCollector(
		c.ccService,
		c.cursorService,
		c.bedrockService,
		c.vertexAIService,
		c.CreateLogger("metrics-collector"),
	)

	// Initialize CSV Export Service
	c.csvExportService = impl.NewCSVExportService(
		c.metricsDataCollector,
		c.csvWriterRepo,
		c.CreateLogger("csv-export"),
	)

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
	// Only create CLI controller if we have ccService or if it's explicitly needed
	// When Bedrock or Vertex AI is enabled, ccService will be nil
	c.cliController = newCLIController(
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
		c.bedrockService,
		c.vertexAIService,
		c.metricsRepo,
		c.config.Prometheus,
		c.CreateLogger("metrics"),
		c.timezoneService,
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
func (c *Container) GetCLIController() interface{} {
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

// GetBedrockService returns the Bedrock service
func (c *Container) GetBedrockService() usecase.BedrockService {
	return c.bedrockService
}

// GetBedrockRepository returns the Bedrock repository
func (c *Container) GetBedrockRepository() repository.BedrockRepository {
	return c.bedrockRepo
}

// GetVertexAIService returns the Vertex AI service
func (c *Container) GetVertexAIService() usecase.VertexAIService {
	return c.vertexAIService
}

// GetVertexAIRepository returns the Vertex AI repository
func (c *Container) GetVertexAIRepository() repository.VertexAIRepository {
	return c.vertexAIRepo
}

// GetStatusService returns the status service
func (c *Container) GetStatusService() usecase.StatusService {
	return c.statusService
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

// GetCSVExportService returns the CSV export service
func (c *Container) GetCSVExportService() usecase.CSVExportService {
	return c.csvExportService
}

// InitDaemonComponents initializes daemon components on demand
func (c *Container) InitDaemonComponents() error {
	return c.initDaemonPlatform()
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
		migrationService := impl.NewConfigMigrationService(tempLogger)
		configService, err := impl.NewConfigService(container.configRepo, migrationService, tempLogger)
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
		container.bedrockService,
		container.vertexAIService,
		container.metricsRepo,
		container.config.Prometheus,
		container.CreateLogger("metrics"),
		container.timezoneService,
	)

	// Initialize daemon components if configured (platform-specific)
	if err := container.initDaemonPlatform(); err != nil {
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
