package session

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewDaemonManager(t *testing.T) {
	tmpDir := t.TempDir()
	dm := NewDaemonManager(tmpDir)

	if dm.ProjectRoot != tmpDir {
		t.Errorf("ProjectRoot = %s, want %s", dm.ProjectRoot, tmpDir)
	}

	if dm.PIDFile == nil {
		t.Error("PIDFile should not be nil")
	}

	expectedLogPath := filepath.Join(tmpDir, ".agentdx", SessionLogFileName)
	if dm.logFile != expectedLogPath {
		t.Errorf("logFile = %s, want %s", dm.logFile, expectedLogPath)
	}
}

func TestDaemonManager_Status_NotRunning(t *testing.T) {
	tmpDir := t.TempDir()
	dm := NewDaemonManager(tmpDir)

	status, err := dm.Status()
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}

	if status.Running {
		t.Error("Status.Running should be false when no daemon is running")
	}

	if status.PID != 0 {
		t.Errorf("Status.PID = %d, want 0", status.PID)
	}

	if status.LogFile == "" {
		t.Error("Status.LogFile should not be empty")
	}
}

func TestDaemonManager_Status_StalePID(t *testing.T) {
	tmpDir := t.TempDir()
	dm := NewDaemonManager(tmpDir)

	// Write a stale PID file
	if err := dm.PIDFile.Write(-1); err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	status, err := dm.Status()
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}

	// Should not be running since PID -1 is invalid
	if status.Running {
		t.Error("Status.Running should be false for invalid PID")
	}

	if status.PID != -1 {
		t.Errorf("Status.PID = %d, want -1", status.PID)
	}
}

func TestDaemonManager_Status_CurrentProcess(t *testing.T) {
	tmpDir := t.TempDir()
	dm := NewDaemonManager(tmpDir)

	// Write current process PID
	currentPID := os.Getpid()
	if err := dm.PIDFile.Write(currentPID); err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	status, err := dm.Status()
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}

	if !status.Running {
		t.Error("Status.Running should be true for current process")
	}

	if status.PID != currentPID {
		t.Errorf("Status.PID = %d, want %d", status.PID, currentPID)
	}

	// StartTime should be set (using PID file mod time as proxy)
	if status.StartTime.IsZero() {
		t.Error("Status.StartTime should be set for running process")
	}
}

func TestDaemonManager_IsRunning(t *testing.T) {
	tmpDir := t.TempDir()
	dm := NewDaemonManager(tmpDir)

	// Not running
	running, err := dm.IsRunning()
	if err != nil {
		t.Fatalf("IsRunning() failed: %v", err)
	}
	if running {
		t.Error("IsRunning() should return false when no daemon")
	}

	// Write current process PID
	currentPID := os.Getpid()
	if err := dm.PIDFile.Write(currentPID); err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	// Running
	running, err = dm.IsRunning()
	if err != nil {
		t.Fatalf("IsRunning() failed: %v", err)
	}
	if !running {
		t.Error("IsRunning() should return true for current process")
	}
}

func TestDaemonManager_GetPID(t *testing.T) {
	tmpDir := t.TempDir()
	dm := NewDaemonManager(tmpDir)

	// No PID file
	pid, err := dm.GetPID()
	if err != nil {
		t.Fatalf("GetPID() failed: %v", err)
	}
	if pid != 0 {
		t.Errorf("GetPID() = %d, want 0", pid)
	}

	// With PID file
	testPID := 12345
	if err := dm.PIDFile.Write(testPID); err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	pid, err = dm.GetPID()
	if err != nil {
		t.Fatalf("GetPID() failed: %v", err)
	}
	if pid != testPID {
		t.Errorf("GetPID() = %d, want %d", pid, testPID)
	}
}

func TestDaemonManager_GetLogFile(t *testing.T) {
	tmpDir := t.TempDir()
	dm := NewDaemonManager(tmpDir)

	expectedLogPath := filepath.Join(tmpDir, ".agentdx", SessionLogFileName)
	if dm.GetLogFile() != expectedLogPath {
		t.Errorf("GetLogFile() = %s, want %s", dm.GetLogFile(), expectedLogPath)
	}
}

