---
work_package_id: "WP02"
title: "EnsurePostgresRunning Function"
lane: "done"
subtasks:
  - "T005"
  - "T006"
  - "T007"
  - "T008"
phase: "Phase 1 - Foundation"
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

# Work Package Prompt: WP02 – EnsurePostgresRunning Function

## Objective

Implement the core function that ensures a PostgreSQL container is running and ready to accept connections. This function will be called by both `watch` and `session start` commands.

## Context

The function should:
1. Check if Docker is available
2. Check if container exists (by name)
3. Start container if stopped
4. Create container with volume if doesn't exist
5. Wait for PostgreSQL to be ready
6. Return the DSN for connection

## Subtasks

### T005: Create EnsurePostgresRunning Function

**File**: `localsetup/ensure.go` (NEW)

```go
package localsetup

import (
    "context"
    "fmt"
    "time"
)

// EnsurePostgresRunning ensures a PostgreSQL container is running and ready.
// Returns the DSN for connecting to the project database.
func EnsurePostgresRunning(ctx context.Context, projectRoot string, opts ContainerOptions) (string, error) {
    // Apply defaults
    defaults := DefaultContainerOptions()
    opts = defaults.Merge(opts)

    // Check Docker availability
    if !IsDockerAvailable() {
        return "", fmt.Errorf("Docker is not running. Please start Docker and try again")
    }

    // Check if container exists
    exists, err := ContainerExists(opts.Name)
    if err != nil {
        return "", fmt.Errorf("failed to check container: %w", err)
    }

    if exists {
        // Check if running
        running, err := ContainerRunning(opts.Name)
        if err != nil {
            return "", fmt.Errorf("failed to check container state: %w", err)
        }

        if !running {
            // Start stopped container
            if err := StartContainer(opts.Name); err != nil {
                return "", fmt.Errorf("failed to start container: %w", err)
            }
        }
    } else {
        // Create new container with volume
        cfg := ContainerConfig{
            Name:          opts.Name,
            Image:         containerImage,
            HostPort:      fmt.Sprintf("%d", opts.Port),
            ContainerPort: containerPort,
            RestartPolicy: "always",
            VolumeName:    opts.VolumeName(),
            EnvVars: map[string]string{
                "POSTGRES_USER":     defaultPostgresUser,
                "POSTGRES_PASSWORD": defaultPostgresPassword,
            },
        }

        if err := CreateContainer(cfg); err != nil {
            // Check if port is in use
            if isPortInUse(opts.Port) {
                return "", fmt.Errorf("Port %d is already in use. Try a different port with --pg-port", opts.Port)
            }
            return "", fmt.Errorf("failed to create container: %w", err)
        }
    }

    // Wait for PostgreSQL to be ready
    dsn := fmt.Sprintf("postgres://%s:%s@localhost:%d/postgres?sslmode=disable",
        defaultPostgresUser, defaultPostgresPassword, opts.Port)

    if err := WaitForPostgres(dsn, 30*time.Second); err != nil {
        return "", fmt.Errorf("PostgreSQL not ready after 30s. Check container logs: docker logs %s", opts.Name)
    }

    // Return project-specific DSN
    projectName := filepath.Base(projectRoot)
    dbName := "agentdx_" + ToSlug(projectName)

    // Create database if needed
    if err := CreateDatabase(dsn, dbName); err != nil {
        return "", fmt.Errorf("failed to create database: %w", err)
    }

    return fmt.Sprintf("postgres://%s:%s@localhost:%d/%s?sslmode=disable",
        defaultPostgresUser, defaultPostgresPassword, opts.Port, dbName), nil
}

// isPortInUse checks if a port is already in use.
func isPortInUse(port int) bool {
    // Implementation using net.Listen
}
```

### T006: Update database.go with Parameters

**File**: `localsetup/database.go` (MODIFY)

Add functions that accept host/port parameters:

```go
// PostgresDSNWithPort returns a DSN for connecting to postgres with a custom port.
func PostgresDSNWithPort(port int) string {
    return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
        defaultPostgresUser, defaultPostgresPassword,
        defaultPostgresHost, port,
        defaultPostgresDB)
}

// ProjectDSNWithPort returns a DSN for the project database with a custom port.
func ProjectDSNWithPort(dbName string, port int) string {
    return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
        defaultPostgresUser, defaultPostgresPassword,
        defaultPostgresHost, port,
        dbName)
}
```

### T007: Add Clear Error Messages

Ensure all error paths have user-friendly messages:

| Condition | Error Message |
|-----------|---------------|
| Docker not running | `Docker is not running. Please start Docker and try again` |
| Port in use | `Port {port} is already in use. Try a different port with --pg-port` |
| Container creation fails | `failed to create container: {docker error}` |
| PostgreSQL not ready | `PostgreSQL not ready after 30s. Check container logs: docker logs {name}` |

### T008: Write Integration Test

**File**: `localsetup/ensure_test.go` (NEW)

Test cases:
- EnsurePostgresRunning creates container when none exists
- EnsurePostgresRunning reuses running container
- EnsurePostgresRunning starts stopped container
- EnsurePostgresRunning returns error when Docker not available
- EnsurePostgresRunning returns correct DSN

## Acceptance Criteria

- [ ] `EnsurePostgresRunning` creates container if none exists
- [ ] `EnsurePostgresRunning` reuses existing running container
- [ ] `EnsurePostgresRunning` starts stopped container
- [ ] Container is created with persistent volume
- [ ] Clear error message when Docker not available
- [ ] Clear error message when port in use
- [ ] Integration tests pass

## Files Changed

| File | Change |
|------|--------|
| `localsetup/ensure.go` | NEW |
| `localsetup/ensure_test.go` | NEW |
| `localsetup/database.go` | MODIFY - add port-parameterized functions |

## Testing Commands

```bash
# Run integration tests (requires Docker)
go test ./localsetup/... -v -run TestEnsurePostgres

# Test without Docker (should fail gracefully)
# Stop Docker, then run tests
```

## Activity Log

- 2026-01-22T08:41:35Z – unknown – lane=doing – Moved to doing
- 2026-01-22T08:44:39Z – unknown – lane=for_review – Moved to for_review
- 2026-01-22T09:00:02Z – unknown – lane=done – Moved to done
