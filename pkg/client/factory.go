// Package client - factory.go
// This file provides factory functions for creating individual pillar clients,
// enabling library users to use specific RAGO functionality without initializing unused pillars.

package client

import (
	"context"
	"fmt"

	"github.com/liliang-cn/rago/v2/pkg/agents"
	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/llm"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
)

// ===== INDIVIDUAL PILLAR CLIENT IMPLEMENTATIONS =====

// llmClient implements core.LLMClient for LLM-only usage
type llmClient struct {
	service core.LLMService
	ctx     context.Context
	cancel  context.CancelFunc
}

func (c *llmClient) AddProvider(name string, config core.ProviderConfig) error {
	return c.service.AddProvider(name, config)
}

func (c *llmClient) RemoveProvider(name string) error {
	return c.service.RemoveProvider(name)
}

func (c *llmClient) ListProviders() []core.ProviderInfo {
	return c.service.ListProviders()
}

func (c *llmClient) GetProviderHealth() map[string]core.HealthStatus {
	return c.service.GetProviderHealth()
}

func (c *llmClient) Generate(ctx context.Context, req core.GenerationRequest) (*core.GenerationResponse, error) {
	return c.service.Generate(ctx, req)
}

func (c *llmClient) Stream(ctx context.Context, req core.GenerationRequest, callback core.StreamCallback) error {
	return c.service.Stream(ctx, req, callback)
}

func (c *llmClient) GenerateWithTools(ctx context.Context, req core.ToolGenerationRequest) (*core.ToolGenerationResponse, error) {
	return c.service.GenerateWithTools(ctx, req)
}

func (c *llmClient) StreamWithTools(ctx context.Context, req core.ToolGenerationRequest, callback core.ToolStreamCallback) error {
	return c.service.StreamWithTools(ctx, req, callback)
}

func (c *llmClient) GenerateBatch(ctx context.Context, requests []core.GenerationRequest) ([]core.GenerationResponse, error) {
	return c.service.GenerateBatch(ctx, requests)
}

func (c *llmClient) GenerateStructured(ctx context.Context, req core.StructuredGenerationRequest) (*core.StructuredResult, error) {
	return c.service.GenerateStructured(ctx, req)
}

func (c *llmClient) ExtractMetadata(ctx context.Context, req core.MetadataExtractionRequest) (*core.ExtractedMetadata, error) {
	return c.service.ExtractMetadata(ctx, req)
}

