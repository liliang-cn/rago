package core

import (
	"time"
)

// ===== CONFIGURATION TYPES =====

// Config represents the unified RAGO configuration supporting all four pillars
type Config struct {
	// Global settings
	DataDir  string `toml:"data_dir"`
	LogLevel string `toml:"log_level"`

	// Pillar-specific configurations
	LLM    LLMConfig    `toml:"llm"`
	RAG    RAGConfig    `toml:"rag"`
	MCP    MCPConfig    `toml:"mcp"`
	Agents AgentsConfig `toml:"agents"`

	// Operational modes for selective pillar activation
	Mode ModeConfig `toml:"mode"`
}

// ModeConfig controls which pillars are active/inactive
type ModeConfig struct {
	RAGOnly      bool `toml:"rag_only"`       // Only RAG pillar
	LLMOnly      bool `toml:"llm_only"`       // Only LLM pillar
	DisableMCP   bool `toml:"disable_mcp"`    // Disable MCP pillar
	DisableAgent bool `toml:"disable_agents"` // Disable Agent pillar
}

// LLMConfig contains configuration for the LLM pillar
type LLMConfig struct {
	DefaultProvider string                    `toml:"default_provider"`
	LoadBalancing   LoadBalancingConfig       `toml:"load_balancing"`
	Providers       map[string]ProviderConfig `toml:"providers"`
	HealthCheck     HealthCheckConfig         `toml:"health_check"`
}

// RAGConfig contains configuration for the RAG pillar
type RAGConfig struct {
	StorageBackend   string              `toml:"storage_backend"`
	ChunkingStrategy ChunkingConfig      `toml:"chunking"`
	VectorStore      VectorStoreConfig   `toml:"vector_store"`
	KeywordStore     KeywordStoreConfig  `toml:"keyword_store"`
	Search           SearchConfig        `toml:"search"`
	Embedding        EmbeddingConfig     `toml:"embedding"`
}

// MCPConfig contains configuration for the MCP pillar
type MCPConfig struct {
	ServersPath          string                   `toml:"servers_path"`
	Servers              []ServerConfig           `toml:"servers"`
	HealthCheck          HealthCheckConfig        `toml:"health_check"`
	ToolExecution        ToolExecutionConfig      `toml:"tool_execution"`
	HealthCheckInterval  time.Duration            `toml:"health_check_interval"`
	CacheSize            int                      `toml:"cache_size"`
	CacheTTL             time.Duration            `toml:"cache_ttl"`
}

// AgentsConfig contains configuration for the Agent pillar
type AgentsConfig struct {
	WorkflowEngine   WorkflowEngineConfig    `toml:"workflow_engine"`
	Scheduling       SchedulingConfig        `toml:"scheduling"`
	StateStorage     StateStorageConfig      `toml:"state_storage"`
	ReasoningChains  ReasoningChainsConfig   `toml:"reasoning_chains"`
}

// ===== HEALTH AND STATUS TYPES =====

// HealthStatus represents the health state of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// HealthReport provides comprehensive health information
type HealthReport struct {
	Overall   HealthStatus             `json:"overall"`
	Pillars   map[string]HealthStatus  `json:"pillars"`
	Providers map[string]HealthStatus  `json:"providers"`
	Servers   map[string]HealthStatus  `json:"servers"`
	LastCheck time.Time                `json:"last_check"`
	Details   map[string]interface{}   `json:"details"`
}

// ===== LLM PILLAR TYPES =====

// ProviderConfig defines configuration for LLM providers
type ProviderConfig struct {
	Type       string            `toml:"type"`        // "ollama", "openai", "lmstudio"
	BaseURL    string            `toml:"base_url"`
	APIKey     string            `toml:"api_key"`
	Model      string            `toml:"model"`
	Parameters map[string]interface{} `toml:"parameters"`
	Weight     int               `toml:"weight"`      // for load balancing
	Timeout    time.Duration     `toml:"timeout"`
}

// ProviderInfo provides information about a registered provider
type ProviderInfo struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Model     string            `json:"model"`
	Health    HealthStatus      `json:"health"`
	Weight    int               `json:"weight"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// GenerationRequest represents a request to generate text
type GenerationRequest struct {
	Prompt      string                 `json:"prompt"`
	Model       string                 `json:"model,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Context     []Message              `json:"context,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Temperature float32                `json:"temperature,omitempty"`
}

