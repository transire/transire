//go:build local

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package local

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/transire/transire/pkg/transire"
)

// TestLocalDevEndpoints_QueueMessage tests sending a queue message via dev endpoint
func TestLocalDevEndpoints_QueueMessage(t *testing.T) {
	// Create app with queue handler and custom port
	config := &transire.Config{}
	config.SetDefaults()
	config.Development.HTTPPort = 3101

	app := transire.New(transire.WithConfig(config))

	// Track if handler was called (use channel for synchronization)
	handlerCalled := make(chan transire.Message, 1)

	// Register a test queue handler
	testHandler := &callbackQueueHandler{
		queueName: "test-queue",
		handleFunc: func(ctx context.Context, messages []transire.Message) ([]string, error) {
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
	time.Sleep(200 * time.Millisecond)

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

	resp, err := http.Post("http://localhost:3101/__dev/queues/send", "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		t.Fatalf("Failed to send dev queue message: %v", err)
	}
	defer resp.Body.Close()

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
	config := &transire.Config{}
	config.SetDefaults()
	config.Development.HTTPPort = 3102

	app := transire.New(transire.WithConfig(config))

	// Track if handler was called (use channel for synchronization)
	handlerCalled := make(chan transire.ScheduleEvent, 1)

	// Register a test schedule handler
	testHandler := &callbackScheduleHandler{
		name:     "test-schedule",
		schedule: "0 * * * *",
		handleFunc: func(ctx context.Context, event transire.ScheduleEvent) error {
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
	time.Sleep(200 * time.Millisecond)

	// Execute the schedule via dev endpoint
	req := devScheduleExecuteRequest{
		ScheduleName: "test-schedule",
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := http.Post("http://localhost:3102/__dev/schedules/execute", "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		t.Fatalf("Failed to execute dev schedule: %v", err)
	}
	defer resp.Body.Close()

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

// TestLocalDevEndpoints_ListQueues tests listing queues via dev endpoint
func TestLocalDevEndpoints_ListQueues(t *testing.T) {
	config := &transire.Config{}
	config.SetDefaults()
	config.Development.HTTPPort = 3103

	app := transire.New(transire.WithConfig(config))

	// Register multiple queue handlers
	app.RegisterQueueHandler(&callbackQueueHandler{queueName: "queue-1"})
	app.RegisterQueueHandler(&callbackQueueHandler{queueName: "queue-2"})

	// Start the app in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = app.Run(ctx)
	}()

	time.Sleep(200 * time.Millisecond)

	resp, err := http.Get("http://localhost:3103/__dev/queues/list")
	if err != nil {
		t.Fatalf("Failed to list queues: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	queues, ok := result["queues"].([]interface{})
	if !ok {
		t.Fatal("Expected 'queues' array in response")
	}

	if len(queues) != 2 {
		t.Errorf("Expected 2 queues, got %d", len(queues))
	}
}

// TestLocalDevEndpoints_ListSchedules tests listing schedules via dev endpoint
func TestLocalDevEndpoints_ListSchedules(t *testing.T) {
	config := &transire.Config{}
	config.SetDefaults()
	config.Development.HTTPPort = 3104

	app := transire.New(transire.WithConfig(config))

	// Register multiple schedule handlers
	app.RegisterScheduleHandler(&callbackScheduleHandler{name: "schedule-1", schedule: "0 * * * *"})
	app.RegisterScheduleHandler(&callbackScheduleHandler{name: "schedule-2", schedule: "0 0 * * *"})

	// Start the app in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = app.Run(ctx)
	}()

	time.Sleep(200 * time.Millisecond)

	resp, err := http.Get("http://localhost:3104/__dev/schedules/list")
	if err != nil {
		t.Fatalf("Failed to list schedules: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	schedules, ok := result["schedules"].([]interface{})
	if !ok {
		t.Fatal("Expected 'schedules' array in response")
	}

	if len(schedules) != 2 {
		t.Errorf("Expected 2 schedules, got %d", len(schedules))
	}
}

// Test helper types with callbacks
type callbackQueueHandler struct {
	queueName  string
	handleFunc func(context.Context, []transire.Message) ([]string, error)
}

func (h *callbackQueueHandler) QueueName() string {
	return h.queueName
}

func (h *callbackQueueHandler) Config() transire.QueueConfig {
	return transire.QueueConfig{
		BatchSize:                10,
		VisibilityTimeoutSeconds: 30,
		MaxReceiveCount:          3,
	}
}

func (h *callbackQueueHandler) HandleMessages(ctx context.Context, messages []transire.Message) ([]string, error) {
	if h.handleFunc != nil {
		return h.handleFunc(ctx, messages)
	}
	return nil, nil
}

type callbackScheduleHandler struct {
	name       string
	schedule   string
	handleFunc func(context.Context, transire.ScheduleEvent) error
}

func (h *callbackScheduleHandler) Name() string {
	return h.name
}

func (h *callbackScheduleHandler) Schedule() string {
	return h.schedule
}

func (h *callbackScheduleHandler) Config() transire.ScheduleConfig {
	return transire.ScheduleConfig{
		Enabled:        true,
		Timezone:       "UTC",
		TimeoutSeconds: 300,
	}
}

func (h *callbackScheduleHandler) HandleSchedule(ctx context.Context, event transire.ScheduleEvent) error {
	if h.handleFunc != nil {
		return h.handleFunc(ctx, event)
	}
	return nil
}

// Dev API request types
type devQueueMessageRequest struct {
	QueueName string `json:"queue_name"`
	Message   string `json:"message"`
}

type devScheduleExecuteRequest struct {
	ScheduleName string `json:"schedule_name"`
}
