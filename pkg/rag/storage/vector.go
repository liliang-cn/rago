package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/sqvect"
)

// SQLiteVectorBackend implements VectorBackend using V3 architecture.
type SQLiteVectorBackend struct {
	sqvect      *sqvect.SQLiteStore
	config      VectorConfig
	initialized bool
}

// NewSQLiteVectorBackend creates a new SQLite vector backend.
func NewSQLiteVectorBackend(config VectorConfig) (*SQLiteVectorBackend, error) {
	if config.DBPath == "" {
		return nil, core.NewConfigurationError("storage", "db_path", "database path is required", nil)
	}

	// Don't create the database yet - do it lazily when needed
	return &SQLiteVectorBackend{
		sqvect:      nil,
		config:      config,
		initialized: false,
	}, nil
}

// ensureInitialized ensures the sqvect client is created and initialized.
func (b *SQLiteVectorBackend) ensureInitialized(ctx context.Context) error {
	if b.initialized && b.sqvect != nil {
		return nil
	}

	log.Printf("[VECTOR] Lazy initializing sqvect database at %s", b.config.DBPath)
	
	// Create sqvect client with auto-detection (dimension 0)
	client, err := sqvect.New(b.config.DBPath, 0)
	if err != nil {
		return fmt.Errorf("failed to create sqvect client: %w", err)
	}

	// Initialize the database
	if err := client.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize sqvect: %w", err)
	}

	b.sqvect = client
	b.initialized = true
	log.Printf("[VECTOR] Successfully initialized sqvect database")
	
	return nil
}

// StoreVectors stores vector embeddings for document chunks.
func (b *SQLiteVectorBackend) StoreVectors(ctx context.Context, docID string, chunks []VectorChunk) error {
	if len(chunks) == 0 {
		return nil
	}

	// Ensure sqvect is initialized before using it
	if err := b.ensureInitialized(ctx); err != nil {
		return core.NewServiceError("vector", "StoreVectors", "failed to initialize storage", err)
	}

	log.Printf("[VECTOR] Storing %d vector chunks for document %s", len(chunks), docID)

	for _, chunk := range chunks {
		if len(chunk.Vector) == 0 {
			continue
		}

		// Convert []float64 to []float32 for sqvect
		vector := make([]float32, len(chunk.Vector))
		for i, v := range chunk.Vector {
			vector[i] = float32(v)
		}

		// Convert metadata to string map, handling slices and maps as JSON strings
		metadata := make(map[string]string)
		if chunk.Metadata != nil {
			for k, v := range chunk.Metadata {
				switch val := v.(type) {
				case []string, map[string]interface{}, []interface{}:
					jsonBytes, err := json.Marshal(val)
					if err == nil {
						metadata[k] = string(jsonBytes)
					} else {
						metadata[k] = fmt.Sprintf("%v", v)
					}
				default:
					metadata[k] = fmt.Sprintf("%v", v)
				}
			}
		}

		// Mark as chunk for filtering during search
		metadata["_type"] = "chunk"
		metadata["_position"] = fmt.Sprintf("%d", chunk.Position)

		embedding := &sqvect.Embedding{
			ID:       chunk.ChunkID,
			Vector:   vector,
			Content:  chunk.Content,
			DocID:    chunk.DocumentID,
			Metadata: metadata,
		}

		if err := b.sqvect.Upsert(ctx, embedding); err != nil {
			return core.NewServiceError("vector", "StoreVectors", "failed to store chunk", err)
		}
	}

	return nil
}