// GenerationResponse represents the response from text generation
type GenerationResponse struct {
	Content     string                 `json:"content"`
	Model       string                 `json:"model"`
	Provider    string                 `json:"provider"`
	Usage       TokenUsage             `json:"usage"`
	Metadata    map[string]interface{} `json:"metadata"`
	Duration    time.Duration          `json:"duration"`
}

// ToolGenerationRequest represents a request with tool calling capability
type ToolGenerationRequest struct {
	GenerationRequest
	Tools         []ToolInfo `json:"tools"`
	ToolChoice    string     `json:"tool_choice,omitempty"`
	MaxToolCalls  int        `json:"max_tool_calls,omitempty"`
}

// ToolGenerationResponse represents response with potential tool calls
type ToolGenerationResponse struct {
	GenerationResponse
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// StreamChunk represents a chunk of streamed content
type StreamChunk struct {
	Content   string        `json:"content"`
	Delta     string        `json:"delta"`
	Finished  bool          `json:"finished"`
	Usage     TokenUsage    `json:"usage,omitempty"`
	Duration  time.Duration `json:"duration"`
}

// ToolStreamChunk represents a chunk of streamed content with tools
type ToolStreamChunk struct {
	StreamChunk
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// Message represents a conversation message
type Message struct {
	Role    string `json:"role"`    // "user", "assistant", "system", "tool"
	Content string `json:"content"`
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// TokenUsage tracks token consumption
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ===== RAG PILLAR TYPES =====

// IngestRequest represents a document ingestion request
type IngestRequest struct {
	DocumentID   string                 `json:"document_id"`
	Content      string                 `json:"content,omitempty"`
	FilePath     string                 `json:"file_path,omitempty"`
	URL          string                 `json:"url,omitempty"`
	ContentType  string                 `json:"content_type"`
	Metadata     map[string]interface{} `json:"metadata"`
	ChunkingHint string                 `json:"chunking_hint,omitempty"`
}

// IngestResponse represents the response from document ingestion
type IngestResponse struct {
	DocumentID   string        `json:"document_id"`
	ChunksCount  int           `json:"chunks_count"`
	ProcessedAt  time.Time     `json:"processed_at"`
	Duration     time.Duration `json:"duration"`
	StorageSize  int64         `json:"storage_size"`
}

// BatchIngestResponse represents response from batch ingestion
type BatchIngestResponse struct {
	Responses       []IngestResponse `json:"responses"`
	TotalDocuments  int              `json:"total_documents"`
	SuccessfulCount int              `json:"successful_count"`
	FailedCount     int              `json:"failed_count"`
	Duration        time.Duration    `json:"duration"`
}

// SearchRequest represents a search query
type SearchRequest struct {
	Query     string                 `json:"query"`
	Limit     int                    `json:"limit,omitempty"`
	Offset    int                    `json:"offset,omitempty"`
	Filter    map[string]interface{} `json:"filter,omitempty"`
	Threshold float32                `json:"threshold,omitempty"`
}

// SearchResponse represents search results
type SearchResponse struct {
	Results   []SearchResult `json:"results"`
	Total     int            `json:"total"`
	Duration  time.Duration  `json:"duration"`
	Query     string         `json:"query"`
}

// HybridSearchRequest combines vector and keyword search
type HybridSearchRequest struct {
	SearchRequest
	VectorWeight  float32 `json:"vector_weight"`
	KeywordWeight float32 `json:"keyword_weight"`
	RRFParams     RRFParams `json:"rrf_params"`
}

// HybridSearchResponse represents hybrid search results
type HybridSearchResponse struct {
	SearchResponse
	VectorResults  []SearchResult `json:"vector_results"`
	KeywordResults []SearchResult `json:"keyword_results"`
	FusionMethod   string         `json:"fusion_method"`
}

// SearchResult represents a single search result
type SearchResult struct {
	DocumentID  string                 `json:"document_id"`
	ChunkID     string                 `json:"chunk_id"`
	Content     string                 `json:"content"`
	Score       float32                `json:"score"`
	Metadata    map[string]interface{} `json:"metadata"`
	Highlights  []string               `json:"highlights,omitempty"`
}

// Document represents a stored document
type Document struct {
	ID          string                 `json:"id"`
	Content     string                 `json:"content"`
	ContentType string                 `json:"content_type"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Size        int64                  `json:"size"`
	ChunksCount int                    `json:"chunks_count"`
}

// DocumentFilter defines filtering criteria for document listing
type DocumentFilter struct {
	ContentType string                 `json:"content_type,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAfter  time.Time            `json:"created_after,omitempty"`
	CreatedBefore time.Time            `json:"created_before,omitempty"`
	Limit       int                    `json:"limit,omitempty"`
	Offset      int                    `json:"offset,omitempty"`
}

// RAGStats provides statistics about the RAG system
type RAGStats struct {
	TotalDocuments int                    `json:"total_documents"`
	TotalChunks    int                    `json:"total_chunks"`
	StorageSize    int64                  `json:"storage_size"`
	IndexSize      int64                  `json:"index_size"`
	ByContentType  map[string]int         `json:"by_content_type"`
	LastOptimized  time.Time              `json:"last_optimized"`
	Performance    map[string]interface{} `json:"performance"`
}

// ===== MCP PILLAR TYPES =====

// ServerConfig defines configuration for MCP servers
type ServerConfig struct {
	Name             string            `toml:"name"`
	Description      string            `toml:"description"`
	Command          []string          `toml:"command"`
	Args             []string          `toml:"args,omitempty"`
	Env              map[string]string `toml:"env,omitempty"`
	WorkingDir       string            `toml:"working_dir,omitempty"`
	Timeout          time.Duration     `toml:"timeout"`
	Retries          int               `toml:"retries"`
	AutoStart        bool              `toml:"auto_start"`
	RestartOnFailure bool              `toml:"restart_on_failure"`
	MaxRestarts      int               `toml:"max_restarts"`
	RestartDelay     time.Duration     `toml:"restart_delay"`
	Capabilities     []string          `toml:"capabilities"`
}

// ServerInfo provides information about a registered MCP server
type ServerInfo struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Status       string                 `json:"status"`
	Version      string                 `json:"version"`
	Capabilities []string               `json:"capabilities"`
	ToolCount    int                    `json:"tool_count"`
	Tools        []string               `json:"tools"`
	LastSeen     time.Time              `json:"last_seen"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// ToolInfo describes an available tool
type ToolInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	ServerName  string                 `json:"server_name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	InputSchema map[string]interface{} `json:"input_schema"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ToolCallRequest represents a request to call a tool
type ToolCallRequest struct {
	ToolName   string                 `json:"tool_name"`
	Arguments  map[string]interface{} `json:"arguments"`
	Timeout    time.Duration          `json:"timeout,omitempty"`
	Async      bool                   `json:"async,omitempty"`
}

// ToolCallResponse represents the response from a tool call
type ToolCallResponse struct {
	ToolName  string                 `json:"tool_name"`
	Success   bool                   `json:"success"`
	Result    interface{}            `json:"result,omitempty"`
	Error     error                  `json:"error,omitempty"`
	Duration  time.Duration          `json:"duration"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ToolCall represents a tool call in generation
type ToolCall struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters"`
}

// ===== AGENT PILLAR TYPES =====

// WorkflowDefinition defines a workflow structure
type WorkflowDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Steps       []WorkflowStep         `json:"steps"`
	Inputs      []WorkflowInput        `json:"inputs"`
	Outputs     []WorkflowOutput       `json:"outputs"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// WorkflowStep represents a single step in a workflow
type WorkflowStep struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // "llm", "rag", "mcp", "agent"
	Parameters  map[string]interface{} `json:"parameters"`
	Dependencies []string              `json:"dependencies,omitempty"`
	Condition   string                 `json:"condition,omitempty"`
}

// WorkflowInput defines workflow input parameters
type WorkflowInput struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Description string      `json:"description"`
}

// WorkflowOutput defines workflow output parameters
type WorkflowOutput struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// WorkflowRequest represents a workflow execution request
type WorkflowRequest struct {
	WorkflowName string                 `json:"workflow_name"`
	Inputs       map[string]interface{} `json:"inputs"`
	Context      map[string]interface{} `json:"context,omitempty"`
}

// WorkflowResponse represents the response from workflow execution
type WorkflowResponse struct {
	WorkflowName string                 `json:"workflow_name"`
	Status       string                 `json:"status"`
	Outputs      map[string]interface{} `json:"outputs"`
	Steps        []StepResult           `json:"steps"`
	Duration     time.Duration          `json:"duration"`
	StartedAt    time.Time              `json:"started_at"`
	CompletedAt  time.Time              `json:"completed_at"`
}

// StepResult represents the result of a workflow step
type StepResult struct {
	StepID   string                 `json:"step_id"`
	Status   string                 `json:"status"`
	Output   interface{}            `json:"output"`
	Error    string                 `json:"error,omitempty"`
	Duration time.Duration          `json:"duration"`
}

// AgentDefinition defines an agent structure
type AgentDefinition struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Type         string                 `json:"type"`
	Instructions string                 `json:"instructions"`
	Tools        []string               `json:"tools"`
	Parameters   map[string]interface{} `json:"parameters"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// AgentRequest represents an agent execution request
type AgentRequest struct {
	AgentName string                 `json:"agent_name"`
	Task      string                 `json:"task"`
	Context   map[string]interface{} `json:"context,omitempty"`
	MaxSteps  int                    `json:"max_steps,omitempty"`
}

// AgentResponse represents the response from agent execution
type AgentResponse struct {
	AgentName   string        `json:"agent_name"`
	Status      string        `json:"status"`
	Result      string        `json:"result"`
	Steps       []AgentStep   `json:"steps"`
	Duration    time.Duration `json:"duration"`
	StartedAt   time.Time     `json:"started_at"`
	CompletedAt time.Time     `json:"completed_at"`
}

// AgentStep represents a single step in agent execution
type AgentStep struct {
	StepNumber int                    `json:"step_number"`
	Action     string                 `json:"action"`
	Input      interface{}            `json:"input"`
	Output     interface{}            `json:"output"`
	Duration   time.Duration          `json:"duration"`
}

// WorkflowInfo provides information about a workflow
type WorkflowInfo struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	StepsCount  int       `json:"steps_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AgentInfo provides information about an agent
type AgentInfo struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	ToolsCount  int       `json:"tools_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ScheduleConfig defines scheduling configuration
type ScheduleConfig struct {
	Type       string            `json:"type"` // "cron", "interval", "event"
	Expression string            `json:"expression"`
	Timezone   string            `json:"timezone,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ScheduledTask represents a scheduled task
type ScheduledTask struct {
	ID           string         `json:"id"`
	WorkflowName string         `json:"workflow_name"`
	Schedule     ScheduleConfig `json:"schedule"`
	NextRun      time.Time      `json:"next_run"`
	LastRun      time.Time      `json:"last_run"`
	Status       string         `json:"status"`
}

// ===== HIGH-LEVEL OPERATION TYPES =====

// ChatRequest represents a high-level chat request
type ChatRequest struct {
	Message     string                 `json:"message"`
	Context     []Message              `json:"context,omitempty"`
	UseRAG      bool                   `json:"use_rag"`
	UseTools    bool                   `json:"use_tools"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// ChatResponse represents the response from a chat request
type ChatResponse struct {
	Response      string         `json:"response"`
	Context       []Message      `json:"context"`
	Sources       []SearchResult `json:"sources,omitempty"`
	ToolCalls     []ToolCall     `json:"tool_calls,omitempty"`
	Duration      time.Duration  `json:"duration"`
	Usage         TokenUsage     `json:"usage"`
}

// DocumentRequest represents a high-level document processing request
type DocumentRequest struct {
	Action      string                 `json:"action"` // "analyze", "summarize", "extract"
	DocumentID  string                 `json:"document_id,omitempty"`
	Content     string                 `json:"content,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// DocumentResponse represents the response from document processing
type DocumentResponse struct {
	Result      string        `json:"result"`
	DocumentID  string        `json:"document_id"`
	Action      string        `json:"action"`
	Duration    time.Duration `json:"duration"`
	Usage       TokenUsage    `json:"usage"`
}

// TaskRequest represents a high-level task execution request
type TaskRequest struct {
	Task        string                 `json:"task"`
	Agent       string                 `json:"agent,omitempty"`
	Workflow    string                 `json:"workflow,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// TaskResponse represents the response from task execution
type TaskResponse struct {
	Result      string        `json:"result"`
	Task        string        `json:"task"`
	Duration    time.Duration `json:"duration"`
	Steps       []StepResult  `json:"steps,omitempty"`
	Usage       TokenUsage    `json:"usage,omitempty"`
}

// ===== CONFIGURATION SUB-TYPES =====

// LoadBalancingConfig defines load balancing settings
type LoadBalancingConfig struct {
	Strategy      string        `toml:"strategy"` // "round_robin", "weighted", "least_connections"
	HealthCheck   bool          `toml:"health_check"`
	CheckInterval time.Duration `toml:"check_interval"`
}

// HealthCheckConfig defines health checking settings
type HealthCheckConfig struct {
	Enabled   bool          `toml:"enabled"`
	Interval  time.Duration `toml:"interval"`
	Timeout   time.Duration `toml:"timeout"`
	Retries   int           `toml:"retries"`
}

// ChunkingConfig defines chunking strategy settings
type ChunkingConfig struct {
	Strategy    string `toml:"strategy"` // "fixed", "sentence", "paragraph", "semantic"
	ChunkSize   int    `toml:"chunk_size"`
	ChunkOverlap int   `toml:"chunk_overlap"`
	MinChunkSize int   `toml:"min_chunk_size"`
}

// VectorStoreConfig defines vector storage settings
type VectorStoreConfig struct {
	Backend    string `toml:"backend"` // "sqvect", "chroma", "qdrant"
	Dimensions int    `toml:"dimensions"`
	Metric     string `toml:"metric"` // "cosine", "euclidean", "dot_product"
	IndexType  string `toml:"index_type"`
}

// KeywordStoreConfig defines keyword storage settings
type KeywordStoreConfig struct {
	Backend   string   `toml:"backend"` // "bleve", "elasticsearch"
	Analyzer  string   `toml:"analyzer"`
	Languages []string `toml:"languages"`
	Stemming  bool     `toml:"stemming"`
}

// SearchConfig defines search settings
type SearchConfig struct {
	DefaultLimit    int     `toml:"default_limit"`
	MaxLimit        int     `toml:"max_limit"`
	DefaultThreshold float32 `toml:"default_threshold"`
	HybridWeights   struct {
		Vector  float32 `toml:"vector"`
		Keyword float32 `toml:"keyword"`
	} `toml:"hybrid_weights"`
}

// EmbeddingConfig defines embedding settings
type EmbeddingConfig struct {
	Provider   string `toml:"provider"`
	Model      string `toml:"model"`
	Dimensions int    `toml:"dimensions"`
	BatchSize  int    `toml:"batch_size"`
}

// ToolExecutionConfig defines tool execution settings
type ToolExecutionConfig struct {
	MaxConcurrent  int           `toml:"max_concurrent"`
	DefaultTimeout time.Duration `toml:"default_timeout"`
	EnableCache    bool          `toml:"enable_cache"`
	CacheTTL       time.Duration `toml:"cache_ttl"`
}

// WorkflowEngineConfig defines workflow engine settings
type WorkflowEngineConfig struct {
	MaxSteps         int           `toml:"max_steps"`
	StepTimeout      time.Duration `toml:"step_timeout"`
	StateBackend     string        `toml:"state_backend"`
	EnableRecovery   bool          `toml:"enable_recovery"`
}

// SchedulingConfig defines scheduling settings
type SchedulingConfig struct {
	Backend       string `toml:"backend"` // "memory", "redis", "database"
	MaxConcurrent int    `toml:"max_concurrent"`
	QueueSize     int    `toml:"queue_size"`
}

// StateStorageConfig defines state storage settings
type StateStorageConfig struct {
	Backend    string        `toml:"backend"` // "memory", "file", "database"
	Persistent bool          `toml:"persistent"`
	TTL        time.Duration `toml:"ttl"`
}

// ReasoningChainsConfig defines reasoning chain settings
type ReasoningChainsConfig struct {
	MaxSteps      int           `toml:"max_steps"`
	MaxMemorySize int           `toml:"max_memory_size"`
	StepTimeout   time.Duration `toml:"step_timeout"`
}

// RRFParams defines Reciprocal Rank Fusion parameters
type RRFParams struct {
	K float32 `json:"k"` // RRF parameter, typically 60
}