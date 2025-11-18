// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/transire/transire/internal/cli/scaffold"
	"github.com/transire/transire/pkg/transire"
)

// NewInitCommand creates the init command
func NewInitCommand() *cobra.Command {
	var (
		language = "go"
		cloud    = "aws"
		runtime  = "lambda"
		iac      = "cdk"
		ci       = "github"
		force    = false
	)

	cmd := &cobra.Command{
		Use:   "init [project-name]",
		Short: "Initialize a new Transire project",
		Long: `Initialize a new Transire project with sane defaults.

Creates a minimal, idiomatic application with:
- Application code scaffolding
- Transire configuration
- Infrastructure as Code setup
- CI/CD pipeline configuration

Examples:
  transire init my-app
  transire init my-app --language go --cloud aws --runtime lambda
  transire init . --force  # Initialize in current directory`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine project directory
			var projectDir string
			if len(args) > 0 {
				projectDir = args[0]
			} else {
				// Default to current directory name or prompt user
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}
				projectDir = filepath.Base(cwd)
			}

			// Resolve absolute path
			absPath, err := filepath.Abs(projectDir)
			if err != nil {
				return fmt.Errorf("failed to resolve project path: %w", err)
			}

			projectName := filepath.Base(absPath)

			// Check if directory already exists and is not empty
			if !force {
				if err := checkDirectoryEmpty(absPath); err != nil {
					return err
				}
			}

			// Create project configuration
			config := &transire.Config{
				Name:     projectName,
				Language: language,
				Cloud:    cloud,
				Runtime:  runtime,
				IaC:      iac,
				CI:       ci,
			}

			// Set defaults
			config.SetDefaults()

			// Validate configuration
			if err := config.Validate(); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}

			// Create scaffolder
			scaffolder := scaffold.New(config, absPath)

			// Generate project
			if err := scaffolder.Generate(); err != nil {
				return fmt.Errorf("failed to generate project: %w", err)
			}

			fmt.Printf("✅ Successfully initialized Transire project '%s'\n", projectName)
			fmt.Printf("📁 Project directory: %s\n", absPath)
			fmt.Printf("\n🚀 Next steps:\n")
			fmt.Printf("   cd %s\n", projectDir)
			fmt.Printf("   transire run\n")

			return nil
		},
	}

	// Add flags
	cmd.Flags().StringVar(&language, "language", "go", "Programming language (go)")
	cmd.Flags().StringVar(&cloud, "cloud", "aws", "Cloud provider (aws)")
	cmd.Flags().StringVar(&runtime, "runtime", "lambda", "Runtime platform (lambda)")
	cmd.Flags().StringVar(&iac, "iac", "cdk", "Infrastructure as Code tool (cdk)")
	cmd.Flags().StringVar(&ci, "ci", "github", "CI/CD platform (github)")
	cmd.Flags().BoolVar(&force, "force", false, "Force initialization even if directory is not empty")

	return cmd
}

// checkDirectoryEmpty checks if directory is empty or doesn't exist
func checkDirectoryEmpty(dir string) error {
	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// Directory doesn't exist, that's fine
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check directory: %w", err)
	}

	// Directory exists, check if it's empty
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	if len(files) > 0 {
		return fmt.Errorf("directory '%s' is not empty. Use --force to initialize anyway", dir)
	}

	return nil
}
