# Feature Specification: PostgreSQL Container Lifecycle Management

**Feature Branch**: `008-postgres-container-lifecycle`
**Created**: 2026-01-22
**Status**: Draft
**Input**: User description: "Fix postgres container issues - auto-start/stop container, configurable name/port, random containers for parallel tests"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Auto-start PostgreSQL on Watch (Priority: P1)

As a developer, I want `agentdx watch` to automatically start the PostgreSQL container if it's not running, so I don't have to manually manage Docker containers.

**Why this priority**: This is the primary pain point - users currently get connection refused errors when the container isn't running. Fixing this eliminates the most common failure mode.

**Independent Test**: Can be fully tested by running `agentdx watch` without a pre-started PostgreSQL container and verifying it starts indexing successfully.

**Acceptance Scenarios**:

1. **Given** no PostgreSQL container is running, **When** I run `agentdx watch`, **Then** the system starts a PostgreSQL container with default name `agentdx-postgres` on port `55432` and begins indexing
2. **Given** a PostgreSQL container named `agentdx-postgres` is already running on port `55432`, **When** I run `agentdx watch`, **Then** the system connects to the existing container without creating a new one
3. **Given** the PostgreSQL container is started by agentdx, **When** the `agentdx watch` process exits (Ctrl+C), **Then** the container continues running (persists)

---

### User Story 2 - Auto-start PostgreSQL on Session Start (Priority: P1)

As a developer, I want `agentdx session start` to automatically start the PostgreSQL container if it's not running, so my session daemon can immediately begin working.

**Why this priority**: Session start is equally important as watch - both need the same PostgreSQL availability guarantee.

**Independent Test**: Can be fully tested by running `agentdx session start` without a pre-started PostgreSQL container and verifying the daemon starts successfully.

**Acceptance Scenarios**:

1. **Given** no PostgreSQL container is running, **When** I run `agentdx session start`, **Then** the system starts a PostgreSQL container and the daemon starts successfully
2. **Given** a PostgreSQL container is already running, **When** I run `agentdx session start`, **Then** the system connects to the existing container

---

### User Story 3 - Custom Container Name and Port (Priority: P2)

As a developer working on multiple projects or with port conflicts, I want to specify a custom container name and port, so I can run multiple agentdx instances or avoid conflicts with other services.

**Why this priority**: While defaults cover most cases, power users need flexibility to avoid conflicts.

**Independent Test**: Can be fully tested by running `agentdx watch --pg-name mydb --pg-port 5433` and verifying a container named `mydb` is created on port `5433`.

**Acceptance Scenarios**:

1. **Given** I have another service on port 55432, **When** I run `agentdx watch --pg-port 5433`, **Then** the PostgreSQL container runs on port `5433`
2. **Given** I want a custom container name, **When** I run `agentdx watch --pg-name my-agentdx-pg`, **Then** the container is named `my-agentdx-pg`
3. **Given** I prefer short flags, **When** I run `agentdx watch -n mydb -p 5433`, **Then** the system uses `mydb` as container name and `5433` as port
4. **Given** I specify custom name/port via config file, **When** I run `agentdx watch`, **Then** the system uses the config values

---

### User Story 4 - Persistent Data via Docker Volume (Priority: P2)

As a developer, I want the PostgreSQL data to persist across container restarts, so I don't lose my index when the container stops.

**Why this priority**: Data persistence is essential for a good UX - users shouldn't need to re-index after a reboot.

**Independent Test**: Can be fully tested by indexing a project, restarting the container, and verifying the index is intact.

**Acceptance Scenarios**:

1. **Given** agentdx has indexed my project, **When** I restart the PostgreSQL container, **Then** my index data is preserved
2. **Given** I use a custom container name `mydb`, **When** the container is created, **Then** it uses a volume named `mydb-data` for persistence

---

### User Story 5 - Parallel Test Execution with Random Containers (Priority: P3)

As a developer running tests, I want each test package to use a unique PostgreSQL container with random name and port, so I can run tests in parallel without conflicts.

**Why this priority**: Important for CI and local test runs, but less urgent than the user-facing commands.

**Independent Test**: Can be fully tested by running `go test ./... -parallel 4` and verifying all tests pass without container name conflict errors.

**Acceptance Scenarios**:

1. **Given** I run tests in parallel, **When** each test package starts, **Then** it creates a PostgreSQL container with a unique random name like `agentdx-test-a1b2c3`
2. **Given** a test package creates a test container, **When** the test completes (success or failure), **Then** the container is automatically removed
3. **Given** tests are running in parallel, **When** two test packages need PostgreSQL, **Then** they each get their own container on different ports

---

### Edge Cases

- What happens when Docker daemon is not running?
  - System should fail with a clear error message: "Docker is not running. Please start Docker and try again."

- What happens when the specified port is already in use?
  - System should fail with a clear error: "Port 55432 is already in use. Try a different port with --pg-port."

- What happens when the container exists but is stopped?
  - System should restart the existing container rather than creating a new one.

- What happens during a test if the container fails to start?
  - Test should be skipped with `t.Skip("PostgreSQL container failed to start: ...")`

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST start a PostgreSQL Docker container if one is not running when `agentdx watch` is invoked
- **FR-002**: System MUST start a PostgreSQL Docker container if one is not running when `agentdx session start` is invoked
- **FR-003**: System MUST support `--pg-name` (alias `-n`) flag to specify custom container name
- **FR-004**: System MUST support `--pg-port` (alias `-p`) flag to specify custom port
- **FR-005**: System MUST persist container data using Docker volumes
- **FR-006**: System MUST NOT stop the PostgreSQL container when agentdx exits
- **FR-007**: Test helper MUST create containers with random names (format: `agentdx-test-{random}`)
- **FR-008**: Test helper MUST select random available ports for test containers
- **FR-009**: Test helper MUST clean up test containers after test completion
- **FR-010**: System MUST check if container already exists and reuse it
- **FR-011**: System MUST restart stopped containers instead of creating new ones

### Key Entities

- **PostgresContainer**: Represents a Docker container running PostgreSQL. Key attributes: name, port, volume name, running state
- **TestContainer**: Ephemeral PostgreSQL container for testing. Attributes: random name, random port, cleanup handler

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: `agentdx watch` succeeds on a fresh system with only Docker installed (no manual container setup)
- **SC-002**: `agentdx session start` succeeds on a fresh system with only Docker installed
- **SC-003**: Running `go test ./... -parallel 4` completes without container name conflict errors
- **SC-004**: Data persists across container restarts (index survives `docker restart agentdx-postgres`)
- **SC-005**: Custom port/name via flags takes precedence over defaults
