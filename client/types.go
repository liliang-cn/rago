// Package client provides types for the RAGO client
package client

// GenerateOptions represents options for text generation
type GenerateOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
}

// QueryOptions represents options for RAG queries
type QueryOptions struct {
	TopK        int               `json:"top_k,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	ShowSources bool              `json:"show_sources,omitempty"`
	Filters     map[string]string `json:"filters,omitempty"`
}


// IngestOptions represents options for document ingestion
type IngestOptions struct {
	ChunkSize       int               // Size of text chunks
	Overlap         int               // Overlap between chunks
	Metadata        map[string]string // Additional metadata
	RecursiveDir    bool              // Recursively scan directories
	ExcludePatterns []string          // Patterns to exclude
	FileTypes       []string          // Specific file types to include
}

// IngestRequest represents a request to ingest documents
type IngestRequest struct {
	Path            string            `json:"path"`
	ChunkSize       int               `json:"chunk_size"`
	Overlap         int               `json:"overlap"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	RecursiveDir    bool              `json:"recursive_dir,omitempty"`
	ExcludePatterns []string          `json:"exclude_patterns,omitempty"`
	FileTypes       []string          `json:"file_types,omitempty"`
}

// IngestResponse represents the response from document ingestion
type IngestResponse struct {
	Success         bool   `json:"success"`
	DocumentsCount  int    `json:"documents_count"`
	ChunksCount     int    `json:"chunks_count"`
	Message         string `json:"message"`
}

// QueryRequest represents a RAG query request
type QueryRequest struct {
	Query         string            `json:"query"`
	TopK          int               `json:"top_k"`
	Temperature   float64           `json:"temperature"`
	MaxTokens     int               `json:"max_tokens"`
	ShowSources   bool              `json:"show_sources"`
	Filters       map[string]string `json:"filters,omitempty"`
	IncludeImages bool              `json:"include_images,omitempty"`
}

// QueryResponse represents a RAG query response
type QueryResponse struct {
	Answer  string         `json:"answer"`
	Sources []SearchResult `json:"sources,omitempty"`
	Images  []string       `json:"images,omitempty"`
}

// TaskRequest represents a task execution request
type TaskRequest struct {
	Task    string `json:"task"`
	Verbose bool   `json:"verbose"`
	Timeout int    `json:"timeout,omitempty"`
}

// TaskResponse represents a task execution response
type TaskResponse struct {
	Success bool                   `json:"success"`
	Output  map[string]interface{} `json:"output,omitempty"`
	Steps   []StepResult           `json:"steps,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// TaskStep represents a single step in task execution
type TaskStep struct {
	Name    string      `json:"name"`
	Success bool        `json:"success"`
	Output  interface{} `json:"output,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// SearchOptions represents options for search operations
type SearchOptions struct {
	TopK           int               `json:"top_k,omitempty"`
	Filters        map[string]string `json:"filters,omitempty"`
	Threshold      float64           `json:"threshold,omitempty"`
	IncludeContent bool              `json:"include_content,omitempty"`
}

// AgentOptions represents options for agent operations
type AgentOptions struct {
	Verbose bool `json:"verbose,omitempty"`
	Timeout int  `json:"timeout,omitempty"`
}

// ToolInfo represents information about a tool
type ToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// StepResult represents the result of a single step
type StepResult struct {
	Name    string      `json:"name"`
	Success bool        `json:"success"`
	Output  interface{} `json:"output,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// PlanStep represents a step in a plan
type PlanStep struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// PlanResponse represents a planning response
type PlanResponse struct {
	Task  string     `json:"task"`
	Steps []PlanStep `json:"steps"`
}