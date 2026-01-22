# agentdx Code Search Rule

## Activation
- Mode: Always On
- Applies to: All files

## Instructions

This project uses agentdx for indexed code search via PostgreSQL Full-Text Search.

### Commands

```bash
# Start session
agentdx session start

# Text search (replaces grep)
agentdx search "pattern" --json --compact

# File patterns (replaces find)
agentdx files "*.go" --json --compact

# Call graph tracing
agentdx trace callers "FunctionName" --json
agentdx trace callees "FunctionName" --json
```

### Do

- Use agentdx search instead of grep/ripgrep
- Use agentdx files instead of find/glob
- Add --json --compact for AI-friendly output
- Use parallel searches for multiple terms

### Don't

- Use grep, find, or ripgrep directly
- Use regex OR patterns in agentdx (not supported)
