package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/config"
)

// ServerManager handles MCP server lifecycle and configuration
type ServerManager struct {
	config        *config.MCPConfig
	servers       map[string]*ServerInstance
	serverConfigs map[string]*ServerConfig
	mu            sync.RWMutex
}

// ServerInstance represents a running MCP server
type ServerInstance struct {
	Name        string
	Config      *ServerConfig
	Process     *exec.Cmd
	Client      *Client
	Status      ServerStatus
	StartedAt   time.Time
	LastPing    time.Time
	ErrorCount  int
}

// ServerConfig represents MCP server configuration
type ServerConfig struct {
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	Description string            `json:"description"`
	Env         map[string]string `json:"env,omitempty"`
	Timeout     int               `json:"timeout,omitempty"`
	AutoStart   bool              `json:"autoStart,omitempty"`
	Required    bool              `json:"required,omitempty"`
}

// ServerStatus represents the status of an MCP server
type ServerStatus string

const (
	ServerStatusStopped  ServerStatus = "stopped"
	ServerStatusStarting ServerStatus = "starting"
	ServerStatusRunning  ServerStatus = "running"
	ServerStatusError    ServerStatus = "error"
	ServerStatusUnknown  ServerStatus = "unknown"
)

// MCPServersConfig represents the complete servers configuration
type MCPServersConfig struct {
	MCPServers    map[string]*ServerConfig `json:"mcpServers"`
	ServerDefaults struct {
		Timeout       int `json:"timeout"`
		RetryAttempts int `json:"retryAttempts"`
		RetryDelay    int `json:"retryDelay"`
	} `json:"serverDefaults"`
	ServerGroups map[string][]string `json:"serverGroups"`
}

// NewServerManager creates a new MCP server manager
func NewServerManager(config *config.MCPConfig) (*ServerManager, error) {
	sm := &ServerManager{
		config:        config,
		servers:       make(map[string]*ServerInstance),
		serverConfigs: make(map[string]*ServerConfig),
	}

	// Load server configurations
	if err := sm.loadServerConfigs(); err != nil {
		return nil, fmt.Errorf("failed to load server configs: %w", err)
	}

	return sm, nil
}

// loadServerConfigs loads server configurations from mcpServers.json
func (sm *ServerManager) loadServerConfigs() error {
	configPath := sm.config.ServersConfigPath
	if configPath == "" {
		configPath = "mcpServers.json"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Use default configuration
			return sm.loadDefaultConfigs()
		}
		return fmt.Errorf("failed to read servers config: %w", err)
	}

	var config MCPServersConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse servers config: %w", err)
	}

	sm.serverConfigs = config.MCPServers
	return nil
}

// loadDefaultConfigs loads default server configurations
func (sm *ServerManager) loadDefaultConfigs() error {
	sm.serverConfigs = map[string]*ServerConfig{
		"filesystem": {
			Command:     "npx",
			Args:        []string{"@modelcontextprotocol/server-filesystem", "--allowed-directories", "./", "/tmp"},
			Description: "File system operations with sandboxed directory access",
			AutoStart:   true,
		},
		"fetch": {
			Command:     "npx",
			Args:        []string{"@modelcontextprotocol/server-fetch"},
			Description: "HTTP/HTTPS fetch operations for web content retrieval",
			AutoStart:   true,
		},
		"memory": {
			Command:     "npx",
			Args:        []string{"@modelcontextprotocol/server-memory"},
			Description: "In-memory key-value store for temporary data storage",
			AutoStart:   true,
		},
		"sequential-thinking": {
			Command:     "npx",
			Args:        []string{"@modelcontextprotocol/server-sequential-thinking"},
			Description: "Enhanced reasoning through step-by-step problem decomposition",
			AutoStart:   false,
		},
		"time": {
			Command:     "npx",
			Args:        []string{"@modelcontextprotocol/server-time"},
			Description: "Time and date utilities with timezone support",
			AutoStart:   true,
		},
	}
	return nil
}

