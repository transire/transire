// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package transire

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigLoadingIntegration_Issue5(t *testing.T) {
	// Test for Issue #5: Verify that when config is loaded properly, the http_port is respected
	t.Run("should respect http_port configuration when config is loaded", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a transire.yaml with custom http_port
		configContent := `name: test-app
language: go
cloud: aws
runtime: lambda
iac: cdk
ci: github

lambda:
  architecture: arm64
  timeout_seconds: 30
  memory_mb: 128

development:
  http_port: 8080
  queue_port: 4000
  scheduler_port: 5000
  auto_reload: true
  log_level: info
`

		configPath := filepath.Join(tmpDir, "transire.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write test config file: %v", err)
		}

		// Load the config
		config, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() failed: %v", err)
		}

		// Verify the config was loaded correctly
		if config.Development.HTTPPort != 8080 {
			t.Errorf("Expected HTTPPort to be 8080, got %d", config.Development.HTTPPort)
		}

		// Create app with loaded config
		app := New(WithConfig(config))

		// Verify the app has the correct config
		appConfig := app.GetConfig()
		if appConfig.Development.HTTPPort != 8080 {
			t.Errorf("App config HTTPPort should be 8080, got %d", appConfig.Development.HTTPPort)
		}

		// Create local runtime with the config to verify it uses the correct port
		runtime := newLocalRuntime(config)
		localRt, ok := runtime.(*localRuntime)
		if !ok {
			t.Fatalf("Expected localRuntime, got %T", runtime)
		}

		// Verify the runtime will use the correct port
		// We can't actually start the server in tests due to port conflicts,
		// but we can verify the config is set correctly
		if localRt.config.Development.HTTPPort != 8080 {
			t.Errorf("Runtime config HTTPPort should be 8080, got %d", localRt.config.Development.HTTPPort)
		}
	})

	t.Run("should use default port when config not loaded", func(t *testing.T) {
		// This demonstrates the current broken behavior
		// When no config is loaded (like in generated main.go), it uses defaults
		app := New() // This is what the scaffold generates - NO CONFIG LOADING

		config := app.GetConfig()
		if config.Development.HTTPPort != 3000 {
			t.Errorf("Default HTTPPort should be 3000, got %d", config.Development.HTTPPort)
		}

		// This shows the problem: the generated template always uses defaults
		// regardless of what's in transire.yaml
	})
}
