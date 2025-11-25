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
	queue string
	body  []byte
}

type recordingSender struct {
	messages []queuedMessage
}

func (r *recordingSender) Send(ctx context.Context, queue string, payload []byte) error {
	r.messages = append(r.messages, queuedMessage{queue: queue, body: payload})
	return nil
}

func TestHTTPHandlerEnqueuesWork(t *testing.T) {
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
		t.Fatalf("unexpected status %d", rr.Code)
	}
	if len(sender.messages) != 1 {
		t.Fatalf("expected one queued message, got %d", len(sender.messages))
	}

	var payload WorkPayload
	if err := json.Unmarshal(sender.messages[0].body, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Source != "http" {
		t.Fatalf("expected source http, got %s", payload.Source)
	}
	if payload.Detail != "demo" {
		t.Fatalf("expected detail demo, got %s", payload.Detail)
	}
}

func TestWorkHandlerForwardsSummary(t *testing.T) {
	sender := &recordingSender{}
	app := transire.New()
	app.SetQueueSender(sender)
	RegisterQueues(app)

	work := WorkPayload{Source: "schedule", Detail: "beat"}
	body, err := json.Marshal(work)
	if err != nil {
		t.Fatalf("marshal work payload: %v", err)
	}

	handler := app.QueueHandlers()[WorkQueue]
	if handler == nil {
		t.Fatalf("work handler missing")
	}
	err = handler(transire.Context{Context: context.Background(), Queues: sender}, transire.Message{
		Queue: WorkQueue,
		Body:  body,
	})
	if err != nil {
		t.Fatalf("work handler error: %v", err)
	}

	if len(sender.messages) != 1 {
		t.Fatalf("expected summary enqueue, got %d messages", len(sender.messages))
	}
	if sender.messages[0].queue != SummaryQueue {
		t.Fatalf("expected send to %s, got %s", SummaryQueue, sender.messages[0].queue)
	}

	var summary SummaryPayload
	if err := json.Unmarshal(sender.messages[0].body, &summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if summary.Source != "schedule" {
		t.Fatalf("expected summary source schedule, got %s", summary.Source)
	}
	if len(summary.Steps) != 1 {
		t.Fatalf("expected one step, got %d", len(summary.Steps))
	}
	if summary.Steps[0] != "work accepted: beat" {
		t.Fatalf("unexpected step: %s", summary.Steps[0])
	}
}

func TestSummaryHandlerForwardsLog(t *testing.T) {
	sender := &recordingSender{}
	app := transire.New()
	app.SetQueueSender(sender)
	RegisterQueues(app)

	summary := SummaryPayload{Source: "http", Steps: []string{"work accepted: demo"}}
	body, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("marshal summary: %v", err)
	}

	handler := app.QueueHandlers()[SummaryQueue]
	if handler == nil {
		t.Fatalf("summary handler missing")
	}
	err = handler(transire.Context{Context: context.Background(), Queues: sender}, transire.Message{
		Queue: SummaryQueue,
		Body:  body,
	})
	if err != nil {
		t.Fatalf("summary handler error: %v", err)
	}

	if len(sender.messages) != 1 {
		t.Fatalf("expected log enqueue, got %d messages", len(sender.messages))
	}
	if sender.messages[0].queue != LogQueue {
		t.Fatalf("expected send to %s, got %s", LogQueue, sender.messages[0].queue)
	}

	var logPayload LogPayload
	if err := json.Unmarshal(sender.messages[0].body, &logPayload); err != nil {
		t.Fatalf("decode log payload: %v", err)
	}
	if logPayload.Message != "work accepted: demo -> forwarded to log" {
		t.Fatalf("unexpected log message: %s", logPayload.Message)
	}
}

func TestLogHandlerNoop(t *testing.T) {
	sender := &recordingSender{}
	app := transire.New()
	app.SetQueueSender(sender)
	RegisterQueues(app)

	handler := app.QueueHandlers()[LogQueue]
	if handler == nil {
		t.Fatalf("log handler missing")
	}
	err := handler(transire.Context{Context: context.Background(), Queues: sender}, transire.Message{
		Queue: LogQueue,
		Body:  []byte(`{"message":"hi"}`),
	})
	if err != nil {
		t.Fatalf("log handler returned error: %v", err)
	}
	if len(sender.messages) != 0 {
		t.Fatalf("log handler should not enqueue more messages, got %d", len(sender.messages))
	}
}

func TestScheduleHandlerEnqueuesWork(t *testing.T) {
	sender := &recordingSender{}
	app := transire.New()
	app.SetQueueSender(sender)
	RegisterSchedules(app)

	sched, ok := app.Schedules()["heartbeat"]
	if !ok {
		t.Fatalf("schedule heartbeat missing")
	}
	now := time.Now()

	err := sched.Handler(transire.Context{Context: context.Background(), Queues: sender}, now)
	if err != nil {
		t.Fatalf("schedule handler error: %v", err)
	}

	if len(sender.messages) != 1 {
		t.Fatalf("expected one queued message, got %d", len(sender.messages))
	}
	if sender.messages[0].queue != WorkQueue {
		t.Fatalf("expected send to %s, got %s", WorkQueue, sender.messages[0].queue)
	}

	var payload WorkPayload
	if err := json.Unmarshal(sender.messages[0].body, &payload); err != nil {
		t.Fatalf("decode work payload: %v", err)
	}
	if payload.Source != "schedule" {
		t.Fatalf("expected schedule source, got %s", payload.Source)
	}
	if payload.Detail == "" {
		t.Fatalf("expected detail to be populated")
	}
}
