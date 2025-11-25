// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package discover

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestScanFindsHandlers(t *testing.T) {
	dir := writeModule(t, `package handlers
import (
	"time"
	"github.com/transire/transire"
)
func Register(app *transire.App) {
	app.RegisterQueueHandler("alpha", func(ctx transire.Context, msg transire.Message) error { return nil })
	app.RegisterScheduleHandler("tick", time.Minute, func(ctx transire.Context, at time.Time) error { return nil })
}`)

	layout, err := Scan(dir)
	if err != nil {
		t.Fatalf("scan error: %v", err)
	}
	if len(layout.Queues) != 1 || layout.Queues[0].Name != "alpha" {
		t.Fatalf("unexpected queues: %+v", layout.Queues)
	}
	if len(layout.Schedules) != 1 || layout.Schedules[0].Name != "tick" || layout.Schedules[0].Every != time.Minute {
		t.Fatalf("unexpected schedules: %+v", layout.Schedules)
	}
}

func TestScanFindsConstNames(t *testing.T) {
	dir := writeModule(t, `package handlers
import (
	"time"
	"github.com/transire/transire"
)
const queueName = "const-queue"
const schedName = "const-schedule"
func Register(app *transire.App) {
	app.RegisterQueueHandler(queueName, func(ctx transire.Context, msg transire.Message) error { return nil })
	app.RegisterScheduleHandler(schedName, time.Minute, func(ctx transire.Context, at time.Time) error { return nil })
}`)

	layout, err := Scan(dir)
	if err != nil {
		t.Fatalf("scan error: %v", err)
	}
	if len(layout.Queues) != 1 || layout.Queues[0].Name != "const-queue" {
		t.Fatalf("unexpected queues: %+v", layout.Queues)
	}
	if len(layout.Schedules) != 1 || layout.Schedules[0].Name != "const-schedule" {
		t.Fatalf("unexpected schedules: %+v", layout.Schedules)
	}
}

func TestScanIgnoresNonLiterals(t *testing.T) {
	dir := writeModule(t, `package handlers
import "github.com/transire/transire"
func Register(app *transire.App) {
	name := "dyn"
	app.RegisterQueueHandler(name, func(ctx transire.Context, msg transire.Message) error { return nil })
}`)
	layout, err := Scan(dir)
	if err != nil {
		t.Fatalf("scan error: %v", err)
	}
	if len(layout.Queues) != 0 {
		t.Fatalf("expected no queues, got %+v", layout.Queues)
	}
}

func writeModule(t *testing.T, src string) string {
	t.Helper()
	dir := t.TempDir()
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	goMod := "module example.com/test\n\nrequire github.com/transire/transire v0.0.0\n\nreplace github.com/transire/transire => " + repoRoot + "\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "handlers.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v, out: %s", err, out)
	}
	return dir
}
