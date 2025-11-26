// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cli

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/transire/transire/internal/config"
	"github.com/transire/transire/internal/discover"
)

func newSendCmd() *cobra.Command {
	var manifestPath string
	var base64Data bool
	var profile string
	var region string
	var env string
	cmd := &cobra.Command{
		Use:   "send <queue> <message>",
		Short: "Send an ad-hoc message to a discovered queue handler",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			queue := args[0]
			data := args[1]
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
			if !queueExists(layout, queue) {
				return fmt.Errorf("queue %q not discovered in project", queue)
			}
			payload := []byte(data)
			if base64Data {
				decoded, err := base64.StdEncoding.DecodeString(data)
				if err != nil {
					return fmt.Errorf("invalid base64 payload: %w", err)
				}
				payload = decoded
			}

			isLocalEnv := env == "" || strings.EqualFold(env, "local")

			if isLocalEnv {
				baseURL, err := ensureLocalRunning(cmd.Context())
				if err != nil {
					return err
				}
				return sendLocalQueue(cmd.Context(), baseURL, queue, payload)
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
			sqsClient := sqs.NewFromConfig(cfg)
			return sendAWSQueue(ctx, sqsClient, outputs, queue, payload)
		},
	}
	cmd.Flags().StringVar(&manifestPath, "manifest", "", "path to transire.yaml (defaults to ./transire.yaml)")
	cmd.Flags().BoolVar(&base64Data, "base64", false, "interpret message as base64")
	cmd.Flags().StringVar(&profile, "profile", "transire-sandbox", "AWS profile to use")
	cmd.Flags().StringVar(&region, "region", "", "AWS region (overrides AWS SDK defaults when set)")
	cmd.Flags().StringVar(&env, "env", "", "environment key from transire.yaml envs section")
	return cmd
}

func queueExists(layout discover.Layout, name string) bool {
	for _, q := range layout.Queues {
		if q.Name == name {
			return true
		}
	}
	return false
}
