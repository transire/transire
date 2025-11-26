// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/transire/transire/internal/version"
)

// Generate creates a starter Transire project in the target directory.
func Generate(targetDir, moduleName string) error {
	if moduleName == "" {
		moduleName = filepath.Base(targetDir)
	}

	if err := os.MkdirAll(filepath.Join(targetDir, "cmd", "app"), 0o755); err != nil {
		return err
	}

	files := map[string]string{
		"cmd/app/main.go":      mainTemplate(moduleName),
		"handlers/http.go":     httpTemplate(),
		"handlers/queue.go":    queueTemplate(),
		"handlers/schedule.go": scheduleTemplate(),
		"handlers/model.go":    modelTemplate(),
		"infra/.gitkeep":       "",
		"LICENSE":              mitLicense,
		"go.mod":               goModTemplate(moduleName),
		".gitignore":           gitignoreTemplate,
		"README.md":            readmeTemplate,
		"transire.yaml":        manifestTemplate(moduleName),
	}

	for rel, contents := range files {
		path := filepath.Join(targetDir, rel)
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("refusing to overwrite existing file: %s", rel)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			return err
		}
	}

	return nil
}

func mainTemplate(module string) string {
	return fmt.Sprintf(`package main

import (
	"context"
	"log"

	"github.com/go-chi/chi/v5/middleware"
	"%s/handlers"
	"github.com/transire/transire"
	"github.com/transire/transire/dispatcher"
)

func main() {
	app := transire.New()
	app.Router().Use(middleware.Logger)

	handlers.RegisterHTTP(app)
	handlers.RegisterQueues(app)
	handlers.RegisterSchedules(app)

	d, err := dispatcher.Auto()
	if err != nil {
		log.Fatal(err)
	}
	app.SetDispatcher(d)

	if err := app.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
`, module)
}

func httpTemplate() string {
	return `package handlers

import (
	"fmt"
	"net/http"

	"github.com/transire/transire"
)

func RegisterHTTP(app *transire.App) {
	app.Router().Get("/", func(w http.ResponseWriter, r *http.Request) {
		ctx, ok := transire.RequestContext(r)
		if !ok {
			http.Error(w, "missing transire context", http.StatusInternalServerError)
			return
		}

		msg := r.URL.Query().Get("msg")
		if msg == "" {
			msg = "hello from http"
		}
		if err := ctx.Queues.Send(r.Context(), WorkQueue, []byte(msg)); err != nil {
			http.Error(w, "failed to enqueue", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(fmt.Sprintf("queued: %s", msg)))
	})
}
`
}

func queueTemplate() string {
	return `package handlers

import (
	"fmt"
	"log"

	"github.com/transire/transire"
)

func RegisterQueues(app *transire.App) {
	app.RegisterQueueHandler(WorkQueue, func(ctx transire.Context, msg transire.Message) error {
		log.Printf("work queue received: %s", string(msg.Body))
		return ctx.Queues.Send(ctx, AuditQueue, []byte(fmt.Sprintf("audit: %s", msg.Body)))
	})

	app.RegisterQueueHandler(AuditQueue, func(ctx transire.Context, msg transire.Message) error {
		log.Printf("audit log: %s", string(msg.Body))
		return nil
	})
}
`
}

func scheduleTemplate() string {
	return `package handlers

import (
	"fmt"
	"log"
	"time"

	"github.com/transire/transire"
)

func RegisterSchedules(app *transire.App) {
	app.RegisterScheduleHandler("heartbeat", time.Minute, func(ctx transire.Context, at time.Time) error {
		log.Printf("heartbeat at %s", at.UTC())
		return ctx.Queues.Send(ctx, WorkQueue, []byte(fmt.Sprintf("heartbeat at %s", at.UTC().Format(time.RFC3339))))
	})
}
`
}

func modelTemplate() string {
	return `package handlers

const (
	WorkQueue  = "work"
	AuditQueue = "audit-log"
)
`
}

func goModTemplate(module string) string {
	return fmt.Sprintf(`module %s

go 1.25.4

require (
	github.com/go-chi/chi/v5 v5.2.3
	github.com/transire/transire %s
)
`, module, version.Version)
}

var gitignoreTemplate = strings.TrimSpace(`
bin/
dist/
.DS_Store
`) + "\n"

var readmeTemplate = strings.TrimSpace(`
# Transire app

This project was bootstrapped by "transire init". It shows HTTP, queue, and schedule handlers all producing queue messages.

## Local quickstart

- Run: transire run --port 8080
- HTTP: curl "http://localhost:8080/?msg=hi"
- Send to queue: transire send work "manual message" (defaults to env=local)
- Trigger schedule: transire trigger heartbeat (defaults to env=local)

## AWS (profile: transire-sandbox)

- Deploy: transire deploy --profile transire-sandbox --env dev
- Find endpoints/queues: transire info --env dev --profile transire-sandbox
- HTTP: curl "https://<api-endpoint>/?msg=hi"
- Queue: transire send work "manual message" --env dev --profile transire-sandbox
- Schedule: transire trigger heartbeat --env dev --profile transire-sandbox
`) + "\n"

func manifestTemplate(module string) string {
	name := filepath.Base(module)
	return fmt.Sprintf(`app:
  name: %s
envs:
  dev:
    profile: transire-sandbox
`, name)
}

const mitLicense = `MIT License

Copyright (c) 2023 Transire Contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
`
