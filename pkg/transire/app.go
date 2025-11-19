// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package transire

import (
	"context"
	"sync"

	"github.com/go-chi/chi/v5"
)

var (
	defaultRuntimeFactory func(*Config) Runtime
	runtimeMu             sync.RWMutex
)

// RegisterDefaultRuntime is called by runtime packages during init()
func RegisterDefaultRuntime(factory func(*Config) Runtime) {
	runtimeMu.Lock()
	defer runtimeMu.Unlock()

	if defaultRuntimeFactory != nil {
		panic("Runtime already registered - multiple runtime build tags detected")
	}

	defaultRuntimeFactory = factory
}

// App is the main application abstraction
type App struct {
	router        *chi.Mux
	queueHandlers []QueueHandler
	schedHandlers []SchedulerHandler
	config        *Config
	provider      Provider
	runtime       Runtime
	queueProducer QueueProducer
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

	// Runtime MUST be provided via build-time registration
	if app.runtime == nil {
		runtimeMu.RLock()
		factory := defaultRuntimeFactory
		runtimeMu.RUnlock()

		if factory == nil {
			panic(`No runtime registered.

This is a build configuration error. Ensure you:
1. Build with a runtime tag: -tags=local OR -tags=lambda
2. Use the Transire CLI which automatically adds correct build tags:
   transire-cli run          # For local development
   transire-cli build        # For Lambda deployment
   transire-cli deploy       # For deployment

Example manual builds:
  go build -tags=local -o myapp .
  go build -tags=lambda -o bootstrap .
`)
		}

		app.runtime = factory(app.config)
	}

	// Initialize queue producer at BUILD TIME (during app construction)
	if app.queueProducer == nil {
		producer, err := app.runtime.CreateQueueProducer()
		if err != nil {
			// Log but don't fail - queue producer is optional
			// TODO: Once implemented, this should be required
			// panic(fmt.Sprintf("Failed to create queue producer: %v", err))
		} else {
			app.queueProducer = producer
		}
	}

	return app
}

// Router returns the Chi router for HTTP route registration
func (a *App) Router() *chi.Mux {
	return a.router
}

// Config returns the application configuration
func (a *App) Config() *Config {
	return a.config
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

// QueueProducer returns the queue producer for this runtime
func (a *App) QueueProducer() QueueProducer {
	if a.queueProducer == nil {
		panic("QueueProducer not initialized - this is a runtime implementation bug")
	}
	return a.queueProducer
}

// Run starts the application (local or cloud)
func (a *App) Run(ctx context.Context) error {
	// Runtime should already be initialized in New()
	if a.runtime == nil {
		panic("Runtime not initialized - this should never happen if New() was called correctly")
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

// defaultConfig creates a default configuration
func defaultConfig() *Config {
	config := &Config{}
	config.setDefaults()
	return config
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
