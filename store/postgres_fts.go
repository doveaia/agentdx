package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresFTSStore implements CodeStore using PostgreSQL Full Text Search.
// It uses pg_textsearch extension for true BM25 ranking when available,
// falling back to ts_rank with 'simple' configuration for code content.
type PostgresFTSStore struct {
	pool          *pgxpool.Pool
	projectID     string
	hasBM25       bool   // true if pg_textsearch extension is available
	bm25IndexName string // name of the BM25 index for explicit queries
	dsn           string
	dbName        string
	dbHost        string
}

// BackendStatus returns the backend status
func (s *PostgresFTSStore) BackendStatus(ctx context.Context) *BackendStatus {
	healthy := s.pool != nil
	if healthy {
		if err := s.pool.Ping(ctx); err != nil {
			healthy = false
		}
	}
	return &BackendStatus{
		Type:    "postgres",
		Host:    s.dbHost,
		Name:    s.dbName,
		Healthy: healthy,
	}
}

// NewPostgresFTSStore creates a new PostgresFTSStore with FTS support
func NewPostgresFTSStore(ctx context.Context, dsn string, projectID string) (*PostgresFTSStore, error) {
	// Parse DSN to extract database name
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	store := &PostgresFTSStore{
		pool:          pool,
		projectID:     projectID,
		hasBM25:       false,
		bm25IndexName: "idx_chunks_fts_bm25",
		dsn:           dsn,
		dbName:        config.ConnConfig.Config.Database,
		dbHost:        config.ConnConfig.Config.Host,
	}

	if err := store.ensureSchema(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return store, nil
}

func (s *PostgresFTSStore) ensureSchema(ctx context.Context) error {
	// First, try to enable pg_textsearch extension for BM25 support
	_, err := s.pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pg_textsearch`)
	if err == nil {
		s.hasBM25 = true
	}
	// If extension is not available, we'll fall back to ts_rank

	queries := []string{
		// Create chunks table with content for FTS
		// Using 'simple' config to avoid stopword filtering (important for code)
		`CREATE TABLE IF NOT EXISTS chunks_fts (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			file_path TEXT NOT NULL,
			start_line INTEGER NOT NULL,
			end_line INTEGER NOT NULL,
			content TEXT NOT NULL,
			content_tsv tsvector,
			hash TEXT NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		// Index for project filtering
		`CREATE INDEX IF NOT EXISTS idx_chunks_fts_project ON chunks_fts(project_id)`,
		// Composite index for file-based operations
		`CREATE INDEX IF NOT EXISTS idx_chunks_fts_file ON chunks_fts(project_id, file_path)`,
		// Documents table for tracking indexed files
		`CREATE TABLE IF NOT EXISTS documents_fts (
			path TEXT NOT NULL,
			project_id TEXT NOT NULL,
			hash TEXT NOT NULL,
			mod_time TIMESTAMP NOT NULL,
			chunk_ids TEXT[] NOT NULL,
			PRIMARY KEY (project_id, path)
		)`,
	}

	for _, query := range queries {
		if _, err := s.pool.Exec(ctx, query); err != nil {
			return fmt.Errorf("failed to execute schema query: %w", err)
		}
	}

	// Create search indexes based on available features
	if s.hasBM25 {
		// Use pg_textsearch BM25 index for true BM25 ranking
		// 'simple' config preserves all tokens without stemming (important for code)
		_, err := s.pool.Exec(ctx, fmt.Sprintf(
			`CREATE INDEX IF NOT EXISTS %s ON chunks_fts USING bm25(content) WITH (text_config='simple')`,
			s.bm25IndexName,
		))
		if err != nil {
			// BM25 index creation failed, fall back to GIN
			s.hasBM25 = false
		}
	}

	if !s.hasBM25 {
		// Fall back to GIN index with tsvector for ts_rank scoring
		_, err := s.pool.Exec(ctx,
			`CREATE INDEX IF NOT EXISTS idx_chunks_fts_tsv ON chunks_fts USING GIN(content_tsv)`,
		)
		if err != nil {
			return fmt.Errorf("failed to create GIN index: %w", err)
		}
	}

	return nil
}

// SaveChunks stores multiple chunks with tsvector data
func (s *PostgresFTSStore) SaveChunks(ctx context.Context, chunks []Chunk) error {
	batch := &pgx.Batch{}

	for _, chunk := range chunks {
		// Use 'simple' text search configuration to preserve all tokens
		// This is important for code since we don't want stopword removal
		// or stemming that would drop important programming keywords
		batch.Queue(
			`INSERT INTO chunks_fts (id, project_id, file_path, start_line, end_line, content, content_tsv, hash, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, to_tsvector('simple', $6), $7, $8)
			ON CONFLICT (id) DO UPDATE SET
				file_path = EXCLUDED.file_path,
				start_line = EXCLUDED.start_line,
				end_line = EXCLUDED.end_line,
				content = EXCLUDED.content,
				content_tsv = EXCLUDED.content_tsv,
				hash = EXCLUDED.hash,
				updated_at = EXCLUDED.updated_at`,
			chunk.ID, s.projectID, chunk.FilePath, chunk.StartLine, chunk.EndLine,
			chunk.Content, chunk.Hash, chunk.UpdatedAt,
		)
	}

	results := s.pool.SendBatch(ctx, batch)
	defer results.Close()

	for range chunks {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("failed to save chunk: %w", err)
		}
	}

	return nil
}

