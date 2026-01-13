---
title: Introduction
description: What is agentdx and why use it
---

## What is agentdx?

**agentdx** is a privacy-first semantic code search tool that indexes the *meaning* of your code using vector embeddings, enabling natural language searches.

Unlike traditional tools like `grep` or `ripgrep` that search for exact text matches, agentdx understands what your code *does*, not just what it *says*.

## Why agentdx?

### The Problem

When working on large codebases, finding relevant code is challenging:

- **grep/ripgrep**: Great for exact matches, but useless when you don't know the exact variable name or function
- **IDE search**: Limited to the files you have open
- **AI assistants**: Often lack full context of your codebase

### The Solution

agentdx maintains a real-time "mental map" of your project:

1. **Indexes your code** using vector embeddings (local or cloud)
2. **Watches for changes** and updates the index automatically
3. **Searches semantically** - find code by describing what it does

## Example Searches

```bash
# Find authentication code without knowing function names
agentdx search "user login validation"

# Find error handling patterns
agentdx search "how are errors handled in API requests"

# Find database operations
agentdx search "where are users saved to the database"
```

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Your Code     │     │    Embedder     │     │  Vector Store   │
│   (files)       │ ──► │  (Ollama/OpenAI)│ ──► │  (GOB/Postgres) │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                                                        │
                                                        ▼
                              ┌─────────────────────────────────────┐
                              │  Semantic Search                    │
                              │  "authentication flow" → results    │
                              └─────────────────────────────────────┘
```

## Next Steps

- [Installation](/agentdx/installation/) - Install agentdx on your system
- [Quick Start](/agentdx/quickstart/) - Get up and running in 5 minutes
- [Configuration](/agentdx/configuration/) - Customize agentdx for your needs