func TestDaemonManager_TailLog(t *testing.T) {
	tmpDir := t.TempDir()
	dm := NewDaemonManager(tmpDir)

	// Empty log
	lines, err := dm.TailLog(10)
	if err != nil {
		t.Fatalf("TailLog() failed: %v", err)
	}
	if len(lines) != 0 {
		t.Errorf("TailLog() = %v, want empty slice", lines)
	}

	// Create log file with some content (ensure directory exists)
	logContent := "line 1\nline 2\nline 3\nline 4\nline 5\n"
	if err := os.MkdirAll(filepath.Dir(dm.GetLogFile()), 0755); err != nil {
		t.Fatalf("Failed to create log directory: %v", err)
	}
	if err := os.WriteFile(dm.GetLogFile(), []byte(logContent), 0644); err != nil {
		t.Fatalf("Failed to write log file: %v", err)
	}

	// Tail 3 lines
	lines, err = dm.TailLog(3)
	if err != nil {
		t.Fatalf("TailLog() failed: %v", err)
	}

	if len(lines) != 3 {
		t.Fatalf("TailLog() returned %d lines, want 3", len(lines))
	}

	expectedLines := []string{"line 3", "line 4", "line 5"}
	for i, line := range lines {
		if line != expectedLines[i] {
			t.Errorf("TailLog()[%d] = %s, want %s", i, line, expectedLines[i])
		}
	}

	// Tail more lines than exist
	lines, err = dm.TailLog(10)
	if err != nil {
		t.Fatalf("TailLog() failed: %v", err)
	}
	if len(lines) != 5 {
		t.Errorf("TailLog() returned %d lines, want 5", len(lines))
	}
}

func TestDaemonManager_Stop_NotRunning(t *testing.T) {
	tmpDir := t.TempDir()
	dm := NewDaemonManager(tmpDir)

	ctx := context.Background()
	err := dm.Stop(ctx, false)
	if err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	// Should be no error even though nothing was running
}

func TestDaemonManager_Stop_StalePID(t *testing.T) {
	tmpDir := t.TempDir()
	dm := NewDaemonManager(tmpDir)

	// Write a stale PID
	if err := dm.PIDFile.Write(-1); err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	ctx := context.Background()
	err := dm.Stop(ctx, false)
	if err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	// PID file should be cleaned up
	if dm.PIDFile.Exists() {
		t.Error("PID file should be removed after stopping stale daemon")
	}
}

func TestParsePidString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{"valid PID", "12345", 12345, false},
		{"valid PID 1", "1", 1, false},
		{"zero", "0", 0, true},
		{"negative", "-1", 0, true},
		{"invalid", "abc", 0, true},
		{"with spaces", " 12345 ", 0, true},
		{"float", "123.45", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePidString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePidString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParsePidString() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []string
	}{
		{
			name:     "empty",
			input:    []byte{},
			expected: []string{},
		},
		{
			name:     "single line",
			input:    []byte("line 1"),
			expected: []string{"line 1"},
		},
		{
			name:     "multiple lines",
			input:    []byte("line 1\nline 2\nline 3"),
			expected: []string{"line 1", "line 2", "line 3"},
		},
		{
			name:     "lines with trailing newline",
			input:    []byte("line 1\nline 2\n"),
			expected: []string{"line 1", "line 2"},
		},
		{
			name:     "CRLF line endings",
			input:    []byte("line 1\r\nline 2\r\n"),
			expected: []string{"line 1", "line 2"},
		},
		{
			name:     "mixed line endings",
			input:    []byte("line 1\r\nline 2\nline 3"),
			expected: []string{"line 1", "line 2", "line 3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitLines(tt.input)
			if len(got) != len(tt.expected) {
				t.Fatalf("splitLines() returned %d lines, want %d", len(got), len(tt.expected))
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("splitLines()[%d] = %s, want %s", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestTimestamp(t *testing.T) {
	ts := timestamp()
	if ts == "" {
		t.Error("timestamp() should not return empty string")
	}

	// Parse it to verify it's a valid RFC3339 timestamp
	_, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		t.Errorf("timestamp() returned invalid RFC3339 format: %v", err)
	}
}

// TestDaemonManager_Log tests the logging functionality
func TestDaemonManager_Log(t *testing.T) {
	tmpDir := t.TempDir()
	dm := NewDaemonManager(tmpDir)

	// Log a message
	dm.log("test message")

	// Give it a moment to write
	time.Sleep(10 * time.Millisecond)

	// Check if log file exists and contains the message
	lines, err := dm.TailLog(1)
	if err != nil {
		t.Fatalf("TailLog() failed: %v", err)
	}

	if len(lines) == 0 {
		t.Fatal("Log file should contain at least one line")
	}

	// Check that the message contains our text
	// The line will have a timestamp prefix, so we just check for the message content
	found := false
	for _, line := range lines {
		if len(line) >= len("test message") && line[len(line)-len("test message"):] == "test message" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Log file should contain 'test message', got: %v", lines)
	}
}

// TestDaemonManager_Start_AlreadyRunning tests that start is idempotent
func TestDaemonManager_Start_AlreadyRunning(t *testing.T) {
	tmpDir := t.TempDir()
	dm := NewDaemonManager(tmpDir)

	// Write a "running" PID (current process)
	currentPID := os.Getpid()
	if err := dm.PIDFile.Write(currentPID); err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	ctx := context.Background()
	err := dm.Start(ctx)

	// Should not error even though "daemon" is already running
	if err != nil {
		t.Errorf("Start() should succeed when daemon already running: %v", err)
	}

	// PID file should still exist with same PID
	pid, _ := dm.GetPID()
	if pid != currentPID {
		t.Errorf("PID should remain %d, got %d", currentPID, pid)
	}
}
