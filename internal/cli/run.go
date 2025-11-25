// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	var port string
	var watch bool
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the current project locally",
		RunE: func(cmd *cobra.Command, args []string) error {
			if watch {
				return runWithWatch(cmd.Context(), port)
			}
			return runOnce(cmd.Context(), port)
		},
	}
	cmd.Flags().StringVar(&port, "port", "", "port to serve locally (overrides PORT/TRANSIRE_PORT)")
	cmd.Flags().BoolVar(&watch, "watch", true, "restart automatically when source files change")
	return cmd
}

func runOnce(ctx context.Context, port string) error {
	runCmd := exec.CommandContext(ctx, "go", "run", "./cmd/app")
	if port != "" {
		runCmd.Env = append(os.Environ(),
			fmt.Sprintf("PORT=%s", port),
			fmt.Sprintf("TRANSIRE_PORT=%s", port),
			fmt.Sprintf("TRANSIRE_HTTP_ADDR=:%s", port),
		)
	} else {
		runCmd.Env = os.Environ()
	}
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr
	runCmd.Stdin = os.Stdin
	return runCmd.Run()
}

func runWithWatch(ctx context.Context, port string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	root, err := os.Getwd()
	if err != nil {
		return err
	}

	if err := addWatchDirs(watcher, root); err != nil {
		return err
	}

	restartCh := make(chan struct{}, 1)

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if shouldRestart(event) {
					select {
					case restartCh <- struct{}{}:
					default:
					}
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						_ = watcher.Add(event.Name)
					}
				}
			case <-watcher.Errors:
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		runCtx, cancel := context.WithCancel(ctx)
		cmd := exec.CommandContext(runCtx, "go", "run", "./cmd/app")
		if port != "" {
			cmd.Env = append(os.Environ(),
				fmt.Sprintf("PORT=%s", port),
				fmt.Sprintf("TRANSIRE_PORT=%s", port),
				fmt.Sprintf("TRANSIRE_HTTP_ADDR=:%s", port),
			)
		} else {
			cmd.Env = os.Environ()
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Start(); err != nil {
			cancel()
			return err
		}

		waitDone := make(chan error, 1)
		go func() {
			waitDone <- cmd.Wait()
		}()

		select {
		case <-ctx.Done():
			cancel()
			<-waitDone
			return ctx.Err()
		case err := <-waitDone:
			cancel()
			return err
		case <-restartCh:
			cancel()
			<-waitDone
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func addWatchDirs(w *fsnotify.Watcher, root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		name := info.Name()
		if strings.HasPrefix(name, ".git") || name == "dist" || name == "node_modules" || name == "vendor" {
			return filepath.SkipDir
		}
		return w.Add(path)
	})
}

func shouldRestart(event fsnotify.Event) bool {
	if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
		return false
	}
	ext := strings.ToLower(filepath.Ext(event.Name))
	switch ext {
	case ".go", ".yaml", ".yml":
		return true
	default:
		return false
	}
}
