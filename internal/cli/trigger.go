// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/transire/transire/internal/config"
	"github.com/transire/transire/internal/discover"
)

func newTriggerCmd() *cobra.Command {
	var manifestPath string
	var profile string
	var region string
	var env string
	cmd := &cobra.Command{
		Use:   "trigger <schedule>",
		Short: "Trigger an ad-hoc schedule handler",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sched := args[0]
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			mp := manifestPath
			if mp == "" {
				mp = filepath.Join(root, "transire.yaml")
			}
			m, err := config.LoadManifest(mp)
			if err != nil {
				return err
			}
			envCfg := resolveEnv(m, env, profile, region)
			layout, err := discover.Scan(root)
			if err != nil {
				return err
			}
			if !scheduleExists(layout, sched) {
				return fmt.Errorf("schedule %q not discovered in project", sched)
			}

			isLocalEnv := env == "" || strings.EqualFold(env, "local")
			if isLocalEnv {
				baseURL, err := ensureLocalRunning(cmd.Context())
				if err != nil {
					return err
				}
				return triggerLocalSchedule(cmd.Context(), baseURL, sched)
			}

			ctx := cmd.Context()
			cfg, err := awsConfig(ctx, envCfg.profile, envCfg.region)
			if err != nil {
				return err
			}
			cfn := cloudformation.NewFromConfig(cfg)
			outputs, err := fetchOutputs(ctx, cfn, stackName(m.App.Name))
			if err != nil {
				return err
			}
			lc := lambda.NewFromConfig(cfg)
			return triggerScheduleLambda(ctx, lc, outputs, sched)
		},
	}
	cmd.Flags().StringVar(&manifestPath, "manifest", "", "path to transire.yaml (defaults to ./transire.yaml)")
	cmd.Flags().StringVar(&profile, "profile", "transire-sandbox", "AWS profile to use")
	cmd.Flags().StringVar(&region, "region", "", "AWS region (defaults to manifest aws.region)")
	cmd.Flags().StringVar(&env, "env", "", "environment key from transire.yaml envs section")
	return cmd
}

func scheduleExists(layout discover.Layout, name string) bool {
	for _, s := range layout.Schedules {
		if s.Name == name {
			return true
		}
	}
	return false
}
