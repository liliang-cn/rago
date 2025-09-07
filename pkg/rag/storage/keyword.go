package storage

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

// BleveKeywordBackend implements KeywordBackend using V3 architecture.
type BleveKeywordBackend struct {
	index       bleve.Index
	config      KeywordConfig
	initialized bool
}

// NewBleveKeywordBackend creates a new Bleve keyword backend.
func NewBleveKeywordBackend(config KeywordConfig) (*BleveKeywordBackend, error) {
	if config.IndexPath == "" {
		return nil, core.NewConfigurationError("storage", "index_path", "index path is required", nil)
	}

	// Don't create the index yet - do it lazily when needed
	return &BleveKeywordBackend{
		index:       nil,
		config:      config,
		initialized: false,
	}, nil
}

// ensureInitialized ensures the Bleve index is created and initialized.
func (b *BleveKeywordBackend) ensureInitialized(ctx context.Context) error {
	if b.initialized && b.index != nil {
		return nil
	}

	log.Printf("[KEYWORD] Lazy initializing Bleve index at %s", b.config.IndexPath)
	
	index, err := b.openOrCreateBleveIndex(b.config.IndexPath)
	if err != nil {
		return fmt.Errorf("failed to create Bleve index: %w", err)
	}

	b.index = index
	b.initialized = true
	log.Printf("[KEYWORD] Successfully initialized Bleve index")
	
	return nil
}

// openOrCreateBleveIndex handles the logic of opening an existing index or creating a new one.
func (b *BleveKeywordBackend) openOrCreateBleveIndex(path string) (bleve.Index, error) {
	// Check if the index already exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Create a new index with a standard mapping
		mapping := bleve.NewIndexMapping()
		// Customize mapping based on configuration
		if b.config.Analyzer != "" {
			mapping.DefaultAnalyzer = b.config.Analyzer
		}
		index, err := bleve.New(path, mapping)
		if err != nil {
			return nil, err
		}
		return index, nil
	}

	// Open the existing index
	index, err := bleve.Open(path)
	if err != nil {
		return nil, err
	}
	return index, nil
}

