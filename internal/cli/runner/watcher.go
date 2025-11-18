// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package runner

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches for file changes
type Watcher struct {
	projectDir string
	watcher    *fsnotify.Watcher
	onChange   func()
	debounce   time.Duration
	lastChange time.Time
}

// NewWatcher creates a new file watcher
func NewWatcher(projectDir string, onChange func()) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		projectDir: projectDir,
		watcher:    fsWatcher,
		onChange:   onChange,
		debounce:   500 * time.Millisecond, // Debounce changes
	}, nil
}

// Start starts watching for file changes
func (w *Watcher) Start() error {
	// Add project directory to watch
	if err := w.addDirectoryRecursive(w.projectDir); err != nil {
		return err
	}

	// Start watching in goroutine
	go w.watch()

	log.Printf("Watching for file changes in: %s", w.projectDir)
	return nil
}

// Stop stops the watcher
func (w *Watcher) Stop() error {
	return w.watcher.Close()
}

// watch handles file system events
func (w *Watcher) watch() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Filter events
			if w.shouldIgnore(event.Name) {
				continue
			}

			// Only trigger on write and create events
			if event.Op&fsnotify.Write == fsnotify.Write ||
				event.Op&fsnotify.Create == fsnotify.Create {
				w.handleChange(event.Name)
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

// handleChange handles a file change event with debouncing
func (w *Watcher) handleChange(path string) {
	now := time.Now()

	// Debounce rapid changes
	if now.Sub(w.lastChange) < w.debounce {
		return
	}

	w.lastChange = now
	log.Printf("File changed: %s", path)

	// Trigger onChange callback
	if w.onChange != nil {
		w.onChange()
	}
}

// shouldIgnore checks if a file should be ignored
func (w *Watcher) shouldIgnore(path string) bool {
	// Ignore hidden files and directories
	if strings.Contains(path, "/.") {
		return true
	}

	// Ignore build artifacts
	if strings.Contains(path, ".transire") {
		return true
	}

	// Ignore vendor directory
	if strings.Contains(path, "/vendor/") {
		return true
	}

	// Only watch Go files and config files
	ext := filepath.Ext(path)
	if ext != ".go" && ext != ".yaml" && ext != ".yml" {
		return true
	}

	return false
}

// addDirectoryRecursive adds a directory and all subdirectories to the watcher
func (w *Watcher) addDirectoryRecursive(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip ignored directories
		if info.IsDir() && w.shouldIgnore(path) {
			return filepath.SkipDir
		}

		// Add directory to watcher
		if info.IsDir() {
			return w.watcher.Add(path)
		}

		return nil
	})
}
