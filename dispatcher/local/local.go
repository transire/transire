// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package local

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	transire "github.com/transire/transire"
)

// Dispatcher provides a lightweight dispatcher for local development and testing.
type Dispatcher struct {
	HTTPAddr string
}

// Name identifies the dispatcher.
func (d *Dispatcher) Name() string {
	return "local"
}

// Run starts the HTTP server and wires in local queue handling.
func (d *Dispatcher) Run(ctx context.Context, app *transire.App) error {
	addr := resolveAddr(d.HTTPAddr)

	ensureQueueSender(app)
	root := buildHandler(app)

	startSchedules(ctx, app)

	server := &http.Server{
		Addr:    addr,
		Handler: root,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	log.Printf("transire local dispatcher listening on %s\n", addr)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func resolveAddr(addr string) string {
	if addr != "" {
		return addr
	}
	if env := os.Getenv("TRANSIRE_HTTP_ADDR"); env != "" {
		return env
	}
	if env := os.Getenv("PORT"); env != "" {
		return ":" + env
	}
	if env := os.Getenv("TRANSIRE_PORT"); env != "" {
		return ":" + env
	}
	return ":8080"
}

func ensureQueueSender(app *transire.App) {
	if app.QueueSender() == nil {
		app.SetQueueSender(&queueSender{app: app})
	}
}

func buildHandler(app *transire.App) http.Handler {
	ensureQueueSender(app)

	root := chi.NewRouter()
	root.Use(transire.InjectContext(app.QueueSender()))

	root.Route("/_transire", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})

		r.Post("/queues/{name}", func(w http.ResponseWriter, r *http.Request) {
			queue := chi.URLParam(r, "name")
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read body", http.StatusBadRequest)
				return
			}
			if err := app.QueueSender().Send(r.Context(), queue, body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusAccepted)
		})

		r.Post("/schedules/{name}", func(w http.ResponseWriter, r *http.Request) {
			schedule := chi.URLParam(r, "name")
			sched, ok := app.Schedules()[schedule]
			if !ok {
				http.NotFound(w, r)
				return
			}
			if sched.Handler == nil {
				http.Error(w, "schedule handler missing", http.StatusBadRequest)
				return
			}
			if err := sched.Handler(transire.Context{
				Context: r.Context(),
				Queues:  app.QueueSender(),
			}, time.Now()); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusAccepted)
		})
	})

	root.Mount("/", app.Router())
	return root
}

type queueSender struct {
	app *transire.App
}

func (q *queueSender) Send(ctx context.Context, queue string, payload []byte) error {
	handler, ok := q.app.QueueHandlers()[queue]
	if !ok {
		return fmt.Errorf("queue %q not registered", queue)
	}

	go func() {
		handler(transire.Context{
			Context: ctx,
			Queues:  q,
		}, transire.Message{
			ID:         fmt.Sprintf("local-%d", time.Now().UnixNano()),
			Queue:      queue,
			Body:       payload,
			Attributes: map[string]string{},
		})
	}()

	return nil
}

func startSchedules(ctx context.Context, app *transire.App) {
	for name, sched := range app.Schedules() {
		interval := sched.Every
		if interval <= 0 {
			log.Printf("schedule %s has non-positive interval; skipping\n", name)
			continue
		}
		s := sched
		go func() {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case t := <-ticker.C:
					_ = s.Handler(transire.Context{
						Context: ctx,
						Queues:  app.QueueSender(),
					}, t)
				case <-ctx.Done():
					return
				}
			}
		}()
	}
}
