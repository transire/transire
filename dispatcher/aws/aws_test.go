// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package aws

import "testing"

func TestQueueEnvVar(t *testing.T) {
	got := queueEnvVar("hello-queue")
	if got != "TRANSIRE_QUEUE_HELLO_QUEUE_URL" {
		t.Fatalf("unexpected env var: %s", got)
	}
}

func TestQueueNameEnvVar(t *testing.T) {
	got := queueNameEnvVar("hello-queue")
	if got != "TRANSIRE_QUEUE_HELLO_QUEUE_NAME" {
		t.Fatalf("unexpected name env var: %s", got)
	}
}

func TestScheduleNameEnvVar(t *testing.T) {
	got := scheduleNameEnvVar("heartbeat")
	if got != "TRANSIRE_SCHEDULE_HEARTBEAT_NAME" {
		t.Fatalf("unexpected schedule env var: %s", got)
	}
}

func TestExtractQueueName(t *testing.T) {
	arn := "arn:aws:sqs:eu-west-2:123456789012:my-queue"
	if got := extractQueueName(arn); got != "my-queue" {
		t.Fatalf("expected my-queue, got %s", got)
	}
}

func TestExtractRuleName(t *testing.T) {
	arn := "arn:aws:events:region:acct:rule/my-rule"
	if got := extractRuleName([]string{arn}); got != "my-rule" {
		t.Fatalf("expected my-rule, got %s", got)
	}
	if got := extractRuleName(nil); got != "" {
		t.Fatalf("expected empty for no resources, got %s", got)
	}
}

func TestInvert(t *testing.T) {
	in := map[string]string{"a": "1", "b": ""}
	out := invert(in)
	if len(out) != 1 || out["1"] != "a" {
		t.Fatalf("unexpected invert result: %+v", out)
	}
}
