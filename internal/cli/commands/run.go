package commands

import (
	"github.com/spf13/cobra"
	"github.com/transire/transire/internal/cli/runner"
)

// NewRunCommand creates the run command
func NewRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the application locally with hot reload",
		Long: `Run the Transire application in local development mode.

This command:
- Builds your Go application automatically
- Runs it as a subprocess
- Watches for file changes (*.go, *.yaml files)
- Automatically rebuilds and restarts on changes

Your application runs using the local Transire runtime with:
- HTTP server for web routes (default port 3000)
- Queue simulator for message processing
- Scheduler simulator for cron jobs

Examples:
  transire run
  transire run --config custom-transire.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get current working directory
			projectDir := "."

			// Create runner manager
			manager := runner.NewManager(projectDir)

			// Run with hot reload
			return manager.Run()
		},
	}

	return cmd
}
