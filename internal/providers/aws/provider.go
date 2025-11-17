package aws

import (
	"context"
	"fmt"

	"github.com/transire-org/transire/pkg/transire"
)

// Provider implements the AWS cloud provider
type Provider struct {
	region    string
	accountID string
}

// NewProvider creates a new AWS provider
func NewProvider(region string) *Provider {
	return &Provider{
		region: region,
	}
}

// Name returns the provider identifier
func (p *Provider) Name() string {
	return "aws"
}

// Runtime returns the supported runtime
func (p *Provider) Runtime() string {
	return "lambda"
}

// BuildArtifacts creates deployable artifacts for AWS Lambda
func (p *Provider) BuildArtifacts(ctx context.Context, config transire.BuildConfig) error {
	builder := NewLambdaBuilder(config)
	return builder.Build(ctx)
}

// GenerateIaC creates CDK infrastructure definitions
func (p *Provider) GenerateIaC(ctx context.Context, config transire.IaCConfig) error {
	generator := NewCDKGenerator(p.region)
	return generator.Generate(ctx, config)
}

// Deploy applies infrastructure and artifacts using CDK
func (p *Provider) Deploy(ctx context.Context, config transire.DeployConfig) error {
	deployer := NewCDKDeployer(p.region)
	return deployer.Deploy(ctx, config)
}

// CreateRuntime returns a runtime implementation for AWS
func (p *Provider) CreateRuntime(ctx context.Context, config transire.RuntimeConfig) (transire.Runtime, error) {
	switch config.Runtime {
	case "lambda":
		return NewLambdaRuntime(), nil
	case "ecs":
		return nil, fmt.Errorf("ECS runtime not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported AWS runtime: %s", config.Runtime)
	}
}