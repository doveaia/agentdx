---
work_package_id: "WP03"
title: "CLI Flags for watch Command"
lane: "done"
subtasks:
  - "T009"
  - "T010"
  - "T011"
  - "T012"
phase: "Phase 2 - CLI Integration"
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

# Work Package Prompt: WP03 – CLI Flags for watch Command 🎯 MVP

## Objective

Add `--pg-name` (alias `-n`) and `--pg-port` (alias `-p`) flags to the `agentdx watch` command, and integrate with `EnsurePostgresRunning` to auto-start the PostgreSQL container.

## Context

Currently, `agentdx watch` assumes PostgreSQL is already running and fails with a connection error if it's not. This work package makes watch self-sufficient by auto-starting the container.

## Subtasks

### T009: Add Flags to watchCmd

**File**: `cli/watch.go` (MODIFY)

Add package-level variables for the flags:

```go
var (
    daemonMode bool
    pgName     string
    pgPort     int
)

func init() {
    watchCmd.Flags().BoolVar(&daemonMode, "daemon", false, "Run in daemon mode (for session management)")
    watchCmd.Flags().StringVarP(&pgName, "pg-name", "n", "", "PostgreSQL container name (default: agentdx-postgres)")
    watchCmd.Flags().IntVarP(&pgPort, "pg-port", "p", 0, "PostgreSQL host port (default: 55432)")
}
```

### T010: Call EnsurePostgresRunning Before Connecting

**File**: `cli/watch.go` (MODIFY)

Update `runWatch` to ensure PostgreSQL is running before connecting:

```go
func runWatch(cmd *cobra.Command, args []string) error {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    // Find project root
    projectRoot, err := config.FindProjectRoot()
    if err != nil {
        return err
    }

    // Load configuration
    cfg, err := config.Load(projectRoot)
    if err != nil {
        return fmt.Errorf("failed to load configuration: %w", err)
    }

    // Build container options from flags and config
    opts := localsetup.ContainerOptions{
        Name: pgName,  // empty string means use default/config
        Port: pgPort,  // 0 means use default/config
    }
    // TODO: Merge with config values when WP05 is complete

    // Ensure PostgreSQL is running
    dsn, err := localsetup.EnsurePostgresRunning(ctx, projectRoot, opts)
    if err != nil {
        return err
    }

    if !daemonMode {
        fmt.Printf("Starting agentdx watch in %s\n", projectRoot)
        fmt.Printf("Backend: PostgreSQL FTS\n")
    }

    // Initialize PostgreSQL FTS store with the DSN from EnsurePostgresRunning
    st, err := store.NewPostgresFTSStore(ctx, dsn, projectRoot)
    if err != nil {
        return fmt.Errorf("failed to connect to postgres: %w", err)
    }
    defer st.Close()

    // ... rest of the function unchanged
}
```

### T011: Update Watch Command Help Text

**File**: `cli/watch.go` (MODIFY)

Update the command's Long description:

```go
var watchCmd = &cobra.Command{
    Use:   "watch",
    Short: "Start the real-time file watcher daemon",
    Long: `Start a background process that monitors file changes and maintains the index.

The watcher will:
- Start a PostgreSQL container if not already running (requires Docker)
- Perform an initial scan comparing disk state with existing index
- Remove obsolete entries and index new files
- Monitor filesystem events (create, modify, delete, rename)
- Apply debouncing (500ms) to batch rapid changes
- Handle atomic updates to avoid duplicate vectors

Container Options:
  --pg-name, -n    Custom container name (default: agentdx-postgres)
  --pg-port, -p    Custom host port (default: 55432)

The PostgreSQL container persists after agentdx exits to preserve your index.`,
    RunE: runWatch,
}
```

### T012: Write Integration Test

**File**: `cli/watch_integration_test.go` (NEW or MODIFY)

Test cases:
- watch with default settings creates/uses agentdx-postgres container
- watch with --pg-name creates container with custom name
- watch with --pg-port uses custom port
- watch with both flags uses custom name and port
- watch reuses existing running container

## Acceptance Criteria

- [ ] `agentdx watch --help` shows --pg-name and --pg-port flags
- [ ] `agentdx watch` auto-starts container if not running
- [ ] `agentdx watch --pg-name mydb` uses custom container name
- [ ] `agentdx watch --pg-port 5433` uses custom port
- [ ] Short aliases `-n` and `-p` work
- [ ] Existing running container is reused (not recreated)
- [ ] Integration tests pass

## Files Changed

| File | Change |
|------|--------|
| `cli/watch.go` | MODIFY - add flags, call EnsurePostgresRunning |
| `cli/watch_integration_test.go` | NEW/MODIFY |

## Testing Commands

```bash
# Test help output
go run . watch --help

# Test with default settings (Docker required)
go run . watch

# Test with custom settings
go run . watch --pg-name test-watch --pg-port 5433

# Test short aliases
go run . watch -n test-alias -p 5434

# Run integration tests
go test ./cli/... -v -run TestWatch
```

## Activity Log

- 2026-01-22T08:44:40Z – unknown – lane=doing – Moved to doing
- 2026-01-22T08:46:43Z – unknown – lane=for_review – Moved to for_review
- 2026-01-22T09:00:02Z – unknown – lane=done – Moved to done
