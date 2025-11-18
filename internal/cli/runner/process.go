package runner

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Process manages a running subprocess
type Process struct {
	binaryPath string
	cmd        *exec.Cmd
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewProcess creates a new Process manager
func NewProcess(binaryPath string) *Process {
	return &Process{
		binaryPath: binaryPath,
	}
}

// Start starts the process
func (p *Process) Start() error {
	// Ensure binary exists
	if _, err := os.Stat(p.binaryPath); err != nil {
		return fmt.Errorf("binary does not exist: %w", err)
	}

	// Create cancellable context
	p.ctx, p.cancel = context.WithCancel(context.Background())

	// Create command
	p.cmd = exec.CommandContext(p.ctx, p.binaryPath)
	p.cmd.Stdout = os.Stdout
	p.cmd.Stderr = os.Stderr
	p.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Create new process group
	}

	// Start the process
	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	log.Printf("Started process: %s (PID: %d)", p.binaryPath, p.cmd.Process.Pid)

	// Monitor process in goroutine
	go func() {
		if err := p.cmd.Wait(); err != nil {
			if p.ctx.Err() != context.Canceled {
				log.Printf("Process exited with error: %v", err)
			}
		}
	}()

	return nil
}

// Stop stops the process gracefully
func (p *Process) Stop() error {
	if p.cmd == nil || p.cmd.Process == nil {
		return nil // Nothing to stop
	}

	log.Printf("Stopping process (PID: %d)", p.cmd.Process.Pid)

	// Cancel context to signal shutdown
	if p.cancel != nil {
		p.cancel()
	}

	// Send SIGTERM to process group
	if err := syscall.Kill(-p.cmd.Process.Pid, syscall.SIGTERM); err != nil {
		log.Printf("Failed to send SIGTERM: %v", err)
	}

	// Wait for graceful shutdown with timeout
	done := make(chan error, 1)
	go func() {
		done <- p.cmd.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		// Force kill if not stopped gracefully
		log.Printf("Process did not stop gracefully, sending SIGKILL")
		_ = syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
		return fmt.Errorf("process did not stop gracefully")
	case err := <-done:
		if err != nil && p.ctx.Err() != context.Canceled {
			return fmt.Errorf("process stopped with error: %w", err)
		}
		return nil
	}
}

// IsRunning checks if the process is running
func (p *Process) IsRunning() bool {
	if p.cmd == nil || p.cmd.Process == nil {
		return false
	}

	// Try to send signal 0 to check if process exists
	err := p.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// Restart stops and starts the process
func (p *Process) Restart() error {
	if p.IsRunning() {
		if err := p.Stop(); err != nil {
			return fmt.Errorf("failed to stop process: %w", err)
		}
	}

	// Small delay to ensure port is released
	time.Sleep(100 * time.Millisecond)

	return p.Start()
}
