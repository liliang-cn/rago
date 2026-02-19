package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AddServerOptions contains options for adding a new MCP server
type AddServerOptions struct {
	// ConfigFilePath is the path to mcpServers.json (optional, auto-detected if empty)
	ConfigFilePath string
}

// AddServerResult contains the result of adding a server
type AddServerResult struct {
	ServerName string
	Config     ServerConfig
	ConfigPath string
	Overwritten bool // True if an existing server was overwritten
}

// AddStdioServer adds a stdio-type MCP server
func AddStdioServer(name, command string, args []string, opts *AddServerOptions) (*AddServerResult, error) {
	if name == "" {
		return nil, fmt.Errorf("server name is required")
	}
	if command == "" {
		return nil, fmt.Errorf("command is required for stdio server")
	}

	config := ServerConfig{
		Name:        name,
		Description: fmt.Sprintf("MCP server: %s", name),
		Type:        ServerTypeStdio,
		Command:     []string{command},
		Args:        args,
		AutoStart:   true,
	}

	return addServerToConfig(name, config, opts)
}

// AddHTTPServer adds an HTTP-type MCP server
func AddHTTPServer(name, url string, headers map[string]string, opts *AddServerOptions) (*AddServerResult, error) {
	if name == "" {
		return nil, fmt.Errorf("server name is required")
	}
	if url == "" {
		return nil, fmt.Errorf("url is required for HTTP server")
	}

	config := ServerConfig{
		Name:        name,
		Description: fmt.Sprintf("MCP server: %s", name),
		Type:        ServerTypeHTTP,
		URL:         url,
		Headers:     headers,
		AutoStart:   true,
	}

	return addServerToConfig(name, config, opts)
}

// AddServerFromJSON adds an MCP server from JSON configuration
func AddServerFromJSON(name, jsonConfig string, opts *AddServerOptions) (*AddServerResult, error) {
	if name == "" {
		return nil, fmt.Errorf("server name is required")
	}
	if jsonConfig == "" {
		return nil, fmt.Errorf("json config is required")
	}

	// Parse JSON config
	var simpleCfg SimpleServerConfig
	if err := json.Unmarshal([]byte(jsonConfig), &simpleCfg); err != nil {
		return nil, fmt.Errorf("failed to parse JSON config: %w", err)
	}

	// Determine server type
	serverType := ServerTypeStdio
	if simpleCfg.Type != "" {
		serverType = ServerType(simpleCfg.Type)
	} else if simpleCfg.URL != "" {
		serverType = ServerTypeHTTP
	}

	// Validate
	if serverType == ServerTypeHTTP && simpleCfg.URL == "" {
		return nil, fmt.Errorf("HTTP server requires 'url' field")
	}
	if serverType == ServerTypeStdio && simpleCfg.Command == "" {
		return nil, fmt.Errorf("stdio server requires 'command' field")
	}

	config := ServerConfig{
		Name:             name,
		Description:      fmt.Sprintf("MCP server: %s", name),
		Type:             serverType,
		Args:             simpleCfg.Args,
		URL:              simpleCfg.URL,
		Headers:          simpleCfg.Headers,
		WorkingDir:       simpleCfg.WorkingDir,
		Env:              simpleCfg.Env,
		AutoStart:        true,
		RestartOnFailure: true,
		MaxRestarts:      3,
		RestartDelay:     5e9, // 5 seconds in nanoseconds
	}

	if serverType == ServerTypeStdio && simpleCfg.Command != "" {
		config.Command = []string{simpleCfg.Command}
	}

	return addServerToConfig(name, config, opts)
}

// RemoveServer removes an MCP server from configuration
func RemoveServer(name string, opts *AddServerOptions) error {
	if name == "" {
		return fmt.Errorf("server name is required")
	}

	configPath := resolveConfigPath(opts)

	// Load existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config file not found: %s", configPath)
		}
		return fmt.Errorf("failed to read config: %w", err)
	}

	var jsonConfig JSONServersConfig
	if err := json.Unmarshal(data, &jsonConfig); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Check if server exists
	if _, exists := jsonConfig.MCPServers[name]; !exists {
		return fmt.Errorf("server %s not found in configuration", name)
	}

	// Remove server
	delete(jsonConfig.MCPServers, name)

	// Write back
	return saveJSONConfig(configPath, &jsonConfig)
}

