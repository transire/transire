package transire

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestLocalDevEndpoints_QueueMessage tests sending a queue message via dev endpoint
func TestLocalDevEndpoints_QueueMessage(t *testing.T) {
	// Create app with queue handler and custom port
	config := &Config{}
	config.setDefaults()
	config.Development.HTTPPort = 3001
	app := New(WithConfig(config))

	// Track if handler was called (use channel for synchronization)
	handlerCalled := make(chan Message, 1)

	// Register a test queue handler
	testHandler := &callbackQueueHandler{
		queueName: "test-queue",
		handleFunc: func(ctx context.Context, messages []Message) ([]string, error) {
			if len(messages) > 0 {
				handlerCalled <- messages[0]
			}
			return nil, nil
		},
	}
	app.RegisterQueueHandler(testHandler)

	// Start the app in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := app.Run(ctx); err != nil && ctx.Err() == nil {
			t.Errorf("App.Run failed: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Send a test message via dev endpoint
	messageBody := map[string]string{
		"test": "data",
	}
	bodyJSON, err := json.Marshal(messageBody)
	if err != nil {
		t.Fatalf("Failed to marshal message body: %v", err)
	}

	req := devQueueMessageRequest{
		QueueName: "test-queue",
		Message:   string(bodyJSON),
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := http.Post("http://localhost:3001/__dev/queues/send", "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		t.Fatalf("Failed to send dev queue message: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Wait for handler to be called
	select {
	case receivedMessage := <-handlerCalled:
		if string(receivedMessage.Body()) != string(bodyJSON) {
			t.Errorf("Expected message body %s, got %s", bodyJSON, receivedMessage.Body())
		}
	case <-time.After(2 * time.Second):
		t.Error("Queue handler was not called within timeout")
	}
}

// TestLocalDevEndpoints_ScheduleExecution tests executing a schedule via dev endpoint
func TestLocalDevEndpoints_ScheduleExecution(t *testing.T) {
	// Create app with schedule handler and custom port
	config := &Config{}
	config.setDefaults()
	config.Development.HTTPPort = 3002
	app := New(WithConfig(config))

	// Track if handler was called (use channel for synchronization)
	handlerCalled := make(chan ScheduleEvent, 1)

	// Register a test schedule handler
	testHandler := &callbackScheduleHandler{
		name:     "test-schedule",
		schedule: "0 * * * *",
		handleFunc: func(ctx context.Context, event ScheduleEvent) error {
			handlerCalled <- event
			return nil
		},
	}
	app.RegisterScheduleHandler(testHandler)

	// Start the app in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := app.Run(ctx); err != nil && ctx.Err() == nil {
			t.Errorf("App.Run failed: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Execute the schedule via dev endpoint
	req := devScheduleExecuteRequest{
		ScheduleName: "test-schedule",
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := http.Post("http://localhost:3002/__dev/schedules/execute", "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		t.Fatalf("Failed to execute dev schedule: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Wait for handler to be called
	select {
	case receivedEvent := <-handlerCalled:
		if receivedEvent.Name != "test-schedule" {
			t.Errorf("Expected event name 'test-schedule', got '%s'", receivedEvent.Name)
		}
	case <-time.After(2 * time.Second):
		t.Error("Schedule handler was not called within timeout")
	}
}

// Test helper types with callbacks
type callbackQueueHandler struct {
	queueName  string
	handleFunc func(context.Context, []Message) ([]string, error)
}

func (h *callbackQueueHandler) QueueName() string {
	return h.queueName
}

func (h *callbackQueueHandler) Config() QueueConfig {
	return QueueConfig{
		BatchSize:                10,
		VisibilityTimeoutSeconds: 30,
		MaxReceiveCount:          3,
	}
}

func (h *callbackQueueHandler) HandleMessages(ctx context.Context, messages []Message) ([]string, error) {
	return h.handleFunc(ctx, messages)
}

type callbackScheduleHandler struct {
	name       string
	schedule   string
	handleFunc func(context.Context, ScheduleEvent) error
}

func (h *callbackScheduleHandler) Name() string {
	return h.name
}

func (h *callbackScheduleHandler) Schedule() string {
	return h.schedule
}

func (h *callbackScheduleHandler) Config() ScheduleConfig {
	return ScheduleConfig{
		Enabled:        true,
		Timezone:       "UTC",
		TimeoutSeconds: 300,
	}
}

func (h *callbackScheduleHandler) HandleSchedule(ctx context.Context, event ScheduleEvent) error {
	return h.handleFunc(ctx, event)
}

// Dev API request types
type devQueueMessageRequest struct {
	QueueName string `json:"queue_name"`
	Message   string `json:"message"`
}

type devScheduleExecuteRequest struct {
	ScheduleName string `json:"schedule_name"`
}
