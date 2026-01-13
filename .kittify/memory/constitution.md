<!--
Sync Impact Report:
- Version change: (initial) → 1.0.0
- Modified principles: N/A (initial creation)
- Added sections: All core principles, Testing Standards, Performance Requirements, Development Workflow
- Removed sections: N/A
- Templates requiring updates:
  - ✅ .kittify/templates/plan-template.md (reviewed for alignment)
  - ✅ .kittify/templates/spec-template.md (reviewed for alignment)
  - ✅ .kittify/templates/tasks-template.md (reviewed for alignment)
  - ✅ .kittify/templates/commands/*.md (reviewed for generic guidance)
- Follow-up TODOs: None
-->

# agentdx Constitution

## Core Principles

### I. Interface-First Architecture

Every major component MUST expose a well-defined interface. Interfaces define the contract between components, enabling:
- Swappable implementations (e.g., different embedder providers, storage backends)
- Independent testing through mocks
- Clear separation of concerns

**Rationale**: agentdx is designed for extensension. Users may want to use different embedding providers (Ollama, OpenAI, custom) or storage backends (GOB files, PostgreSQL, etc.). Interface-based design makes this possible without core changes.

### II. Semantic Search as Primary Discovery

Code exploration MUST prioritize semantic understanding over text matching:
- Use `agentdx search` for finding code by intent/purpose
- Reserve exact text matching (Grep/Glob) for literal searches only
- English queries yield better results due to embedding model training

**Rationale**: Vector-based semantic search understands code meaning, enabling developers to find functionality without knowing exact variable/function names.

### III. Code Quality (NON-NEGOTIABLE)

All code MUST meet these quality standards:
- Pass `make lint` with zero violations (golangci-lint)
- Pass `make test` with race detection enabled
- Follow Go conventions and effective Go practices
- Include godoc comments for exported types, functions, and constants

**Rationale**: Consistent code quality reduces bugs, improves maintainability, and ensures the codebase remains accessible to all contributors.

### IV. Testing Standards

Testing discipline is mandatory:
- Unit tests: All packages MUST have test coverage for core logic
- Integration tests: Required for cross-component interactions (embedder + store, file system operations)
- Race detection: `make test` includes `-race` flag; race conditions MUST be resolved
- Coverage: New features should not significantly reduce overall coverage percentage

**Rationale**: agentdx operates on user codebases. Bugs can corrupt indexes or produce incorrect results. Comprehensive testing is essential for reliability.

### V. User Experience Consistency

CLI behavior MUST be predictable and consistent:
- All commands follow: `command [flags] [args]` pattern
- Errors go to stderr, normal output to stdout
- Support both human-readable and JSON output formats where applicable
- Configuration via `.agentdx/config.yaml` in project root
- Respect gitignore and common ignore patterns automatically

**Rationale**: Developers use agentdx across many projects. Consistent behavior reduces cognitive load and makes the tool intuitive.

## Performance Requirements

### Indexing Performance

- Initial indexing SHOULD complete within reasonable time for typical projects (<5 min for 10k LOC)
- Incremental updates (file watcher) SHOULD complete within 1 second per changed file
- Memory usage MUST remain bounded; large projects should not require excessive RAM

### Search Performance

- Search queries SHOULD return results within 500ms for typical indexes
- Similarity search MUST use approximate nearest neighbor (ANN) algorithms for large indexes
- Result ranking MUST be deterministic for identical queries

### Resource Limits

- Chunking parameters MUST be configurable (size, overlap)
- Embedding requests MUST batch when possible to reduce API overhead
- File watching MUST debounce rapid successive changes

**Rationale**: agentdx is a developer tool that runs frequently during development. Performance directly impacts developer productivity.

## Development Workflow

### Commit Convention

Follow conventional commits: `type(scope): description`

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

Examples:
- `feat(embedder): add Cohere embedding provider`
- `fix(store): correct pgvector index creation`
- `chore(deps): upgrade dependencies`

### Branch Policy

- NEVER push directly to `main`
- All changes via feature branches: `<type>/<issue-number>-<short-description>`
- Pull required for all changes
- CI MUST pass before merge
- Code review required for non-trivial changes

### Quality Gates

Before committing:
1. `make lint` - zero linter violations
2. `make test` - all tests pass with race detection
3. Manual testing for CLI changes (verify help text, flags, error messages)

**Rationale**: These gates prevent broken code from entering the codebase and maintain code quality standards.

## Security & Privacy

### Local-First Default

- Local embeddings (Ollama) are preferred over cloud APIs
- When cloud APIs are used, clearly document data transmission
- Configuration files MUST NOT contain sensitive credentials

### File System Safety

- NEVER modify user code files
- Respect `.gitignore` and other ignore files
- Warn before operations that may consume significant resources (disk, network, API quotas)

## Governance

### Constitution Authority

This constitution supersedes all other practices in this project. In case of conflict, the constitution takes precedence.

### Amendment Procedure

1. Propose changes with clear rationale
2. Update this document with version bump according to semantic versioning:
   - MAJOR: Backward incompatible governance/principle removals
   - MINOR: New principles or material expansions
   - PATCH: Clarifications, wording fixes
3. All PRs MUST verify compliance with current constitution
4. Update dependent templates and documentation to reflect changes

### Compliance Review

- All features must align with core principles
- Complexity must be justified (YAGNI principles apply)
- Use `CLAUDE.md` for runtime development guidance

**Version**: 1.0.0 | **Ratified**: 2026-01-13 | **Last Amended**: 2026-01-13
