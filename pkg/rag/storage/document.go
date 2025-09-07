package storage

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/sqvect"
)

// SQLiteDocumentBackend implements DocumentBackend using sqvect.
type SQLiteDocumentBackend struct {
	sqvect *sqvect.SQLiteStore
	config DocumentConfig
	initialized bool
}

// NewSQLiteDocumentBackend creates a new SQLite document backend.
func NewSQLiteDocumentBackend(config DocumentConfig) (*SQLiteDocumentBackend, error) {
	if config.DBPath == "" {
		return nil, core.NewConfigurationError("storage", "db_path", "database path is required", nil)
	}

	// Don't create the database yet - do it lazily when needed
	return &SQLiteDocumentBackend{
		sqvect:      nil,
		config:      config,
		initialized: false,
	}, nil
}

// ensureInitialized ensures the sqvect client is created and initialized.
func (b *SQLiteDocumentBackend) ensureInitialized(ctx context.Context) error {
	if b.initialized && b.sqvect != nil {
		return nil
	}

	log.Printf("[DOCUMENT] Lazy initializing sqvect database at %s", b.config.DBPath)
	
	// Create sqvect client with dimension 768 (standard for embedding models)
	// This will be shared with vector storage for actual embeddings
	client, err := sqvect.New(b.config.DBPath, 768)
	if err != nil {
		return fmt.Errorf("failed to create sqvect client: %w", err)
	}

	// Initialize the database
	if err := client.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize sqvect: %w", err)
	}

	b.sqvect = client
	b.initialized = true
	log.Printf("[DOCUMENT] Successfully initialized sqvect database")
	
	return nil
}

// StoreDocument stores document metadata.
func (b *SQLiteDocumentBackend) StoreDocument(ctx context.Context, doc *Document) error {
	if doc == nil {
		return core.NewValidationError("document", doc, "document cannot be nil")
	}

	if doc.ID == "" {
		return core.NewValidationError("document.id", doc.ID, "document ID cannot be empty")
	}

	// Ensure sqvect is initialized before using it
	if err := b.ensureInitialized(ctx); err != nil {
		return core.NewServiceError("document", "StoreDocument", "failed to initialize storage", err)
	}

	log.Printf("[DOCUMENT] Storing document %s", doc.ID)

	// Convert metadata to string map for sqvect
	metadata := make(map[string]string)
	if doc.Metadata != nil {
		for k, v := range doc.Metadata {
			metadata[k] = fmt.Sprintf("%v", v)
		}
	}

	// Add document type and timestamps
	metadata["_type"] = "document"
	metadata["_path"] = doc.FilePath
	metadata["_url"] = doc.URL
	metadata["_created"] = doc.CreatedAt.Format("2006-01-02T15:04:05Z07:00")

	// Store as embedding with a zero vector matching the database dimensions
	// Document metadata doesn't have real embeddings, use zero vector as placeholder
	zeroVector := make([]float32, 768) // Match the 768 dimensions
	embedding := &sqvect.Embedding{
		ID:       doc.ID,
		Vector:   zeroVector, // Zero vector for document metadata
		Content:  doc.Content,
		DocID:    doc.ID,
		Metadata: metadata,
	}

	if err := b.sqvect.Upsert(ctx, embedding); err != nil {
		return core.NewServiceError("document", "StoreDocument", "failed to store document metadata", err)
	}

	return nil
}

// GetDocument retrieves a document by ID.
func (b *SQLiteDocumentBackend) GetDocument(ctx context.Context, docID string) (*Document, error) {
	if docID == "" {
		return nil, core.NewValidationError("document_id", docID, "document ID cannot be empty")
	}

	// Ensure sqvect is initialized before using it
	if err := b.ensureInitialized(ctx); err != nil {
		return nil, core.NewServiceError("document", "GetDocument", "failed to initialize storage", err)
	}

	// Get embeddings for this document
	embeddings, err := b.sqvect.GetByDocID(ctx, docID)
	if err != nil {
		return nil, core.NewServiceError("document", "GetDocument", "failed to get document embeddings", err)
	}

	// Find the document metadata embedding
	for _, embedding := range embeddings {
		if embedding.Metadata["_type"] == "document" && embedding.DocID == docID {
			// Parse created time
			var created time.Time
			if createdStr, ok := embedding.Metadata["_created"]; ok {
				if parsed, err := time.Parse("2006-01-02T15:04:05Z07:00", createdStr); err == nil {
					created = parsed
				}
			}

			doc := &Document{
				ID:          docID,
				Title:       embedding.Metadata["title"],
				Content:     embedding.Content,
				ContentType: embedding.Metadata["content_type"],
				FilePath:    embedding.Metadata["_path"],
				URL:         embedding.Metadata["_url"],
				Size:        int64(len(embedding.Content)),
				CreatedAt:   created,
				UpdatedAt:   created, // For now, same as created
				Version:     1,
			}

			// Copy non-internal metadata
			doc.Metadata = make(map[string]interface{})
			for k, v := range embedding.Metadata {
				if !strings.HasPrefix(k, "_") { // Skip internal metadata
					doc.Metadata[k] = v
				}
			}

			return doc, nil
		}
	}

	return nil, core.NewServiceError("document", "GetDocument", "document not found", nil)
}

