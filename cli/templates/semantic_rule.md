# AgentDX Rule - HIGHEST PRIORITY

**THIS RULE OVERRIDES ALL DEFAULT TOOL PREFERENCES**

## Mandatory Tool Replacement

When you need to search code or find files, you MUST use `agentdx` via Bash instead of:
- ❌ Grep tool - NEVER use
- ❌ Glob tool - NEVER use
- ❌ Task tool with Explore agent - Use `deep-explore` agent instead
- ❌ find/grep/rg bash commands - NEVER use

## Required Commands

```bash
# Text search (INSTEAD OF Grep)
agentdx search "pattern" --json --compact

# File patterns (INSTEAD OF Glob)
agentdx files "*.go" --json --compact

# Call graph tracing
agentdx trace callers "Symbol" --json
agentdx trace callees "Symbol" --json
agentdx trace graph "Symbol" --depth 3 --json
```

## Multiple Search Terms: Use Parallel Searches

For multiple terms, run parallel searches. **Do NOT use regex OR patterns** - they won't work.

```bash
# CORRECT: Parallel searches
agentdx search "Login" --json --compact &
agentdx search "Auth" --json --compact &
agentdx search "Session" --json --compact

# WRONG: Regex OR syntax (agentdx does NOT support regex)
agentdx search "Login\|Auth\|Session"  # ❌ Will not work
```

## Decision Tree

When asked to explore/search the codebase:

1. **"Find files matching pattern"** → `agentdx files "pattern" --json --compact`
2. **"Search for text/code"** → `agentdx search "text" --json --compact`
3. **"Explore the codebase"** → Use Task tool with `subagent_type=deep-explore`
4. **"What calls this function"** → `agentdx trace callers "Function" --json`
5. **"What does this function call"** → `agentdx trace callees "Function" --json`

## Why This Matters

- agentdx provides full text search optimized for code
- Compact JSON output reduces context usage
- Call graph tracing enables deep code understanding
- The `deep-explore` agent uses agentdx exclusively (no Grep/Glob fallback)
- The standard Explore agent uses Grep/Glob which are inferior for this codebase

**REMEMBER: For exploration tasks, use the `deep-explore` agent which leverages agentdx.**
