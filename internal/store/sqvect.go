package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/liliang-cn/rago/internal/domain"
	"github.com/liliang-cn/sqvect"
)

type SQLiteStore struct {
	sqvect *sqvect.SQLiteStore
}

func NewSQLiteStore(dbPath string, vectorDim int, maxConns int, batchSize int) (*SQLiteStore, error) {
	config := sqvect.DefaultConfig()
	config.Path = dbPath
	config.VectorDim = vectorDim
	config.MaxConns = maxConns
	config.BatchSize = batchSize

	// Create sqvect store with custom configuration
	client, err := sqvect.NewWithConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create sqvect client: %w", err)
	}

	// Initialize the database
	ctx := context.Background()
	if err := client.Init(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize sqvect: %w", err)
	}

	return &SQLiteStore{
		sqvect: client,
	}, nil
}

func (s *SQLiteStore) Store(ctx context.Context, chunks []domain.Chunk) error {
	if len(chunks) == 0 {
		return nil
	}

	for _, chunk := range chunks {
		if len(chunk.Vector) == 0 {
			continue
		}

		// Convert []float64 to []float32 for sqvect
		vector := make([]float32, len(chunk.Vector))
		for i, v := range chunk.Vector {
			vector[i] = float32(v)
		}

		// Convert metadata to string map
		metadata := make(map[string]string)
		if chunk.Metadata != nil {
			for k, v := range chunk.Metadata {
				metadata[k] = fmt.Sprintf("%v", v)
			}
		}

		embedding := &sqvect.Embedding{
			ID:       chunk.ID,
			Vector:   vector,
			Content:  chunk.Content,
			DocID:    chunk.DocumentID,
			Metadata: metadata,
		}

		if err := s.sqvect.Upsert(ctx, embedding); err != nil {
			return fmt.Errorf("%w: failed to store chunk: %v", domain.ErrVectorStoreFailed, err)
		}
	}

	return nil
}

func (s *SQLiteStore) Search(ctx context.Context, vector []float64, topK int) ([]domain.Chunk, error) {
	if len(vector) == 0 {
		return nil, fmt.Errorf("%w: empty query vector", domain.ErrInvalidInput)
	}

	if topK <= 0 {
		topK = 5
	}

	// Convert []float64 to []float32 for sqvect
	queryVector := make([]float32, len(vector))
	for i, v := range vector {
		queryVector[i] = float32(v)
	}

	results, err := s.sqvect.Search(ctx, queryVector, sqvect.SearchOptions{
		TopK:      topK,
		Threshold: 0.0, // Return all results, let caller filter
	})
	if err != nil {
		return nil, fmt.Errorf("%w: search failed: %v", domain.ErrVectorStoreFailed, err)
	}

	chunks := make([]domain.Chunk, len(results))
	for i, result := range results {
		// Convert []float32 back to []float64
		resultVector := make([]float64, len(result.Vector))
		for j, v := range result.Vector {
			resultVector[j] = float64(v)
		}

		// Convert metadata back to interface{}
		metadata := make(map[string]interface{})
		for k, v := range result.Metadata {
			metadata[k] = v
		}

		chunks[i] = domain.Chunk{
			ID:         result.ID,
			DocumentID: result.DocID,
			Content:    result.Content,
			Vector:     resultVector,
			Score:      float64(result.Score),
			Metadata:   metadata,
		}
	}

	return chunks, nil
}

func (s *SQLiteStore) Delete(ctx context.Context, documentID string) error {
	if documentID == "" {
		return fmt.Errorf("%w: empty document ID", domain.ErrInvalidInput)
	}

	if err := s.sqvect.DeleteByDocID(ctx, documentID); err != nil {
		return fmt.Errorf("%w: failed to delete document: %v", domain.ErrVectorStoreFailed, err)
	}

	return nil
}

