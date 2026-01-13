---
work_package_id: "WP01"
subtasks:
  - "T001"
  - "T002"
  - "T003"
  - "T004"
title: "Config Mode Field & Slug Utility"
phase: "Phase 1 - Foundation"
lane: "for_review"
assignee: ""
agent: "claude"
shell_pid: "54835"
review_status: ""
reviewed_by: ""
history:
  - timestamp: "2026-01-13T15:30:00Z"
    lane: "planned"
    agent: "system"
    shell_pid: ""
    action: "Prompt generated via /spec-kitty.tasks"
---

# Work Package Prompt: WP01 – Config Mode Field & Slug Utility

## Review Feedback

*[This section is empty initially. Reviewers will populate it if the work is returned from review.]*

---

## Objectives & Success Criteria

**Goal**: Add `mode` field to config struct and create foundational utilities for the localsetup package.

**Success Criteria**:
- `make build` passes with no errors
- `make lint` passes with no violations
- Config struct includes `Mode` field with YAML tag `mode`
- `DefaultConfig()` returns config with `Mode: "remote"`
- New `localsetup/` package created with interfaces and slug utility
- `ToSlug()` function correctly sanitizes project folder names

---

## Context & Constraints

**Reference Documents**:
- Constitution: `.kittify/memory/constitution.md` (interface-first architecture, code quality)
- Plan: `kitty-specs/001-local-init-auto-postgresql-setup/plan.md`
- Data Model: `kitty-specs/001-local-init-auto-postgresql-setup/data-model.md`
- Spec: `kitty-specs/001-local-init-auto-postgresql-setup/spec.md`

**Key Constraints**:
- Must not break existing config loading (backward compatible)
- Follow Go conventions and effective Go practices
- All exported types must have godoc comments

---

## Subtasks & Detailed Guidance

### Subtask T001 – Add Mode field to Config struct

- **Purpose**: Enable tracking whether agentdx was initialized for local or remote use.
- **Files**: `config/config.go`
- **Steps**:
  1. Add `Mode` field to `Config` struct:
     ```go
     type Config struct {
         Version int          `yaml:"version"`
         Mode    string       `yaml:"mode"`  // "local" or "remote"
         Index   IndexSection `yaml:"index"`
     }
     ```
  2. Add godoc comment explaining the field
- **Parallel?**: No, must be done before T002
- **Notes**: YAML unmarshaling handles new field automatically for existing configs (defaults to empty string)

### Subtask T002 – Update DefaultConfig() to set mode: "remote"

- **Purpose**: Ensure default config has remote mode for backward compatibility.
- **Files**: `config/config.go`
- **Steps**:
  1. In `DefaultConfig()` function, set `Mode: "remote"`:
     ```go
     func DefaultConfig() *Config {
         return &Config{
             Version: 1,
             Mode:    "remote",
             Index: IndexSection{
                 // ... existing defaults
             },
         }
     }
     ```
- **Parallel?**: No, depends on T001
- **Notes**: This is the default for `agentdx init` without --local flag

### Subtask T003 – Create localsetup/interfaces.go

- **Purpose**: Define interfaces for Docker and database operations (interface-first architecture).
- **Files**: `localsetup/interfaces.go` (new file)
- **Steps**:
  1. Create `localsetup/` directory
  2. Create `interfaces.go` with:
     ```go
     package localsetup

     import "time"

     // DockerClient defines operations for Docker container management.
     type DockerClient interface {
         IsAvailable() bool
         ContainerExists(name string) (bool, error)
         ContainerRunning(name string) (bool, error)
         CreateContainer(cfg ContainerConfig) error
         StartContainer(name string) error
     }

     // ContainerConfig specifies Docker container settings.
     type ContainerConfig struct {
         Name          string
         Image         string
         EnvVars       map[string]string
         HostPort      string
         ContainerPort string
         RestartPolicy string
     }

     // DatabaseClient defines operations for PostgreSQL database management.
     type DatabaseClient interface {
         WaitForReady(dsn string, timeout time.Duration) error
         CreateDatabase(dsn, dbName string) error
     }
     ```
- **Parallel?**: Yes, can be done in parallel with T004
- **Notes**: Interfaces enable mocking for tests and future extensibility

### Subtask T004 – Create localsetup/slug.go

- **Purpose**: Convert project folder names to valid database-safe slugs.
- **Files**: `localsetup/slug.go` (new file)
- **Steps**:
  1. Create `slug.go` with `ToSlug` function:
     ```go
     package localsetup

     import (
         "regexp"
         "strings"
     )

     // ToSlug converts a project folder name to a database-safe slug.
     // Rules:
     // - Convert to lowercase
     // - Replace hyphens and spaces with underscores
     // - Remove all characters except a-z, 0-9, _
     // - Collapse multiple underscores to single underscore
     // - Trim leading/trailing underscores
     func ToSlug(name string) string {
         // Convert to lowercase
         s := strings.ToLower(name)

         // Replace hyphens and spaces with underscores
         s = strings.ReplaceAll(s, "-", "_")
         s = strings.ReplaceAll(s, " ", "_")

         // Remove non-alphanumeric characters (except underscore)
         re := regexp.MustCompile(`[^a-z0-9_]`)
         s = re.ReplaceAllString(s, "")

         // Collapse multiple underscores
         re = regexp.MustCompile(`_+`)
         s = re.ReplaceAllString(s, "_")

         // Trim leading/trailing underscores
         s = strings.Trim(s, "_")

         return s
     }
     ```
- **Parallel?**: Yes, can be done in parallel with T003
- **Notes**: Test cases to consider: "my-project" → "my_project", "My Project" → "my_project", "café-app" → "caf_app"

---

## Definition of Done Checklist

- [ ] T001: Mode field added to Config struct with godoc comment
- [ ] T002: DefaultConfig() returns Mode: "remote"
- [ ] T003: localsetup/interfaces.go created with DockerClient and DatabaseClient interfaces
- [ ] T004: localsetup/slug.go created with ToSlug function
- [ ] `make build` passes
- [ ] `make lint` passes
- [ ] Code follows Go conventions (godoc comments, proper naming)

---

## Review Guidance

- Verify Mode field is correctly placed in Config struct (after Version, before Index)
- Check YAML tag is exactly `mode` (lowercase)
- Verify interfaces match the design in data-model.md
- Test ToSlug with edge cases: empty string, all special chars, unicode

---

## Activity Log

- 2026-01-13T15:30:00Z – system – lane=planned – Prompt created.
- 2026-01-13T16:00:46Z – claude – shell_pid=50020 – lane=doing – Started implementation
- 2026-01-13T16:08:13Z – claude – shell_pid=54835 – lane=for_review – Ready for review
