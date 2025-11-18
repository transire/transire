// Copyright (c) 2024 Transire Contributors
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/transire/transire/pkg/transire"
)

// TodoReminderQueue sends reminders for todos approaching their due date
type TodoReminderQueue struct {
	store *TodoStore
}

func (h *TodoReminderQueue) QueueName() string {
	return "todo-reminders"
}

func (h *TodoReminderQueue) Config() transire.QueueConfig {
	return transire.QueueConfig{
		VisibilityTimeoutSeconds: 30,
		MaxReceiveCount:          3,
		BatchSize:                10,
		WaitTimeSeconds:          5,
	}
}

func (h *TodoReminderQueue) HandleMessages(ctx context.Context, messages []transire.Message) ([]string, error) {
	log.Printf("Processing %d todo reminder messages", len(messages))

	var failedIDs []string

	for _, msg := range messages {
		var reminder TodoReminderMessage
		if err := json.Unmarshal(msg.Body(), &reminder); err != nil {
			log.Printf("Failed to parse reminder from message %s: %v", msg.ID(), err)
			continue
		}

		// Get the todo from store
		todo, ok := h.store.Get(reminder.TodoID)
		if !ok {
			log.Printf("Todo %s not found for reminder %s", reminder.TodoID, msg.ID())
			continue
		}

		// Send reminder (mock implementation)
		if err := sendTodoReminder(todo, reminder.UserEmail); err != nil {
			log.Printf("Failed to send reminder for todo %s (message %s): %v", todo.ID, msg.ID(), err)
			failedIDs = append(failedIDs, msg.ID())
		} else {
			log.Printf("Successfully sent reminder for todo '%s' to %s (message %s)", todo.Title, reminder.UserEmail, msg.ID())
		}
	}

	return failedIDs, nil
}

// TodoNotificationQueue sends notifications when todos are created, updated, or completed
type TodoNotificationQueue struct {
	store *TodoStore
}

func (h *TodoNotificationQueue) QueueName() string {
	return "todo-notifications"
}

func (h *TodoNotificationQueue) Config() transire.QueueConfig {
	return transire.QueueConfig{
		VisibilityTimeoutSeconds: 60,
		MaxReceiveCount:          5,
		BatchSize:                20,
	}
}

func (h *TodoNotificationQueue) HandleMessages(ctx context.Context, messages []transire.Message) ([]string, error) {
	log.Printf("Processing %d todo notification messages", len(messages))

	var failedIDs []string

	for _, msg := range messages {
		var notification TodoNotificationMessage
		if err := json.Unmarshal(msg.Body(), &notification); err != nil {
			log.Printf("Failed to parse notification from message %s: %v", msg.ID(), err)
			continue
		}

		if err := sendTodoNotification(notification); err != nil {
			log.Printf("Failed to send notification for message %s: %v", msg.ID(), err)
			failedIDs = append(failedIDs, msg.ID())
		} else {
			log.Printf("Successfully sent %s notification for todo %s to %s (message %s)",
				notification.EventType, notification.TodoID, notification.UserEmail, msg.ID())
		}
	}

	return failedIDs, nil
}

// CleanupCompletedTodosSchedule removes completed todos older than 30 days
type CleanupCompletedTodosSchedule struct {
	store *TodoStore
}

func (h *CleanupCompletedTodosSchedule) Name() string {
	return "cleanup-completed-todos"
}

func (h *CleanupCompletedTodosSchedule) Schedule() string {
	return "0 3 * * *" // Daily at 3 AM UTC
}

func (h *CleanupCompletedTodosSchedule) Config() transire.ScheduleConfig {
	return transire.ScheduleConfig{
		Timezone:       "UTC",
		Enabled:        true,
		TimeoutSeconds: 300,
		RetryAttempts:  3,
		RetryDelay:     30 * time.Second,
	}
}

func (h *CleanupCompletedTodosSchedule) HandleSchedule(ctx context.Context, event transire.ScheduleEvent) error {
	log.Printf("Starting cleanup of completed todos at %v", event.ScheduledTime)

	cutoffDate := time.Now().AddDate(0, 0, -30) // 30 days ago
	todos := h.store.List()
	deletedCount := 0

	for _, todo := range todos {
		if todo.Completed && todo.UpdatedAt.Before(cutoffDate) {
			h.store.Delete(todo.ID)
			deletedCount++
			log.Printf("Deleted completed todo: %s (completed on %v)", todo.Title, todo.UpdatedAt)
		}
	}

	log.Printf("Cleanup completed: deleted %d old completed todos", deletedCount)
	return nil
}

// DailyTodoSummarySchedule generates a daily summary of pending todos
type DailyTodoSummarySchedule struct {
	store *TodoStore
}

func (h *DailyTodoSummarySchedule) Name() string {
	return "daily-todo-summary"
}

func (h *DailyTodoSummarySchedule) Schedule() string {
	return "0 9 * * *" // Daily at 9 AM UTC
}

func (h *DailyTodoSummarySchedule) Config() transire.ScheduleConfig {
	return transire.ScheduleConfig{
		Timezone:       "UTC",
		Enabled:        true,
		TimeoutSeconds: 120,
		RetryAttempts:  2,
		RetryDelay:     15 * time.Second,
	}
}

func (h *DailyTodoSummarySchedule) HandleSchedule(ctx context.Context, event transire.ScheduleEvent) error {
	log.Printf("Generating daily todo summary at %v", event.ScheduledTime)

	todos := h.store.List()
	var pendingCount, completedCount, overdueCount int

	now := time.Now()
	for _, todo := range todos {
		if todo.Completed {
			completedCount++
		} else {
			pendingCount++
			if !todo.DueDate.IsZero() && todo.DueDate.Before(now) {
				overdueCount++
			}
		}
	}

	summary := TodoSummary{
		TotalTodos:     len(todos),
		PendingTodos:   pendingCount,
		CompletedTodos: completedCount,
		OverdueTodos:   overdueCount,
		GeneratedAt:    time.Now(),
	}

	log.Printf("Daily Summary - Total: %d, Pending: %d, Completed: %d, Overdue: %d",
		summary.TotalTodos, summary.PendingTodos, summary.CompletedTodos, summary.OverdueTodos)

	// In a real system, we would send this summary via email or store it
	return nil
}

// Message types
type TodoReminderMessage struct {
	TodoID    string `json:"todo_id"`
	UserEmail string `json:"user_email"`
}

type TodoNotificationMessage struct {
	TodoID    string `json:"todo_id"`
	UserEmail string `json:"user_email"`
	EventType string `json:"event_type"` // created, updated, completed
	TodoTitle string `json:"todo_title"`
}

type TodoSummary struct {
	TotalTodos     int       `json:"total_todos"`
	PendingTodos   int       `json:"pending_todos"`
	CompletedTodos int       `json:"completed_todos"`
	OverdueTodos   int       `json:"overdue_todos"`
	GeneratedAt    time.Time `json:"generated_at"`
}

// Mock implementations
func sendTodoReminder(todo *Todo, userEmail string) error {
	// Simulate reminder sending
	time.Sleep(50 * time.Millisecond)

	// Simulate occasional failures
	if userEmail == "fail-reminder@example.com" {
		return fmt.Errorf("reminder service unavailable")
	}

	return nil
}

func sendTodoNotification(notification TodoNotificationMessage) error {
	// Simulate notification sending
	time.Sleep(30 * time.Millisecond)

	// Simulate occasional failures
	if notification.UserEmail == "fail-notification@example.com" {
		return fmt.Errorf("notification service unavailable")
	}

	return nil
}
