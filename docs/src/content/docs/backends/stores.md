---
title: Vector Stores
description: Configure storage backends for agentdx
---

Vector stores persist the embeddings and enable similarity search.

## Available Stores

| Backend | Type | Pros | Cons |
|---------|------|------|------|
| GOB | File-based | Simple, no setup | Single machine only |
| PostgreSQL | Database | Scalable, team-friendly | Requires PostgreSQL + pgvector |

## GOB (File-based)

The default storage backend. Stores vectors in a binary file.

### Configuration

```yaml
store:
  backend: gob
  gob:
    path: .agentdx/index.gob
```

### Characteristics

- **Pros**:
  - Zero dependencies
  - Fast for small/medium codebases
  - Portable (just copy the file)

- **Cons**:
  - Single machine only
  - Loads entire index into memory
  - No concurrent access

### Best For

- Personal projects
- Quick experimentation
- CI/CD pipelines (ephemeral index)

## PostgreSQL with pgvector

Scalable vector storage using PostgreSQL and the pgvector extension.

### Setup

1. Install PostgreSQL with pgvector:

```bash
# macOS with Homebrew
brew install postgresql pgvector

# Ubuntu/Debian
sudo apt install postgresql postgresql-contrib
# Then install pgvector from source or package
```

2. Create the database and extension:

```sql
CREATE DATABASE agentdx;
\c agentdx
CREATE EXTENSION vector;
```

3. Create the table (auto-created by agentdx, but for reference):

```sql
CREATE TABLE IF NOT EXISTS chunks (
    id SERIAL PRIMARY KEY,
    file_path TEXT NOT NULL,
    start_line INT NOT NULL,
    end_line INT NOT NULL,
    content TEXT NOT NULL,
    embedding vector(768),  -- Adjust dimensions based on embedder
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX ON chunks USING ivfflat (embedding vector_cosine_ops);
```

### Configuration

```yaml
store:
  backend: postgres
  postgres:
    connection_string: postgres://user:password@localhost:5432/agentdx
```

### Environment Variable

```bash
export GREPAI_POSTGRES_URL="postgres://user:password@localhost:5432/agentdx"
```

```yaml
store:
  backend: postgres
  postgres:
    connection_string: ${GREPAI_POSTGRES_URL}
```

### Characteristics

- **Pros**:
  - Scales to large codebases
  - Concurrent access (team use)
  - Advanced filtering with SQL
  - Persistent and reliable

- **Cons**:
  - Requires PostgreSQL setup
  - More complex deployment

### Best For

- Team environments
- Large codebases (100k+ lines)
- Production deployments
- When you need SQL queries on metadata

## Adding a New Store

To add a new storage backend:

1. Implement the `VectorStore` interface in `store/`:

```go
type VectorStore interface {
    Store(ctx context.Context, chunks []Chunk) error
    Search(ctx context.Context, embedding []float32, limit int) ([]Result, error)
    Delete(ctx context.Context, filePath string) error
    Close() error
}
```

2. Add configuration in `config/config.go`

3. Wire it up in the CLI commands

See [Contributing](/agentdx/contributing/) for more details.
