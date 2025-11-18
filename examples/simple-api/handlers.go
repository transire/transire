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

// EmailQueueHandler processes email sending requests
type EmailQueueHandler struct{}

func (h *EmailQueueHandler) QueueName() string {
	return "email-queue"
}

func (h *EmailQueueHandler) Config() transire.QueueConfig {
	return transire.QueueConfig{
		VisibilityTimeoutSeconds: 30,
		MaxReceiveCount:          3,
		BatchSize:                10,
		WaitTimeSeconds:          5, // Long polling
	}
}

func (h *EmailQueueHandler) HandleMessages(ctx context.Context, messages []transire.Message) ([]string, error) {
	log.Printf("Processing %d email messages", len(messages))

	var failedIDs []string

	for _, msg := range messages {
		var emailReq EmailRequest
		if err := json.Unmarshal(msg.Body(), &emailReq); err != nil {
			log.Printf("Failed to parse email request from message %s: %v", msg.ID(), err)
			// Skip malformed messages (don't retry)
			continue
		}

		if err := sendEmail(emailReq); err != nil {
			log.Printf("Failed to send email for message %s: %v", msg.ID(), err)
			failedIDs = append(failedIDs, msg.ID())
		} else {
			log.Printf("Successfully sent email to %s (message %s)", emailReq.To, msg.ID())
		}
	}

	return failedIDs, nil
}

// NotificationQueueHandler processes push notifications
type NotificationQueueHandler struct{}

func (h *NotificationQueueHandler) QueueName() string {
	return "notification-queue"
}

func (h *NotificationQueueHandler) Config() transire.QueueConfig {
	return transire.QueueConfig{
		VisibilityTimeoutSeconds: 60,
		MaxReceiveCount:          5,
		BatchSize:                5,
	}
}

func (h *NotificationQueueHandler) HandleMessages(ctx context.Context, messages []transire.Message) ([]string, error) {
	log.Printf("Processing %d notification messages", len(messages))

	var failedIDs []string

	for _, msg := range messages {
		var notificationReq NotificationRequest
		if err := json.Unmarshal(msg.Body(), &notificationReq); err != nil {
			log.Printf("Failed to parse notification request from message %s: %v", msg.ID(), err)
			continue
		}

		if err := sendNotification(notificationReq); err != nil {
			log.Printf("Failed to send notification for message %s: %v", msg.ID(), err)
			failedIDs = append(failedIDs, msg.ID())
		} else {
			log.Printf("Successfully sent notification to %s (message %s)", notificationReq.UserID, msg.ID())
		}
	}

	return failedIDs, nil
}

// CleanupHandler runs scheduled maintenance tasks
type CleanupHandler struct{}

func (h *CleanupHandler) Name() string {
	return "daily-cleanup"
}

func (h *CleanupHandler) Schedule() string {
	return "0 2 * * *" // Daily at 2 AM UTC
}

func (h *CleanupHandler) Config() transire.ScheduleConfig {
	return transire.ScheduleConfig{
		Timezone:       "UTC",
		Enabled:        true,
		TimeoutSeconds: 300, // 5 minutes
		RetryAttempts:  3,
		RetryDelay:     30 * time.Second,
	}
}

func (h *CleanupHandler) HandleSchedule(ctx context.Context, event transire.ScheduleEvent) error {
	log.Printf("Starting daily cleanup at %v", event.ScheduledTime)

	// Cleanup old temporary files
	if err := cleanupTempFiles(); err != nil {
		return fmt.Errorf("failed to cleanup temp files: %w", err)
	}

	// Cleanup expired sessions
	if err := cleanupExpiredSessions(); err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	// Cleanup old logs
	if err := cleanupOldLogs(); err != nil {
		return fmt.Errorf("failed to cleanup old logs: %w", err)
	}

	log.Println("Daily cleanup completed successfully")
	return nil
}

// Message types
type EmailRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	From    string `json:"from,omitempty"`
}

type NotificationRequest struct {
	UserID  string `json:"user_id"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Type    string `json:"type"` // push, sms, slack, etc.
}

// Mock implementations for demo
func sendEmail(req EmailRequest) error {
	// Simulate email sending
	time.Sleep(100 * time.Millisecond)

	// Simulate occasional failures
	if req.To == "fail@example.com" {
		return fmt.Errorf("email service unavailable")
	}

	return nil
}

func sendNotification(req NotificationRequest) error {
	// Simulate notification sending
	time.Sleep(50 * time.Millisecond)

	// Simulate occasional failures
	if req.UserID == "fail-user" {
		return fmt.Errorf("notification service unavailable")
	}

	return nil
}

func cleanupTempFiles() error {
	log.Println("Cleaning up temporary files...")
	time.Sleep(500 * time.Millisecond)
	return nil
}

func cleanupExpiredSessions() error {
	log.Println("Cleaning up expired sessions...")
	time.Sleep(300 * time.Millisecond)
	return nil
}

func cleanupOldLogs() error {
	log.Println("Cleaning up old logs...")
	time.Sleep(200 * time.Millisecond)
	return nil
}
