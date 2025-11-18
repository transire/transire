package aws

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/transire/transire/pkg/transire"
)

// LambdaBuilder builds artifacts for AWS Lambda deployment
type LambdaBuilder struct {
	config transire.BuildConfig
}

// NewLambdaBuilder creates a new Lambda builder
func NewLambdaBuilder(config transire.BuildConfig) *LambdaBuilder {
	return &LambdaBuilder{
		config: config,
	}
}

// Build creates the Lambda deployment package
func (b *LambdaBuilder) Build(ctx context.Context) error {
	// Ensure output directory exists
	if err := os.MkdirAll(b.config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build the Go binary for ARM64 Lambda
	if err := b.buildGoBinary(ctx); err != nil {
		return fmt.Errorf("failed to build Go binary: %w", err)
	}

	// Create the Lambda ZIP package
	if err := b.createZipPackage(); err != nil {
		return fmt.Errorf("failed to create ZIP package: %w", err)
	}

	return nil
}

// buildGoBinary compiles the Go application for Lambda
func (b *LambdaBuilder) buildGoBinary(ctx context.Context) error {
	// Build tags to exclude local development code
	buildTags := append(b.config.ExcludeTags, "!local")

	// Prepare build command
	args := []string{"build"}

	if len(buildTags) > 0 {
		args = append(args, "-tags", joinTags(buildTags))
	}

	if b.config.Optimizations {
		// Strip debug information for smaller binary
		args = append(args, "-ldflags", "-s -w")
	}

	// Set output binary name (Lambda requires "bootstrap")
	binaryPath := filepath.Join(b.config.OutputDir, "bootstrap")
	args = append(args, "-o", binaryPath, ".")

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = b.config.AppPath
	cmd.Env = append(os.Environ(),
		"GOOS=linux",
		"GOARCH="+b.config.Architecture,
		"CGO_ENABLED=0",
	)

	// Set additional environment variables
	for key, value := range b.config.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// createZipPackage creates the Lambda deployment ZIP file
func (b *LambdaBuilder) createZipPackage() error {
	binaryPath := filepath.Join(b.config.OutputDir, "bootstrap")
	zipPath := filepath.Join(b.config.OutputDir, "function.zip")

	// Use zip command to create the package
	cmd := exec.Command("zip", "-j", zipPath, binaryPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("zip creation failed: %w\nOutput: %s", err, string(output))
	}

	// Remove the binary file, keep only the ZIP
	if err := os.Remove(binaryPath); err != nil {
		return fmt.Errorf("failed to remove binary file: %w", err)
	}

	return nil
}

// joinTags joins build tags with appropriate logic
func joinTags(tags []string) string {
	// Simple implementation - could be enhanced for complex tag logic
	result := ""
	for i, tag := range tags {
		if i > 0 {
			result += " && "
		}
		result += tag
	}
	return result
}
