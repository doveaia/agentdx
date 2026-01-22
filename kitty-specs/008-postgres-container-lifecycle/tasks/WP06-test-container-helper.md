---
work_package_id: "WP06"
title: "Test Container Helper"
lane: "done"
subtasks:
  - "T021"
  - "T022"
  - "T023"
  - "T024"
  - "T025"
  - "T026"
phase: "Phase 3 - Testing Infrastructure"
assignee: ""
agent: ""
shell_pid: ""
review_status: ""
reviewed_by: ""
history:
  - timestamp: "2026-01-22T09:00:00Z"
    lane: "planned"
    agent: "system"
    action: "Prompt generated via /spec-kitty.tasks"
---

# Work Package Prompt: WP06 – Test Container Helper

## Objective

Create a test helper that automatically provisions isolated PostgreSQL containers with random names and ports, enabling parallel test execution without conflicts.

## Context

Current tests use a hardcoded container name (`agentdx-postgres`), which causes conflicts when running tests in parallel. Each test package should get its own isolated container that is automatically cleaned up.

## Subtasks

### T021: Create TestContainer Helper

**File**: `localsetup/testcontainer.go` (NEW)

```go
package localsetup

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "net"
    "testing"
    "time"
)

// TestContainer represents an ephemeral PostgreSQL container for testing.
type TestContainer struct {
    Name    string
    Port    int
    DSN     string
    t       testing.TB
    cleanup func()
}

// NewTestContainer creates a new PostgreSQL container with a random name and port.
// The container is automatically cleaned up when the test completes.
func NewTestContainer(t testing.TB) *TestContainer {
    t.Helper()

    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }

    if !IsDockerAvailable() {
        t.Skip("Docker not available")
    }

    // Generate random name and find available port
    name := generateRandomName()
    port := findAvailablePort()

    tc := &TestContainer{
        Name: name,
        Port: port,
        t:    t,
    }

    // Create the container
    ctx := context.Background()
    cfg := ContainerConfig{
        Name:          name,
        Image:         containerImage,
        HostPort:      fmt.Sprintf("%d", port),
        ContainerPort: containerPort,
        RestartPolicy: "no", // Don't restart test containers
        VolumeName:    "",   // No volume for test containers
        EnvVars: map[string]string{
            "POSTGRES_USER":     defaultPostgresUser,
            "POSTGRES_PASSWORD": defaultPostgresPassword,
        },
    }

    if err := CreateContainer(cfg); err != nil {
        t.Fatalf("failed to create test container: %v", err)
    }

    // Register cleanup
    tc.cleanup = func() {
        _ = RemoveContainer(name)
    }
    t.Cleanup(tc.cleanup)

    // Wait for PostgreSQL to be ready
    dsn := fmt.Sprintf("postgres://%s:%s@localhost:%d/postgres?sslmode=disable",
        defaultPostgresUser, defaultPostgresPassword, port)

    if err := WaitForPostgres(dsn, 30*time.Second); err != nil {
        tc.cleanup()
        t.Fatalf("test PostgreSQL not ready: %v", err)
    }

    tc.DSN = dsn
    return tc
}

// Close explicitly removes the container. Usually called automatically via t.Cleanup.
func (tc *TestContainer) Close() {
    if tc.cleanup != nil {
        tc.cleanup()
    }
}

// CreateDatabase creates a database in the test container and returns the DSN.
func (tc *TestContainer) CreateDatabase(dbName string) string {
    if err := CreateDatabase(tc.DSN, dbName); err != nil {
        tc.t.Fatalf("failed to create test database: %v", err)
    }
    return fmt.Sprintf("postgres://%s:%s@localhost:%d/%s?sslmode=disable",
        defaultPostgresUser, defaultPostgresPassword, tc.Port, dbName)
}
```

### T022: Implement Random Name Generation

**File**: `localsetup/testcontainer.go` (continuation)

```go
// generateRandomName creates a unique container name for testing.
func generateRandomName() string {
    b := make([]byte, 4) // 8 hex characters
    if _, err := rand.Read(b); err != nil {
        // Fallback to timestamp if crypto/rand fails
        return fmt.Sprintf("agentdx-test-%d", time.Now().UnixNano())
    }
    return fmt.Sprintf("agentdx-test-%s", hex.EncodeToString(b))
}
```

