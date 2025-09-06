// Package storage provides pluggable storage backends for the RAG pillar.
// This package implements the storage abstraction layer allowing different
// backends (SQLite, Postgres, Chroma, etc.) to be used interchangeably.
package storage

import (
	"context"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
)

// ===== STORAGE BACKEND INTERFACES =====

// Embedder generates vector embeddings for text.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float64, error)
}

// VectorBackend defines the interface for vector storage operations.
type VectorBackend interface {
	// Store stores vector embeddings for document chunks
	StoreVectors(ctx context.Context, docID string, chunks []VectorChunk) error
	
	// Search performs vector similarity search
	SearchVectors(ctx context.Context, queryVector []float64, options VectorSearchOptions) (*VectorSearchResult, error)
	
	// Delete removes all vectors for a document
	DeleteDocument(ctx context.Context, docID string) error
	
	// Get retrieves vectors for a specific document
	GetDocumentVectors(ctx context.Context, docID string) ([]VectorChunk, error)
	
	// Stats returns storage statistics
	GetStats(ctx context.Context) (*VectorStats, error)
	
	// Optimize performs index optimization
	Optimize(ctx context.Context) error
	
	// Reset clears all data
	Reset(ctx context.Context) error
	
	// Close closes the backend
	Close() error
}

// KeywordBackend defines the interface for full-text search operations.
type KeywordBackend interface {
	// Index indexes text chunks for full-text search
	IndexChunks(ctx context.Context, docID string, chunks []KeywordChunk) error
	
	// Search performs full-text search
	SearchKeywords(ctx context.Context, query string, options KeywordSearchOptions) (*KeywordSearchResult, error)
	
	// Delete removes all indexed content for a document
	DeleteDocument(ctx context.Context, docID string) error
	
	// GetIndexedContent retrieves indexed content for a document
	GetDocumentContent(ctx context.Context, docID string) ([]KeywordChunk, error)
	
	// Stats returns indexing statistics
	GetStats(ctx context.Context) (*KeywordStats, error)
	
	// Optimize performs index optimization
	Optimize(ctx context.Context) error
	
	// Reset clears all indexed data
	Reset(ctx context.Context) error
	
	// Close closes the backend
	Close() error
}

// DocumentBackend defines the interface for document metadata storage.
type DocumentBackend interface {
	// Store stores document metadata
	StoreDocument(ctx context.Context, doc *Document) error
	
	// Get retrieves a document by ID
	GetDocument(ctx context.Context, docID string) (*Document, error)
	
	// List lists documents with optional filtering
	ListDocuments(ctx context.Context, filter DocumentFilter) ([]Document, error)
	
	// Update updates document metadata
	UpdateDocument(ctx context.Context, docID string, updates DocumentUpdate) error
	
	// Delete removes a document
	DeleteDocument(ctx context.Context, docID string) error
	
	// Stats returns document storage statistics
	GetStats(ctx context.Context) (*DocumentStats, error)
	
	// Optimize performs storage optimization
	Optimize(ctx context.Context) error
	
	// Reset clears all documents
	Reset(ctx context.Context) error
	
	// Close closes the backend
	Close() error
}

// ===== DATA TYPES =====

// VectorChunk represents a text chunk with its vector embedding.
type VectorChunk struct {
	ChunkID     string                 `json:"chunk_id"`
	DocumentID  string                 `json:"document_id"`
	Content     string                 `json:"content"`
	Vector      []float64              `json:"vector"`
	Metadata    map[string]interface{} `json:"metadata"`
	Position    int                    `json:"position"`
	CreatedAt   time.Time              `json:"created_at"`
}

// KeywordChunk represents a text chunk for full-text indexing.
type KeywordChunk struct {
	ChunkID     string                 `json:"chunk_id"`
	DocumentID  string                 `json:"document_id"`
	Content     string                 `json:"content"`
	Title       string                 `json:"title,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	Position    int                    `json:"position"`
	CreatedAt   time.Time              `json:"created_at"`
}

// Document represents document metadata and content.
type Document struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title,omitempty"`
	Content     string                 `json:"content"`
	ContentType string                 `json:"content_type"`
	FilePath    string                 `json:"file_path,omitempty"`
	URL         string                 `json:"url,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	Size        int64                  `json:"size"`
	ChunksCount int                    `json:"chunks_count"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Version     int                    `json:"version"`
}

