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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const (
	baseURL = "http://localhost:3000"
	cliPath = "../../transire-cli"
)

// TestE2E_HTTPHandlers tests all HTTP endpoints
func TestE2E_HTTPHandlers(t *testing.T) {
	t.Run("Health Check", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/health")
		if err != nil {
			t.Fatalf("Failed to call /health: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", result["status"])
		}

		if result["service"] != "transire-simple-api" {
			t.Errorf("Expected service 'transire-simple-api', got '%s'", result["service"])
		}
	})

	t.Run("Home Endpoint", func(t *testing.T) {
		resp, err := http.Get(baseURL)
		if err != nil {
			t.Fatalf("Failed to call /: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result["message"] != "Welcome to Transire Simple API" {
			t.Errorf("Expected welcome message, got '%s'", result["message"])
		}

		if result["version"] != "1.0.0" {
			t.Errorf("Expected version '1.0.0', got '%s'", result["version"])
		}
	})

	t.Run("CRUD User Operations", func(t *testing.T) {
		// Test Create User
		user := map[string]string{
			"name":  "John Doe",
			"email": "john@example.com",
		}
		userJSON, _ := json.Marshal(user)

		resp, err := http.Post(baseURL+"/api/v1/users", "application/json", bytes.NewBuffer(userJSON))
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", resp.StatusCode)
		}

		var createdUser User
		if err := json.NewDecoder(resp.Body).Decode(&createdUser); err != nil {
			t.Fatalf("Failed to decode created user: %v", err)
		}

		if createdUser.Name != "John Doe" {
			t.Errorf("Expected name 'John Doe', got '%s'", createdUser.Name)
		}

		if createdUser.Email != "john@example.com" {
			t.Errorf("Expected email 'john@example.com', got '%s'", createdUser.Email)
		}

		if createdUser.ID == "" {
			t.Error("Expected non-empty user ID")
		}

		// Test Get User
		resp, err = http.Get(baseURL + "/api/v1/users/123")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var fetchedUser User
		if err := json.NewDecoder(resp.Body).Decode(&fetchedUser); err != nil {
			t.Fatalf("Failed to decode fetched user: %v", err)
		}

		if fetchedUser.ID != "123" {
			t.Errorf("Expected ID '123', got '%s'", fetchedUser.ID)
		}

		// Test Update User
		updateData := map[string]string{
			"name":  "Jane Doe",
			"email": "jane@example.com",
		}
		updateJSON, _ := json.Marshal(updateData)

		req, _ := http.NewRequest("PUT", baseURL+"/api/v1/users/123", bytes.NewBuffer(updateJSON))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("Failed to update user: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Test Delete User
		req, _ = http.NewRequest("DELETE", baseURL+"/api/v1/users/123", nil)
		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("Failed to delete user: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", resp.StatusCode)
		}
	})
}

// TestE2E_QueueHandlers tests queue functionality using CLI
func TestE2E_QueueHandlers(t *testing.T) {
	// Change to the examples/simple-api directory for CLI commands
	workDir := "/Users/jamie/personal/transire2/examples/simple-api"

	t.Run("List Queues", func(t *testing.T) {
		cmd := exec.Command(cliPath, "dev", "queues", "list", "-c", "transire.yaml")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to list queues: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "email-queue") {
			t.Error("Expected 'email-queue' in output")
		}
		if !strings.Contains(outputStr, "notification-queue") {
			t.Error("Expected 'notification-queue' in output")
		}

		t.Logf("Queue list output: %s", outputStr)
	})

	t.Run("Send Email Queue Message", func(t *testing.T) {
		emailMessage := EmailRequest{
			To:      "test@example.com",
			Subject: "Test Email",
			Body:    "This is a test email",
			From:    "sender@example.com",
		}

		messageJSON, _ := json.Marshal(emailMessage)

		cmd := exec.Command(cliPath, "dev", "queues", "send", "-c", "transire.yaml", "email-queue", string(messageJSON))
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to send email queue message: %v, output: %s", err, string(output))
		}

		t.Logf("Email queue send output: %s", string(output))

		// Wait a bit for message processing
		time.Sleep(2 * time.Second)

		// Check the application logs to verify message was processed
		// (This would be more comprehensive in a real system with proper logging)
	})

	t.Run("Send Email Queue Message with Failure", func(t *testing.T) {
		// Test failure scenario
		emailMessage := EmailRequest{
			To:      "fail@example.com", // This will trigger the failure simulation
			Subject: "Test Email Failure",
			Body:    "This email should fail",
			From:    "sender@example.com",
		}

		messageJSON, _ := json.Marshal(emailMessage)

		cmd := exec.Command(cliPath, "dev", "queues", "send", "-c", "transire.yaml", "email-queue", string(messageJSON))
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to send failing email queue message: %v, output: %s", err, string(output))
		}

		t.Logf("Failing email queue send output: %s", string(output))

		// Wait for processing
		time.Sleep(2 * time.Second)
	})

	t.Run("Send Notification Queue Message", func(t *testing.T) {
		notificationMessage := NotificationRequest{
			UserID:  "user123",
			Title:   "Test Notification",
			Message: "This is a test notification",
			Type:    "push",
		}

		messageJSON, _ := json.Marshal(notificationMessage)

		cmd := exec.Command(cliPath, "dev", "queues", "send", "-c", "transire.yaml", "notification-queue", string(messageJSON))
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to send notification queue message: %v, output: %s", err, string(output))
		}

		t.Logf("Notification queue send output: %s", string(output))

		// Wait for processing
		time.Sleep(2 * time.Second)
	})

	t.Run("Send Notification Queue Message with Failure", func(t *testing.T) {
		// Test failure scenario
		notificationMessage := NotificationRequest{
			UserID:  "fail-user", // This will trigger the failure simulation
			Title:   "Test Notification Failure",
			Message: "This notification should fail",
			Type:    "push",
		}

		messageJSON, _ := json.Marshal(notificationMessage)

		cmd := exec.Command(cliPath, "dev", "queues", "send", "-c", "transire.yaml", "notification-queue", string(messageJSON))
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to send failing notification queue message: %v, output: %s", err, string(output))
		}

		t.Logf("Failing notification queue send output: %s", string(output))

		// Wait for processing
		time.Sleep(2 * time.Second)
	})
}

