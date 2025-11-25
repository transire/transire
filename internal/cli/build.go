// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/transire/transire/internal/build"
	"github.com/transire/transire/internal/config"
	"github.com/transire/transire/internal/discover"
)

func newBuildCmd() *cobra.Command {
	var manifestPath string
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build the Lambda asset and infrastructure templates",
		Run: func(cmd *cobra.Command, args []string) {
			root, err := os.Getwd()
			exitOnError(err)

			mp := manifestPath
			if mp == "" {
				mp = filepath.Join(root, "transire.yaml")
			}

			layout, err := discover.Scan(root)
			exitOnError(err)

			m, err := config.LoadManifest(mp)
			exitOnError(err)

			fmt.Fprintln(cmd.OutOrStdout(), "Building AWS assets...")
			exitOnError(build.BuildAWS(context.Background(), root, m, layout))
			fmt.Fprintln(cmd.OutOrStdout(), "Build complete: dist/aws contains bootstrap.zip and CDK app")
		},
	}
	cmd.Flags().StringVar(&manifestPath, "manifest", "", "path to transire.yaml (defaults to ./transire.yaml)")
	return cmd
}