### T023: Implement Random Port Selection

**File**: `localsetup/testcontainer.go` (continuation)

```go
// findAvailablePort finds an available TCP port.
func findAvailablePort() int {
    listener, err := net.Listen("tcp", ":0")
    if err != nil {
        // Fallback to a random port in a high range
        return 50000 + int(time.Now().UnixNano()%10000)
    }
    defer listener.Close()
    return listener.Addr().(*net.TCPAddr).Port
}
```

### T024: Implement Cleanup via t.Cleanup

Already included in T021. The `t.Cleanup` function ensures the container is removed even if the test panics or fails.

### T025: Write Tests for Test Container Helper

**File**: `localsetup/testcontainer_test.go` (NEW)

```go
package localsetup

import (
    "testing"
)

func TestNewTestContainer(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }

    tc := NewTestContainer(t)

    // Verify container was created
    exists, err := ContainerExists(tc.Name)
    if err != nil {
        t.Fatalf("failed to check container: %v", err)
    }
    if !exists {
        t.Error("container should exist")
    }

    // Verify DSN is set
    if tc.DSN == "" {
        t.Error("DSN should be set")
    }

    // Verify port is valid
    if tc.Port < 1024 || tc.Port > 65535 {
        t.Errorf("port %d is out of valid range", tc.Port)
    }
}

func TestTestContainerParallel(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }

    // Run multiple tests in parallel to verify no conflicts
    for i := 0; i < 3; i++ {
        i := i
        t.Run(fmt.Sprintf("parallel-%d", i), func(t *testing.T) {
            t.Parallel()
            tc := NewTestContainer(t)
            t.Logf("Container %d: %s on port %d", i, tc.Name, tc.Port)
        })
    }
}

func TestTestContainerCreateDatabase(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }

    tc := NewTestContainer(t)

    dsn := tc.CreateDatabase("test_db")
    if dsn == "" {
        t.Error("DSN should not be empty")
    }

    // Verify we can connect to the new database
    // (connection test here)
}
```

### T026: Migrate Existing Tests to Use TestContainer

**Files to update**:
- `localsetup/localsetup_test.go`
- `cli/init_integration_test.go`
- Any other tests that use the hardcoded container name

**Example migration**:

Before:
```go
func TestRunLocalSetup(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }
    if !IsDockerAvailable() {
        t.Skip("Docker not available")
    }

    // Clean up any existing container before running the test
    _ = RemoveContainer(containerName)
    // ...
    t.Cleanup(func() {
        _ = RemoveContainer(containerName)
    })
}
```

After:
```go
func TestRunLocalSetup(t *testing.T) {
    tc := NewTestContainer(t)  // Handles short mode, Docker check, cleanup

    // Use tc.Name and tc.Port for test-specific container
    // ...
}
```

## Acceptance Criteria

- [ ] `NewTestContainer(t)` creates container with random name
- [ ] Container uses a dynamically assigned port
- [ ] Container is automatically cleaned up via `t.Cleanup`
- [ ] Parallel tests run without container name conflicts
- [ ] `go test ./... -parallel 4` passes
- [ ] Existing tests migrated to use TestContainer

## Files Changed

| File | Change |
|------|--------|
| `localsetup/testcontainer.go` | NEW |
| `localsetup/testcontainer_test.go` | NEW |
| `localsetup/localsetup_test.go` | MODIFY - use TestContainer |
| `cli/init_integration_test.go` | MODIFY - use TestContainer |

## Testing Commands

```bash
# Run test container tests
go test ./localsetup/... -v -run TestTestContainer

# Run parallel test to verify no conflicts
go test ./localsetup/... -v -run TestTestContainerParallel

# Run all tests in parallel
go test ./... -v -parallel 4

# Verify no orphaned containers after tests
docker ps -a | grep agentdx-test
# Should show no containers (all cleaned up)
```

## Activity Log

- 2026-01-22T08:52:54Z – unknown – lane=doing – Moved to doing
- 2026-01-22T08:56:09Z – unknown – lane=for_review – Moved to for_review
- 2026-01-22T09:00:03Z – unknown – lane=done – Moved to done
