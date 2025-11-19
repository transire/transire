//go:build local

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package local

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/transire/transire/pkg/transire"
)

// Runtime implements the transire.Runtime interface for local development
type Runtime struct {
	config   *transire.Config
	server   *http.Server
	stopChan chan struct{}
	mu       sync.Mutex
}

// NewLocalRuntime creates a new local runtime
func NewLocalRuntime(config *transire.Config) *Runtime {
	return &Runtime{
		config:   config,
		stopChan: make(chan struct{}),
	}
}

// Start begins processing in the local environment
func (r *Runtime) Start(ctx context.Context, app *transire.App) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.server != nil {
		return fmt.Errorf("runtime already started")
	}

	// Start HTTP server
	port := r.config.Development.HTTPPort
	if port == 0 {
		port = 3000
	}

	addr := ":" + strconv.Itoa(port)
	r.server = &http.Server{
		Addr:    addr,
		Handler: newDevHandler(app),
	}

	log.Printf("Starting Transire local runtime")
	log.Printf("HTTP server listening on %s", addr)

	// Log registered handlers
	r.logRegisteredHandlers(app)

	// Start server in goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		if err := r.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrChan <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		log.Printf("Shutting down due to context cancellation")
		return r.Stop(context.Background())
	case err := <-serverErrChan:
		return fmt.Errorf("server error: %w", err)
	case <-r.stopChan:
		return nil
	}
}

// Stop gracefully shuts down the local runtime
func (r *Runtime) Stop(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.server == nil {
		return nil
	}

	log.Printf("Stopping Transire local runtime")

	// Shutdown HTTP server
	if err := r.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("error shutting down server: %w", err)
	}

	r.server = nil

	// Signal stop
	select {
	case r.stopChan <- struct{}{}:
	default:
	}

	return nil
}

// IsLocal returns true since this is the local runtime
func (r *Runtime) IsLocal() bool {
	return true
}

// CreateQueueProducer returns a queue producer for local development
func (r *Runtime) CreateQueueProducer() (transire.QueueProducer, error) {
	// TODO: Implement local queue producer with in-memory simulator
	return nil, fmt.Errorf("queue producer not yet implemented for local runtime")
}

// logRegisteredHandlers logs information about registered handlers
func (r *Runtime) logRegisteredHandlers(app *transire.App) {
	queueHandlers := app.GetQueueHandlers()
	schedHandlers := app.GetScheduleHandlers()

	if len(queueHandlers) > 0 {
		log.Printf("Registered queue handlers:")
		for _, handler := range queueHandlers {
			log.Printf("  - %s", handler.QueueName())
		}
	}

	if len(schedHandlers) > 0 {
		log.Printf("Registered schedule handlers:")
		for _, handler := range schedHandlers {
			log.Printf("  - %s (%s)", handler.Name(), handler.Schedule())
		}
	}
}

// init registers the local runtime during package initialization
func init() {
	transire.RegisterDefaultRuntime(func(config *transire.Config) transire.Runtime {
		return NewLocalRuntime(config)
	})
}
