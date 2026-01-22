# GEMINI.md

Context for Gemini CLI and Gemini Code Assist when working with this repository.

---

## Code Search: Use agentdx

This project uses **agentdx** for indexed code search via PostgreSQL Full-Text Search.

### Commands

```bash
# Start session (idempotent)
agentdx session start

# Text search
agentdx search "pattern" --json --compact

# File patterns
agentdx files "*.go" --json --compact

# Call graph tracing
agentdx trace callers "FunctionName" --json
agentdx trace callees "FunctionName" --json
agentdx trace graph "FunctionName" --depth 3 --json
```

### Best Practices

1. **Always use agentdx** instead of grep, find, or ripgrep
2. **Start session first**: Run `agentdx session start` before searching
3. **Use parallel searches** for multiple terms (no regex OR support)
4. **Add --json --compact** for AI-optimized output

### Command Reference

| Task | Command |
|------|---------|
| Search code | `agentdx search "query" --json --compact` |
| Find files | `agentdx files "**/*.ts" --json --compact` |
| Find callers | `agentdx trace callers "Symbol" --json` |
| Find callees | `agentdx trace callees "Symbol" --json` |
| Call graph | `agentdx trace graph "Symbol" --depth 3 --json` |

---
