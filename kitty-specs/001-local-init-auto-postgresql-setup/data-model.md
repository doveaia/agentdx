# Data Model: Local Init with Auto PostgreSQL Setup

**Feature**: 001-local-init-auto-postgresql-setup
**Date**: 2026-01-13

## Configuration Changes

### Config Struct (config/config.go)

**Current Structure**:
```go
type Config struct {
    Version int          `yaml:"version"`
    Index   IndexSection `yaml:"index"`
}
```

**Modified Structure**:
```go
type Config struct {
    Version int          `yaml:"version"`
    Mode    string       `yaml:"mode"`  // NEW: "local" or "remote"
    Index   IndexSection `yaml:"index"`
}
```

### Mode Field

| Field | Type | YAML Key | Values | Default |
|-------|------|----------|--------|---------|
| Mode | string | `mode` | `"local"`, `"remote"` | `"remote"` |

**Semantics**:
- `"remote"`: Standard initialization (interactive or with flags), intended for cloud/production use
- `"local"`: Non-interactive local development setup with auto-PostgreSQL configuration

### Example config.yaml (after `agentdx init --local`)

```yaml
version: 1
mode: local
index:
  embedder:
    provider: postgres
    model: none
    endpoint: none
    dimensions: 1536
  store:
    backend: postgres
    postgres:
      dsn: postgres://agentdx:agentdx@localhost:55432/agentdx_my_project?sslmode=disable
  chunking:
    size: 512
    overlap: 50
  watch:
    debounce_ms: 500
  search:
    hybrid:
      enabled: false
      k: 60
    boost:
      enabled: true
      # ... boost rules
  trace:
    mode: fast
    enabled_languages:
      - .go
      - .js
      # ...
  update:
    check_on_startup: false
  ignore:
    - .git
    - .agentdx
    # ...
```

## New Data Types (localsetup package)

### SetupResult

```go
// SetupResult contains the outcome of a local setup operation
type SetupResult struct {
    Mode           string // "local"
    DSN            string // Full PostgreSQL connection string
    DatabaseName   string // e.g., "agentdx_my_project"
    ContainerName  string // "agentdx-postgres"
    DockerUsed     bool   // Whether Docker was available and used
    ComposeGenerated bool // Whether compose.yaml was generated
    ComposeFilePath string // Path to generated compose.yaml
}
```

### ContainerConfig

```go
// ContainerConfig specifies Docker container settings
type ContainerConfig struct {
    Name          string            // Container name
    Image         string            // Docker image
    EnvVars       map[string]string // Environment variables
    HostPort      string            // Host port to expose
    ContainerPort string            // Container port to map
    RestartPolicy string            // "always", "unless-stopped", etc.
}
```

## Project Slug Generation

### Input → Output Examples

| Project Folder Name | Generated Slug |
|---------------------|----------------|
| `my-project` | `my_project` |
| `My Project` | `my_project` |
| `project_name` | `project_name` |
| `Project 123` | `project_123` |
| `café-app` | `caf_app` |
| `test@project!` | `testproject` |
| `123-numbers-first` | `123_numbers_first` |

### Slug Rules

1. Convert to lowercase
2. Replace hyphens and spaces with underscores
3. Remove all characters except: `a-z`, `0-9`, `_`
4. Collapse multiple consecutive underscores to single underscore
5. Trim leading/trailing underscores

## Compose File Structure (.agentdx/compose.yaml)

```yaml
services:
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
```

## Database Schema

No application-specific tables are created during init. The `agentdx_<slug>` database is created as an empty database. The existing `store/postgres.go` handles table creation during the first `agentdx watch` run.

## Validation Rules

1. **Mode field**: Must be either `"local"` or `"remote"`. Empty defaults to `"remote"`.
2. **DSN format**: When mode is `"local"`, DSN must follow pattern:
   ```
   postgres://agentdx:agentdx@localhost:55432/agentdx_<slug>?sslmode=disable
   ```
3. **Slug**: Must be non-empty after sanitization. If project folder name produces empty slug, error and suggest renaming.
