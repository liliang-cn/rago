// Package server provides server lifecycle management for the MCP pillar.
package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
)

// Manager handles the lifecycle of MCP servers including registration,
// health monitoring, and automatic recovery.
type Manager struct {
	mu                sync.RWMutex
	servers           map[string]*ServerInstance
	config            core.MCPConfig
	healthMonitor     *HealthMonitor
	discoveryService  *DiscoveryService
	shutdownCh        chan struct{}
	metricsCollector  *MetricsCollector
	registrationHooks []RegistrationHook
	clientFactory     ClientFactory
}

// ClientFactory creates MCP clients for server configurations
type ClientFactory func(config core.ServerConfig) (MCPClient, error)

// ServerInstance represents a managed MCP server instance
type ServerInstance struct {
	Config      core.ServerConfig
	Client      MCPClient
	Info        *ServerInfo
	Status      ServerStatus
	LastHealthy time.Time
	RestartCount int
	StartedAt   time.Time
	mu          sync.RWMutex
}

// ServerInfo contains runtime information about a server
type ServerInfo struct {
	Name         string
	Version      string
	Description  string
	Capabilities []string
	ToolCount    int
	Metadata     map[string]interface{}
}

// ServerStatus represents the current status of a server
type ServerStatus int

const (
	StatusUnknown ServerStatus = iota
	StatusStarting
	StatusHealthy
	StatusUnhealthy
	StatusRestarting
	StatusStopped
	StatusFailed
)

// RegistrationHook is called when a server is registered or unregistered
type RegistrationHook func(server *ServerInstance, registered bool)

// NewManager creates a new server manager
func NewManager(config core.MCPConfig) (*Manager, error) {
	m := &Manager{
		servers:    make(map[string]*ServerInstance),
		config:     config,
		shutdownCh: make(chan struct{}),
	}
	
	// Initialize health monitor
	m.healthMonitor = NewHealthMonitor(m)
	
	// Initialize discovery service
	m.discoveryService = NewDiscoveryService(m)
	
	// Initialize metrics collector
	m.metricsCollector = NewMetricsCollector()
	
	return m, nil
}

// SetClientFactory sets the client factory for creating MCP clients
func (m *Manager) SetClientFactory(factory ClientFactory) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clientFactory = factory
}

// Start starts the server manager and its background services
func (m *Manager) Start(ctx context.Context) error {
	// Start health monitoring
	if err := m.healthMonitor.Start(ctx); err != nil {
		return fmt.Errorf("failed to start health monitor: %w", err)
	}
	
	// Start discovery service
	if err := m.discoveryService.Start(ctx); err != nil {
		return fmt.Errorf("failed to start discovery service: %w", err)
	}
	
	// Auto-start configured servers
	if err := m.autoStartServers(ctx); err != nil {
		return fmt.Errorf("failed to auto-start servers: %w", err)
	}
	
	// Start metrics collection
	m.metricsCollector.Start(ctx)
	
	return nil
}

// RegisterServer registers and starts a new MCP server
func (m *Manager) RegisterServer(ctx context.Context, config core.ServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if server already exists
	if _, exists := m.servers[config.Name]; exists {
		return fmt.Errorf("server %s already registered", config.Name)
	}
	
	// Create server instance
	instance := &ServerInstance{
		Config:    config,
		Status:    StatusStarting,
		StartedAt: time.Now(),
	}
	
	// Start the server
	if err := m.startServerInternal(ctx, instance); err != nil {
		instance.Status = StatusFailed
		return fmt.Errorf("failed to start server %s: %w", config.Name, err)
	}
	
	// Register the instance
	m.servers[config.Name] = instance
	
	// Call registration hooks
	for _, hook := range m.registrationHooks {
		hook(instance, true)
	}
	
	// Record metrics
	m.metricsCollector.RecordServerRegistration(config.Name)
	
	return nil
}

// UnregisterServer stops and unregisters an MCP server
func (m *Manager) UnregisterServer(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	instance, exists := m.servers[name]
	if !exists {
		return core.ErrServerNotFound
	}
	
	// Stop the server
	if err := m.stopServerInternal(ctx, instance); err != nil {
		return fmt.Errorf("failed to stop server %s: %w", name, err)
	}
	
	// Remove from registry
	delete(m.servers, name)
	
	// Call registration hooks
	for _, hook := range m.registrationHooks {
		hook(instance, false)
	}
	
	// Record metrics
	m.metricsCollector.RecordServerUnregistration(name)
	
	return nil
}

// GetServer returns a server instance by name
func (m *Manager) GetServer(name string) (*ServerInstance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	instance, exists := m.servers[name]
	if !exists {
		return nil, core.ErrServerNotFound
	}
	
	return instance, nil
}