// SearchVectors performs vector similarity search.
func (b *SQLiteVectorBackend) SearchVectors(ctx context.Context, queryVector []float64, options VectorSearchOptions) (*VectorSearchResult, error) {
	if len(queryVector) == 0 {
		return nil, core.NewValidationError("query_vector", queryVector, "query vector cannot be empty")
	}

	// Ensure sqvect is initialized before using it
	if err := b.ensureInitialized(ctx); err != nil {
		return nil, core.NewServiceError("vector", "SearchVectors", "failed to initialize storage", err)
	}

	// Set defaults
	limit := options.Limit
	if limit <= 0 {
		limit = 10
	}

	// Check if there are any vectors in the database first
	count, err := b.getVectorCount(ctx)
	if err != nil {
		return nil, core.NewServiceError("vector", "SearchVectors", "failed to check vector count", err)
	}

	if count == 0 {
		// Return empty results if no vectors exist
		result := &VectorSearchResult{
			Chunks:   []VectorSearchHit{},
			Total:    0,
			MaxScore: 0.0,
		}
		if options.IncludeVector {
			result.QueryVector = queryVector
		}
		return result, nil
	}

	// Convert []float64 to []float32 for sqvect
	queryVec := make([]float32, len(queryVector))
	for i, v := range queryVector {
		queryVec[i] = float32(v)
	}

	// Use SearchWithFilter to exclude document metadata
	filters := map[string]interface{}{
		"_type": "chunk", // Only return chunks, not document metadata
	}

	// Add user-specified filters
	if len(options.Filter) > 0 {
		for k, v := range options.Filter {
			filters[k] = v
		}
	}

	results, err := b.sqvect.SearchWithFilter(ctx, queryVec, sqvect.SearchOptions{
		TopK:      limit,
		Threshold: float64(options.Threshold),
	}, filters)
	if err != nil {
		return nil, core.NewServiceError("vector", "SearchVectors", "vector search failed", err)
	}

	chunks := make([]VectorSearchHit, len(results))
	maxScore := 0.0
	for i, result := range results {
		// Convert []float32 back to []float64
		resultVector := make([]float64, len(result.Vector))
		for j, v := range result.Vector {
			resultVector[j] = float64(v)
		}

		// Convert metadata back to interface{}
		metadata := make(map[string]interface{})
		for k, v := range result.Metadata {
			if !strings.HasPrefix(k, "_") { // Skip internal metadata
				metadata[k] = v
			}
		}

		// Parse position if available
		position := 0
		if posStr, ok := result.Metadata["_position"]; ok {
			if pos, err := strconv.Atoi(posStr); err == nil {
				position = pos
			}
		}

		vectorChunk := VectorChunk{
			ChunkID:    result.ID,
			DocumentID: result.DocID,
			Content:    result.Content,
			Vector:     resultVector,
			Metadata:   metadata,
			Position:   position,
			CreatedAt:  time.Now(), // We don't store creation time in v0.9.0, use current time
		}

		if !options.IncludeVector {
			vectorChunk.Vector = nil // Remove vector from result if not requested
		}

		score := float64(result.Score)
		if score > maxScore {
			maxScore = score
		}

		chunks[i] = VectorSearchHit{
			VectorChunk: vectorChunk,
			Score:       score,
		}
	}

	result := &VectorSearchResult{
		Chunks:   chunks,
		Total:    len(chunks),
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

	// Ensure sqvect is initialized before using it
	if err := b.ensureInitialized(ctx); err != nil {
		return core.NewServiceError("vector", "DeleteDocument", "failed to initialize storage", err)
	}

	log.Printf("[VECTOR] Deleting vectors for document %s", docID)

	if err := b.sqvect.DeleteByDocID(ctx, docID); err != nil {
		return core.NewServiceError("vector", "DeleteDocument", "failed to delete document vectors", err)
	}

	return nil
}

// GetDocumentVectors retrieves vectors for a specific document.
func (b *SQLiteVectorBackend) GetDocumentVectors(ctx context.Context, docID string) ([]VectorChunk, error) {
	if docID == "" {
		return nil, core.NewValidationError("document_id", docID, "document ID cannot be empty")
	}

	// Ensure sqvect is initialized before using it
	if err := b.ensureInitialized(ctx); err != nil {
		return nil, core.NewServiceError("vector", "GetDocumentVectors", "failed to initialize storage", err)
	}

	// Get embeddings for this document
	embeddings, err := b.sqvect.GetByDocID(ctx, docID)
	if err != nil {
		return nil, core.NewServiceError("vector", "GetDocumentVectors", "failed to get document vectors", err)
	}

	chunks := make([]VectorChunk, 0)
	for _, embedding := range embeddings {
		// Only include chunks, not document metadata
		if embedding.Metadata["_type"] == "chunk" {
			// Convert []float32 back to []float64
			resultVector := make([]float64, len(embedding.Vector))
			for j, v := range embedding.Vector {
				resultVector[j] = float64(v)
			}

			// Convert metadata back to interface{}
			metadata := make(map[string]interface{})
			for k, v := range embedding.Metadata {
				if !strings.HasPrefix(k, "_") { // Skip internal metadata
					metadata[k] = v
				}
			}

			// Parse position if available
			position := 0
			if posStr, ok := embedding.Metadata["_position"]; ok {
				if pos, err := strconv.Atoi(posStr); err == nil {
					position = pos
				}
			}

			chunks = append(chunks, VectorChunk{
				ChunkID:    embedding.ID,
				DocumentID: embedding.DocID,
				Content:    embedding.Content,
				Vector:     resultVector,
				Metadata:   metadata,
				Position:   position,
				CreatedAt:  time.Now(), // We don't store creation time in v0.9.0, use current time
			})
		}
	}

	return chunks, nil
}

// getVectorCount returns the number of vectors in the database.
func (b *SQLiteVectorBackend) getVectorCount(ctx context.Context) (int64, error) {
	// Since sqvect doesn't have a direct Count method, we'll count documents
	documents, err := b.sqvect.ListDocuments(ctx)
	if err != nil {
		return 0, err
	}
	return int64(len(documents)), nil
}

// GetStats returns storage statistics.
func (b *SQLiteVectorBackend) GetStats(ctx context.Context) (*VectorStats, error) {
	// Initialize with defaults
	stats := &VectorStats{
		TotalDocuments: 0,
		TotalVectors:   0,
		StorageSize:    0,
		Dimensions:     b.config.Dimensions,
		Performance:    make(map[string]interface{}),
		LastOptimized:  time.Now(),
	}

	// If not initialized, return empty stats
	if !b.initialized || b.sqvect == nil {
		return stats, nil
	}

	// Try to get actual stats
	if count, err := b.getVectorCount(ctx); err == nil {
		stats.TotalVectors = count
	}

	// Count unique documents by listing and filtering
	if docInfos, err := b.sqvect.ListDocumentsWithInfo(ctx); err == nil {
		stats.TotalDocuments = int64(len(docInfos))
	} else if documents, err := b.sqvect.ListDocuments(ctx); err == nil {
		stats.TotalDocuments = int64(len(documents))
	}

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

	// Ensure sqvect is initialized before using it
	if err := b.ensureInitialized(ctx); err != nil {
		return core.NewServiceError("vector", "Reset", "failed to initialize storage", err)
	}

	// Use the Clear method from sqvect
	if err := b.sqvect.Clear(ctx); err != nil {
		return core.NewServiceError("vector", "Reset", "failed to clear vector store", err)
	}

	return nil
}

// Close closes the backend.
func (b *SQLiteVectorBackend) Close() error {
	log.Printf("[VECTOR] Closing SQLite vector backend")
	if b.sqvect != nil {
		return b.sqvect.Close()
	}
	return nil
}

// GetStore returns the underlying SQLite store for backend integration.
// This is used by the document backend which shares the same SQLite database.
func (b *SQLiteVectorBackend) GetStore() interface{} {
	return b.sqvect
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