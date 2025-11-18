// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package transire

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// localRuntime implements the Runtime interface for local development
type localRuntime struct {
	config   *Config
	server   *http.Server
	stopChan chan struct{}
	mu       sync.Mutex
}

// newLocalRuntime creates a new local runtime
func newLocalRuntime(config *Config) Runtime {
	return &localRuntime{
		config:   config,
		stopChan: make(chan struct{}),
	}
}

// Start begins processing in the local environment
func (r *localRuntime) Start(ctx context.Context, app *App) error {
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

	// TODO: Start queue and scheduler simulators

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
func (r *localRuntime) Stop(ctx context.Context) error {
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

	// TODO: Stop queue and scheduler simulators

	return nil
}

// IsLocal returns true since this is the local runtime
func (r *localRuntime) IsLocal() bool {
	return true
}

// logRegisteredHandlers logs information about registered handlers
func (r *localRuntime) logRegisteredHandlers(app *App) {
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

// localMessage implements the Message interface for local development
type localMessage struct {
	id            string
	body          []byte
	attributes    map[string]string
	deliveryCount int
	enqueuedAt    time.Time
}

func (m *localMessage) ID() string {
	return m.id
}

func (m *localMessage) Body() []byte {
	return m.body
}

func (m *localMessage) Attributes() map[string]string {
	return m.attributes
}

func (m *localMessage) DeliveryCount() int {
	return m.deliveryCount
}

func (m *localMessage) EnqueuedAt() time.Time {
	return m.enqueuedAt
}
