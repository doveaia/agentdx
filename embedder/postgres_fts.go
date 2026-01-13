package embedder

import (
	"context"
)

// PostgresFTSEmbedder is a no-op embedder for PostgreSQL Full Text Search.
// When using PostgreSQL FTS, embeddings are not needed - the database handles
// text indexing via to_tsvector. This embedder returns empty vectors to satisfy
// the interface requirements.
type PostgresFTSEmbedder struct{}

// NewPostgresFTSEmbedder creates a new PostgresFTSEmbedder
func NewPostgresFTSEmbedder() *PostgresFTSEmbedder {
	return &PostgresFTSEmbedder{}
}

// Embed returns an empty vector since FTS doesn't use embeddings
func (e *PostgresFTSEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	// Return empty vector - FTS uses tsvector in the database instead
	return []float32{}, nil
}

// EmbedBatch returns empty vectors for all texts
func (e *PostgresFTSEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = []float32{}
	}
	return result, nil
}

// Dimensions returns 0 since FTS doesn't use vector dimensions
func (e *PostgresFTSEmbedder) Dimensions() int {
	return 0
}

// Close is a no-op for FTS embedder
func (e *PostgresFTSEmbedder) Close() error {
	return nil
}
