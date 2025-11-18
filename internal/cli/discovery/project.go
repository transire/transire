// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package discovery

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/transire/transire/pkg/transire"
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

	// Parse schedule handlers from Go source code
	schedulesByName, err := parseScheduleHandlers(project.Path)
	if err != nil {
		// Fall back to config-based discovery with default schedule
		for scheduleName := range project.Config.Schedules {
			project.ScheduleHandlers = append(project.ScheduleHandlers, transire.ScheduleHandlerSpec{
				Name:     scheduleName,
				Schedule: "0 * * * *", // Default hourly
				Function: "main",
				Config:   project.Config.Schedules[scheduleName],
			})
		}
	} else {
		// Use parsed schedules from code
		for scheduleName, schedule := range schedulesByName {
			if config, exists := project.Config.Schedules[scheduleName]; exists {
				project.ScheduleHandlers = append(project.ScheduleHandlers, transire.ScheduleHandlerSpec{
					Name:     scheduleName,
					Schedule: schedule,
					Function: "main",
					Config:   config,
				})
			}
		}
	}

	return nil
}

// parseScheduleHandlers parses Go source files to find Schedule() method implementations
// Returns a map of schedule name -> cron expression
func parseScheduleHandlers(projectPath string) (map[string]string, error) {
	schedules := make(map[string]string)
	fset := token.NewFileSet()

	// Walk through all Go files in the project
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip vendor and hidden directories
		if strings.Contains(path, "/vendor/") || strings.Contains(path, "/.") {
			return nil
		}

		// Parse the Go file
		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil // Skip files that can't be parsed
		}

		// Find Schedule() methods
		ast.Inspect(node, func(n ast.Node) bool {
			funcDecl, ok := n.(*ast.FuncDecl)
			if !ok || funcDecl.Recv == nil || funcDecl.Name.Name != "Schedule" {
				return true
			}

			// Check if this method returns a string
			if funcDecl.Type.Results == nil || len(funcDecl.Type.Results.List) != 1 {
				return true
			}

			// Extract the return value from the function body
			if funcDecl.Body != nil {
				for _, stmt := range funcDecl.Body.List {
					returnStmt, ok := stmt.(*ast.ReturnStmt)
					if !ok || len(returnStmt.Results) != 1 {
						continue
					}

					// Get the string literal from the return statement
					basicLit, ok := returnStmt.Results[0].(*ast.BasicLit)
					if ok && basicLit.Kind == token.STRING {
						// Remove quotes from string literal
						schedule := strings.Trim(basicLit.Value, `"`)

						// Find the Name() method to get the schedule name
						handlerName := findHandlerName(node, funcDecl.Recv)
						if handlerName != "" {
							schedules[handlerName] = schedule
						}
					}
				}
			}

			return true
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	return schedules, nil
}

// findHandlerName finds the Name() method for a given receiver type to get the schedule name
func findHandlerName(node *ast.File, recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}

	// Get the receiver type
	var recvTypeName string
	switch typ := recv.List[0].Type.(type) {
	case *ast.StarExpr:
		if ident, ok := typ.X.(*ast.Ident); ok {
			recvTypeName = ident.Name
		}
	case *ast.Ident:
		recvTypeName = typ.Name
	}

	if recvTypeName == "" {
		return ""
	}

	// Find the Name() method for this receiver
	var handlerName string
	ast.Inspect(node, func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil || funcDecl.Name.Name != "Name" {
			return true
		}

		// Check if this is the same receiver type
		var thisRecvTypeName string
		switch typ := funcDecl.Recv.List[0].Type.(type) {
		case *ast.StarExpr:
			if ident, ok := typ.X.(*ast.Ident); ok {
				thisRecvTypeName = ident.Name
			}
		case *ast.Ident:
			thisRecvTypeName = typ.Name
		}

		if thisRecvTypeName != recvTypeName {
			return true
		}

		// Extract the return value
		if funcDecl.Body != nil {
			for _, stmt := range funcDecl.Body.List {
				returnStmt, ok := stmt.(*ast.ReturnStmt)
				if !ok || len(returnStmt.Results) != 1 {
					continue
				}

				basicLit, ok := returnStmt.Results[0].(*ast.BasicLit)
				if ok && basicLit.Kind == token.STRING {
					handlerName = strings.Trim(basicLit.Value, `"`)
					return false // Found it, stop searching
				}
			}
		}

		return true
	})

	return handlerName
}
