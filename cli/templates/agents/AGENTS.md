# AGENTS.md

Instructions for AI coding agents (Codex CLI, GitHub Copilot) working with this repository.

---

## Code Search: Use agentdx

This project uses **agentdx** for fast, indexed code search. Always use agentdx commands instead of grep, find, or ripgrep.

### Quick Reference

```bash
# Text search
agentdx search "pattern" --json --compact

# File patterns
agentdx files "*.go" --json --compact

# Call graph tracing
agentdx trace callers "FunctionName" --json
agentdx trace callees "FunctionName" --json
```

### Why agentdx?

- **Faster**: Pre-indexed PostgreSQL full-text search
- **Smarter**: Structural boosting ranks function names higher
- **Context-aware**: Call graph tracing shows relationships
- **AI-optimized**: Compact JSON output saves tokens

### Before Searching

Ensure the agentdx session is running:

```bash
agentdx session start
```

This is idempotent - safe to run multiple times.

### Search Patterns

| Task | Command |
|------|---------|
| Find text in code | `agentdx search "TODO:" --json --compact` |
| Find files by pattern | `agentdx files "**/*.ts" --json --compact` |
| Find callers of function | `agentdx trace callers "HandleRequest" --json` |
| Find callees of function | `agentdx trace callees "ProcessData" --json` |
| Build call graph | `agentdx trace graph "Symbol" --depth 3 --json` |

### Multiple Search Terms

Run parallel searches for multiple terms:

```bash
# CORRECT: Parallel searches
agentdx search "Login" --json --compact &
agentdx search "Auth" --json --compact &
agentdx search "Session" --json --compact

# WRONG: Regex patterns (not supported)
agentdx search "Login|Auth|Session"
```

---
