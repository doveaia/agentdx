package session

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"
)

const (
	// SessionLogFileName is the name of the session log file
	SessionLogFileName = "session.log"
	// GracefulShutdownTimeout is the maximum time to wait for graceful shutdown
	GracefulShutdownTimeout = 5 * time.Second
)

// DaemonOptions holds optional configuration for the daemon manager
type DaemonOptions struct {
	PgName string // PostgreSQL container name
	PgPort int    // PostgreSQL host port
}

// DaemonStatus represents the current state of the session daemon
type DaemonStatus struct {
	Running   bool      `json:"running"`
	PID       int       `json:"pid,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
	LogFile   string    `json:"log_file,omitempty"`
}

// DaemonManager handles session daemon lifecycle
type DaemonManager struct {
	ProjectRoot string
	PIDFile     *PIDFile
	logFile     string
	opts        DaemonOptions
	mu          sync.Mutex
}

// NewDaemonManager creates a daemon manager for the project
func NewDaemonManager(projectRoot string) *DaemonManager {
	return &DaemonManager{
		ProjectRoot: projectRoot,
		PIDFile:     NewPIDFile(projectRoot),
		logFile:     filepath.Join(projectRoot, ".agentdx", SessionLogFileName),
		opts:        DaemonOptions{}, // Default options
	}
}

// NewDaemonManagerWithOptions creates a daemon manager with custom options
func NewDaemonManagerWithOptions(projectRoot string, opts DaemonOptions) *DaemonManager {
	return &DaemonManager{
		ProjectRoot: projectRoot,
		PIDFile:     NewPIDFile(projectRoot),
		logFile:     filepath.Join(projectRoot, ".agentdx", SessionLogFileName),
		opts:        opts,
	}
}

// Start starts the watch daemon if not already running
// Returns nil if daemon is already running (idempotent)
func (d *DaemonManager) Start(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if already running
	running, err := d.PIDFile.IsProcessRunning()
	if err != nil {
		return fmt.Errorf("failed to check daemon status: %w", err)
	}
	if running {
		// Already running, log and return success
		d.log("Daemon already running (PID: %d)", mustGetPid(d.PIDFile))
		return nil
	}

	// Clean up stale PID file if it exists
	if d.PIDFile.Exists() {
		if err := d.PIDFile.Cleanup(); err != nil {
			d.log("Warning: failed to cleanup stale PID file: %v", err)
		}
	}

	// Get the agentdx binary path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Ensure log directory exists
	if err := os.MkdirAll(filepath.Dir(d.logFile), 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file for appending
	logF, err := os.OpenFile(d.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logF.Close()

	// Create the command with optional flags
	args := []string{"watch", "--daemon"}
	if d.opts.PgName != "" {
		args = append(args, "--pg-name", d.opts.PgName)
	}
	if d.opts.PgPort != 0 {
		args = append(args, "--pg-port", strconv.Itoa(d.opts.PgPort))
	}

	cmd := exec.CommandContext(ctx, execPath, args...)
	cmd.Dir = d.ProjectRoot

	// Redirect stdout and stderr to log file
	cmd.Stdout = logF
	cmd.Stderr = logF

	// Set up process group for clean termination (platform-specific)
	cmd.SysProcAttr = getSysProcAttr()

	// Start the daemon
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Write PID file
	pid := cmd.Process.Pid
	if err := d.PIDFile.Write(pid); err != nil {
		// If we can't write the PID file, kill the process and return error
		_ = cmd.Process.Kill()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	d.log("[%s] Daemon started (PID: %d)", timestamp(), pid)
	return nil
}

// Stop stops the watch daemon gracefully
// Uses SIGTERM with timeout, falls back to SIGKILL if force is true
func (d *DaemonManager) Stop(ctx context.Context, force bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if PID file exists
	if !d.PIDFile.Exists() {
		d.log("Stop requested but no PID file found")
		return nil // Not running, nothing to do
	}

	pid, err := d.PIDFile.Read()
	if err != nil {
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	// Validate PID is positive
	if pid <= 0 {
		// Invalid PID, just clean up the file
		d.log("Cleaning up invalid PID file (PID: %d)", pid)
		return d.PIDFile.Remove()
	}

	// Check if process is running
	process, err := os.FindProcess(pid)
	if err != nil {
		// Process doesn't exist, clean up stale PID file
		d.log("Cleaning up stale PID file (PID: %d)", pid)
		return d.PIDFile.Remove()
	}

	// Try to signal the process with signal 0 to check if it's running
	if err := process.Signal(syscall.Signal(0)); err != nil {
		if err == syscall.ESRCH {
			// Process doesn't exist, clean up stale PID file
			d.log("Cleaning up stale PID file (PID: %d)", pid)
			return d.PIDFile.Remove()
		}
		// On some systems, sending signal to invalid PID returns "process already released"
		// or other errors. Treat these as process not existing.
		d.log("Process not accessible (PID: %d), cleaning up PID file", pid)
		return d.PIDFile.Remove()
	}

	// Send SIGTERM for graceful shutdown
	d.log("[%s] Sending SIGTERM to daemon (PID: %d)", timestamp(), pid)
	if err := process.Signal(syscall.SIGTERM); err != nil {
		// Process already gone - this is actually success
		d.log("Daemon already terminated (PID: %d)", pid)
		return d.PIDFile.Remove()
	}

	// Wait for graceful shutdown (unless force is true)
	if !force {
		deadline := time.After(GracefulShutdownTimeout)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-deadline:
				// Timeout exceeded, fall through to force kill
				d.log("Graceful shutdown timeout, forcing...")
				force = true
			case <-ticker.C:
				// Check if process is still running
				if err := process.Signal(syscall.Signal(0)); err != nil {
					if err == syscall.ESRCH {
						// Process terminated gracefully
						d.log("[%s] Daemon stopped gracefully (PID: %d)", timestamp(), pid)
						return d.PIDFile.Remove()
					}
				}
				if !force {
					continue
				}
			case <-ctx.Done():
				return ctx.Err()
			}

			if force {
				break
			}
		}
	}

	// Force kill with SIGKILL
	d.log("[%s] Sending SIGKILL to daemon (PID: %d)", timestamp(), pid)
	if err := process.Signal(syscall.SIGKILL); err != nil {
		// Process already gone - this is actually success
		// ESRCH is "no such process", but there may be other errors like "process already finished"
		d.log("Daemon already terminated (PID: %d)", pid)
		return d.PIDFile.Remove()
	}

	// Give it a moment to terminate
	time.Sleep(100 * time.Millisecond)

	d.log("[%s] Daemon stopped (PID: %d)", timestamp(), pid)
	return d.PIDFile.Remove()
}

// Status returns the current daemon status
func (d *DaemonManager) Status() (DaemonStatus, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	status := DaemonStatus{
		LogFile: d.logFile,
	}

	if !d.PIDFile.Exists() {
		status.Running = false
		return status, nil
	}

	pid, err := d.PIDFile.Read()
	if err != nil {
		return status, fmt.Errorf("failed to read PID file: %w", err)
	}

	status.PID = pid

	// Check if process is running
	running, err := d.PIDFile.IsProcessRunning()
	if err != nil {
		return status, fmt.Errorf("failed to check process status: %w", err)
	}

	status.Running = running

	if running {
		// Try to get process start time (platform-specific)
		// For simplicity, we'll use the PID file modification time as a proxy
		if info, err := os.Stat(d.PIDFile.Path); err == nil {
			status.StartTime = info.ModTime()
		}
	}

	return status, nil
}

// log writes a message to the session log file
func (d *DaemonManager) log(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp(), msg)

	// Ensure log directory exists
	if err := os.MkdirAll(filepath.Dir(d.logFile), 0755); err != nil {
		return
	}

	// Open file in append mode, ignore errors - logging is best-effort
	f, err := os.OpenFile(d.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(logEntry)
}

// timestamp returns a formatted timestamp for logging
func timestamp() string {
	return time.Now().Format(time.RFC3339)
}

// mustGetPid gets the PID from the PID file, panicking on error
// This is used internally when we've already verified the process is running
func mustGetPid(pf *PIDFile) int {
	pid, err := pf.Read()
	if err != nil {
		return -1
	}
	return pid
}

// IsRunning returns whether the daemon is currently running
func (d *DaemonManager) IsRunning() (bool, error) {
	status, err := d.Status()
	if err != nil {
		return false, err
	}
	return status.Running, nil
}

// GetLogFile returns the path to the session log file
func (d *DaemonManager) GetLogFile() string {
	return d.logFile
}

// TailLog returns the last n lines from the session log file
func (d *DaemonManager) TailLog(n int) ([]string, error) {
	data, err := os.ReadFile(d.logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	// Split into lines and get last n
	lines := splitLines(data)
	if len(lines) <= n {
		return lines, nil
	}
	return lines[len(lines)-n:], nil
}

// splitLines splits bytes into lines, handling both \n and \r\n
func splitLines(data []byte) []string {
	var lines []string
	var line []byte

	for _, b := range data {
		if b == '\n' {
			lines = append(lines, string(line))
			line = nil
		} else if b == '\r' {
			// Skip \r in \r\n sequences
			continue
		} else {
			line = append(line, b)
		}
	}

	// Add last line if it doesn't end with newline
	if len(line) > 0 {
		lines = append(lines, string(line))
	}

	return lines
}

// GetPID returns the PID of the daemon if running, 0 otherwise
func (d *DaemonManager) GetPID() (int, error) {
	if !d.PIDFile.Exists() {
		return 0, nil
	}
	return d.PIDFile.Read()
}

// ParsePidString parses a PID string and validates it
func ParsePidString(s string) (int, error) {
	pid, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid PID: %w", err)
	}
	if pid <= 0 {
		return 0, fmt.Errorf("invalid PID: must be positive")
	}
	return pid, nil
}
