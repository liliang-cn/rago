// Package client provides the unified client interface for all RAGO pillars.
// This package implements the four-pillar architecture allowing access to
// LLM, RAG, MCP, and Agent services through a single interface.
package client

import (
	"context"
	
	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/llm"
	"github.com/liliang-cn/rago/v2/pkg/rag"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/agents"
)

// Client implements the unified RAGO client interface.
// It provides access to all four pillars and high-level operations.
type Client struct {
	config core.Config
	
	// Individual pillar services
	llmService    *llm.Service
	ragService    *rag.Service
	mcpService    *mcp.Service
	agentService  *agents.Service
}

// New creates a new unified RAGO client with the provided configuration.
func New(config core.Config) (*Client, error) {
	client := &Client{
		config: config,
	}
	
	var err error
	
	// Initialize LLM pillar if not disabled
	if !config.Mode.LLMOnly || !config.Mode.DisableAgent {
		client.llmService, err = llm.NewService(config.LLM)
		if err != nil {
			return nil, core.WrapErrorWithContext(err, "client", "New", "failed to initialize LLM service")
		}
	}
	
	// Initialize RAG pillar if not disabled
	if !config.Mode.RAGOnly || !config.Mode.DisableAgent {
		client.ragService, err = rag.NewService(config.RAG)
		if err != nil {
			return nil, core.WrapErrorWithContext(err, "client", "New", "failed to initialize RAG service")
		}
	}
	
	// Initialize MCP pillar if not disabled
	if !config.Mode.DisableMCP {
		client.mcpService, err = mcp.NewService(config.MCP)
		if err != nil {
			return nil, core.WrapErrorWithContext(err, "client", "New", "failed to initialize MCP service")
		}
	}
	
	// Initialize Agent pillar if not disabled
	if !config.Mode.DisableAgent {
		client.agentService, err = agents.NewService(config.Agents)
		if err != nil {
			return nil, core.WrapErrorWithContext(err, "client", "New", "failed to initialize Agent service")
		}
	}
	
	return client, nil
}

// ===== PILLAR ACCESS =====

// LLM returns the LLM service interface.
func (c *Client) LLM() core.LLMService {
	return c.llmService
}

// RAG returns the RAG service interface.
func (c *Client) RAG() core.RAGService {
	return c.ragService
}

// MCP returns the MCP service interface.
func (c *Client) MCP() core.MCPService {
	return c.mcpService
}

// Agents returns the Agent service interface.
func (c *Client) Agents() core.AgentService {
	return c.agentService
}

// ===== HIGH-LEVEL OPERATIONS =====

// Chat provides a high-level chat interface using multiple pillars.
func (c *Client) Chat(ctx context.Context, req core.ChatRequest) (*core.ChatResponse, error) {
	// TODO: Implement high-level chat using RAG + LLM + MCP
	return nil, core.ErrServiceUnavailable
}

// StreamChat provides a streaming chat interface using multiple pillars.
func (c *Client) StreamChat(ctx context.Context, req core.ChatRequest, callback core.StreamCallback) error {
	// TODO: Implement streaming chat using RAG + LLM + MCP
	return core.ErrServiceUnavailable
}

// ProcessDocument provides high-level document processing using multiple pillars.
func (c *Client) ProcessDocument(ctx context.Context, req core.DocumentRequest) (*core.DocumentResponse, error) {
	// TODO: Implement document processing using RAG + LLM + Agents
	return nil, core.ErrServiceUnavailable
}

// ExecuteTask provides high-level task execution using multiple pillars.
func (c *Client) ExecuteTask(ctx context.Context, req core.TaskRequest) (*core.TaskResponse, error) {
	// TODO: Implement task execution using Agents + MCP + LLM
	return nil, core.ErrServiceUnavailable
}

// ===== CONVENIENCE CONSTRUCTORS =====

// NewFromPath creates a client from a configuration file path.
func NewFromPath(configPath string) (core.Client, error) {
	config, err := LoadCoreConfigFromPath(configPath)
	if err != nil {
		return nil, core.WrapError(err, "failed to load configuration")
	}
	return New(*config)
}

// NewDefault creates a client from default configuration locations.
func NewDefault() (core.Client, error) {
	config, err := LoadDefaultCoreConfig()
	if err != nil {
		return nil, core.WrapError(err, "failed to load default configuration")
	}
	return New(*config)
}

// ===== CLIENT MANAGEMENT =====

// Close closes all pillar services and cleans up resources.
func (c *Client) Close() error {
	var lastErr error
	
	if c.llmService != nil {
		if err := c.llmService.Close(); err != nil {
			lastErr = err
		}
	}
	
	if c.ragService != nil {
		if err := c.ragService.Close(); err != nil {
			lastErr = err
		}
	}
	
	if c.mcpService != nil {
		if err := c.mcpService.Close(); err != nil {
			lastErr = err
		}
	}
	
	if c.agentService != nil {
		if err := c.agentService.Close(); err != nil {
			lastErr = err
		}
	}
	
	return lastErr
}

// Health returns the health status of all pillars.
func (c *Client) Health() core.HealthReport {
	report := core.HealthReport{
		Overall: core.HealthStatusHealthy,
		Pillars: make(map[string]core.HealthStatus),
		Providers: make(map[string]core.HealthStatus),
		Servers: make(map[string]core.HealthStatus),
		Details: make(map[string]interface{}),
	}
	
	// TODO: Implement comprehensive health checking across all pillars
	
	return report
}