// IndexChunks indexes text chunks for full-text search.
func (b *BleveKeywordBackend) IndexChunks(ctx context.Context, docID string, chunks []KeywordChunk) error {
	if len(chunks) == 0 {
		return nil
	}

	// Ensure Bleve is initialized before using it
	if err := b.ensureInitialized(ctx); err != nil {
		return core.NewServiceError("keyword", "IndexChunks", "failed to initialize index", err)
	}

	log.Printf("[KEYWORD] Indexing %d text chunks for document %s", len(chunks), docID)

	// Create a batch for efficient indexing
	batch := b.index.NewBatch()

	for _, chunk := range chunks {
		// Create a document for Bleve with the chunk data
		doc := map[string]interface{}{
			"chunk_id":    chunk.ChunkID,
			"document_id": chunk.DocumentID,
			"content":     chunk.Content,
			"title":       chunk.Title,
			"position":    chunk.Position,
			"created_at":  chunk.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		// Add metadata
		if chunk.Metadata != nil {
			for k, v := range chunk.Metadata {
				doc[k] = v
			}
		}

		// Add to batch
		if err := batch.Index(chunk.ChunkID, doc); err != nil {
			return core.NewServiceError("keyword", "IndexChunks", "failed to add chunk to batch", err)
		}
	}

	// Execute the batch
	if err := b.index.Batch(batch); err != nil {
		return core.NewServiceError("keyword", "IndexChunks", "failed to execute batch index", err)
	}

	return nil
}

// SearchKeywords performs full-text search.
func (b *BleveKeywordBackend) SearchKeywords(ctx context.Context, query string, options KeywordSearchOptions) (*KeywordSearchResult, error) {
	if query == "" {
		return nil, core.NewValidationError("query", query, "query cannot be empty")
	}

	// Ensure Bleve is initialized before using it
	if err := b.ensureInitialized(ctx); err != nil {
		return nil, core.NewServiceError("keyword", "SearchKeywords", "failed to initialize index", err)
	}

	// Set defaults
	limit := options.Limit
	if limit <= 0 {
		limit = 10
	}

	// Create search request with the appropriate query
	var searchRequest *bleve.SearchRequest
	if options.Fuzzy {
		// Use fuzzy search
		fuzzyQuery := bleve.NewFuzzyQuery(query)
		searchRequest = bleve.NewSearchRequest(fuzzyQuery)
	} else {
		// Use match query for better relevance
		matchQuery := bleve.NewMatchQuery(query)
		searchRequest = bleve.NewSearchRequest(matchQuery)
	}
	searchRequest.Size = limit
	searchRequest.From = options.Offset
	searchRequest.Fields = []string{"*"} // Request all stored fields

	// Add highlighting if requested
	if options.Highlight {
		searchRequest.Highlight = bleve.NewHighlight()
		searchRequest.Highlight.AddField("content")
		searchRequest.Highlight.AddField("title")
	}

	// Execute the search
	searchResult, err := b.index.Search(searchRequest)
	if err != nil {
		return nil, core.NewServiceError("keyword", "SearchKeywords", "search failed", err)
	}

	// Process the results
	chunks := make([]KeywordSearchHit, len(searchResult.Hits))
	maxScore := 0.0

	for i, hit := range searchResult.Hits {
		// Safely extract fields with type checking
		chunk := KeywordChunk{
			ChunkID:    hit.ID,
			DocumentID: b.extractStringField(hit.Fields, "document_id"),
			Content:    b.extractStringField(hit.Fields, "content"),
			Title:      b.extractStringField(hit.Fields, "title"),
			Position:   b.extractIntField(hit.Fields, "position"),
			CreatedAt:  b.extractTimeField(hit.Fields, "created_at"),
		}

		// Extract metadata (skip known system fields)
		metadata := make(map[string]interface{})
		for k, v := range hit.Fields {
			if !b.isSystemField(k) {
				metadata[k] = v
			}
		}
		chunk.Metadata = metadata

		// Extract highlights
		highlights := make([]string, 0)
		if hit.Fragments != nil {
			for field, fragments := range hit.Fragments {
				for _, fragment := range fragments {
					highlights = append(highlights, fmt.Sprintf("%s: %s", field, fragment))
				}
			}
		}
		if len(highlights) == 0 && options.Highlight {
			// Generate basic highlights if Bleve didn't provide any
			highlights = b.generateHighlights(chunk.Content, query)
		}

		if hit.Score > maxScore {
			maxScore = hit.Score
		}

		chunks[i] = KeywordSearchHit{
			KeywordChunk: chunk,
			Score:        hit.Score,
			Highlights:   highlights,
			Context:      make(map[string]string), // Could be enhanced with more context
		}
	}

	return &KeywordSearchResult{
		Chunks:   chunks,
		Total:    int(searchResult.Total),
		MaxScore: maxScore,
		Query:    query,
	}, nil
}

// DeleteDocument removes all indexed content for a document.
func (b *BleveKeywordBackend) DeleteDocument(ctx context.Context, docID string) error {
	if docID == "" {
		return core.NewValidationError("document_id", docID, "document ID cannot be empty")
	}

	// Ensure Bleve is initialized before using it
	if err := b.ensureInitialized(ctx); err != nil {
		return core.NewServiceError("keyword", "DeleteDocument", "failed to initialize index", err)
	}

	log.Printf("[KEYWORD] Deleting indexed content for document %s", docID)

	// Find all chunks for this document
	query := bleve.NewTermQuery(docID)
	query.SetField("document_id")

	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Size = 1000 // Adjust size as needed
	searchResult, err := b.index.Search(searchRequest)
	if err != nil {
		return core.NewServiceError("keyword", "DeleteDocument", "failed to find chunks for deletion", err)
	}

	// Delete all found chunks
	batch := b.index.NewBatch()
	for _, hit := range searchResult.Hits {
		batch.Delete(hit.ID)
	}

	if err := b.index.Batch(batch); err != nil {
		return core.NewServiceError("keyword", "DeleteDocument", "failed to delete chunks", err)
	}

	return nil
}

// GetDocumentContent retrieves indexed content for a document.
func (b *BleveKeywordBackend) GetDocumentContent(ctx context.Context, docID string) ([]KeywordChunk, error) {
	if docID == "" {
		return nil, core.NewValidationError("document_id", docID, "document ID cannot be empty")
	}

	// Ensure Bleve is initialized before using it
	if err := b.ensureInitialized(ctx); err != nil {
		return nil, core.NewServiceError("keyword", "GetDocumentContent", "failed to initialize index", err)
	}

	// Search for all chunks with this document ID
	query := bleve.NewTermQuery(docID)
	query.SetField("document_id")

	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Size = 1000 // Large enough to get all chunks
	searchRequest.Fields = []string{"*"}

	searchResult, err := b.index.Search(searchRequest)
	if err != nil {
		return nil, core.NewServiceError("keyword", "GetDocumentContent", "failed to search for document chunks", err)
	}

	// Convert hits to KeywordChunk objects
	chunks := make([]KeywordChunk, len(searchResult.Hits))
	for i, hit := range searchResult.Hits {
		chunks[i] = KeywordChunk{
			ChunkID:    hit.ID,
			DocumentID: b.extractStringField(hit.Fields, "document_id"),
			Content:    b.extractStringField(hit.Fields, "content"),
			Title:      b.extractStringField(hit.Fields, "title"),
			Position:   b.extractIntField(hit.Fields, "position"),
			CreatedAt:  b.extractTimeField(hit.Fields, "created_at"),
			Metadata:   make(map[string]interface{}), // Could extract if needed
		}
	}

	return chunks, nil
}

// GetStats returns indexing statistics.
func (b *BleveKeywordBackend) GetStats(ctx context.Context) (*KeywordStats, error) {
	// Initialize with defaults
	stats := &KeywordStats{
		TotalChunks:    0,
		TotalDocuments: 0,
		IndexSize:      0,
		TermsCount:     0,
		LastOptimized:  time.Now(),
		Performance:    make(map[string]interface{}),
	}

	// If not initialized, return empty stats
	if !b.initialized || b.index == nil {
		return stats, nil
	}

	// Get basic statistics from Bleve
	indexStats := b.index.Stats()
	if indexStats != nil {
		stats.Performance["index_stats"] = indexStats
	}

	// Try to estimate document count by searching
	// This is not very efficient but gives us some stats
	allQuery := bleve.NewMatchAllQuery()
	allRequest := bleve.NewSearchRequest(allQuery)
	allRequest.Size = 0 // We only want the count

	if result, err := b.index.Search(allRequest); err == nil {
		stats.TotalChunks = int64(result.Total)
	}

	// Count unique documents by searching for distinct document_ids
	// This is approximate - in a real implementation we'd need better analytics
	docQuery := bleve.NewMatchAllQuery()
	docRequest := bleve.NewSearchRequest(docQuery)
	docRequest.Size = 1000
	docRequest.Fields = []string{"document_id"}

	if result, err := b.index.Search(docRequest); err == nil {
		uniqueDocIDs := make(map[string]bool)
		for _, hit := range result.Hits {
			if docID := b.extractStringField(hit.Fields, "document_id"); docID != "" {
				uniqueDocIDs[docID] = true
			}
		}
		stats.TotalDocuments = int64(len(uniqueDocIDs))
	}

	return stats, nil
}

// Optimize performs index optimization.
func (b *BleveKeywordBackend) Optimize(ctx context.Context) error {
	log.Printf("[KEYWORD] Performing index optimization")

	// Ensure Bleve is initialized before using it
	if err := b.ensureInitialized(ctx); err != nil {
		return core.NewServiceError("keyword", "Optimize", "failed to initialize index", err)
	}

	// Bleve doesn't have explicit optimize methods like Elasticsearch
	// But we can perform some maintenance operations
	// For now, this is a no-op but could be enhanced with:
	// - Index compaction
	// - Segment merging
	// - Cache cleanup
	return nil
}

// Reset clears all indexed data.
func (b *BleveKeywordBackend) Reset(ctx context.Context) error {
	log.Printf("[KEYWORD] Resetting keyword index")

	// Ensure Bleve is initialized before using it
	if err := b.ensureInitialized(ctx); err != nil {
		return core.NewServiceError("keyword", "Reset", "failed to initialize index", err)
	}

	// Close the current index
	if err := b.index.Close(); err != nil {
		// Log error but continue, as we are about to delete the directory anyway
		log.Printf("Warning: failed to close index during reset: %v", err)
	}

	// Remove the index directory
	if err := os.RemoveAll(b.config.IndexPath); err != nil {
		return core.NewServiceError("keyword", "Reset", "failed to remove index directory", err)
	}

	// Create a new index
	newIndex, err := b.openOrCreateBleveIndex(b.config.IndexPath)
	if err != nil {
		return core.NewServiceError("keyword", "Reset", "failed to create new index", err)
	}

	b.index = newIndex
	return nil
}

// Close closes the backend.
func (b *BleveKeywordBackend) Close() error {
	log.Printf("[KEYWORD] Closing Bleve keyword backend")
	if b.index != nil {
		err := b.index.Close()
		b.index = nil // Prevent double close
		return err
	}
	return nil
}

// Helper methods for field extraction from Bleve search results

// extractStringField safely extracts a string field from Bleve hit fields.
func (b *BleveKeywordBackend) extractStringField(fields map[string]interface{}, fieldName string) string {
	if value, ok := fields[fieldName]; ok && value != nil {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// extractIntField safely extracts an int field from Bleve hit fields.
func (b *BleveKeywordBackend) extractIntField(fields map[string]interface{}, fieldName string) int {
	if value, ok := fields[fieldName]; ok && value != nil {
		switch v := value.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case string:
			if i, err := fmt.Sscanf(v, "%d", new(int)); err == nil && i == 1 {
				var result int
				fmt.Sscanf(v, "%d", &result)
				return result
			}
		}
	}
	return 0
}

// extractTimeField safely extracts a time field from Bleve hit fields.
func (b *BleveKeywordBackend) extractTimeField(fields map[string]interface{}, fieldName string) time.Time {
	if value, ok := fields[fieldName]; ok && value != nil {
		if str, ok := value.(string); ok {
			if t, err := time.Parse("2006-01-02T15:04:05Z07:00", str); err == nil {
				return t
			}
		}
	}
	return time.Now() // Default to current time
}

// isSystemField checks if a field is a system field that should not be included in metadata.
func (b *BleveKeywordBackend) isSystemField(fieldName string) bool {
	systemFields := []string{
		"chunk_id", "document_id", "content", "title", "position", "created_at",
	}
	for _, field := range systemFields {
		if field == fieldName {
			return true
		}
	}
	return false
}

// generateHighlights creates highlighted excerpts for search results.
func (b *BleveKeywordBackend) generateHighlights(content, query string) []string {
	// This is a simplified version - in practice, we'd want to:
	// 1. Parse the query into terms
	// 2. Find matching positions in content
	// 3. Create contextual excerpts with highlighted terms
	// 4. Handle phrase queries, wildcards, etc.

	const maxHighlightLength = 200
	
	// Try to find query terms in content and create contextual highlights
	queryTerms := strings.Fields(strings.ToLower(query))
	contentLower := strings.ToLower(content)
	
	highlights := make([]string, 0)
	
	for _, term := range queryTerms {
		if pos := strings.Index(contentLower, term); pos >= 0 {
			// Create a context around the found term
			start := pos - 50
			if start < 0 {
				start = 0
			}
			end := pos + len(term) + 150
			if end > len(content) {
				end = len(content)
			}
			
			excerpt := content[start:end]
			if start > 0 {
				excerpt = "..." + excerpt
			}
			if end < len(content) {
				excerpt = excerpt + "..."
			}
			
			highlights = append(highlights, excerpt)
			break // Just take the first match for now
		}
	}
	
	// Fallback: return the first part of the content
	if len(highlights) == 0 {
		if len(content) <= maxHighlightLength {
			highlights = append(highlights, content)
		} else {
			highlights = append(highlights, content[:maxHighlightLength]+"...")
		}
	}
	
	return highlights
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