func (s *SQLiteStore) List(ctx context.Context) ([]domain.Document, error) {
	// Try using ListDocumentsWithInfo first (sqvect v0.3.0)
	docInfos, err := s.sqvect.ListDocumentsWithInfo(ctx)
	if err != nil {
		// Fall back to old method if ListDocumentsWithInfo is not available
		return s.listWithFallback(ctx)
	}

	if len(docInfos) == 0 {
		return []domain.Document{}, nil
	}

	documents := make([]domain.Document, 0, len(docInfos))

	for _, docInfo := range docInfos {
		// Get the document details using GetByDocID
		embeddings, err := s.sqvect.GetByDocID(ctx, docInfo.DocID)
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

				doc := domain.Document{
					ID:      embedding.DocID,
					Path:    embedding.Metadata["_path"],
					URL:     embedding.Metadata["_url"],
					Content: embedding.Content,
					Created: created,
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

// Fallback method for compatibility
func (s *SQLiteStore) listWithFallback(ctx context.Context) ([]domain.Document, error) {
	// Try GetDocumentsByType
	embeddings, err := s.sqvect.GetDocumentsByType(ctx, "document")
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get documents: %v", domain.ErrVectorStoreFailed, err)
	}

	if len(embeddings) == 0 {
		// Final fallback: ListDocuments + GetByDocID
		docIDs, err := s.sqvect.ListDocuments(ctx)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to list documents: %v", domain.ErrVectorStoreFailed, err)
		}

		documentMap := make(map[string]domain.Document)

		for _, docID := range docIDs {
			docEmbeddings, err := s.sqvect.GetByDocID(ctx, docID)
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

					doc := domain.Document{
						ID:      embedding.DocID,
						Path:    embedding.Metadata["_path"],
						URL:     embedding.Metadata["_url"],
						Content: embedding.Content,
						Created: created,
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
		documents := make([]domain.Document, 0, len(documentMap))
		for _, doc := range documentMap {
			documents = append(documents, doc)
		}

		return documents, nil
	}

	// Process GetDocumentsByType results
	documentMap := make(map[string]domain.Document)

	for _, embedding := range embeddings {
		if embedding.Metadata["_type"] == "document" {
			// Parse created time
			var created time.Time
			if createdStr, ok := embedding.Metadata["_created"]; ok {
				if parsed, err := time.Parse("2006-01-02T15:04:05Z07:00", createdStr); err == nil {
					created = parsed
				}
			}

			doc := domain.Document{
				ID:      embedding.DocID,
				Path:    embedding.Metadata["_path"],
				URL:     embedding.Metadata["_url"],
				Content: embedding.Content,
				Created: created,
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
	documents := make([]domain.Document, 0, len(documentMap))
	for _, doc := range documentMap {
		documents = append(documents, doc)
	}

	return documents, nil
}

func (s *SQLiteStore) Reset(ctx context.Context) error {
	// Use the new Clear method from sqvect v0.3.0
	if err := s.sqvect.Clear(ctx); err != nil {
		return fmt.Errorf("%w: failed to clear store: %v", domain.ErrVectorStoreFailed, err)
	}
	return nil
}

func (s *SQLiteStore) Close() error {
	return s.sqvect.Close()
}

// DocumentStore is a simple wrapper that uses sqvect for document storage too
type DocumentStore struct {
	sqvect *sqvect.SQLiteStore
}

func NewDocumentStore(sqvectStore *sqvect.SQLiteStore) *DocumentStore {
	return &DocumentStore{
		sqvect: sqvectStore,
	}
}

func (s *DocumentStore) Store(ctx context.Context, doc domain.Document) error {
	// Get vector dimension from the sqvect store config
	stats, err := s.sqvect.Stats(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to get store stats: %v", domain.ErrDocumentStoreFailed, err)
	}

	// Store document as a special embedding with zero vector
	metadata := make(map[string]string)
	if doc.Metadata != nil {
		for k, v := range doc.Metadata {
			metadata[k] = fmt.Sprintf("%v", v)
		}
	}
	metadata["_type"] = "document"
	metadata["_path"] = doc.Path
	metadata["_url"] = doc.URL
	metadata["_created"] = doc.Created.Format("2006-01-02T15:04:05Z07:00")

	embedding := &sqvect.Embedding{
		ID:       doc.ID,
		Vector:   make([]float32, stats.Dimensions), // Use actual dimensions from store
		Content:  doc.Content,
		DocID:    doc.ID,
		Metadata: metadata,
	}

	if err := s.sqvect.Upsert(ctx, embedding); err != nil {
		return fmt.Errorf("%w: failed to store document: %v", domain.ErrDocumentStoreFailed, err)
	}

	return nil
}

func (s *DocumentStore) Get(ctx context.Context, id string) (domain.Document, error) {
	// Use the new GetByDocID method from sqvect v0.3.0
	embeddings, err := s.sqvect.GetByDocID(ctx, id)
	if err != nil {
		return domain.Document{}, fmt.Errorf("%w: failed to get document by ID: %v", domain.ErrDocumentStoreFailed, err)
	}

	// Find the document metadata embedding
	for _, embedding := range embeddings {
		if embedding.Metadata["_type"] == "document" && embedding.DocID == id {
			// Parse created time
			var created time.Time
			if createdStr, ok := embedding.Metadata["_created"]; ok {
				if parsed, err := time.Parse("2006-01-02T15:04:05Z07:00", createdStr); err == nil {
					created = parsed
				}
			}

			doc := domain.Document{
				ID:      id,
				Path:    embedding.Metadata["_path"],
				URL:     embedding.Metadata["_url"],
				Content: embedding.Content,
				Created: created,
			}

			// Copy non-internal metadata
			doc.Metadata = make(map[string]interface{})
			for k, v := range embedding.Metadata {
				if !strings.HasPrefix(k, "_") {
					doc.Metadata[k] = v
				}
			}

			return doc, nil
		}
	}

	return domain.Document{}, domain.ErrDocumentNotFound
}

func (s *DocumentStore) List(ctx context.Context) ([]domain.Document, error) {
	// Use the same optimized approach as SQLiteStore
	return s.listWithOptimizedAPI(ctx)
}

// Optimized method using new sqvect v0.3.0 APIs
func (s *DocumentStore) listWithOptimizedAPI(ctx context.Context) ([]domain.Document, error) {
	// Try using ListDocumentsWithInfo first (sqvect v0.3.0)
	docInfos, err := s.sqvect.ListDocumentsWithInfo(ctx)
	if err != nil {
		// Fall back to old method if ListDocumentsWithInfo is not available
		return s.listWithFallback(ctx)
	}

	if len(docInfos) == 0 {
		return []domain.Document{}, nil
	}

	documents := make([]domain.Document, 0, len(docInfos))

	for _, docInfo := range docInfos {
		// Get the document details using GetByDocID
		embeddings, err := s.sqvect.GetByDocID(ctx, docInfo.DocID)
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

				doc := domain.Document{
					ID:      embedding.DocID,
					Path:    embedding.Metadata["_path"],
					URL:     embedding.Metadata["_url"],
					Content: embedding.Content,
					Created: created,
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

// Fallback method for DocumentStore
func (s *DocumentStore) listWithFallback(ctx context.Context) ([]domain.Document, error) {
	// Try GetDocumentsByType
	embeddings, err := s.sqvect.GetDocumentsByType(ctx, "document")
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get documents: %v", domain.ErrDocumentStoreFailed, err)
	}

	if len(embeddings) == 0 {
		// Final fallback: ListDocuments + GetByDocID
		docIDs, err := s.sqvect.ListDocuments(ctx)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to list documents: %v", domain.ErrDocumentStoreFailed, err)
		}

		documentMap := make(map[string]domain.Document)

		for _, docID := range docIDs {
			docEmbeddings, err := s.sqvect.GetByDocID(ctx, docID)
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

					doc := domain.Document{
						ID:      embedding.DocID,
						Path:    embedding.Metadata["_path"],
						URL:     embedding.Metadata["_url"],
						Content: embedding.Content,
						Created: created,
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
		documents := make([]domain.Document, 0, len(documentMap))
		for _, doc := range documentMap {
			documents = append(documents, doc)
		}

		return documents, nil
	}

	// Process GetDocumentsByType results
	documentMap := make(map[string]domain.Document)

	for _, embedding := range embeddings {
		if embedding.Metadata["_type"] == "document" {
			// Parse created time
			var created time.Time
			if createdStr, ok := embedding.Metadata["_created"]; ok {
				if parsed, err := time.Parse("2006-01-02T15:04:05Z07:00", createdStr); err == nil {
					created = parsed
				}
			}

			doc := domain.Document{
				ID:      embedding.DocID,
				Path:    embedding.Metadata["_path"],
				URL:     embedding.Metadata["_url"],
				Content: embedding.Content,
				Created: created,
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
	documents := make([]domain.Document, 0, len(documentMap))
	for _, doc := range documentMap {
		documents = append(documents, doc)
	}

	return documents, nil
}

func (s *DocumentStore) Delete(ctx context.Context, id string) error {
	if err := s.sqvect.Delete(ctx, id); err != nil {
		return fmt.Errorf("%w: failed to delete document: %v", domain.ErrDocumentStoreFailed, err)
	}
	return nil
}

// Helper function to get sqvect client for DocumentStore creation
func (s *SQLiteStore) GetSqvectStore() *sqvect.SQLiteStore {
	return s.sqvect
}
