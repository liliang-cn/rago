package client

import (
	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/llm"
	"github.com/liliang-cn/rago/v2/pkg/rag"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/agents"
)

// ===== INDIVIDUAL PILLAR CLIENTS =====

// LLMClient implements the individual LLM pillar client.
type LLMClient struct {
	*llm.Service
}

// NewLLMClient creates a new standalone LLM client.
func NewLLMClient(config core.LLMConfig) (core.LLMClient, error) {
	service, err := llm.NewService(config)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "factory", "NewLLMClient", "failed to create LLM service")
	}
	
	return &LLMClient{Service: service}, nil
}

// Close closes the LLM client and cleans up resources.
func (c *LLMClient) Close() error {
	return c.Service.Close()
}

// RAGClient implements the individual RAG pillar client.
type RAGClient struct {
	*rag.Service
}

// NewRAGClient creates a new standalone RAG client.
func NewRAGClient(config core.RAGConfig) (core.RAGClient, error) {
	service, err := rag.NewService(config)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "factory", "NewRAGClient", "failed to create RAG service")
	}
	
	return &RAGClient{Service: service}, nil
}

// Close closes the RAG client and cleans up resources.
func (c *RAGClient) Close() error {
	return c.Service.Close()
}

// MCPClient implements the individual MCP pillar client.
type MCPClient struct {
	*mcp.Service
}

// NewMCPClient creates a new standalone MCP client.
func NewMCPClient(config core.MCPConfig) (core.MCPClient, error) {
	service, err := mcp.NewService(config)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "factory", "NewMCPClient", "failed to create MCP service")
	}
	
	return &MCPClient{Service: service}, nil
}

// Close closes the MCP client and cleans up resources.
func (c *MCPClient) Close() error {
	return c.Service.Close()
}

// AgentClient implements the individual Agent pillar client.
type AgentClient struct {
	*agents.Service
}

// NewAgentClient creates a new standalone Agent client.
func NewAgentClient(config core.AgentsConfig) (core.AgentClient, error) {
	service, err := agents.NewService(config)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "factory", "NewAgentClient", "failed to create Agent service")
	}
	
	return &AgentClient{Service: service}, nil
}

// Close closes the Agent client and cleans up resources.
func (c *AgentClient) Close() error {
	return c.Service.Close()
}

// ===== FACTORY FUNCTIONS =====

// NewFromConfig creates the appropriate client based on configuration mode.
func NewFromConfig(config core.Config) (core.Client, error) {
	// Check for single-pillar modes
	if config.Mode.LLMOnly {
		return NewLLMOnlyClient(config)
	}
	
	if config.Mode.RAGOnly {
		return NewRAGOnlyClient(config)
	}
	
	// Create full unified client
	return New(config)
}

// NewLLMOnlyClient creates a client with only LLM pillar enabled.
func NewLLMOnlyClient(config core.Config) (core.Client, error) {
	// Disable other pillars
	config.Mode.DisableMCP = true
	config.Mode.DisableAgent = true
	
	return New(config)
}

// NewRAGOnlyClient creates a client with only RAG pillar enabled.
func NewRAGOnlyClient(config core.Config) (core.Client, error) {
	// Disable other pillars
	config.Mode.DisableMCP = true
	config.Mode.DisableAgent = true
	
	return New(config)
}

// NewMinimalClient creates a client with minimal pillars (LLM + RAG only).
func NewMinimalClient(config core.Config) (core.Client, error) {
	// Disable optional pillars
	config.Mode.DisableMCP = true
	config.Mode.DisableAgent = true
	
	return New(config)
}