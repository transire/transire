package transire

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// TestQueueConfigYAMLUnmarshaling tests that QueueConfig correctly unmarshals from YAML
func TestQueueConfigYAMLUnmarshaling(t *testing.T) {
	yamlContent := `
todo-reminders:
  visibility_timeout_seconds: 30
  max_receive_count: 3
  batch_size: 10
  wait_time_seconds: 5
  fifo: false
`

	var queues map[string]QueueConfig
	err := yaml.Unmarshal([]byte(yamlContent), &queues)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	config, ok := queues["todo-reminders"]
	if !ok {
		t.Fatal("Expected 'todo-reminders' queue to be present")
	}

	// Test that values are correctly unmarshaled from snake_case YAML
	if config.VisibilityTimeoutSeconds != 30 {
		t.Errorf("Expected VisibilityTimeoutSeconds to be 30, got %d", config.VisibilityTimeoutSeconds)
	}
	if config.MaxReceiveCount != 3 {
		t.Errorf("Expected MaxReceiveCount to be 3, got %d", config.MaxReceiveCount)
	}
	if config.BatchSize != 10 {
		t.Errorf("Expected BatchSize to be 10, got %d", config.BatchSize)
	}
	if config.WaitTimeSeconds != 5 {
		t.Errorf("Expected WaitTimeSeconds to be 5, got %d", config.WaitTimeSeconds)
	}
	if config.FIFO != false {
		t.Errorf("Expected FIFO to be false, got %t", config.FIFO)
	}
}

// TestScheduleConfigYAMLUnmarshaling tests that ScheduleConfig correctly unmarshals from YAML
func TestScheduleConfigYAMLUnmarshaling(t *testing.T) {
	yamlContent := `
cleanup-completed-todos:
  timezone: "UTC"
  enabled: true
  timeout_seconds: 300
  retry_attempts: 3
`

	var schedules map[string]ScheduleConfig
	err := yaml.Unmarshal([]byte(yamlContent), &schedules)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	config, ok := schedules["cleanup-completed-todos"]
	if !ok {
		t.Fatal("Expected 'cleanup-completed-todos' schedule to be present")
	}

	// Test that values are correctly unmarshaled from snake_case YAML
	if config.Timezone != "UTC" {
		t.Errorf("Expected Timezone to be 'UTC', got %s", config.Timezone)
	}
	if config.Enabled != true {
		t.Errorf("Expected Enabled to be true, got %t", config.Enabled)
	}
	if config.TimeoutSeconds != 300 {
		t.Errorf("Expected TimeoutSeconds to be 300, got %d", config.TimeoutSeconds)
	}
	if config.RetryAttempts != 3 {
		t.Errorf("Expected RetryAttempts to be 3, got %d", config.RetryAttempts)
	}
}

// TestFullConfigUnmarshaling tests that a complete transire.yaml config unmarshals correctly
func TestFullConfigUnmarshaling(t *testing.T) {
	yamlContent := `
name: todo-app
language: go
cloud: aws
runtime: lambda

queues:
  todo-reminders:
    visibility_timeout_seconds: 30
    max_receive_count: 3
    batch_size: 10
  todo-notifications:
    visibility_timeout_seconds: 60
    max_receive_count: 5
    batch_size: 20

schedules:
  cleanup-completed-todos:
    timezone: "UTC"
    enabled: true
  daily-todo-summary:
    timezone: "UTC"
    enabled: true

development:
  http_port: 3000
  queue_port: 4000
`

	var config Config
	err := yaml.Unmarshal([]byte(yamlContent), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	// Test queue configs
	if len(config.Queues) != 2 {
		t.Errorf("Expected 2 queues, got %d", len(config.Queues))
	}

	reminderQueue := config.Queues["todo-reminders"]
	if reminderQueue.VisibilityTimeoutSeconds != 30 {
		t.Errorf("Expected todo-reminders VisibilityTimeoutSeconds to be 30, got %d", reminderQueue.VisibilityTimeoutSeconds)
	}
	if reminderQueue.BatchSize != 10 {
		t.Errorf("Expected todo-reminders BatchSize to be 10, got %d", reminderQueue.BatchSize)
	}

	notificationQueue := config.Queues["todo-notifications"]
	if notificationQueue.VisibilityTimeoutSeconds != 60 {
		t.Errorf("Expected todo-notifications VisibilityTimeoutSeconds to be 60, got %d", notificationQueue.VisibilityTimeoutSeconds)
	}
	if notificationQueue.BatchSize != 20 {
		t.Errorf("Expected todo-notifications BatchSize to be 20, got %d", notificationQueue.BatchSize)
	}

	// Test schedule configs
	if len(config.Schedules) != 2 {
		t.Errorf("Expected 2 schedules, got %d", len(config.Schedules))
	}

	cleanupSchedule := config.Schedules["cleanup-completed-todos"]
	if cleanupSchedule.Timezone != "UTC" {
		t.Errorf("Expected cleanup-completed-todos Timezone to be 'UTC', got %s", cleanupSchedule.Timezone)
	}
	if cleanupSchedule.Enabled != true {
		t.Errorf("Expected cleanup-completed-todos Enabled to be true, got %t", cleanupSchedule.Enabled)
	}
}
