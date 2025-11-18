// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package runner

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Manager coordinates building, running, and watching
type Manager struct {
	projectDir string
	builder    *Builder
	process    *Process
	watcher    *Watcher
	rebuilding bool
}

// NewManager creates a new Manager
func NewManager(projectDir string) *Manager {
	return &Manager{
		projectDir: projectDir,
		builder:    NewBuilder(projectDir),
	}
}

// Run starts the application with hot reload
func (m *Manager) Run() error {
	log.Println("🚀 Starting Transire development server...")

	// Initial build
	if err := m.buildAndStart(); err != nil {
		return fmt.Errorf("initial build failed: %w", err)
	}

	// Start file watcher
	watcher, err := NewWatcher(m.projectDir, m.onFileChange)
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	m.watcher = watcher

	if err := m.watcher.Start(); err != nil {
		return fmt.Errorf("failed to start file watcher: %w", err)
	}

	// Handle interruption signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("✅ Ready! Press Ctrl+C to stop")

	// Wait for interrupt
	<-sigChan

	log.Println("\n🛑 Shutting down...")
	return m.Stop()
}

// buildAndStart builds and starts the application
func (m *Manager) buildAndStart() error {
	log.Println("📦 Building application...")

	// Build the application
	binaryPath, err := m.builder.Build()
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	log.Printf("✅ Build successful: %s", binaryPath)

	// Stop existing process if running
	if m.process != nil && m.process.IsRunning() {
		log.Println("🔄 Restarting application...")
		if err := m.process.Stop(); err != nil {
			log.Printf("Warning: failed to stop previous process: %v", err)
		}
		// Wait a bit for the port to be released
		time.Sleep(200 * time.Millisecond)
	}

	// Start new process
	m.process = NewProcess(binaryPath)
	if err := m.process.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	log.Println("✅ Application started")
	return nil
}

// onFileChange is called when files change
func (m *Manager) onFileChange() {
	if m.rebuilding {
		log.Println("⏳ Rebuild already in progress, skipping...")
		return
	}

	m.rebuilding = true
	defer func() { m.rebuilding = false }()

	log.Println("\n🔄 File change detected, rebuilding...")

	if err := m.buildAndStart(); err != nil {
		log.Printf("❌ Rebuild failed: %v", err)
		log.Println("⏳ Watching for changes...")
		return
	}

	log.Println("✅ Rebuild complete, watching for changes...")
}

// Stop stops the manager
func (m *Manager) Stop() error {
	var errors []error

	// Stop watcher
	if m.watcher != nil {
		if err := m.watcher.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("failed to stop watcher: %w", err))
		}
	}

	// Stop process
	if m.process != nil {
		if err := m.process.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("failed to stop process: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errors)
	}

	log.Println("👋 Goodbye!")
	return nil
}