// ListServers lists all configured MCP servers
func ListServers(opts *AddServerOptions) (map[string]SimpleServerConfig, error) {
	configPath := resolveConfigPath(opts)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]SimpleServerConfig), nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var jsonConfig JSONServersConfig
	if err := json.Unmarshal(data, &jsonConfig); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return jsonConfig.MCPServers, nil
}

// GetServer retrieves a specific server configuration
func GetServer(name string, opts *AddServerOptions) (*SimpleServerConfig, error) {
	servers, err := ListServers(opts)
	if err != nil {
		return nil, err
	}

	cfg, exists := servers[name]
	if !exists {
		return nil, fmt.Errorf("server %s not found", name)
	}

	return &cfg, nil
}

// Helper functions

func addServerToConfig(name string, config ServerConfig, opts *AddServerOptions) (*AddServerResult, error) {
	configPath := resolveConfigPath(opts)

	// Load existing config
	var jsonConfig JSONServersConfig
	data, err := os.ReadFile(configPath)
	if err == nil {
		if err := json.Unmarshal(data, &jsonConfig); err != nil {
			return nil, fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Check if overwriting
	overwritten := false
	if _, exists := jsonConfig.MCPServers[name]; exists {
		overwritten = true
	}

	// Convert to SimpleServerConfig for storage
	simpleCfg := ServerConfigToSimple(&config)

	// Add to config
	if jsonConfig.MCPServers == nil {
		jsonConfig.MCPServers = make(map[string]SimpleServerConfig)
	}
	jsonConfig.MCPServers[name] = simpleCfg

	// Save
	if err := saveJSONConfig(configPath, &jsonConfig); err != nil {
		return nil, err
	}

	return &AddServerResult{
		ServerName:  name,
		Config:      config,
		ConfigPath:  configPath,
		Overwritten: overwritten,
	}, nil
}

func resolveConfigPath(opts *AddServerOptions) string {
	if opts != nil && opts.ConfigFilePath != "" {
		return opts.ConfigFilePath
	}

	// Auto-detect: local first, then home directory
	localPath := "./mcpServers.json"
	if _, err := os.Stat(localPath); err == nil {
		return localPath
	}

	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".rago", "mcpServers.json")
}

func saveJSONConfig(path string, config *JSONServersConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// ServerConfigToSimple converts ServerConfig to SimpleServerConfig for JSON storage
func ServerConfigToSimple(cfg *ServerConfig) SimpleServerConfig {
	simple := SimpleServerConfig{
		Type:       string(cfg.Type),
		Args:       cfg.Args,
		URL:        cfg.URL,
		Headers:    cfg.Headers,
		WorkingDir: cfg.WorkingDir,
		Env:        cfg.Env,
	}

	// Extract command (first element if present)
	if len(cfg.Command) > 0 {
		simple.Command = cfg.Command[0]
	}

	return simple
}

// SimpleToServerConfig converts SimpleServerConfig to ServerConfig
func SimpleToServerConfig(name string, simple SimpleServerConfig) ServerConfig {
	serverType := ServerTypeStdio
	if simple.Type != "" {
		serverType = ServerType(simple.Type)
	} else if simple.URL != "" {
		serverType = ServerTypeHTTP
	}

	return ServerConfig{
		Name:        name,
		Description: fmt.Sprintf("MCP server: %s", name),
		Type:        serverType,
		Command:     []string{simple.Command},
		Args:        simple.Args,
		URL:         simple.URL,
		Headers:     simple.Headers,
		WorkingDir:  simple.WorkingDir,
		Env:         simple.Env,
		AutoStart:   true,
	}
}
