---
work_package_id: "WP05"
title: "Config File Support"
lane: "done"
subtasks:
  - "T017"
  - "T018"
  - "T019"
  - "T020"
phase: "Phase 3 - Configuration"
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

# Work Package Prompt: WP05 – Config File Support

## Objective

Add `container_name` and `port` fields to the config.yaml so users can set default container settings per project. CLI flags should override config values.

## Context

Currently, container settings can only be specified via CLI flags. Adding config file support allows users to set project-specific defaults that persist across sessions.

## Subtasks

### T017: Add Config Fields

**File**: `config/config.go` (MODIFY)

Add new fields to `PostgresConfig`:

```go
type PostgresConfig struct {
    DSN           string `yaml:"dsn"`
    ContainerName string `yaml:"container_name,omitempty"` // optional, default: agentdx-postgres
    Port          int    `yaml:"port,omitempty"`           // optional, default: 55432
}
```

### T018: Update DefaultConfig

**File**: `config/config.go` (MODIFY)

The defaults are already in `localsetup.DefaultContainerOptions()`, so we don't need to duplicate them in config. Just document the fields.

**File**: `cli/init.go` (MODIFY if needed)

Update the config template to show the new fields (commented out):

```yaml
index:
  store:
    postgres:
      dsn: "postgres://agentdx:agentdx@localhost:55432/agentdx_project?sslmode=disable"
      # container_name: "agentdx-postgres"  # Optional: custom container name
      # port: 55432  # Optional: custom host port
```

### T019: Implement Option Merging

**File**: `cli/watch.go` (MODIFY)

Add function to merge options from flags and config:

```go
func buildContainerOptions(cfg *config.Config, flagName string, flagPort int) localsetup.ContainerOptions {
    // Start with defaults
    opts := localsetup.DefaultContainerOptions()

    // Apply config values (if set)
    if cfg.Index.Store.Postgres.ContainerName != "" {
        opts.Name = cfg.Index.Store.Postgres.ContainerName
    }
    if cfg.Index.Store.Postgres.Port != 0 {
        opts.Port = cfg.Index.Store.Postgres.Port
    }

    // Apply flag values (highest priority)
    if flagName != "" {
        opts.Name = flagName
    }
    if flagPort != 0 {
        opts.Port = flagPort
    }

    return opts
}
```

Update `runWatch` to use this function:

```go
// Build container options: flags > config > defaults
opts := buildContainerOptions(cfg, pgName, pgPort)

// Ensure PostgreSQL is running
dsn, err := localsetup.EnsurePostgresRunning(ctx, projectRoot, opts)
```

**File**: `cli/session.go` (MODIFY)

Apply the same pattern to session start.

### T020: Write Test for Config Loading

**File**: `config/config_test.go` (MODIFY)

Add test cases:

```go
func TestConfigLoadWithContainerSettings(t *testing.T) {
    // Create temp dir with config
    // Config should have container_name and port
    // Verify they load correctly
}

func TestConfigLoadWithPartialContainerSettings(t *testing.T) {
    // Only container_name set, port should be zero
    // Only port set, container_name should be empty
}
```

**File**: `cli/watch_test.go` (NEW or MODIFY)

Test option merging:

```go
func TestBuildContainerOptions(t *testing.T) {
    tests := []struct {
        name      string
        cfgName   string
        cfgPort   int
        flagName  string
        flagPort  int
        wantName  string
        wantPort  int
    }{
        {
            name:     "all defaults",
            wantName: "agentdx-postgres",
            wantPort: 55432,
        },
        {
            name:     "config only",
            cfgName:  "my-config-db",
            cfgPort:  5433,
            wantName: "my-config-db",
            wantPort: 5433,
        },
        {
            name:     "flags override config",
            cfgName:  "config-db",
            cfgPort:  5433,
            flagName: "flag-db",
            flagPort: 5434,
            wantName: "flag-db",
            wantPort: 5434,
        },
        {
            name:     "partial override",
            cfgName:  "config-db",
            cfgPort:  5433,
            flagPort: 5434,
            wantName: "config-db",  // from config
            wantPort: 5434,         // from flag
        },
    }
    // ...
}
```

## Acceptance Criteria

- [ ] Config file can specify `container_name` and `port`
- [ ] Empty/missing config values use defaults
- [ ] CLI flags override config values
- [ ] Unit tests for option merging pass
- [ ] Config loading tests pass

## Files Changed

| File | Change |
|------|--------|
| `config/config.go` | MODIFY - add ContainerName, Port fields |
| `config/config_test.go` | MODIFY - add tests |
| `cli/watch.go` | MODIFY - add buildContainerOptions |
| `cli/watch_test.go` | NEW/MODIFY - test option merging |
| `cli/session.go` | MODIFY - use buildContainerOptions |

## Example Config

```yaml
version: 1
mode: local
index:
  store:
    postgres:
      dsn: "postgres://agentdx:agentdx@localhost:55432/agentdx_myproject?sslmode=disable"
      container_name: "my-project-postgres"
      port: 55433
  chunking:
    size: 512
    overlap: 50
  # ... rest of config
```

## Testing Commands

```bash
# Create test config
cat > .agentdx/config.yaml << EOF
version: 1
mode: local
index:
  store:
    postgres:
      dsn: "postgres://agentdx:agentdx@localhost:5433/agentdx_test?sslmode=disable"
      container_name: "test-config-pg"
      port: 5433
EOF

# Verify config is loaded
go run . watch  # Should use test-config-pg on port 5433

# Verify flags override
go run . watch --pg-port 5434  # Should use test-config-pg on port 5434

# Run unit tests
go test ./config/... -v
go test ./cli/... -v -run TestBuildContainerOptions
```

## Activity Log

- 2026-01-22T08:50:32Z – unknown – lane=doing – Moved to doing
- 2026-01-22T08:52:54Z – unknown – lane=for_review – Moved to for_review
- 2026-01-22T09:00:02Z – unknown – lane=done – Moved to done
