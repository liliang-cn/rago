package storage

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/store"
)

// SQLiteVectorBackend implements VectorBackend using the existing SQLite store.
type SQLiteVectorBackend struct {
	store  *store.SQLiteStore
	config VectorConfig
}

// NewSQLiteVectorBackend creates a new SQLite vector backend.
func NewSQLiteVectorBackend(config VectorConfig) (*SQLiteVectorBackend, error) {
	if config.DBPath == "" {
		return nil, core.NewConfigurationError("storage", "db_path", "database path is required", nil)
	}

	store, err := store.NewSQLiteStore(config.DBPath)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "storage", "NewSQLiteVectorBackend", "failed to create SQLite store")
	}

	return &SQLiteVectorBackend{
		store:  store,
		config: config,
	}, nil
}

// StoreVectors stores vector embeddings for document chunks.
func (b *SQLiteVectorBackend) StoreVectors(ctx context.Context, docID string, chunks []VectorChunk) error {
	if len(chunks) == 0 {
		return nil
	}

	log.Printf("[VECTOR] Storing %d vector chunks for document %s", len(chunks), docID)

	// Convert to domain chunks for the existing store
	domainChunks := make([]domain.Chunk, len(chunks))
	for i, chunk := range chunks {
		domainChunks[i] = domain.Chunk{
			ID:         chunk.ChunkID,
			DocumentID: chunk.DocumentID,
			Content:    chunk.Content,
			Vector:     chunk.Vector,
			Metadata:   chunk.Metadata,
		}
	}

	return b.store.Store(ctx, domainChunks)
}

// SearchVectors performs vector similarity search.
func (b *SQLiteVectorBackend) SearchVectors(ctx context.Context, queryVector []float64, options VectorSearchOptions) (*VectorSearchResult, error) {
	if len(queryVector) == 0 {
		return nil, core.NewValidationError("query_vector", queryVector, "query vector cannot be empty")
	}

	// Set defaults
	limit := options.Limit
	if limit <= 0 {
		limit = 10
	}

	var chunks []domain.Chunk
	var err error

	// Use filtered search if filters are provided
	if len(options.Filter) > 0 {
		chunks, err = b.store.SearchWithFilters(ctx, queryVector, limit, options.Filter)
	} else {
		chunks, err = b.store.Search(ctx, queryVector, limit)
	}

	if err != nil {
		return nil, core.WrapErrorWithContext(err, "storage", "SearchVectors", "vector search failed")
	}

	// Convert results
	hits := make([]VectorSearchHit, len(chunks))
	var maxScore float64

	for i, chunk := range chunks {
		hit := VectorSearchHit{
			VectorChunk: VectorChunk{
				ChunkID:    chunk.ID,
				DocumentID: chunk.DocumentID,
				Content:    chunk.Content,
				Metadata:   chunk.Metadata,
				CreatedAt:  time.Now(), // TODO: Get actual created time from metadata
			},
			Score: chunk.Score,
		}

		// Include vector if requested
		if options.IncludeVector {
			hit.Vector = chunk.Vector
		}

		hits[i] = hit

		if chunk.Score > maxScore {
			maxScore = chunk.Score
		}
	}

	result := &VectorSearchResult{
		Chunks:   hits,
		Total:    len(hits), // TODO: Get actual total count
		MaxScore: maxScore,
	}

	if options.IncludeVector {
		result.QueryVector = queryVector
	}

	return result, nil
}

// DeleteDocument removes all vectors for a document.
func (b *SQLiteVectorBackend) DeleteDocument(ctx context.Context, docID string) error {
	if docID == "" {
		return core.NewValidationError("document_id", docID, "document ID cannot be empty")
	}

	return b.store.Delete(ctx, docID)
}

// GetDocumentVectors retrieves vectors for a specific document.
func (b *SQLiteVectorBackend) GetDocumentVectors(ctx context.Context, docID string) ([]VectorChunk, error) {
	if docID == "" {
		return nil, core.NewValidationError("document_id", docID, "document ID cannot be empty")
	}

	// TODO: Implement GetByDocID in the existing store interface
	// For now, we'll use a search with filters
	docs, err := b.store.List(ctx)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "storage", "GetDocumentVectors", "failed to list documents")
	}

	var chunks []VectorChunk
	for _, doc := range docs {
		if doc.ID == docID {
			// This is a simplified approach - in practice we'd need to get the actual chunks
			// TODO: Implement proper chunk retrieval by document ID
		}
	}

	return chunks, nil
}

// GetStats returns storage statistics.
func (b *SQLiteVectorBackend) GetStats(ctx context.Context) (*VectorStats, error) {
	// Get document list to calculate basic stats
	docs, err := b.store.List(ctx)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "storage", "GetStats", "failed to get documents")
	}

	stats := &VectorStats{
		TotalDocuments: int64(len(docs)),
		Dimensions:     b.config.Dimensions,
		Performance:    make(map[string]interface{}),
	}

	// Calculate total vectors by summing chunk counts
	var totalVectors int64
	var totalSize int64
	for _, doc := range docs {
		// TODO: Get actual chunk count from metadata or store
		// For now, estimate based on content length
		estimatedChunks := int64(len(doc.Content) / 1000) // Rough estimate
		if estimatedChunks == 0 {
			estimatedChunks = 1
		}
		totalVectors += estimatedChunks
		totalSize += int64(len(doc.Content))
	}

	stats.TotalVectors = totalVectors
	stats.StorageSize = totalSize

	return stats, nil
}

// Optimize performs index optimization.
func (b *SQLiteVectorBackend) Optimize(ctx context.Context) error {
	log.Printf("[VECTOR] Performing optimization")
	// SQLite auto-optimizes, but we could implement VACUUM or other optimizations here
	// For now, this is a no-op
	return nil
}

// Reset clears all data.
func (b *SQLiteVectorBackend) Reset(ctx context.Context) error {
	log.Printf("[VECTOR] Resetting vector store")
	return b.store.Reset(ctx)
}

// Close closes the backend.
func (b *SQLiteVectorBackend) Close() error {
	log.Printf("[VECTOR] Closing SQLite vector backend")
	return b.store.Close()
}

// GetStore returns the underlying SQLite store for backend integration.
// This is used by the document backend which shares the same SQLite database.
func (b *SQLiteVectorBackend) GetStore() *store.SQLiteStore {
	return b.store
}

// ===== FACTORY FUNCTION =====

// NewVectorBackend creates a vector backend based on the configuration.
func NewVectorBackend(config VectorConfig) (VectorBackend, error) {
	switch config.Backend {
	case "sqvect", "sqlite":
		return NewSQLiteVectorBackend(config)
	// TODO: Add other backends like Chroma, Qdrant, etc.
	// case "chroma":
	//     return NewChromaVectorBackend(config)
	// case "qdrant":
	//     return NewQdrantVectorBackend(config)
	default:
		return nil, core.NewConfigurationError("storage", "backend", 
			fmt.Sprintf("unsupported vector backend: %s", config.Backend), nil)
	}
}

// ===== UTILITY TYPES =====