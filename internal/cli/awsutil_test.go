// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cli

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestQueueOutputKey(t *testing.T) {
	if got := queueOutputKey("hello-queue"); got != "helloqueueQueueUrl" {
		t.Fatalf("unexpected queue output key: %s", got)
	}
}

func TestScheduleOutputKey(t *testing.T) {
	if got := scheduleOutputKey("heartbeat"); got != "heartbeatScheduleName" {
		t.Fatalf("unexpected schedule output key: %s", got)
	}
}

func TestFetchOutputs(t *testing.T) {
	mock := mockCfn{Outputs: map[string]string{"ApiEndpoint": "https://example"}}
	out, err := fetchOutputs(context.Background(), mock, "stack")
	if err != nil {
		t.Fatalf("fetch outputs err: %v", err)
	}
	if out["ApiEndpoint"] != "https://example" {
		t.Fatalf("missing output")
	}
}

func TestSendAWSQueue(t *testing.T) {
	sqs := &mockSQS{}
	out := map[string]string{queueOutputKey("q"): "url"}
	err := sendAWSQueue(context.Background(), sqs, out, "q", []byte("hi"))
	if err != nil {
		t.Fatalf("send err: %v", err)
	}
	if sqs.LastURL != "url" || sqs.LastMessage != "hi" {
		t.Fatalf("unexpected send inputs: %+v", sqs)
	}
	if err := sendAWSQueue(context.Background(), sqs, map[string]string{}, "missing", []byte("x")); err == nil {
		t.Fatalf("expected error for missing queue url")
	}
}

func TestTriggerScheduleLambda(t *testing.T) {
	lm := &mockLambda{}
	out := map[string]string{
		scheduleOutputKey("heartbeat"): "rule-name",
		"LambdaName":                   "my-lambda",
	}
	if err := triggerScheduleLambda(context.Background(), lm, out, "heartbeat"); err != nil {
		t.Fatalf("trigger err: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(lm.InvokedWith, &payload); err != nil {
		t.Fatalf("payload not json: %v", err)
	}
	res := payload["resources"].([]any)
	if res[0] != "rule-name" {
		t.Fatalf("unexpected resources: %v", res)
	}
}

func TestStackName(t *testing.T) {
	if got := stackName("app"); got != "app-stack" {
		t.Fatalf("unexpected stack name: %s", got)
	}
}

func TestStackLambdaNameFallback(t *testing.T) {
	out := map[string]string{"LambdaName": "fn"}
	if got := stackLambdaName(out); got != "fn" {
		t.Fatalf("unexpected lambda name: %s", got)
	}
	out = map[string]string{"SomeKey": "val"}
	if got := stackLambdaName(out); got != "" {
		t.Fatalf("expected empty fallback, got %s", got)
	}
}

func TestSafeID(t *testing.T) {
	if got := safeID("a-b_c"); got != "abc" {
		t.Fatalf("unexpected safeID: %s", got)
	}
	if got := safeID(""); got != "id" {
		t.Fatalf("expected id fallback")
	}
	if strings.Contains(safeID("id"), "-") {
		t.Fatalf("safeID contains dash")
	}
}
