// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/transire/transire/internal/scaffold"
)

func newInitCmd() *cobra.Command {
	var module string
	cmd := &cobra.Command{
		Use:   "init [path]",
		Short: "Create a new Transire app",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			target := "."
			if len(args) == 1 {
				target = args[0]
			}

			if err := os.MkdirAll(target, 0o755); err != nil {
				exitOnError(err)
			}

			abs, err := filepath.Abs(target)
			if err != nil {
				exitOnError(err)
			}

			if err := scaffold.Generate(abs, module); err != nil {
				exitOnError(err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Created Transire app in %s\n", abs)
		},
	}
	cmd.Flags().StringVar(&module, "module", "", "module path to use in go.mod (defaults to directory name)")
	return cmd
}
