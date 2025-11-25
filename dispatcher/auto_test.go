// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package dispatcher

import (
	"testing"
)

func TestAuto_DefaultsToLocal(t *testing.T) {
	t.Setenv(dispatcherEnv, "")
	t.Setenv("AWS_LAMBDA_RUNTIME_API", "")
	t.Setenv("AWS_EXECUTION_ENV", "")
	t.Setenv("LAMBDA_TASK_ROOT", "")

	d, err := Auto()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := d.Name(); got != "local" {
		t.Fatalf("expected local dispatcher, got %s", got)
	}
}

func TestAuto_PicksAWSInLambdaEnv(t *testing.T) {
	t.Setenv(dispatcherEnv, "")
	t.Setenv("AWS_LAMBDA_RUNTIME_API", "http://localhost:9001")

	d, err := Auto()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := d.Name(); got != "aws" {
		t.Fatalf("expected aws dispatcher, got %s", got)
	}
}

func TestAuto_Override(t *testing.T) {
	t.Setenv(dispatcherEnv, "local")
	d, err := Auto()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := d.Name(); got != "local" {
		t.Fatalf("expected local dispatcher, got %s", got)
	}

	t.Setenv(dispatcherEnv, "aws")
	d, err = Auto()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := d.Name(); got != "aws" {
		t.Fatalf("expected aws dispatcher, got %s", got)
	}
}

func TestAuto_InvalidOverride(t *testing.T) {
	t.Setenv(dispatcherEnv, "nope")
	if _, err := Auto(); err == nil {
		t.Fatalf("expected error for invalid override")
	}
}
