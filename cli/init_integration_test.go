//go:build !windows

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/doveaia/agentdx/config"
	"github.com/doveaia/agentdx/localsetup"
)

// TestMain runs setup and cleanup for all integration tests in this package.
func TestMain(m *testing.M) {
	// Clean up any stale containers before running tests
	if localsetup.IsDockerAvailable() {
		_ = localsetup.RemoveContainer("agentdx-postgres")
	}
	os.Exit(m.Run())
}

// TestSetupPostgresBackend_DockerAvailable verifies that setupPostgresBackend
// returns a valid SetupResult when Docker is available and creates/starts the container.
func TestSetupPostgresBackend_DockerAvailable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if !localsetup.IsDockerAvailable() {
		t.Skip("Docker not available")
	}

	// Clean up any existing container before running the test
	_ = localsetup.RemoveContainer("agentdx-postgres")

	tmpDir, err := os.MkdirTemp("", "agentdx-pg-setup-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
		// Clean up the container after the test
		_ = localsetup.RemoveContainer("agentdx-postgres")
	})

	result, err := setupPostgresBackend(tmpDir)
	if err != nil {
		t.Fatalf("setupPostgresBackend() failed: %v", err)
	}
	if result == nil {
		t.Fatal("setupPostgresBackend() returned nil result, expected non-nil when Docker is available")
	}

	// Verify result fields
	if result.DSN == "" {
		t.Error("result.DSN is empty")
	}
	if !strings.HasPrefix(result.DSN, "postgres://") {
		t.Errorf("result.DSN should start with 'postgres://', got: %s", result.DSN)
	}
	if result.DatabaseName == "" {
		t.Error("result.DatabaseName is empty")
	}
	if result.ContainerName != "agentdx-postgres" {
		t.Errorf("result.ContainerName = %q, want 'agentdx-postgres'", result.ContainerName)
	}
	if !result.DockerUsed {
		t.Error("result.DockerUsed should be true when Docker is available")
	}
	if !result.ComposeGenerated {
		t.Error("result.ComposeGenerated should be true")
	}
	if result.ComposeFilePath == "" {
		t.Error("result.ComposeFilePath is empty")
	}

	// Verify compose.yaml was created
	if _, err := os.Stat(result.ComposeFilePath); os.IsNotExist(err) {
		t.Errorf("compose.yaml was not created at %s", result.ComposeFilePath)
	}
}

// TestSetupPostgresBackend_NoDocker verifies that setupPostgresBackend
// returns nil, nil when Docker is not available.
func TestSetupPostgresBackend_NoDocker(t *testing.T) {
	// This test can only run when Docker is actually not available.
	// If Docker IS available, we skip this test since we can't mock the unavailability.
	if localsetup.IsDockerAvailable() {
		t.Skip("Docker is available - cannot test Docker unavailable path")
	}

	tmpDir, err := os.MkdirTemp("", "agentdx-pg-nodocker-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	result, err := setupPostgresBackend(tmpDir)
	if err != nil {
		t.Fatalf("setupPostgresBackend() returned error: %v", err)
	}
	if result != nil {
		t.Errorf("setupPostgresBackend() returned non-nil result when Docker unavailable: %+v", result)
	}

	// But compose.yaml should still be generated
	composePath := filepath.Join(tmpDir, ".agentdx", "compose.yaml")
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		t.Error("compose.yaml was not generated even when Docker is unavailable")
	}
}

// TestSetupPostgresBackend_Idempotent verifies that calling setupPostgresBackend
// multiple times with the same directory works correctly (container already running).
func TestSetupPostgresBackend_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if !localsetup.IsDockerAvailable() {
		t.Skip("Docker not available")
	}

	// Clean up any existing container before running the test
	_ = localsetup.RemoveContainer("agentdx-postgres")

	tmpDir, err := os.MkdirTemp("", "agentdx-pg-idempotent-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
		// Clean up the container after the test
		_ = localsetup.RemoveContainer("agentdx-postgres")
	})

	// First call
	result1, err1 := setupPostgresBackend(tmpDir)
	if err1 != nil {
		t.Fatalf("first setupPostgresBackend() failed: %v", err1)
	}
	if result1 == nil {
		t.Fatal("first setupPostgresBackend() returned nil")
	}

	// Second call (container should already be running)
	result2, err2 := setupPostgresBackend(tmpDir)
	if err2 != nil {
		t.Fatalf("second setupPostgresBackend() failed: %v", err2)
	}
	if result2 == nil {
		t.Fatal("second setupPostgresBackend() returned nil")
	}

	// Results should be consistent
	if result1.DSN != result2.DSN {
		t.Errorf("DSN changed between calls: %q -> %q", result1.DSN, result2.DSN)
	}
	if result1.DatabaseName != result2.DatabaseName {
		t.Errorf("DatabaseName changed between calls: %q -> %q", result1.DatabaseName, result2.DatabaseName)
	}
}

// TestInitPostgresBackend_DSNConfigured verifies that when config is saved
// with a postgres DSN, it can be loaded correctly.
func TestInitPostgresBackend_DSNConfigured(t *testing.T) {
	tmpDir := t.TempDir()

	testDSN := "postgres://testuser:testpass@localhost:55432/testdb?sslmode=disable"

	// Create a config with postgres DSN
	cfg := config.DefaultConfig()
	cfg.Index.Store.Postgres.DSN = testDSN

	if err := cfg.Save(tmpDir); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify config can be loaded
	loadedCfg, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loadedCfg.Index.Store.Postgres.DSN != testDSN {
		t.Errorf("DSN = %q, want %q", loadedCfg.Index.Store.Postgres.DSN, testDSN)
	}
}

// TestSetupPostgresBackend_ResultFromDockerRun verifies the result
// structure matches what's expected from localsetup.RunLocalSetup.
func TestSetupPostgresBackend_ResultStructure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if !localsetup.IsDockerAvailable() {
		t.Skip("Docker not available")
	}

	// Clean up any existing container before running the test
	_ = localsetup.RemoveContainer("agentdx-postgres")

	tmpDir, err := os.MkdirTemp("", "agentdx-pg-struct-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
		// Clean up the container after the test
		_ = localsetup.RemoveContainer("agentdx-postgres")
	})

	result, err := setupPostgresBackend(tmpDir)
	if err != nil {
		t.Fatalf("setupPostgresBackend() failed: %v", err)
	}

	// Verify Mode is set
	if result.Mode != "local" {
		t.Errorf("result.Mode = %q, want 'local'", result.Mode)
	}

	// Verify DSN contains expected components
	expectedDSN := "postgres://agentdx:agentdx@localhost:55432/"
	if !strings.HasPrefix(result.DSN, expectedDSN) {
		t.Errorf("result.DN should start with %q, got %q", expectedDSN, result.DSN)
	}
	// Should end with database name and sslmode parameter
	if !strings.HasSuffix(result.DSN, "?sslmode=disable") && !strings.Contains(result.DSN, "?sslmode=disable") {
		t.Logf("Note: DSN doesn't end with ?sslmode=disable: %s", result.DSN)
	}
}
