// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	chiproxy "github.com/awslabs/aws-lambda-go-api-proxy/chi"
	"github.com/go-chi/chi/v5"
	transire "github.com/transire/transire"
)

const queueEnvPrefix = "TRANSIRE_QUEUE_"
const queueNameEnvSuffix = "_NAME"
const scheduleEnvPrefix = "TRANSIRE_SCHEDULE_"

// Dispatcher wires AWS events (API Gateway v2, SQS, EventBridge) into handlers.
type Dispatcher struct {
	Region string
}

// Name identifies the dispatcher.
func (d *Dispatcher) Name() string {
	return "aws"
}

// Run sets up the Lambda handler for API Gateway, SQS, and EventBridge events.
func (d *Dispatcher) Run(ctx context.Context, app *transire.App) error {
	region := d.Region
	if region == "" {
		region = os.Getenv("AWS_REGION")
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return fmt.Errorf("load aws config: %w", err)
	}

	queueURLs := make(map[string]string)
	queueNames := make(map[string]string)
	scheduleNames := make(map[string]string)
	for name := range app.QueueHandlers() {
		envKey := queueEnvVar(name)
		url := os.Getenv(envKey)
		if url == "" {
			log.Printf("queue %s missing URL in env %s; messages to this queue will fail\n", name, envKey)
		}
		queueURLs[name] = url

		nameKey := queueNameEnvVar(name)
		queueNames[name] = os.Getenv(nameKey)
	}

	for name := range app.Schedules() {
		nameKey := scheduleNameEnvVar(name)
		scheduleNames[name] = os.Getenv(nameKey)
	}

	sqsClient := sqs.NewFromConfig(cfg)
	queueSender := &awsQueueSender{
		client: sqsClient,
		urls:   queueURLs,
	}
	app.SetQueueSender(queueSender)

	root := chi.NewRouter()
	root.Use(transire.InjectContext(app.QueueSender()))
	root.Mount("/", app.Router())

	adapter := chiproxy.NewV2(root)
	fqdnToLogical := invert(queueNames)
	for k, v := range invert(scheduleNames) {
		fqdnToLogical[k] = v
	}

	handler := func(ctx context.Context, raw json.RawMessage) (any, error) {
		// Detect API Gateway HTTP event
		if looksLikeAPIGateway(raw) {
			var req events.APIGatewayV2HTTPRequest
			if err := json.Unmarshal(raw, &req); err != nil {
				return nil, err
			}
			return adapter.ProxyWithContextV2(ctx, req)
		}

		// Detect SQS
		if looksLikeSQS(raw) {
			var sqsEvent events.SQSEvent
			if err := json.Unmarshal(raw, &sqsEvent); err != nil {
				return nil, err
			}
			return nil, d.handleSQSEvent(ctx, app, sqsEvent, fqdnToLogical)
		}

		// Treat as EventBridge schedule
		var ev events.CloudWatchEvent
		if err := json.Unmarshal(raw, &ev); err != nil {
			return nil, err
		}
		return nil, d.handleSchedule(ctx, app, ev, fqdnToLogical)
	}

	lambda.Start(handler)
	return nil
}

func (d *Dispatcher) handleSQSEvent(ctx context.Context, app *transire.App, ev events.SQSEvent, fqdnToLogical map[string]string) error {
	for _, record := range ev.Records {
		queueFQDN := extractQueueName(record.EventSourceARN)
		queueName := fqdnToLogical[queueFQDN]
		if queueName == "" {
			queueName = queueFQDN
		}
		handler, ok := app.QueueHandlers()[queueName]
		if !ok {
			log.Printf("no handler for queue %s (fqdn %s)", queueName, queueFQDN)
			continue
		}

		msg := transire.Message{
			ID:         record.MessageId,
			Queue:      queueName,
			Body:       []byte(record.Body),
			Attributes: map[string]string{},
		}
		for k, v := range record.MessageAttributes {
			msg.Attributes[k] = *v.StringValue
		}

		if err := handler(transire.Context{
			Context: ctx,
			Queues:  app.QueueSender(),
		}, msg); err != nil {
			log.Printf("handler for queue %s failed: %v", queueName, err)
		}
	}
	return nil
}

func (d *Dispatcher) handleSchedule(ctx context.Context, app *transire.App, ev events.CloudWatchEvent, fqdnToLogical map[string]string) error {
	name := extractRuleName(ev.Resources)
	if mapped := fqdnToLogical[name]; mapped != "" {
		name = mapped
	}
	sched, ok := app.Schedules()[name]
	if !ok {
		log.Printf("no schedule handler for %s", name)
		return nil
	}
	return sched.Handler(transire.Context{
		Context: ctx,
		Queues:  app.QueueSender(),
	}, ev.Time)
}

type awsQueueSender struct {
	client *sqs.Client
	urls   map[string]string
}

func (s *awsQueueSender) Send(ctx context.Context, queue string, payload []byte) error {
	url := s.urls[queue]
	if url == "" {
		return fmt.Errorf("queue %s has no URL configured", queue)
	}

	_, err := s.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(url),
		MessageBody: aws.String(string(payload)),
	})
	return err
}

func queueEnvVar(queue string) string {
	name := strings.ToUpper(queue)
	name = strings.ReplaceAll(name, "-", "_")
	return queueEnvPrefix + name + "_URL"
}

func queueNameEnvVar(queue string) string {
	name := strings.ToUpper(queue)
	name = strings.ReplaceAll(name, "-", "_")
	return queueEnvPrefix + name + queueNameEnvSuffix
}

func scheduleNameEnvVar(name string) string {
	up := strings.ToUpper(name)
	up = strings.ReplaceAll(up, "-", "_")
	return scheduleEnvPrefix + up + queueNameEnvSuffix
}

func extractQueueName(arn string) string {
	parts := strings.Split(arn, ":")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return arn
}

func extractRuleName(resources []string) string {
	if len(resources) == 0 {
		return ""
	}
	arn := resources[0]
	parts := strings.Split(arn, "/")
	return parts[len(parts)-1]
}

func looksLikeAPIGateway(raw json.RawMessage) bool {
	return strings.Contains(string(raw), `"requestContext"`) && strings.Contains(string(raw), `"http"`)
}

func looksLikeSQS(raw json.RawMessage) bool {
	return strings.Contains(string(raw), `"Records"`) && strings.Contains(string(raw), `"aws:sqs"`)
}

func invert(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		if v == "" {
			continue
		}
		out[v] = k
	}
	return out
}
