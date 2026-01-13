---
work_package_id: "WP03"
subtasks:
  - "T006"
  - "T007"
title: "Database Setup & Compose Generation"
phase: "Phase 2 - Core Implementation"
lane: "doing"
assignee: ""
agent: "claude"
shell_pid: "56661"
review_status: ""
reviewed_by: ""
history:
  - timestamp: "2026-01-13T15:30:00Z"
    lane: "planned"
    agent: "system"
    shell_pid: ""
    action: "Prompt generated via /spec-kitty.tasks"
---

# Work Package Prompt: WP03 – Database Setup & Compose Generation

## Review Feedback

*[This section is empty initially. Reviewers will populate it if the work is returned from review.]*

---

## Objectives & Success Criteria

**Goal**: Implement PostgreSQL database creation with retry logic and compose.yaml generation.

**Success Criteria**:
- `WaitForPostgres()` polls until PostgreSQL is ready (max 30s)
- `CreateDatabase()` creates database if it doesn't exist
- `GenerateComposeYAML()` returns correct Docker Compose content
- `WriteComposeFile()` writes to `.agentdx/compose.yaml`
- Retry logic uses exponential backoff
- Database creation handles "already exists" gracefully

---

## Context & Constraints

**Reference Documents**:
- Constitution: `.kittify/memory/constitution.md`
- Plan: `kitty-specs/001-local-init-auto-postgresql-setup/plan.md`
- Data Model: `kitty-specs/001-local-init-auto-postgresql-setup/data-model.md`
- Spec: `kitty-specs/001-local-init-auto-postgresql-setup/spec.md` (FR-016 through FR-020)

**Key Constraints**:
- Use `database/sql` with `lib/pq` driver
- Retry with exponential backoff, 30s total timeout
- Connect to `postgres` database first, then CREATE DATABASE
- Compose file uses `doveaia/timescaledb:latest-pg17-ts` image
- Port mapping: 55432:5432

**Dependencies**:
- Depends on WP01 (interfaces.go for DatabaseClient interface)

---

## Subtasks & Detailed Guidance

### Subtask T006 – Create localsetup/database.go

- **Purpose**: Implement PostgreSQL database operations with retry logic.
- **Files**: `localsetup/database.go` (new file)
- **Steps**:

1. Create the file with imports:
```go
package localsetup

import (
    "database/sql"
    "fmt"
    "time"

    _ "github.com/lib/pq"
)

const (
    maxRetries     = 20
    initialBackoff = 500 * time.Millisecond
    maxBackoff     = 5 * time.Second
)
```

2. Implement `WaitForPostgres()`:
```go
// WaitForPostgres polls the PostgreSQL server until it's ready or timeout expires.
// Uses exponential backoff starting at 500ms, maxing at 5s between attempts.
func WaitForPostgres(dsn string, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    backoff := initialBackoff

    for time.Now().Before(deadline) {
        db, err := sql.Open("postgres", dsn)
        if err != nil {
            return fmt.Errorf("failed to open database connection: %w", err)
        }

        err = db.Ping()
        db.Close()

        if err == nil {
            return nil
        }

        // Sleep with exponential backoff
        time.Sleep(backoff)
        backoff *= 2
        if backoff > maxBackoff {
            backoff = maxBackoff
        }
    }

    return fmt.Errorf("timeout waiting for PostgreSQL to be ready after %v", timeout)
}
```

3. Implement `CreateDatabase()`:
```go
// CreateDatabase creates a new database if it doesn't already exist.
// Connects to the 'postgres' default database to execute CREATE DATABASE.
func CreateDatabase(dsn, dbName string) error {
    // Connect to default postgres database
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
    }
    defer db.Close()

    // Check if database already exists
    var exists bool
    err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists)
    if err != nil {
        return fmt.Errorf("failed to check database existence: %w", err)
    }

    if exists {
        return nil // Database already exists, nothing to do
    }

    // Create the database (can't use parameterized query for CREATE DATABASE)
    // dbName is derived from ToSlug() which only allows alphanumeric and underscore
    _, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
    if err != nil {
        return fmt.Errorf("failed to create database %s: %w", dbName, err)
    }

    return nil
}
```

