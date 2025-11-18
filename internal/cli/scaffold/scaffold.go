// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package scaffold

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/transire/transire/pkg/transire"
)

// Scaffolder generates project structure
type Scaffolder struct {
	config     *transire.Config
	projectDir string
}

// New creates a new scaffolder
func New(config *transire.Config, projectDir string) *Scaffolder {
	return &Scaffolder{
		config:     config,
		projectDir: projectDir,
	}
}

// Generate creates the project structure
func (s *Scaffolder) Generate() error {
	// Create project directory
	if err := os.MkdirAll(s.projectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Generate based on language
	switch s.config.Language {
	case "go":
		return s.generateGoProject()
	default:
		return fmt.Errorf("unsupported language: %s", s.config.Language)
	}
}

// generateGoProject generates a Go project structure
func (s *Scaffolder) generateGoProject() error {
	// Generate go.mod
	if err := s.generateGoMod(); err != nil {
		return fmt.Errorf("failed to generate go.mod: %w", err)
	}

	// Generate main.go
	if err := s.generateMainGo(); err != nil {
		return fmt.Errorf("failed to generate main.go: %w", err)
	}

	// Generate transire.yaml
	if err := s.generateConfig(); err != nil {
		return fmt.Errorf("failed to generate transire.yaml: %w", err)
	}

	// Generate infrastructure (CDK)
	if s.config.IaC == "cdk" {
		if err := s.generateCDKInfrastructure(); err != nil {
			return fmt.Errorf("failed to generate CDK infrastructure: %w", err)
		}
	}

	// Generate CI/CD
	if s.config.CI == "github" {
		if err := s.generateGitHubActions(); err != nil {
			return fmt.Errorf("failed to generate GitHub Actions: %w", err)
		}
	}

	// Generate .gitignore
	if err := s.generateGitIgnore(); err != nil {
		return fmt.Errorf("failed to generate .gitignore: %w", err)
	}

	return nil
}

// generateGoMod creates go.mod file
func (s *Scaffolder) generateGoMod() error {
	content := fmt.Sprintf(`module %s

go 1.21

require (
	github.com/go-chi/chi/v5 v5.0.12
	github.com/transire/transire v0.1.0
)
`, s.config.Name)

	return s.writeFile("go.mod", content)
}

// generateMainGo creates main.go file
func (s *Scaffolder) generateMainGo() error {
	content := fmt.Sprintf(`package main

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/transire/transire/pkg/transire"
)

func main() {
	// Load configuration from transire.yaml
	config, err := transire.LoadConfig("")
	if err != nil {
		log.Fatalf("Failed to load config: %%v", err)
	}

	// Create Transire app with configuration
	app := transire.New(transire.WithConfig(config))

	// Get Chi router - use exactly like normal Chi
	r := app.Router()

	// Standard Chi middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Standard Chi routes
	r.Get("/", homeHandler)
	r.Get("/health", healthHandler)

	// TODO: Add your routes here
	// r.Post("/api/users", createUserHandler)
	// r.Get("/api/users/{id}", getUserHandler)

	// TODO: Register queue handlers
	// app.RegisterQueueHandler(&MyQueueHandler{})

	// TODO: Register schedule handlers
	// app.RegisterScheduleHandler(&MyScheduleHandler{})

	// Run the app (works locally and in Lambda)
	if err := app.Run(context.Background()); err != nil {
		panic(err)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	html := `+"`"+`<!DOCTYPE html>
<html>
<head>
    <title>%s</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .container { max-width: 600px; margin: 0 auto; }
        .logo { color: #6366f1; font-size: 24px; font-weight: bold; }
        .status { color: #10b981; }
    </style>
</head>
<body>
    <div class="container">
        <h1 class="logo">🚀 Transire</h1>
        <h2>Welcome to %s</h2>
        <p class="status">✅ Your application is running successfully!</p>
        <p>This is a Transire application that runs consistently across local development and cloud deployments.</p>
        <ul>
            <li><a href="/health">Health Check</a></li>
            <li>Add your routes in main.go</li>
            <li>Configure infrastructure in transire.yaml</li>
        </ul>
    </div>
</body>
</html>`+"`"+`
	w.Write([]byte(html))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`+"`"+`{"status": "ok", "service": "%s"}`+"`"+`))
}
`, s.config.Name, s.config.Name, s.config.Name)

	return s.writeFile("main.go", content)
}

// generateConfig creates transire.yaml file
func (s *Scaffolder) generateConfig() error {
	configContent := fmt.Sprintf(`# Transire configuration
name: %s
language: %s
cloud: %s
runtime: %s
iac: %s
ci: %s

# Lambda configuration
lambda:
  architecture: arm64
  timeout_seconds: 30
  memory_mb: 128

# Function grouping (optional)
functions:
  main:
    include:
      - http_handlers: "*"
      - queue_handlers: "*"
      - schedule_handlers: "*"

# Environment variables
environment:
  NODE_ENV: production

# Development settings
development:
  http_port: 3000
  queue_port: 4000
  scheduler_port: 5000
  auto_reload: true
  log_level: info

# Queue configuration (uncomment to add queues)
# queues:
#   email-queue:
#     visibility_timeout_seconds: 60
#     max_receive_count: 3
#     batch_size: 10

# Schedule configuration (uncomment to add schedules)
# schedules:
#   daily-cleanup:
#     timezone: "UTC"
#     enabled: true
`,
		s.config.Name,
		s.config.Language,
		s.config.Cloud,
		s.config.Runtime,
		s.config.IaC,
		s.config.CI,
	)

	return s.writeFile("transire.yaml", configContent)
}

// generateCDKInfrastructure creates CDK infrastructure files
func (s *Scaffolder) generateCDKInfrastructure() error {
	infraDir := filepath.Join(s.projectDir, "infrastructure")
	if err := os.MkdirAll(filepath.Join(infraDir, "lib"), 0755); err != nil {
		return err
	}

	// Generate placeholder CDK files
	// The actual implementation would be handled by the CDK generator
	packageJSON := `{
  "name": "transire-infrastructure",
  "version": "0.1.0",
  "main": "app.js",
  "scripts": {
    "build": "tsc",
    "watch": "tsc -w",
    "cdk": "cdk"
  },
  "devDependencies": {
    "@types/node": "^18.0.0",
    "typescript": "^4.9.0",
    "aws-cdk": "^2.87.0"
  },
  "dependencies": {
    "aws-cdk-lib": "^2.87.0",
    "constructs": "^10.0.0"
  }
}`

	return os.WriteFile(filepath.Join(infraDir, "package.json"), []byte(packageJSON), 0644)
}

// generateGitHubActions creates GitHub Actions workflow
func (s *Scaffolder) generateGitHubActions() error {
	workflowDir := filepath.Join(s.projectDir, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		return err
	}

	workflow := `name: Deploy Transire Application

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

env:
  GO_VERSION: 1.21

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: ${{ env.GO_VERSION }}

    - name: Run tests
      run: go test -v ./...

  build:
    needs: test
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    steps:
    - uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: ${{ env.GO_VERSION }}

    - name: Install Transire CLI
      run: |
        # TODO: Install Transire CLI from releases
        go install github.com/transire/transire/cmd/transire@latest

    - name: Build artifacts
      run: transire build

    - name: Configure AWS credentials
      uses: aws-actions/configure-aws-credentials@v2
      with:
        aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
        aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
        aws-region: us-east-1

    - name: Deploy to AWS
      run: transire deploy
`

	return os.WriteFile(filepath.Join(workflowDir, "deploy.yml"), []byte(workflow), 0644)
}

// generateGitIgnore creates .gitignore file
func (s *Scaffolder) generateGitIgnore() error {
	content := `# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, built with go test -c
*.test

# Output of the go coverage tool
*.out

# Go workspace file
go.work

# Transire build outputs
dist/
build/

# CDK outputs
infrastructure/cdk.out/
infrastructure/node_modules/
infrastructure/*.js
infrastructure/*.d.ts

# Environment files
.env
.env.local

# OS files
.DS_Store
Thumbs.db

# IDE files
.vscode/
.idea/
*.swp
*.swo
*~
`

	return s.writeFile(".gitignore", content)
}

// writeFile writes content to a file in the project directory
func (s *Scaffolder) writeFile(filename, content string) error {
	filePath := filepath.Join(s.projectDir, filename)
	return os.WriteFile(filePath, []byte(content), 0644)
}
