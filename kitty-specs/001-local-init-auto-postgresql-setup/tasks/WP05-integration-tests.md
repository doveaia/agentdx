---
work_package_id: "WP05"
subtasks:
  - "T012"
  - "T013"
  - "T014"
  - "T015"
  - "T016"
title: "Integration Tests"
phase: "Phase 4 - Validation"
lane: "done"
assignee: ""
agent: "claude"
shell_pid: "27793"
review_status: ""
reviewed_by: ""
history:
  - timestamp: "2026-01-13T15:30:00Z"
    lane: "planned"
    agent: "system"
    shell_pid: ""
    action: "Prompt generated via /spec-kitty.tasks"
  - timestamp: "2026-01-13T22:40:00Z"
    lane: "doing"
    agent: "claude"
    shell_pid: "27793"
    action: "Started implementation"
---

# Work Package Prompt: WP05 – Integration Tests

## Review Feedback

*[This section is empty initially. Reviewers will populate it if the work is returned from review.]*

---

## Objectives & Success Criteria

**Goal**: Validate the complete feature with integration tests against real Docker.

**Success Criteria**:
- All tests pass when Docker is available
- Tests skip gracefully when Docker unavailable
- Slug generation tested with edge cases
- Full local setup flow tested end-to-end
- Compose.yaml generation tested
- `make test` passes with race detection

---

## Context & Constraints

**Reference Documents**:
- Constitution: `.kittify/memory/constitution.md` (Testing Standards section)
- Plan: `kitty-specs/001-local-init-auto-postgresql-setup/plan.md`

**Key Constraints**:
- Use `testing.Short()` to skip Docker tests in quick runs
- Check `IsDockerAvailable()` at test start, skip with `t.Skip()` if not available
- Use table-driven tests where appropriate
- Clean up test databases after tests

**Dependencies**:
- Depends on WP04 (all code must be implemented first)

---

## Subtasks & Detailed Guidance

### Subtask T012 – Create localsetup/slug_test.go

- **Purpose**: Test slug generation with various edge cases.
- **Files**: `localsetup/slug_test.go` (new file)
- **Steps**:

1. Create table-driven tests:
```go
package localsetup

import "testing"

func TestToSlug(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"simple", "myproject", "myproject"},
        {"with hyphen", "my-project", "my_project"},
        {"with spaces", "My Project", "my_project"},
        {"mixed case", "MyProject", "myproject"},
        {"with numbers", "project123", "project123"},
        {"numbers and hyphens", "123-numbers-first", "123_numbers_first"},
        {"special chars", "test@project!", "testproject"},
        {"unicode", "café-app", "caf_app"},
        {"multiple hyphens", "my--project", "my_project"},
        {"multiple spaces", "my   project", "my_project"},
        {"leading hyphen", "-project", "project"},
        {"trailing hyphen", "project-", "project"},
        {"underscores preserved", "my_project", "my_project"},
        {"mixed separators", "my-project_name", "my_project_name"},
        {"empty string", "", ""},
        {"only special chars", "!@#$%", ""},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := ToSlug(tt.input)
            if got != tt.expected {
                t.Errorf("ToSlug(%q) = %q, want %q", tt.input, got, tt.expected)
            }
        })
    }
}
```

- **Parallel?**: Yes, can be done in parallel with other test files
- **Notes**: Cover all edge cases from data-model.md

### Subtask T013 – Create localsetup/docker_test.go

- **Purpose**: Test Docker detection and container operations.
- **Files**: `localsetup/docker_test.go` (new file)
- **Steps**:

1. Create tests with Docker skip logic:
```go
package localsetup

import (
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

    // This test uses a lightweight test container
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

    // Test CreateContainer
    cfg := ContainerConfig{
        Name:          testContainer,
        Image:         testImage,
        HostPort:      "0", // Random port
        ContainerPort: "80",
        RestartPolicy: "no",
        EnvVars:       map[string]string{"TEST": "value"},
    }
    if err := CreateContainer(cfg); err != nil {
        t.Fatalf("CreateContainer failed: %v", err)
    }

    // Test ContainerExists (should be true now)
    exists, err = ContainerExists(testContainer)
    if err != nil {
        t.Fatalf("ContainerExists failed: %v", err)
    }
    if !exists {
        t.Fatal("expected container to exist after creation")
    }
}
```

