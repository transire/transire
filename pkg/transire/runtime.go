package transire

import (
	"os"
)

// RuntimeType represents the execution environment
type RuntimeType string

const (
	RuntimeLocal     RuntimeType = "local"
	RuntimeAWSLambda RuntimeType = "aws_lambda"
	RuntimeGCPRun    RuntimeType = "gcp_cloudrun"   // Future
	RuntimeAzureFunc RuntimeType = "azure_function" // Future
)

// detectRuntime determines current execution environment
func detectRuntime() RuntimeType {
	// Check for explicit Transire environment override first
	if env := os.Getenv("TRANSIRE_RUNTIME"); env != "" {
		return RuntimeType(env)
	}

	// Check for AWS Lambda environment
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		return RuntimeAWSLambda
	}

	// Check for Google Cloud Run environment
	if os.Getenv("K_SERVICE") != "" {
		return RuntimeGCPRun
	}

	// Check for Azure Functions environment
	if os.Getenv("FUNCTIONS_WORKER_RUNTIME") != "" {
		return RuntimeAzureFunc
	}

	// Default to local development
	return RuntimeLocal
}
