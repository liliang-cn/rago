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
	ShowSources  bool                   `json:"show_sources"`
	Filters      map[string]interface{} `json:"filters,omitempty"`
	ToolsEnabled bool                   `json:"tools_enabled"`
	AllowedTools []string               `json:"allowed_tools,omitempty"`
	MaxToolCalls int                    `json:"max_tool_calls"`
}

type QueryResponse struct {
	Answer    string             `json:"answer"`
	Sources   []Chunk            `json:"sources"`
	Elapsed   string             `json:"elapsed"`
	ToolCalls []ExecutedToolCall `json:"tool_calls,omitempty"`
	ToolsUsed []string           `json:"tools_used,omitempty"`
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
	Summary      string                 `json:"summary"`
	Keywords     []string               `json:"keywords"`
	DocumentType string                 `json:"document_type"`
	CreationDate string                 `json:"creation_date"`
	Collection   string                 `json:"collection"`               // LLM-determined collection name
	// Enhanced metadata fields
	TemporalRefs map[string]string      `json:"temporal_refs,omitempty"` // e.g., "today": "2025-09-12", "tomorrow": "2025-09-13"
	Entities     map[string][]string    `json:"entities,omitempty"`      // e.g., "person": ["张三"], "location": ["华西医院"]
	Events       []string               `json:"events,omitempty"`         // e.g., ["手术前检查", "玻璃体切割术"]
	CustomMeta   map[string]interface{} `json:"custom_meta,omitempty"`   // For any additional metadata
}

type Stats struct {
	TotalDocuments int `json:"total_documents"`
	TotalChunks    int `json:"total_chunks"`
}

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float64, error)
}

// Message represents a conversation message, used for tool calling
type Message struct {
	Role       string     `json:"role"` // user, assistant, tool
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type Generator interface {
	Generate(ctx context.Context, prompt string, opts *GenerationOptions) (string, error)
	Stream(ctx context.Context, prompt string, opts *GenerationOptions, callback func(string)) error
	GenerateWithTools(ctx context.Context, messages []Message, tools []ToolDefinition, opts *GenerationOptions) (*GenerationResult, error)
	StreamWithTools(ctx context.Context, messages []Message, tools []ToolDefinition, opts *GenerationOptions, callback ToolCallCallback) error
	GenerateStructured(ctx context.Context, prompt string, schema interface{}, opts *GenerationOptions) (*StructuredResult, error)
}

type GenerationOptions struct {
	Temperature float64
	MaxTokens   int
	Think       *bool
	ToolChoice  string // "auto", "required", "none", or specific function name
}

// Tool calling related types

// ToolDefinition represents a tool that can be called by the LLM
type ToolDefinition struct {
	Type     string       `json:"type"` // Always "function"
	Function ToolFunction `json:"function"`
}

// ToolFunction defines a function that can be called
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall represents a call to a tool made by the LLM
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall represents the function call details
type FunctionCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// GenerationResult represents the result of LLM generation with potential tool calls
type GenerationResult struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Finished  bool       `json:"finished"`
}

// StructuredResult represents the result of structured generation
type StructuredResult struct {
	Data  interface{} `json:"data"`  // Parsed structured data
	Raw   string      `json:"raw"`   // Raw JSON string
	Valid bool        `json:"valid"` // Whether the response passed schema validation
}

// ToolCallCallback is called during streaming when tool calls are detected
type ToolCallCallback func(chunk string, toolCalls []ToolCall) error

// ExecutedToolCall represents a tool call that has been executed
type ExecutedToolCall struct {
	ToolCall
	Result  interface{} `json:"result"`
	Elapsed string      `json:"elapsed"`
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
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
