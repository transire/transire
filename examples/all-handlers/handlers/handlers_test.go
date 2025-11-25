// Copyright (c) 2025 Transire contributors
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/transire/transire"
)

type queuedMessage struct {
	queue   string
	payload []byte
}

type recordingSender struct {
	messages []queuedMessage
}

func (r *recordingSender) Send(ctx context.Context, queue string, payload []byte) error {
	r.messages = append(r.messages, queuedMessage{queue: queue, payload: payload})
	return nil
}

func TestHTTPHandlerSendsWorkQueue(t *testing.T) {
	sender := &recordingSender{}
	app := transire.New()
	app.SetQueueSender(sender)

	RegisterHTTP(app)

	root := chi.NewRouter()
	root.Use(transire.InjectContext(sender))
	root.Mount("/", app.Router())

	req := httptest.NewRequest(http.MethodGet, "/?msg=demo", nil)
	rr := httptest.NewRecorder()
	root.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
	if len(sender.messages) != 1 {
		t.Fatalf("expected 1 queued message, got %d", len(sender.messages))
	}

	var payload WorkPayload
	if err := json.Unmarshal(sender.messages[0].payload, &payload); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}
	if payload.Source != "http" {
		t.Fatalf("expected source http, got %s", payload.Source)
	}
	if payload.Detail != "demo" {
		t.Fatalf("expected detail demo, got %s", payload.Detail)
	}
}

func TestQueueHandlerForwardsToNotification(t *testing.T) {
	sender := &recordingSender{}
	app := transire.New()
	app.SetQueueSender(sender)
	RegisterQueues(app)

	work := WorkPayload{Source: "schedule", Detail: "beat"}
	body, err := json.Marshal(work)
	if err != nil {
		t.Fatalf("marshal work: %v", err)
	}

	handler := app.QueueHandlers()[WorkQueue]
	if handler == nil {
		t.Fatalf("work queue handler missing")
	}
	err = handler(transire.Context{Context: context.Background(), Queues: sender}, transire.Message{
		Queue: WorkQueue,
		Body:  body,
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if len(sender.messages) != 1 {
		t.Fatalf("expected notification to be queued, got %d messages", len(sender.messages))
	}
	if sender.messages[0].queue != NotificationsQueue {
		t.Fatalf("expected send to %s, got %s", NotificationsQueue, sender.messages[0].queue)
	}

	var note NotificationPayload
	if err := json.Unmarshal(sender.messages[0].payload, &note); err != nil {
		t.Fatalf("decode notification: %v", err)
	}
	if note.Stage != "work-processed" {
		t.Fatalf("unexpected stage: %s", note.Stage)
	}
}

func TestNotificationHandlerForwardsToLogQueue(t *testing.T) {
	sender := &recordingSender{}
	app := transire.New()
	app.SetQueueSender(sender)
	RegisterQueues(app)

	note := NotificationPayload{Stage: "work-processed", Detail: "done"}
	body, err := json.Marshal(note)
	if err != nil {
		t.Fatalf("marshal note: %v", err)
	}

	handler := app.QueueHandlers()[NotificationsQueue]
	if handler == nil {
		t.Fatalf("notification queue handler missing")
	}
	err = handler(transire.Context{Context: context.Background(), Queues: sender}, transire.Message{
		Queue: NotificationsQueue,
		Body:  body,
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if len(sender.messages) != 1 {
		t.Fatalf("expected 1 message forwarded, got %d", len(sender.messages))
	}
	if sender.messages[0].queue != NotificationLog {
		t.Fatalf("expected send to %s, got %s", NotificationLog, sender.messages[0].queue)
	}
}

func TestScheduleHandlerEnqueuesWork(t *testing.T) {
	sender := &recordingSender{}
	app := transire.New()
	app.SetQueueSender(sender)
	RegisterSchedules(app)

	sched, ok := app.Schedules()["heartbeat"]
	if !ok {
		t.Fatalf("schedule missing")
	}

	err := sched.Handler(transire.Context{Context: context.Background(), Queues: sender}, time.Now())
	if err != nil {
		t.Fatalf("schedule handler error: %v", err)
	}

	if len(sender.messages) != 1 {
		t.Fatalf("expected work message queued, got %d", len(sender.messages))
	}
	if sender.messages[0].queue != WorkQueue {
		t.Fatalf("expected send to %s, got %s", WorkQueue, sender.messages[0].queue)
	}
}
