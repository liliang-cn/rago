package store

import (
	"context"
	"fmt"

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
	// sqvect doesn't have a direct List method for documents
	// This would need to be implemented based on your sqvect API
	// For now, return empty slice
	return []domain.Document{}, nil
}

func (s *SQLiteStore) Reset(ctx context.Context) error {
	// sqvect doesn't have Clear method, delete all by querying all doc_ids first
	// For now, we'll implement a simple truncate-like operation
	// This is a placeholder - you might want to implement this differently based on your sqvect API
	return fmt.Errorf("Reset not implemented - sqvect doesn't have Clear method")
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
	// This would need to be implemented based on your sqvect API
	// For now, return empty document
	return domain.Document{}, domain.ErrDocumentNotFound
}

func (s *DocumentStore) List(ctx context.Context) ([]domain.Document, error) {
	// This would need to be implemented based on your sqvect API
	// For now, return empty slice
	return []domain.Document{}, nil
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
