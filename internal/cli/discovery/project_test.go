package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/transire/transire/pkg/transire"
)

// TestScheduleHandlerDiscovery tests that schedule handlers are discovered
// with their actual cron expressions from the code, not hardcoded values
func TestScheduleHandlerDiscovery(t *testing.T) {
	// Create a temporary test project
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module test-app

go 1.21

require github.com/transire/transire v0.1.0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create main.go with schedule handlers
	mainGo := `package main

import (
	"context"
	"time"
	"github.com/transire/transire/pkg/transire"
)

type DailyCleanupHandler struct{}

func (h *DailyCleanupHandler) Name() string {
	return "daily-cleanup"
}

func (h *DailyCleanupHandler) Schedule() string {
	return "0 2 * * *" // Daily at 2 AM
}

func (h *DailyCleanupHandler) Config() transire.ScheduleConfig {
	return transire.ScheduleConfig{}
}

func (h *DailyCleanupHandler) HandleSchedule(ctx context.Context, event transire.ScheduleEvent) error {
	return nil
}

type HourlyReportHandler struct{}

func (h *HourlyReportHandler) Name() string {
	return "hourly-report"
}

func (h *HourlyReportHandler) Schedule() string {
	return "0 * * * *" // Every hour
}

func (h *HourlyReportHandler) Config() transire.ScheduleConfig {
	return transire.ScheduleConfig{}
}

func (h *HourlyReportHandler) HandleSchedule(ctx context.Context, event transire.ScheduleEvent) error {
	return nil
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatalf("Failed to create main.go: %v", err)
	}

	// Create config with schedule definitions
	config := &transire.Config{
		Name:     "test-app",
		Language: "go",
		Schedules: map[string]transire.ScheduleConfig{
			"daily-cleanup": {
				Timezone: "UTC",
				Enabled:  true,
			},
			"hourly-report": {
				Timezone: "UTC",
				Enabled:  true,
			},
		},
	}

	// Discover project
	project, err := DiscoverProject(tmpDir, config)
	if err != nil {
		t.Fatalf("DiscoverProject() error = %v", err)
	}

	// Test cases for discovered schedules
	tests := []struct {
		name             string
		expectedSchedule string
	}{
		{
			name:             "daily-cleanup",
			expectedSchedule: "0 2 * * *", // Should match actual code, not "0 * * * *"
		},
		{
			name:             "hourly-report",
			expectedSchedule: "0 * * * *",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Find the schedule handler
			var found bool
			var actualSchedule string
			for _, handler := range project.ScheduleHandlers {
				if handler.Name == tt.name {
					found = true
					actualSchedule = handler.Schedule
					break
				}
			}

			if !found {
				t.Errorf("Schedule handler %q not found in discovered handlers", tt.name)
				return
			}

			if actualSchedule != tt.expectedSchedule {
				t.Errorf("Schedule handler %q has schedule %q, want %q",
					tt.name, actualSchedule, tt.expectedSchedule)
			}
		})
	}
}

// TestScheduleHandlerDiscoveryParsingRequired tests that we actually parse
// the Go source code to extract the Schedule() method return value
func TestScheduleHandlerDiscoveryParsingRequired(t *testing.T) {
	tmpDir := t.TempDir()

	// Create minimal project structure
	goMod := `module test-app
go 1.21
require github.com/transire/transire v0.1.0
`
	mainGo := `package main

import (
	"context"
	"github.com/transire/transire/pkg/transire"
)

type WeeklyBackupHandler struct{}

func (h *WeeklyBackupHandler) Name() string {
	return "weekly-backup"
}

func (h *WeeklyBackupHandler) Schedule() string {
	return "0 0 * * 0" // Weekly on Sunday
}

func (h *WeeklyBackupHandler) Config() transire.ScheduleConfig {
	return transire.ScheduleConfig{}
}

func (h *WeeklyBackupHandler) HandleSchedule(ctx context.Context, event transire.ScheduleEvent) error {
	return nil
}
`

	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)

	config := &transire.Config{
		Name:     "test-app",
		Language: "go",
		Schedules: map[string]transire.ScheduleConfig{
			"weekly-backup": {
				Timezone: "UTC",
				Enabled:  true,
			},
		},
	}

	project, err := DiscoverProject(tmpDir, config)
	if err != nil {
		t.Fatalf("DiscoverProject() error = %v", err)
	}

	// Find the weekly-backup handler
	var found bool
	var actualSchedule string
	for _, handler := range project.ScheduleHandlers {
		if handler.Name == "weekly-backup" {
			found = true
			actualSchedule = handler.Schedule
			break
		}
	}

	if !found {
		t.Fatalf("weekly-backup handler not found")
	}

	// The schedule should be "0 0 * * 0" (weekly), NOT the hardcoded "0 * * * *" (hourly)
	expectedSchedule := "0 0 * * 0"
	if actualSchedule != expectedSchedule {
		t.Errorf("Schedule = %q, want %q (should parse actual Schedule() method, not use hardcoded default)",
			actualSchedule, expectedSchedule)
	}
}
