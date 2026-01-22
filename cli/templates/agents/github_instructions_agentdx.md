# agentdx Instructions

Use agentdx for all code search operations in this repository.

## Commands

```bash
agentdx session start                    # Start daemon
agentdx search "pattern" --json --compact  # Text search
agentdx files "*.go" --json --compact      # File patterns
agentdx trace callers "Symbol" --json      # Find callers
agentdx trace callees "Symbol" --json      # Find callees
```

## Rules

- Replace grep with `agentdx search`
- Replace find with `agentdx files`
- Always add `--json --compact`
- Use parallel searches for multiple terms
