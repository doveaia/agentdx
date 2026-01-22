---
work_package_id: "WP01"
title: "Container Options & Volume Support"
lane: "done"
subtasks:
  - "T001"
  - "T002"
  - "T003"
  - "T004"
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

# Work Package Prompt: WP01 – Container Options & Volume Support

## Objective

Create the foundational types and modify Docker container creation to support configurable container names, ports, and persistent volumes.

## Context

Currently, container name (`agentdx-postgres`) and port (`55432`) are hardcoded constants. This work package introduces a `ContainerOptions` type that allows these values to be customized, and adds Docker volume support for data persistence.

## Subtasks

### T001: Create ContainerOptions Type

**File**: `localsetup/options.go` (NEW)

Create a new file with:

```go
package localsetup

// ContainerOptions holds configuration for the PostgreSQL container.
type ContainerOptions struct {
    Name string // Container name (default: "agentdx-postgres")
    Port int    // Host port (default: 55432)
}

// DefaultContainerOptions returns the default container configuration.
func DefaultContainerOptions() ContainerOptions {
    return ContainerOptions{
        Name: "agentdx-postgres",
        Port: 55432,
    }
}

// VolumeName returns the Docker volume name for this container.
func (o ContainerOptions) VolumeName() string {
    return o.Name + "-data"
}

// Merge returns a new ContainerOptions with non-zero values from other taking precedence.
func (o ContainerOptions) Merge(other ContainerOptions) ContainerOptions {
    result := o
    if other.Name != "" {
        result.Name = other.Name
    }
    if other.Port != 0 {
        result.Port = other.Port
    }
    return result
}
```

### T002: Add Volume Support to CreateContainer

**File**: `localsetup/docker.go` (MODIFY)

Update `CreateContainer` to accept a volume name and add the `-v` flag:

```go
// In ContainerConfig struct, add:
VolumeName string // Docker volume name for data persistence

// In CreateContainer function, after other args:
if cfg.VolumeName != "" {
    args = append(args, "-v", fmt.Sprintf("%s:/var/lib/postgresql/data", cfg.VolumeName))
}
```

### T003: Update ContainerConfig Interface

**File**: `localsetup/interfaces.go` (MODIFY)

Add `VolumeName` field to `ContainerConfig`:

```go
type ContainerConfig struct {
    // ... existing fields ...
    VolumeName string // Docker volume name for data persistence
}
```

### T004: Write Unit Tests

**File**: `localsetup/options_test.go` (NEW)

Test cases:
- DefaultContainerOptions returns correct defaults
- VolumeName() generates correct volume name
- Merge() correctly prioritizes non-zero values
- Merge() preserves original values when other is zero

## Acceptance Criteria

- [ ] `ContainerOptions` type exists with `Name` and `Port` fields
- [ ] `DefaultContainerOptions()` returns name="agentdx-postgres", port=55432
- [ ] `VolumeName()` returns "{name}-data"
- [ ] `Merge()` correctly combines options
- [ ] `CreateContainer` includes `-v` flag when volume name provided
- [ ] All unit tests pass

## Files Changed

| File | Change |
|------|--------|
| `localsetup/options.go` | NEW |
| `localsetup/options_test.go` | NEW |
| `localsetup/docker.go` | MODIFY - add volume args |
| `localsetup/interfaces.go` | MODIFY - add VolumeName field |

## Testing Commands

```bash
# Run unit tests
go test ./localsetup/... -v -run TestContainerOptions

# Verify no regressions
go test ./localsetup/... -v
```

## Activity Log

- 2026-01-22T08:35:33Z – unknown – lane=doing – Moved to doing
- 2026-01-22T08:41:01Z – unknown – lane=for_review – Moved to for_review
- 2026-01-22T09:00:02Z – unknown – lane=done – Moved to done
