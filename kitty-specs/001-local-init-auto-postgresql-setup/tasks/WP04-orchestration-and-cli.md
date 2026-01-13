---
work_package_id: "WP04"
subtasks:
  - "T008"
  - "T009"
  - "T010"
  - "T011"
title: "Orchestration & CLI Integration"
phase: "Phase 3 - Integration"
lane: "done"
assignee: ""
agent: "claude"
shell_pid: "16923"
review_status: ""
reviewed_by: ""
history:
  - timestamp: "2026-01-13T15:30:00Z"
    lane: "planned"
    agent: "system"
    shell_pid: ""
    action: "Prompt generated via /spec-kitty.tasks"
  - timestamp: "2026-01-13T22:31:00Z"
    lane: "doing"
    agent: "claude"
    shell_pid: "16923"
    action: "Started implementation"
  - timestamp: "2026-01-13T22:35:00Z"
    lane: "for_review"
    agent: "claude"
    shell_pid: "16923"
    action: "Ready for review"
---

# Work Package Prompt: WP04 – Orchestration & CLI Integration

## Review Feedback

*[This section is empty initially. Reviewers will populate it if the work is returned from review.]*

---

## Objectives & Success Criteria

**Goal**: Wire everything together - main orchestration function and CLI flag integration.

**Success Criteria**:
- `agentdx init --local` runs end-to-end without prompts
- `agentdx init -l` works as alias
- Docker container created/started when Docker available
- Database created after PostgreSQL ready
- compose.yaml always generated
- Fallback instructions shown when Docker unavailable
- Existing `agentdx init` behavior unchanged (mode: remote)
- Config saved with correct DSN and mode

---

## Context & Constraints

**Reference Documents**:
- Constitution: `.kittify/memory/constitution.md`
- Plan: `kitty-specs/001-local-init-auto-postgresql-setup/plan.md`
- Spec: `kitty-specs/001-local-init-auto-postgresql-setup/spec.md` (all FRs)
- Existing code: `cli/init.go`

**Key Constraints**:
- Must not break existing interactive init behavior
- compose.yaml generated regardless of Docker availability
- Clear output messages for user feedback
- Graceful fallback when Docker unavailable

**Dependencies**:
- Depends on WP01, WP02, WP03 (all foundation packages)

---

## Subtasks & Detailed Guidance

### Subtask T008 – Create localsetup/localsetup.go

- **Purpose**: Main orchestration that ties all components together.
- **Files**: `localsetup/localsetup.go` (new file)
- **Steps**:

1. Create the file with imports and result struct:
```go
package localsetup

import (
    "fmt"
    "path/filepath"
    "time"
)

const (
    postgresReadyTimeout = 30 * time.Second
)

// SetupResult contains the outcome of a local setup operation.
type SetupResult struct {
    Mode             string // "local"
    DSN              string // Full PostgreSQL connection string
    DatabaseName     string // e.g., "agentdx_my_project"
    ContainerName    string // "agentdx-postgres"
    DockerUsed       bool   // Whether Docker was available and used
    ComposeGenerated bool   // Whether compose.yaml was generated
    ComposeFilePath  string // Path to generated compose.yaml
}
```

