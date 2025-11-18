// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package aws

import (
	"testing"
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
