package domain

import (
	"context"
	"time"
)

type Document struct {
	ID       string                 `json:"id"`
	Path     string                 `json:"path,omitempty"`
	URL      string                 `json:"url,omitempty"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Created  time.Time              `json:"created"`
}

type Chunk struct {
	ID         string                 `json:"id"`
	DocumentID string                 `json:"document_id"`
	Content    string                 `json:"content"`
	Vector     []float64              `json:"vector,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Score      float64                `json:"score,omitempty"`
}

type QueryRequest struct {
	Query        string                 `json:"query"`
	TopK         int                    `json:"top_k"`
	Temperature  float64                `json:"temperature"`
	MaxTokens    int                    `json:"max_tokens"`
	Stream       bool                   `json:"stream"`
	ShowThinking bool                   `json:"show_thinking"`
	Filters      map[string]interface{} `json:"filters,omitempty"`
}

type QueryResponse struct {
	Answer  string  `json:"answer"`
	Sources []Chunk `json:"sources"`
	Elapsed string  `json:"elapsed"`
}

type IngestRequest struct {
	Content   string                 `json:"content,omitempty"`
	FilePath  string                 `json:"file_path,omitempty"`
	URL       string                 `json:"url,omitempty"`
	ChunkSize int                    `json:"chunk_size"`
	Overlap   int                    `json:"overlap"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type IngestResponse struct {
	Success    bool   `json:"success"`
	DocumentID string `json:"document_id"`
	ChunkCount int    `json:"chunk_count"`
	Message    string `json:"message"`
}

// ExtractedMetadata holds the data extracted from a document by an LLM.

type ExtractedMetadata struct {
	Summary      string   `json:"summary"`
	Keywords     []string `json:"keywords"`
	DocumentType string   `json:"document_type"`
	CreationDate string   `json:"creation_date"`
}

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float64, error)
}

type Generator interface {
	Generate(ctx context.Context, prompt string, opts *GenerationOptions) (string, error)
	Stream(ctx context.Context, prompt string, opts *GenerationOptions, callback func(string)) error
	IsAlmostSame(ctx context.Context, input, output string) (bool, error)
}

type GenerationOptions struct {
	Temperature float64
	MaxTokens   int
	Think       *bool
}

type Chunker interface {
	Split(text string, options ChunkOptions) ([]string, error)
}

type ChunkOptions struct {
	Size    int
	Overlap int
	Method  string
}

type VectorStore interface {
	Store(ctx context.Context, chunks []Chunk) error
	Search(ctx context.Context, vector []float64, topK int) ([]Chunk, error)
	SearchWithFilters(ctx context.Context, vector []float64, topK int, filters map[string]interface{}) ([]Chunk, error)
	Delete(ctx context.Context, documentID string) error
	List(ctx context.Context) ([]Document, error)
	Reset(ctx context.Context) error
}

type KeywordStore interface {
	Index(ctx context.Context, chunk Chunk) error
	Search(ctx context.Context, query string, topK int) ([]Chunk, error)
	Delete(ctx context.Context, documentID string) error
	Reset(ctx context.Context) error
	Close() error
}

type DocumentStore interface {
	Store(ctx context.Context, doc Document) error
	Get(ctx context.Context, id string) (Document, error)
	List(ctx context.Context) ([]Document, error)
	Delete(ctx context.Context, id string) error
}

type Processor interface {
	Ingest(ctx context.Context, req IngestRequest) (IngestResponse, error)
	Query(ctx context.Context, req QueryRequest) (QueryResponse, error)
}