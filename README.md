# agentdx

[![Go](https://github.com/doveaia/agentdx/actions/workflows/ci.yml/badge.svg)](https://github.com/doveaia/agentdx/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/doveaia/agentdx)](https://goreportcard.com/report/github.com/doveaia/agentdx)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**Full-text search for AI agents — because that's how agents actually search code.**

`agentdx` uses PostgreSQL full-text search optimized for code patterns. Not semantic search, not vector embeddings — just precise, token-efficient full-text search that matches how AI coding assistants actually explore codebases.

## Inspiration

`agentdx` is inspired by and built on the foundation of [grepai](https://github.com/yoanbernabeu/grepai), a semantic code search tool that uses vector embeddings to find code by meaning.

While **grepai** takes a semantic search approach (finding code by *what it does* using embeddings), **agentdx** takes a different path: full-text search optimized for how AI agents actually work. We observed that AI coding assistants search by exact patterns — function names, class names, identifiers — not by semantic similarity. This insight led us to fork the architecture and replace vector search with PostgreSQL full-text search, achieving 73% token reduction while maintaining precise results.

## Token Efficiency: 73% Reduction

AI agents search code by matching text patterns, not by semantic similarity. Full-text search aligns with this workflow and dramatically reduces token consumption.

| Without agentdx | With agentdx |
|-----------------|--------------|
| ![Without agentdx](assets/without-agentdx.png) | ![With agentdx](assets/with-agentdx.png) |
| **188k tokens** | **51k tokens** |

## Why full-text search instead of semantic?

`grep` was built in 1973 for exact text matching. Modern codebases need code-aware search. But semantic/vector search is wrong for AI agents — they search by text patterns, not "meanings."

|                      | `grep` / `ripgrep`           | Semantic / Vector Search          | `agentdx` FTS                     |
|----------------------|------------------------------|-----------------------------------|-----------------------------------|
| **Search type**      | Exact text / regex           | Vector similarity                 | Full-text search with code awareness |
| **Query**            | `"func.*Login"`              | `"authentication logic"`          | `"Login" OR "login"`             |
| **Finds**            | Exact pattern matches        | "Related" concepts (misses exact) | Precise text matches + ranking   |
| **AI Agent workflow**| Requires many searches       | Mismatched with agent behavior    | **Matches how agents actually search** |
| **Token efficiency** | High (too many results)      | Medium                            | **73% reduction**                |

**Why agents prefer full-text search:** AI coding assistants search by function names, class names, and exact code patterns. They don't need "similar" code — they need the exact code that matches their query. Full-text search with proper ranking delivers exactly that.

## Getting Started

### Installation

```bash
curl -sSL https://raw.githubusercontent.com/doveaia/agentdx/main/install.sh | sh
```

Or download from [Releases](https://github.com/doveaia/agentdx/releases).

### Quick Start

```bash
agentdx init                        # Initialize in your project
agentdx watch                       # Start background indexing daemon
agentdx search "error handling"     # Search code
agentdx trace callers "Login"       # Find who calls a function
```

## Commands

| Command                  | Description                            |
|--------------------------|----------------------------------------|
| `agentdx init`            | Initialize agentdx in current directory |
| `agentdx watch`           | Start real-time file watcher daemon    |
| `agentdx search <query>`  | Full-text search codebase              |
| `agentdx trace <cmd>`     | Analyze call graph (callers/callees)   |
| `agentdx files <pattern>` | List indexed files matching glob pattern |
| `agentdx status`          | Browse index state and hooks status    |
| `agentdx setup`     | Configure AI agents integration        |
| `agentdx update`          | Update agentdx to the latest version    |
| `agentdx session`         | Manage watch daemon session            |

```bash
agentdx search "authentication" -n 5       # Limit results (default: 10)
agentdx search "authentication" --json     # JSON output for AI agents
agentdx search "authentication" --json -c  # Compact JSON (~80% fewer tokens)
```

## Automatic Session Management

agentdx can automatically start and stop the watch daemon when you use AI coding agents like Claude Code. This ensures your code index is always up-to-date during your coding sessions without manual intervention.

### Quick Setup

1. Configure agent integration (installs hooks automatically):
   ```bash
   agentdx setup
   ```

2. That's it! The watch daemon will now start automatically when you begin a coding session and stop when you end it.

### How It Works

When properly configured with hooks:
1. **Session Start** → Hook runs `agentdx session start` → Watch daemon begins
2. **During Session** → Daemon indexes file changes in real-time
3. **Session End** → Hook runs `agentdx session stop` → Daemon stops cleanly

### Manual Control

If you prefer manual control or need to troubleshoot:

```bash
# Start the watch daemon
agentdx session start

# Check daemon status (includes hooks status)
agentdx status

# Stop the watch daemon
agentdx session stop

# Force stop if unresponsive
agentdx session stop --force
```

### Supported Coding Agents

| Agent | Hook Location | Status |
|-------|--------------|--------|
| Claude Code | `.claude/hooks/PreToolUse/` and `.claude/hooks/Stop/` | ✓ Supported |
| Codex | `.codex/hooks/start/` and `.codex/hooks/stop/` | Planned |
| OpenCode | `.opencode/hooks/start/` and `.opencode/hooks/stop/` | Planned |

All hooks are project-scoped (installed in your project directory, not globally).

### Troubleshooting

**Docker Not Running:**
```
Error: Docker is not running. Please start Docker and try again.
```
**Solution**: Start Docker Desktop or the Docker daemon.

**Port Already in Use:**
```
Error: Port 55432 is already in use. Try a different port with --pg-port.
```
**Solution**:
```bash
# Use a different port
agentdx watch --pg-port 55433

# Or find what's using the port
lsof -i :55432
```

**Container Won't Start:**
```
Error: failed to create container: ...
```
**Solutions**:
```bash
# Check container logs
docker logs agentdx-postgres

# Remove stale container
docker rm agentdx-postgres

# Try again
agentdx watch
```

**PostgreSQL Not Ready:**
```
Error: PostgreSQL not ready after 30s. Check container logs: docker logs agentdx-postgres
```
**Solution**: Check container logs for startup errors.

**Test Failures with Container Conflicts:**
If tests fail with container name conflicts:
```bash
# Check for orphaned test containers
docker ps -a | grep agentdx-test

# Clean up orphaned containers
docker rm $(docker ps -aq -f name=agentdx-test-)
```

**Daemon won't start:**
- Check if agentdx is initialized: `ls .agentdx/config.yaml`
- Check logs: `cat .agentdx/session.log`
- Verify PostgreSQL is accessible

**Stale PID file:**
- Symptom: "Daemon already running" but it's not actually running
- Solution: `rm .agentdx/session.pid && agentdx session start`

**Hooks not triggering:**
- Verify hooks are installed: `agentdx status` (shows hooks status)
- Check hook permissions: `ls -la .claude/hooks/PreToolUse/`
- Ensure you're in an agentdx-initialized directory

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

agentdx integrates natively with popular AI coding assistants. Run `agentdx setup` to auto-configure.

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
agentdx serve    # Start MCP server (stdio transport)
```

Configure in your AI tool's MCP settings:

```json
{
  "mcpServers": {
    "agentdx": {
      "command": "agentdx",
      "args": ["serve"]
    }
  }
}
```

Available MCP tools:
- `agentdx_search` — Full-text code search
- `agentdx_trace_callers` — Find function callers
- `agentdx_trace_callees` — Find function callees
- `agentdx_trace_graph` — Build call graph
- `agentdx_index_status` — Check index health

### Claude Code Subagent

For enhanced exploration capabilities in Claude Code, `agentdx setup` creates a specialized subagent at `.claude/agents/deep-explore.md` with:
- Full-text search via `agentdx search`
- Call graph tracing via `agentdx trace`
- Workflow guidance for code exploration

Claude Code automatically uses this agent for deep codebase exploration tasks.

## Configuration

Stored in `.agentdx/config.yaml`:

```yaml
version: 1
mode: local
index:
  store:
    postgres:
      dsn: "postgres://agentdx:agentdx@localhost:55432/agentdx_project?sslmode=disable"
      container_name: "agentdx-postgres"  # Optional: custom container name
      port: 55432  # Optional: custom host port
  chunking:
    size: 512
    overlap: 50
  search:
    boost:
      enabled: true           # Structural boosting for better relevance
  trace:
    mode: fast                # fast (regex) | precise (tree-sitter)
```

### Custom Container Settings

You can customize the PostgreSQL container name and port via CLI flags or config file:

```bash
# Use a custom container name
agentdx watch --pg-name my-project-db

# Use a custom port
agentdx watch --pg-port 5433

# Or both (short flags)
agentdx watch -n my-db -p 5433

# Session daemon with custom settings
agentdx session start --pg-name my-project-pg --pg-port 5433
```

Or set defaults in `.agentdx/config.yaml`:

```yaml
index:
  store:
    postgres:
      container_name: "my-project-postgres"
      port: 55433
```

CLI flags always take precedence over config file settings.

### Data Persistence

PostgreSQL data is stored in a Docker volume named `{container_name}-data`, so your index survives container restarts and system reboots. The container persists after `agentdx watch` exits to preserve your index.

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

Customize or disable in `.agentdx/config.yaml`. See [documentation](https://doveaia.github.io/agentdx/configuration/) for details.

### Storage Backend

agentdx uses PostgreSQL with full-text search. Run `agentdx init` to auto-configure.

## Requirements

- PostgreSQL 12+ (auto-configured on `agentdx init`)
- Go 1.22+ (only for building from source)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

[MIT License](LICENSE)
