---
work_package_id: "WP02"
subtasks:
  - "T005"
title: "Docker Container Management"
phase: "Phase 2 - Core Implementation"
lane: "done"
assignee: ""
agent: "claude"
shell_pid: "56255"
review_status: ""
reviewed_by: ""
history:
  - timestamp: "2026-01-13T15:30:00Z"
    lane: "planned"
    agent: "system"
    shell_pid: ""
    action: "Prompt generated via /spec-kitty.tasks"
---

# Work Package Prompt: WP02 – Docker Container Management

## Review Feedback

*[This section is empty initially. Reviewers will populate it if the work is returned from review.]*

---

## Objectives & Success Criteria

**Goal**: Implement Docker CLI wrapper for container lifecycle management.

**Success Criteria**:
- `IsDockerAvailable()` correctly detects Docker CLI presence
- `ContainerExists()` accurately checks if container exists
- `ContainerRunning()` correctly checks container running state
- `CreateContainer()` creates container with all specified settings
- `StartContainer()` starts stopped containers
- All functions handle errors gracefully with clear messages
- Commands use timeouts to prevent hanging

---

## Context & Constraints

**Reference Documents**:
- Constitution: `.kittify/memory/constitution.md`
- Plan: `kitty-specs/001-local-init-auto-postgresql-setup/plan.md`
- Spec: `kitty-specs/001-local-init-auto-postgresql-setup/spec.md` (FR-009 through FR-015)

**Key Constraints**:
- Use `os/exec` for Docker commands
- Use `exec.CommandContext` with 30s timeout to prevent hangs
- Container name: `agentdx-postgres`
- Image: `doveaia/timescaledb:latest-pg17-ts`
- Port mapping: 5432 → 55432
- Restart policy: always

**Dependencies**:
- Depends on WP01 (interfaces.go for DockerClient interface)

---

## Subtasks & Detailed Guidance

### Subtask T005 – Create localsetup/docker.go

- **Purpose**: Implement Docker container management operations.
- **Files**: `localsetup/docker.go` (new file)
- **Steps**:

1. Create the file with package declaration and imports:
```go
package localsetup

import (
    "context"
    "encoding/json"
    "fmt"
    "os/exec"
    "strings"
    "time"
)

const (
    dockerCommandTimeout = 30 * time.Second
    containerName        = "agentdx-postgres"
    containerImage       = "doveaia/timescaledb:latest-pg17-ts"
    hostPort             = "55432"
    containerPort        = "5432"
)
```

2. Implement `IsDockerAvailable()`:
```go
// IsDockerAvailable checks if the docker CLI is available in PATH.
func IsDockerAvailable() bool {
    _, err := exec.LookPath("docker")
    return err == nil
}
```

3. Implement `ContainerExists()`:
```go
// ContainerExists checks if a container with the given name exists.
func ContainerExists(name string) (bool, error) {
    ctx, cancel := context.WithTimeout(context.Background(), dockerCommandTimeout)
    defer cancel()

    cmd := exec.CommandContext(ctx, "docker", "inspect", name)
    err := cmd.Run()
    if err != nil {
        // docker inspect returns exit code 1 if container doesn't exist
        if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
            return false, nil
        }
        return false, fmt.Errorf("failed to check container: %w", err)
    }
    return true, nil
}
```

4. Implement `ContainerRunning()`:
```go
// ContainerRunning checks if a container is currently running.
func ContainerRunning(name string) (bool, error) {
    ctx, cancel := context.WithTimeout(context.Background(), dockerCommandTimeout)
    defer cancel()

    cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Running}}", name)
    output, err := cmd.Output()
    if err != nil {
        return false, fmt.Errorf("failed to check container state: %w", err)
    }
    return strings.TrimSpace(string(output)) == "true", nil
}
```

5. Implement `CreateContainer()`:
```go
// CreateContainer creates a new Docker container with the specified configuration.
func CreateContainer(cfg ContainerConfig) error {
    ctx, cancel := context.WithTimeout(context.Background(), dockerCommandTimeout)
    defer cancel()

    args := []string{
        "run", "-d",
        "--name", cfg.Name,
        "--restart", cfg.RestartPolicy,
        "-p", fmt.Sprintf("%s:%s", cfg.HostPort, cfg.ContainerPort),
    }

    for key, value := range cfg.EnvVars {
        args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
    }

    args = append(args, cfg.Image)

    cmd := exec.CommandContext(ctx, "docker", args...)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("failed to create container: %s: %w", string(output), err)
    }
    return nil
}
```

6. Implement `StartContainer()`:
```go
// StartContainer starts a stopped container.
func StartContainer(name string) error {
    ctx, cancel := context.WithTimeout(context.Background(), dockerCommandTimeout)
    defer cancel()

    cmd := exec.CommandContext(ctx, "docker", "start", name)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("failed to start container: %s: %w", string(output), err)
    }
    return nil
}
```

7. Add helper function to get default container config:
```go
// DefaultContainerConfig returns the default configuration for the agentdx-postgres container.
func DefaultContainerConfig() ContainerConfig {
    return ContainerConfig{
        Name:          containerName,
        Image:         containerImage,
        HostPort:      hostPort,
        ContainerPort: containerPort,
        RestartPolicy: "always",
        EnvVars: map[string]string{
            "POSTGRES_USER":     "agentdx",
            "POSTGRES_PASSWORD": "agentdx",
        },
    }
}
```

- **Parallel?**: Can be developed in parallel with WP03
- **Notes**:
  - Error messages should be user-friendly
  - Consider wrapping context timeout errors with clearer messages

---

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Docker command hangs | Use `context.WithTimeout` with 30s limit |
| Permission denied | Clear error message about Docker access |
| Context deadline exceeded | Wrap with user-friendly timeout message |

---

## Definition of Done Checklist

- [ ] T005: localsetup/docker.go created with all functions
- [ ] IsDockerAvailable() works correctly
- [ ] ContainerExists() handles both existing and non-existing containers
- [ ] ContainerRunning() correctly reports running/stopped state
- [ ] CreateContainer() creates container with all settings
- [ ] StartContainer() starts stopped containers
- [ ] All functions use timeouts
- [ ] `make build` passes
- [ ] `make lint` passes

---

## Review Guidance

- Verify all Docker commands use `exec.CommandContext` with timeout
- Check error handling covers common failure cases
- Verify container config matches spec requirements (image, ports, env vars, restart)
- Test with Docker unavailable, container exists/not exists, running/stopped states

---

## Activity Log

- 2026-01-13T15:30:00Z – system – lane=planned – Prompt created.
- 2026-01-13T16:08:41Z – claude – shell_pid=55216 – lane=doing – Started implementation
- 2026-01-13T16:10:08Z – claude – shell_pid=56255 – lane=for_review – Ready for review
- 2026-01-13T22:01:40Z – claude – lane=doing – Started review via workflow command
- 2026-01-13T22:01:51Z – claude – shell_pid=56255 – lane=done – Review passed - all acceptance criteria met