// TextChunk represents a text chunk from document processing.
type TextChunk struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
	Position int                    `json:"position"`
}

// ===== SEARCH OPTIONS AND RESULTS =====

// VectorSearchOptions defines options for vector search.
type VectorSearchOptions struct {
	Limit         int                    `json:"limit"`
	Offset        int                    `json:"offset"`
	Threshold     float64                `json:"threshold"`
	Filter        map[string]interface{} `json:"filter,omitempty"`
	IncludeVector bool                   `json:"include_vector"`
}

// KeywordSearchOptions defines options for keyword search.
type KeywordSearchOptions struct {
	Limit       int                    `json:"limit"`
	Offset      int                    `json:"offset"`
	Filter      map[string]interface{} `json:"filter,omitempty"`
	Fuzzy       bool                   `json:"fuzzy"`
	Boost       map[string]float64     `json:"boost,omitempty"`
	Highlight   bool                   `json:"highlight"`
}

// VectorSearchResult represents vector search results.
type VectorSearchResult struct {
	Chunks      []VectorSearchHit `json:"chunks"`
	Total       int               `json:"total"`
	MaxScore    float64           `json:"max_score"`
	QueryVector []float64         `json:"query_vector,omitempty"`
}

// KeywordSearchResult represents keyword search results.
type KeywordSearchResult struct {
	Chunks   []KeywordSearchHit `json:"chunks"`
	Total    int                `json:"total"`
	MaxScore float64            `json:"max_score"`
	Query    string             `json:"query"`
}

// VectorSearchHit represents a single vector search hit.
type VectorSearchHit struct {
	VectorChunk
	Score float64 `json:"score"`
}

// KeywordSearchHit represents a single keyword search hit.
type KeywordSearchHit struct {
	KeywordChunk
	Score      float64           `json:"score"`
	Highlights []string          `json:"highlights,omitempty"`
	Context    map[string]string `json:"context,omitempty"`
}

// ===== FILTERS AND UPDATES =====

// DocumentFilter defines criteria for filtering documents.
type DocumentFilter struct {
	ContentTypes  []string               `json:"content_types,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAfter  *time.Time             `json:"created_after,omitempty"`
	CreatedBefore *time.Time             `json:"created_before,omitempty"`
	UpdatedAfter  *time.Time             `json:"updated_after,omitempty"`
	UpdatedBefore *time.Time             `json:"updated_before,omitempty"`
	MinSize       int64                  `json:"min_size,omitempty"`
	MaxSize       int64                  `json:"max_size,omitempty"`
	Limit         int                    `json:"limit,omitempty"`
	Offset        int                    `json:"offset,omitempty"`
}

// DocumentUpdate defines fields that can be updated in a document.
type DocumentUpdate struct {
	Title       *string                `json:"title,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ContentType *string                `json:"content_type,omitempty"`
}

// ===== STATISTICS =====

// VectorStats provides statistics about vector storage.
type VectorStats struct {
	TotalVectors    int64                  `json:"total_vectors"`
	TotalDocuments  int64                  `json:"total_documents"`
	StorageSize     int64                  `json:"storage_size"`
	IndexSize       int64                  `json:"index_size"`
	Dimensions      int                    `json:"dimensions"`
	LastOptimized   time.Time              `json:"last_optimized"`
	Performance     map[string]interface{} `json:"performance"`
}

// KeywordStats provides statistics about keyword indexing.
type KeywordStats struct {
	TotalChunks     int64                  `json:"total_chunks"`
	TotalDocuments  int64                  `json:"total_documents"`
	IndexSize       int64                  `json:"index_size"`
	TermsCount      int64                  `json:"terms_count"`
	LastOptimized   time.Time              `json:"last_optimized"`
	Performance     map[string]interface{} `json:"performance"`
}

