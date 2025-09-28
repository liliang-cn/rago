package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// SwaggerManager manages multiple Swagger-based MCP servers
type SwaggerManager struct {
	servers map[string]*SwaggerServer
	mu      sync.RWMutex
}

// NewSwaggerManager creates a new Swagger manager
func NewSwaggerManager() *SwaggerManager {
	return &SwaggerManager{
		servers: make(map[string]*SwaggerServer),
	}
}

// AddServer adds a new Swagger server configuration
func (m *SwaggerManager) AddServer(name string, config *SwaggerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.servers[name]; exists {
		return fmt.Errorf("server %s already exists", name)
	}
	
	server, err := NewSwaggerServer(config)
	if err != nil {
		return fmt.Errorf("failed to create server %s: %w", name, err)
	}
	
	m.servers[name] = server
	return nil
}

// RemoveServer removes a Swagger server
func (m *SwaggerManager) RemoveServer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	server, exists := m.servers[name]
	if !exists {
		return fmt.Errorf("server %s not found", name)
	}
	
	if err := server.Stop(); err != nil {
		return fmt.Errorf("failed to stop server %s: %w", name, err)
	}
	
	delete(m.servers, name)
	return nil
}

// GetServer gets a specific Swagger server
func (m *SwaggerManager) GetServer(name string) (*SwaggerServer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	server, exists := m.servers[name]
	if !exists {
		return nil, fmt.Errorf("server %s not found", name)
	}
	
	return server, nil
}

// ListServers returns all server names
func (m *SwaggerManager) ListServers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	names := make([]string, 0, len(m.servers))
	for name := range m.servers {
		names = append(names, name)
	}
	
	return names
}

// StartAll starts all configured Swagger servers
func (m *SwaggerManager) StartAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for name, server := range m.servers {
		if err := server.Start(ctx); err != nil {
			return fmt.Errorf("failed to start server %s: %w", name, err)
		}
	}
	
	return nil
}

// StopAll stops all running Swagger servers
func (m *SwaggerManager) StopAll() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var errs []error
	for name, server := range m.servers {
		if err := server.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop server %s: %w", name, err))
		}
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("errors stopping servers: %v", errs)
	}
	
	return nil
}

// GetAllTools returns all tools from all Swagger servers
func (m *SwaggerManager) GetAllTools() (map[string][]*Tool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	allTools := make(map[string][]*Tool)
	
	for name, server := range m.servers {
		tools, err := server.GetTools()
		if err != nil {
			return nil, fmt.Errorf("failed to get tools from server %s: %w", name, err)
		}
		
		// Convert to internal Tool type
		internalTools := make([]*Tool, 0, len(tools))
		for _, tool := range tools {
			internalTool := &Tool{
				Name:        fmt.Sprintf("%s.%s", name, tool.Name), // Prefix with server name
				Description: tool.Description,
				InputSchema: tool.InputSchema,
				Server:      name, // Track which server owns this tool
			}
			internalTools = append(internalTools, internalTool)
		}
		
		allTools[name] = internalTools
	}
	
	return allTools, nil
}

// CallTool calls a tool on a specific server
func (m *SwaggerManager) CallTool(ctx context.Context, serverName, toolName string, args json.RawMessage) (json.RawMessage, error) {
	server, err := m.GetServer(serverName)
	if err != nil {
		return nil, err
	}
	
	return server.CallTool(ctx, toolName, args)
}

// SwaggerIntegration integrates Swagger servers with the main MCP service
type SwaggerIntegration struct {
	manager *SwaggerManager
	service *Service
}

// NewSwaggerIntegration creates a new Swagger integration
func NewSwaggerIntegration(service *Service) *SwaggerIntegration {
	return &SwaggerIntegration{
		manager: NewSwaggerManager(),
		service: service,
	}
}

// LoadSwaggerConfigs loads Swagger configurations and adds them to the manager
func (si *SwaggerIntegration) LoadSwaggerConfigs(configs map[string]*SwaggerConfig) error {
	for name, config := range configs {
		config.Name = name
		if err := si.manager.AddServer(name, config); err != nil {
			return fmt.Errorf("failed to add swagger server %s: %w", name, err)
		}
	}
	
	return nil
}

// RegisterWithService registers Swagger tools with the main MCP service
func (si *SwaggerIntegration) RegisterWithService() error {
	// Get all tools from Swagger servers
	allTools, err := si.manager.GetAllTools()
	if err != nil {
		return fmt.Errorf("failed to get swagger tools: %w", err)
	}
	
	// Register each tool with the MCP service
	for serverName, tools := range allTools {
		for _, tool := range tools {
			// Register tool handler
			si.service.RegisterToolHandler(tool.Name, func(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
				// Extract the actual tool name (remove server prefix)
				actualToolName := tool.Name
				if len(serverName) > 0 {
					actualToolName = actualToolName[len(serverName)+1:] // Remove "servername." prefix
				}
				return si.manager.CallTool(ctx, serverName, actualToolName, args)
			})
		}
	}
	
	return nil
}

// Start starts all Swagger servers
func (si *SwaggerIntegration) Start(ctx context.Context) error {
	return si.manager.StartAll(ctx)
}

// Stop stops all Swagger servers
func (si *SwaggerIntegration) Stop() error {
	return si.manager.StopAll()
}

// DefaultSwaggerConfigs returns some default Swagger configurations for testing
func DefaultSwaggerConfigs() map[string]*SwaggerConfig {
	return map[string]*SwaggerConfig{
		"petstore": {
			Name:       "petstore",
			SwaggerURL: "https://petstore.swagger.io/v2/swagger.json",
			Transport:  "stdio",
		},
		// Add more default configurations as needed
	}
}