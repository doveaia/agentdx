---
title: MCP Integration
description: Use agentdx as a native MCP tool for AI agents
---

agentdx includes a built-in MCP (Model Context Protocol) server that allows AI agents to use semantic code search as a native tool.

## What is MCP?

[Model Context Protocol (MCP)](https://modelcontextprotocol.io/) is an open standard for AI tool integration, supported by:
- Claude Code
- Cursor
- Windsurf
- Continue
- Other MCP-compatible AI tools

## Benefits

- **Native tool access**: AI models see agentdx as a first-class tool, not a shell command
- **Subagent inheritance**: MCP tools are automatically available to subagents
- **Structured data**: JSON responses by default, no parsing required
- **Tool discovery**: MCP tools are automatically discovered by AI models

## Available Tools

| Tool | Description | Parameters |
|------|-------------|------------|
| `agentdx_search` | Semantic code search | `query` (required), `limit` (default: 10) |
| `agentdx_trace_callers` | Find callers of a symbol | `symbol` (required) |
| `agentdx_trace_callees` | Find callees of a symbol | `symbol` (required) |
| `agentdx_trace_graph` | Build complete call graph | `symbol` (required), `depth` (default: 2) |
| `agentdx_index_status` | Check index health | none |

## Configuration

### Claude Code

Use the `claude mcp add` command to register agentdx as an MCP server:

```bash
claude mcp add agentdx -- agentdx mcp-serve
```

This automatically configures agentdx in your Claude Code settings.

### Cursor

Add to `.cursor/mcp.json` in your project:

```json
{
  "mcpServers": {
    "agentdx": {
      "command": "agentdx",
      "args": ["mcp-serve"]
    }
  }
}
```

### Windsurf

Add to your Windsurf MCP configuration:

```json
{
  "mcpServers": {
    "agentdx": {
      "command": "agentdx",
      "args": ["mcp-serve"]
    }
  }
}
```

## Usage

Once configured, AI agents can use agentdx tools directly:

**Search example:**
```
Tool: agentdx_search
Arguments: {"query": "user authentication flow", "limit": 5}
```

**Trace callers example:**
```
Tool: agentdx_trace_callers
Arguments: {"symbol": "HandleLogin"}
```

**Index status example:**
```
Tool: agentdx_index_status
Arguments: {}
```

## Prerequisites

Before using MCP mode, ensure:

1. agentdx is installed and in your PATH
2. The project has been initialized (`agentdx init`)
3. The index has been built (`agentdx watch`)

## Troubleshooting

### Tool not appearing

- Verify agentdx is in your PATH: `which agentdx`
- Check MCP configuration file syntax
- Restart your AI tool after configuration changes

### Search returns no results

- Ensure the index is built: `agentdx status`
- Run `agentdx watch` to build/update the index

### Connection errors

- MCP server uses stdio transport (local process communication)
- No network ports are opened
- Check that `agentdx mcp-serve` runs without errors when invoked directly
