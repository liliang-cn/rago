package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
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

	// Create transport based on server type
	var transport mcp.Transport
	var err error

	switch c.config.Type {
	case ServerTypeHTTP:
		// Create HTTP transport for HTTP-based MCP servers
		if c.config.URL == "" {
			return fmt.Errorf("URL is required for HTTP MCP server %s", c.config.Name)
		}
		transport, err = c.createHTTPTransport()
		if err != nil {
			return fmt.Errorf("failed to create HTTP transport for %s: %w", c.config.Name, err)
		}

	case ServerTypeStdio, "":
		// Default to stdio for backward compatibility
		if len(c.config.Command) == 0 {
			return fmt.Errorf("command is required for stdio MCP server %s", c.config.Name)
		}
		transport, err = c.createStdioTransport(ctx)
		if err != nil {
			return fmt.Errorf("failed to create stdio transport for %s: %w", c.config.Name, err)
		}

	default:
		return fmt.Errorf("unsupported server type: %s", c.config.Type)
	}

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

// createStdioTransport creates a command transport for stdio-based servers
func (c *Client) createStdioTransport(ctx context.Context) (mcp.Transport, error) {
	// Create command for the MCP server
	cmd := exec.CommandContext(ctx, c.config.Command[0], c.config.Args...)

	// Set working directory if specified
	if c.config.WorkingDir != "" {
		cmd.Dir = c.config.WorkingDir
	}

	// Set environment variables - inherit parent environment and add custom ones
	cmd.Env = os.Environ()
	if len(c.config.Env) > 0 {
		for key, value := range c.config.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	// Create command transport
	return &mcp.CommandTransport{Command: cmd}, nil
}

// createHTTPTransport creates an HTTP transport for HTTP-based servers
func (c *Client) createHTTPTransport() (mcp.Transport, error) {
	// MCP SDK provides StreamableClientTransport for HTTP connections
	// This uses Server-Sent Events (SSE) for bidirectional communication
	
	// Create HTTP client with custom headers if needed
	httpClient := &http.Client{}
	
	// If headers are specified, we'll need to wrap the HTTP client
	// to add them to each request (this would need custom RoundTripper)
	if len(c.config.Headers) > 0 {
		httpClient.Transport = &headerTransport{
			headers: c.config.Headers,
			base:    http.DefaultTransport,
		}
	}
	
	transport := &mcp.StreamableClientTransport{
		Endpoint:   c.config.URL,
		HTTPClient: httpClient,
		MaxRetries: 5,
	}

	return transport, nil
}

// headerTransport adds custom headers to all HTTP requests
type headerTransport struct {
	headers map[string]string
	base    http.RoundTripper
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	req = req.Clone(req.Context())
	
	// Add custom headers
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}
	
	return t.base.RoundTrip(req)
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
	loadedServers := m.config.GetLoadedServers()
	for _, config := range loadedServers {
		if config.Name == serverName {
			cfg := config // Create a copy to get a stable address
			serverConfig = &cfg
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

// ========================================
// Agent Layer Helper Methods
// ========================================

// AgentToolInfo represents tool information for agent layer
type AgentToolInfo struct {
	Name        string                 // Full prefixed name (mcp_server_tool)
	ServerName  string                 // Server that provides this tool
	ActualName  string                 // Actual tool name on server (without prefix)
	Description string                 // Tool description
	Parameters  []string               // List of parameter names (for backward compatibility)
	InputSchema map[string]interface{} // Full parameter schema for LLM
}

// GetAvailableTools returns structured information about all available tools
func (m *Manager) GetAvailableTools(ctx context.Context) []AgentToolInfo {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var tools []AgentToolInfo

	for serverName, client := range m.clients {
		if client == nil || !client.IsConnected() {
			continue
		}

		serverTools := client.GetTools()
		for toolName, tool := range serverTools {
			// Extract parameters from InputSchema
			var params []string
			var inputSchema map[string]interface{}

			if tool.InputSchema != nil {
				// Convert InputSchema to map for easier access
				if schemaBytes, err := json.Marshal(tool.InputSchema); err == nil {
					var schemaMap map[string]interface{}
					if err := json.Unmarshal(schemaBytes, &schemaMap); err == nil {
						inputSchema = schemaMap
						if props, ok := schemaMap["properties"].(map[string]interface{}); ok {
							for paramName := range props {
								params = append(params, paramName)
							}
						}
					}
				}
			}

			tools = append(tools, AgentToolInfo{
				Name:        fmt.Sprintf("mcp_%s_%s", serverName, toolName),
				ServerName:  serverName,
				ActualName:  toolName,
				Description: tool.Description,
				Parameters:  params,
				InputSchema: inputSchema,
			})
		}
	}

	return tools
}

// GetToolsDescription returns a formatted string description of all available tools
func (m *Manager) GetToolsDescription(ctx context.Context) string {
	tools := m.GetAvailableTools(ctx)
	
	if len(tools) == 0 {
		return "No MCP tools available"
	}
	
	var descriptions []string
	for _, tool := range tools {
		desc := fmt.Sprintf("- %s: %s", tool.Name, tool.Description)
		if len(tool.Parameters) > 0 {
			desc += fmt.Sprintf(" | Parameters: %s", strings.Join(tool.Parameters, ", "))
		}
		descriptions = append(descriptions, desc)
	}
	
	return strings.Join(descriptions, "\n")
}

// FindToolProvider finds which server provides a tool and returns the server info
func (m *Manager) FindToolProvider(ctx context.Context, toolName string) (*AgentToolInfo, *Client, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	// Handle both formats: "mcp_server_tool" and "tool"
	for serverName, client := range m.clients {
		if client == nil || !client.IsConnected() {
			continue
		}
		
		tools := client.GetTools()
		for actualToolName, tool := range tools {
			// Check exact match
			if actualToolName == toolName {
				return &AgentToolInfo{
					Name:        fmt.Sprintf("mcp_%s_%s", serverName, actualToolName),
					ServerName:  serverName,
					ActualName:  actualToolName,
					Description: tool.Description,
				}, client, nil
			}
			
			// Check prefixed format match (mcp_server_tool)
			prefixedName := fmt.Sprintf("mcp_%s_%s", serverName, actualToolName)
			if prefixedName == toolName {
				return &AgentToolInfo{
					Name:        prefixedName,
					ServerName:  serverName,
					ActualName:  actualToolName,
					Description: tool.Description,
				}, client, nil
			}
		}
	}
	
	return nil, nil, fmt.Errorf("tool %s not found in any connected MCP server", toolName)
}

// CallTool finds the appropriate server and calls the tool
func (m *Manager) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*ToolResult, error) {
	// First try to find in already connected clients
	toolInfo, client, err := m.FindToolProvider(ctx, toolName)
	if err != nil {
		// Tool not found in connected clients, try to start the server on-demand
		// Extract server name from tool name (format: mcp_server_tool or server_tool)
		serverName := m.extractServerNameFromTool(toolName)
		if serverName == "" {
			return nil, fmt.Errorf("tool %s not found in any connected MCP server", toolName)
		}

		// Try to start the server
		_, startErr := m.StartServer(ctx, serverName)
		if startErr != nil {
			return nil, fmt.Errorf("tool %s not found, failed to start server %s: %w", toolName, serverName, startErr)
		}

		// Try again to find the tool
		toolInfo, client, err = m.FindToolProvider(ctx, toolName)
		if err != nil {
			return nil, fmt.Errorf("tool %s still not found after starting server %s: %w", toolName, serverName, err)
		}
	}

	// Call the tool with its actual name (without prefix)
	return client.CallTool(ctx, toolInfo.ActualName, arguments)
}

// extractServerNameFromTool extracts server name from tool name
// Handles formats: "mcp_server_tool", "server_tool", "tool"
func (m *Manager) extractServerNameFromTool(toolName string) string {
	// Try mcp_server_tool format first
	if strings.HasPrefix(toolName, "mcp_") {
		parts := strings.SplitN(toolName, "_", 3) // mcp_server_tool
		if len(parts) >= 2 {
			return parts[1]
		}
	}

	// Check if it matches any configured server's tools
	m.mutex.RLock()
	loadedServers := m.config.GetLoadedServers()
	m.mutex.RUnlock()

	for _, serverConfig := range loadedServers {
		expectedPrefix := fmt.Sprintf("mcp_%s_", serverConfig.Name)
		if strings.HasPrefix(toolName, expectedPrefix) {
			return serverConfig.Name
		}
	}

	return ""
}
