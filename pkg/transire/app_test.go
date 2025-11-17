package transire

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Run("creates app with default config", func(t *testing.T) {
		app := New()

		if app == nil {
			t.Fatal("New() returned nil")
		}

		if app.router == nil {
			t.Error("App router is nil")
		}

		if app.config == nil {
			t.Error("App config is nil")
		}

		// Check default config values
		if app.config.Language != "go" {
			t.Errorf("Expected default language 'go', got '%s'", app.config.Language)
		}

		if app.config.Cloud != "aws" {
			t.Errorf("Expected default cloud 'aws', got '%s'", app.config.Cloud)
		}
	})

	t.Run("creates app with custom config", func(t *testing.T) {
		config := &Config{
			Name:     "test-app",
			Language: "go",
			Cloud:    "aws",
			Runtime:  "lambda",
		}

		app := New(WithConfig(config))

		if app.config.Name != "test-app" {
			t.Errorf("Expected config name 'test-app', got '%s'", app.config.Name)
		}
	})
}

func TestAppRouter(t *testing.T) {
	app := New()
	router := app.Router()

	if router == nil {
		t.Fatal("Router() returned nil")
	}

	// Router should be usable
	// Just verify we can call methods on it without panicking
	router.Use(func(next http.Handler) http.Handler {
		return next
	})
}

func TestRegisterQueueHandler(t *testing.T) {
	app := New()

	handler := &testQueueHandler{
		queueName: "test-queue",
	}

	app.RegisterQueueHandler(handler)

	if len(app.queueHandlers) != 1 {
		t.Errorf("Expected 1 queue handler, got %d", len(app.queueHandlers))
	}

	if app.queueHandlers[0] != handler {
		t.Error("Queue handler not registered correctly")
	}
}

func TestRegisterScheduleHandler(t *testing.T) {
	app := New()

	handler := &testScheduleHandler{
		name: "test-schedule",
		schedule: "0 * * * *",
	}

	app.RegisterScheduleHandler(handler)

	if len(app.schedHandlers) != 1 {
		t.Errorf("Expected 1 schedule handler, got %d", len(app.schedHandlers))
	}

	if app.schedHandlers[0] != handler {
		t.Error("Schedule handler not registered correctly")
	}
}

func TestFindQueueHandler(t *testing.T) {
	app := New()

	handler1 := &testQueueHandler{queueName: "queue1"}
	handler2 := &testQueueHandler{queueName: "queue2"}

	app.RegisterQueueHandler(handler1)
	app.RegisterQueueHandler(handler2)

	found := app.FindQueueHandler("queue1")
	if found != handler1 {
		t.Error("FindQueueHandler did not return correct handler")
	}

	found = app.FindQueueHandler("nonexistent")
	if found != nil {
		t.Error("FindQueueHandler should return nil for nonexistent queue")
	}
}

func TestFindScheduleHandler(t *testing.T) {
	app := New()

	handler1 := &testScheduleHandler{name: "schedule1"}
	handler2 := &testScheduleHandler{name: "schedule2"}

	app.RegisterScheduleHandler(handler1)
	app.RegisterScheduleHandler(handler2)

	found := app.FindScheduleHandler("schedule1")
	if found != handler1 {
		t.Error("FindScheduleHandler did not return correct handler")
	}

	found = app.FindScheduleHandler("nonexistent")
	if found != nil {
		t.Error("FindScheduleHandler should return nil for nonexistent schedule")
	}
}

// Test helper implementations
type testQueueHandler struct {
	queueName string
}

func (h *testQueueHandler) HandleMessages(ctx context.Context, messages []Message) ([]string, error) {
	return nil, nil
}

func (h *testQueueHandler) QueueName() string {
	return h.queueName
}

func (h *testQueueHandler) Config() QueueConfig {
	return QueueConfig{
		VisibilityTimeoutSeconds: 30,
		MaxReceiveCount:         3,
		BatchSize:              10,
	}
}

type testScheduleHandler struct {
	name     string
	schedule string
}

func (h *testScheduleHandler) HandleSchedule(ctx context.Context, event ScheduleEvent) error {
	return nil
}

func (h *testScheduleHandler) Schedule() string {
	return h.schedule
}

func (h *testScheduleHandler) Name() string {
	return h.name
}

func (h *testScheduleHandler) Config() ScheduleConfig {
	return ScheduleConfig{
		Timezone:       "UTC",
		Enabled:        true,
		TimeoutSeconds: 60,
	}
}

type testHTTPHandler struct {
	path    string
	methods []string
}

func (h *testHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (h *testHTTPHandler) Metadata() HTTPHandlerMetadata {
	return HTTPHandlerMetadata{
		Methods: h.methods,
		Path:    h.path,
		Timeout: 30 * time.Second,
	}
}