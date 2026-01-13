---
title: Embedders
description: Configure embedding providers for agentdx
---

Embedders convert text (code chunks) into vector representations that enable semantic search.

## Available Embedders

| Provider | Type | Pros | Cons |
|----------|------|------|------|
| Ollama | Local | Privacy, free, no internet | Requires local resources |
| LM Studio | Local | Privacy, OpenAI-compatible API, GUI | Requires local resources |
| OpenAI | Cloud | High quality, fast | Costs money, sends code to cloud |
| PostgreSQL FTS | Database | True BM25 ranking, no ML model needed | Requires PostgreSQL 17+ with pg_textsearch |

## Ollama (Local)

### Setup

1. Install Ollama:
```bash
# macOS
brew install ollama

# Linux
curl -fsSL https://ollama.com/install.sh | sh
```

2. Start the server:
```bash
ollama serve
```

3. Pull an embedding model:
```bash
ollama pull nomic-embed-text
```

### Configuration

```yaml
embedder:
  provider: ollama
  model: nomic-embed-text
  endpoint: http://localhost:11434
  dimensions: 768
```

### Available Models

| Model | Dimensions | Speed | Quality |
|-------|------------|-------|---------|
| `nomic-embed-text` | 768 | Fast | Good |
| `mxbai-embed-large` | 1024 | Medium | Better |
| `all-minilm` | 384 | Very Fast | Basic |
| `qwen3-embedding:0.6b` | 1024 | Medium | Excellent |
| `embeddinggemma` | 768 | Very Fast | Better |

### Troubleshooting

```bash
# Check if Ollama is running
curl http://localhost:11434/api/tags

# Test embedding
curl http://localhost:11434/api/embeddings -d '{
  "model": "nomic-embed-text",
  "prompt": "Hello world"
}'
```

## LM Studio (Local)

LM Studio provides an OpenAI-compatible API for running embedding models locally with a user-friendly GUI.

### Setup

1. Download and install [LM Studio](https://lmstudio.ai/)

2. Start LM Studio and load an embedding model (e.g., `nomic-embed-text`)

3. Enable the local server (default: http://127.0.0.1:1234)

### Configuration

```yaml
embedder:
  provider: lmstudio
  model: text-embedding-nomic-embed-text-v1.5
  endpoint: http://127.0.0.1:1234
```

### Available Models

Any embedding model supported by LM Studio, including:

| Model | Dimensions | Notes |
|-------|------------|-------|
| `nomic-embed-text-v1.5` | 768 | Good general purpose |
| `bge-small-en-v1.5` | 384 | Fast, smaller |
| `bge-large-en-v1.5` | 1024 | Higher quality |

### Troubleshooting

```bash
# Check if LM Studio server is running
curl http://127.0.0.1:1234/v1/models

# Test embedding
curl http://127.0.0.1:1234/v1/embeddings -d '{
  "model": "text-embedding-nomic-embed-text-v1.5",
  "input": ["Hello world"]
}'
```

## OpenAI (Cloud)

### Setup

1. Get an API key from [OpenAI Platform](https://platform.openai.com/api-keys)

2. Set the environment variable:
```bash
export OPENAI_API_KEY=sk-...
```

### Configuration

```yaml
embedder:
  provider: openai
  model: text-embedding-3-small
  api_key: ${OPENAI_API_KEY}
  dimensions: 1536
```

### Azure OpenAI / Microsoft Foundry

For Azure OpenAI or other OpenAI-compatible providers, use a custom endpoint:

```yaml
embedder:
  provider: openai
  model: text-embedding-ada-002
  endpoint: https://YOUR-RESOURCE.openai.azure.com/v1
  api_key: ${AZURE_OPENAI_API_KEY}
  dimensions: 1536
```

### Available Models

| Model | Dimensions | Price (per 1M tokens) |
|-------|------------|----------------------|
| `text-embedding-3-small` | 1536 | $0.02 |
| `text-embedding-3-large` | 3072 | $0.13 |

### Cost Estimation

For a typical codebase:
- 10,000 lines of code ≈ 50,000 tokens
- Initial index: ~$0.001 with `text-embedding-3-small`
- Ongoing updates: negligible

## PostgreSQL Full Text Search (Database)

PostgreSQL FTS provides true BM25 ranking without requiring any external ML embedding service. This is ideal when you want fast keyword-based search with proper relevance ranking.

### Requirements

- PostgreSQL 17 or higher
- [pg_textsearch](https://github.com/timescale/pg_textsearch) extension for true BM25 ranking

### Setup

1. Install pg_textsearch extension (see [pg_textsearch installation guide](https://github.com/timescale/pg_textsearch#installation))

2. Create a database for agentdx:
```bash
createdb agentdx_index
```

3. Initialize agentdx with postgres provider:
```bash
agentdx init
# Select option 4 (postgres)
# Enter your PostgreSQL DSN
```

### Configuration

```yaml
embedder:
  provider: postgres

store:
  backend: postgres
  postgres:
    dsn: postgres://user:password@localhost:5432/agentdx_index?sslmode=disable
```

### How It Works

When using PostgreSQL FTS:

1. **Indexing**: Code chunks are stored with a BM25 index using `to_tsvector('simple', content)`. The 'simple' configuration preserves all tokens without stemming, which is important for code.

2. **Searching**: Queries use the pg_textsearch `<@>` operator with `to_bm25query()` for true BM25 relevance scoring.

3. **Fallback**: If pg_textsearch is not available, the system falls back to PostgreSQL's native `ts_rank()` function with GIN indexes.

### BM25 Ranking

BM25 (Best Matching 25) is a probabilistic ranking function that considers:
- **Term frequency**: How often the search terms appear in a document
- **Inverse document frequency**: Rare terms get higher weight
- **Document length normalization**: Longer documents don't unfairly dominate

This provides more accurate relevance scoring than simple keyword matching.

### Troubleshooting

```bash
# Check if pg_textsearch is installed
psql -d agentdx_index -c "SELECT * FROM pg_extension WHERE extname = 'pg_textsearch';"

# Check BM25 index
psql -d agentdx_index -c "SELECT indexname FROM pg_indexes WHERE indexdef LIKE '%bm25%';"

# Test BM25 search directly
psql -d agentdx_index -c "SELECT content <@> 'search term' as score FROM chunks_fts ORDER BY score LIMIT 5;"
```

## Adding a New Embedder

To add a new embedding provider:

1. Implement the `Embedder` interface in `embedder/`:

```go
type Embedder interface {
    Embed(ctx context.Context, text string) ([]float32, error)
    EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
    Dimensions() int
}
```

2. Add configuration in `config/config.go`

3. Wire it up in the CLI commands

See [Contributing](/agentdx/contributing/) for more details.
