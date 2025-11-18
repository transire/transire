// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package commands

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/transire/transire/pkg/transire"
)

// NewDeployCommand creates the deploy command
func NewDeployCommand() *cobra.Command {
	var (
		configPath  = ""
		environment = ""
		dryRun      = false
		region      = ""
	)

	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy to cloud provider",
		Long: `Deploy your Transire application to the configured cloud provider.

This command:
- Applies the generated Infrastructure as Code
- Deploys the built artifacts
- Configures cloud resources (Lambda, API Gateway, SQS, EventBridge)
- Sets up IAM permissions and networking

Make sure to run 'transire build' before deploying.

Examples:
  transire deploy
  transire deploy --environment production
  transire deploy --dry-run  # Preview changes without applying
  transire deploy --region us-west-2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			config, err := loadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Override environment if specified
			if environment != "" {
				// Could modify config based on environment
				log.Printf("🌍 Deploying to environment: %s", environment)
			}

			// Override region if specified
			deployRegion := detectDeployRegion(region)

			log.Printf("🚀 Deploying Transire application: %s", config.Name)
			log.Printf("🌎 Target: %s/%s in %s", config.Cloud, config.Runtime, deployRegion)

			// Create provider
			provider, err := createProviderWithRegion(config, deployRegion)
			if err != nil {
				return fmt.Errorf("failed to create provider: %w", err)
			}

			ctx := context.Background()

			// Deploy
			deployConfig := transire.DeployConfig{
				StackName:   config.Name + "-stack",
				Region:      deployRegion,
				Environment: environment,
				DryRun:      dryRun,
			}

			if dryRun {
				log.Printf("🔍 Dry run mode - previewing changes without applying")
			}

			if err := provider.Deploy(ctx, deployConfig); err != nil {
				return fmt.Errorf("failed to deploy: %w", err)
			}

			if !dryRun {
				log.Printf("✅ Deployment completed successfully")
				log.Printf("🎯 Stack: %s", deployConfig.StackName)
				log.Printf("🌍 Region: %s", deployConfig.Region)
			}

			return nil
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to configuration file")
	cmd.Flags().StringVarP(&environment, "environment", "e", "", "Deployment environment (dev, staging, prod)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without applying them")
	cmd.Flags().StringVarP(&region, "region", "r", "", "AWS region (overrides AWS_DEFAULT_REGION)")

	return cmd
}

// detectDeployRegion detects the AWS region to use for deployment
// Following AWS SDK's region resolution order:
// 1. Explicit region flag (--region)
// 2. AWS_REGION environment variable
// 3. AWS_DEFAULT_REGION environment variable
// 4. Default to us-east-1
func detectDeployRegion(regionFlag string) string {
	// Override region if specified via flag
	if regionFlag != "" {
		return regionFlag
	}

	// AWS SDK checks AWS_REGION first, then AWS_DEFAULT_REGION
	if region := os.Getenv("AWS_REGION"); region != "" {
		return region
	}

	if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
		return region
	}

	// Default fallback
	return "us-east-1"
}