// DeleteByFile removes all chunks for a given file path
func (s *PostgresFTSStore) DeleteByFile(ctx context.Context, filePath string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM chunks_fts WHERE project_id = $1 AND file_path = $2`,
		s.projectID, filePath,
	)
	if err != nil {
		return fmt.Errorf("failed to delete chunks: %w", err)
	}
	return nil
}

// SearchFTS performs full-text search using the query text directly.
// When pg_textsearch is available, it uses true BM25 ranking via the <@> operator.
// Otherwise, it falls back to ts_rank with normalization.
func (s *PostgresFTSStore) SearchFTS(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	words := strings.Fields(query)
	if len(words) == 0 {
		return nil, nil
	}

	var rows pgx.Rows
	var err error

	if s.hasBM25 {
		// Use pg_textsearch BM25 ranking with <@> operator
		// The operator returns negative BM25 scores (lower = more relevant)
		// We negate the score to get positive values where higher = more relevant
		//
		// Using to_bm25query with explicit index name for compatibility with
		// all query evaluation strategies
		rows, err = s.pool.Query(ctx,
			fmt.Sprintf(`SELECT id, file_path, start_line, end_line, content, hash, updated_at,
				-(content <@> to_bm25query($1, '%s')) as score
			FROM chunks_fts
			WHERE project_id = $2
			ORDER BY content <@> to_bm25query($1, '%s')
			LIMIT $3`, s.bm25IndexName, s.bm25IndexName),
			query, s.projectID, limit,
		)
	} else {
		// Fall back to ts_rank with tsvector
		// Build tsquery: word1 & word2 & word3 (all words must match)
		tsqueryParts := make([]string, len(words))
		for i, word := range words {
			// Escape special characters and use prefix matching with :*
			escapedWord := strings.ReplaceAll(word, "'", "''")
			tsqueryParts[i] = escapedWord + ":*"
		}
		tsqueryStr := strings.Join(tsqueryParts, " & ")

		// Use ts_rank with normalization to get scores
		// Normalization 32 = divide rank by (rank + 1) to get 0-1 range
		rows, err = s.pool.Query(ctx,
			`SELECT id, file_path, start_line, end_line, content, hash, updated_at,
				ts_rank(content_tsv, to_tsquery('simple', $1), 32) as score
			FROM chunks_fts
			WHERE project_id = $2
				AND content_tsv @@ to_tsquery('simple', $1)
			ORDER BY score DESC
			LIMIT $3`,
			tsqueryStr, s.projectID, limit,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var chunk Chunk
		var score float32

		if err := rows.Scan(
			&chunk.ID, &chunk.FilePath, &chunk.StartLine, &chunk.EndLine,
			&chunk.Content, &chunk.Hash, &chunk.UpdatedAt, &score,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		results = append(results, SearchResult{
			Chunk: chunk,
			Score: score,
		})
	}

	return results, rows.Err()
}

// GetDocument retrieves document metadata by path
func (s *PostgresFTSStore) GetDocument(ctx context.Context, filePath string) (*Document, error) {
	var doc Document
	var modTime time.Time

	err := s.pool.QueryRow(ctx,
		`SELECT path, hash, mod_time, chunk_ids FROM documents_fts WHERE project_id = $1 AND path = $2`,
		s.projectID, filePath,
	).Scan(&doc.Path, &doc.Hash, &modTime, &doc.ChunkIDs)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	doc.ModTime = modTime
	return &doc, nil
}

// SaveDocument stores document metadata
func (s *PostgresFTSStore) SaveDocument(ctx context.Context, doc Document) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO documents_fts (path, project_id, hash, mod_time, chunk_ids)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (project_id, path) DO UPDATE SET
			hash = EXCLUDED.hash,
			mod_time = EXCLUDED.mod_time,
			chunk_ids = EXCLUDED.chunk_ids`,
		doc.Path, s.projectID, doc.Hash, doc.ModTime, doc.ChunkIDs,
	)
	if err != nil {
		return fmt.Errorf("failed to save document: %w", err)
	}
	return nil
}