2. Implement `RunLocalSetup()`:
```go
// RunLocalSetup orchestrates the complete local development setup.
// It creates/starts the Docker container if Docker is available,
// waits for PostgreSQL to be ready, creates the project database,
// and always generates the compose.yaml file.
func RunLocalSetup(projectRoot string) (*SetupResult, error) {
    // Get project folder name and convert to slug
    projectName := filepath.Base(projectRoot)
    dbName := "agentdx_" + ToSlug(projectName)

    if dbName == "agentdx_" {
        return nil, fmt.Errorf("project folder name '%s' produces empty slug", projectName)
    }

    result := &SetupResult{
        Mode:          "local",
        DatabaseName:  dbName,
        ContainerName: "agentdx-postgres",
        DSN:           ProjectDSN(dbName),
    }

    // Always generate compose.yaml
    if err := WriteComposeFile(projectRoot); err != nil {
        return nil, fmt.Errorf("failed to generate compose.yaml: %w", err)
    }
    result.ComposeGenerated = true
    result.ComposeFilePath = filepath.Join(projectRoot, ".agentdx", "compose.yaml")

    // Check if Docker is available
    if !IsDockerAvailable() {
        // Docker not available - compose.yaml generated, return with instructions
        result.DockerUsed = false
        return result, nil
    }

    result.DockerUsed = true

    // Check if container exists
    exists, err := ContainerExists(result.ContainerName)
    if err != nil {
        return nil, fmt.Errorf("failed to check container: %w", err)
    }

    if !exists {
        // Create the container
        cfg := DefaultContainerConfig()
        if err := CreateContainer(cfg); err != nil {
            return nil, fmt.Errorf("failed to create container: %w", err)
        }
    } else {
        // Container exists, check if running
        running, err := ContainerRunning(result.ContainerName)
        if err != nil {
            return nil, fmt.Errorf("failed to check container state: %w", err)
        }
        if !running {
            // Start the stopped container
            if err := StartContainer(result.ContainerName); err != nil {
                return nil, fmt.Errorf("failed to start container: %w", err)
            }
        }
    }

    // Wait for PostgreSQL to be ready
    if err := WaitForPostgres(PostgresDSN(), postgresReadyTimeout); err != nil {
        return nil, fmt.Errorf("PostgreSQL not ready: %w", err)
    }

    // Create the project database
    if err := CreateDatabase(PostgresDSN(), dbName); err != nil {
        return nil, fmt.Errorf("failed to create database: %w", err)
    }

    return result, nil
}
```

- **Parallel?**: No (main orchestration)
- **Notes**:
  - Flow: generate compose → check Docker → create/start container → wait for PG → create DB
  - compose.yaml generated first so it's available even if later steps fail

### Subtask T009 – Add --local/-l flag to cli/init.go

- **Purpose**: Add the flag that triggers local setup mode.
- **Files**: `cli/init.go`
- **Steps**:

1. Add flag variable at top of file (with existing flags):
```go
var (
    initProvider       string
    initBackend        string
    initNonInteractive bool
    initLocal          bool  // NEW
)
```

2. Register the flag in `init()` function:
```go
func init() {
    initCmd.Flags().StringVarP(&initProvider, "provider", "p", "", "Embedding provider (ollama, lmstudio, openai, or postgres)")
    initCmd.Flags().StringVarP(&initBackend, "backend", "b", "", "Storage backend (gob or postgres)")
    initCmd.Flags().BoolVar(&initNonInteractive, "yes", false, "Use defaults without prompting")
    initCmd.Flags().BoolVarP(&initLocal, "local", "l", false, "Non-interactive local setup with PostgreSQL FTS")  // NEW
}
```

- **Parallel?**: No, must be done before T010
- **Notes**: `-l` is the short form, `--local` is the long form

### Subtask T010 – Implement local setup flow in cli/init.go

- **Purpose**: Handle the --local flag by delegating to localsetup package.
- **Files**: `cli/init.go`
- **Steps**:

1. Add import for localsetup package:
```go
import (
    // ... existing imports
    "github.com/doveaia/agentdx/localsetup"
)
```

2. Add local setup handling at the start of `runInit()` (after getting cwd):
```go
func runInit(cmd *cobra.Command, args []string) error {
    cwd, err := os.Getwd()
    if err != nil {
        return fmt.Errorf("failed to get current directory: %w", err)
    }

    // Handle --local flag
    if initLocal {
        return runLocalInit(cwd)
    }

    // ... rest of existing runInit code
}
```