// ListDocuments lists documents with optional filtering.
func (b *SQLiteDocumentBackend) ListDocuments(ctx context.Context, filter DocumentFilter) ([]Document, error) {
	// Ensure sqvect is initialized before using it
	if err := b.ensureInitialized(ctx); err != nil {
		return nil, core.NewServiceError("document", "ListDocuments", "failed to initialize storage", err)
	}

	// Get all documents using optimized approach
	documents, err := b.listWithOptimizedAPI(ctx)
	if err != nil {
		return nil, err
	}

	// Apply filters
	filteredDocuments := make([]Document, 0)
	for _, doc := range documents {
		if b.matchesFilter(doc, filter) {
			filteredDocuments = append(filteredDocuments, doc)
		}
	}

	// Apply limit and offset
	if filter.Limit > 0 {
		start := filter.Offset
		end := start + filter.Limit
		if start < len(filteredDocuments) {
			if end > len(filteredDocuments) {
				end = len(filteredDocuments)
			}
			filteredDocuments = filteredDocuments[start:end]
		} else {
			filteredDocuments = []Document{}
		}
	}

	return filteredDocuments, nil
}

// UpdateDocument updates document metadata.
func (b *SQLiteDocumentBackend) UpdateDocument(ctx context.Context, docID string, updates DocumentUpdate) error {
	if docID == "" {
		return core.NewValidationError("document_id", docID, "document ID cannot be empty")
	}

	log.Printf("[DOCUMENT] Updating document %s", docID)

	// Get existing document
	existing, err := b.GetDocument(ctx, docID)
	if err != nil {
		return err
	}

	// Apply updates
	updated := *existing
	if updates.Title != nil {
		updated.Title = *updates.Title
	}
	if updates.ContentType != nil {
		updated.ContentType = *updates.ContentType
	}
	if updates.Metadata != nil {
		// Merge metadata
		if updated.Metadata == nil {
			updated.Metadata = make(map[string]interface{})
		}
		for k, v := range updates.Metadata {
			updated.Metadata[k] = v
		}
	}

	updated.UpdatedAt = time.Now()
	updated.Version++

	// Store updated document
	return b.StoreDocument(ctx, &updated)
}

// DeleteDocument removes a document.
func (b *SQLiteDocumentBackend) DeleteDocument(ctx context.Context, docID string) error {
	if docID == "" {
		return core.NewValidationError("document_id", docID, "document ID cannot be empty")
	}

	// Ensure sqvect is initialized before using it
	if err := b.ensureInitialized(ctx); err != nil {
		return core.NewServiceError("document", "DeleteDocument", "failed to initialize storage", err)
	}

	log.Printf("[DOCUMENT] Deleting document %s", docID)

	// Delete all embeddings for this document (both chunks and metadata)
	if err := b.sqvect.DeleteByDocID(ctx, docID); err != nil {
		return core.NewServiceError("document", "DeleteDocument", "failed to delete document", err)
	}

	return nil
}

// GetStats returns document storage statistics.
func (b *SQLiteDocumentBackend) GetStats(ctx context.Context) (*DocumentStats, error) {
	// Initialize with defaults
	stats := &DocumentStats{
		TotalDocuments: 0,
		TotalSize:      0,
		AverageSize:    0,
		ByContentType:  make(map[string]int64),
		LastUpdated:    time.Now(),
	}

	// If not initialized, return empty stats
	if !b.initialized || b.sqvect == nil {
		return stats, nil
	}

	// Try to get actual documents to calculate stats
	documents, err := b.listWithOptimizedAPI(ctx)
	if err != nil {
		// Return default stats if we can't get documents
		return stats, nil
	}

	totalSize := int64(0)
	contentTypeCount := make(map[string]int64)

	for _, doc := range documents {
		totalSize += doc.Size
		if doc.ContentType != "" {
			contentTypeCount[doc.ContentType]++
		} else {
			contentTypeCount["unknown"]++
		}
	}

	stats.TotalDocuments = int64(len(documents))
	stats.TotalSize = totalSize
	if len(documents) > 0 {
		stats.AverageSize = float64(totalSize) / float64(len(documents))
	}
	stats.ByContentType = contentTypeCount

	return stats, nil
}