// TestE2E_ScheduleHandlers tests schedule functionality using CLI
func TestE2E_ScheduleHandlers(t *testing.T) {
	workDir := "/Users/jamie/personal/transire2/examples/simple-api"

	t.Run("List Schedules", func(t *testing.T) {
		cmd := exec.Command(cliPath, "dev", "schedules", "list", "-c", "transire.yaml")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to list schedules: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "daily-cleanup") {
			t.Error("Expected 'daily-cleanup' in output")
		}

		t.Logf("Schedule list output: %s", outputStr)
	})

	t.Run("Execute Schedule Manually", func(t *testing.T) {
		cmd := exec.Command(cliPath, "dev", "schedules", "execute", "-c", "transire.yaml", "daily-cleanup")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to execute schedule: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		t.Logf("Schedule execute output: %s", outputStr)

		// Wait for execution to complete
		time.Sleep(3 * time.Second)

		// The schedule execution should trigger the cleanup tasks
		// In a real system, we would verify that cleanup actually happened
	})
}

// TestE2E_IntegrationWorkflow tests a complete workflow
func TestE2E_IntegrationWorkflow(t *testing.T) {
	workDir := "/Users/jamie/personal/transire2/examples/simple-api"

	t.Run("Complete User Workflow with Notifications", func(t *testing.T) {
		// 1. Create a user via HTTP API
		user := map[string]string{
			"name":  "Integration Test User",
			"email": "integration@test.com",
		}
		userJSON, _ := json.Marshal(user)

		resp, err := http.Post(baseURL+"/api/v1/users", "application/json", bytes.NewBuffer(userJSON))
		if err != nil {
			t.Fatalf("Failed to create user in workflow: %v", err)
		}
		defer resp.Body.Close()

		var createdUser User
		json.NewDecoder(resp.Body).Decode(&createdUser)

		// 2. Send welcome email via queue
		welcomeEmail := EmailRequest{
			To:      createdUser.Email,
			Subject: "Welcome!",
			Body:    fmt.Sprintf("Welcome %s to our service!", createdUser.Name),
			From:    "noreply@example.com",
		}
		emailJSON, _ := json.Marshal(welcomeEmail)

		cmd := exec.Command(cliPath, "dev", "queues", "send", "-c", "transire.yaml", "email-queue", string(emailJSON))
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to send welcome email: %v, output: %s", err, string(output))
		}

		// 3. Send push notification via queue
		notification := NotificationRequest{
			UserID:  createdUser.ID,
			Title:   "Account Created",
			Message: "Your account has been successfully created",
			Type:    "push",
		}
		notificationJSON, _ := json.Marshal(notification)

		cmd = exec.Command(cliPath, "dev", "queues", "send", "-c", "transire.yaml", "notification-queue", string(notificationJSON))
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to send notification: %v, output: %s", err, string(output))
		}

		// 4. Wait for queue processing
		time.Sleep(3 * time.Second)

		// 5. Trigger cleanup (simulating maintenance)
		cmd = exec.Command(cliPath, "dev", "schedules", "execute", "-c", "transire.yaml", "daily-cleanup")
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to execute cleanup: %v, output: %s", err, string(output))
		}

		// 6. Update user
		updateData := map[string]string{
			"name":  "Updated Integration User",
			"email": "updated-integration@test.com",
		}
		updateJSON, _ := json.Marshal(updateData)

		req, _ := http.NewRequest("PUT", baseURL+"/api/v1/users/"+createdUser.ID, bytes.NewBuffer(updateJSON))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("Failed to update user: %v", err)
		}
		defer resp.Body.Close()

		// 7. Get updated user
		resp, err = http.Get(baseURL + "/api/v1/users/" + createdUser.ID)
		if err != nil {
			t.Fatalf("Failed to get updated user: %v", err)
		}
		defer resp.Body.Close()

		var updatedUser User
		json.NewDecoder(resp.Body).Decode(&updatedUser)

		if updatedUser.ID != createdUser.ID {
			t.Errorf("Expected ID to remain %s, got %s", createdUser.ID, updatedUser.ID)
		}

		t.Logf("Successfully completed integration workflow for user %s", updatedUser.ID)
	})
}

// TestE2E_ErrorHandling tests error scenarios
func TestE2E_ErrorHandling(t *testing.T) {
	t.Run("Invalid JSON to Create User", func(t *testing.T) {
		resp, err := http.Post(baseURL+"/api/v1/users", "application/json", strings.NewReader("invalid json"))
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Invalid JSON to Update User", func(t *testing.T) {
		req, _ := http.NewRequest("PUT", baseURL+"/api/v1/users/123", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})
}

// Helper function to check if the application is ready
func TestE2E_ApplicationReady(t *testing.T) {
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(baseURL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				t.Log("Application is ready")
				return
			}
		}
		time.Sleep(time.Second)
	}
	t.Fatal("Application did not become ready in time")
}
