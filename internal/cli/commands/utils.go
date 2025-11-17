package commands

import (
	"fmt"
	"os"

	"github.com/transire-org/transire/internal/providers/aws"
	"github.com/transire-org/transire/pkg/transire"
)

// createProvider creates the appropriate cloud provider
func createProvider(config *transire.Config) (transire.Provider, error) {
	switch config.Cloud {
	case "aws":
		// Default to us-east-1 if not specified
		region := "us-east-1"
		if awsRegion := os.Getenv("AWS_DEFAULT_REGION"); awsRegion != "" {
			region = awsRegion
		}
		return aws.NewProvider(region), nil
	default:
		return nil, fmt.Errorf("unsupported cloud provider: %s", config.Cloud)
	}
}

// createProviderWithRegion creates a provider with a specific region
func createProviderWithRegion(config *transire.Config, region string) (transire.Provider, error) {
	switch config.Cloud {
	case "aws":
		return aws.NewProvider(region), nil
	default:
		return nil, fmt.Errorf("unsupported cloud provider: %s", config.Cloud)
	}
}

// loadConfig loads the Transire configuration
func loadConfig(configPath string) (*transire.Config, error) {
	if configPath == "" {
		// Try default locations
		for _, path := range []string{"transire.yaml", "transire.yml"} {
			if _, err := os.Stat(path); err == nil {
				configPath = path
				break
			}
		}
	}

	if configPath == "" {
		// No config file found, use defaults
		config := &transire.Config{}
		config.SetDefaults()
		return config, nil
	}

	return transire.LoadConfig(configPath)
}