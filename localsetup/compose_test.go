package localsetup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateComposeYAML(t *testing.T) {
	content := GenerateComposeYAML()

	// Verify required content
	checks := []string{
		"doveaia/timescaledb:latest-pg17-ts",
		"agentdx-postgres",
		"POSTGRES_USER: agentdx",
		"POSTGRES_PASSWORD: agentdx",
		"55432:5432",
		"restart: always",
		"pg_isready",
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("compose.yaml missing expected content: %q", check)
		}
	}
}

func TestWriteComposeFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compose-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	// Write compose file
	if err := WriteComposeFile(tmpDir); err != nil {
		t.Fatalf("WriteComposeFile failed: %v", err)
	}

	// Verify file exists
	composePath := filepath.Join(tmpDir, ".agentdx", "compose.yaml")
	data, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatalf("failed to read compose.yaml: %v", err)
	}

	// Verify content matches
	expected := GenerateComposeYAML()
	if string(data) != expected {
		t.Error("written content doesn't match generated content")
	}
}

func TestWriteComposeFile_CreatesDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compose-dir-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	// .agentdx directory doesn't exist yet
	agentdxDir := filepath.Join(tmpDir, ".agentdx")
	if _, err := os.Stat(agentdxDir); !os.IsNotExist(err) {
		t.Fatal(".agentdx directory should not exist initially")
	}

	// Write compose file should create directory
	if err := WriteComposeFile(tmpDir); err != nil {
		t.Fatalf("WriteComposeFile failed: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(agentdxDir); os.IsNotExist(err) {
		t.Error(".agentdx directory should have been created")
	}
}
