package local

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/transire/transire"
)

func TestAdminQueueEndpointDispatches(t *testing.T) {
	app := transire.New()
	received := make(chan transire.Message, 1)

	app.RegisterQueueHandler("demo-queue", func(ctx transire.Context, msg transire.Message) error {
		received <- msg
		return nil
	})

	server := httptest.NewServer(buildHandler(app))
	t.Cleanup(server.Close)

	res, err := http.Post(server.URL+"/_transire/queues/demo-queue", "application/octet-stream", strings.NewReader("payload"))
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", res.StatusCode)
	}

	select {
	case msg := <-received:
		if msg.Queue != "demo-queue" || string(msg.Body) != "payload" {
			t.Fatalf("unexpected message: %#v", msg)
		}
	case <-time.After(time.Second):
		t.Fatalf("queue handler not invoked")
	}
}

func TestAdminScheduleEndpointInvokesHandler(t *testing.T) {
	app := transire.New()
	ran := make(chan time.Time, 1)

	app.RegisterScheduleHandler("tick", time.Minute, func(ctx transire.Context, at time.Time) error {
		ran <- at
		return nil
	})

	server := httptest.NewServer(buildHandler(app))
	t.Cleanup(server.Close)

	res, err := http.Post(server.URL+"/_transire/schedules/tick", "application/json", nil)
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("unexpected status: %d (%s)", res.StatusCode, string(body))
	}

	select {
	case at := <-ran:
		if time.Since(at) > time.Second {
			t.Fatalf("unexpected trigger time: %s", at)
		}
	case <-time.After(time.Second):
		t.Fatalf("schedule handler not invoked")
	}
}
