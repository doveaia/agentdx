# GitHub Copilot Instructions

## Code Search: Use agentdx

This repository uses **agentdx** for indexed code search. Always prefer agentdx over grep, find, or ripgrep.

### Getting Started

```bash
# Start the agentdx session (idempotent)
agentdx session start
```

### Search Commands

```bash
# Text search (replaces grep)
agentdx search "pattern" --json --compact

# File patterns (replaces find)
agentdx files "**/*.ts" --json --compact

# Call graph tracing
agentdx trace callers "FunctionName" --json
agentdx trace callees "FunctionName" --json
agentdx trace graph "Symbol" --depth 3 --json
```

### Best Practices

1. Always use agentdx commands for code search
2. Add `--json --compact` for cleaner output
3. Use parallel searches for multiple terms (no regex OR support)
4. Run `agentdx session start` before first search

### Why agentdx?

- Pre-indexed PostgreSQL full-text search
- Structural boosting for code identifiers
- Call graph tracing for relationships
- AI-optimized compact JSON output
