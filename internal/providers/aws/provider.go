// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package aws

import (
	"context"

	"github.com/transire/transire/pkg/transire"
)

// Provider implements the AWS cloud provider
type Provider struct {
	region string
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
