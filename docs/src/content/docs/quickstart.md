---
title: Quick Start
description: Get up and running with agentdx in 5 minutes
---

## 1. Initialize agentdx

Navigate to your project directory and run:

```bash
cd /path/to/your/project
agentdx init
```

This creates a `.agentdx/` directory with a `config.yaml` file.

## 2. Start the Indexing Daemon

```bash
agentdx watch
```

This will:
1. Scan your codebase
2. Split files into chunks
3. Generate embeddings for each chunk
4. Store vectors in the local index
5. Watch for file changes and update the index in real-time

You'll see a progress bar during the initial indexing.

## 3. Search Your Code

Open a new terminal and search:

```bash
# Find authentication code
agentdx search "user authentication"

# Find error handling
agentdx search "how errors are handled"

# Find API endpoints
agentdx search "REST API routes"

# Limit results
agentdx search "database queries" --limit 10

# JSON output for AI agents (--compact saves ~80% tokens)
agentdx search "authentication" --json --compact
```

## 4. Check Index Status

```bash
agentdx status
```

This shows:
- Number of indexed files
- Number of chunks
- Storage backend status
- Last update time

## Example Output

```
$ agentdx search "error handling middleware"

Score: 0.89 | middleware/error.go:15-45
────────────────────────────────────────
func ErrorHandler() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
        if len(c.Errors) > 0 {
            err := c.Errors.Last()
            // ... handle error
        }
    }
}

Score: 0.82 | handlers/api.go:78-95
────────────────────────────────────────
// handleAPIError wraps errors with context
func handleAPIError(w http.ResponseWriter, err error) {
    // ...
}
```

## Tips for Better Searches

- **Be descriptive**: "user login validation" works better than "login"
- **Use natural language**: "where are users saved" instead of "save user"
- **Think about intent**: describe *what* the code does, not *how* it's written

## Next Steps

- [Semantic Search Guide](/agentdx/search-guide/) - Master natural language queries
- [File Watching Guide](/agentdx/watch-guide/) - Understand the indexing daemon
- [Call Graph Analysis](/agentdx/trace/) - Explore function relationships
- [Configuration](/agentdx/configuration/) - Customize chunking, embedders, and storage
- [Commands Reference](/agentdx/commands/agentdx/) - Full CLI documentation
- [AI Agent Setup](/agentdx/commands/agentdx_agent-setup/) - Integrate with Cursor or Claude Code
