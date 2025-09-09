package mcp

import (
	"context"
	"fmt"
	"os/exec"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewClient creates a new MCP client for the given server configuration
func NewClient(config *ServerConfig) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("server config cannot be nil")
	}

	if len(config.Command) == 0 {
		return nil, fmt.Errorf("server command cannot be empty")
	}

	return &Client{
		config:    config,
		tools:     make(map[string]*mcp.Tool),
		connected: false,
	}, nil
}

// Connect establishes connection to the MCP server
func (c *Client) Connect(ctx context.Context) error {
	if c.connected {
		return nil
	}

	// Create command for the MCP server
	cmd := exec.CommandContext(ctx, c.config.Command[0], c.config.Args...)

	// Set working directory if specified
	if c.config.WorkingDir != "" {
		cmd.Dir = c.config.WorkingDir
	}

	// Set environment variables
	if len(c.config.Env) > 0 {
		for key, value := range c.config.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	// Create command transport
	transport := &mcp.CommandTransport{Command: cmd}

	// Create MCP client with implementation info
	clientImpl := &mcp.Implementation{
		Name:    "rago",
		Version: "1.0.0",
	}
	clientOpts := &mcp.ClientOptions{}
	client := mcp.NewClient(clientImpl, clientOpts)

	// Connect and get session
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to MCP server %s: %w", c.config.Name, err)
	}

	c.session = session
	c.connected = true

	// Load tools
	if err := c.loadTools(ctx); err != nil {
		return fmt.Errorf("failed to load tools from %s: %w", c.config.Name, err)
	}

	return nil
}

// loadTools fetches and caches the available tools from the server
func (c *Client) loadTools(ctx context.Context) error {
	if !c.connected || c.session == nil {
		return fmt.Errorf("client not connected")
	}

	toolsResponse, err := c.session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	// Clear existing tools and load new ones
	c.tools = make(map[string]*mcp.Tool)
	for _, tool := range toolsResponse.Tools {
		c.tools[tool.Name] = tool
	}

	return nil
}

// GetTools returns the available tools from this server
func (c *Client) GetTools() map[string]*mcp.Tool {
	return c.tools
}

// CallTool calls a tool on the MCP server
func (c *Client) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*ToolResult, error) {
	if !c.connected || c.session == nil {
		return nil, fmt.Errorf("client not connected")
	}

	// Check if tool exists
	tool, exists := c.tools[toolName]
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found on server '%s'", toolName, c.config.Name)
	}

	// Call the tool
	response, err := c.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      tool.Name,
		Arguments: arguments,
	})
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   fmt.Sprintf("tool call failed: %v", err),
		}, nil
	}

	// Process response - handle different content types
	var data interface{}
	if len(response.Content) > 0 {
		content := response.Content[0]
		// Try to cast to TextContent
		if textContent, ok := content.(*mcp.TextContent); ok {
			data = textContent.Text
		} else {
			// For other content types, return as-is
			data = content
		}
	}

	return &ToolResult{
		Success: true,
		Data:    data,
	}, nil
}

// Close closes the connection to the MCP server
func (c *Client) Close() error {
	if c.connected {
		var err error
		if c.session != nil {
			// Close the session
			err = c.session.Close()
		}
		c.connected = false
		c.session = nil
		return err
	}
	return nil
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	return c.connected
}

// GetServerInfo returns information about the connected server
func (c *Client) GetServerInfo() *mcp.Implementation {
	if c.session != nil {
		return c.session.InitializeResult().ServerInfo
	}
	return nil
}

// Manager manages multiple MCP clients
type Manager struct {
	clients map[string]*Client
	config  *Config
	mutex   sync.RWMutex
}

// NewManager creates a new MCP manager
func NewManager(config *Config) *Manager {
	if config == nil {
		defaultConfig := DefaultConfig()
		config = &defaultConfig
	}

	return &Manager{
		clients: make(map[string]*Client),
		config:  config,
	}
}

// StartServer starts an MCP server and creates a client connection
func (m *Manager) StartServer(ctx context.Context, serverName string) (*Client, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if client already exists
	if client, exists := m.clients[serverName]; exists {
		if client.IsConnected() {
			return client, nil
		}
		// Remove disconnected client
		delete(m.clients, serverName)
	}

	// Find server config
	var serverConfig *ServerConfig
	for _, config := range m.config.LoadedServers {
		if config.Name == serverName {
			serverConfig = &config
			break
		}
	}

	if serverConfig == nil {
		return nil, fmt.Errorf("server configuration not found: %s", serverName)
	}

	// Create and connect client
	client, err := NewClient(serverConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create client for %s: %w", serverName, err)
	}

	if err := client.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", serverName, err)
	}

	m.clients[serverName] = client
	return client, nil
}

// GetClient returns an existing client by server name
func (m *Manager) GetClient(serverName string) (*Client, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	client, exists := m.clients[serverName]
	return client, exists
}

// ListClients returns all active clients
func (m *Manager) ListClients() map[string]*Client {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	clients := make(map[string]*Client)
	for name, client := range m.clients {
		clients[name] = client
	}
	return clients
}

// StopServer stops an MCP server and closes its client connection
func (m *Manager) StopServer(serverName string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	client, exists := m.clients[serverName]
	if !exists {
		return fmt.Errorf("server not found: %s", serverName)
	}

	err := client.Close()
	delete(m.clients, serverName)
	return err
}

// Close closes all MCP clients
func (m *Manager) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var lastError error
	for name, client := range m.clients {
		if err := client.Close(); err != nil {
			lastError = fmt.Errorf("failed to close client %s: %w", name, err)
		}
	}

	m.clients = make(map[string]*Client)
	return lastError
}
