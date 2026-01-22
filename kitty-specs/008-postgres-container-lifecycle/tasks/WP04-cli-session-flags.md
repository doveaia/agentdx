---
work_package_id: "WP04"
title: "CLI Flags for session start Command"
lane: "done"
subtasks:
  - "T013"
  - "T014"
  - "T015"
  - "T016"
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

# Work Package Prompt: WP04 – CLI Flags for session start Command

## Objective

Add `--pg-name` (alias `-n`) and `--pg-port` (alias `-p`) flags to the `agentdx session start` command. The session daemon needs to know these settings to pass them to the watch subprocess.

## Context

The `session start` command spawns a background daemon that runs `agentdx watch --daemon`. The container settings need to be propagated to this subprocess.

## Subtasks

### T013: Add Flags to sessionStartCmd

**File**: `cli/session.go` (MODIFY)

Add package-level variables for the flags:

```go
var (
    quietMode     bool
    forceStop     bool
    jsonOutput    bool
    sessionPgName string
    sessionPgPort int
)

func init() {
    // ... existing flag setup ...

    // session start flags
    sessionStartCmd.Flags().BoolVarP(&quietMode, "quiet", "q", false, "Suppress output")
    sessionStartCmd.Flags().StringVarP(&sessionPgName, "pg-name", "n", "", "PostgreSQL container name (default: agentdx-postgres)")
    sessionStartCmd.Flags().IntVarP(&sessionPgPort, "pg-port", "p", 0, "PostgreSQL host port (default: 55432)")

    // ... rest of flag setup ...
}
```

### T014: Propagate Flags to Daemon Process

**File**: `cli/session.go` (MODIFY)

The daemon manager needs to know the container settings. Pass them via environment variables or command line args to the subprocess.

**Option A: Environment Variables** (Recommended)

In `runSessionStart`:

```go
func runSessionStart(cmd *cobra.Command, args []string) error {
    ctx := context.Background()

    projectRoot, err := config.FindProjectRoot()
    if err != nil {
        if !quietMode {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        }
        return err
    }

    // Ensure PostgreSQL is running BEFORE starting daemon
    opts := localsetup.ContainerOptions{
        Name: sessionPgName,
        Port: sessionPgPort,
    }
    _, err = localsetup.EnsurePostgresRunning(ctx, projectRoot, opts)
    if err != nil {
        if !quietMode {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        }
        return err
    }

    // Create daemon manager with container options
    dm := session.NewDaemonManagerWithOptions(projectRoot, session.DaemonOptions{
        PgName: sessionPgName,
        PgPort: sessionPgPort,
    })

    // ... rest of function
}
```

**File**: `session/daemon.go` (MODIFY)

Update DaemonManager to accept and propagate options:

```go
type DaemonOptions struct {
    PgName string
    PgPort int
}

func NewDaemonManagerWithOptions(projectRoot string, opts DaemonOptions) *DaemonManager {
    // ...
}

func (dm *DaemonManager) Start(ctx context.Context) error {
    // Build command with flags if set
    args := []string{"watch", "--daemon"}
    if dm.opts.PgName != "" {
        args = append(args, "--pg-name", dm.opts.PgName)
    }
    if dm.opts.PgPort != 0 {
        args = append(args, "--pg-port", strconv.Itoa(dm.opts.PgPort))
    }
    // ...
}
```

### T015: Update Session Start Help Text

**File**: `cli/session.go` (MODIFY)

Update the command's Long description and Example:

```go
var sessionStartCmd = &cobra.Command{
    Use:   "start",
    Short: "Start the watch daemon",
    Long: `Start the agentdx watch daemon as a background process.

If the daemon is already running, this command does nothing (idempotent).
If PostgreSQL is not running, it will be started automatically (requires Docker).

Container Options:
  --pg-name, -n    Custom container name (default: agentdx-postgres)
  --pg-port, -p    Custom host port (default: 55432)`,
    Example: `  # Start daemon (typical usage)
  agentdx session start

  # Start with custom container settings
  agentdx session start --pg-name my-project-pg --pg-port 5433

  # Start silently (for scripts/hooks)
  agentdx session start --quiet`,
    RunE: runSessionStart,
}
```

### T016: Write Integration Test

**File**: `cli/session_integration_test.go` (NEW or MODIFY)

Test cases:
- session start with default settings creates/uses agentdx-postgres container
- session start with --pg-name propagates to daemon
- session start with --pg-port propagates to daemon
- daemon uses correct container settings

## Acceptance Criteria

- [ ] `agentdx session start --help` shows --pg-name and --pg-port flags
- [ ] `agentdx session start` auto-starts container if not running
- [ ] `agentdx session start --pg-name mydb` uses custom container name
- [ ] `agentdx session start --pg-port 5433` uses custom port
- [ ] Settings are propagated to the daemon subprocess
- [ ] Integration tests pass

## Files Changed

| File | Change |
|------|--------|
| `cli/session.go` | MODIFY - add flags, propagate to daemon |
| `session/daemon.go` | MODIFY - accept container options |
| `cli/session_integration_test.go` | NEW/MODIFY |

## Testing Commands

```bash
# Test help output
go run . session start --help

# Test with default settings (Docker required)
go run . session start
go run . session status
go run . session stop

# Test with custom settings
go run . session start --pg-name test-session --pg-port 5435
go run . session status
go run . session stop

# Run integration tests
go test ./cli/... -v -run TestSession
```

## Activity Log

- 2026-01-22T08:46:43Z – unknown – lane=doing – Moved to doing
- 2026-01-22T08:50:32Z – unknown – lane=for_review – Moved to for_review
- 2026-01-22T09:00:02Z – unknown – lane=done – Moved to done
