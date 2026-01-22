//go:build !windows

package localsetup

import (
	"os/exec"
	"testing"
)

func TestIsDockerAvailable(t *testing.T) {
	// This test always runs - just reports Docker availability
	available := IsDockerAvailable()
	t.Logf("Docker available: %v", available)
}

func TestContainerExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Docker test in short mode")
	}
	if !IsDockerAvailable() {
		t.Skip("Docker not available")
	}

	// Test with non-existent container
	exists, err := ContainerExists("nonexistent-container-12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected non-existent container to return false")
	}
}

func TestContainerOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Docker test in short mode")
	}
	if !IsDockerAvailable() {
		t.Skip("Docker not available")
	}

	// This test uses a lightweight test container with a sleep command to keep it running
	testContainer := "agentdx-test-container"
	testImage := "alpine:latest"

	// Cleanup function
	cleanup := func() {
		exec.Command("docker", "rm", "-f", testContainer).Run()
	}
	t.Cleanup(cleanup)
	cleanup() // Ensure clean state

	// Test ContainerExists (should be false)
	exists, err := ContainerExists(testContainer)
	if err != nil {
		t.Fatalf("ContainerExists failed: %v", err)
	}
	if exists {
		t.Fatal("expected container to not exist initially")
	}

	// Test CreateContainer with a long-running command
	// Note: We use docker run directly with a sleep command instead of CreateContainer
	// because CreateContainer doesn't support specifying the command to run
	cmd := exec.Command("docker", "run", "-d", "--name", testContainer, testImage, "sleep", "30")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to create test container: %s: %v", string(output), err)
	}

	// Test ContainerExists (should be true now)
	exists, err = ContainerExists(testContainer)
	if err != nil {
		t.Fatalf("ContainerExists failed: %v", err)
	}
	if !exists {
		t.Fatal("expected container to exist after creation")
	}

	// Test ContainerRunning - give it a moment to ensure it's fully started
	running, err := ContainerRunning(testContainer)
	if err != nil {
		t.Fatalf("ContainerRunning failed: %v", err)
	}
	if !running {
		t.Error("expected container to be running after creation")
	}
}
