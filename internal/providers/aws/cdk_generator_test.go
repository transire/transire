// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package aws

import (
	"testing"

	"github.com/transire/transire/pkg/transire"
)

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"single word", "todo", "Todo"},
		{"hyphenated", "todo-app", "TodoApp"},
		{"multiple hyphens", "todo-app-stack", "TodoAppStack"},
		{"with underscore", "todo_app", "TodoApp"},
		{"already pascal", "TodoApp", "TodoApp"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toPascalCase(tt.input)
			if got != tt.expected {
				t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"single word", "todo", "todo"},
		{"hyphenated", "todo-app", "todoApp"},
		{"queue name", "todo-reminders", "todoReminders"},
		{"with underscore", "todo_app", "todoApp"},
		{"already camel", "todoApp", "todoApp"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toCamelCase(tt.input)
			if got != tt.expected {
				t.Errorf("toCamelCase(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestToKebabCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"single word", "todo", "todo"},
		{"hyphenated", "todo-app", "todo-app"},
		{"camelCase", "todoApp", "todo-app"},
		{"PascalCase", "TodoApp", "todo-app"},
		{"with underscore", "todo_app", "todo-app"},
		{"already kebab", "todo-app", "todo-app"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toKebabCase(tt.input)
			if got != tt.expected {
				t.Errorf("toKebabCase(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBuildStackData_IncludesEnvironmentVariables(t *testing.T) {
	generator := NewCDKGenerator("us-east-1")

	config := transire.IaCConfig{
		StackName:   "test-stack",
		AppName:     "test-app",
		Environment: "dev",
		FunctionGroups: map[string]transire.FunctionGroupSpec{
			"main": {
				Include: transire.IncludeSpec{
					HTTPHandlers:     "*",
					QueueHandlers:    "*",
					ScheduleHandlers: "*",
				},
			},
		},
		HTTPHandlers: []transire.HTTPHandlerSpec{
			{Path: "/health", Methods: []string{"GET"}, Function: "main"},
		},
	}

	stackData := generator.buildStackData(config)

	// Type assert to access fields
	data, ok := stackData.(struct {
		StackClassName    string
		AppName           string
		Environment       string
		Functions         []FunctionData
		HasHTTPHandlers   bool
		MainFunctionVar   string
		MainFunctionAlias string
		Queues            []QueueData
		Schedules         []ScheduleData
	})

	if !ok {
		t.Fatal("buildStackData returned unexpected type")
	}

	// Verify AppName and Environment are set
	if data.AppName != "test-app" {
		t.Errorf("Expected AppName 'test-app', got '%s'", data.AppName)
	}

	if data.Environment != "dev" {
		t.Errorf("Expected Environment 'dev', got '%s'", data.Environment)
	}

	// Verify function data structure
	if len(data.Functions) != 1 {
		t.Fatalf("Expected 1 function, got %d", len(data.Functions))
	}

	// Verify HasHTTPHandlers is true
	if !data.HasHTTPHandlers {
		t.Error("Expected HasHTTPHandlers to be true")
	}
}

func TestBuildStackData_QueueHandlers(t *testing.T) {
	generator := NewCDKGenerator("us-east-1")

	config := transire.IaCConfig{
		StackName:   "test-stack",
		AppName:     "my-app",
		Environment: "prod",
		FunctionGroups: map[string]transire.FunctionGroupSpec{
			"main": {},
		},
		QueueHandlers: []transire.QueueHandlerSpec{
			{
				QueueName: "email-queue",
				Function:  "main",
				Config: transire.QueueConfig{
					VisibilityTimeoutSeconds: 30,
					MaxReceiveCount:          3,
					BatchSize:                10,
				},
			},
			{
				QueueName: "notification-queue",
				Function:  "main",
				Config: transire.QueueConfig{
					VisibilityTimeoutSeconds: 60,
					MaxReceiveCount:          5,
					BatchSize:                5,
				},
			},
		},
	}

	stackData := generator.buildStackData(config)

	data, ok := stackData.(struct {
		StackClassName    string
		AppName           string
		Environment       string
		Functions         []FunctionData
		HasHTTPHandlers   bool
		MainFunctionVar   string
		MainFunctionAlias string
		Queues            []QueueData
		Schedules         []ScheduleData
	})

	if !ok {
		t.Fatal("buildStackData returned unexpected type")
	}

	// Verify queue data
	if len(data.Queues) != 2 {
		t.Fatalf("Expected 2 queues, got %d", len(data.Queues))
	}

	// Check first queue
	if data.Queues[0].Name != "email-queue" {
		t.Errorf("Expected queue name 'email-queue', got '%s'", data.Queues[0].Name)
	}

	if data.Queues[0].VisibilityTimeoutSeconds != 30 {
		t.Errorf("Expected visibility timeout 30, got %d", data.Queues[0].VisibilityTimeoutSeconds)
	}
}

func TestBuildStackData_ScheduleHandlers(t *testing.T) {
	generator := NewCDKGenerator("us-east-1")

	config := transire.IaCConfig{
		StackName:   "test-stack",
		AppName:     "my-app",
		Environment: "prod",
		FunctionGroups: map[string]transire.FunctionGroupSpec{
			"main": {},
		},
		ScheduleHandlers: []transire.ScheduleHandlerSpec{
			{
				Name:     "daily-cleanup",
				Schedule: "0 2 * * *",
				Function: "main",
			},
		},
	}

	stackData := generator.buildStackData(config)

	data, ok := stackData.(struct {
		StackClassName    string
		AppName           string
		Environment       string
		Functions         []FunctionData
		HasHTTPHandlers   bool
		MainFunctionVar   string
		MainFunctionAlias string
		Queues            []QueueData
		Schedules         []ScheduleData
	})

	if !ok {
		t.Fatal("buildStackData returned unexpected type")
	}

	// Verify schedule data
	if len(data.Schedules) != 1 {
		t.Fatalf("Expected 1 schedule, got %d", len(data.Schedules))
	}

	// Check schedule
	if data.Schedules[0].Name != "daily-cleanup" {
		t.Errorf("Expected schedule name 'daily-cleanup', got '%s'", data.Schedules[0].Name)
	}

	// Verify cron expression was converted to EventBridge format
	if data.Schedules[0].CronExpression != "cron(0 2 * * ? *)" {
		t.Errorf("Expected cron expression 'cron(0 2 * * ? *)', got '%s'", data.Schedules[0].CronExpression)
	}
}
