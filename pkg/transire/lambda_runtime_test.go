package transire

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

// TestAPIGatewayV2EventDetection tests detection of API Gateway HTTP API (v2) events
func TestAPIGatewayV2EventDetection(t *testing.T) {
	runtime := &lambdaRuntime{}

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
	runtime := &lambdaRuntime{}

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
	runtime := &lambdaRuntime{}

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

// TestExtractScheduleName tests extraction of schedule handler names from EventBridge events
func TestExtractScheduleName(t *testing.T) {
	runtime := &lambdaRuntime{}

	tests := []struct {
		name         string
		event        events.CloudWatchEvent
		expectedName string
		description  string
	}{
		{
			name: "should extract handler name from rule with standard naming",
			event: events.CloudWatchEvent{
				Resources: []string{
					"arn:aws:events:us-east-1:123456789012:rule/cleanup-completed-todos",
				},
			},
			expectedName: "cleanup-completed-todos",
			description:  "Simple rule name without CloudFormation suffix",
		},
		{
			name: "should extract handler name from CloudFormation-generated rule name",
			event: events.CloudWatchEvent{
				Resources: []string{
					"arn:aws:events:eu-west-2:123456789012:rule/todo-app-stack-CleanupCompletedTodosRuleF0BFE00D-F30EO7W5SEws",
				},
			},
			expectedName: "cleanup-completed-todos",
			description:  "Should extract logical handler name from CDK-generated rule name",
		},
		{
			name: "should extract handler name from rule with hyphenated prefix",
			event: events.CloudWatchEvent{
				Resources: []string{
					"arn:aws:events:us-east-1:123456789012:rule/my-app-stack-DailyTodoSummaryRule123ABC-XYZABC123",
				},
			},
			expectedName: "daily-todo-summary",
			description:  "Should handle different CloudFormation naming patterns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runtime.extractScheduleName(tt.event)
			if result != tt.expectedName {
				t.Errorf("extractScheduleName() = %v, want %v (%s)", result, tt.expectedName, tt.description)
			}
		})
	}
}

// TestExtractQueueName tests extraction of queue names from SQS event source ARNs
func TestExtractQueueName(t *testing.T) {
	runtime := &lambdaRuntime{}

	tests := []struct {
		name         string
		arn          string
		expectedName string
	}{
		{
			name:         "should extract simple queue name",
			arn:          "arn:aws:sqs:us-east-1:123456789012:my-queue",
			expectedName: "my-queue",
		},
		{
			name:         "should extract queue name with hyphens",
			arn:          "arn:aws:sqs:us-east-1:123456789012:todo-reminders",
			expectedName: "todo-reminders",
		},
		{
			name:         "should handle FIFO queue names",
			arn:          "arn:aws:sqs:us-east-1:123456789012:my-queue.fifo",
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
