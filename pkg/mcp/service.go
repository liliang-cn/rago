// Package mcp implements the MCP (Model Context Protocol) pillar.
// This pillar focuses on tool integration and external service coordination.
package mcp

import (
	"context"
	"fmt"
	"time"
	
	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/mcp/server"
	"github.com/liliang-cn/rago/v2/pkg/mcp/tools"
)

// Service implements the MCP pillar service interface.
// This is the main entry point for all MCP operations including server
// management and tool operations.
type Service struct {
	config           core.MCPConfig
	serverManager    *server.Manager
	toolRegistry     *tools.Registry
	toolExecutor     *tools.Executor
	toolCache        *tools.Cache
	discoveryService *server.DiscoveryService
	started          bool
}

// NewService creates a new MCP service instance.
func NewService(config core.MCPConfig) (*Service, error) {
	// Create server manager
	serverManager, err := server.NewManager(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create server manager: %w", err)
	}
	
	// Set client factory
	serverManager.SetClientFactory(CreateClientFactory())
	
	// Create tool registry
	toolRegistry := tools.NewRegistry()
	
	// Create tool cache
	cacheSize := 1000
	if config.CacheSize > 0 {
		cacheSize = config.CacheSize
	}
	cacheTTL := 5 * time.Minute
	if config.CacheTTL > 0 {
		cacheTTL = config.CacheTTL
	}
	toolCache := tools.NewCache(cacheSize, cacheTTL, tools.LRU)
	
	// Create server adapter for tool executor
	serverAdapter := &serverManagerAdapter{manager: serverManager}
	
	// Create tool executor
	toolExecutor := tools.NewExecutor(toolRegistry, toolCache, serverAdapter)
	
	service := &Service{
		config:           config,
		serverManager:    serverManager,
		toolRegistry:     toolRegistry,
		toolExecutor:     toolExecutor,
		toolCache:        toolCache,
		discoveryService: serverManager.GetDiscoveryService(),
	}
	
	// Set up server registration hooks
	serverManager.AddRegistrationHook(service.onServerRegistration)
	
	// Set up tool registry hooks
	toolRegistry.AddHook(service.onToolRegistration)
	
	return service, nil
}

// Start starts the MCP service and all its components.
func (s *Service) Start(ctx context.Context) error {
	if s.started {
		return nil
	}
	
	// Start server manager
	if err := s.serverManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start server manager: %w", err)
	}
	
	// Start tool executor
	if err := s.toolExecutor.Start(ctx); err != nil {
		return fmt.Errorf("failed to start tool executor: %w", err)
	}
	
	// Load server configurations if provided
	if err := s.loadServerConfigurations(ctx); err != nil {
		return fmt.Errorf("failed to load server configurations: %w", err)
	}
	
	s.started = true
	return nil
}

// ===== SERVER MANAGEMENT =====

// RegisterServer registers a new MCP server.
func (s *Service) RegisterServer(config core.ServerConfig) error {
	ctx := context.Background()
	return s.serverManager.RegisterServer(ctx, config)
}

// UnregisterServer unregisters an MCP server.
func (s *Service) UnregisterServer(name string) error {
	ctx := context.Background()
	return s.serverManager.UnregisterServer(ctx, name)
}

// ListServers lists all registered MCP servers.
func (s *Service) ListServers() []core.ServerInfo {
	servers := s.serverManager.ListServers()
	result := make([]core.ServerInfo, 0, len(servers))
	
	for _, srv := range servers {
		info := core.ServerInfo{
			Name:         srv.Config.Name,
			Description:  srv.Config.Description,
			Status:       s.convertServerStatus(srv.Status),
			Capabilities: srv.Config.Capabilities,
		}
		
		if srv.Info != nil {
			info.Version = srv.Info.Version
			info.ToolCount = srv.Info.ToolCount
			info.Metadata = srv.Info.Metadata
		}
		
		result = append(result, info)
	}
	
	return result
}

// GetServerHealth gets the health status of a specific server.
func (s *Service) GetServerHealth(name string) core.HealthStatus {
	if s.serverManager == nil || s.serverManager.GetHealthMonitor() == nil {
		return core.HealthStatusUnknown
	}
	return s.serverManager.GetHealthMonitor().GetHealthStatus(name)
}

// ===== TOOL OPERATIONS =====

