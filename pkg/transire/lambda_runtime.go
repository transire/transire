package transire

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// lambdaRuntime implements the Runtime interface for AWS Lambda
type lambdaRuntime struct {
	app *App
}

// newLambdaRuntime creates a new Lambda runtime
func newLambdaRuntime() Runtime {
	return &lambdaRuntime{}
}

// Start begins processing in the Lambda environment
func (r *lambdaRuntime) Start(ctx context.Context, app *App) error {
	r.app = app

	// Start the AWS Lambda handler
	lambda.Start(r.handleLambdaEvent)
	return nil
}

// Stop is not applicable for Lambda runtime
func (r *lambdaRuntime) Stop(ctx context.Context) error {
	// Lambda runtime doesn't need explicit stopping
	return nil
}

// IsLocal returns false since this is the Lambda runtime
func (r *lambdaRuntime) IsLocal() bool {
	return false
}

// handleLambdaEvent is the main Lambda handler that routes events
func (r *lambdaRuntime) handleLambdaEvent(ctx context.Context, event json.RawMessage) (interface{}, error) {
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
func (r *lambdaRuntime) handleHTTP(ctx context.Context, event json.RawMessage) (events.APIGatewayV2HTTPResponse, error) {
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
func (r *lambdaRuntime) handleQueue(ctx context.Context, event json.RawMessage) (events.SQSEventResponse, error) {
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
func (r *lambdaRuntime) handleSchedule(ctx context.Context, event json.RawMessage) (interface{}, error) {
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
	schedEvent := ScheduleEvent{
		ScheduledTime: ebEvent.Time,
		Name:          scheduleName,
		EventID:       ebEvent.ID,
	}

	// Process schedule
	return nil, handler.HandleSchedule(ctx, schedEvent)
}

// Event type detection helpers
func (r *lambdaRuntime) isAPIGatewayEvent(event json.RawMessage) bool {
	eventStr := string(event)
	hasRequestContext := strings.Contains(eventStr, "requestContext")
	isV2 := strings.Contains(eventStr, "\"version\"") && strings.Contains(eventStr, "\"2.0\"") // API Gateway v2 HTTP API
	isV1REST := strings.Contains(eventStr, "httpMethod")                                       // API Gateway v1 REST API
	hasAPIGateway := strings.Contains(eventStr, "apigateway")                                  // Other API Gateway formats

	return hasRequestContext && (isV2 || isV1REST || hasAPIGateway)
}

func (r *lambdaRuntime) isSQSEvent(event json.RawMessage) bool {
	return strings.Contains(string(event), "Records") && strings.Contains(string(event), "eventSource") &&
		strings.Contains(string(event), "aws:sqs")
}

func (r *lambdaRuntime) isEventBridgeEvent(event json.RawMessage) bool {
	return strings.Contains(string(event), "source") && strings.Contains(string(event), "detail-type") &&
		strings.Contains(string(event), "aws.events")
}

// Conversion helpers
func (r *lambdaRuntime) apiGatewayToHTTPRequest(event events.APIGatewayV2HTTPRequest) (*http.Request, error) {
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
		// Store in request context for Chi
		req = req.WithContext(context.WithValue(req.Context(), contextKey(key), value))
	}

	return req, nil
}

func (r *lambdaRuntime) httpResponseToAPIGateway(recorder *httptest.ResponseRecorder) events.APIGatewayV2HTTPResponse {
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

func (r *lambdaRuntime) extractQueueName(eventSourceARN string) string {
	// Extract queue name from ARN: arn:aws:sqs:region:account:queue-name
	parts := strings.Split(eventSourceARN, ":")
	if len(parts) >= 6 {
		return parts[5]
	}
	return "unknown"
}

func (r *lambdaRuntime) extractScheduleName(event events.CloudWatchEvent) string {
	// Try to extract from resources first
	for _, resource := range event.Resources {
		if strings.Contains(resource, "rule/") {
			parts := strings.Split(resource, "/")
			if len(parts) >= 2 {
				ruleName := parts[len(parts)-1]
				// Convert CloudFormation-generated rule name to handler name
				// e.g., "todo-app-stack-CleanupCompletedTodosRuleF0BFE00D-ABCD123" -> "cleanup-completed-todos"
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
// It extracts the PascalCase portion and converts it to kebab-case
func (r *lambdaRuntime) ruleNameToHandlerName(ruleName string) string {
	// Pattern: prefix-PascalCaseRuleSuffix-cfnHash
	// We want to extract "PascalCase" and convert to "pascal-case"

	// First, try to find the pattern with "Rule" in it
	// Look for uppercase letter followed by "Rule"
	var extracted string

	// Find segments that look like CloudFormation logical resource names (PascalCase)
	// They typically contain "Rule" and end with a CloudFormation hash
	ruleIdx := strings.Index(ruleName, "Rule")
	if ruleIdx > 0 {
		// Find the start of the PascalCase part (first uppercase after hyphens)
		start := 0
		for i := ruleIdx - 1; i >= 0; i-- {
			if ruleName[i] == '-' {
				start = i + 1
				break
			}
		}

		// Extract from start to just before "Rule" suffix
		// e.g., "CleanupCompletedTodosRule" -> "CleanupCompletedTodos"
		extracted = ruleName[start:ruleIdx]

		// Remove any remaining CloudFormation hash suffix
		// Find next hyphen after "Rule"
		remaining := ruleName[ruleIdx+4:] // Skip "Rule"
		nextHyphen := strings.Index(remaining, "-")
		if nextHyphen == -1 {
			// No more hyphens, check if there's a CloudFormation hash at the end
			// CloudFormation hashes are typically alphanumeric uppercase
			if len(remaining) > 0 && isCloudFormationHash(remaining) {
				// This is just "Rule" + hash, so we use the extracted part
			} else {
				// Include any suffix after "Rule" that's not a hash
				extracted += remaining
			}
		}
	} else {
		// No "Rule" found, just use the whole name
		extracted = ruleName
	}

	// Convert PascalCase to kebab-case
	return pascalToKebab(extracted)
}

// isCloudFormationHash checks if a string looks like a CloudFormation hash
func isCloudFormationHash(s string) bool {
	if len(s) == 0 {
		return false
	}
	// CloudFormation hashes are typically 8-12 characters, alphanumeric, mixed case
	if len(s) < 8 || len(s) > 16 {
		return false
	}
	for _, c := range s {
		if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

// pascalToKebab converts PascalCase to kebab-case
func pascalToKebab(s string) string {
	if len(s) == 0 {
		return s
	}

	var result []rune
	for i, r := range s {
		// Add hyphen before uppercase letters (except the first character)
		if i > 0 && r >= 'A' && r <= 'Z' {
			// Don't add hyphen if previous char was also uppercase (for acronyms like "API")
			// unless this is the last char or next char is lowercase
			prevUpper := i > 0 && s[i-1] >= 'A' && s[i-1] <= 'Z'
			nextLower := i < len(s)-1 && s[i+1] >= 'a' && s[i+1] <= 'z'

			if !prevUpper || nextLower {
				result = append(result, '-')
			}
		}
		// Convert to lowercase
		if r >= 'A' && r <= 'Z' {
			result = append(result, r+32)
		} else {
			result = append(result, r)
		}
	}

	return string(result)
}

func (r *lambdaRuntime) sqsRecordsToMessages(records []events.SQSMessage) []Message {
	messages := make([]Message, len(records))
	for i, record := range records {
		messages[i] = &sqsMessage{
			id:         record.MessageId,
			body:       []byte(record.Body),
			attributes: record.MessageAttributes,
		}
	}
	return messages
}

// sqsMessage implements the Message interface for SQS messages
type sqsMessage struct {
	id         string
	body       []byte
	attributes map[string]events.SQSMessageAttribute
}

func (m *sqsMessage) ID() string {
	return m.id
}

func (m *sqsMessage) Body() []byte {
	return m.body
}

func (m *sqsMessage) Attributes() map[string]string {
	attrs := make(map[string]string)
	for key, attr := range m.attributes {
		if attr.StringValue != nil {
			attrs[key] = *attr.StringValue
		}
	}
	return attrs
}

func (m *sqsMessage) DeliveryCount() int {
	// SQS doesn't directly expose delivery count, would need to track separately
	return 1
}

func (m *sqsMessage) EnqueuedAt() time.Time {
	// Would need to extract from message attributes or approximate
	return time.Now()
}
