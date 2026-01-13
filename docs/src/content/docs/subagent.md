---
title: Claude Code Subagent
description: Use agentdx as a specialized exploration agent in Claude Code
---

Claude Code uses subagents for specialized tasks. agentdx provides a ready-to-use exploration subagent that leverages semantic search and call graph tracing.

## Why a Subagent?

When Claude Code spawns an "Explore" subagent:

- The subagent operates in an isolated context
- It doesn't inherit CLAUDE.md instructions
- It uses standard Grep/Glob instead of semantic search

The agentdx deep-explore subagent solves this by providing direct access to agentdx tools.

## Installation

```bash
agentdx agent-setup --with-subagent
```

This creates `.claude/agents/deep-explore.md` in your project.

## What It Does

The deep-explore subagent provides:

| Capability | Tool | Description |
|------------|------|-------------|
| Semantic Search | `agentdx search` | Find code by intent, not just text |
| Call Graph | `agentdx trace` | Understand function relationships |
| Standard Tools | Grep, Glob, Read | Available as fallback |

## Usage

Claude Code automatically selects the deep-explore agent when:

- Exploring unfamiliar code areas
- Understanding code architecture
- Finding implementations by intent
- Analyzing function relationships

You can also explicitly request it:

> "Use the deep-explore agent to understand the authentication flow"

## Subagent vs MCP

| Feature | Subagent | MCP |
|---------|----------|-----|
| Setup | `--with-subagent` flag | Separate MCP config |
| Access | Bash commands | Native tools |
| Context | Isolated | Shared |
| Use case | Exploration tasks | All tasks |

Both approaches complement each other. MCP provides native tool access, while the subagent ensures exploration tasks use agentdx.

## Manual Setup

If you prefer manual setup, create `.claude/agents/deep-explore.md`:

```yaml
---
name: deep-explore
description: Deep codebase exploration using agentdx semantic search and call graph tracing.
tools: Read, Grep, Glob, Bash
model: inherit
---

## Instructions

You are a specialized code exploration agent with access to agentdx.

### Primary Tools

Use `agentdx search` for semantic code search:
- agentdx search "authentication flow"
- agentdx search "error handling"

Use `agentdx trace` for call graph analysis:
- agentdx trace callers "Login"
- agentdx trace callees "HandleRequest"
- agentdx trace graph "ProcessOrder" --depth 3

### Workflow

1. Start with agentdx search to find relevant code
2. Use agentdx trace to understand function relationships
3. Use Read to examine files in detail
4. Synthesize findings into a clear summary
```

## Troubleshooting

### Subagent not appearing

- Verify the file exists: `cat .claude/agents/deep-explore.md`
- Restart Claude Code after creating the file
- Ensure the YAML frontmatter is valid

### agentdx commands failing in subagent

- Ensure agentdx is in your PATH
- Verify the index is built: `agentdx status`
- Run `agentdx watch` to build/update the index
