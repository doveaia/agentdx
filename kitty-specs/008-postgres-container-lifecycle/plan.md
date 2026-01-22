# Implementation Plan: PostgreSQL Container Lifecycle Management

**Branch**: `008-postgres-container-lifecycle` | **Date**: 2026-01-22 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/kitty-specs/008-postgres-container-lifecycle/spec.md`

## Summary

This feature adds automatic PostgreSQL container lifecycle management to agentdx. The `watch` and `session start` commands will auto-start a Docker container with persistent volume storage if PostgreSQL isn't running. Users can customize container name (`--pg-name`/`-n`) and port (`--pg-port`/`-p`). Tests will use isolated containers with random names/ports to enable parallel execution.

## Technical Context

**Language/Version**: Go 1.23
**Primary Dependencies**: Docker CLI (shelled out via os/exec), pgx/v5, spf13/cobra
**Storage**: PostgreSQL (Docker container with volume persistence)
**Testing**: go test with race detection
**Target Platform**: macOS, Linux (Docker Desktop or Docker Engine)
**Project Type**: Single CLI application
**Performance Goals**: Container should be ready within 30 seconds
**Constraints**: No new Go dependencies - use existing Docker CLI wrapper pattern
**Scale/Scope**: Single developer machine, one container per project

## Constitution Check

✅ No new dependencies introduced
✅ Uses existing patterns (`localsetup` package)
✅ Maintains backward compatibility (defaults unchanged)

## Project Structure

### Documentation (this feature)

```
kitty-specs/008-postgres-container-lifecycle/
├── spec.md              # Feature specification
├── plan.md              # This file
├── checklists/
│   └── requirements.md  # Requirements checklist
└── tasks/               # Task breakdown
```

### Source Code (repository root)

```
localsetup/
├── docker.go            # MODIFY: Add volume support, parameterize name/port
├── database.go          # MODIFY: Make DSN functions accept parameters
├── localsetup.go        # MODIFY: Accept ContainerOptions
├── testcontainer.go     # NEW: Test helper for random containers
└── testcontainer_test.go # NEW: Tests for test helper

cli/
├── watch.go             # MODIFY: Add --pg-name, --pg-port flags, auto-start container
├── session.go           # MODIFY: Add --pg-name, --pg-port flags to session start

config/
├── config.go            # MODIFY: Add postgres.container_name, postgres.port to config
```

**Structure Decision**: Single project structure. Changes are localized to `localsetup/` (container management), `cli/` (flags), and `config/` (persistence).

## Design

### Architecture Changes

```
┌─────────────────────────────────────────────────────────────────┐
│                       CLI Layer                                  │
├─────────────────────────────────────────────────────────────────┤
│  watch.go             │  session.go                              │
│  --pg-name, --pg-port │  --pg-name, --pg-port                   │
│         │                       │                                │
│         └───────────┬───────────┘                                │
│                     ▼                                            │
│         EnsurePostgresRunning(opts)                              │
└─────────────────────────────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                   localsetup Layer                               │
├─────────────────────────────────────────────────────────────────┤
│  ContainerOptions {Name, Port}                                   │
│                                                                  │
│  EnsurePostgresRunning(opts) → checks if running, starts if not │
│  CreateContainerWithVolume(opts) → creates with named volume     │
│                                                                  │
│  For tests:                                                      │
│  NewTestContainer(t) → random name/port, cleanup on t.Cleanup() │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow

1. **CLI Startup** (`watch` or `session start`):
   - Parse `--pg-name` and `--pg-port` flags
   - Load config, merge with flag values (flags take precedence)
   - Call `localsetup.EnsurePostgresRunning(opts)`

2. **EnsurePostgresRunning**:
   - Check if Docker is available → fail with clear message if not
   - Check if container exists by name
   - If exists but stopped → start it
   - If not exists → create with volume
   - Wait for PostgreSQL to accept connections
   - Return DSN for connection