func (c *llmClient) Close() error {
	c.cancel()
	if closer, ok := c.service.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

// ragClient implements core.RAGClient for RAG-only usage
type ragClient struct {
	service core.RAGService
	ctx     context.Context
	cancel  context.CancelFunc
}

func (c *ragClient) IngestDocument(ctx context.Context, req core.IngestRequest) (*core.IngestResponse, error) {
	return c.service.IngestDocument(ctx, req)
}

func (c *ragClient) IngestBatch(ctx context.Context, requests []core.IngestRequest) (*core.BatchIngestResponse, error) {
	return c.service.IngestBatch(ctx, requests)
}

func (c *ragClient) DeleteDocument(ctx context.Context, docID string) error {
	return c.service.DeleteDocument(ctx, docID)
}

func (c *ragClient) ListDocuments(ctx context.Context, filter core.DocumentFilter) ([]core.Document, error) {
	return c.service.ListDocuments(ctx, filter)
}

func (c *ragClient) Search(ctx context.Context, req core.SearchRequest) (*core.SearchResponse, error) {
	return c.service.Search(ctx, req)
}

func (c *ragClient) HybridSearch(ctx context.Context, req core.HybridSearchRequest) (*core.HybridSearchResponse, error) {
	return c.service.HybridSearch(ctx, req)
}

func (c *ragClient) GetStats(ctx context.Context) (*core.RAGStats, error) {
	return c.service.GetStats(ctx)
}

func (c *ragClient) Optimize(ctx context.Context) error {
	return c.service.Optimize(ctx)
}

func (c *ragClient) Reset(ctx context.Context) error {
	return c.service.Reset(ctx)
}

func (c *ragClient) Close() error {
	c.cancel()
	if closer, ok := c.service.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

// mcpClient implements core.MCPClient for MCP-only usage
type mcpClient struct {
	service core.MCPService
	ctx     context.Context
	cancel  context.CancelFunc
}

func (c *mcpClient) GetTools() []core.ToolInfo {
	return c.service.GetTools()
}

func (c *mcpClient) GetToolsForLLM() []core.ToolInfo {
	return c.service.GetToolsForLLM()
}

func (c *mcpClient) FindTool(name string) (*core.ToolInfo, bool) {
	return c.service.FindTool(name)
}

func (c *mcpClient) SearchTools(query string) []core.ToolInfo {
	return c.service.SearchTools(query)
}

func (c *mcpClient) GetToolsByCategory(category string) []core.ToolInfo {
	return c.service.GetToolsByCategory(category)
}

func (c *mcpClient) AddTool(tool core.ToolInfo) {
	c.service.AddTool(tool)
}

func (c *mcpClient) ReloadTools() error {
	return c.service.ReloadTools()
}

func (c *mcpClient) GetMetrics() map[string]interface{} {
	return c.service.GetMetrics()
}

func (c *mcpClient) Close() error {
	c.cancel()
	return nil
}

// agentClient implements core.AgentClient for Agent-only usage
type agentClient struct {
	service core.AgentService
	ctx     context.Context
	cancel  context.CancelFunc
}

func (c *agentClient) CreateWorkflow(definition core.WorkflowDefinition) error {
	return c.service.CreateWorkflow(definition)
}

func (c *agentClient) ExecuteWorkflow(ctx context.Context, req core.WorkflowRequest) (*core.WorkflowResponse, error) {
	return c.service.ExecuteWorkflow(ctx, req)
}

func (c *agentClient) ListWorkflows() []core.WorkflowInfo {
	return c.service.ListWorkflows()
}

func (c *agentClient) DeleteWorkflow(name string) error {
	return c.service.DeleteWorkflow(name)
}

func (c *agentClient) CreateAgent(definition core.AgentDefinition) error {
	return c.service.CreateAgent(definition)
}

func (c *agentClient) ExecuteAgent(ctx context.Context, req core.AgentRequest) (*core.AgentResponse, error) {
	return c.service.ExecuteAgent(ctx, req)
}

func (c *agentClient) ListAgents() []core.AgentInfo {
	return c.service.ListAgents()
}

func (c *agentClient) DeleteAgent(name string) error {
	return c.service.DeleteAgent(name)
}

func (c *agentClient) ScheduleWorkflow(name string, schedule core.ScheduleConfig) error {
	return c.service.ScheduleWorkflow(name, schedule)
}

func (c *agentClient) GetScheduledTasks() []core.ScheduledTask {
	return c.service.GetScheduledTasks()
}

func (c *agentClient) Close() error {
	c.cancel()
	if closer, ok := c.service.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

// ===== FACTORY FUNCTIONS =====

// NewLLMClient creates a standalone LLM client for language model operations only.
// This is ideal for applications that only need LLM functionality without RAG, MCP, or Agents.
func NewLLMClient(config core.LLMConfig) (core.LLMClient, error) {
	service, err := llm.NewService(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM service: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	return &llmClient{
		service: service,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

// NewRAGClient creates a standalone RAG client for document operations only.
// This is ideal for applications that only need document ingestion and retrieval.
// NOTE: Currently commented out due to interface mismatch - will be fixed in next iteration
func NewRAGClient(config core.RAGConfig) (core.RAGClient, error) {
	// TODO: Fix interface mismatch - rag.NewService needs different parameters
	return nil, fmt.Errorf("RAG client creation not yet implemented - interface alignment needed")
	
	// service, err := rag.NewService(config)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create RAG service: %w", err)
	// }

	// ctx, cancel := context.WithCancel(context.Background())
	
	// return &ragClient{
	// 	service: service,
	// 	ctx:     ctx,
	// 	cancel:  cancel,
	// }, nil
}

// NewMCPClient creates a standalone MCP client for tool operations only.
// This is ideal for applications that only need external tool integrations.
func NewMCPClient(config core.MCPConfig) (core.MCPClient, error) {
	service, err := mcp.NewService(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP service: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	return &mcpClient{
		service: service,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

// NewAgentClient creates a standalone Agent client for workflow operations only.
// This is ideal for applications that only need agent-based automation.
func NewAgentClient(config core.AgentsConfig) (core.AgentClient, error) {
	service, err := agents.NewService(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Agent service: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	return &agentClient{
		service: service,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

// ===== BUILDER PATTERN FOR CUSTOM CONFIGURATIONS =====

// Builder provides a flexible way to construct RAGO clients with specific pillar combinations.
// This follows the builder pattern for maximum flexibility in client configuration.
type Builder struct {
	llmConfig    *core.LLMConfig
	ragConfig    *core.RAGConfig
	mcpConfig    *core.MCPConfig
	agentsConfig *core.AgentsConfig
	dataDir      string
	logLevel     string
}

// NewBuilder creates a new client builder for custom configurations.
func NewBuilder() *Builder {
	return &Builder{
		dataDir:  "~/.rago",
		logLevel: "info",
	}
}

// WithLLM enables the LLM pillar with the provided configuration.
func (b *Builder) WithLLM(config core.LLMConfig) *Builder {
	b.llmConfig = &config
	return b
}

// WithRAG enables the RAG pillar with the provided configuration.
func (b *Builder) WithRAG(config core.RAGConfig) *Builder {
	b.ragConfig = &config
	return b
}

// WithMCP enables the MCP pillar with the provided configuration.
func (b *Builder) WithMCP(config core.MCPConfig) *Builder {
	b.mcpConfig = &config
	return b
}

// WithAgents enables the Agents pillar with the provided configuration.
func (b *Builder) WithAgents(config core.AgentsConfig) *Builder {
	b.agentsConfig = &config
	return b
}

// WithoutLLM disables the LLM pillar (useful for RAG-only or MCP-only configurations).
func (b *Builder) WithoutLLM() *Builder {
	b.llmConfig = nil
	return b
}

// WithoutRAG disables the RAG pillar (useful for LLM-only or MCP-only configurations).
func (b *Builder) WithoutRAG() *Builder {
	b.ragConfig = nil
	return b
}

// WithoutMCP disables the MCP pillar (useful for LLM+RAG only configurations).
func (b *Builder) WithoutMCP() *Builder {
	b.mcpConfig = nil
	return b
}

// WithoutAgents disables the Agents pillar (useful for simpler configurations).
func (b *Builder) WithoutAgents() *Builder {
	b.agentsConfig = nil
	return b
}

// WithDataDir sets the data directory for all pillars.
func (b *Builder) WithDataDir(dataDir string) *Builder {
	b.dataDir = dataDir
	return b
}

// WithLogLevel sets the logging level.
func (b *Builder) WithLogLevel(level string) *Builder {
	b.logLevel = level
	return b
}

// Build constructs the RAGO client with the configured pillars.
func (b *Builder) Build() (*Client, error) {
	// Validate that at least one pillar is enabled
	if b.llmConfig == nil && b.ragConfig == nil && b.mcpConfig == nil && b.agentsConfig == nil {
		return nil, fmt.Errorf("at least one pillar must be enabled")
	}

	config := &Config{
		Config: core.Config{
			DataDir:  b.dataDir,
			LogLevel: b.logLevel,
			Mode: core.ModeConfig{
				RAGOnly:      b.ragConfig != nil && b.llmConfig == nil && b.mcpConfig == nil && b.agentsConfig == nil,
				LLMOnly:      b.llmConfig != nil && b.ragConfig == nil && b.mcpConfig == nil && b.agentsConfig == nil,
				DisableMCP:   b.mcpConfig == nil,
				DisableAgent: b.agentsConfig == nil,
			},
		},
		ClientName:    "rago-client",
		ClientVersion: "3.0.0",
		LegacyMode:    false,
	}

	// Set pillar configs if provided
	if b.llmConfig != nil {
		config.LLM = *b.llmConfig
	}
	if b.ragConfig != nil {
		config.RAG = *b.ragConfig
	}
	if b.mcpConfig != nil {
		config.MCP = *b.mcpConfig
	}
	if b.agentsConfig != nil {
		config.Agents = *b.agentsConfig
	}

	return NewWithConfig(config)
}

// ===== CONVENIENCE FUNCTIONS FOR COMMON CONFIGURATIONS =====

// NewLLMOnlyClient creates a client with only the LLM pillar enabled.
func NewLLMOnlyClient(llmConfig core.LLMConfig) (*Client, error) {
	return NewBuilder().
		WithLLM(llmConfig).
		WithoutRAG().
		WithoutMCP().
		WithoutAgents().
		Build()
}

// NewRAGOnlyClient creates a client with only the RAG pillar enabled.
func NewRAGOnlyClient(ragConfig core.RAGConfig) (*Client, error) {
	return NewBuilder().
		WithRAG(ragConfig).
		WithoutLLM().
		WithoutMCP().
		WithoutAgents().
		Build()
}

// NewLLMRAGClient creates a client with LLM and RAG pillars enabled.
// This is a common configuration for AI applications with knowledge retrieval.
func NewLLMRAGClient(llmConfig core.LLMConfig, ragConfig core.RAGConfig) (*Client, error) {
	return NewBuilder().
		WithLLM(llmConfig).
		WithRAG(ragConfig).
		WithoutMCP().
		WithoutAgents().
		Build()
}

// NewToolIntegratedClient creates a client with LLM, RAG, and MCP pillars.
// This provides AI capabilities with external tool integration.
func NewToolIntegratedClient(llmConfig core.LLMConfig, ragConfig core.RAGConfig, mcpConfig core.MCPConfig) (*Client, error) {
	return NewBuilder().
		WithLLM(llmConfig).
		WithRAG(ragConfig).
		WithMCP(mcpConfig).
		WithoutAgents().
		Build()
}