// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package build

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/transire/transire/internal/config"
	"github.com/transire/transire/internal/discover"
)

func TestToCDKDuration(t *testing.T) {
	if got := toCDKDuration(2 * time.Hour); !strings.Contains(got, "hours(2)") {
		t.Fatalf("unexpected hours duration: %s", got)
	}
	if got := toCDKDuration(90 * time.Second); !strings.Contains(got, "seconds(90)") {
		t.Fatalf("unexpected seconds duration: %s", got)
	}
	if got := toCDKDuration(0); !strings.Contains(got, "minutes(1)") {
		t.Fatalf("unexpected default duration: %s", got)
	}
}

func TestBuildAWSGeneratesArtifacts(t *testing.T) {
	root := t.TempDir()

	// Write a minimal manifest and handlers to drive discovery.
	if err := os.WriteFile(filepath.Join(root, "transire.yaml"), []byte("app:\n  name: testapp\naws:\n  region: us-east-1\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	_, file, _, _ := runtime.Caller(0)
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	goMod := "module testapp\n\nrequire github.com/transire/transire v0.0.0\n\nreplace github.com/transire/transire => " + repoRoot + "\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(root, "handlers"), 0o755); err != nil {
		t.Fatalf("mkdir handlers: %v", err)
	}
	handlerSrc := `package handlers
import (
	"time"
	"github.com/transire/transire"
)
func Register(app *transire.App) {
	app.RegisterQueueHandler("q", func(ctx transire.Context, msg transire.Message) error { return nil })
	app.RegisterScheduleHandler("s", time.Minute, func(ctx transire.Context, at time.Time) error { return nil })
}`
	if err := os.WriteFile(filepath.Join(root, "handlers", "handlers.go"), []byte(handlerSrc), 0o644); err != nil {
		t.Fatalf("write handler: %v", err)
	}

	// Minimal chi router usage to satisfy imports.
	appSrc := `package main
import (
	"context"
	"github.com/transire/transire"
	"github.com/transire/transire/dispatcher/local"
	"testapp/handlers"
)
func main() {
	app := transire.New()
	handlers.Register(app)
	app.SetDispatcher(&local.Dispatcher{})
	_ = app.Run(context.Background())
}`
	if err := os.MkdirAll(filepath.Join(root, "cmd", "app"), 0o755); err != nil {
		t.Fatalf("mkdir cmd: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "app", "main.go"), []byte(appSrc), 0o644); err != nil {
		t.Fatalf("write app main: %v", err)
	}

	lambdaSrc := `package main
import (
	"context"
	"log"
	"testapp/handlers"
	"github.com/transire/transire"
	"github.com/transire/transire/dispatcher/aws"
)
func main() {
	app := transire.New()
	handlers.Register(app)
	app.SetDispatcher(&aws.Dispatcher{})
	if err := app.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}`
	if err := os.MkdirAll(filepath.Join(root, "cmd", "lambda"), 0o755); err != nil {
		t.Fatalf("mkdir lambda: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "lambda", "main.go"), []byte(lambdaSrc), 0o644); err != nil {
		t.Fatalf("write lambda main: %v", err)
	}

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = root
	if out, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v, out: %s", err, out)
	}

	layout, err := discover.Scan(root)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	m, err := config.LoadManifest(filepath.Join(root, "transire.yaml"))
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}

	if err := BuildAWS(context.Background(), root, m, layout); err != nil {
		t.Fatalf("BuildAWS failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "dist", "aws", "lambda", "bootstrap")); err != nil {
		t.Fatalf("bootstrap not built: %v", err)
	}
	stackFile := filepath.Join(root, "dist", "aws", "cdk", "lib", "app-stack.ts")
	data, err := os.ReadFile(stackFile)
	if err != nil {
		t.Fatalf("read stack: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "ApiEndpoint") || !strings.Contains(content, "QueueUrl") {
		t.Fatalf("stack outputs missing expected entries")
	}
}