// StartServer starts a specific MCP server
func (sm *ServerManager) StartServer(ctx context.Context, name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	config, exists := sm.serverConfigs[name]
	if !exists {
		return fmt.Errorf("server '%s' not found in configuration", name)
	}

	// Check if already running
	if instance, exists := sm.servers[name]; exists && instance.Status == ServerStatusRunning {
		return fmt.Errorf("server '%s' is already running", name)
	}

	// Create server instance
	instance := &ServerInstance{
		Name:      name,
		Config:    config,
		Status:    ServerStatusStarting,
		StartedAt: time.Now(),
	}

	// Start the server process
	cmd := exec.CommandContext(ctx, config.Command, config.Args...)
	
	// Set environment variables
	if config.Env != nil {
		env := os.Environ()
		for k, v := range config.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}

	// Set up stdio pipes for MCP communication
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		instance.Status = ServerStatusError
		return fmt.Errorf("failed to start server '%s': %w", name, err)
	}

	instance.Process = cmd

	// Create MCP client for this server
	client, err := NewClientWithPipes(stdin, stdout, stderr)
	if err != nil {
		cmd.Process.Kill()
		instance.Status = ServerStatusError
		return fmt.Errorf("failed to create client for server '%s': %w", name, err)
	}

	instance.Client = client
	instance.Status = ServerStatusRunning
	instance.LastPing = time.Now()

	sm.servers[name] = instance

	// Start health monitoring
	go sm.monitorServer(ctx, name)

	return nil
}

// StopServer stops a specific MCP server
func (sm *ServerManager) StopServer(name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	instance, exists := sm.servers[name]
	if !exists {
		return fmt.Errorf("server '%s' is not running", name)
	}

	// Close the client
	if instance.Client != nil {
		instance.Client.Close()
	}

	// Terminate the process
	if instance.Process != nil && instance.Process.Process != nil {
		if err := instance.Process.Process.Kill(); err != nil {
			return fmt.Errorf("failed to stop server '%s': %w", name, err)
		}
	}

	instance.Status = ServerStatusStopped
	delete(sm.servers, name)

	return nil
}

