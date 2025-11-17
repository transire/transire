package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Builder handles building Go applications
type Builder struct {
	projectDir string
}

// NewBuilder creates a new Builder
func NewBuilder(projectDir string) *Builder {
	return &Builder{
		projectDir: projectDir,
	}
}

// Build compiles the Go application and returns the output path
func (b *Builder) Build() (string, error) {
	// Ensure project directory exists
	if _, err := os.Stat(b.projectDir); err != nil {
		return "", fmt.Errorf("project directory does not exist: %w", err)
	}

	// Check for go.mod
	goModPath := filepath.Join(b.projectDir, "go.mod")
	if _, err := os.Stat(goModPath); err != nil {
		return "", fmt.Errorf("no go.mod found in %s", b.projectDir)
	}

	// Create .transire directory for build artifacts
	transireDir := filepath.Join(b.projectDir, ".transire")
	if err := os.MkdirAll(transireDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create .transire directory: %w", err)
	}

	// Determine output path
	outputPath := b.GetOutputPath()

	// Build the application
	cmd := exec.Command("go", "build", "-o", outputPath, ".")
	cmd.Dir = b.projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("build failed: %w", err)
	}

	return outputPath, nil
}

// GetOutputPath returns the path where the binary will be built
func (b *Builder) GetOutputPath() string {
	// Get the module name from go.mod to use as binary name
	moduleName := b.getModuleName()
	if moduleName == "" {
		moduleName = "app"
	}

	// Use just the last component of the module name
	parts := strings.Split(moduleName, "/")
	binaryName := parts[len(parts)-1]

	transireDir := filepath.Join(b.projectDir, ".transire")
	return filepath.Join(transireDir, binaryName)
}

// Clean removes build artifacts
func (b *Builder) Clean() error {
	transireDir := filepath.Join(b.projectDir, ".transire")
	if _, err := os.Stat(transireDir); os.IsNotExist(err) {
		return nil // Already clean
	}

	return os.RemoveAll(transireDir)
}

// getModuleName extracts the module name from go.mod
func (b *Builder) getModuleName() string {
	goModPath := filepath.Join(b.projectDir, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return ""
	}

	// Parse module name from first line
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module"))
		}
	}

	return ""
}