// DeleteDocument removes document metadata
func (s *PostgresFTSStore) DeleteDocument(ctx context.Context, filePath string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM documents_fts WHERE project_id = $1 AND path = $2`,
		s.projectID, filePath,
	)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	return nil
}

// ListDocuments returns all indexed document paths
func (s *PostgresFTSStore) ListDocuments(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT path FROM documents_fts WHERE project_id = $1`,
		s.projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, fmt.Errorf("failed to scan path: %w", err)
		}
		paths = append(paths, path)
	}

	return paths, rows.Err()
}

// Close closes the database connection pool
func (s *PostgresFTSStore) Close() error {
	s.pool.Close()
	return nil
}

// GetStats returns index statistics
func (s *PostgresFTSStore) GetStats(ctx context.Context) (*IndexStats, error) {
	var stats IndexStats

	// Get file count
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM documents_fts WHERE project_id = $1`,
		s.projectID,
	).Scan(&stats.TotalFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to count documents: %w", err)
	}

	// Get chunk count and last updated
	err = s.pool.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(MAX(updated_at), '1970-01-01'::timestamp) FROM chunks_fts WHERE project_id = $1`,
		s.projectID,
	).Scan(&stats.TotalChunks, &stats.LastUpdated)
	if err != nil {
		return nil, fmt.Errorf("failed to count chunks: %w", err)
	}

	// IndexSize not applicable for Postgres (data stored remotely)
	stats.IndexSize = 0

	return &stats, nil
}

// ListFilesWithStats returns all files with their chunk counts
func (s *PostgresFTSStore) ListFilesWithStats(ctx context.Context) ([]FileStats, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT path, mod_time, array_length(chunk_ids, 1) FROM documents_fts WHERE project_id = $1`,
		s.projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	defer rows.Close()

	var files []FileStats
	for rows.Next() {
		var f FileStats
		var chunkCount *int
		if err := rows.Scan(&f.Path, &f.ModTime, &chunkCount); err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}
		if chunkCount != nil {
			f.ChunkCount = *chunkCount
		}
		files = append(files, f)
	}

	return files, rows.Err()
}

// GetChunksForFile returns all chunks for a specific file
func (s *PostgresFTSStore) GetChunksForFile(ctx context.Context, filePath string) ([]Chunk, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, file_path, start_line, end_line, content, hash, updated_at
		FROM chunks_fts WHERE project_id = $1 AND file_path = $2
		ORDER BY start_line`,
		s.projectID, filePath,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunks: %w", err)
	}
	defer rows.Close()

	var chunks []Chunk
	for rows.Next() {
		var c Chunk
		if err := rows.Scan(&c.ID, &c.FilePath, &c.StartLine, &c.EndLine, &c.Content, &c.Hash, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}
		chunks = append(chunks, c)
	}

	return chunks, rows.Err()
}

// GetAllChunks returns all chunks in the store
func (s *PostgresFTSStore) GetAllChunks(ctx context.Context) ([]Chunk, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, file_path, start_line, end_line, content, hash, updated_at
		FROM chunks_fts WHERE project_id = $1`,
		s.projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get all chunks: %w", err)
	}
	defer rows.Close()

	var chunks []Chunk
	for rows.Next() {
		var c Chunk
		if err := rows.Scan(&c.ID, &c.FilePath, &c.StartLine, &c.EndLine, &c.Content, &c.Hash, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}
		chunks = append(chunks, c)
	}

	return chunks, rows.Err()
}

// FTSSearcher is an interface for stores that support full-text search
type FTSSearcher interface {
	SearchFTS(ctx context.Context, query string, limit int) ([]SearchResult, error)
}