4. Add helper to build postgres DSN:
```go
// PostgresDSN returns a DSN for connecting to the postgres default database.
func PostgresDSN() string {
    return "postgres://agentdx:agentdx@localhost:55432/postgres?sslmode=disable"
}

// ProjectDSN returns a DSN for connecting to the project-specific database.
func ProjectDSN(dbName string) string {
    return fmt.Sprintf("postgres://agentdx:agentdx@localhost:55432/%s?sslmode=disable", dbName)
}
```

- **Parallel?**: No (single file)
- **Notes**:
  - Use the `postgres` database for initial connection and CREATE DATABASE
  - dbName comes from ToSlug() so it's safe to use in SQL (alphanumeric + underscore only)
  - Still, be cautious with string formatting in SQL

### Subtask T007 – Create localsetup/compose.go

- **Purpose**: Generate Docker Compose file for manual setup.
- **Files**: `localsetup/compose.go` (new file)
- **Steps**:

1. Create the file:
```go
package localsetup

import (
    "fmt"
    "os"
    "path/filepath"
)

const composeTemplate = `services:
  postgres:
    image: doveaia/timescaledb:latest-pg17-ts
    container_name: agentdx-postgres
    environment:
      POSTGRES_USER: agentdx
      POSTGRES_PASSWORD: agentdx
    ports:
      - "55432:5432"
    volumes:
      - agentdx-pgdata:/var/lib/postgresql/data
    restart: always
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U agentdx"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  agentdx-pgdata:
`

// GenerateComposeYAML returns the Docker Compose file content.
func GenerateComposeYAML() string {
    return composeTemplate
}

// WriteComposeFile writes the compose.yaml file to the .agentdx directory.
func WriteComposeFile(projectRoot string) error {
    agentdxDir := filepath.Join(projectRoot, ".agentdx")

    // Ensure .agentdx directory exists
    if err := os.MkdirAll(agentdxDir, 0755); err != nil {
        return fmt.Errorf("failed to create .agentdx directory: %w", err)
    }

    composePath := filepath.Join(agentdxDir, "compose.yaml")
    content := GenerateComposeYAML()

    if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
        return fmt.Errorf("failed to write compose.yaml: %w", err)
    }

    return nil
}
```

- **Parallel?**: Yes, can be done in parallel with T006
- **Notes**:
  - Compose uses port 55432 to avoid conflicts with existing PostgreSQL
  - restart: always ensures container starts on boot
  - Healthcheck enables compose to wait for PostgreSQL readiness

---

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| PostgreSQL slow to start | Exponential backoff with 30s total timeout |
| Database already exists | Check before creating, return nil if exists |
| Connection errors | Clear error messages with wrapped context |

---

## Definition of Done Checklist

- [ ] T006: localsetup/database.go created
- [ ] WaitForPostgres() implements retry with exponential backoff
- [ ] CreateDatabase() checks existence before creating
- [ ] Helper functions for DSN generation
- [ ] T007: localsetup/compose.go created
- [ ] GenerateComposeYAML() returns valid compose content
- [ ] WriteComposeFile() creates .agentdx directory if needed
- [ ] `make build` passes
- [ ] `make lint` passes

---

## Review Guidance

- Verify exponential backoff timing is reasonable (500ms → 1s → 2s → 4s → 5s cap)
- Check database existence query is correct
- Verify compose.yaml matches spec requirements (image, ports, env, restart, healthcheck)
- Test with existing database to ensure no error

---

## Activity Log

- 2026-01-13T15:30:00Z – system – lane=planned – Prompt created.
- 2026-01-13T16:10:42Z – claude – shell_pid=56661 – lane=doing – Started implementation
