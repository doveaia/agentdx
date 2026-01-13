# Feature Specification: Local Init with Auto PostgreSQL Setup

**Feature Branch**: `001-local-init-auto-postgresql-setup`
**Created**: 2026-01-13
**Status**: Draft
**Input**: User description: "Add --local (-l) flag to agentdx init command for automated local PostgreSQL setup"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Developer Quick Local Setup (Priority: P1)

A developer wants to quickly initialize agentdx for local development without answering interactive prompts. They run `agentdx init --local` and the tool automatically configures PostgreSQL Full Text Search backend, sets up the Docker container if available, creates the database, and writes the configuration file.

**Why this priority**: This is the core value proposition of the feature - enabling one-command local setup for the most common development use case.

**Independent Test**: Can be fully tested by running `agentdx init --local` in a project directory with Docker installed. Delivers immediate value by eliminating manual setup steps.

**Acceptance Scenarios**:

1. **Given** a project directory without `.agentdx/config.yaml` and Docker installed, **When** user runs `agentdx init --local`, **Then** config.yaml is created with `mode: local`, PostgreSQL FTS backend configured, Docker container `agentdx-postgres` is created and started, database is created, and command completes without any prompts.

2. **Given** a project directory without `.agentdx/config.yaml` and Docker installed with existing stopped `agentdx-postgres` container, **When** user runs `agentdx init --local`, **Then** the existing container is started (not recreated), database is created, and config.yaml is written correctly.

3. **Given** a project directory without `.agentdx/config.yaml` and Docker installed with running `agentdx-postgres` container, **When** user runs `agentdx init --local`, **Then** the container is reused, database is created if not exists, and config.yaml is written correctly.

---

### User Story 2 - Non-Docker Fallback Setup (Priority: P2)

A developer without Docker installed wants to use `--local` flag. The tool should still create the configuration, generate a docker-compose file for future use, and provide clear instructions on manual database setup.

**Why this priority**: Important for users who cannot or choose not to use Docker, ensuring the feature degrades gracefully.

**Independent Test**: Can be tested by running `agentdx init --local` on a machine without Docker CLI. Delivers value by providing configuration and clear next steps.

**Acceptance Scenarios**:

1. **Given** a project directory without Docker CLI available, **When** user runs `agentdx init --local`, **Then** config.yaml is created with correct DSN, `.agentdx/compose.yaml` is generated, and output displays the DSN and instructions for manual database creation.

2. **Given** a project directory without Docker CLI available, **When** user runs `agentdx init --local`, **Then** output references PostgreSQL 17 with pg_search extensions and provides the GitHub link for pg_textsearch.

---

### User Story 3 - Short Flag Alias (Priority: P3)

A developer prefers using the short `-l` flag instead of `--local` for brevity.

**Why this priority**: Convenience feature that improves developer experience but is not essential for core functionality.

**Independent Test**: Can be tested by running `agentdx init -l` and verifying identical behavior to `--local`.

**Acceptance Scenarios**:

1. **Given** any project directory, **When** user runs `agentdx init -l`, **Then** the behavior is identical to `agentdx init --local`.

---

### User Story 4 - Default Remote Mode (Priority: P3)

When initializing without the `--local` flag, the existing interactive behavior should remain unchanged, but config.yaml should now include `mode: remote` as the default.

**Why this priority**: Ensures backward compatibility while adding the new mode field.

**Independent Test**: Can be tested by running `agentdx init` (without --local) and verifying prompts appear and `mode: remote` is in config.yaml.

**Acceptance Scenarios**:

1. **Given** a project directory without `.agentdx/config.yaml`, **When** user runs `agentdx init` (without --local), **Then** interactive prompts are shown as before and config.yaml includes `mode: remote`.

---

### Edge Cases

- What happens when the project folder name contains special characters? Only alphanumeric and underscores allowed in slug; other characters are removed or replaced with underscores.
- How does system handle PostgreSQL connection timeout? Retry with exponential backoff for approximately 30 seconds, then fail with clear error message.
- What happens if database already exists? Skip creation, continue successfully.
- What happens if Docker container creation fails? Display error, fall back to generating compose.yaml and showing manual instructions.
- What happens if `.agentdx/config.yaml` already exists? Existing behavior applies - overwrite with new configuration.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST accept `--local` flag on the `init` command to enable non-interactive local setup mode
- **FR-002**: System MUST accept `-l` as a short alias for `--local` flag
- **FR-003**: System MUST add `mode` field to config.yaml with value `local` when `--local` flag is used
- **FR-004**: System MUST add `mode` field to config.yaml with value `remote` when `--local` flag is NOT used (default)
- **FR-005**: System MUST skip all interactive prompts when `--local` flag is used
- **FR-006**: System MUST configure PostgreSQL Full Text Search as the store backend when `--local` flag is used
- **FR-007**: System MUST generate DSN in format `postgres://agentdx:agentdx@localhost:55432/agentdx_<project_slug>?sslmode=disable`
- **FR-008**: System MUST convert project folder name to slug format (lowercase, spaces to underscores, only alphanumeric and underscores retained)
- **FR-009**: System MUST check for Docker CLI availability before attempting container operations
- **FR-010**: System MUST create Docker container `agentdx-postgres` if Docker is available and container does not exist
- **FR-011**: System MUST use image `doveaia/timescaledb:latest-pg17-ts` for container creation
- **FR-012**: System MUST set container environment variables `POSTGRES_USER=agentdx` and `POSTGRES_PASSWORD=agentdx`
- **FR-013**: System MUST set container restart policy to `always`
- **FR-014**: System MUST map container port 5432 to host port 55432
- **FR-015**: System MUST start existing stopped `agentdx-postgres` container if found
- **FR-016**: System MUST wait for PostgreSQL to be ready with retry mechanism (approximately 30 second timeout)
- **FR-017**: System MUST create database `agentdx_<project_slug>` after PostgreSQL is ready
- **FR-018**: System MUST generate `.agentdx/compose.yaml` when Docker CLI is not available
- **FR-019**: System MUST display DSN and manual setup instructions when Docker CLI is not available
- **FR-020**: System MUST reference pg_search extensions (https://github.com/timescale/pg_textsearch) in manual setup instructions
- **FR-021**: System MUST preserve existing `init` command behavior when `--local` flag is not used

### Key Entities *(include if feature involves data)*

- **Config Mode**: A field in config.yaml that indicates whether agentdx was initialized for local development (`local`) or remote/cloud setup (`remote`)
- **Project Slug**: A sanitized version of the project folder name used in database naming, following the pattern: lowercase, underscores only, alphanumeric characters only
- **Docker Container (agentdx-postgres)**: A PostgreSQL container instance managed by agentdx for local development, using TimescaleDB image with pg_search extensions

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can complete local initialization in under 60 seconds (excluding Docker image download time)
- **SC-002**: Zero interactive prompts are shown when using `--local` flag
- **SC-003**: 100% of project folder names are correctly converted to valid database-safe slugs
- **SC-004**: PostgreSQL readiness check succeeds within 30 seconds for a healthy container
- **SC-005**: Users without Docker receive actionable instructions and a ready-to-use compose.yaml file
- **SC-006**: Existing `agentdx init` behavior remains unchanged when `--local` flag is not used
