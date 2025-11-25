// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

// Execute runs the CLI.
func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transire",
		Short: "Transire CLI for building and deploying cloud-agnostic Go apps",
	}
	cmd.PersistentFlags().Bool("verbose", false, "enable verbose output")

	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newRunCmd())
	cmd.AddCommand(newBuildCmd())
	cmd.AddCommand(newDeployCmd())
	cmd.AddCommand(newInfoCmd())
	cmd.AddCommand(newSendCmd())
	cmd.AddCommand(newTriggerCmd())
	cmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	})
	return cmd
}

func exitOnError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
