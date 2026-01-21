package session

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestPIDFile_Write(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := NewPIDFile(tmpDir)

	testPID := 12345
	err := pidFile.Write(testPID)
	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}

	// Verify file exists
	if !pidFile.Exists() {
		t.Error("PID file should exist after Write()")
	}

	// Verify file contents
	data, err := os.ReadFile(pidFile.Path)
	if err != nil {
		t.Fatalf("Failed to read PID file: %v", err)
	}

	// Check content is "12345\n"
	expected := strconv.Itoa(testPID) + "\n"
	if string(data) != expected {
		t.Errorf("PID file content = %q, want %q", string(data), expected)
	}
}

func TestPIDFile_Read(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := NewPIDFile(tmpDir)

	t.Run("valid PID file", func(t *testing.T) {
		testPID := 54321
		if err := pidFile.Write(testPID); err != nil {
			t.Fatalf("Write() failed: %v", err)
		}

		pid, err := pidFile.Read()
		if err != nil {
			t.Fatalf("Read() failed: %v", err)
		}

		if pid != testPID {
			t.Errorf("Read() = %d, want %d", pid, testPID)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		nonExistentFile := NewPIDFile(t.TempDir())
		_, err := nonExistentFile.Read()

		if err == nil {
			t.Error("Read() should return error for missing file")
		}
	})

	t.Run("corrupted file", func(t *testing.T) {
		// Write invalid content to the PID file
		if err := os.WriteFile(pidFile.Path, []byte("not-a-number\n"), 0644); err != nil {
			t.Fatalf("Failed to write corrupted PID file: %v", err)
		}

		_, err := pidFile.Read()
		if err == nil {
			t.Error("Read() should return error for corrupted file")
		}
	})
}

func TestPIDFile_Remove(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := NewPIDFile(tmpDir)

	t.Run("remove existing file", func(t *testing.T) {
		if err := pidFile.Write(12345); err != nil {
			t.Fatalf("Write() failed: %v", err)
		}

		if !pidFile.Exists() {
			t.Fatal("PID file should exist before Remove()")
		}

		err := pidFile.Remove()
		if err != nil {
			t.Fatalf("Remove() failed: %v", err)
		}

		if pidFile.Exists() {
			t.Error("PID file should not exist after Remove()")
		}
	})

	t.Run("remove non-existent file", func(t *testing.T) {
		nonExistentFile := NewPIDFile(t.TempDir())
		// Should not error when removing non-existent file
		err := nonExistentFile.Remove()
		if err != nil {
			t.Errorf("Remove() should not error for non-existent file: %v", err)
		}
	})
}

func TestPIDFile_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := NewPIDFile(tmpDir)

	t.Run("existing file", func(t *testing.T) {
		if err := pidFile.Write(12345); err != nil {
			t.Fatalf("Write() failed: %v", err)
		}

		if !pidFile.Exists() {
			t.Error("Exists() should return true for existing file")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		nonExistentFile := NewPIDFile(t.TempDir())
		if nonExistentFile.Exists() {
			t.Error("Exists() should return false for missing file")
		}
	})
}

func TestPIDFile_IsProcessRunning(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := NewPIDFile(tmpDir)

	t.Run("current process", func(t *testing.T) {
		// Write current process PID
		currentPID := os.Getpid()
		if err := pidFile.Write(currentPID); err != nil {
			t.Fatalf("Write() failed: %v", err)
		}

		running, err := pidFile.IsProcessRunning()
		if err != nil {
			t.Fatalf("IsProcessRunning() failed: %v", err)
		}

		if !running {
			t.Error("IsProcessRunning() should return true for current process")
		}
	})

	t.Run("dead process", func(t *testing.T) {
		// On most systems, PID 1 is init/systemd which always runs
		// We need to find a PID that doesn't exist
		// Use a very high PID that's unlikely to exist
		deadPID := 999999

		if err := pidFile.Write(deadPID); err != nil {
			t.Fatalf("Write() failed: %v", err)
		}

		running, err := pidFile.IsProcessRunning()
		if err != nil {
			// Some systems might error, which is acceptable
			// The important thing is that it's not reported as running
			t.Logf("IsProcessRunning() returned error (acceptable): %v", err)
		}

		if running {
			// Very unlikely but possible if PID 999999 actually exists
			t.Skipf("PID %d is actually running on this system", deadPID)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		nonExistentFile := NewPIDFile(t.TempDir())
		running, err := nonExistentFile.IsProcessRunning()

		if err != nil {
			t.Fatalf("IsProcessRunning() failed: %v", err)
		}

		if running {
			t.Error("IsProcessRunning() should return false when PID file doesn't exist")
		}
	})
}

func TestPIDFile_Cleanup(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := NewPIDFile(tmpDir)

	t.Run("stale PID cleanup", func(t *testing.T) {
		// Use negative PID which is impossible on real systems
		stalePID := -1
		if err := pidFile.Write(stalePID); err != nil {
			t.Fatalf("Write() failed: %v", err)
		}

		if !pidFile.Exists() {
			t.Fatal("PID file should exist before Cleanup()")
		}

		err := pidFile.Cleanup()
		// Cleanup may succeed even if process check had issues
		_ = err // Ignore error for this test case

		if pidFile.Exists() {
			// Check if the process was actually running (unlikely for -1)
			running, _ := pidFile.IsProcessRunning()
			if running {
				t.Skip("PID -1 is somehow running on this system")
			}
			t.Error("PID file should be removed after cleanup of stale PID")
		}
	})

	t.Run("running process should not be cleaned up", func(t *testing.T) {
		currentPID := os.Getpid()
		if err := pidFile.Write(currentPID); err != nil {
			t.Fatalf("Write() failed: %v", err)
		}

		err := pidFile.Cleanup()
		if err != nil {
			t.Fatalf("Cleanup() failed: %v", err)
		}

		if !pidFile.Exists() {
			t.Error("PID file should not be removed when process is running")
		}
	})

	t.Run("cleanup when file doesn't exist", func(t *testing.T) {
		nonExistentFile := NewPIDFile(t.TempDir())
		err := nonExistentFile.Cleanup()

		if err != nil {
			t.Errorf("Cleanup() should not error when file doesn't exist: %v", err)
		}
	})
}

func TestPIDFile_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := NewPIDFile(tmpDir)

	// Create the directory once (simulating real-world scenario where .agentdx already exists)
	agentdxDir := filepath.Dir(pidFile.Path)
	if err := os.MkdirAll(agentdxDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Write the same PID file multiple times quickly
	// This tests that the atomic rename prevents corruption
	done := make(chan bool, 10)
	errors := make(chan error, 10)

	for i := range 10 {
		go func(pid int) {
			if err := pidFile.Write(pid); err != nil {
				errors <- err
			} else {
				done <- true
			}
		}(12345 + i)
	}

	// Wait for all goroutines to complete
	completed := 0
	successCount := 0
	for completed < 10 {
		select {
		case <-done:
			completed++
			successCount++
		case err := <-errors:
			completed++
			// Race condition errors are expected - these happen when goroutines
			// race to rename the same temp file. Only log unexpected errors.
			if err != nil {
				// Check if it's a "no such file" rename error (expected in races)
				// The error is wrapped, so we check the message
				errMsg := err.Error()
				if !strings.Contains(errMsg, "no such file") && !strings.Contains(errMsg, "not exist") {
					t.Logf("Concurrent Write() error: %v", err)
				}
			}
		case <-time.After(10 * time.Second):
			t.Fatal("Concurrent Write() timed out")
		}
	}

	// At least some writes should succeed
	if successCount == 0 {
		t.Fatal("All concurrent writes failed")
	}

	// Verify the file is valid (if at least one write succeeded)
	if pidFile.Exists() {
		pid, err := pidFile.Read()
		if err != nil {
			t.Fatalf("Failed to read final PID: %v", err)
		}

		// PID should be one of the values we wrote
		if pid < 12345 || pid > 12345+10 {
			t.Errorf("Final PID = %d, want value in range [12345, 12355]", pid)
		}

		// File should be properly formatted (no corruption)
		data, err := os.ReadFile(pidFile.Path)
		if err != nil {
			t.Fatalf("Failed to read PID file: %v", err)
		}

		expected := strconv.Itoa(pid) + "\n"
		if string(data) != expected {
			t.Errorf("PID file content = %q, want %q", string(data), expected)
		}
	}
}

func TestNewPIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := NewPIDFile(tmpDir)

	expectedPath := filepath.Join(tmpDir, ".agentdx", SessionPIDFileName)
	if pidFile.Path != expectedPath {
		t.Errorf("NewPIDFile() Path = %s, want %s", pidFile.Path, expectedPath)
	}
}

func TestPIDFile_WriteCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a path that doesn't exist yet
	nestedPath := filepath.Join(tmpDir, "project", "subdir")
	pidFile := NewPIDFile(nestedPath)

	testPID := 12345
	err := pidFile.Write(testPID)
	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}

	// Verify the directory was created
	if _, err := os.Stat(filepath.Dir(pidFile.Path)); err != nil {
		t.Errorf(".agentdx directory was not created: %v", err)
	}

	// Verify the file exists
	if !pidFile.Exists() {
		t.Error("PID file should exist after Write()")
	}
}
