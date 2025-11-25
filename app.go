// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package transire

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// QueueSender provides an abstraction for sending messages to queues.
type QueueSender interface {
	Send(ctx context.Context, queue string, payload []byte) error
}

// Context is passed to all handlers and exposes cloud-agnostic primitives.
type Context struct {
	context.Context
	Queues QueueSender
}

// Message represents a queue message.
type Message struct {
	ID         string
	Queue      string
	Body       []byte
	Attributes map[string]string
}

// QueueHandler processes a message pulled from a queue.
type QueueHandler func(ctx Context, msg Message) error

// ScheduleHandler processes a scheduled invocation.
type ScheduleHandler func(ctx Context, at time.Time) error

// Dispatcher routes events from a concrete runtime (local, AWS, etc) into handlers.
type Dispatcher interface {
	Run(ctx context.Context, app *App) error
	Name() string
}

// App is the root container for an application built with Transire.
// It aggregates HTTP, queue, and scheduler handlers behind cloud-agnostic interfaces.
// Schedule represents a scheduled task with a fixed rate.
type Schedule struct {
	Name     string
	Every    time.Duration
	Handler  ScheduleHandler
	Metadata map[string]string
}

type App struct {
	router        *chi.Mux
	queueHandlers map[string]QueueHandler
	schedules     map[string]Schedule
	dispatcher    Dispatcher
	queueSender   QueueSender
}

// New creates a new application with a chi router and empty handler registries.
func New() *App {
	r := chi.NewRouter()
	return &App{
		router:        r,
		queueHandlers: map[string]QueueHandler{},
		schedules:     map[string]Schedule{},
	}
}

// Router returns the chi router so users can add middleware and routes.
func (a *App) Router() *chi.Mux {
	return a.router
}

// Use registers chi middlewares on the application's router.
func (a *App) Use(middlewares ...func(http.Handler) http.Handler) {
	a.router.Use(middlewares...)
}

// RegisterQueueHandler binds a handler to a named queue.
func (a *App) RegisterQueueHandler(queue string, handler QueueHandler) {
	a.queueHandlers[queue] = handler
}

// RegisterScheduleHandler binds a schedule name to a handler that runs every duration.
func (a *App) RegisterScheduleHandler(name string, every time.Duration, handler ScheduleHandler) {
	a.schedules[name] = Schedule{
		Name:    name,
		Every:   every,
		Handler: handler,
	}
}

// QueueHandlers exposes registered queue handlers.
func (a *App) QueueHandlers() map[string]QueueHandler {
	return a.queueHandlers
}

// Schedules exposes registered schedules.
func (a *App) Schedules() map[string]Schedule {
	return a.schedules
}

// RouterHandler exposes the chi router for HTTP serving.
func (a *App) RouterHandler() http.Handler {
	return a.router
}

// SetQueueSender configures the queue sender used inside handler contexts.
func (a *App) SetQueueSender(sender QueueSender) {
	a.queueSender = sender
}

// QueueSender returns the configured queue sender.
func (a *App) QueueSender() QueueSender {
	return a.queueSender
}

// SetDispatcher defines which dispatcher should run the app.
func (a *App) SetDispatcher(dispatcher Dispatcher) {
	a.dispatcher = dispatcher
}

// Dispatcher returns the configured dispatcher.
func (a *App) Dispatcher() Dispatcher {
	return a.dispatcher
}

// Run wires the dispatcher and starts dispatching events.
func (a *App) Run(ctx context.Context) error {
	if a.dispatcher == nil {
		return ErrNoDispatcher
	}
	return a.dispatcher.Run(ctx, a)
}

// ErrNoDispatcher signals Run was called without wiring a dispatcher.
var ErrNoDispatcher = errors.New("transire: no dispatcher configured")
