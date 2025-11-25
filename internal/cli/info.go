// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/transire/transire/internal/config"
	"github.com/transire/transire/internal/discover"
)

func newInfoCmd() *cobra.Command {
	var manifestPath string
	var env string
	var profile string
	var region string
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show discovered handlers, and optionally stack outputs for an env",
		Run: func(cmd *cobra.Command, args []string) {
			root, err := os.Getwd()
			exitOnError(err)

			layout, err := discover.Scan(root)
			exitOnError(err)

			mp := manifestPath
			if mp == "" {
				mp = filepath.Join(root, "transire.yaml")
			}
			m, err := config.LoadManifest(mp)
			exitOnError(err)

			fmt.Fprintf(cmd.OutOrStdout(), "App: %s\n", m.App.Name)

			if len(layout.Queues) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Queues: none discovered")
			} else {
				var names []string
				for _, q := range layout.Queues {
					names = append(names, q.Name)
				}
				sort.Strings(names)
				fmt.Fprintf(cmd.OutOrStdout(), "Queues (%d): %s\n", len(names), strings.Join(names, ", "))
			}

			if len(layout.Schedules) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Schedules: none discovered")
			} else {
				var rows []string
				for _, s := range layout.Schedules {
					rows = append(rows, fmt.Sprintf("%s every %s", s.Name, humanDuration(s.Every)))
				}
				sort.Strings(rows)
				fmt.Fprintf(cmd.OutOrStdout(), "Schedules (%d): %s\n", len(rows), strings.Join(rows, "; "))
			}

			if env != "" {
				settings := resolveEnv(m, env, profile, region)
				cfg, err := awsConfig(cmd.Context(), settings.profile, settings.region)
				exitOnError(err)
				cfn := cloudformation.NewFromConfig(cfg)
				outputs, err := fetchOutputs(cmd.Context(), cfn, stackName(m.App.Name))
				if err != nil {
					exitOnError(err)
				}
				if len(outputs) > 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "Stack outputs:")
					keys := make([]string, 0, len(outputs))
					for k := range outputs {
						keys = append(keys, k)
					}
					sort.Strings(keys)
					for _, k := range keys {
						fmt.Fprintf(cmd.OutOrStdout(), "  %s=%s\n", k, outputs[k])
					}
				}
			}
		},
	}
	cmd.Flags().StringVar(&manifestPath, "manifest", "", "path to transire.yaml (defaults to ./transire.yaml)")
	cmd.Flags().StringVar(&env, "env", "", "environment key from transire.yaml envs section to fetch outputs")
	cmd.Flags().StringVar(&profile, "profile", "transire-sandbox", "AWS profile to use when fetching outputs")
	cmd.Flags().StringVar(&region, "region", "", "AWS region (defaults to manifest aws.region)")
	return cmd
}

func humanDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	if d%time.Hour == 0 {
		return fmt.Sprintf("%dh", int(d/time.Hour))
	}
	if d%time.Minute == 0 {
		return fmt.Sprintf("%dm", int(d/time.Minute))
	}
	return d.String()
}
