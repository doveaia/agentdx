---
title: Installation
description: How to install agentdx
---

## Prerequisites

- **Ollama** (for local embeddings) or an **OpenAI API key** (for cloud embeddings)

## Quick Install (Recommended)

```bash
curl -sSL https://raw.githubusercontent.com/yoanbernabeu/agentdx/main/install.sh | sh
```

Or download directly from [Releases](https://github.com/yoanbernabeu/agentdx/releases).

## Install from Source

Requires **Go 1.24+**.

```bash
# Clone the repository
git clone https://github.com/yoanbernabeu/agentdx.git
cd agentdx

# Build the binary
make build

# The binary is created at ./bin/agentdx
# Move it to your PATH
sudo mv ./bin/agentdx /usr/local/bin/
```

## Install Ollama (Recommended)

For privacy-first local embeddings, install Ollama:

```bash
# macOS
brew install ollama

# Linux
curl -fsSL https://ollama.com/install.sh | sh

# Start Ollama
ollama serve

# Pull the embedding model
ollama pull nomic-embed-text
```

## Verify Installation

```bash
# Check agentdx is installed
agentdx version

# Check Ollama is running (if using local embeddings)
curl http://localhost:11434/api/tags
```

## Updating

Keep agentdx up to date with the built-in update command:

```bash
# Check for available updates
agentdx update --check

# Download and install the latest version
agentdx update
```

The update command will:
- Fetch the latest release from GitHub
- Download the appropriate binary for your platform
- Verify checksum integrity
- Replace the current binary automatically

## Next Steps

- [Quick Start](/agentdx/quickstart/) - Initialize and start using agentdx
- [Configuration](/agentdx/configuration/) - Configure embedders and storage backends
