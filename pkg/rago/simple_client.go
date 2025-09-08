// Package rago provides a simplified client for the RAGO system
package rago

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
)

// SimpleClient provides a simplified interface to RAGO with MCP integration
type SimpleClient struct {
	config     *config.Config
	mcpManager *mcp.MCPToolManager
	mcpEnabled bool
}

// NewSimpleClient creates a new simple RAGO client with MCP enabled by default
func NewSimpleClient() (*SimpleClient, error) {
	// Load or create config
	cfg, err := config.Load("")
	if err != nil {
		// Create a minimal default config
		cfg = &config.Config{}
	}

	// Ensure MCP is enabled
	cfg.MCP.Enabled = true

	// Add default MCP servers if not configured
	ensureDefaultServers(cfg)

	client := &SimpleClient{
		config:     cfg,
		mcpEnabled: true,
	}

	// Initialize MCP manager
	client.mcpManager = mcp.NewMCPToolManager(&cfg.MCP)

	// Start MCP servers
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.mcpManager.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to start MCP servers: %v\n", err)
	}

	return client, nil
}

// ensureDefaultServers ensures default MCP servers are configured
func ensureDefaultServers(cfg *config.Config) {
	// Default server configurations that would be in mcpServers.json
	defaultServers := []mcp.ServerConfig{
		{
			Name:        "filesystem",
			Command:     []string{"npx", "@modelcontextprotocol/server-filesystem", "--allowed-directories", "./", "/tmp"},
			Description: "File system operations",
			AutoStart:   true,
		},
		{
			Name:        "fetch",
			Command:     []string{"npx", "@modelcontextprotocol/server-fetch"},
			Description: "HTTP/HTTPS fetch operations",
			AutoStart:   true,
		},
		{
			Name:        "memory",
			Command:     []string{"npx", "@modelcontextprotocol/server-memory"},
			Description: "In-memory key-value storage",
			AutoStart:   true,
		},
		{
			Name:        "time",
			Command:     []string{"npx", "@modelcontextprotocol/server-time"},
			Description: "Time and date utilities",
			AutoStart:   true,
		},
	}

	// Since Servers is now []string (file paths), we need to handle this differently
	// If no server files are configured, we'll add the default servers directly to LoadedServers
	if cfg.MCP.Servers == nil || len(cfg.MCP.Servers) == 0 {
		// No server configuration files specified, use defaults
		cfg.MCP.LoadedServers = defaultServers
	} else {
		// Load servers from JSON files first
		if err := cfg.MCP.LoadServersFromJSON(); err == nil {
			// Add missing default servers to LoadedServers
			existing := make(map[string]bool)
			for _, s := range cfg.MCP.LoadedServers {
				existing[s.Name] = true
			}

			for _, s := range defaultServers {
				if !existing[s.Name] {
					cfg.MCP.LoadedServers = append(cfg.MCP.LoadedServers, s)
				}
			}
		} else {
			// If loading fails, use defaults
			cfg.MCP.LoadedServers = defaultServers
		}
	}
}

// ListTools returns available MCP tools
func (c *SimpleClient) ListTools() map[string]*mcp.MCPToolWrapper {
	if c.mcpManager == nil {
		return nil
	}
	return c.mcpManager.ListTools()
}

// CallTool calls an MCP tool
func (c *SimpleClient) CallTool(ctx context.Context, toolName string, args interface{}) (interface{}, error) {
	if c.mcpManager == nil {
		return nil, fmt.Errorf("MCP manager not initialized")
	}

	// Call the tool through the MCP manager
	argsMap, ok := args.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("args must be a map[string]interface{}")
	}
	return c.mcpManager.CallTool(ctx, toolName, argsMap)
}

// GetServerStatus returns the status of MCP servers
func (c *SimpleClient) GetServerStatus() map[string]bool {
	if c.mcpManager == nil {
		return nil
	}
	return c.mcpManager.GetServerStatus()
}

// Close closes the client
func (c *SimpleClient) Close() error {
	if c.mcpManager != nil {
		return c.mcpManager.Close()
	}
	return nil
}
