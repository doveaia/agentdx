//go:build !windows

package localsetup

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain runs setup and cleanup for all tests in this package.
func TestMain(m *testing.M) {
	// Clean up any stale containers before running tests
	if IsDockerAvailable() {
		_ = RemoveContainer(containerName)
	}
	os.Exit(m.Run())
}

func TestRunLocalSetup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if !IsDockerAvailable() {
		t.Skip("Docker not available")
	}

	// Clean up any existing container before running the test
	_ = RemoveContainer(containerName)

	// Create a temporary directory for the test project
	tmpDir, err := os.MkdirTemp("", "agentdx-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
		// Clean up the container after the test
		_ = RemoveContainer(containerName)
	})

	// Create a uniquely named test directory
	projectDir := filepath.Join(tmpDir, "test-local-setup-project")
	if err := os.Mkdir(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Run the local setup
	result, err := RunLocalSetup(projectDir)
	if err != nil {
		t.Fatalf("RunLocalSetup failed: %v", err)
	}

	// Verify result
	if result.Mode != "local" {
		t.Errorf("expected Mode='local', got %q", result.Mode)
	}
	if result.DatabaseName != "agentdx_test_local_setup_project" {
		t.Errorf("expected DatabaseName='agentdx_test_local_setup_project', got %q", result.DatabaseName)
	}
	if !result.DockerUsed {
		t.Error("expected DockerUsed=true when Docker is available")
	}
	if !result.ComposeGenerated {
		t.Error("expected ComposeGenerated=true")
	}

	// Verify compose.yaml was created
	composePath := filepath.Join(projectDir, ".agentdx", "compose.yaml")
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		t.Error("compose.yaml was not created")
	}
}

func TestRunLocalSetup_EmptySlug(t *testing.T) {
	// Create a temp directory and manually rename it to have only special chars
	// Since os.MkdirTemp adds random prefix, we need a different approach.
	// Instead, we'll use a subdirectory with special chars.
	tmpDir, err := os.MkdirTemp("", "agentdx-slug-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	// Create a subdirectory that will produce an empty slug after ToSlug
	// ToSlug removes all non-alphanumeric chars except underscore
	// So "!@#$%" becomes empty string
	specialDir := filepath.Join(tmpDir, "!@#$%")
	if err := os.Mkdir(specialDir, 0755); err != nil {
		// Some filesystems may not allow this, skip if so
		t.Skipf("cannot create directory with special chars: %v", err)
	}

	_, err = RunLocalSetup(specialDir)
	if err == nil {
		t.Error("expected error for empty slug, got nil")
	}
}

func TestRunLocalSetup_NoDockerFallback(t *testing.T) {
	// This test verifies behavior when Docker operations would fail
	// We can't easily simulate no Docker, but we can verify compose.yaml
	// is still generated even in that case

	tmpDir, err := os.MkdirTemp("", "agentdx-fallback-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	projectDir := filepath.Join(tmpDir, "fallback-project")
	if err := os.Mkdir(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Even if Docker is available, verify compose.yaml generation
	result, err := RunLocalSetup(projectDir)
	if err != nil && !IsDockerAvailable() {
		// Expected when Docker not available - still check compose was generated
		t.Logf("Expected error without Docker: %v", err)
	}

	if result != nil && !result.ComposeGenerated {
		t.Error("compose.yaml should always be generated")
	}

	composePath := filepath.Join(projectDir, ".agentdx", "compose.yaml")
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		t.Error("compose.yaml should be generated even without Docker")
	}
}
