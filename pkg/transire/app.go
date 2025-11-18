package transire

import (
	"context"
	"fmt"

	"github.com/go-chi/chi/v5"
)

// App is the main application abstraction
type App struct {
	router        *chi.Mux
	queueHandlers []QueueHandler
	schedHandlers []SchedulerHandler
	config        *Config
	provider      Provider
	runtime       Runtime
}

// New creates a new Transire application
func New(opts ...Option) *App {
	app := &App{
		router: chi.NewRouter(),
		config: defaultConfig(),
	}

	for _, opt := range opts {
		opt(app)
	}

	return app
}

// Router returns the Chi router for HTTP route registration
func (a *App) Router() *chi.Mux {
	return a.router
}

// RegisterQueueHandler adds a queue handler
func (a *App) RegisterQueueHandler(handler QueueHandler) {
	a.queueHandlers = append(a.queueHandlers, handler)
}

// RegisterScheduleHandler adds a scheduled handler
func (a *App) RegisterScheduleHandler(handler SchedulerHandler) {
	a.schedHandlers = append(a.schedHandlers, handler)
}

// FindQueueHandler returns the handler for the specified queue name
func (a *App) FindQueueHandler(queueName string) QueueHandler {
	for _, handler := range a.queueHandlers {
		if handler.QueueName() == queueName {
			return handler
		}
	}
	return nil
}

// FindScheduleHandler returns the handler for the specified schedule name
func (a *App) FindScheduleHandler(scheduleName string) SchedulerHandler {
	for _, handler := range a.schedHandlers {
		if handler.Name() == scheduleName {
			return handler
		}
	}
	return nil
}

// GetQueueHandlers returns all registered queue handlers
func (a *App) GetQueueHandlers() []QueueHandler {
	return a.queueHandlers
}

// GetScheduleHandlers returns all registered schedule handlers
func (a *App) GetScheduleHandlers() []SchedulerHandler {
	return a.schedHandlers
}

// GetConfig returns the app configuration
func (a *App) GetConfig() *Config {
	return a.config
}

// SetProvider sets the cloud provider
func (a *App) SetProvider(provider Provider) {
	a.provider = provider
}

// GetProvider returns the current provider
func (a *App) GetProvider() Provider {
	return a.provider
}

// Run starts the application (local or cloud)
func (a *App) Run(ctx context.Context) error {
	// Ensure we have a runtime
	if a.runtime == nil {
		runtime, err := a.createRuntime(ctx)
		if err != nil {
			return fmt.Errorf("failed to create runtime: %w", err)
		}
		a.runtime = runtime
	}

	// Start the runtime
	return a.runtime.Start(ctx, a)
}

// Stop gracefully stops the application
func (a *App) Stop(ctx context.Context) error {
	if a.runtime != nil {
		return a.runtime.Stop(ctx)
	}
	return nil
}

// createRuntime creates the appropriate runtime based on environment
func (a *App) createRuntime(ctx context.Context) (Runtime, error) {
	// Detect the runtime environment
	runtimeType := detectRuntime()

	// If we have a provider, use it to create the runtime
	if a.provider != nil {
		runtimeConfig := RuntimeConfig{
			Environment: string(runtimeType),
			Provider:    a.provider.Name(),
			Runtime:     a.provider.Runtime(),
		}

		return a.provider.CreateRuntime(ctx, runtimeConfig)
	}

	// Fallback to default runtime detection/creation
	return createDefaultRuntime(runtimeType, a.config)
}

// defaultConfig creates a default configuration
func defaultConfig() *Config {
	config := &Config{}
	config.setDefaults()
	return config
}

// createDefaultRuntime creates a runtime without a provider (for local development)
func createDefaultRuntime(runtimeType RuntimeType, config *Config) (Runtime, error) {
	switch runtimeType {
	case RuntimeLocal:
		return newLocalRuntime(config), nil
	case RuntimeAWSLambda:
		return newLambdaRuntime(), nil
	default:
		return nil, fmt.Errorf("unsupported runtime: %v", runtimeType)
	}
}

// Global app instance for convenience (optional)
var globalApp *App

// SetGlobalApp sets the global app instance
func SetGlobalApp(app *App) {
	globalApp = app
}

// GetGlobalApp returns the global app instance
func GetGlobalApp() *App {
	if globalApp == nil {
		globalApp = New()
	}
	return globalApp
}

// GetApp returns the global app instance (alias for GetGlobalApp)
func GetApp() *App {
	return GetGlobalApp()
}
