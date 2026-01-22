# Work Packages: PostgreSQL Container Lifecycle Management

**Inputs**: Design documents from `/kitty-specs/008-postgres-container-lifecycle/`
**Prerequisites**: plan.md, spec.md

**Organization**: Fine-grained subtasks (`Txxx`) roll up into work packages (`WPxx`). Each work package is independently deliverable and testable.

---

## Work Package WP01: Container Options & Volume Support (Priority: P0)

**Goal**: Create ContainerOptions type and add Docker volume support.
**Independent Test**: Unit tests pass for option merging and volume creation args.
**Prompt**: `/tasks/WP01-container-options-volume.md`

### Included Subtasks
- [ ] T001 Create `localsetup/options.go` with ContainerOptions type
- [ ] T002 Modify `localsetup/docker.go` to add volume support in CreateContainer
- [ ] T003 Update `localsetup/interfaces.go` with Volume field in ContainerConfig
- [ ] T004 Write unit tests for ContainerOptions and volume args

### Implementation Notes
- ContainerOptions: Name (string), Port (int) with defaults
- Volume naming convention: `{container_name}-data`
- Docker args: `-v {volume_name}:/var/lib/postgresql/data`

### Dependencies
- None (foundation work package)

---

## Work Package WP02: EnsurePostgresRunning Function (Priority: P0)

**Goal**: Implement the core function that checks/starts PostgreSQL container.
**Independent Test**: Function correctly starts container when needed, reuses when running.
**Prompt**: `/tasks/WP02-ensure-postgres-running.md`

### Included Subtasks
- [ ] T005 Create `localsetup/ensure.go` with EnsurePostgresRunning function
- [ ] T006 Update `localsetup/database.go` to accept host/port parameters
- [ ] T007 Add clear error messages for Docker not available, port in use
- [ ] T008 Write integration test for EnsurePostgresRunning

### Implementation Notes
- Check Docker available → check container exists → start if stopped → create if missing → wait for ready
- Return DSN for successful connection

### Dependencies
- Depends on WP01 (ContainerOptions, volume support)

---

## Work Package WP03: CLI Flags for watch Command (Priority: P1) 🎯 MVP

**Goal**: Add `--pg-name`/`-n` and `--pg-port`/`-p` flags to `agentdx watch`.
**Independent Test**: `agentdx watch --pg-name test --pg-port 5433` starts container with custom settings.
**Prompt**: `/tasks/WP03-cli-watch-flags.md`

### Included Subtasks
- [ ] T009 Add --pg-name and --pg-port flags to watchCmd
- [ ] T010 Call EnsurePostgresRunning before connecting to store
- [ ] T011 Update watch command help text with flag descriptions
- [ ] T012 Write integration test for watch with custom flags

### Implementation Notes
- Flags should use cobra persistent flags with aliases
- Merge flags with config values (flags take precedence)

### Dependencies
- Depends on WP02 (EnsurePostgresRunning)

---

## Work Package WP04: CLI Flags for session start Command (Priority: P1)

**Goal**: Add `--pg-name`/`-n` and `--pg-port`/`-p` flags to `agentdx session start`.
**Independent Test**: `agentdx session start --pg-name test --pg-port 5433` starts container correctly.
**Prompt**: `/tasks/WP04-cli-session-flags.md`

### Included Subtasks
- [ ] T013 Add --pg-name and --pg-port flags to sessionStartCmd
- [ ] T014 [P] Propagate flags to daemon process via environment or args
- [ ] T015 Update session start help text
- [ ] T016 Write integration test for session start with custom flags

### Implementation Notes
- Session daemon needs to know the container settings
- Pass via environment variables (AGENTDX_PG_NAME, AGENTDX_PG_PORT)

### Parallel Opportunities
- T013 and T015 can proceed in parallel

### Dependencies
- Depends on WP02 (EnsurePostgresRunning)
- Can proceed in parallel with WP03

---

## Work Package WP05: Config File Support (Priority: P2)

**Goal**: Add container_name and port to config.yaml for persistent settings.
**Independent Test**: Config loads container settings, flags override config values.
**Prompt**: `/tasks/WP05-config-support.md`

### Included Subtasks
- [ ] T017 Add ContainerName, Port fields to PostgresConfig in config.go
- [ ] T018 Update DefaultConfig() with default container settings
- [ ] T019 Implement option merging: flags > config > defaults
- [ ] T020 Write test for config loading with container settings