3. Create `runLocalInit()` function:
```go
func runLocalInit(cwd string) error {
    // Check if already initialized (same check as interactive mode)
    if config.Exists(cwd) {
        fmt.Println("agentdx is already initialized in this directory.")
        fmt.Printf("Configuration: %s\n", config.GetConfigPath(cwd))
        return nil
    }

    fmt.Println("Initializing agentdx with local PostgreSQL setup...")

    // Run the local setup
    result, err := localsetup.RunLocalSetup(cwd)
    if err != nil {
        return fmt.Errorf("local setup failed: %w", err)
    }

    // Create and configure the config
    cfg := config.DefaultConfig()
    cfg.Mode = "local"
    cfg.Index.Embedder.Provider = "postgres"
    cfg.Index.Embedder.Model = "none"
    cfg.Index.Embedder.Endpoint = "none"
    cfg.Index.Embedder.Dimensions = 1536
    cfg.Index.Store.Backend = "postgres"
    cfg.Index.Store.Postgres.DSN = result.DSN

    // Save configuration
    if err := cfg.Save(cwd); err != nil {
        return fmt.Errorf("failed to save configuration: %w", err)
    }

    fmt.Printf("\nCreated configuration at %s\n", config.GetConfigPath(cwd))

    // Add .agentdx/ to .gitignore
    gitignorePath := cwd + "/.gitignore"
    if _, err := os.Stat(gitignorePath); err == nil {
        if err := indexer.AddToGitignore(cwd, ".agentdx/"); err != nil {
            fmt.Printf("Warning: could not update .gitignore: %v\n", err)
        } else {
            fmt.Println("Added .agentdx/ to .gitignore")
        }
    }

    // Print results
    if result.DockerUsed {
        fmt.Println("\nagentdx initialized successfully!")
        fmt.Printf("  Container: %s (running)\n", result.ContainerName)
        fmt.Printf("  Database:  %s\n", result.DatabaseName)
        fmt.Printf("  DSN:       %s\n", result.DSN)
    } else {
        fmt.Println("\nagentdx initialized (Docker not available).")
        fmt.Printf("  Database:  %s (needs manual creation)\n", result.DatabaseName)
        fmt.Printf("  DSN:       %s\n", result.DSN)
        fmt.Println("\nTo set up the database manually:")
        fmt.Println("  1. Install PostgreSQL 17 with pg_search extensions")
        fmt.Println("     See: https://github.com/timescale/pg_textsearch")
        fmt.Println("  2. Or install Docker and run:")
        fmt.Printf("     docker compose -f %s up -d\n", result.ComposeFilePath)
        fmt.Printf("  3. Create database: CREATE DATABASE %s;\n", result.DatabaseName)
    }

    if result.ComposeGenerated {
        fmt.Printf("\nDocker Compose file: %s\n", result.ComposeFilePath)
    }

    fmt.Println("\nNext steps:")
    fmt.Println("  1. Start the indexing daemon: agentdx watch")
    fmt.Println("  2. Search your code: agentdx search \"your query\"")

    return nil
}
```

- **Parallel?**: No, depends on T009
- **Notes**:
  - Early return after handling --local to avoid running interactive flow
  - PostgreSQL FTS config: provider=postgres, model=none, endpoint=none
  - Output different messages based on Docker availability

### Subtask T011 – Update existing init flow to set mode: remote

- **Purpose**: Ensure default (non-local) init sets mode to remote.
- **Files**: `cli/init.go`
- **Steps**:

1. In the existing `runInit()` function, after creating the default config:
```go
cfg := config.DefaultConfig()
// DefaultConfig() already sets Mode: "remote" (from T002)
// No additional changes needed here - just verify it works
```

2. This subtask is essentially verification that the config changes from WP01 work correctly with the existing flow.

- **Parallel?**: No
- **Notes**: The work is already done in T002, this is verification

---

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Breaking existing init | Early return for --local, don't modify existing flow |
| Port 55432 in use | Clear error from Docker, suggest checking port |
| Empty project name | Check slug is non-empty before proceeding |

---

## Definition of Done Checklist

- [ ] T008: localsetup/localsetup.go created with RunLocalSetup()
- [ ] T009: --local/-l flag added to init command
- [ ] T010: runLocalInit() implemented with full flow
- [ ] T011: Verified existing init sets mode: remote
- [ ] `agentdx init --local` works end-to-end (with Docker)
- [ ] `agentdx init --local` shows fallback (without Docker)
- [ ] `agentdx init` still works with prompts
- [ ] `make build` passes
- [ ] `make lint` passes

---

## Review Guidance

- Verify --local flag triggers runLocalInit(), not existing flow
- Check all output messages are user-friendly
- Test with Docker available and unavailable
- Verify config.yaml has correct DSN format
- Test with existing config (should say already initialized)

---

## Activity Log

- 2026-01-13T15:30:00Z – system – lane=planned – Prompt created.
- 2026-01-13T22:01:41Z – claude – lane=doing – Started review via workflow command
- 2026-01-13T22:01:52Z – claude – shell_pid=16923 – lane=done – Review passed - all acceptance criteria met
