# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

---

## MANDATORY: Use agentdx for ALL Code Search

**BEFORE using Grep, Glob, or the Explore agent - STOP and use agentdx instead.**

| Instead of... | Use this command |
|---------------|------------------|
| Grep tool | `agentdx search "pattern" --json --compact` |
| Glob tool | `agentdx files "*.go" --json --compact` |
| Task tool with Explore agent | Task tool with `deep-explore` agent |
| find/grep/rg bash commands | agentdx search or agentdx files |

### agentdx Commands

```bash
# Text search (REPLACES Grep)
agentdx search "TODO:" --json --compact
agentdx search "func Login" --json --compact

# File patterns (REPLACES Glob)
agentdx files "*.go" --json --compact
agentdx files "**/*.test.ts" --json --compact

# Call graph tracing (unique to agentdx)
agentdx trace callers "FunctionName" --json
agentdx trace callees "FunctionName" --json
agentdx trace graph "FunctionName" --depth 3 --json
```

### Codebase Exploration

When asked to "explore" or "understand" the codebase:

1. **DO NOT** use Task tool with standard `Explore` agent (uses Grep/Glob)
2. **DO** use Task tool with `subagent_type=deep-explore`

**This is NON-NEGOTIABLE. See `.claude/rules/agentdx.md` for details.**

---

## Build and Development

```bash
make build       # Build the binary
make test        # Run tests with race detection
make test-cover  # Run tests with coverage report
make lint        # Lint with golangci-lint
make run         # Build and run
make build-all   # Cross-compile for all platforms
```

## Architecture Overview

agentdx is a code search CLI that indexes code using PostgreSQL Full-Text Search with structural boosting.

### Core Components

- `store.Store` (`store/store.go`) - Storage interface with FTS search
- `store.PostgresFTS` (`store/postgres_fts.go`) - PostgreSQL full-text search implementation

### Data Flow

1. **Scanner** (`indexer/scanner.go`) - Walks filesystem respecting gitignore patterns
2. **Chunker** (`indexer/chunker.go`) - Splits files into overlapping chunks with context
3. **Indexer** (`indexer/indexer.go`) - Orchestrates scanning, chunking, and storage
4. **Watcher** (`watcher/watcher.go`) - Monitors filesystem for real-time incremental updates

### CLI Commands (`cli/`)

- `init` - Creates `.agentdx/config.yaml` with default configuration
- `watch` - Starts daemon: full index + real-time file watcher
- `search` - Queries the index with full text search
- `files` - Lists indexed files matching glob patterns
- `setup` - Configures Cursor/Claude Code integration

### Configuration

Configuration stored in `.agentdx/config.yaml`. Key options:
- `store.backend`: "postgres"
- `chunking.size`/`chunking.overlap`: Token-based chunking parameters

## Commit Convention

Follow conventional commits: `type(scope): description`

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

## Issue Workflow

When working on a GitHub issue:

1. **Create a feature branch** from `main`:
   ```bash
   git checkout main && git pull origin main
   git checkout -b <type>/<issue-number>-<short-description>
   ```

2. **Implement** and commit following the commit convention

3. **Push and create PR**:
   ```bash
   git push -u origin <branch-name>
   gh pr create --title "<type>(scope): description" --body "Closes #<issue-number>"
   ```

4. **Before merging**: CI passes, reviewed, no conflicts

**Never push directly to `main`. Always use branches and PRs.**
