// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package transire

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stubDispatcher struct {
	runCalled bool
}

func (s *stubDispatcher) Run(ctx context.Context, app *App) error {
	s.runCalled = true
	return nil
}
func (s *stubDispatcher) Name() string { return "stub" }

func TestRegisterHandlers(t *testing.T) {
	app := New()
	app.RegisterQueueHandler("q1", func(ctx Context, msg Message) error { return nil })
	app.RegisterScheduleHandler("job1", time.Minute, func(ctx Context, at time.Time) error { return nil })

	if _, ok := app.QueueHandlers()["q1"]; !ok {
		t.Fatalf("queue handler not registered")
	}
	if sched, ok := app.Schedules()["job1"]; !ok || sched.Every != time.Minute {
		t.Fatalf("schedule handler not registered correctly")
	}
}

func TestRunRequiresDispatcher(t *testing.T) {
	app := New()
	if err := app.Run(context.Background()); !errors.Is(err, ErrNoDispatcher) {
		t.Fatalf("expected ErrNoDispatcher, got %v", err)
	}
}

func TestRunInvokesDispatcher(t *testing.T) {
	app := New()
	d := &stubDispatcher{}
	app.SetDispatcher(d)
	if err := app.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !d.runCalled {
		t.Fatalf("dispatcher Run was not called")
	}
}
