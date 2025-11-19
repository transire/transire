//go:build lambda

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lambda

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/transire/transire/pkg/transire"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// Runtime implements the transire.Runtime interface for AWS Lambda
type Runtime struct {
	app *transire.App
}

// NewLambdaRuntime creates a new Lambda runtime
func NewLambdaRuntime() *Runtime {
	return &Runtime{}
}

// Start begins processing in the Lambda environment
func (r *Runtime) Start(ctx context.Context, app *transire.App) error {
	r.app = app

	// Start the AWS Lambda handler
	lambda.Start(r.handleLambdaEvent)
	return nil
}

// Stop is not applicable for Lambda runtime
func (r *Runtime) Stop(ctx context.Context) error {
	// Lambda runtime doesn't need explicit stopping
	return nil
}

// IsLocal returns false since this is the Lambda runtime
func (r *Runtime) IsLocal() bool {
	return false
}

// CreateQueueProducer returns a queue producer for AWS Lambda
func (r *Runtime) CreateQueueProducer() (transire.QueueProducer, error) {
	// TODO: Implement AWS SQS queue producer
	return nil, fmt.Errorf("queue producer not yet implemented for lambda runtime")
}

// handleLambdaEvent is the main Lambda handler that routes events
func (r *Runtime) handleLambdaEvent(ctx context.Context, event json.RawMessage) (interface{}, error) {
	log.Printf("Received Lambda event: %s", string(event))

	// Try to determine event type and route accordingly
	if r.isAPIGatewayEvent(event) {
		return r.handleHTTP(ctx, event)
	} else if r.isSQSEvent(event) {
		return r.handleQueue(ctx, event)
	} else if r.isEventBridgeEvent(event) {
		return r.handleSchedule(ctx, event)
	}

	return nil, fmt.Errorf("unknown event type: %s", string(event))
}

// handleHTTP processes API Gateway events
func (r *Runtime) handleHTTP(ctx context.Context, event json.RawMessage) (events.APIGatewayV2HTTPResponse, error) {
	var apiEvent events.APIGatewayV2HTTPRequest
	if err := json.Unmarshal(event, &apiEvent); err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       "Internal Server Error",
		}, fmt.Errorf("failed to unmarshal API Gateway event: %w", err)
	}

	// Convert API Gateway event to standard HTTP request
	req, err := r.apiGatewayToHTTPRequest(apiEvent)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       "Internal Server Error",
		}, fmt.Errorf("failed to convert API Gateway event to HTTP request: %w", err)
	}

	// Process request through Chi router
	recorder := httptest.NewRecorder()
	r.app.Router().ServeHTTP(recorder, req)

	// Convert response back to API Gateway format
	return r.httpResponseToAPIGateway(recorder), nil
}

