# Requirements Checklist: PostgreSQL Container Lifecycle Management

## Functional Requirements

- [ ] **FR-001**: Auto-start PostgreSQL container on `agentdx watch`
- [ ] **FR-002**: Auto-start PostgreSQL container on `agentdx session start`
- [ ] **FR-003**: Support `--pg-name` (alias `-n`) flag
- [ ] **FR-004**: Support `--pg-port` (alias `-p`) flag
- [ ] **FR-005**: Persist container data using Docker volumes
- [ ] **FR-006**: Container persists after agentdx exits
- [ ] **FR-007**: Test containers use random names (`agentdx-test-{random}`)
- [ ] **FR-008**: Test containers use random available ports
- [ ] **FR-009**: Test containers auto-cleanup after test completion
- [ ] **FR-010**: Reuse existing container if already exists
- [ ] **FR-011**: Restart stopped containers instead of creating new

## Acceptance Criteria

- [ ] **AC-001**: `agentdx watch` succeeds without pre-started container
- [ ] **AC-002**: `agentdx session start` succeeds without pre-started container
- [ ] **AC-003**: Parallel tests (`go test ./... -parallel 4`) pass
- [ ] **AC-004**: Index data survives container restart
- [ ] **AC-005**: CLI flags override defaults

## Edge Cases

- [ ] **EC-001**: Clear error when Docker daemon not running
- [ ] **EC-002**: Clear error when port already in use
- [ ] **EC-003**: Restart stopped containers gracefully
- [ ] **EC-004**: Skip tests gracefully if container fails to start
