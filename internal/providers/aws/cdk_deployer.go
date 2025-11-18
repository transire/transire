// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/transire/transire/pkg/transire"
)

// CDKDeployer handles deployment using AWS CDK
type CDKDeployer struct {
	region string
}

// NewCDKDeployer creates a new CDK deployer
func NewCDKDeployer(region string) *CDKDeployer {
	return &CDKDeployer{
		region: region,
	}
}

// Deploy applies infrastructure and artifacts using CDK
func (d *CDKDeployer) Deploy(ctx context.Context, config transire.DeployConfig) error {
	infraDir := "infrastructure"

	// Install CDK dependencies
	if err := d.installDependencies(ctx, infraDir); err != nil {
		return fmt.Errorf("failed to install dependencies: %w", err)
	}

	// Build TypeScript
	if err := d.buildTypeScript(ctx, infraDir); err != nil {
		return fmt.Errorf("failed to build TypeScript: %w", err)
	}

	// Deploy with CDK
	if err := d.deployCDK(ctx, infraDir, config); err != nil {
		return fmt.Errorf("failed to deploy CDK: %w", err)
	}

	return nil
}

// installDependencies installs npm dependencies
func (d *CDKDeployer) installDependencies(ctx context.Context, infraDir string) error {
	cmd := exec.CommandContext(ctx, "npm", "install")
	cmd.Dir = infraDir
	cmd.Env = cmd.Environ() // Inherit environment

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("npm install failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// buildTypeScript compiles TypeScript to JavaScript
func (d *CDKDeployer) buildTypeScript(ctx context.Context, infraDir string) error {
	cmd := exec.CommandContext(ctx, "npm", "run", "build")
	cmd.Dir = infraDir
	cmd.Env = cmd.Environ() // Inherit environment

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("TypeScript build failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// deployCDK runs cdk deploy
func (d *CDKDeployer) deployCDK(ctx context.Context, infraDir string, config transire.DeployConfig) error {
	// Get AWS account ID
	accountID, err := d.getAWSAccountID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get AWS account ID: %w", err)
	}

	args := []string{"deploy"}

	if config.DryRun {
		args = []string{"diff"}
	}

	// Add stack name
	args = append(args, config.StackName)

	// Add require approval flag for CI/CD
	if !config.DryRun {
		args = append(args, "--require-approval", "never")
	}

	cmd := exec.CommandContext(ctx, "npx", append([]string{"cdk"}, args...)...)
	cmd.Dir = infraDir

	// Inherit all environment variables (includes AWS_PROFILE, AWS credentials, etc.)
	cmd.Env = os.Environ()

	// Add CDK-specific environment variables
	cmd.Env = append(cmd.Env, fmt.Sprintf("CDK_DEFAULT_REGION=%s", config.Region))
	cmd.Env = append(cmd.Env, fmt.Sprintf("CDK_DEFAULT_ACCOUNT=%s", accountID))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("CDK deploy failed: %w\nOutput: %s", err, string(output))
	}

	fmt.Printf("CDK deploy output:\n%s\n", string(output))
	return nil
}

// getAWSAccountID retrieves the AWS account ID using STS
func (d *CDKDeployer) getAWSAccountID(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "aws", "sts", "get-caller-identity", "--query", "Account", "--output", "text")
	cmd.Env = cmd.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try with JSON output as fallback
		cmd = exec.CommandContext(ctx, "aws", "sts", "get-caller-identity")
		cmd.Env = cmd.Environ()
		output, err = cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to get caller identity: %w\nOutput: %s", err, string(output))
		}

		var identity struct {
			Account string `json:"Account"`
		}
		if err := json.Unmarshal(output, &identity); err != nil {
			return "", fmt.Errorf("failed to parse caller identity: %w", err)
		}
		return identity.Account, nil
	}

	// Trim whitespace from text output
	accountID := string(output)
	accountID = accountID[:len(accountID)-1] // Remove newline
	return accountID, nil
}
