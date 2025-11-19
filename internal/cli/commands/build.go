// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package commands

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/transire/transire/internal/cli/discovery"
	"github.com/transire/transire/pkg/transire"
)

// NewBuildCommand creates the build command
func NewBuildCommand() *cobra.Command {
	var (
		configPath  = ""
		outputDir   = "dist"
		environment = "dev"
	)

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build deployable artifacts",
		Long: `Build deployable artifacts for your Transire application.

This command:
- Compiles the application for the target runtime (ARM64 for Lambda)
- Excludes local development dependencies
- Creates deployment packages (ZIP files for Lambda)
- Generates Infrastructure as Code definitions
- Updates CDK stacks based on registered handlers

The build process respects build tags to exclude local-only code.

Examples:
  transire build
  transire build --output ./build
  transire build --config production-transire.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			config, err := loadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			log.Printf("🔨 Building Transire application: %s", config.Name)

			// Discover application handlers
			project, err := discovery.DiscoverProject(".", config)
			if err != nil {
				return fmt.Errorf("failed to discover project: %w", err)
			}

			// Create output directory
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}

			// Create provider based on configuration
			provider, err := createProvider(config)
			if err != nil {
				return fmt.Errorf("failed to create provider: %w", err)
			}

			ctx := context.Background()

			// Build artifacts
			log.Printf("📦 Building artifacts for %s/%s", config.Cloud, config.Runtime)
			buildConfig := transire.BuildConfig{
				AppPath:       ".",
				OutputDir:     outputDir,
				Architecture:  config.Lambda.Architecture,
				Environment:   config.Environment,
				ExcludeTags:   []string{"local"},
				Optimizations: true,
			}

			if err := provider.BuildArtifacts(ctx, buildConfig); err != nil {
				return fmt.Errorf("failed to build artifacts: %w", err)
			}

			// Generate Infrastructure as Code
			log.Printf("🏗️  Generating infrastructure definitions")
			iacConfig := transire.IaCConfig{
				StackName:         config.Name + "-" + environment,
				AppName:           config.Name,
				Environment:       environment,
				FunctionGroups:    convertFunctionGroups(config.Functions),
				HTTPHandlers:      project.HTTPHandlers,
				QueueHandlers:     project.QueueHandlers,
				ScheduleHandlers:  project.ScheduleHandlers,
				ExistingResources: config.ExistingResources,
			}

			if err := provider.GenerateIaC(ctx, iacConfig); err != nil {
				return fmt.Errorf("failed to generate infrastructure: %w", err)
			}

			log.Printf("✅ Build completed successfully")
			log.Printf("📁 Artifacts location: %s", outputDir)
			log.Printf("🏗️  Infrastructure: infrastructure/")

			return nil
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to configuration file")
	cmd.Flags().StringVarP(&outputDir, "output", "o", "dist", "Output directory for artifacts")
	cmd.Flags().StringVarP(&environment, "environment", "e", "dev", "Target environment (dev, staging, prod)")

	return cmd
}

// convertFunctionGroups converts config function groups to IaC format
func convertFunctionGroups(functions map[string]transire.FunctionConfig) map[string]transire.FunctionGroupSpec {
	result := make(map[string]transire.FunctionGroupSpec)

	for name, config := range functions {
		spec := transire.FunctionGroupSpec{
			Include:             config.Include[0], // Simplification for now
			MemoryMB:            config.MemoryMB,
			TimeoutSeconds:      config.TimeoutSeconds,
			ReservedConcurrency: config.ReservedConcurrency,
			Environment:         config.Environment,
		}
		result[name] = spec
	}

	return result
}
