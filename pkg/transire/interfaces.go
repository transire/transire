// Package transire provides the core abstractions for cloud-agnostic applications
package transire

import (
	"context"
	"net/http"
	"time"
)

// HTTPHandler is a standard http.Handler with optional metadata
type HTTPHandler interface {
	http.Handler
	// Metadata returns optional handler configuration
	Metadata() HTTPHandlerMetadata
}

// HTTPHandlerMetadata provides routing and configuration information
type HTTPHandlerMetadata struct {
	Methods     []string      // HTTP methods (GET, POST, etc.)
	Path        string        // Route path
	Middlewares []Middleware  // Applied middlewares
	Timeout     time.Duration // Request timeout
}

// Middleware represents HTTP middleware
type Middleware interface {
	Handle(next http.Handler) http.Handler
}

// QueueHandler processes messages in batches
type QueueHandler interface {
	// HandleMessages processes a batch of messages
	// Returns message IDs that failed and should be retried
	HandleMessages(ctx context.Context, messages []Message) ([]string, error)

	// QueueName returns the logical queue name
	QueueName() string

	// Config returns queue configuration
	Config() QueueConfig
}

// SchedulerHandler handles scheduled/cron events
type SchedulerHandler interface {
	// HandleSchedule executes on the defined schedule
	HandleSchedule(ctx context.Context, event ScheduleEvent) error

	// Schedule returns cron expression or interval
	Schedule() string

	// Name returns unique identifier for this scheduled task
	Name() string

	// Config returns scheduler configuration
	Config() ScheduleConfig
}

// Message represents a queue message
type Message interface {
	ID() string
	Body() []byte
	Attributes() map[string]string
	DeliveryCount() int
	EnqueuedAt() time.Time
}

// ScheduleEvent provides context for scheduled execution
type ScheduleEvent struct {
	ScheduledTime time.Time
	Name          string
	Payload       []byte
	EventID       string
}

// QueueConfig configures queue behavior
type QueueConfig struct {
	VisibilityTimeoutSeconds int  `yaml:"visibility_timeout_seconds"` // How long messages are invisible after delivery
	MaxReceiveCount          int  `yaml:"max_receive_count"`          // Max delivery attempts before DLQ
	BatchSize                int  `yaml:"batch_size"`                 // Max messages per batch
	WaitTimeSeconds          int  `yaml:"wait_time_seconds"`          // Long polling wait time
	FIFO                     bool `yaml:"fifo"`                       // Whether queue is FIFO
}

// ScheduleConfig configures scheduled task behavior
type ScheduleConfig struct {
	Timezone       string        `yaml:"timezone"`        // Timezone for cron expression
	Enabled        bool          `yaml:"enabled"`         // Whether schedule is enabled
	TimeoutSeconds int           `yaml:"timeout_seconds"` // Execution timeout
	RetryAttempts  int           `yaml:"retry_attempts"`  // Number of retry attempts on failure
	RetryDelay     time.Duration `yaml:"retry_delay"`     // Delay between retries
}

// Option configures an App instance
type Option func(*App)

// WithProvider sets the cloud provider
func WithProvider(provider Provider) Option {
	return func(a *App) {
		a.provider = provider
	}
}

// WithConfig sets the application configuration
func WithConfig(config *Config) Option {
	return func(a *App) {
		a.config = config
	}
}

// Provider abstracts cloud provider implementations
type Provider interface {
	// Name returns provider identifier (aws, gcp, azure)
	Name() string

	// Runtime returns supported runtime (lambda, ecs, cloudrun)
	Runtime() string

	// BuildArtifacts creates deployable artifacts
	BuildArtifacts(ctx context.Context, config BuildConfig) error

	// GenerateIaC creates infrastructure definitions
	GenerateIaC(ctx context.Context, config IaCConfig) error

	// Deploy applies infrastructure and artifacts
	Deploy(ctx context.Context, config DeployConfig) error

	// CreateRuntime returns a runtime implementation for local/cloud execution
	CreateRuntime(ctx context.Context, config RuntimeConfig) (Runtime, error)
}

// Runtime handles the execution environment
type Runtime interface {
	// Start begins processing in the current environment
	Start(ctx context.Context, app *App) error

	// Stop gracefully shuts down the runtime
	Stop(ctx context.Context) error

	// IsLocal returns true if running in local development mode
	IsLocal() bool
}

// BuildConfig configures artifact building
type BuildConfig struct {
	AppPath       string
	OutputDir     string
	Architecture  string // arm64 for Lambda
	Environment   map[string]string
	ExcludeTags   []string // Build tags to exclude (e.g., "local")
	Optimizations bool     // Enable build optimizations
}

// IaCConfig configures infrastructure generation
type IaCConfig struct {
	StackName         string
	HTTPHandlers      []HTTPHandlerSpec
	QueueHandlers     []QueueHandlerSpec
	ScheduleHandlers  []ScheduleHandlerSpec
	FunctionGroups    map[string]FunctionGroupSpec
	Extensions        map[string]interface{} // User customizations
	ExistingResources ExistingResourcesConfig
}

// DeployConfig configures deployment
type DeployConfig struct {
	StackName   string
	Region      string
	Environment string // dev, staging, prod
	DryRun      bool   // Preview changes without applying
}

// RuntimeConfig configures runtime creation
type RuntimeConfig struct {
	Environment string
	Provider    string
	Runtime     string
	Local       LocalConfig
}

// LocalConfig configures local development runtime
type LocalConfig struct {
	HTTPPort      int
	QueuePort     int
	SchedulerPort int
	AutoReload    bool
	LogLevel      string
}

// Handler specifications for IaC generation
type HTTPHandlerSpec struct {
	Path     string
	Methods  []string
	Function string // Function group name
	Timeout  time.Duration
}

type QueueHandlerSpec struct {
	QueueName string
	Function  string // Function group name
	Config    QueueConfig
}

type ScheduleHandlerSpec struct {
	Name     string
	Schedule string
	Function string // Function group name
	Config   ScheduleConfig
}

// FunctionGroupSpec defines how to group handlers into functions
type FunctionGroupSpec struct {
	Include             IncludeSpec
	MemoryMB            int
	TimeoutSeconds      int
	ReservedConcurrency *int
	Environment         map[string]string
}

// IncludeSpec defines which handlers to include in a function group
type IncludeSpec struct {
	HTTPHandlers     interface{} // "*" or []string
	QueueHandlers    interface{} // "*" or []string
	ScheduleHandlers interface{} // "*" or []string
}
