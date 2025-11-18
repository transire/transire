package discovery

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/transire-org/transire/pkg/transire"
)

// Project represents a discovered Transire project
type Project struct {
	Type             ProjectType
	Path             string
	Config           *transire.Config
	HTTPHandlers     []transire.HTTPHandlerSpec
	QueueHandlers    []transire.QueueHandlerSpec
	ScheduleHandlers []transire.ScheduleHandlerSpec
}

// ProjectType represents the project language/type
type ProjectType string

const (
	ProjectTypeGo   ProjectType = "go"
	ProjectTypeRust ProjectType = "rust" // Future
)

// DiscoverProject discovers a Transire project in the given directory
func DiscoverProject(dir string, config *transire.Config) (*Project, error) {
	// Determine project type
	projectType, err := detectProjectType(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to detect project type: %w", err)
	}

	project := &Project{
		Type:   projectType,
		Path:   dir,
		Config: config,
	}

	// Discover handlers based on project type
	switch projectType {
	case ProjectTypeGo:
		if err := discoverGoHandlers(project); err != nil {
			return nil, fmt.Errorf("failed to discover Go handlers: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported project type: %s", projectType)
	}

	return project, nil
}

// LoadApplication loads a Transire application from the given directory
func LoadApplication(dir string, config *transire.Config) (*transire.App, error) {
	// For now, create a basic app with the configuration
	// In a real implementation, this would dynamically load the user's app
	app := transire.New(transire.WithConfig(config))

	// TODO: Actually load and register the user's handlers
	// This would involve:
	// 1. Building the Go module as a plugin or using go/types to parse
	// 2. Extracting handler registrations
	// 3. Instantiating and registering handlers
	//
	// For now, we'll return a basic app that can at least start

	return app, nil
}

// detectProjectType detects the type of project
func detectProjectType(dir string) (ProjectType, error) {
	// Check for Go project
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return ProjectTypeGo, nil
	}

	// Check for Rust project
	if _, err := os.Stat(filepath.Join(dir, "Cargo.toml")); err == nil {
		return ProjectTypeRust, nil
	}

	return "", fmt.Errorf("no supported project found in directory: %s", dir)
}

// discoverGoHandlers discovers handlers in a Go project
func discoverGoHandlers(project *Project) error {
	// This is a simplified implementation
	// In practice, this would parse Go source code to find handler registrations
	// For now, we'll create some placeholder handlers based on configuration

	// Create placeholder HTTP handlers
	project.HTTPHandlers = []transire.HTTPHandlerSpec{
		{
			Path:     "/health",
			Methods:  []string{"GET"},
			Function: "main",
		},
	}

	// Add queue handlers if defined in config
	for queueName := range project.Config.Queues {
		project.QueueHandlers = append(project.QueueHandlers, transire.QueueHandlerSpec{
			QueueName: queueName,
			Function:  "main",
			Config:    project.Config.Queues[queueName],
		})
	}

	// Add schedule handlers if defined in config
	for scheduleName := range project.Config.Schedules {
		project.ScheduleHandlers = append(project.ScheduleHandlers, transire.ScheduleHandlerSpec{
			Name:     scheduleName,
			Schedule: "0 * * * *", // Default hourly
			Function: "main",
			Config:   project.Config.Schedules[scheduleName],
		})
	}

	return nil
}
