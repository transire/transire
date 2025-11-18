package aws

import (
	"context"

	"github.com/transire-org/transire/pkg/transire"
)

// LambdaRuntime wraps the core Lambda runtime with AWS-specific functionality
type LambdaRuntime struct {
	coreRuntime transire.Runtime
}

// NewLambdaRuntime creates a new AWS Lambda runtime
func NewLambdaRuntime() transire.Runtime {
	// Use the core lambda runtime from the main package
	return &LambdaRuntime{
		coreRuntime: newCoreRuntime(),
	}
}

// Start begins processing in the Lambda environment
func (r *LambdaRuntime) Start(ctx context.Context, app *transire.App) error {
	// Could add AWS-specific initialization here
	return r.coreRuntime.Start(ctx, app)
}

// Stop gracefully shuts down the runtime
func (r *LambdaRuntime) Stop(ctx context.Context) error {
	return r.coreRuntime.Stop(ctx)
}

// IsLocal returns false since this is the Lambda runtime
func (r *LambdaRuntime) IsLocal() bool {
	return false
}

// newCoreRuntime creates the core lambda runtime
// This is a bridge to the main package's lambda runtime
func newCoreRuntime() transire.Runtime {
	// For now, we'll create this directly
	// In practice, this would use the factory from the main package
	return &lambdaRuntimeBridge{}
}

// lambdaRuntimeBridge implements the Runtime interface as a bridge
type lambdaRuntimeBridge struct{}

func (r *lambdaRuntimeBridge) Start(ctx context.Context, app *transire.App) error {
	// This would delegate to the actual lambda runtime implementation
	// For now, just return an error indicating this needs the actual runtime
	panic("Lambda runtime bridge not fully implemented - use transire.NewLambdaRuntime() instead")
}

func (r *lambdaRuntimeBridge) Stop(ctx context.Context) error {
	return nil
}

func (r *lambdaRuntimeBridge) IsLocal() bool {
	return false
}
