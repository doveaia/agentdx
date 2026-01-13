# agentdx

[![Go](https://github.com/yoanbernabeu/agentdx/actions/workflows/ci.yml/badge.svg)](https://github.com/yoanbernabeu/agentdx/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/yoanbernabeu/agentdx)](https://goreportcard.com/report/github.com/yoanbernabeu/agentdx)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**A privacy-first, CLI-native way to semantically search your codebase.**

Search code by *what it does*, not just what it's called. `agentdx` indexes the meaning of your code using vector embeddings, enabling natural language queries that find conceptually related code—even when naming conventions vary.

## Why agentdx?

`grep` was built in 1973 for exact text matching. Modern codebases need semantic understanding.

|                      | `grep` / `ripgrep`           | `agentdx`                          |
|----------------------|------------------------------|-----------------------------------|
| **Search type**      | Exact text / regex           | Semantic understanding            |
| **Query**            | `"func.*Login"`              | `"user authentication flow"`      |
| **Finds**            | Exact pattern matches        | Conceptually related code         |
| **AI Agent context** | Requires many searches       | Fewer, more relevant results      |

### Built for AI Agents

agentdx is designed to provide **high-quality context** to AI coding assistants. By returning semantically relevant code chunks, your agents spend less time searching and more time coding.

## Getting Started

### Installation

```bash
curl -sSL https://raw.githubusercontent.com/yoanbernabeu/agentdx/main/install.sh | sh
```

Or download from [Releases](https://github.com/yoanbernabeu/agentdx/releases).

### Quick Start

```bash
agentdx init                        # Initialize in your project
agentdx watch                       # Start background indexing daemon
agentdx search "error handling"     # Search semantically
agentdx trace callers "Login"       # Find who calls a function
```

## Commands

| Command                  | Description                            |
|--------------------------|----------------------------------------|
| `agentdx init`            | Initialize agentdx in current directory |
| `agentdx watch`           | Start real-time file watcher daemon    |
| `agentdx search <query>`  | Search codebase with natural language  |
| `agentdx trace <cmd>`     | Analyze call graph (callers/callees)   |
| `agentdx status`          | Browse index state interactively       |
| `agentdx agent-setup`     | Configure AI agents integration        |
| `agentdx update`          | Update agentdx to the latest version    |

```bash
agentdx search "authentication" -n 5       # Limit results (default: 10)
agentdx search "authentication" --json     # JSON output for AI agents
agentdx search "authentication" --json -c  # Compact JSON (~80% fewer tokens)
```

### Self-Update

Keep agentdx up to date:

```bash
agentdx update --check    # Check for available updates
agentdx update            # Download and install latest version
agentdx update --force    # Force update even if already on latest
```

The update command:
- Fetches the latest release from GitHub
- Verifies checksum integrity
- Replaces the binary automatically
- Works on all supported platforms (Linux, macOS, Windows)

### Call Graph Analysis

Find function relationships in your codebase:

```bash
agentdx trace callers "Login"           # Who calls Login?
agentdx trace callees "HandleRequest"   # What does HandleRequest call?
agentdx trace graph "ProcessOrder" --depth 3  # Full call graph
```

Output as JSON for AI agents:
```bash
agentdx trace callers "Login" --json
```

## AI Agent Integration

agentdx integrates natively with popular AI coding assistants. Run `agentdx agent-setup` to auto-configure.

| Agent        | Configuration File                     |
|--------------|----------------------------------------|
| Cursor       | `.cursorrules`                         |
| Windsurf     | `.windsurfrules`                       |
| Claude Code  | `CLAUDE.md` / `.claude/settings.md`    |
| Gemini CLI   | `GEMINI.md`                            |
| OpenAI Codex | `AGENTS.md`                            |

### MCP Server Mode

agentdx can run as an MCP (Model Context Protocol) server, making it available as a native tool for AI agents:

```bash
agentdx mcp-serve    # Start MCP server (stdio transport)
```

Configure in your AI tool's MCP settings:

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

Available MCP tools:
- `agentdx_search` — Semantic code search
- `agentdx_trace_callers` — Find function callers
- `agentdx_trace_callees` — Find function callees
- `agentdx_trace_graph` — Build call graph
- `agentdx_index_status` — Check index health

### Claude Code Subagent

For enhanced exploration capabilities in Claude Code, create a specialized subagent:

```bash
agentdx agent-setup --with-subagent
```

This creates `.claude/agents/deep-explore.md` with:
- Semantic search via `agentdx search`
- Call graph tracing via `agentdx trace`
- Workflow guidance for code exploration

Claude Code automatically uses this agent for deep codebase exploration tasks.

## Configuration

Stored in `.agentdx/config.yaml`:

```yaml
embedder:
  provider: ollama          # ollama | lmstudio | openai
  model: nomic-embed-text
  endpoint: http://localhost:11434  # Custom endpoint (for Azure OpenAI, etc.)
  dimensions: 768           # Vector dimensions (depends on model)
store:
  backend: gob              # gob | postgres
chunking:
  size: 512
  overlap: 50
search:
  boost:
    enabled: true           # Structural boosting for better relevance
trace:
  mode: fast                # fast (regex) | precise (tree-sitter)
```

> **Note**: Old configs without `endpoint` or `dimensions` are automatically updated with sensible defaults.

### Search Boost (enabled by default)

agentdx automatically adjusts search scores based on file paths. Patterns are language-agnostic:

| Category | Patterns | Factor |
|----------|----------|--------|
| Tests | `/tests/`, `/test/`, `__tests__`, `_test.`, `.test.`, `.spec.` | ×0.5 |
| Mocks | `/mocks/`, `/mock/`, `.mock.` | ×0.4 |
| Fixtures | `/fixtures/`, `/testdata/` | ×0.4 |
| Generated | `/generated/`, `.generated.`, `.gen.` | ×0.4 |
| Docs | `.md`, `/docs/` | ×0.6 |
| Source | `/src/`, `/lib/`, `/app/` | ×1.1 |

Customize or disable in `.agentdx/config.yaml`. See [documentation](https://yoanbernabeu.github.io/agentdx/configuration/) for details.

### Hybrid Search (optional)

Enable hybrid search to combine vector similarity with text matching:

```yaml
search:
  hybrid:
    enabled: true
    k: 60
```

Uses [Reciprocal Rank Fusion](https://plg.uwaterloo.ca/~gvcormac/cormacksigir09-rrf.pdf) to merge results. Useful when queries contain exact identifiers.

### Embedding Providers

**Ollama (Default)** — Privacy-first, runs locally:

```bash
ollama pull nomic-embed-text
```

**LM Studio** — Local, OpenAI-compatible API:

```bash
# Start LM Studio and load an embedding model
# Default endpoint: http://127.0.0.1:1234
```

**OpenAI** — Cloud-based:

```bash
export OPENAI_API_KEY=sk-...
```

### Storage Backends

- **GOB (Default)**: File-based, zero config
- **PostgreSQL + pgvector**: For large monorepos

## Requirements

- Ollama, LM Studio, or OpenAI API key (for embeddings)
- Go 1.22+ (only for building from source)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

[MIT License](LICENSE) - Yoan Bernabeu 2026
