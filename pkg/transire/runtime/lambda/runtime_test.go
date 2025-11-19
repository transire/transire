//go:build lambda

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lambda

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

// TestAPIGatewayV2EventDetection tests detection of API Gateway HTTP API (v2) events
func TestAPIGatewayV2EventDetection(t *testing.T) {
	runtime := &Runtime{}

	tests := []struct {
		name     string
		event    string
		expected bool
	}{
		{
			name: "API Gateway v2 HTTP event should be detected",
			event: `{
				"version": "2.0",
				"routeKey": "$default",
				"rawPath": "/",
				"rawQueryString": "",
				"headers": {
					"accept": "*/*",
					"content-length": "0",
					"host": "example.execute-api.us-east-1.amazonaws.com"
				},
				"requestContext": {
					"accountId": "123456789012",
					"apiId": "api-id",
					"domainName": "example.execute-api.us-east-1.amazonaws.com",
					"http": {
						"method": "GET",
						"path": "/",
						"protocol": "HTTP/1.1",
						"sourceIp": "1.2.3.4",
						"userAgent": "curl/7.64.1"
					},
					"requestId": "request-id",
					"stage": "$default",
					"time": "12/Nov/2025:12:34:56 +0000",
					"timeEpoch": 1699791296123
				},
				"isBase64Encoded": false
			}`,
			expected: true,
		},
		{
			name: "API Gateway v1 REST event should be detected",
			event: `{
				"resource": "/",
				"path": "/",
				"httpMethod": "GET",
				"headers": {
					"Accept": "*/*"
				},
				"requestContext": {
					"accountId": "123456789012",
					"apiId": "api-id",
					"protocol": "HTTP/1.1",
					"httpMethod": "GET",
					"path": "/",
					"stage": "prod"
				}
			}`,
			expected: true,
		},
		{
			name: "SQS event should not be detected as API Gateway",
			event: `{
				"Records": [{
					"messageId": "msg-123",
					"body": "test message",
					"eventSource": "aws:sqs"
				}]
			}`,
			expected: false,
		},
		{
			name: "EventBridge event should not be detected as API Gateway",
			event: `{
				"version": "0",
				"id": "event-id",
				"detail-type": "Scheduled Event",
				"source": "aws.events",
				"account": "123456789012",
				"time": "2025-11-17T12:00:00Z",
				"region": "us-east-1",
				"resources": ["arn:aws:events:us-east-1:123456789012:rule/my-rule"],
				"detail": {}
			}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runtime.isAPIGatewayEvent(json.RawMessage(tt.event))
			if result != tt.expected {
				t.Errorf("isAPIGatewayEvent() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestSQSEventDetection tests detection of SQS events
func TestSQSEventDetection(t *testing.T) {
	runtime := &Runtime{}

	tests := []struct {
		name     string
		event    string
		expected bool
	}{
		{
			name: "SQS event should be detected",
			event: `{
				"Records": [{
					"messageId": "msg-123",
					"receiptHandle": "receipt-handle",
					"body": "test message",
					"attributes": {
						"ApproximateReceiveCount": "1"
					},
					"messageAttributes": {},
					"md5OfBody": "md5-hash",
					"eventSource": "aws:sqs",
					"eventSourceARN": "arn:aws:sqs:us-east-1:123456789012:my-queue",
					"awsRegion": "us-east-1"
				}]
			}`,
			expected: true,
		},
		{
			name: "API Gateway event should not be detected as SQS",
			event: `{
				"version": "2.0",
				"routeKey": "$default",
				"requestContext": {
					"http": {
						"method": "GET"
					}
				}
			}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runtime.isSQSEvent(json.RawMessage(tt.event))
			if result != tt.expected {
				t.Errorf("isSQSEvent() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestEventBridgeEventDetection tests detection of EventBridge events
func TestEventBridgeEventDetection(t *testing.T) {
	runtime := &Runtime{}

	tests := []struct {
		name     string
		event    string
		expected bool
	}{
		{
			name: "EventBridge scheduled event should be detected",
			event: `{
				"version": "0",
				"id": "event-id",
				"detail-type": "Scheduled Event",
				"source": "aws.events",
				"account": "123456789012",
				"time": "2025-11-17T12:00:00Z",
				"region": "us-east-1",
				"resources": ["arn:aws:events:us-east-1:123456789012:rule/my-rule"],
				"detail": {}
			}`,
			expected: true,
		},
		{
			name: "API Gateway event should not be detected as EventBridge",
			event: `{
				"version": "2.0",
				"requestContext": {
					"http": {
						"method": "GET"
					}
				}
			}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runtime.isEventBridgeEvent(json.RawMessage(tt.event))
			if result != tt.expected {
				t.Errorf("isEventBridgeEvent() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestQueueNameToHandlerName tests conversion of CloudFormation queue names to handler names
func TestQueueNameToHandlerName(t *testing.T) {
	runtime := &Runtime{}

	tests := []struct {
		name          string
		fullQueueName string
		appName       string
		environment   string
		expectedName  string
	}{
		{
			name:          "should strip simple app-env prefix",
			fullQueueName: "myapp-dev-email-queue",
			appName:       "myapp",
			environment:   "dev",
			expectedName:  "email-queue",
		},
		{
			name:          "should handle app name with hyphens",
			fullQueueName: "simple-api-dev-notification-queue",
			appName:       "simple-api",
			environment:   "dev",
			expectedName:  "notification-queue",
		},
		{
			name:          "should handle multi-hyphen app names",
			fullQueueName: "my-cool-app-production-user-events-queue",
			appName:       "my-cool-app",
			environment:   "production",
			expectedName:  "user-events-queue",
		},
		{
			name:          "should return as-is when no env vars set",
			fullQueueName: "some-queue-name",
			appName:       "",
			environment:   "",
			expectedName:  "some-queue-name",
		},
		{
			name:          "should return as-is when prefix doesn't match",
			fullQueueName: "different-prefix-email-queue",
			appName:       "myapp",
			environment:   "dev",
			expectedName:  "different-prefix-email-queue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for this test
			if tt.appName != "" {
				os.Setenv("TRANSIRE_APP_NAME", tt.appName)
				defer os.Unsetenv("TRANSIRE_APP_NAME")
			}
			if tt.environment != "" {
				os.Setenv("TRANSIRE_ENVIRONMENT", tt.environment)
				defer os.Unsetenv("TRANSIRE_ENVIRONMENT")
			}

			result := runtime.queueNameToHandlerName(tt.fullQueueName)
			if result != tt.expectedName {
				t.Errorf("queueNameToHandlerName() = %v, want %v", result, tt.expectedName)
			}
		})
	}
}

// TestRuleNameToHandlerName tests conversion of CloudFormation rule names to handler names
func TestRuleNameToHandlerName(t *testing.T) {
	runtime := &Runtime{}

	tests := []struct {
		name         string
		ruleName     string
		appName      string
		environment  string
		expectedName string
	}{
		{
			name:         "should strip simple app-env prefix from rule",
			ruleName:     "myapp-dev-daily-cleanup",
			appName:      "myapp",
			environment:  "dev",
			expectedName: "daily-cleanup",
		},
		{
			name:         "should handle app name with hyphens in rule",
			ruleName:     "simple-api-dev-user-reminder",
			appName:      "simple-api",
			environment:  "dev",
			expectedName: "user-reminder",
		},
		{
			name:         "should handle multi-hyphen app names in rules",
			ruleName:     "my-cool-app-production-hourly-sync",
			appName:      "my-cool-app",
			environment:  "production",
			expectedName: "hourly-sync",
		},
		{
			name:         "should return as-is when no env vars set",
			ruleName:     "some-rule-name",
			appName:      "",
			environment:  "",
			expectedName: "some-rule-name",
		},
		{
			name:         "should return as-is when prefix doesn't match",
			ruleName:     "different-prefix-daily-task",
			appName:      "myapp",
			environment:  "dev",
			expectedName: "different-prefix-daily-task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for this test
			if tt.appName != "" {
				os.Setenv("TRANSIRE_APP_NAME", tt.appName)
				defer os.Unsetenv("TRANSIRE_APP_NAME")
			}
			if tt.environment != "" {
				os.Setenv("TRANSIRE_ENVIRONMENT", tt.environment)
				defer os.Unsetenv("TRANSIRE_ENVIRONMENT")
			}

			result := runtime.ruleNameToHandlerName(tt.ruleName)
			if result != tt.expectedName {
				t.Errorf("ruleNameToHandlerName() = %v, want %v", result, tt.expectedName)
			}
		})
	}
}

// TestExtractQueueName tests extraction of queue names from SQS event source ARNs
func TestExtractQueueName(t *testing.T) {
	runtime := &Runtime{}

	// Set env vars for name conversion
	os.Setenv("TRANSIRE_APP_NAME", "myapp")
	os.Setenv("TRANSIRE_ENVIRONMENT", "dev")
	defer os.Unsetenv("TRANSIRE_APP_NAME")
	defer os.Unsetenv("TRANSIRE_ENVIRONMENT")

	tests := []struct {
		name         string
		arn          string
		expectedName string
	}{
		{
			name:         "should extract and convert CloudFormation queue name",
			arn:          "arn:aws:sqs:us-east-1:123456789012:myapp-dev-email-queue",
			expectedName: "email-queue",
		},
		{
			name:         "should handle queue names with multiple hyphens",
			arn:          "arn:aws:sqs:us-east-1:123456789012:myapp-dev-user-notification-queue",
			expectedName: "user-notification-queue",
		},
		{
			name:         "should handle FIFO queue names",
			arn:          "arn:aws:sqs:us-east-1:123456789012:myapp-dev-my-queue.fifo",
			expectedName: "my-queue.fifo",
		},
		{
			name:         "should return unknown for invalid ARN",
			arn:          "invalid-arn",
			expectedName: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runtime.extractQueueName(tt.arn)
			if result != tt.expectedName {
				t.Errorf("extractQueueName() = %v, want %v", result, tt.expectedName)
			}
		})
	}
}

// TestExtractScheduleName tests extraction of schedule names from EventBridge events
func TestExtractScheduleName(t *testing.T) {
	runtime := &Runtime{}

	// Set env vars for name conversion
	os.Setenv("TRANSIRE_APP_NAME", "myapp")
	os.Setenv("TRANSIRE_ENVIRONMENT", "dev")
	defer os.Unsetenv("TRANSIRE_APP_NAME")
	defer os.Unsetenv("TRANSIRE_ENVIRONMENT")

	tests := []struct {
		name         string
		event        events.CloudWatchEvent
		expectedName string
	}{
		{
			name: "should extract and convert CloudFormation rule name",
			event: events.CloudWatchEvent{
				Resources: []string{
					"arn:aws:events:us-east-1:123456789012:rule/myapp-dev-daily-cleanup",
				},
			},
			expectedName: "daily-cleanup",
		},
		{
			name: "should handle rules with multiple hyphens",
			event: events.CloudWatchEvent{
				Resources: []string{
					"arn:aws:events:us-east-1:123456789012:rule/myapp-dev-hourly-user-sync",
				},
			},
			expectedName: "hourly-user-sync",
		},
		{
			name: "should fallback to detail type when no resources",
			event: events.CloudWatchEvent{
				Resources:  []string{},
				DetailType: "Scheduled Event",
			},
			expectedName: "Scheduled Event",
		},
		{
			name:         "should return unknown for empty event",
			event:        events.CloudWatchEvent{},
			expectedName: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runtime.extractScheduleName(tt.event)
			if result != tt.expectedName {
				t.Errorf("extractScheduleName() = %v, want %v", result, tt.expectedName)
			}
		})
	}
}
