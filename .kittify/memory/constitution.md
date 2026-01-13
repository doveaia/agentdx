<!--
Sync Impact Report
Version change: Initial -> 1.0.0
Modified principles: N/A (initial creation)
Added sections: All
Removed sections: N/A
Templates requiring updates:
  - .kittify/missions/software-dev/templates/plan-template.md (no constitution gates specified - using generic template)
  - .kittify/missions/software-dev/templates/spec-template.md (aligned with quality principles)
  - .kittify/missions/software-dev/templates/tasks-template.md (aligned with testing requirements)
Follow-up TODOs: None
-->

# agentdx Constitution

## Core Principles

### I. Code Quality

**Semantic Clarity**: Code must be self-documenting. Function and variable names must clearly describe their purpose. Complex logic requires inline comments explaining the "why," not the "what."

**Interface-First Design**: All core functionality is defined through interfaces (`embedder.Embedder`, `store.VectorStore`). Implementations must satisfy these contracts without breaking existing consumers.

**Error Handling**: Errors MUST be returned explicitly (not silenced) and include context. Use Go's error wrapping (`fmt.Errorf` with `%w`) to preserve error chains for debugging.

**Go Conventions**: Follow standard Go project layout, effective Go idioms, and use `gofumpt` for formatting. Run `make lint` before committing.

### II. Testing Standards (NON-NEGOTIABLE)

**Test Coverage**: All new code requires corresponding tests. Aim for >80% coverage. Critical paths (indexing, search, embedder interfaces) require 100% coverage.

**Test Types**:
- **Unit tests**: Test individual functions and methods in isolation
- **Integration tests**: Test interactions between components (e.g., indexer + embedder + store)
- **Contract tests**: Verify interface implementations satisfy expected behavior

**Test First**: For bug fixes, write a failing test reproducing the bug first, then fix it.

**Mock External Dependencies**: Embedder providers (Ollama, OpenAI) must be mockable for testing. Use interfaces to enable test doubles.

### III. User Experience Consistency

**CLI Interface**: All commands follow consistent patterns:
- Flags use `kebab-case`
- Help text is clear and concise
- Errors go to stderr, normal output to stdout
- JSON output mode available for all list/query commands

**Privacy First**: agentdx is designed for local-first operation. Default configurations must prioritize user privacy (local Ollama vs cloud OpenAI).

**AI Agent Integration**: All search/trace commands support `--json` and `--compact` flags for optimal AI agent consumption.

### IV. Performance Standards

**Indexing Performance**: Initial indexing must handle 100k LOC repositories within 5 minutes on typical hardware (M1/M2 Mac, modern x86_64 Linux).

**Search Latency**: Semantic search queries must return results within 500ms for indices up to 100k chunks.

**Memory Efficiency**: The indexer must process files in streams, not load entire repositories into memory. Chunking operations must be incremental.

**Watcher Responsiveness**: File system changes must be reflected in the index within 2 seconds of detection.

## Performance Benchmarks

**Repository Size Targets**:
- Small: <10k LOC - <30s initial index, <100ms search
- Medium: 10k-100k LOC - <5min initial index, <500ms search
- Large: 100k-1M LOC - <30min initial index, <2s search

**Metrics Collection**: Critical paths include timing metrics for indexing and search operations. Use these to detect regressions.

## Development Workflow

**Branch Protection**: Never push directly to `main`. All work happens on feature branches with descriptive names: `feat/42-add-new-embedder`, `fix/123-index-out-of-bounds`.

**Conventional Commits**: Follow `type(scope): description` format:
- `feat`: New feature
- `fix`: Bug fix
- `refactor`: Code restructuring without behavior change
- `test`: Adding or updating tests
- `docs`: Documentation changes
- `chore`: Maintenance tasks

**Code Review**: All PRs require:
- Passing CI (lint, test, build)
- At least one approving review for non-trivial changes
- No merge conflicts with main

## Security & Privacy

**No Telemetry by Default**: agentdx must not send user code or usage data to external services without explicit opt-in.

**API Key Safety**: API keys (OpenAI, etc.) must never be logged or included in error messages. Use environment variables or secure config storage.

**Input Validation**: All user inputs (file paths, queries, configs) must be validated and sanitized to prevent path traversal or injection attacks.

## Governance

**Constitution Supersedes**: This constitution takes precedence over personal preferences. When in doubt, principles here override "we usually do it this way."

**Amendment Process**: Changes require:
1. Proposal in issue with rationale
2. Team discussion and consensus
3. Version bump (MAJOR for breaking changes, MINOR for additions)
4. Update to this file and all dependent templates

**Compliance**: All PRs should reference applicable constitution principles in their description. Reviewers may reject changes violating core principles.

**Runtime Guidance**: See `CLAUDE.md` for agent-specific runtime development guidance.

**Version**: 1.0.0 | **Ratified**: 2026-01-13 | **Last Amended**: 2026-01-13
