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

// SQLiteDocumentBackend implements DocumentBackend using the existing SQLite store.
type SQLiteDocumentBackend struct {
	store  *store.DocumentStore
	config DocumentConfig
}

// NewSQLiteDocumentBackend creates a new SQLite document backend.
func NewSQLiteDocumentBackend(config DocumentConfig, vectorStore *store.SQLiteStore) (*SQLiteDocumentBackend, error) {
	if vectorStore == nil {
		return nil, core.NewValidationError("vector_store", nil, "vector store is required for document backend")
	}

	docStore := store.NewDocumentStore(vectorStore.GetSqvectStore())

	return &SQLiteDocumentBackend{
		store:  docStore,
		config: config,
	}, nil
}

// StoreDocument stores document metadata.
func (b *SQLiteDocumentBackend) StoreDocument(ctx context.Context, doc *Document) error {
	if doc == nil {
		return core.NewValidationError("document", doc, "document cannot be nil")
	}

	if doc.ID == "" {
		return core.NewValidationError("document.id", doc.ID, "document ID cannot be empty")
	}

	log.Printf("[DOCUMENT] Storing document %s", doc.ID)

	// Convert to domain document
	domainDoc := domain.Document{
		ID:       doc.ID,
		Path:     doc.FilePath,
		URL:      doc.URL,
		Content:  doc.Content,
		Metadata: doc.Metadata,
		Created:  doc.CreatedAt,
	}

	return b.store.Store(ctx, domainDoc)
}

// GetDocument retrieves a document by ID.
func (b *SQLiteDocumentBackend) GetDocument(ctx context.Context, docID string) (*Document, error) {
	if docID == "" {
		return nil, core.NewValidationError("document_id", docID, "document ID cannot be empty")
	}

	domainDoc, err := b.store.Get(ctx, docID)
	if err != nil {
		if err == domain.ErrDocumentNotFound {
			return nil, core.ErrDocumentNotFound
		}
		return nil, core.WrapErrorWithContext(err, "storage", "GetDocument", "failed to get document")
	}

	// Convert from domain document
	doc := &Document{
		ID:          domainDoc.ID,
		Content:     domainDoc.Content,
		FilePath:    domainDoc.Path,
		URL:         domainDoc.URL,
		Metadata:    domainDoc.Metadata,
		Size:        int64(len(domainDoc.Content)), // Calculate size from content
		CreatedAt:   domainDoc.Created,
		UpdatedAt:   domainDoc.Created, // TODO: Add UpdatedAt to domain.Document
		Version:     1,                 // TODO: Add versioning to domain.Document
	}

	// Infer content type if not in metadata
	if contentType, ok := domainDoc.Metadata["content_type"]; ok {
		if ctStr, ok := contentType.(string); ok {
			doc.ContentType = ctStr
		}
	}

	return doc, nil
}

// ListDocuments lists documents with optional filtering.
func (b *SQLiteDocumentBackend) ListDocuments(ctx context.Context, filter DocumentFilter) ([]Document, error) {
	domainDocs, err := b.store.List(ctx)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "storage", "ListDocuments", "failed to list documents")
	}

	// Convert and apply filters
	var docs []Document
	for _, domainDoc := range domainDocs {
		doc := Document{
			ID:          domainDoc.ID,
			Content:     domainDoc.Content,
			FilePath:    domainDoc.Path,
			URL:         domainDoc.URL,
			Metadata:    domainDoc.Metadata,
			Size:        int64(len(domainDoc.Content)),
			CreatedAt:   domainDoc.Created,
			UpdatedAt:   domainDoc.Created,
			Version:     1,
		}

		// Infer content type
		if contentType, ok := domainDoc.Metadata["content_type"]; ok {
			if ctStr, ok := contentType.(string); ok {
				doc.ContentType = ctStr
			}
		}

		// Apply filters
		if b.matchesFilter(doc, filter) {
			docs = append(docs, doc)
		}
	}

	// Apply pagination
	if filter.Offset > 0 && filter.Offset < len(docs) {
		docs = docs[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(docs) {
		docs = docs[:filter.Limit]
	}

	return docs, nil
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

	log.Printf("[DOCUMENT] Deleting document %s", docID)
	return b.store.Delete(ctx, docID)
}

// GetStats returns document storage statistics.
func (b *SQLiteDocumentBackend) GetStats(ctx context.Context) (*DocumentStats, error) {
	docs, err := b.store.List(ctx)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "storage", "GetStats", "failed to get documents for stats")
	}

	stats := &DocumentStats{
		TotalDocuments: int64(len(docs)),
		ByContentType:  make(map[string]int64),
		LastUpdated:    time.Now(),
	}

	var totalSize int64
	for _, doc := range docs {
		size := int64(len(doc.Content))
		totalSize += size
		
		// Count by content type
		contentType := "unknown"
		if ct, ok := doc.Metadata["content_type"]; ok {
			if ctStr, ok := ct.(string); ok {
				contentType = ctStr
			}
		}
		stats.ByContentType[contentType]++
	}

	stats.TotalSize = totalSize
	if len(docs) > 0 {
		stats.AverageSize = float64(totalSize) / float64(len(docs))
	}

	return stats, nil
}

// Optimize performs storage optimization.
func (b *SQLiteDocumentBackend) Optimize(ctx context.Context) error {
	log.Printf("[DOCUMENT] Performing optimization")
	// TODO: Implement SQLite-specific optimization like VACUUM
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
	// The document store shares the connection with vector store
	// Close is handled at the vector store level
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

// ===== FACTORY FUNCTION =====

// NewDocumentBackend creates a document backend based on the configuration.
func NewDocumentBackend(config DocumentConfig, vectorStore *store.SQLiteStore) (DocumentBackend, error) {
	switch config.Backend {
	case "sqlite", "sqvect":
		return NewSQLiteDocumentBackend(config, vectorStore)
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