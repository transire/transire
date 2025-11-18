package transire

import (
	"os"
	"testing"
)

func TestDetectRuntime(t *testing.T) {
	// Save original environment
	origLambda := os.Getenv("AWS_LAMBDA_FUNCTION_NAME")
	origK8s := os.Getenv("K_SERVICE")
	origAzure := os.Getenv("FUNCTIONS_WORKER_RUNTIME")
	origTransire := os.Getenv("TRANSIRE_RUNTIME")

	// Clean up after test
	defer func() {
		setEnvVar("AWS_LAMBDA_FUNCTION_NAME", origLambda)
		setEnvVar("K_SERVICE", origK8s)
		setEnvVar("FUNCTIONS_WORKER_RUNTIME", origAzure)
		setEnvVar("TRANSIRE_RUNTIME", origTransire)
	}()

	tests := []struct {
		name     string
		envVars  map[string]string
		expected RuntimeType
	}{
		{
			name:     "local development",
			envVars:  map[string]string{},
			expected: RuntimeLocal,
		},
		{
			name: "AWS Lambda",
			envVars: map[string]string{
				"AWS_LAMBDA_FUNCTION_NAME": "my-function",
			},
			expected: RuntimeAWSLambda,
		},
		{
			name: "Google Cloud Run",
			envVars: map[string]string{
				"K_SERVICE": "my-service",
			},
			expected: RuntimeGCPRun,
		},
		{
			name: "Azure Functions",
			envVars: map[string]string{
				"FUNCTIONS_WORKER_RUNTIME": "go",
			},
			expected: RuntimeAzureFunc,
		},
		{
			name: "explicit override",
			envVars: map[string]string{
				"TRANSIRE_RUNTIME":         "local",
				"AWS_LAMBDA_FUNCTION_NAME": "my-function", // Should be ignored
			},
			expected: RuntimeLocal,
		},
		{
			name: "AWS Lambda takes precedence over Cloud Run",
			envVars: map[string]string{
				"AWS_LAMBDA_FUNCTION_NAME": "my-function",
				"K_SERVICE":                "my-service",
			},
			expected: RuntimeAWSLambda,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			clearRuntimeEnv()

			// Set test environment
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			result := detectRuntime()
			if result != tt.expected {
				t.Errorf("detectRuntime() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func clearRuntimeEnv() {
	os.Unsetenv("AWS_LAMBDA_FUNCTION_NAME")
	os.Unsetenv("K_SERVICE")
	os.Unsetenv("FUNCTIONS_WORKER_RUNTIME")
	os.Unsetenv("TRANSIRE_RUNTIME")
}

func setEnvVar(key, value string) {
	if value == "" {
		os.Unsetenv(key)
	} else {
		os.Setenv(key, value)
	}
}
