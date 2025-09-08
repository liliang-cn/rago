package store

import (
	"context"
	"time"
)

// VectorStore defines the interface for vector storage backends
type VectorStore interface {
	// Initialize the store
	Initialize(ctx context.Context) error
	
	// Close the store and cleanup resources
	Close() error
	
	// Store operations
	Store(ctx context.Context, doc *Document) error
	StoreBatch(ctx context.Context, docs []*Document) error
	
	// Search operations
	Search(ctx context.Context, query SearchQuery) (*SearchResult, error)
	HybridSearch(ctx context.Context, query HybridSearchQuery) (*SearchResult, error)
	
	// Management operations
	Delete(ctx context.Context, id string) error
	DeleteBySource(ctx context.Context, source string) error
	Get(ctx context.Context, id string) (*Document, error)
	List(ctx context.Context, opts ListOptions) ([]*Document, error)
	Count(ctx context.Context) (int64, error)
	
	// Index management
	CreateIndex(ctx context.Context, name string, config IndexConfig) error
	DropIndex(ctx context.Context, name string) error
	ListIndexes(ctx context.Context) ([]IndexInfo, error)
}

// Document represents a document with embeddings
type Document struct {
	ID         string                 `json:"id"`
	Content    string                 `json:"content"`
	Embedding  []float32              `json:"embedding"`
	Source     string                 `json:"source,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	ChunkIndex int                    `json:"chunk_index,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// SearchQuery represents a vector similarity search query
type SearchQuery struct {
	Embedding       []float32
	TopK            int
	Threshold       float64
	Filter          map[string]interface{}
	IncludeMetadata bool
	IncludeVector   bool
}

// HybridSearchQuery combines vector and keyword search
type HybridSearchQuery struct {
	Embedding       []float32
	Keywords        string
	TopK            int
	Threshold       float64
	Filter          map[string]interface{}
	VectorWeight    float64 // Weight for vector search (0-1)
	KeywordWeight   float64 // Weight for keyword search (0-1)
	IncludeMetadata bool
	IncludeVector   bool
}

// SearchResult represents search results
type SearchResult struct {
	Documents  []*ScoredDocument `json:"documents"`
	TotalCount int               `json:"total_count"`
	QueryTime  time.Duration     `json:"query_time"`
}

// ScoredDocument represents a document with relevance score
type ScoredDocument struct {
	Document
	Score          float64 `json:"score"`
	VectorScore    float64 `json:"vector_score,omitempty"`
	KeywordScore   float64 `json:"keyword_score,omitempty"`
	HighlightedText string `json:"highlighted_text,omitempty"`
}

// ListOptions for listing documents
type ListOptions struct {
	Offset int
	Limit  int
	Filter map[string]interface{}
	SortBy string
	Order  string // "asc" or "desc"
}

// IndexConfig for creating indexes
type IndexConfig struct {
	Dimensions      int
	Metric          DistanceMetric
	IndexType       string
	Parameters      map[string]interface{}
}

// IndexInfo provides information about an index
type IndexInfo struct {
	Name       string
	Config     IndexConfig
	DocCount   int64
	CreatedAt  time.Time
}

// DistanceMetric for similarity calculation
type DistanceMetric string

const (
	DistanceCosine    DistanceMetric = "cosine"
	DistanceEuclidean DistanceMetric = "euclidean"
	DistanceDotProduct DistanceMetric = "dot_product"
)

// StoreConfig holds configuration for vector stores
type StoreConfig struct {
	Type       string                 `mapstructure:"type"`
	Parameters map[string]interface{} `mapstructure:"parameters"`
}

// Factory interface for creating vector stores
type Factory interface {
	CreateStore(config StoreConfig) (VectorStore, error)
	SupportedTypes() []string
}

// Error types
type ErrDocumentNotFound struct {
	ID string
}

func (e ErrDocumentNotFound) Error() string {
	return "document not found: " + e.ID
}

type ErrIndexAlreadyExists struct {
	Name string
}

func (e ErrIndexAlreadyExists) Error() string {
	return "index already exists: " + e.Name
}

type ErrIndexNotFound struct {
	Name string
}

func (e ErrIndexNotFound) Error() string {
	return "index not found: " + e.Name
}