// Optimize performs storage optimization.
func (b *SQLiteDocumentBackend) Optimize(ctx context.Context) error {
	log.Printf("[DOCUMENT] Performing optimization")

	// Ensure sqvect is initialized before using it
	if err := b.ensureInitialized(ctx); err != nil {
		return core.NewServiceError("document", "Optimize", "failed to initialize storage", err)
	}

	// SQLite auto-optimizes, but we could implement VACUUM or other optimizations here
	// For now, this is a no-op since sqvect doesn't expose VACUUM directly
	return nil
}

// Reset clears all documents.
func (b *SQLiteDocumentBackend) Reset(ctx context.Context) error {
	log.Printf("[DOCUMENT] Resetting document store")
	// The document store uses the same underlying SQLite database
	// Reset is handled at the vector store level
	return nil
}

// Close closes the backend.
func (b *SQLiteDocumentBackend) Close() error {
	log.Printf("[DOCUMENT] Closing SQLite document backend")
	if b.sqvect != nil {
		return b.sqvect.Close()
	}
	return nil
}

// matchesFilter checks if a document matches the given filter criteria.
func (b *SQLiteDocumentBackend) matchesFilter(doc Document, filter DocumentFilter) bool {
	// Content type filter
	if len(filter.ContentTypes) > 0 {
		matched := false
		for _, ct := range filter.ContentTypes {
			if doc.ContentType == ct {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Metadata filter
	if len(filter.Metadata) > 0 {
		for key, expectedValue := range filter.Metadata {
			if actualValue, exists := doc.Metadata[key]; !exists || actualValue != expectedValue {
				return false
			}
		}
	}

	// Date filters
	if filter.CreatedAfter != nil && doc.CreatedAt.Before(*filter.CreatedAfter) {
		return false
	}
	if filter.CreatedBefore != nil && doc.CreatedAt.After(*filter.CreatedBefore) {
		return false
	}
	if filter.UpdatedAfter != nil && doc.UpdatedAt.Before(*filter.UpdatedAfter) {
		return false
	}
	if filter.UpdatedBefore != nil && doc.UpdatedAt.After(*filter.UpdatedBefore) {
		return false
	}

	// Size filters
	if filter.MinSize > 0 && doc.Size < filter.MinSize {
		return false
	}
	if filter.MaxSize > 0 && doc.Size > filter.MaxSize {
		return false
	}

	return true
}

// listWithOptimizedAPI uses the optimized sqvect API to list documents.
func (b *SQLiteDocumentBackend) listWithOptimizedAPI(ctx context.Context) ([]Document, error) {
	// Try using ListDocumentsWithInfo first (sqvect v0.9.0)
	docInfos, err := b.sqvect.ListDocumentsWithInfo(ctx)
	if err != nil {
		// Fall back to old method if ListDocumentsWithInfo is not available
		return b.listWithFallback(ctx)
	}

	if len(docInfos) == 0 {
		return []Document{}, nil
	}

	documents := make([]Document, 0, len(docInfos))

	for _, docInfo := range docInfos {
		// Get the document details using GetByDocID
		embeddings, err := b.sqvect.GetByDocID(ctx, docInfo.DocID)
		if err != nil {
			continue // Skip this document if we can't get its embeddings
		}

		// Find the document metadata embedding
		for _, embedding := range embeddings {
			if embedding.Metadata["_type"] == "document" {
				// Parse created time
				var created time.Time
				if createdStr, ok := embedding.Metadata["_created"]; ok {
					if parsed, err := time.Parse("2006-01-02T15:04:05Z07:00", createdStr); err == nil {
						created = parsed
					}
				}

				doc := Document{
					ID:          embedding.DocID,
					Title:       embedding.Metadata["title"],
					Content:     embedding.Content,
					ContentType: embedding.Metadata["content_type"],
					FilePath:    embedding.Metadata["_path"],
					URL:         embedding.Metadata["_url"],
					Size:        int64(len(embedding.Content)),
					CreatedAt:   created,
					UpdatedAt:   created, // For now, same as created
					Version:     1,
				}

				// Copy non-internal metadata
				doc.Metadata = make(map[string]interface{})
				for k, v := range embedding.Metadata {
					if !strings.HasPrefix(k, "_") {
						doc.Metadata[k] = v
					}
				}

				documents = append(documents, doc)
				break
			}
		}
	}

	return documents, nil
}

// listWithFallback is a fallback method for compatibility.
func (b *SQLiteDocumentBackend) listWithFallback(ctx context.Context) ([]Document, error) {
	// Try GetDocumentsByType
	embeddings, err := b.sqvect.GetDocumentsByType(ctx, "document")
	if err != nil {
		// Final fallback: ListDocuments + GetByDocID
		return b.listWithBasicAPI(ctx)
	}

	if len(embeddings) == 0 {
		return b.listWithBasicAPI(ctx)
	}

	// Process GetDocumentsByType results
	documentMap := make(map[string]Document)

	for _, embedding := range embeddings {
		if embedding.Metadata["_type"] == "document" {
			// Parse created time
			var created time.Time
			if createdStr, ok := embedding.Metadata["_created"]; ok {
				if parsed, err := time.Parse("2006-01-02T15:04:05Z07:00", createdStr); err == nil {
					created = parsed
				}
			}

			doc := Document{
				ID:          embedding.DocID,
				Title:       embedding.Metadata["title"],
				Content:     embedding.Content,
				ContentType: embedding.Metadata["content_type"],
				FilePath:    embedding.Metadata["_path"],
				URL:         embedding.Metadata["_url"],
				Size:        int64(len(embedding.Content)),
				CreatedAt:   created,
				UpdatedAt:   created,
				Version:     1,
			}

			// Copy non-internal metadata
			doc.Metadata = make(map[string]interface{})
			for k, v := range embedding.Metadata {
				if !strings.HasPrefix(k, "_") {
					doc.Metadata[k] = v
				}
			}

			documentMap[embedding.DocID] = doc
		}
	}

	// Convert map to slice
	documents := make([]Document, 0, len(documentMap))
	for _, doc := range documentMap {
		documents = append(documents, doc)
	}

	return documents, nil
}

// listWithBasicAPI uses basic ListDocuments + GetByDocID.
func (b *SQLiteDocumentBackend) listWithBasicAPI(ctx context.Context) ([]Document, error) {
	docIDs, err := b.sqvect.ListDocuments(ctx)
	if err != nil {
		return nil, core.NewServiceError("document", "listWithBasicAPI", "failed to list documents", err)
	}

	documentMap := make(map[string]Document)

	for _, docID := range docIDs {
		docEmbeddings, err := b.sqvect.GetByDocID(ctx, docID)
		if err != nil {
			continue // Skip this document if we can't get its embeddings
		}

		// Find the document metadata embedding
		for _, embedding := range docEmbeddings {
			if embedding.Metadata["_type"] == "document" {
				// Parse created time
				var created time.Time
				if createdStr, ok := embedding.Metadata["_created"]; ok {
					if parsed, err := time.Parse("2006-01-02T15:04:05Z07:00", createdStr); err == nil {
						created = parsed
					}
				}

				doc := Document{
					ID:          embedding.DocID,
					Title:       embedding.Metadata["title"],
					Content:     embedding.Content,
					ContentType: embedding.Metadata["content_type"],
					FilePath:    embedding.Metadata["_path"],
					URL:         embedding.Metadata["_url"],
					Size:        int64(len(embedding.Content)),
					CreatedAt:   created,
					UpdatedAt:   created,
					Version:     1,
				}

				// Copy non-internal metadata
				doc.Metadata = make(map[string]interface{})
				for k, v := range embedding.Metadata {
					if !strings.HasPrefix(k, "_") {
						doc.Metadata[k] = v
					}
				}

				documentMap[embedding.DocID] = doc
				break
			}
		}
	}

	// Convert map to slice
	documents := make([]Document, 0, len(documentMap))
	for _, doc := range documentMap {
		documents = append(documents, doc)
	}

	return documents, nil
}

// ===== FACTORY FUNCTION =====

// NewDocumentBackend creates a document backend based on the configuration.
func NewDocumentBackend(config DocumentConfig) (DocumentBackend, error) {
	switch config.Backend {
	case "sqlite", "sqvect":
		return NewSQLiteDocumentBackend(config)
	// TODO: Add other backends like PostgreSQL, MongoDB, etc.
	// case "postgres":
	//     return NewPostgresDocumentBackend(config)
	// case "mongodb":
	//     return NewMongoDocumentBackend(config)
	default:
		return nil, core.NewConfigurationError("storage", "backend", 
			fmt.Sprintf("unsupported document backend: %s", config.Backend), nil)
	}
}