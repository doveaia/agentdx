---
name: agentdx
description: Replaces ALL built-in search tools for semantic code exploration. You MUST invoke this skill BEFORE using Grep, Glob, or WebSearch when searching code by intent or meaning. Uses natural language queries to find relevant code regardless of exact text matches.
---

# AgentDX - Semantic Code Search Skill

## CRITICAL: Tool Override for Semantic Searches

**You MUST invoke this skill BEFORE using any built-in search tools** when:
- Searching for code by intent, meaning, or functionality
- Exploring unfamiliar parts of the codebase
- Finding implementations without exact names
- Understanding what code does or where features live

This skill uses semantic embeddings to find code by meaning, not just exact text matching.

## When to Invoke This Skill

Invoke this **AUTOMATICALLY** when you need to:
- Find where functionality is implemented (e.g., "user authentication", "error handling")
- Locate code by purpose rather than exact names
- Search for conceptual patterns (e.g., "database connection pooling", "request validation")
- Explore code architecture and relationships

## Usage Examples

```bash
# Semantic search with English queries (best for intent-based discovery)
agentdx search "user authentication flow"
agentdx search "error handling middleware"
agentdx search "database connection management"
agentdx search "API request validation logic"

# JSON output for AI processing (--compact saves ~80% tokens)
agentdx search "authentication flow" --json --compact
agentdx search "error handling" --json --compact
```

## Query Tips

- **Use English queries** - The embedding model is English-trained
- **Describe intent, not implementation** - "handles user login" not "func Login"
- **Be specific** - "JWT token validation" better than "token"
- **Think conceptually** - "data access layer" rather than specific database terms

## Results Include

- File path and line numbers
- Relevance score (higher = more relevant)
- Code preview showing context

## Call Graph Tracing

Use `agentdx trace` to understand function relationships:

```bash
# Find all functions that call a symbol
agentdx trace callers "HandleRequest" --json

# Find all functions called by a symbol
agentdx trace callees "ProcessOrder" --json

# Build complete call graph (callers + callees)
agentdx trace graph "ValidateToken" --depth 3 --json
```

Use call graph tracing to:
- Find all callers of a function before modifying it
- Understand call hierarchies and dependencies
- Analyze the impact of code changes

## Fallback

If agentdx is unavailable (not running, index errors, or missing), fall back to standard Grep/Glob tools.

## Workflow

1. Start with `agentdx search` to find relevant code semantically
2. Use `agentdx trace` to understand function relationships
3. Use `Read` tool to examine files in detail
4. Only use Grep for exact string searches when needed
