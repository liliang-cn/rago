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

// BleveKeywordBackend implements KeywordBackend using the existing Bleve store.
type BleveKeywordBackend struct {
	store  *store.KeywordStore
	config KeywordConfig
}

// NewBleveKeywordBackend creates a new Bleve keyword backend.
func NewBleveKeywordBackend(config KeywordConfig) (*BleveKeywordBackend, error) {
	if config.IndexPath == "" {
		return nil, core.NewConfigurationError("storage", "index_path", "index path is required", nil)
	}

	store, err := store.NewKeywordStore(config.IndexPath)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "storage", "NewBleveKeywordBackend", "failed to create keyword store")
	}

	return &BleveKeywordBackend{
		store:  store,
		config: config,
	}, nil
}

// IndexChunks indexes text chunks for full-text search.
func (b *BleveKeywordBackend) IndexChunks(ctx context.Context, docID string, chunks []KeywordChunk) error {
	if len(chunks) == 0 {
		return nil
	}

	log.Printf("[KEYWORD] Indexing %d text chunks for document %s", len(chunks), docID)

	// Convert to domain chunks for the existing store
	for _, chunk := range chunks {
		domainChunk := domain.Chunk{
			ID:         chunk.ChunkID,
			DocumentID: chunk.DocumentID,
			Content:    chunk.Content,
			Metadata:   chunk.Metadata,
		}

		if err := b.store.Index(ctx, domainChunk); err != nil {
			return core.WrapErrorWithContext(err, "storage", "IndexChunks", 
				fmt.Sprintf("failed to index chunk %s", chunk.ChunkID))
		}
	}

	return nil
}

// SearchKeywords performs full-text search.
func (b *BleveKeywordBackend) SearchKeywords(ctx context.Context, query string, options KeywordSearchOptions) (*KeywordSearchResult, error) {
	if query == "" {
		return nil, core.NewValidationError("query", query, "query cannot be empty")
	}

	// Set defaults
	limit := options.Limit
	if limit <= 0 {
		limit = 10
	}

	chunks, err := b.store.Search(ctx, query, limit)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "storage", "SearchKeywords", "keyword search failed")
	}

	// Convert results
	hits := make([]KeywordSearchHit, len(chunks))
	var maxScore float64

	for i, chunk := range chunks {
		hit := KeywordSearchHit{
			KeywordChunk: KeywordChunk{
				ChunkID:    chunk.ID,
				DocumentID: chunk.DocumentID,
				Content:    chunk.Content,
				Metadata:   chunk.Metadata,
				CreatedAt:  time.Now(), // TODO: Get actual created time
			},
			Score: chunk.Score,
		}

		// Generate highlights if requested
		if options.Highlight {
			hit.Highlights = b.generateHighlights(chunk.Content, query)
		}

		hits[i] = hit

		if chunk.Score > maxScore {
			maxScore = chunk.Score
		}
	}

	return &KeywordSearchResult{
		Chunks:   hits,
		Total:    len(hits), // TODO: Get actual total count from Bleve
		MaxScore: maxScore,
		Query:    query,
	}, nil
}

// DeleteDocument removes all indexed content for a document.
func (b *BleveKeywordBackend) DeleteDocument(ctx context.Context, docID string) error {
	if docID == "" {
		return core.NewValidationError("document_id", docID, "document ID cannot be empty")
	}

	return b.store.Delete(ctx, docID)
}

// GetDocumentContent retrieves indexed content for a document.
func (b *BleveKeywordBackend) GetDocumentContent(ctx context.Context, docID string) ([]KeywordChunk, error) {
	if docID == "" {
		return nil, core.NewValidationError("document_id", docID, "document ID cannot be empty")
	}

	// TODO: Implement document-specific content retrieval
	// The current Bleve store doesn't have a GetByDocID method
	// We would need to search for all chunks with the document ID
	
	// For now, return empty result
	return []KeywordChunk{}, nil
}

// GetStats returns indexing statistics.
func (b *BleveKeywordBackend) GetStats(ctx context.Context) (*KeywordStats, error) {
	// TODO: Get actual stats from Bleve index
	// The current KeywordStore doesn't expose statistics
	// We need to enhance it or query Bleve directly

	stats := &KeywordStats{
		Performance: make(map[string]interface{}),
	}

	return stats, nil
}

// Optimize performs index optimization.
func (b *BleveKeywordBackend) Optimize(ctx context.Context) error {
	log.Printf("[KEYWORD] Performing index optimization")
	// TODO: Implement Bleve index optimization
	// This could involve index compaction, segment merging, etc.
	return nil
}

// Reset clears all indexed data.
func (b *BleveKeywordBackend) Reset(ctx context.Context) error {
	log.Printf("[KEYWORD] Resetting keyword index")
	return b.store.Reset(ctx)
}

// Close closes the backend.
func (b *BleveKeywordBackend) Close() error {
	log.Printf("[KEYWORD] Closing Bleve keyword backend")
	return b.store.Close()
}

// generateHighlights creates highlighted excerpts for search results.
func (b *BleveKeywordBackend) generateHighlights(content, query string) []string {
	// TODO: Implement proper highlighting
	// This is a simplified version - in practice, we'd want to:
	// 1. Parse the query into terms
	// 2. Find matching positions in content
	// 3. Create contextual excerpts with highlighted terms
	// 4. Handle phrase queries, wildcards, etc.

	// For now, return the first part of the content
	const maxHighlightLength = 200
	if len(content) <= maxHighlightLength {
		return []string{content}
	}

	return []string{content[:maxHighlightLength] + "..."}
}

// ===== FACTORY FUNCTION =====

// NewKeywordBackend creates a keyword backend based on the configuration.
func NewKeywordBackend(config KeywordConfig) (KeywordBackend, error) {
	switch config.Backend {
	case "bleve":
		return NewBleveKeywordBackend(config)
	// TODO: Add other backends like Elasticsearch, Tantivy, etc.
	// case "elasticsearch":
	//     return NewElasticsearchKeywordBackend(config)
	// case "tantivy":
	//     return NewTantivyKeywordBackend(config)
	default:
		return nil, core.NewConfigurationError("storage", "backend", 
			fmt.Sprintf("unsupported keyword backend: %s", config.Backend), nil)
	}
}