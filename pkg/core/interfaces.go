// Package core defines the fundamental interfaces and contracts for all RAGO pillars.
// This package establishes the four-pillar architecture: LLM, RAG, MCP, and Agents.
package core

import "context"

// Client is the primary interface for all RAGO functionality.
// It provides unified access to all four pillars while supporting individual pillar usage.
type Client interface {
	// Pillar Access - each pillar is independently accessible
	LLM() LLMService
	RAG() RAGService
	MCP() MCPService
	Agents() AgentService

	// High-Level Operations - coordinated operations using multiple pillars
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	StreamChat(ctx context.Context, req ChatRequest, callback StreamCallback) error
	ProcessDocument(ctx context.Context, req DocumentRequest) (*DocumentResponse, error)
	ExecuteTask(ctx context.Context, req TaskRequest) (*TaskResponse, error)

	// Client Management
	Close() error
	Health() HealthReport
}

// Individual pillar clients for library usage - each pillar can be used independently
type LLMClient interface {
	LLMService
	Close() error
}

type RAGClient interface {
	RAGService
	Close() error
}

type MCPClient interface {
	MCPService
	Close() error
}

type AgentClient interface {
	AgentService
	Close() error
}

// LLMService defines the interface for the LLM pillar.
// Focuses on provider management, load balancing, and generation operations.
type LLMService interface {
	// Provider Management
	AddProvider(name string, config ProviderConfig) error
	RemoveProvider(name string) error
	ListProviders() []ProviderInfo
	GetProviderHealth() map[string]HealthStatus

	// Generation Operations
	Generate(ctx context.Context, req GenerationRequest) (*GenerationResponse, error)
	Stream(ctx context.Context, req GenerationRequest, callback StreamCallback) error

	// Tool Operations - coordination with MCP pillar
	GenerateWithTools(ctx context.Context, req ToolGenerationRequest) (*ToolGenerationResponse, error)
	StreamWithTools(ctx context.Context, req ToolGenerationRequest, callback ToolStreamCallback) error

	// Batch Operations
	GenerateBatch(ctx context.Context, requests []GenerationRequest) ([]GenerationResponse, error)
}

// RAGService defines the interface for the RAG pillar.
// Focuses on document ingestion, storage, and retrieval operations.
type RAGService interface {
	// Document Operations
	IngestDocument(ctx context.Context, req IngestRequest) (*IngestResponse, error)
	IngestBatch(ctx context.Context, requests []IngestRequest) (*BatchIngestResponse, error)
	DeleteDocument(ctx context.Context, docID string) error
	ListDocuments(ctx context.Context, filter DocumentFilter) ([]Document, error)

	// Search Operations
	Search(ctx context.Context, req SearchRequest) (*SearchResponse, error)
	HybridSearch(ctx context.Context, req HybridSearchRequest) (*HybridSearchResponse, error)

	// Management Operations
	GetStats(ctx context.Context) (*RAGStats, error)
	Optimize(ctx context.Context) error
	Reset(ctx context.Context) error
}

// MCPService defines the interface for the MCP pillar.
// Focuses on tool integration and external service coordination.
type MCPService interface {
	// Server Management
	RegisterServer(config ServerConfig) error
	UnregisterServer(name string) error
	ListServers() []ServerInfo
	GetServerHealth(name string) HealthStatus

	// Tool Operations
	ListTools() []ToolInfo
	GetTool(name string) (*ToolInfo, error)
	CallTool(ctx context.Context, req ToolCallRequest) (*ToolCallResponse, error)
	CallToolAsync(ctx context.Context, req ToolCallRequest) (<-chan *ToolCallResponse, error)

	// Batch Operations
	CallToolsBatch(ctx context.Context, requests []ToolCallRequest) ([]ToolCallResponse, error)
}

// AgentService defines the interface for the Agent pillar.
// Focuses on workflow orchestration and multi-step reasoning.
type AgentService interface {
	// Workflow Management
	CreateWorkflow(definition WorkflowDefinition) error
	ExecuteWorkflow(ctx context.Context, req WorkflowRequest) (*WorkflowResponse, error)
	ListWorkflows() []WorkflowInfo
	DeleteWorkflow(name string) error

	// Agent Management
	CreateAgent(definition AgentDefinition) error
	ExecuteAgent(ctx context.Context, req AgentRequest) (*AgentResponse, error)
	ListAgents() []AgentInfo
	DeleteAgent(name string) error

	// Scheduling
	ScheduleWorkflow(name string, schedule ScheduleConfig) error
	GetScheduledTasks() []ScheduledTask
}

// Callback interfaces for streaming operations
type StreamCallback func(chunk StreamChunk) error
type ToolStreamCallback func(chunk ToolStreamChunk) error