- **Parallel?**: Yes, but Docker tests should not run in parallel with each other
- **Notes**: Use lightweight alpine image for quick tests, cleanup after

### Subtask T014 – Create localsetup/localsetup_test.go

- **Purpose**: Test the full local setup orchestration flow.
- **Files**: `localsetup/localsetup_test.go` (new file)
- **Steps**:

1. Create integration test:
```go
package localsetup

import (
    "os"
    "path/filepath"
    "testing"
)

func TestRunLocalSetup(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }
    if !IsDockerAvailable() {
        t.Skip("Docker not available")
    }

    // Create a temporary directory for the test project
    tmpDir, err := os.MkdirTemp("", "agentdx-test-*")
    if err != nil {
        t.Fatalf("failed to create temp dir: %v", err)
    }
    t.Cleanup(func() { os.RemoveAll(tmpDir) })

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
    // Create temp dir with only special chars
    tmpDir, err := os.MkdirTemp("", "!@#$%")
    if err != nil {
        // Might fail on some systems, skip
        t.Skip("cannot create directory with special chars")
    }
    t.Cleanup(func() { os.RemoveAll(tmpDir) })

    _, err = RunLocalSetup(tmpDir)
    if err == nil {
        t.Error("expected error for empty slug, got nil")
    }
}
```

- **Parallel?**: Yes
- **Notes**: Use unique temp directories to avoid conflicts

### Subtask T015 – Test fallback behavior without Docker

- **Purpose**: Verify the system handles missing Docker gracefully.
- **Files**: `localsetup/localsetup_test.go` (add to existing file)
- **Steps**:

1. Add test that simulates no-Docker scenario:
```go
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
```

- **Parallel?**: Yes
- **Notes**: Focus on verifying compose.yaml is always generated

### Subtask T016 – Create localsetup/compose_test.go

- **Purpose**: Test compose.yaml generation.
- **Files**: `localsetup/compose_test.go` (new file)
- **Steps**:

1. Create tests:
```go
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
```

- **Parallel?**: Yes
- **Notes**: Test both content generation and file writing

---

## Test Strategy

**Run all tests**:
```bash
make test
```

**Run only localsetup tests**:
```bash
go test -v -race ./localsetup/...
```

**Skip Docker tests (quick mode)**:
```bash
go test -short ./localsetup/...
```

**Expected behavior**:
- Slug tests always run
- Docker tests skip when Docker unavailable
- Compose tests always run (no Docker dependency)
- Integration tests skip without Docker

---

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Tests flaky in CI | Use unique container/database names, proper cleanup |
| Container pollution | t.Cleanup() to remove test containers |
| Port conflicts | Use random ports for test containers |

---

## Definition of Done Checklist

- [ ] T012: slug_test.go with comprehensive edge cases
- [ ] T013: docker_test.go with skip logic
- [ ] T014: localsetup_test.go integration test
- [ ] T015: Fallback behavior test added
- [ ] T016: compose_test.go with content and file tests
- [ ] All tests pass with `make test`
- [ ] Tests skip gracefully when Docker unavailable
- [ ] `make lint` passes
- [ ] No race conditions detected

---

## Review Guidance

- Verify all Docker tests have proper skip logic
- Check cleanup functions remove test artifacts
- Verify table-driven tests cover edge cases from spec
- Run tests both with and without Docker to verify skip logic

---

## Activity Log

- 2026-01-13T15:30:00Z – system – lane=planned – Prompt created.
- 2026-01-13T22:01:52Z – claude – shell_pid=27793 – lane=done – Review passed - all acceptance criteria met