// handleQueue processes SQS events
func (r *Runtime) handleQueue(ctx context.Context, event json.RawMessage) (events.SQSEventResponse, error) {
	var sqsEvent events.SQSEvent
	if err := json.Unmarshal(event, &sqsEvent); err != nil {
		return events.SQSEventResponse{}, fmt.Errorf("failed to unmarshal SQS event: %w", err)
	}

	if len(sqsEvent.Records) == 0 {
		return events.SQSEventResponse{}, nil
	}

	// Extract queue name from event source ARN
	queueName := r.extractQueueName(sqsEvent.Records[0].EventSourceARN)

	// Find the appropriate handler
	handler := r.app.FindQueueHandler(queueName)
	if handler == nil {
		return events.SQSEventResponse{}, fmt.Errorf("no handler found for queue: %s", queueName)
	}

	// Convert SQS records to messages
	messages := r.sqsRecordsToMessages(sqsEvent.Records)

	// Process messages
	failedIDs, err := handler.HandleMessages(ctx, messages)
	if err != nil {
		// If handler returns error, mark all messages as failed
		var allFailures []events.SQSBatchItemFailure
		for _, record := range sqsEvent.Records {
			allFailures = append(allFailures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
		}
		return events.SQSEventResponse{BatchItemFailures: allFailures}, nil
	}

	// Convert failed IDs to batch response format
	var batchFailures []events.SQSBatchItemFailure
	for _, failedID := range failedIDs {
		batchFailures = append(batchFailures, events.SQSBatchItemFailure{
			ItemIdentifier: failedID,
		})
	}

	return events.SQSEventResponse{BatchItemFailures: batchFailures}, nil
}

// handleSchedule processes EventBridge events
func (r *Runtime) handleSchedule(ctx context.Context, event json.RawMessage) (interface{}, error) {
	var ebEvent events.CloudWatchEvent
	if err := json.Unmarshal(event, &ebEvent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal EventBridge event: %w", err)
	}

	// Extract schedule name from rule name or detail
	scheduleName := r.extractScheduleName(ebEvent)

	// Find the appropriate handler
	handler := r.app.FindScheduleHandler(scheduleName)
	if handler == nil {
		return nil, fmt.Errorf("no handler found for schedule: %s", scheduleName)
	}

	// Create schedule event
	schedEvent := transire.ScheduleEvent{
		ScheduledTime: ebEvent.Time,
		Name:          scheduleName,
		EventID:       ebEvent.ID,
	}

	// Process schedule
	return nil, handler.HandleSchedule(ctx, schedEvent)
}

// Event type detection helpers
func (r *Runtime) isAPIGatewayEvent(event json.RawMessage) bool {
	eventStr := string(event)
	hasRequestContext := strings.Contains(eventStr, "requestContext")
	isV2 := strings.Contains(eventStr, "\"version\"") && strings.Contains(eventStr, "\"2.0\"")
	isV1REST := strings.Contains(eventStr, "httpMethod")
	hasAPIGateway := strings.Contains(eventStr, "apigateway")

	return hasRequestContext && (isV2 || isV1REST || hasAPIGateway)
}

func (r *Runtime) isSQSEvent(event json.RawMessage) bool {
	return strings.Contains(string(event), "Records") && strings.Contains(string(event), "eventSource") &&
		strings.Contains(string(event), "aws:sqs")
}

func (r *Runtime) isEventBridgeEvent(event json.RawMessage) bool {
	return strings.Contains(string(event), "source") && strings.Contains(string(event), "detail-type") &&
		strings.Contains(string(event), "aws.events")
}

// Conversion helpers
func (r *Runtime) apiGatewayToHTTPRequest(event events.APIGatewayV2HTTPRequest) (*http.Request, error) {
	// Create URL
	url := fmt.Sprintf("https://%s%s", event.RequestContext.DomainName, event.RawPath)
	if event.RawQueryString != "" {
		url += "?" + event.RawQueryString
	}

	// Create request
	req, err := http.NewRequest(event.RequestContext.HTTP.Method, url, strings.NewReader(event.Body))
	if err != nil {
		return nil, err
	}

	// Set headers
	for key, value := range event.Headers {
		req.Header.Set(key, value)
	}

	// Set path parameters in context (Chi will handle this)
	for key, value := range event.PathParameters {
		req = req.WithContext(context.WithValue(req.Context(), contextKey(key), value))
	}

	return req, nil
}

func (r *Runtime) httpResponseToAPIGateway(recorder *httptest.ResponseRecorder) events.APIGatewayV2HTTPResponse {
	headers := make(map[string]string)
	for key, values := range recorder.Header() {
		headers[key] = values[0]
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: recorder.Code,
		Headers:    headers,
		Body:       recorder.Body.String(),
	}
}

func (r *Runtime) extractQueueName(eventSourceARN string) string {
	// Extract queue name from ARN: arn:aws:sqs:region:account:queue-name
	parts := strings.Split(eventSourceARN, ":")
	if len(parts) >= 6 {
		fullQueueName := parts[5]
		return r.queueNameToHandlerName(fullQueueName)
	}
	return "unknown"
}

// queueNameToHandlerName converts a CloudFormation queue name to a handler name
// CloudFormation generates names like: {app-name}-{environment}-{logical-queue-name}
// We need to strip the stack prefix (app-name-environment-) to get the logical name
func (r *Runtime) queueNameToHandlerName(fullQueueName string) string {
	// Get stack prefix from environment variables
	appName := os.Getenv("TRANSIRE_APP_NAME")
	environment := os.Getenv("TRANSIRE_ENVIRONMENT")

	if appName != "" && environment != "" {
		// Construct the expected prefix: {name}-{environment}-
		expectedPrefix := appName + "-" + environment + "-"
		if strings.HasPrefix(fullQueueName, expectedPrefix) {
			return strings.TrimPrefix(fullQueueName, expectedPrefix)
		}
	}

	// Fallback: return as-is if we can't determine the prefix
	return fullQueueName
}

func (r *Runtime) extractScheduleName(event events.CloudWatchEvent) string {
	// Try to extract from resources first
	for _, resource := range event.Resources {
		if strings.Contains(resource, "rule/") {
			parts := strings.Split(resource, "/")
			if len(parts) >= 2 {
				ruleName := parts[len(parts)-1]
				return r.ruleNameToHandlerName(ruleName)
			}
		}
	}

	// Fallback to detail type or source
	if event.DetailType != "" {
		return event.DetailType
	}

	return "unknown"
}

// ruleNameToHandlerName converts a CloudFormation rule name to a handler name
// CloudFormation generates names like: {app-name}-{environment}-{logical-schedule-name}
// We need to strip the stack prefix to get the logical name
func (r *Runtime) ruleNameToHandlerName(ruleName string) string {
	// Get stack prefix from environment variables
	appName := os.Getenv("TRANSIRE_APP_NAME")
	environment := os.Getenv("TRANSIRE_ENVIRONMENT")

	if appName != "" && environment != "" {
		// Construct the expected prefix: {name}-{environment}-
		expectedPrefix := appName + "-" + environment + "-"
		if strings.HasPrefix(ruleName, expectedPrefix) {
			return strings.TrimPrefix(ruleName, expectedPrefix)
		}
	}

	// Fallback: return as-is if we can't determine the prefix
	return ruleName
}

func (r *Runtime) sqsRecordsToMessages(records []events.SQSMessage) []transire.Message {
	messages := make([]transire.Message, len(records))
	for i, record := range records {
		messages[i] = &Message{
			id:         record.MessageId,
			body:       []byte(record.Body),
			attributes: record.MessageAttributes,
		}
	}
	return messages
}

// init registers the Lambda runtime during package initialization
func init() {
	transire.RegisterDefaultRuntime(func(config *transire.Config) transire.Runtime {
		return NewLambdaRuntime()
	})
}