### Implementation Notes
```yaml
index:
  store:
    postgres:
      dsn: "..."
      container_name: "agentdx-postgres"  # optional
      port: 55432  # optional
```

### Dependencies
- Depends on WP01 (ContainerOptions)

---

## Work Package WP06: Test Container Helper (Priority: P2)

**Goal**: Create helper for tests to use isolated PostgreSQL containers.
**Independent Test**: Parallel tests run without container name conflicts.
**Prompt**: `/tasks/WP06-test-container-helper.md`

### Included Subtasks
- [ ] T021 Create `localsetup/testcontainer.go` with NewTestContainer function
- [ ] T022 Implement random name generation (agentdx-test-{8 hex chars})
- [ ] T023 Implement random port selection (let OS assign)
- [ ] T024 Implement cleanup via t.Cleanup()
- [ ] T025 Write tests for test container helper
- [ ] T026 Migrate existing tests to use TestContainer

### Implementation Notes
```go
func NewTestContainer(t testing.TB) *TestContainer {
    // Generate random name and port
    // Create container
    // Register cleanup
    // Wait for ready
    // Return with DSN
}
```

### Dependencies
- Depends on WP02 (EnsurePostgresRunning)

---

## Work Package WP07: Polish & Documentation (Priority: P3)

**Goal**: Final cleanup, documentation, and validation.
**Independent Test**: All tests pass, help text is accurate.
**Prompt**: `/tasks/WP07-polish-documentation.md`

### Included Subtasks
- [ ] T027 Update README with new flags documentation
- [ ] T028 Add troubleshooting section for Docker issues
- [ ] T029 Run full test suite with parallel execution
- [ ] T030 Verify backward compatibility with existing containers

### Dependencies
- Depends on WP03, WP04, WP05, WP06

---

## Dependency & Execution Summary

```
WP01 (Container Options) ──┬──> WP02 (EnsurePostgresRunning) ──┬──> WP03 (watch flags) ──┐
                           │                                    │                         │
                           └──> WP05 (Config)                   └──> WP04 (session flags) ├──> WP07 (Polish)
                                                                │                         │
                                                                └──> WP06 (Test Helper) ──┘
```

- **Sequential**: WP01 → WP02 (must complete before parallel streams)
- **Parallel streams after WP02**: WP03, WP04, WP05, WP06 can proceed in parallel
- **MVP Scope**: WP01 + WP02 + WP03 (watch command working)

---

## Subtask Index (Reference)

| Subtask ID | Summary | Work Package | Priority | Parallel? |
|------------|---------|--------------|----------|-----------|
| T001 | Create ContainerOptions type | WP01 | P0 | No |
| T002 | Add volume support to CreateContainer | WP01 | P0 | No |
| T003 | Update ContainerConfig interface | WP01 | P0 | No |
| T004 | Unit tests for options/volume | WP01 | P0 | No |
| T005 | Create EnsurePostgresRunning | WP02 | P0 | No |
| T006 | Update database.go with params | WP02 | P0 | No |
| T007 | Add clear error messages | WP02 | P0 | No |
| T008 | Integration test for ensure | WP02 | P0 | No |
| T009 | Add flags to watchCmd | WP03 | P1 | No |
| T010 | Call EnsurePostgresRunning in watch | WP03 | P1 | No |
| T011 | Update watch help text | WP03 | P1 | Yes |
| T012 | Integration test for watch flags | WP03 | P1 | No |
| T013 | Add flags to sessionStartCmd | WP04 | P1 | No |
| T014 | Propagate flags to daemon | WP04 | P1 | Yes |
| T015 | Update session start help | WP04 | P1 | Yes |
| T016 | Integration test for session | WP04 | P1 | No |
| T017 | Add config fields | WP05 | P2 | No |
| T018 | Update DefaultConfig | WP05 | P2 | No |
| T019 | Implement option merging | WP05 | P2 | No |
| T020 | Test config loading | WP05 | P2 | No |
| T021 | Create testcontainer.go | WP06 | P2 | No |
| T022 | Random name generation | WP06 | P2 | Yes |
| T023 | Random port selection | WP06 | P2 | Yes |
| T024 | Cleanup via t.Cleanup | WP06 | P2 | No |
| T025 | Test container tests | WP06 | P2 | No |
| T026 | Migrate existing tests | WP06 | P2 | No |
| T027 | Update README | WP07 | P3 | Yes |
| T028 | Add troubleshooting docs | WP07 | P3 | Yes |
| T029 | Full test suite run | WP07 | P3 | No |
| T030 | Verify backward compat | WP07 | P3 | No |
