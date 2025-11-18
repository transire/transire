// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/transire/transire/internal/cli/commands"
)

var version = "dev" // Set by build system

func main() {
	root := &cobra.Command{
		Use:   "transire",
		Short: "Cloud-agnostic application runtime",
		Long: `Transire enables developers to write cloud-agnostic applications
that run consistently across local development and cloud deployments.

Write your application using familiar patterns (Chi routing, standard Go),
and Transire handles the runtime abstraction and cloud deployment.`,
		Version: version,
	}

	// Add global flags
	root.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	root.PersistentFlags().StringP("config", "c", "transire.yaml", "Path to configuration file")

	// Add commands
	root.AddCommand(commands.NewInitCommand())
	root.AddCommand(commands.NewRunCommand())
	root.AddCommand(commands.NewBuildCommand())
	root.AddCommand(commands.NewDeployCommand())
	root.AddCommand(commands.NewDevCommand())

	// Execute root command
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
