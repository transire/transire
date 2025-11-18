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

// TestE2E_ApplicationReady checks if the application is running
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

// TestE2E_HTTPHandlers tests all HTTP endpoints for TODO operations
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

		if result["service"] != "transire-todo-api" {
			t.Errorf("Expected service 'transire-todo-api', got '%s'", result["service"])
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

		if result["message"] != "Welcome to Transire TODO API" {
			t.Errorf("Expected welcome message, got '%s'", result["message"])
		}
	})

	t.Run("CRUD Todo Operations", func(t *testing.T) {
		// Create a todo
		newTodo := map[string]interface{}{
			"title":       "Buy groceries",
			"description": "Milk, eggs, bread",
			"due_date":    time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		}
		todoJSON, _ := json.Marshal(newTodo)

		resp, err := http.Post(baseURL+"/api/v1/todos", "application/json", bytes.NewBuffer(todoJSON))
		if err != nil {
			t.Fatalf("Failed to create todo: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", resp.StatusCode)
		}

		var createdTodo Todo
		if err := json.NewDecoder(resp.Body).Decode(&createdTodo); err != nil {
			t.Fatalf("Failed to decode created todo: %v", err)
		}

		if createdTodo.Title != "Buy groceries" {
			t.Errorf("Expected title 'Buy groceries', got '%s'", createdTodo.Title)
		}

		if createdTodo.Completed {
			t.Error("Expected todo to not be completed")
		}

		if createdTodo.ID == "" {
			t.Error("Expected non-empty todo ID")
		}

		todoID := createdTodo.ID
		t.Logf("Created todo with ID: %s", todoID)

		// Get the todo
		resp, err = http.Get(baseURL + "/api/v1/todos/" + todoID)
		if err != nil {
			t.Fatalf("Failed to get todo: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var fetchedTodo Todo
		if err := json.NewDecoder(resp.Body).Decode(&fetchedTodo); err != nil {
			t.Fatalf("Failed to decode fetched todo: %v", err)
		}

		if fetchedTodo.ID != todoID {
			t.Errorf("Expected ID '%s', got '%s'", todoID, fetchedTodo.ID)
		}

		// List todos
		resp, err = http.Get(baseURL + "/api/v1/todos")
		if err != nil {
			t.Fatalf("Failed to list todos: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var listResult struct {
			Todos []Todo `json:"todos"`
			Count int    `json:"count"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&listResult); err != nil {
			t.Fatalf("Failed to decode todo list: %v", err)
		}

		if listResult.Count == 0 {
			t.Error("Expected at least one todo in the list")
		}

		// Update the todo
		updateData := map[string]interface{}{
			"description": "Milk, eggs, bread, butter",
		}
		updateJSON, _ := json.Marshal(updateData)

		req, _ := http.NewRequest("PUT", baseURL+"/api/v1/todos/"+todoID, bytes.NewBuffer(updateJSON))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("Failed to update todo: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Complete the todo
		resp, err = http.Post(baseURL+"/api/v1/todos/"+todoID+"/complete", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to complete todo: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var completedTodo Todo
		if err := json.NewDecoder(resp.Body).Decode(&completedTodo); err != nil {
			t.Fatalf("Failed to decode completed todo: %v", err)
		}

		if !completedTodo.Completed {
			t.Error("Expected todo to be completed")
		}

		// Delete the todo
		req, _ = http.NewRequest("DELETE", baseURL+"/api/v1/todos/"+todoID, nil)
		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("Failed to delete todo: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", resp.StatusCode)
		}

		// Verify deletion
		resp, err = http.Get(baseURL + "/api/v1/todos/" + todoID)
		if err != nil {
			t.Fatalf("Failed to verify deletion: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404 after deletion, got %d", resp.StatusCode)
		}
	})
}

// TestE2E_QueueHandlers tests queue functionality using CLI
func TestE2E_QueueHandlers(t *testing.T) {
	workDir := "/Users/jamie/personal/transire2/examples/todo-app"

	// Create a todo first for reminder testing
	newTodo := map[string]interface{}{
		"title":       "Test Reminder Todo",
		"description": "This todo will have a reminder",
		"due_date":    time.Now().Add(48 * time.Hour).Format(time.RFC3339),
	}
	todoJSON, _ := json.Marshal(newTodo)

	resp, _ := http.Post(baseURL+"/api/v1/todos", "application/json", bytes.NewBuffer(todoJSON))
	var createdTodo Todo
	json.NewDecoder(resp.Body).Decode(&createdTodo)
	resp.Body.Close()
	todoID := createdTodo.ID

	t.Run("List Queues", func(t *testing.T) {
		cmd := exec.Command(cliPath, "dev", "queues", "list", "-c", "transire.yaml")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to list queues: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "todo-reminders") {
			t.Error("Expected 'todo-reminders' in output")
		}
		if !strings.Contains(outputStr, "todo-notifications") {
			t.Error("Expected 'todo-notifications' in output")
		}

		t.Logf("Queue list output: %s", outputStr)
	})

	t.Run("Send Todo Reminder Message", func(t *testing.T) {
		reminderMessage := TodoReminderMessage{
			TodoID:    todoID,
			UserEmail: "user@example.com",
		}

		messageJSON, _ := json.Marshal(reminderMessage)

		cmd := exec.Command(cliPath, "dev", "queues", "send", "-c", "transire.yaml", "todo-reminders", string(messageJSON))
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to send reminder queue message: %v, output: %s", err, string(output))
		}

		t.Logf("Reminder queue send output: %s", string(output))

		// Wait for message processing
		time.Sleep(2 * time.Second)
	})

	t.Run("Send Todo Reminder Message with Failure", func(t *testing.T) {
		reminderMessage := TodoReminderMessage{
			TodoID:    todoID,
			UserEmail: "fail-reminder@example.com", // This will trigger failure
		}

		messageJSON, _ := json.Marshal(reminderMessage)

		cmd := exec.Command(cliPath, "dev", "queues", "send", "-c", "transire.yaml", "todo-reminders", string(messageJSON))
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to send failing reminder message: %v, output: %s", err, string(output))
		}

		t.Logf("Failing reminder queue send output: %s", string(output))

		// Wait for processing
		time.Sleep(2 * time.Second)
	})

	t.Run("Send Todo Notification Message", func(t *testing.T) {
		notificationMessage := TodoNotificationMessage{
			TodoID:    todoID,
			UserEmail: "user@example.com",
			EventType: "created",
			TodoTitle: "Test Reminder Todo",
		}

		messageJSON, _ := json.Marshal(notificationMessage)

		cmd := exec.Command(cliPath, "dev", "queues", "send", "-c", "transire.yaml", "todo-notifications", string(messageJSON))
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to send notification queue message: %v, output: %s", err, string(output))
		}

		t.Logf("Notification queue send output: %s", string(output))

		// Wait for processing
		time.Sleep(2 * time.Second)
	})

	t.Run("Send Todo Notification Message with Failure", func(t *testing.T) {
		notificationMessage := TodoNotificationMessage{
			TodoID:    todoID,
			UserEmail: "fail-notification@example.com", // This will trigger failure
			EventType: "updated",
			TodoTitle: "Test Reminder Todo",
		}

		messageJSON, _ := json.Marshal(notificationMessage)

		cmd := exec.Command(cliPath, "dev", "queues", "send", "-c", "transire.yaml", "todo-notifications", string(messageJSON))
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to send failing notification message: %v, output: %s", err, string(output))
		}

		t.Logf("Failing notification queue send output: %s", string(output))

		// Wait for processing
		time.Sleep(2 * time.Second)
	})
}

// TestE2E_ScheduleHandlers tests schedule functionality using CLI
func TestE2E_ScheduleHandlers(t *testing.T) {
	workDir := "/Users/jamie/personal/transire2/examples/todo-app"

	// Create some test todos - some completed, some pending
	for i := 0; i < 3; i++ {
		todo := map[string]interface{}{
			"title":       fmt.Sprintf("Test Todo %d", i),
			"description": "Test description",
		}
		if i == 0 {
			// Make first todo completed
			todoJSON, _ := json.Marshal(todo)
			resp, _ := http.Post(baseURL+"/api/v1/todos", "application/json", bytes.NewBuffer(todoJSON))
			var createdTodo Todo
			json.NewDecoder(resp.Body).Decode(&createdTodo)
			resp.Body.Close()

			// Complete it
			http.Post(baseURL+"/api/v1/todos/"+createdTodo.ID+"/complete", "application/json", nil)
		} else {
			todoJSON, _ := json.Marshal(todo)
			http.Post(baseURL+"/api/v1/todos", "application/json", bytes.NewBuffer(todoJSON))
		}
	}

	t.Run("List Schedules", func(t *testing.T) {
		cmd := exec.Command(cliPath, "dev", "schedules", "list", "-c", "transire.yaml")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to list schedules: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "cleanup-completed-todos") {
			t.Error("Expected 'cleanup-completed-todos' in output")
		}
		if !strings.Contains(outputStr, "daily-todo-summary") {
			t.Error("Expected 'daily-todo-summary' in output")
		}

		t.Logf("Schedule list output: %s", outputStr)
	})

	t.Run("Execute Cleanup Schedule Manually", func(t *testing.T) {
		cmd := exec.Command(cliPath, "dev", "schedules", "execute", "-c", "transire.yaml", "cleanup-completed-todos")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to execute cleanup schedule: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		t.Logf("Cleanup schedule execute output: %s", outputStr)

		// Wait for execution to complete
		time.Sleep(3 * time.Second)
	})

	t.Run("Execute Daily Summary Schedule Manually", func(t *testing.T) {
		cmd := exec.Command(cliPath, "dev", "schedules", "execute", "-c", "transire.yaml", "daily-todo-summary")
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to execute summary schedule: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		t.Logf("Summary schedule execute output: %s", outputStr)

		// Wait for execution to complete
		time.Sleep(3 * time.Second)
	})
}

// TestE2E_IntegrationWorkflow tests a complete realistic workflow
func TestE2E_IntegrationWorkflow(t *testing.T) {
	workDir := "/Users/jamie/personal/transire2/examples/todo-app"

	t.Run("Complete Todo Workflow with Notifications and Cleanup", func(t *testing.T) {
		// 1. Create a new todo via HTTP API
		newTodo := map[string]interface{}{
			"title":       "Complete project documentation",
			"description": "Write comprehensive documentation for the project",
			"due_date":    time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339),
		}
		todoJSON, _ := json.Marshal(newTodo)

		resp, err := http.Post(baseURL+"/api/v1/todos", "application/json", bytes.NewBuffer(todoJSON))
		if err != nil {
			t.Fatalf("Failed to create todo in workflow: %v", err)
		}
		defer resp.Body.Close()

		var createdTodo Todo
		json.NewDecoder(resp.Body).Decode(&createdTodo)
		todoID := createdTodo.ID

		t.Logf("Created todo: %s (ID: %s)", createdTodo.Title, todoID)

		// 2. Send a notification that the todo was created
		notification := TodoNotificationMessage{
			TodoID:    todoID,
			UserEmail: "project-manager@example.com",
			EventType: "created",
			TodoTitle: createdTodo.Title,
		}
		notificationJSON, _ := json.Marshal(notification)

		cmd := exec.Command(cliPath, "dev", "queues", "send", "-c", "transire.yaml", "todo-notifications", string(notificationJSON))
		cmd.Dir = workDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to send notification: %v, output: %s", err, string(output))
		}

		// 3. Send a reminder for the todo
		reminder := TodoReminderMessage{
			TodoID:    todoID,
			UserEmail: "developer@example.com",
		}
		reminderJSON, _ := json.Marshal(reminder)

		cmd = exec.Command(cliPath, "dev", "queues", "send", "-c", "transire.yaml", "todo-reminders", string(reminderJSON))
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to send reminder: %v, output: %s", err, string(output))
		}

		// Wait for queue processing
		time.Sleep(3 * time.Second)

		// 4. Update the todo
		updateData := map[string]interface{}{
			"description": "Write comprehensive documentation including API references and examples",
		}
		updateJSON, _ := json.Marshal(updateData)

		req, _ := http.NewRequest("PUT", baseURL+"/api/v1/todos/"+todoID, bytes.NewBuffer(updateJSON))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("Failed to update todo: %v", err)
		}
		resp.Body.Close()

		// 5. Complete the todo
		resp, err = http.Post(baseURL+"/api/v1/todos/"+todoID+"/complete", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to complete todo: %v", err)
		}
		resp.Body.Close()

		// 6. Send completion notification
		completionNotification := TodoNotificationMessage{
			TodoID:    todoID,
			UserEmail: "project-manager@example.com",
			EventType: "completed",
			TodoTitle: createdTodo.Title,
		}
		completionJSON, _ := json.Marshal(completionNotification)

		cmd = exec.Command(cliPath, "dev", "queues", "send", "-c", "transire.yaml", "todo-notifications", string(completionJSON))
		cmd.Dir = workDir
		cmd.CombinedOutput()

		// 7. Trigger daily summary
		cmd = exec.Command(cliPath, "dev", "schedules", "execute", "-c", "transire.yaml", "daily-todo-summary")
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to execute daily summary: %v, output: %s", err, string(output))
		}

		// 8. Verify the todo is still there
		resp, err = http.Get(baseURL + "/api/v1/todos/" + todoID)
		if err != nil {
			t.Fatalf("Failed to get todo: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected todo to still exist, got status %d", resp.StatusCode)
		}

		t.Logf("Successfully completed integration workflow for todo %s", todoID)
	})
}

// TestE2E_ErrorHandling tests error scenarios
func TestE2E_ErrorHandling(t *testing.T) {
	t.Run("Create Todo Without Title", func(t *testing.T) {
		todo := map[string]interface{}{
			"description": "No title",
		}
		todoJSON, _ := json.Marshal(todo)

		resp, err := http.Post(baseURL+"/api/v1/todos", "application/json", bytes.NewBuffer(todoJSON))
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Get Non-Existent Todo", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/api/v1/todos/non-existent-id")
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}
	})

	t.Run("Update Non-Existent Todo", func(t *testing.T) {
		updateData := map[string]interface{}{
			"title": "Updated",
		}
		updateJSON, _ := json.Marshal(updateData)

		req, _ := http.NewRequest("PUT", baseURL+"/api/v1/todos/non-existent-id", bytes.NewBuffer(updateJSON))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}
	})

	t.Run("Delete Non-Existent Todo", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", baseURL+"/api/v1/todos/non-existent-id", nil)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		resp, err := http.Post(baseURL+"/api/v1/todos", "application/json", strings.NewReader("invalid json"))
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})
}
