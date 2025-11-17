package commands

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/transire-org/transire/pkg/transire"
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
			deployRegion := region
			if deployRegion == "" {
				// Use AWS_DEFAULT_REGION or fallback
				deployRegion = os.Getenv("AWS_DEFAULT_REGION")
				if deployRegion == "" {
					deployRegion = "us-east-1"
				}
			}

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

