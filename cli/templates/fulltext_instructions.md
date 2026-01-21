---
## 🚨 STOP - READ THIS FIRST 🚨

### MANDATORY: Use agentdx for ALL Code Search & Exploration

**BEFORE using Grep, Glob, or the Explore agent - STOP and use agentdx instead.**

| Instead of... | Use this command |
|---------------|------------------|
| Grep tool | `agentdx search "pattern" --json --compact` |
| Glob tool | `agentdx files "*.go" --json --compact` |
| Task tool with Explore agent | Task tool with `deep-explore` agent |
| find/grep/rg bash commands | agentdx search or agentdx files |

### Quick Reference

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

### Codebase Exploration Workflow

When asked to "explore" or "understand" the codebase:

1. **DO NOT** use Task tool with standard `Explore` agent (uses Grep/Glob)
2. **DO** use Task tool with `subagent_type=deep-explore`:
   - The `deep-explore` agent uses agentdx exclusively
   - It provides semantic search, file patterns, and call graph tracing
   - No fallback to Grep/Glob - agentdx is the only search method

**This is NON-NEGOTIABLE. See `.claude/rules/agentdx.md` for details.**

---