// DocumentStats provides statistics about document storage.
type DocumentStats struct {
	TotalDocuments  int64                  `json:"total_documents"`
	TotalSize       int64                  `json:"total_size"`
	ByContentType   map[string]int64       `json:"by_content_type"`
	AverageSize     float64                `json:"average_size"`
	LastUpdated     time.Time              `json:"last_updated"`
}

// Stats provides combined storage statistics.
type Stats struct {
	Vector   VectorStats   `json:"vector"`
	Keyword  KeywordStats  `json:"keyword"`
	Document DocumentStats `json:"document"`
}

// ===== CONFIGURATION =====

// VectorConfig defines configuration for vector storage backends.
type VectorConfig struct {
	Backend    string                 `toml:"backend"`     // "sqvect", "chroma", "qdrant", etc.
	DBPath     string                 `toml:"db_path"`
	Dimensions int                    `toml:"dimensions"`
	Metric     string                 `toml:"metric"`      // "cosine", "euclidean", "dot_product"
	IndexType  string                 `toml:"index_type"`
	Options    map[string]interface{} `toml:"options"`
}

// KeywordConfig defines configuration for keyword storage backends.
type KeywordConfig struct {
	Backend   string   `toml:"backend"`   // "bleve", "elasticsearch", "tantivy"
	IndexPath string   `toml:"index_path"`
	Analyzer  string   `toml:"analyzer"`
	Languages []string `toml:"languages"`
	Stemming  bool     `toml:"stemming"`
	Options   map[string]interface{} `toml:"options"`
}

// DocumentConfig defines configuration for document storage backends.
type DocumentConfig struct {
	Backend  string                 `toml:"backend"`  // "sqlite", "postgres", "mongodb"
	DBPath   string                 `toml:"db_path"`
	TableName string                `toml:"table_name"`
	Options  map[string]interface{} `toml:"options"`
}

// ===== HELPER FUNCTIONS =====

// ConvertToVectorChunk converts from storage TextChunk to storage VectorChunk.
func ConvertToVectorChunk(textChunk TextChunk, docID string, vector []float64) VectorChunk {
	return VectorChunk{
		ChunkID:    textChunk.ID,
		DocumentID: docID,
		Content:    textChunk.Content,
		Vector:     vector,
		Metadata:   textChunk.Metadata,
		Position:   textChunk.Position,
		CreatedAt:  time.Now(),
	}
}

// ConvertToKeywordChunk converts from storage TextChunk to storage KeywordChunk.
func ConvertToKeywordChunk(textChunk TextChunk, docID string) KeywordChunk {
	return KeywordChunk{
		ChunkID:    textChunk.ID,
		DocumentID: docID,
		Content:    textChunk.Content,
		Metadata:   textChunk.Metadata,
		Position:   textChunk.Position,
		CreatedAt:  time.Now(),
	}
}

// ConvertToCoreDocument converts from storage Document to core Document.
func ConvertToCoreDocument(doc Document) core.Document {
	return core.Document{
		ID:          doc.ID,
		Content:     doc.Content,
		ContentType: doc.ContentType,
		Metadata:    doc.Metadata,
		CreatedAt:   doc.CreatedAt,
		UpdatedAt:   doc.UpdatedAt,
		Size:        doc.Size,
		ChunksCount: doc.ChunksCount,
	}
}

// ConvertFromCoreDocument converts from core Document to storage Document.
func ConvertFromCoreDocument(coreDoc core.Document) Document {
	return Document{
		ID:          coreDoc.ID,
		Content:     coreDoc.Content,
		ContentType: coreDoc.ContentType,
		Metadata:    coreDoc.Metadata,
		Size:        coreDoc.Size,
		ChunksCount: coreDoc.ChunksCount,
		CreatedAt:   coreDoc.CreatedAt,
		UpdatedAt:   coreDoc.UpdatedAt,
	}
}