// ListTools lists all available tools from all servers.
func (s *Service) ListTools() []core.ToolInfo {
	registeredTools := s.toolRegistry.ListAvailable()
	result := make([]core.ToolInfo, 0, len(registeredTools))
	
	for _, tool := range registeredTools {
		info := core.ToolInfo{
			ID:          tool.ID,
			Name:        tool.Name,
			ServerName:  tool.ServerName,
			Description: tool.Description,
			Category:    tool.Category,
			InputSchema: tool.InputSchema,
			Metadata:    tool.Metadata,
		}
		result = append(result, info)
	}
	
	return result
}

// GetTool gets information about a specific tool.
func (s *Service) GetTool(name string) (*core.ToolInfo, error) {
	tool, err := s.toolRegistry.Get(name)
	if err != nil {
		return nil, err
	}
	
	return &core.ToolInfo{
		ID:          tool.ID,
		Name:        tool.Name,
		ServerName:  tool.ServerName,
		Description: tool.Description,
		Category:    tool.Category,
		InputSchema: tool.InputSchema,
		Metadata:    tool.Metadata,
	}, nil
}

// CallTool calls a tool synchronously.
func (s *Service) CallTool(ctx context.Context, req core.ToolCallRequest) (*core.ToolCallResponse, error) {
	if !s.started {
		return nil, fmt.Errorf("MCP service not started")
	}
	
	return s.toolExecutor.Execute(ctx, &req)
}

// CallToolAsync calls a tool asynchronously.
func (s *Service) CallToolAsync(ctx context.Context, req core.ToolCallRequest) (<-chan *core.ToolCallResponse, error) {
	if !s.started {
		return nil, fmt.Errorf("MCP service not started")
	}
	
	return s.toolExecutor.ExecuteAsync(ctx, &req)
}

// ===== BATCH OPERATIONS =====

// CallToolsBatch calls multiple tools in a batch operation.
func (s *Service) CallToolsBatch(ctx context.Context, requests []core.ToolCallRequest) ([]core.ToolCallResponse, error) {
	if !s.started {
		return nil, fmt.Errorf("MCP service not started")
	}
	
	return s.toolExecutor.ExecuteBatch(ctx, requests)
}

// ===== ADVANCED OPERATIONS =====

// SearchTools searches for tools matching criteria.
func (s *Service) SearchTools(query string) []core.ToolInfo {
	criteria := tools.SearchCriteria{
		Name: query,
	}
	
	registeredTools := s.toolRegistry.Search(criteria)
	result := make([]core.ToolInfo, 0, len(registeredTools))
	
	for _, tool := range registeredTools {
		info := core.ToolInfo{
			ID:          tool.ID,
			Name:        tool.Name,
			ServerName:  tool.ServerName,
			Description: tool.Description,
			Category:    tool.Category,
			InputSchema: tool.InputSchema,
			Metadata:    tool.Metadata,
		}
		result = append(result, info)
	}
	
	return result
}

// GetToolsByCategory returns tools in a specific category.
func (s *Service) GetToolsByCategory(category string) []core.ToolInfo {
	registeredTools := s.toolRegistry.GetByCategory(category)
	result := make([]core.ToolInfo, 0, len(registeredTools))
	
	for _, tool := range registeredTools {
		info := core.ToolInfo{
			ID:          tool.ID,
			Name:        tool.Name,
			ServerName:  tool.ServerName,
			Description: tool.Description,
			Category:    tool.Category,
			InputSchema: tool.InputSchema,
			Metadata:    tool.Metadata,
		}
		result = append(result, info)
	}
	
	return result
}

// GetMetrics returns MCP service metrics.
func (s *Service) GetMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})
	
	// Server metrics
	if s.serverManager != nil {
		metrics["servers"] = s.serverManager.GetMetrics()
	}
	
	// Tool registry metrics
	if s.toolRegistry != nil {
		metrics["registry"] = s.toolRegistry.GetStats()
	}
	
	// Executor metrics
	if s.toolExecutor != nil {
		metrics["executor"] = s.toolExecutor.GetMetrics()
	}
	
	// Cache metrics
	if s.toolCache != nil {
		metrics["cache"] = s.toolCache.GetStats()
	}
	
	return metrics
}

// InvalidateCache invalidates cache entries matching a pattern.
func (s *Service) InvalidateCache(pattern string) int {
	if s.toolCache == nil {
		return 0
	}
	return s.toolCache.Invalidate(pattern)
}

// ClearCache clears all cache entries.
func (s *Service) ClearCache() {
	if s.toolCache != nil {
		s.toolCache.Clear()
	}
}

