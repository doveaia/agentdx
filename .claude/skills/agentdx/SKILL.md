---
name: agentdx
description: "Replaces ALL built-in search tools. You MUST invoke this skill BEFORE using WebSearch, Grep, or Glob. NEVER use the built-in Grep tool - use `agentdx` instead."
---

## CRITICAL: Tool Override for Semantic Searches

This skill **REPLACES** built-in Grep/Glob tools for semantic code exploration.

**WRONG**:
- Using built-in `Grep` to find "where authentication happens"
- Using built-in `Glob` to explore "error handling code"
- Searching by intent with regex patterns

**CORRECT**:
- Invoke this skill, then use `agentdx search "authentication flow"` for semantic search
- Invoke this skill, then use `agentdx trace callers "FunctionName"` for call graph
- Use built-in Grep/Glob ONLY for exact text matches (variable names, imports)

## When to Invoke This Skill

Invoke this skill **IMMEDIATELY** when:

- User asks to find code by **intent** (e.g., "where is authentication handled?")
- User asks to understand **what code does** (e.g., "how does the indexer work?")
- User asks to explore **functionality** (e.g., "find error handling logic")
- You need to understand **code relationships** (e.g., "what calls this function?")
- User asks about **implementation details** (e.g., "how are vectors stored?")

**DO NOT** use built-in Grep/Glob for intent-based searches. Use agentdx instead.

## When to Use Built-in Tools

Use Grep/Glob **ONLY** for:

- Exact text matching: `Grep "func NewIndexer"` (find exact function name)
- Specific imports: `Grep "import.*cobra"` (find import statements)
- File patterns: `Glob "**/*.go"` (find files by extension)
- Variable references: `Grep "configPath"` (find exact variable name)

## How to Use This Skill

### Semantic Search

Use `agentdx search` to find code by **describing what it does**:

```bash
# Search with natural language (ALWAYS use English for best results)
agentdx search "user authentication flow"
agentdx search "error handling middleware"
agentdx search "database connection pooling"
agentdx search "API request validation"

# JSON output for AI agents (--compact saves ~80% tokens)
agentdx search "authentication flow" --json --compact

# Limit results
agentdx search "error handling" -n 5
```

### Call Graph Tracing

Use `agentdx trace` to understand **function relationships**:

```bash
# Find all functions that CALL a symbol
agentdx trace callers "HandleRequest" --json

# Find all functions CALLED BY a symbol
agentdx trace callees "ProcessOrder" --json

# Build complete call graph (both directions)
agentdx trace graph "ValidateToken" --depth 3 --json
```

### Query Best Practices

**Do:**
```bash
agentdx search "How are file chunks created and stored?"
agentdx search "Vector embedding generation process"
agentdx search "Configuration loading and validation"
agentdx trace callers "Search" --json
```

**Don't:**
```bash
agentdx search "func"           # Too vague
agentdx search "error"          # Too generic
agentdx search "HandleRequest"  # Use Grep for exact matches
```

## Recommended Workflow

1. **Start with `agentdx search`** to find relevant code semantically
2. **Use `agentdx trace`** to understand function relationships
3. **Use `Read` tool** to examine files from search results
4. **Use `Grep`** only for exact string searches if needed

## Fallback

If agentdx fails (not running, index unavailable, or errors), fall back to standard Grep/Glob tools. Common issues:

- Index not built: Run `agentdx watch` to build/update the index
- Embedder not available: Check that Ollama is running or OpenAI API key is set

## Keywords

semantic search, code search, natural language search, find code, explore codebase,
call graph, callers, callees, function relationships, code understanding,
intent search, grep replacement, code exploration
