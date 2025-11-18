// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/transire/transire/pkg/transire"
)

func TestScaffolder_GenerateGoMod_Issue6(t *testing.T) {
	// Test for Issue #6: Generated go.mod should use proper semantic version
	t.Run("should generate valid v0.1.0 version", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := &transire.Config{
			Name:     "test-app",
			Language: "go",
			Cloud:    "aws",
			Runtime:  "lambda",
			IaC:      "cdk",
		}

		scaffolder := New(config, tmpDir)

		// Generate the go.mod file
		err := scaffolder.generateGoMod()
		if err != nil {
			t.Fatalf("generateGoMod() failed: %v", err)
		}

		// Read the generated go.mod file
		goModPath := filepath.Join(tmpDir, "go.mod")
		content, err := os.ReadFile(goModPath)
		if err != nil {
			t.Fatalf("Failed to read generated go.mod: %v", err)
		}

		goModContent := string(content)

		// Should contain the proper semantic version
		if !strings.Contains(goModContent, "github.com/transire/transire v0.1.0") {
			t.Errorf("go.mod should contain semantic version v0.1.0")
			t.Logf("Generated go.mod content:\n%s", goModContent)
		}

		// Should contain a valid version reference
		if !strings.Contains(goModContent, "github.com/transire/transire") {
			t.Errorf("go.mod should contain github.com/transire/transire dependency")
		}
	})
}

func TestScaffolder_GenerateMainGo_Issue5(t *testing.T) {
	// Test for Issue #5: Generated main.go doesn't load config
	t.Run("should generate main.go that loads transire.yaml config", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := &transire.Config{
			Name:     "test-app",
			Language: "go",
			Cloud:    "aws",
			Runtime:  "lambda",
			IaC:      "cdk",
		}

		scaffolder := New(config, tmpDir)

		// Generate the main.go file
		err := scaffolder.generateMainGo()
		if err != nil {
			t.Fatalf("generateMainGo() failed: %v", err)
		}

		// Read the generated main.go file
		mainGoPath := filepath.Join(tmpDir, "main.go")
		content, err := os.ReadFile(mainGoPath)
		if err != nil {
			t.Fatalf("Failed to read generated main.go: %v", err)
		}

		mainGoContent := string(content)

		// Should contain config loading
		// For now, we expect this test to fail until we fix the issue
		if !strings.Contains(mainGoContent, "LoadConfig") || !strings.Contains(mainGoContent, "WithConfig") {
			t.Errorf("main.go should load transire.yaml config using LoadConfig and WithConfig")
			t.Logf("Generated main.go content:\n%s", mainGoContent)
		}

		// Current broken implementation just calls transire.New() without config
		if strings.Contains(mainGoContent, "app := transire.New()") && !strings.Contains(mainGoContent, "transire.New(transire.WithConfig") {
			t.Errorf("main.go should not call transire.New() without loading config")
		}
	})
}

func TestScaffolder_GenerateGitIgnore_Issue7(t *testing.T) {
	// Test for Issue #7: Should include patterns to prevent Go module pollution
	t.Run("should generate .gitignore that prevents CDK node_modules pollution", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := &transire.Config{
			Name:     "test-app",
			Language: "go",
			Cloud:    "aws",
			Runtime:  "lambda",
			IaC:      "cdk",
		}

		scaffolder := New(config, tmpDir)

		// Generate the .gitignore file
		err := scaffolder.generateGitIgnore()
		if err != nil {
			t.Fatalf("generateGitIgnore() failed: %v", err)
		}

		// Read the generated .gitignore file
		gitIgnorePath := filepath.Join(tmpDir, ".gitignore")
		content, err := os.ReadFile(gitIgnorePath)
		if err != nil {
			t.Fatalf("Failed to read generated .gitignore: %v", err)
		}

		gitIgnoreContent := string(content)

		// Should contain the node_modules ignore pattern
		if !strings.Contains(gitIgnoreContent, "infrastructure/node_modules/") {
			t.Errorf(".gitignore should contain infrastructure/node_modules/ pattern")
			t.Logf("Generated .gitignore content:\n%s", gitIgnoreContent)
		}

		// Should also verify this actually works by testing go module detection
		// But for now, the .gitignore pattern is good enough prevention
	})
}

func TestScaffolder_FullGeneration(t *testing.T) {
	// Integration test for full project generation
	t.Run("should generate complete project structure", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := &transire.Config{
			Name:     "test-app",
			Language: "go",
			Cloud:    "aws",
			Runtime:  "lambda",
			IaC:      "cdk",
			CI:       "github",
		}
		config.SetDefaults()

		scaffolder := New(config, tmpDir)

		// Generate the full project
		err := scaffolder.Generate()
		if err != nil {
			t.Fatalf("Generate() failed: %v", err)
		}

		// Verify all expected files exist
		expectedFiles := []string{
			"go.mod",
			"main.go",
			"transire.yaml",
			".gitignore",
			"infrastructure/package.json",
			".github/workflows/deploy.yml",
		}

		for _, file := range expectedFiles {
			filePath := filepath.Join(tmpDir, file)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Errorf("Expected file %s does not exist", file)
			}
		}
	})
}