3. **Test Container**:
   - Generate random suffix: `agentdx-test-{8 random hex chars}`
   - Find available port (let OS assign)
   - Create container, register `t.Cleanup()`
   - Return DSN and cleanup function

### New Types

```go
// localsetup/options.go
type ContainerOptions struct {
    Name string // Container name (default: "agentdx-postgres")
    Port int    // Host port (default: 55432)
}

func DefaultContainerOptions() ContainerOptions {
    return ContainerOptions{
        Name: "agentdx-postgres",
        Port: 55432,
    }
}
```

```go
// localsetup/testcontainer.go
type TestContainer struct {
    Name    string
    Port    int
    DSN     string
    cleanup func()
}

func NewTestContainer(t testing.TB) *TestContainer
func (tc *TestContainer) Close()
```

### Config Changes

```yaml
# .agentdx/config.yaml
index:
  store:
    postgres:
      dsn: "postgres://..."  # existing
      container_name: "agentdx-postgres"  # NEW - optional
      port: 55432  # NEW - optional
```

### Volume Strategy

Docker volume naming: `{container_name}-data`

Example:
- Container: `agentdx-postgres` → Volume: `agentdx-postgres-data`
- Container: `my-custom-pg` → Volume: `my-custom-pg-data`

```bash
docker run -d \
  --name agentdx-postgres \
  -p 55432:5432 \
  -v agentdx-postgres-data:/var/lib/postgresql/data \
  -e POSTGRES_USER=agentdx \
  -e POSTGRES_PASSWORD=agentdx \
  doveaia/timescaledb:latest-pg17-ts
```

### CLI Flag Design

```
agentdx watch [--pg-name NAME | -n NAME] [--pg-port PORT | -p PORT]
agentdx session start [--pg-name NAME | -n NAME] [--pg-port PORT | -p PORT]
```

Flag precedence:
1. CLI flag (highest)
2. Config file
3. Default value (lowest)

### Error Messages

| Condition | Error Message |
|-----------|---------------|
| Docker not available | `Docker is not running. Please start Docker and try again.` |
| Port in use | `Port 55432 is already in use. Try a different port with --pg-port.` |
| Container creation fails | `Failed to start PostgreSQL container: {docker error}` |
| PostgreSQL not ready | `PostgreSQL not ready after 30s. Check container logs: docker logs {name}` |

## Testing Strategy

### Unit Tests
- `localsetup/options_test.go`: Test option merging and defaults
- `localsetup/docker_test.go`: Test volume creation args

### Integration Tests
- `localsetup/testcontainer_test.go`: Test container lifecycle
- `cli/watch_integration_test.go`: Test watch with auto-start
- Parallel test execution validation

### Test Container Lifecycle

```go
func TestSomethingWithPostgres(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    tc := localsetup.NewTestContainer(t)
    // tc.DSN is ready to use
    // Container auto-cleaned on test completion
}
```

## Migration Path

**Backward Compatibility**: Existing users with `agentdx-postgres` container will continue to work - the defaults are unchanged. The container will be detected and reused.

**New Behavior**: If no container exists, it will be auto-created on first `agentdx watch` or `agentdx session start`.

## Files Changed

| File | Change Type | Description |
|------|-------------|-------------|
| `localsetup/options.go` | NEW | ContainerOptions type and defaults |
| `localsetup/docker.go` | MODIFY | Add volume support, parameterize CreateContainer |
| `localsetup/ensure.go` | NEW | EnsurePostgresRunning function |
| `localsetup/testcontainer.go` | NEW | Test container helper |
| `localsetup/database.go` | MODIFY | Make DSN functions accept host/port params |
| `cli/watch.go` | MODIFY | Add flags, call EnsurePostgresRunning |
| `cli/session.go` | MODIFY | Add flags to session start |
| `config/config.go` | MODIFY | Add container_name, port to PostgresConfig |
| Tests | NEW/MODIFY | Test new functionality |