// StartAllServers starts all configured servers with AutoStart=true
func (sm *ServerManager) StartAllServers(ctx context.Context) error {
	var errors []error

	for name, config := range sm.serverConfigs {
		if config.AutoStart {
			if err := sm.StartServer(ctx, name); err != nil {
				if config.Required {
					return fmt.Errorf("failed to start required server '%s': %w", name, err)
				}
				errors = append(errors, fmt.Errorf("server '%s': %w", name, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("some servers failed to start: %v", errors)
	}

	return nil
}

// StopAllServers stops all running servers
func (sm *ServerManager) StopAllServers() error {
	sm.mu.RLock()
	serverNames := make([]string, 0, len(sm.servers))
	for name := range sm.servers {
		serverNames = append(serverNames, name)
	}
	sm.mu.RUnlock()

	var errors []error
	for _, name := range serverNames {
		if err := sm.StopServer(name); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to stop some servers: %v", errors)
	}

	return nil
}

// GetServerStatus returns the status of a specific server
func (sm *ServerManager) GetServerStatus(name string) (ServerStatus, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	instance, exists := sm.servers[name]
	if !exists {
		if _, configured := sm.serverConfigs[name]; configured {
			return ServerStatusStopped, nil
		}
		return ServerStatusUnknown, fmt.Errorf("server '%s' not configured", name)
	}

	return instance.Status, nil
}

// GetAllServerStatus returns status of all configured servers
func (sm *ServerManager) GetAllServerStatus() map[string]ServerStatus {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	status := make(map[string]ServerStatus)
	
	// Include all configured servers
	for name := range sm.serverConfigs {
		if instance, exists := sm.servers[name]; exists {
			status[name] = instance.Status
		} else {
			status[name] = ServerStatusStopped
		}
	}

	return status
}

// GetServerClient returns the MCP client for a running server
func (sm *ServerManager) GetServerClient(name string) (*Client, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	instance, exists := sm.servers[name]
	if !exists {
		return nil, fmt.Errorf("server '%s' is not running", name)
	}

	if instance.Status != ServerStatusRunning {
		return nil, fmt.Errorf("server '%s' is not in running state: %s", name, instance.Status)
	}

	return instance.Client, nil
}

// ListAvailableTools returns all tools from all running servers
func (sm *ServerManager) ListAvailableTools() ([]Tool, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var allTools []Tool

	for name, instance := range sm.servers {
		if instance.Status != ServerStatusRunning || instance.Client == nil {
			continue
		}

		tools, err := instance.Client.ListTools()
		if err != nil {
			// Log error but continue with other servers
			fmt.Fprintf(os.Stderr, "Warning: failed to list tools from server '%s': %v\n", name, err)
			continue
		}

		// Add server name to tool metadata
		for i := range tools {
			if tools[i].Metadata == nil {
				tools[i].Metadata = make(map[string]interface{})
			}
			tools[i].Metadata["server"] = name
		}

		allTools = append(allTools, tools...)
	}

	return allTools, nil
}

// CallTool calls a tool on the appropriate server
func (sm *ServerManager) CallTool(ctx context.Context, toolName string, arguments interface{}) (*ToolResult, error) {
	// Find which server provides this tool
	serverName, err := sm.findToolServer(toolName)
	if err != nil {
		return nil, err
	}

	client, err := sm.GetServerClient(serverName)
	if err != nil {
		return nil, fmt.Errorf("failed to get client for server '%s': %w", serverName, err)
	}

	return client.CallTool(ctx, toolName, arguments)
}

// findToolServer finds which server provides a specific tool
func (sm *ServerManager) findToolServer(toolName string) (string, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for name, instance := range sm.servers {
		if instance.Status != ServerStatusRunning || instance.Client == nil {
			continue
		}

		tools, err := instance.Client.ListTools()
		if err != nil {
			continue
		}

		for _, tool := range tools {
			if tool.Name == toolName {
				return name, nil
			}
		}
	}

	return "", fmt.Errorf("tool '%s' not found in any running server", toolName)
}

// monitorServer monitors the health of a running server
func (sm *ServerManager) monitorServer(ctx context.Context, name string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.mu.Lock()
			instance, exists := sm.servers[name]
			if !exists {
				sm.mu.Unlock()
				return
			}

			// Check if process is still running
			if instance.Process != nil && instance.Process.ProcessState != nil && instance.Process.ProcessState.Exited() {
				instance.Status = ServerStatusError
				instance.ErrorCount++
				sm.mu.Unlock()
				
				// Attempt to restart if configured
				if instance.Config.AutoStart && instance.ErrorCount < 3 {
					time.Sleep(5 * time.Second)
					sm.StartServer(ctx, name)
				}
				return
			}

			// Update last ping time
			instance.LastPing = time.Now()
			sm.mu.Unlock()
		}
	}
}

// GetServerInfo returns detailed information about a server
func (sm *ServerManager) GetServerInfo(name string) map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	info := make(map[string]interface{})
	
	config, hasConfig := sm.serverConfigs[name]
	if hasConfig {
		info["configured"] = true
		info["description"] = config.Description
		info["command"] = config.Command
		info["args"] = config.Args
		info["autoStart"] = config.AutoStart
		info["required"] = config.Required
	} else {
		info["configured"] = false
	}

	instance, isRunning := sm.servers[name]
	if isRunning {
		info["status"] = instance.Status
		info["startedAt"] = instance.StartedAt
		info["lastPing"] = instance.LastPing
		info["errorCount"] = instance.ErrorCount
		
		if instance.Client != nil {
			if tools, err := instance.Client.ListTools(); err == nil {
				info["toolCount"] = len(tools)
			}
		}
	} else {
		info["status"] = ServerStatusStopped
	}

	return info
}