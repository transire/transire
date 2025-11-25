// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/transire/transire/internal/build"
	"github.com/transire/transire/internal/config"
	"github.com/transire/transire/internal/discover"
)

func newDeployCmd() *cobra.Command {
	var manifestPath string
	var profile string
	var env string
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy the current project (builds assets first)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(env) == "" {
				return fmt.Errorf("--env is required (no default)")
			}
			root, err := os.Getwd()
			if err != nil {
				return err
			}

			mp := manifestPath
			if mp == "" {
				mp = filepath.Join(root, "transire.yaml")
			}

			layout, err := discover.Scan(root)
			if err != nil {
				return err
			}

			m, err := config.LoadManifest(mp)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Building AWS assets...")
			if err := build.BuildAWS(context.Background(), root, m, layout); err != nil {
				return err
			}

			cdkDir := filepath.Join(root, "dist", "aws", "cdk")
			run := func(name string, args ...string) {
				c := exec.Command(name, args...)
				c.Dir = cdkDir
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				exitOnError(c.Run()) // reuse exitOnError to preserve existing behavior
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Installing CDK dependencies...")
			run("npm", "install")

			fmt.Fprintln(cmd.OutOrStdout(), "Deploying with CDK...")
			run("npx", "cdk", "deploy", "--require-approval", "never", "--profile", profile, fmt.Sprintf("--context env=%s", env))
			return nil
		},
	}
	cmd.Flags().StringVar(&manifestPath, "manifest", "", "path to transire.yaml (defaults to ./transire.yaml)")
	cmd.Flags().StringVar(&profile, "profile", "transire-sandbox", "AWS profile to use for deployment")
	cmd.Flags().StringVar(&env, "env", "", "environment name used for resource naming")
	return cmd
}
