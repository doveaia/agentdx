---
work_package_id: "WP07"
title: "Polish & Documentation"
lane: "done"
subtasks:
  - "T027"
  - "T028"
  - "T029"
  - "T030"
phase: "Phase 4 - Polish"
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

# Work Package Prompt: WP07 – Polish & Documentation

## Objective

Final cleanup, documentation updates, and validation that everything works together correctly.

## Context

After all functional work packages are complete, this package ensures the feature is polished, documented, and fully tested.

## Subtasks

### T027: Update README with New Flags

**File**: `README.md` (MODIFY)

Add documentation for the new container management features:

```markdown
## PostgreSQL Container Management

agentdx automatically manages a PostgreSQL container for storing your code index.

### Quick Start

```bash
# Just run watch - PostgreSQL starts automatically
agentdx watch
```

### Custom Container Settings

```bash
# Use a custom container name
agentdx watch --pg-name my-project-db

# Use a custom port
agentdx watch --pg-port 5433

# Or both (short flags)
agentdx watch -n my-db -p 5433
```

### Session Daemon

```bash
# Start background daemon with custom settings
agentdx session start --pg-name my-project-db --pg-port 5433
```

### Configuration File

You can also set defaults in `.agentdx/config.yaml`:

```yaml
index:
  store:
    postgres:
      container_name: "my-project-postgres"
      port: 55433
```

CLI flags always take precedence over config file settings.

### Data Persistence

PostgreSQL data is stored in a Docker volume named `{container_name}-data`, so your index survives container restarts and system reboots.
```

### T028: Add Troubleshooting Section

**File**: `README.md` or `docs/troubleshooting.md` (NEW/MODIFY)

```markdown
## Troubleshooting

### Docker Not Running

```
Error: Docker is not running. Please start Docker and try again.
```

**Solution**: Start Docker Desktop or the Docker daemon.

### Port Already in Use

```
Error: Port 55432 is already in use. Try a different port with --pg-port.
```

**Solution**:
1. Use a different port: `agentdx watch --pg-port 55433`
2. Or find what's using the port: `lsof -i :55432`

### Container Won't Start

```
Error: failed to create container: ...
```

**Solutions**:
1. Check Docker logs: `docker logs agentdx-postgres`
2. Remove stale container: `docker rm agentdx-postgres`
3. Try again: `agentdx watch`

### PostgreSQL Not Ready

```
Error: PostgreSQL not ready after 30s. Check container logs: docker logs agentdx-postgres
```

**Solution**: Check container logs for startup errors.

### Test Failures with Container Conflicts

If tests fail with container name conflicts:
1. Ensure you're using the latest version with TestContainer support
2. Run `docker ps -a | grep agentdx` to find orphaned containers
3. Clean up: `docker rm $(docker ps -aq -f name=agentdx-test-)`
```

### T029: Run Full Test Suite with Parallel Execution

Run comprehensive tests to verify everything works:

```bash
# Run all tests with race detection and parallelism
go test ./... -race -parallel 4 -v

# Run integration tests specifically
go test ./... -v -run Integration

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

**Expected Results**:
- All tests pass
- No race conditions detected
- No orphaned Docker containers
- Coverage meets project standards

### T030: Verify Backward Compatibility

**Scenarios to test**:

1. **Existing container**: User has existing `agentdx-postgres` container
   ```bash
   # Verify watch reuses it
   agentdx watch  # Should connect to existing container
   ```

2. **Existing config without new fields**: Config file doesn't have `container_name` or `port`
   ```bash
   # Verify defaults are used
   agentdx watch  # Should use agentdx-postgres:55432
   ```

3. **Upgrade scenario**: User upgrades agentdx with existing data
   ```bash
   # Verify data is preserved
   agentdx search "test"  # Should return previous results
   ```

4. **Volume persistence**: Container restart preserves data
   ```bash
   docker restart agentdx-postgres
   agentdx search "test"  # Should return previous results
   ```

## Acceptance Criteria

- [ ] README documents new flags and configuration options
- [ ] Troubleshooting section covers common issues
- [ ] Full test suite passes with `-parallel 4`
- [ ] No race conditions detected
- [ ] Backward compatibility verified for all scenarios
- [ ] No orphaned containers after test runs

## Files Changed

| File | Change |
|------|--------|
| `README.md` | MODIFY - add container management docs |
| `docs/troubleshooting.md` | NEW (optional) |

## Testing Commands

```bash
# Full test suite
go test ./... -race -parallel 4 -v

# Check for orphaned containers
docker ps -a | grep agentdx

# Clean up any orphans
docker rm $(docker ps -aq -f name=agentdx-test-) 2>/dev/null || true

# Verify backward compatibility manually
docker ps -f name=agentdx-postgres
agentdx watch  # Should work with existing or new container
```

## Definition of Done

- All tests pass
- Documentation is complete and accurate
- Backward compatibility is verified
- No known issues or regressions
- Feature ready for release

## Activity Log

- 2026-01-22T08:56:09Z – unknown – lane=doing – Moved to doing
- 2026-01-22T09:00:03Z – unknown – lane=done – Moved to done
