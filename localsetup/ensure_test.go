package localsetup

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestEnsurePostgresRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	if !IsDockerAvailable() {
		t.Skip("Docker not available")
	}

	// Create a unique container name for this test
	containerName := "agentdx-test-ensure-" + time.Now().Format("20060102150405")
	testPort := 55433 // Use a different port to avoid conflicts

	ctx := context.Background()

	// Clean up any existing container with this name
	_ = RemoveContainer(containerName)

	// Create a temp directory for the project root
	tempDir := t.TempDir()

	t.Run("creates container when none exists", func(t *testing.T) {
		opts := ContainerOptions{
			Name: containerName,
			Port: testPort,
		}

		dsn, err := EnsurePostgresRunning(ctx, tempDir, opts)
		if err != nil {
			t.Fatalf("EnsurePostgresRunning failed: %v", err)
		}

		if dsn == "" {
			t.Error("expected non-empty DSN")
		}

		// Verify container exists
		exists, err := ContainerExists(containerName)
		if err != nil {
			t.Fatalf("failed to check container existence: %v", err)
		}
		if !exists {
			t.Error("container should exist")
		}

		// Verify container is running
		running, err := ContainerRunning(containerName)
		if err != nil {
			t.Fatalf("failed to check container state: %v", err)
		}
		if !running {
			t.Error("container should be running")
		}

		// Clean up
		_ = RemoveContainer(containerName)
	})

	t.Run("reuses existing running container", func(t *testing.T) {
		opts := ContainerOptions{
			Name: containerName,
			Port: testPort,
		}

		// First call creates the container
		dsn1, err := EnsurePostgresRunning(ctx, tempDir, opts)
		if err != nil {
			t.Fatalf("first EnsurePostgresRunning failed: %v", err)
		}

		// Second call should reuse the container
		dsn2, err := EnsurePostgresRunning(ctx, tempDir, opts)
		if err != nil {
			t.Fatalf("second EnsurePostgresRunning failed: %v", err)
		}

		// DSNs should be the same
		if dsn1 != dsn2 {
			t.Errorf("DSN mismatch: %s != %s", dsn1, dsn2)
		}

		// Clean up
		_ = RemoveContainer(containerName)
	})

	t.Run("starts stopped container", func(t *testing.T) {
		opts := ContainerOptions{
			Name: containerName,
			Port: testPort,
		}

		// Create the container
		_, err := EnsurePostgresRunning(ctx, tempDir, opts)
		if err != nil {
			t.Fatalf("EnsurePostgresRunning failed: %v", err)
		}

		// Stop the container
		if err := startTestCommand("docker", "stop", containerName); err != nil {
			t.Fatalf("failed to stop container: %v", err)
		}

		// Verify it's stopped
		running, err := ContainerRunning(containerName)
		if err != nil {
			t.Fatalf("failed to check container state: %v", err)
		}
		if running {
			t.Error("container should be stopped")
		}

		// EnsurePostgresRunning should start it again
		dsn, err := EnsurePostgresRunning(ctx, tempDir, opts)
		if err != nil {
			t.Fatalf("EnsurePostgresRunning failed to start stopped container: %v", err)
		}

		if dsn == "" {
			t.Error("expected non-empty DSN after starting stopped container")
		}

		// Verify it's running again
		running, err = ContainerRunning(containerName)
		if err != nil {
			t.Fatalf("failed to check container state: %v", err)
		}
		if !running {
			t.Error("container should be running after EnsurePostgresRunning")
		}

		// Clean up
		_ = RemoveContainer(containerName)
	})

	t.Run("returns correct DSN with custom port", func(t *testing.T) {
		customPort := 55434
		opts := ContainerOptions{
			Name: containerName + "-dsn",
			Port: customPort,
		}

		dsn, err := EnsurePostgresRunning(ctx, tempDir, opts)
		if err != nil {
			t.Fatalf("EnsurePostgresRunning failed: %v", err)
		}

		// Verify DSN contains the custom port
		if !strings.Contains(dsn, fmt.Sprintf(":%d/", customPort)) && !strings.Contains(dsn, fmt.Sprintf(":%d?", customPort)) {
			t.Errorf("DSN does not contain expected port %d: %s", customPort, dsn)
		}

		// Clean up
		_ = RemoveContainer(containerName + "-dsn")
	})
}

func TestIsPortInUse(t *testing.T) {
	// Test with a port that's likely not in use
	freePort := 55435
	if isPortInUse(freePort) {
		t.Skipf("port %d is in use, skipping test", freePort)
	}

	// Listen on the port and verify it's detected as in use
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", freePort))
	if err != nil {
		t.Fatalf("failed to listen on port: %v", err)
	}
	defer listener.Close()

	if !isPortInUse(freePort) {
		t.Error("expected port to be detected as in use")
	}
}

// Helper function to run test commands
func startTestCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}
