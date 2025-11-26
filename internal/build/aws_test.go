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

func TestLibStackTSWithoutExtend(t *testing.T) {
	var m config.Manifest
	m.App.Name = "testapp"
	layout := discover.Layout{
		Queues:    []discover.Queue{{Name: "work"}},
		Schedules: []discover.Schedule{{Name: "heartbeat", Every: time.Minute}},
	}

	content := libStackTS("testapp", m, layout, false)

	if strings.Contains(content, "import * as infra") {
		t.Error("should not contain infra import when hasExtend is false")
	}
	if strings.Contains(content, "infra.extend") {
		t.Error("should not contain extend call when hasExtend is false")
	}
	if strings.Contains(content, "infra.configure") {
		t.Error("should not contain configure call when hasExtend is false")
	}
}

func TestQueueVisibilityTimeout(t *testing.T) {
	var m config.Manifest
	m.App.Name = "testapp"
	layout := discover.Layout{
		Queues: []discover.Queue{{Name: "work"}},
	}

	// Test without extend - should have visibility timeout of 180s (default 30s Lambda timeout × 6)
	// AWS requires SQS visibility timeout >= Lambda timeout for event source mappings
	contentNoExtend := libStackTS("testapp", m, layout, false)
	if !strings.Contains(contentNoExtend, "visibilityTimeout: cdk.Duration.seconds(180)") {
		t.Error("queue should have visibilityTimeout of 180s (30s × 6) when not extending")
	}

	// Test with extend - should have visibility timeout based on config.timeout × 6
	contentExtend := libStackTS("testapp", m, layout, true)
	if !strings.Contains(contentExtend, "config.timeout?.toSeconds()") {
		t.Error("queue visibilityTimeout should use config.timeout when extending")
	}
	if !strings.Contains(contentExtend, "* 6") {
		t.Error("queue visibilityTimeout should use 6× multiplier (AWS recommendation)")
	}
}

func TestQueueVisibilityTimeoutFunction(t *testing.T) {
	// Test the queueVisibilityTimeout helper directly
	noExtend := queueVisibilityTimeout(false)
	if noExtend != "cdk.Duration.seconds(180)" {
		t.Errorf("without extend, expected 180s, got: %s", noExtend)
	}

	withExtend := queueVisibilityTimeout(true)
	if !strings.Contains(withExtend, "config.timeout?.toSeconds()") {
		t.Errorf("with extend, should reference config.timeout, got: %s", withExtend)
	}
	if !strings.Contains(withExtend, "* 6") {
		t.Errorf("with extend, should multiply by 6, got: %s", withExtend)
	}
	if !strings.Contains(withExtend, "?? 30") {
		t.Errorf("with extend, should default to 30s if no timeout configured, got: %s", withExtend)
	}
}

func TestLibStackTSWithExtend(t *testing.T) {
	var m config.Manifest
	m.App.Name = "testapp"
	layout := discover.Layout{
		Queues:    []discover.Queue{{Name: "work"}},
		Schedules: []discover.Schedule{{Name: "heartbeat", Every: time.Minute}},
	}

	content := libStackTS("testapp", m, layout, true)

	// infra/ is symlinked to dist/aws/cdk/infra, so path is ../infra/extend.ts from lib/
	// Use default import for CJS/ESM interop
	if !strings.Contains(content, `import infra from "../infra/extend.ts"`) {
		t.Error("should contain infra import with correct path when hasExtend is true")
	}
	if !strings.Contains(content, "infra.configure?.(this, env)") {
		t.Error("should contain configure call when hasExtend is true")
	}
	if !strings.Contains(content, "infra.extend?.(this, fn, env)") {
		t.Error("should contain extend call when hasExtend is true")
	}
	if !strings.Contains(content, "config.memorySize ?? 512") {
		t.Error("should use config.memorySize with fallback when hasExtend is true")
	}
	if !strings.Contains(content, "...config.environment") {
		t.Error("should spread config.environment when hasExtend is true")
	}
	// Verify environment block has valid syntax (no double commas or misplaced newlines)
	if strings.Contains(content, ",},") {
		t.Error("environment block should not have ',},' - invalid syntax")
	}
	if strings.Contains(content, ",...") {
		t.Error("environment block should not have ',... ' - need comma before spread")
	}
}

func TestCdkJSONUsesTsx(t *testing.T) {
	// Test without extend
	content := cdkJSON(false)
	if !strings.Contains(content, "npx tsx") {
		t.Error("cdk.json should use tsx for TypeScript execution")
	}
	if strings.Contains(content, "ts-node") {
		t.Error("cdk.json should not use ts-node (compatibility issues with Node v23+)")
	}
	if strings.Contains(content, "NODE_PATH") {
		t.Error("cdk.json without extend should not set NODE_PATH")
	}

	// Test with extend
	contentExtend := cdkJSON(true)
	if !strings.Contains(contentExtend, "NODE_PATH=$PWD/node_modules") {
		t.Error("cdk.json with extend should set NODE_PATH for module resolution")
	}
}

func TestCdkPackageJSONIncludesTsx(t *testing.T) {
	content := cdkPackageJSON()

	if !strings.Contains(content, `"tsx"`) {
		t.Error("package.json should include tsx as a dependency")
	}
}

func TestTsconfigJSONWithoutExtend(t *testing.T) {
	content := tsconfigJSON(false)

	if strings.Contains(content, "include") {
		t.Error("should not contain include when hasExtend is false")
	}
}

func TestTsconfigJSONWithExtend(t *testing.T) {
	content := tsconfigJSON(true)

	if !strings.Contains(content, `"include"`) {
		t.Error("should contain include when hasExtend is true")
	}
	// infra/ is symlinked to dist/aws/cdk/infra
	if !strings.Contains(content, `"infra/**/*"`) {
		t.Error("should include infra path when hasExtend is true")
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