// ListServers returns all registered server instances
func (m *Manager) ListServers() []*ServerInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	servers := make([]*ServerInstance, 0, len(m.servers))
	for _, server := range m.servers {
		servers = append(servers, server)
	}
	
	return servers
}

// RestartServer restarts a server
func (m *Manager) RestartServer(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	instance, exists := m.servers[name]
	if !exists {
		return core.ErrServerNotFound
	}
	
	instance.mu.Lock()
	defer instance.mu.Unlock()
	
	// Update status
	oldStatus := instance.Status
	instance.Status = StatusRestarting
	instance.RestartCount++
	
	// Stop the server
	if err := m.stopServerInternal(ctx, instance); err != nil {
		instance.Status = oldStatus
		return fmt.Errorf("failed to stop server during restart: %w", err)
	}
	
	// Add delay before restart
	time.Sleep(instance.Config.RestartDelay)
	
	// Start the server again
	if err := m.startServerInternal(ctx, instance); err != nil {
		instance.Status = StatusFailed
		return fmt.Errorf("failed to start server during restart: %w", err)
	}
	
	// Record metrics
	m.metricsCollector.RecordServerRestart(name)
	
	return nil
}

// AddRegistrationHook adds a hook that's called on server registration/unregistration
func (m *Manager) AddRegistrationHook(hook RegistrationHook) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registrationHooks = append(m.registrationHooks, hook)
}

// GetMetrics returns current metrics for all servers
func (m *Manager) GetMetrics() map[string]interface{} {
	return m.metricsCollector.GetMetrics()
}

// GetDiscoveryService returns the discovery service
func (m *Manager) GetDiscoveryService() *DiscoveryService {
	return m.discoveryService
}

// GetHealthMonitor returns the health monitor
func (m *Manager) GetHealthMonitor() *HealthMonitor {
	return m.healthMonitor
}

// startServerInternal starts a server (must be called with lock held)
func (m *Manager) startServerInternal(ctx context.Context, instance *ServerInstance) error {
	if m.clientFactory == nil {
		return fmt.Errorf("client factory not set")
	}
	
	// Create MCP client using factory
	client, err := m.clientFactory(instance.Config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	
	// Connect to server
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	
	instance.Client = client
	instance.Status = StatusHealthy
	instance.LastHealthy = time.Now()
	instance.StartedAt = time.Now()
	
	// Get server info
	if serverInfo := client.GetServerInfo(); serverInfo != nil {
		instance.Info = &ServerInfo{
			Name:        serverInfo.Name,
			Version:     serverInfo.Version,
			Description: instance.Config.Description,
			ToolCount:   len(client.GetTools()),
		}
	}
	
	return nil
}

// stopServerInternal stops a server (must be called with lock held)
func (m *Manager) stopServerInternal(ctx context.Context, instance *ServerInstance) error {
	if instance.Client != nil {
		if err := instance.Client.Close(); err != nil {
			return err
		}
		instance.Client = nil
	}
	
	instance.Status = StatusStopped
	return nil
}

// autoStartServers starts servers configured for auto-start
func (m *Manager) autoStartServers(ctx context.Context) error {
	for _, serverConfig := range m.config.Servers {
		if serverConfig.AutoStart {
			if err := m.RegisterServer(ctx, serverConfig); err != nil {
				// Log error but continue with other servers
				fmt.Printf("Failed to auto-start server %s: %v\n", serverConfig.Name, err)
			}
		}
	}
	return nil
}

// Shutdown gracefully shuts down the manager
func (m *Manager) Shutdown(ctx context.Context) error {
	close(m.shutdownCh)
	
	// Stop all servers
	m.mu.Lock()
	defer m.mu.Unlock()
	
	var lastErr error
	for name, instance := range m.servers {
		if err := m.stopServerInternal(ctx, instance); err != nil {
			lastErr = fmt.Errorf("failed to stop server %s: %w", name, err)
		}
	}
	
	// Stop background services
	m.healthMonitor.Stop()
	m.discoveryService.Stop()
	m.metricsCollector.Stop()
	
	return lastErr
}

// String returns the string representation of ServerStatus
func (s ServerStatus) String() string {
	switch s {
	case StatusUnknown:
		return "Unknown"
	case StatusStarting:
		return "Starting"
	case StatusHealthy:
		return "Healthy"
	case StatusUnhealthy:
		return "Unhealthy"
	case StatusRestarting:
		return "Restarting"
	case StatusStopped:
		return "Stopped"
	case StatusFailed:
		return "Failed"
	default:
		return "Invalid"
	}
}