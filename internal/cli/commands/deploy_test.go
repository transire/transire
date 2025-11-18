package commands

import (
	"os"
	"testing"
)

// TestDeployRegionDetection tests that the deploy command detects and displays
// the actual AWS region that will be used by the AWS SDK, not just AWS_DEFAULT_REGION
func TestDeployRegionDetection(t *testing.T) {
	tests := []struct {
		name           string
		regionFlag     string
		awsDefaultReg  string
		awsRegion      string
		awsProfile     string
		expectedRegion string
	}{
		{
			name:           "uses flag when provided",
			regionFlag:     "us-west-2",
			awsDefaultReg:  "us-east-1",
			awsRegion:      "",
			expectedRegion: "us-west-2",
		},
		{
			name:           "uses AWS_REGION over AWS_DEFAULT_REGION",
			regionFlag:     "",
			awsDefaultReg:  "us-east-1",
			awsRegion:      "eu-west-2",
			expectedRegion: "eu-west-2",
		},
		{
			name:           "uses AWS_DEFAULT_REGION when AWS_REGION not set",
			regionFlag:     "",
			awsDefaultReg:  "ap-southeast-1",
			awsRegion:      "",
			expectedRegion: "ap-southeast-1",
		},
		{
			name:           "defaults to us-east-1 when no region env vars",
			regionFlag:     "",
			awsDefaultReg:  "",
			awsRegion:      "",
			expectedRegion: "us-east-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env vars
			oldDefaultRegion := os.Getenv("AWS_DEFAULT_REGION")
			oldRegion := os.Getenv("AWS_REGION")
			defer func() {
				os.Setenv("AWS_DEFAULT_REGION", oldDefaultRegion)
				os.Setenv("AWS_REGION", oldRegion)
			}()

			// Set up test environment
			if tt.awsDefaultReg != "" {
				os.Setenv("AWS_DEFAULT_REGION", tt.awsDefaultReg)
			} else {
				os.Unsetenv("AWS_DEFAULT_REGION")
			}

			if tt.awsRegion != "" {
				os.Setenv("AWS_REGION", tt.awsRegion)
			} else {
				os.Unsetenv("AWS_REGION")
			}

			// Test the region detection logic
			detectedRegion := detectDeployRegion(tt.regionFlag)

			if detectedRegion != tt.expectedRegion {
				t.Errorf("detectDeployRegion() = %v, want %v", detectedRegion, tt.expectedRegion)
			}
		})
	}
}
