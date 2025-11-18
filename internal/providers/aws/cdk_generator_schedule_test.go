package aws

import (
	"testing"
)

// TestScheduleExpressionConversion tests that cron expressions are properly formatted for EventBridge
func TestScheduleExpressionConversion(t *testing.T) {
	tests := []struct {
		name          string
		input         string // User's cron expression from transire.yaml
		expected      string // What should be generated in CDK
		expectedError bool
	}{
		{
			name:     "5-field cron should be converted to 6-field",
			input:    "0 * * * *",
			expected: "cron(0 * * * ? *)",
		},
		{
			name:     "daily at 3am",
			input:    "0 3 * * *",
			expected: "cron(0 3 * * ? *)",
		},
		{
			name:     "daily at 9am",
			input:    "0 9 * * *",
			expected: "cron(0 9 * * ? *)",
		},
		{
			name:     "already 6-field with ? for day-of-week",
			input:    "0 12 * * ? *",
			expected: "cron(0 12 * * ? *)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertCronToEventBridge(tt.input)
			if got != tt.expected {
				t.Errorf("convertCronToEventBridge(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