// Close closes the MCP service and cleans up resources.
func (s *Service) Close() error {
	if !s.started {
		return nil
	}
	
	ctx := context.Background()
	
	// Stop tool executor
	if s.toolExecutor != nil {
		s.toolExecutor.Stop()
	}
	
	// Stop cache
	if s.toolCache != nil {
		s.toolCache.Stop()
	}
	
	// Shutdown server manager
	if s.serverManager != nil {
		if err := s.serverManager.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown server manager: %w", err)
		}
	}
	
	s.started = false
	return nil
}

// ===== HELPER METHODS =====

// loadServerConfigurations loads server configurations from config.
func (s *Service) loadServerConfigurations(ctx context.Context) error {
	for _, serverConfig := range s.config.Servers {
		if serverConfig.AutoStart {
			if err := s.serverManager.RegisterServer(ctx, serverConfig); err != nil {
				// Log error but continue with other servers
				fmt.Printf("Failed to register server %s: %v\n", serverConfig.Name, err)
			}
		}
	}
	return nil
}

// onServerRegistration handles server registration events.
func (s *Service) onServerRegistration(srv *server.ServerInstance, registered bool) {
	if registered && srv.Client != nil {
		// Discover and register tools from the server
		s.discoverServerTools(srv)
	} else if !registered {
		// Mark tools from this server as unavailable
		s.toolRegistry.UpdateAvailability(srv.Config.Name, false)
	}
}

// discoverServerTools discovers and registers tools from a server.
func (s *Service) discoverServerTools(srv *server.ServerInstance) {
	if srv.Client == nil || !srv.Client.IsConnected() {
		return
	}
	
	serverTools := srv.Client.GetTools()
	for toolName, tool := range serverTools {
		toolID := fmt.Sprintf("%s:%s", srv.Config.Name, toolName)
		
		registeredTool := &tools.RegisteredTool{
			ID:          toolID,
			Name:        tool.Name,
			ServerName:  srv.Config.Name,
			Description: tool.Description,
			Category:    "general", // Will be refined by discovery service
			InputSchema: s.convertInputSchema(tool.InputSchema),
			Timeout:     30 * time.Second,
			MaxRetries:  3,
			Cacheable:   s.isToolCacheable(tool),
			Available:   true,
		}
		
		if err := s.toolRegistry.Register(registeredTool); err != nil {
			fmt.Printf("Failed to register tool %s: %v\n", toolID, err)
		}
	}
}

// onToolRegistration handles tool registration events.
func (s *Service) onToolRegistration(tool *tools.RegisteredTool, registered bool) {
	// Could emit events or update metrics here
	if registered {
		fmt.Printf("Tool registered: %s\n", tool.ID)
	} else {
		fmt.Printf("Tool unregistered: %s\n", tool.ID)
	}
}

// convertServerStatus converts server status to core.ServerStatus.
func (s *Service) convertServerStatus(status server.ServerStatus) string {
	return status.String()
}

// convertInputSchema converts MCP input schema to generic format.
func (s *Service) convertInputSchema(schema interface{}) map[string]interface{} {
	// Simple conversion - could be enhanced
	if m, ok := schema.(map[string]interface{}); ok {
		return m
	}
	return nil
}

// isToolCacheable determines if a tool should be cached.
func (s *Service) isToolCacheable(tool interface{}) bool {
	// Implement logic to determine if tool results should be cached
	// For now, cache read-only operations
	return true // Simplified - should check tool characteristics
}

// serverManagerAdapter adapts server.Manager to tools.ServerManager interface.
type serverManagerAdapter struct {
	manager *server.Manager
}

func (a *serverManagerAdapter) GetServer(name string) (tools.MCPClient, error) {
	instance, err := a.manager.GetServer(name)
	if err != nil {
		return nil, err
	}
	
	if instance.Client == nil {
		return nil, fmt.Errorf("server %s has no client", name)
	}
	
	// Create an adapter that implements tools.MCPClient
	return &toolsClientAdapter{client: instance.Client}, nil
}

// toolsClientAdapter adapts server.MCPClient to tools.MCPClient
type toolsClientAdapter struct {
	client server.MCPClient
}

func (t *toolsClientAdapter) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*tools.ToolResult, error) {
	result, err := t.client.CallTool(ctx, toolName, arguments)
	if err != nil {
		return nil, err
	}
	
	return &tools.ToolResult{
		Success: result.Success,
		Data:    result.Data,
		Error:   result.Error,
	}, nil
}