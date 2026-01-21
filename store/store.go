package store

import (
	"context"
	"time"
)

// Chunk represents a piece of code with its embedding
type Chunk struct {
	ID        string    `json:"id"`
	FilePath  string    `json:"file_path"`
	StartLine int       `json:"start_line"`
	EndLine   int       `json:"end_line"`
	Content   string    `json:"content"`
	Hash      string    `json:"hash"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Document represents a file with its chunks
type Document struct {
	Path     string    `json:"path"`
	Hash     string    `json:"hash"`
	ModTime  time.Time `json:"mod_time"`
	ChunkIDs []string  `json:"chunk_ids"`
}

// SearchResult represents a search match with its relevance score
type SearchResult struct {
	Chunk Chunk   `json:"chunk"`
	Score float32 `json:"score"`
}

// IndexStats contains statistics about the index
type IndexStats struct {
	TotalFiles  int       `json:"total_files"`
	TotalChunks int       `json:"total_chunks"`
	IndexSize   int64     `json:"index_size"` // bytes
	LastUpdated time.Time `json:"last_updated"`
}

// FileStats contains statistics for a single file
type FileStats struct {
	Path       string    `json:"path"`
	ChunkCount int       `json:"chunk_count"`
	ModTime    time.Time `json:"mod_time"`
}

// BackendStatus represents the status of a storage backend
type BackendStatus struct {
	Type    string `json:"type"`    // Backend type (e.g., "gob", "postgres")
	Host    string `json:"host"`    // Backend host/path (e.g., "localhost", "/path/to/index")
	Name    string `json:"name"`    // Backend name (e.g., database name, index name)
	Healthy bool   `json:"healthy"` // true if backend is reachable and operational
}

// StatusProvider is an optional interface for backends that can report their status
type StatusProvider interface {
	BackendStatus(ctx context.Context) *BackendStatus
}

// CodeStore defines the interface for code storage backends
type CodeStore interface {
	// SaveChunks stores multiple chunks atomically
	SaveChunks(ctx context.Context, chunks []Chunk) error

	// DeleteByFile removes all chunks for a given file path
	DeleteByFile(ctx context.Context, filePath string) error

	// GetDocument retrieves document metadata by path
	GetDocument(ctx context.Context, filePath string) (*Document, error)

	// SaveDocument stores document metadata
	SaveDocument(ctx context.Context, doc Document) error

	// DeleteDocument removes document metadata
	DeleteDocument(ctx context.Context, filePath string) error

	// ListDocuments returns all indexed document paths
	ListDocuments(ctx context.Context) ([]string, error)

	// Close cleanly shuts down the store
	Close() error

	// GetStats returns index statistics
	GetStats(ctx context.Context) (*IndexStats, error)

	// ListFilesWithStats returns all files with their chunk counts
	ListFilesWithStats(ctx context.Context) ([]FileStats, error)

	// GetChunksForFile returns all chunks for a specific file
	GetChunksForFile(ctx context.Context, filePath string) ([]Chunk, error)

	// GetAllChunks returns all chunks in the store (used for text search)
	GetAllChunks(ctx context.Context) ([]Chunk, error)
}
