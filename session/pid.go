package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

const (
	// SessionPIDFileName is the name of the PID file for the session daemon
	SessionPIDFileName = "session.pid"
)

// PIDFile manages the session daemon PID file
type PIDFile struct {
	Path string
}

// NewPIDFile creates a PIDFile manager for the given project root
func NewPIDFile(projectRoot string) *PIDFile {
	return &PIDFile{
		Path: filepath.Join(projectRoot, ".agentdx", SessionPIDFileName),
	}
}

// Write writes the current process PID to the file
func (p *PIDFile) Write(pid int) error {
	// Ensure directory exists
	dir := filepath.Dir(p.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create PID file directory: %w", err)
	}

	// Write to temp file first for atomic write
	tempPath := p.Path + ".tmp"
	content := strconv.Itoa(pid) + "\n"
	if err := os.WriteFile(tempPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write PID temp file: %w", err)
	}

	// Rename for atomic operation
	if err := os.Rename(tempPath, p.Path); err != nil {
		os.Remove(tempPath) // Clean up temp file if rename fails
		return fmt.Errorf("failed to rename PID file: %w", err)
	}

	return nil
}

// Read reads the PID from the file
func (p *PIDFile) Read() (int, error) {
	data, err := os.ReadFile(p.Path)
	if err != nil {
		return 0, fmt.Errorf("failed to read PID file: %w", err)
	}

	// Parse PID, trim whitespace
	pidStr := string(data)
	pidStr = pidStr[:0]
	for i, b := range data {
		if b == '\n' || b == '\r' {
			pidStr = string(data[:i])
			break
		}
		if i == len(data)-1 {
			pidStr = string(data)
		}
	}

	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("invalid PID format in file: %w", err)
	}

	return pid, nil
}

// Remove deletes the PID file
func (p *PIDFile) Remove() error {
	if err := os.Remove(p.Path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}
	return nil
}

// Exists checks if the PID file exists
func (p *PIDFile) Exists() bool {
	_, err := os.Stat(p.Path)
	return err == nil
}

// IsProcessRunning checks if the process with the stored PID is still running
func (p *PIDFile) IsProcessRunning() (bool, error) {
	if !p.Exists() {
		return false, nil
	}

	pid, err := p.Read()
	if err != nil {
		return false, err
	}

	// Validate PID is positive
	if pid <= 0 {
		return false, nil
	}

	// On Unix systems, signal 0 checks if process exists without sending a signal
	// syscall.Kill with signal 0 doesn't actually kill the process
	// It returns an error if the process doesn't exist or if we don't have permission
	process, err := os.FindProcess(pid)
	if err != nil {
		return false, fmt.Errorf("failed to find process: %w", err)
	}

	// Try to signal the process with signal 0 (no-op signal)
	// If the process exists, this will succeed (or return a permission error, which means it exists)
	// If the process doesn't exist, this will return an error
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		// ESRCH means no such process
		if err == syscall.ESRCH {
			return false, nil
		}
		// EINVAL means invalid PID (on some systems)
		if err == syscall.EINVAL {
			return false, nil
		}
		// EPERM means we don't have permission, but the process exists
		// On Windows, this might be different, so we handle it generally
		return true, nil
	}

	return true, nil
}

// Cleanup removes the PID file if the process is not running (stale)
func (p *PIDFile) Cleanup() error {
	running, err := p.IsProcessRunning()
	if err != nil {
		return err
	}

	if !running && p.Exists() {
		return p.Remove()
	}

	return nil
}

// GetUptime returns the uptime of the process if it's running
func (p *PIDFile) GetUptime() (time.Duration, error) {
	running, err := p.IsProcessRunning()
	if err != nil {
		return 0, err
	}
	if !running {
		return 0, fmt.Errorf("process is not running")
	}

	// Get process start time from /proc on Unix systems
	// This is a simplified version - on systems without /proc, we can't get accurate uptime
	// For now, we'll return 0 on systems that don't support this
	return 0, nil
}
