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

	// Track if handler was called
	handlerCalled := false
	var receivedMessage Message

	// Register a test queue handler
	testHandler := &callbackQueueHandler{
		queueName: "test-queue",
		handleFunc: func(ctx context.Context, messages []Message) ([]string, error) {
			handlerCalled = true
			if len(messages) > 0 {
				receivedMessage = messages[0]
			}
			return nil, nil
		},
	}
	app.RegisterQueueHandler(testHandler)

	// Start the app in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = app.Run(ctx)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Send a test message via dev endpoint
	messageBody := map[string]string{
		"test": "data",
	}
	bodyJSON, _ := json.Marshal(messageBody)

	req := devQueueMessageRequest{
		QueueName: "test-queue",
		Message:   string(bodyJSON),
	}
	reqJSON, _ := json.Marshal(req)

	resp, err := http.Post("http://localhost:3001/__dev/queues/send", "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		t.Fatalf("Failed to send dev queue message: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Wait for async processing
	time.Sleep(100 * time.Millisecond)

	// Verify handler was called
	if !handlerCalled {
		t.Error("Queue handler was not called")
	}

	if receivedMessage == nil {
		t.Error("No message received by handler")
	} else if string(receivedMessage.Body()) != string(bodyJSON) {
		t.Errorf("Expected message body %s, got %s", bodyJSON, receivedMessage.Body())
	}
}

// TestLocalDevEndpoints_ScheduleExecution tests executing a schedule via dev endpoint
func TestLocalDevEndpoints_ScheduleExecution(t *testing.T) {
	// Create app with schedule handler and custom port
	config := &Config{}
	config.setDefaults()
	config.Development.HTTPPort = 3002
	app := New(WithConfig(config))

	// Track if handler was called
	handlerCalled := false
	var receivedEvent ScheduleEvent

	// Register a test schedule handler
	testHandler := &callbackScheduleHandler{
		name:     "test-schedule",
		schedule: "0 * * * *",
		handleFunc: func(ctx context.Context, event ScheduleEvent) error {
			handlerCalled = true
			receivedEvent = event
			return nil
		},
	}
	app.RegisterScheduleHandler(testHandler)

	// Start the app in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = app.Run(ctx)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Execute the schedule via dev endpoint
	req := devScheduleExecuteRequest{
		ScheduleName: "test-schedule",
	}
	reqJSON, _ := json.Marshal(req)

	resp, err := http.Post("http://localhost:3002/__dev/schedules/execute", "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		t.Fatalf("Failed to execute dev schedule: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Wait for async processing
	time.Sleep(100 * time.Millisecond)

	// Verify handler was called
	if !handlerCalled {
		t.Error("Schedule handler was not called")
	}

	if receivedEvent.Name != "test-schedule" {
		t.Errorf("Expected event name 'test-schedule', got '%s'", receivedEvent.Name)
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
