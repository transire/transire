// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cfntypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type cfnAPI interface {
	DescribeStacks(context.Context, *cloudformation.DescribeStacksInput, ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error)
}

type sqsAPI interface {
	SendMessage(context.Context, *sqs.SendMessageInput, ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

type lambdaAPI interface {
	Invoke(context.Context, *lambda.InvokeInput, ...func(*lambda.Options)) (*lambda.InvokeOutput, error)
}

func awsConfig(ctx context.Context, profile, region string) (aws.Config, error) {
	opts := []func(*config.LoadOptions) error{}
	if profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}
	return config.LoadDefaultConfig(ctx, opts...)
}

func stackName(appName string) string {
	return appName + "-stack"
}

func queueOutputKey(queue string) string {
	return safeID(queue) + "QueueUrl"
}

func scheduleOutputKey(name string) string {
	return safeID(name) + "ScheduleName"
}

func safeID(name string) string {
	id := strings.ReplaceAll(name, "-", "")
	id = strings.ReplaceAll(id, "_", "")
	if id == "" {
		return "id"
	}
	return id
}

func fetchOutputs(ctx context.Context, cfn cfnAPI, stack string) (map[string]string, error) {
	resp, err := cfn.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(stack),
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Stacks) == 0 {
		return nil, fmt.Errorf("stack %s not found", stack)
	}
	out := map[string]string{}
	for _, o := range resp.Stacks[0].Outputs {
		if o.OutputKey != nil && o.OutputValue != nil {
			out[*o.OutputKey] = *o.OutputValue
		}
	}
	return out, nil
}

func sendAWSQueue(ctx context.Context, sqsClient sqsAPI, outputs map[string]string, queue string, payload []byte) error {
	url := outputs[queueOutputKey(queue)]
	if url == "" {
		return fmt.Errorf("queue %s URL not found in stack outputs", queue)
	}
	_, err := sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(url),
		MessageBody: aws.String(string(payload)),
	})
	return err
}

func triggerScheduleLambda(ctx context.Context, lambdaClient lambdaAPI, outputs map[string]string, schedule string) error {
	name := outputs[scheduleOutputKey(schedule)]
	if name == "" {
		return fmt.Errorf("schedule %s name not found in stack outputs", schedule)
	}
	payload := map[string]any{
		"resources": []string{name},
		"time":      time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = lambdaClient.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   aws.String(stackLambdaName(outputs)),
		Payload:        data,
		InvocationType: lambdatypes.InvocationTypeEvent,
	})
	return err
}

func stackLambdaName(outputs map[string]string) string {
	for k, v := range outputs {
		if strings.Contains(strings.ToLower(k), "lambda") {
			return v
		}
	}
	// fallback: try to infer from API endpoint output if present (not ideal but defensive)
	return outputs["TransireLambdaName"]
}

// mockable helpers for tests
type mockCfn struct {
	Outputs map[string]string
}

func (m mockCfn) DescribeStacks(ctx context.Context, in *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	var outs []cfntypes.Output
	for k, v := range m.Outputs {
		outs = append(outs, cfntypes.Output{OutputKey: aws.String(k), OutputValue: aws.String(v)})
	}
	return &cloudformation.DescribeStacksOutput{
		Stacks: []cfntypes.Stack{{Outputs: outs}},
	}, nil
}

type mockSQS struct {
	LastURL     string
	LastMessage string
}

func (m *mockSQS) SendMessage(ctx context.Context, in *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	m.LastURL = aws.ToString(in.QueueUrl)
	m.LastMessage = aws.ToString(in.MessageBody)
	return &sqs.SendMessageOutput{}, nil
}

type mockLambda struct {
	InvokedWith []byte
}

func (m *mockLambda) Invoke(ctx context.Context, in *lambda.InvokeInput, optFns ...func(*lambda.Options)) (*lambda.InvokeOutput, error) {
	m.InvokedWith = append(m.InvokedWith, in.Payload...)
	return &lambda.InvokeOutput{}, nil
}
