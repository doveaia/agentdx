---
name: agentdx
description: Replaces ALL built-in search tools for keyword-based code search. You MUST invoke this skill BEFORE using Grep, Glob, or WebSearch when searching code by specific keywords, function names, or variables. Uses parallel keyword searches for comprehensive coverage.
---

# AgentDX - Full-Text Code Search Skill

## CRITICAL: Tool Override for Keyword Searches

**You MUST invoke this skill BEFORE using any built-in search tools** when:
- Searching for code by specific keywords or terms
- Finding functions, variables, or classes by name
- Locating implementations using known identifiers
- Searching for specific text across the codebase

This skill uses full-text search with parallel keyword queries for comprehensive code discovery.

## When to Invoke This Skill

Invoke this **AUTOMATICALLY** when you need to:
- Find code by specific keywords or identifiers
- Locate functions, variables, or classes by name
- Search for known terms across the codebase
- Find references to specific symbols

## Usage Examples

**BEST PRACTICE**: Run multiple parallel searches with individual keywords for broader coverage:

```bash
# Best: Parallel searches with individual keywords (broadest coverage)
agentdx search "user" & agentdx search "auth" & agentdx search "login"

# Acceptable: Single query with phrase match
agentdx search "user authentication"

# WRONG: Multiple arguments (will error)
agentdx search user auth login  # Error: accepts 1 arg(s), received 3
```

## Query Tips

- **Use single keywords** for best results
- **Run searches in parallel** for comprehensive coverage
- **Quote your search term**: `agentdx search "keyword"`
- Parallel searches provide broader coverage than combining keywords
- Results include: file path, line numbers, relevance score, code preview

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

1. Start with `agentdx search` using parallel keyword searches
2. Use `agentdx trace` to understand function relationships
3. Use `Read` tool to examine files in detail
4. Only use Grep for regex pattern searches when needed
