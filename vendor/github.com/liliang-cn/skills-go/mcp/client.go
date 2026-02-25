package mcp

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"sync"

	"github.com/liliang-cn/skills-go/skill"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPServer represents a connected MCP server that can be used by skills
type MCPServer struct {
	config       *ServerConfig
	client       *mcpsdk.Client
	session      *mcpsdk.ClientSession
	capabilities *ServerCapabilities
	mu           sync.RWMutex
}

// Manager manages multiple MCP server connections
type Manager struct {
	servers map[string]*MCPServer
	mu      sync.RWMutex
}

// NewManager creates a new MCP server manager
func NewManager() *Manager {
	return &Manager{
		servers: make(map[string]*MCPServer),
	}
}

// Connect connects to an MCP server
func (m *Manager) Connect(ctx context.Context, cfg *ServerConfig) (*MCPServer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already connected
	if srv, exists := m.servers[cfg.Name]; exists {
		return srv, nil
	}

	transport, err := createTransportForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}

	client := mcpsdk.NewClient(&mcpsdk.Implementation{
		Name:    "skills-go-mcp-manager",
		Version: "1.0.0",
	}, nil)

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server %s: %w", cfg.Name, err)
	}

	// Discover capabilities
	caps, err := discoverCapabilities(ctx, session, DefaultInclude())
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to discover capabilities: %w", err)
	}

	srv := &MCPServer{
		config:      cfg,
		client:      client,
		session:     session,
		capabilities: caps,
	}

	m.servers[cfg.Name] = srv
	return srv, nil
}

// Disconnect disconnects from an MCP server
func (m *Manager) Disconnect(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	srv, exists := m.servers[name]
	if !exists {
		return nil
	}

	if err := srv.session.Close(); err != nil {
		return fmt.Errorf("failed to close session: %w", err)
	}

	delete(m.servers, name)
	return nil
}

// DisconnectAll disconnects all servers
func (m *Manager) DisconnectAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for name, srv := range m.servers {
		if err := srv.session.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close %s: %w", name, err))
		}
	}

	m.servers = make(map[string]*MCPServer)

	if len(errs) > 0 {
		return fmt.Errorf("encountered %d errors: %v", len(errs), errs)
	}

	return nil
}

// GetServer returns a connected server by name
func (m *Manager) GetServer(name string) (*MCPServer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	srv, exists := m.servers[name]
	return srv, exists
}

// ListServers returns all connected server names
func (m *Manager) ListServers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.servers))
	for name := range m.servers {
		names = append(names, name)
	}
	return names
}

// CallTool calls a tool on a connected server
func (s *MCPServer) CallTool(ctx context.Context, name string, args map[string]any) (*mcpsdk.CallToolResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.session == nil {
		return nil, fmt.Errorf("server not connected")
	}

	params := &mcpsdk.CallToolParams{
		Name:      name,
		Arguments: args,
	}

	return s.session.CallTool(ctx, params)
}

// ReadResource reads a resource from a connected server
func (s *MCPServer) ReadResource(ctx context.Context, uri string) (*mcpsdk.ReadResourceResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.session == nil {
		return nil, fmt.Errorf("server not connected")
	}

	params := &mcpsdk.ReadResourceParams{
		URI: uri,
	}

	return s.session.ReadResource(ctx, params)
}

// GetPrompt gets a prompt from a connected server
func (s *MCPServer) GetPrompt(ctx context.Context, name string, args map[string]string) (*mcpsdk.GetPromptResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.session == nil {
		return nil, fmt.Errorf("server not connected")
	}

	params := &mcpsdk.GetPromptParams{
		Name:      name,
		Arguments: args,
	}

	return s.session.GetPrompt(ctx, params)
}

// Capabilities returns the server's capabilities
func (s *MCPServer) Capabilities() *ServerCapabilities {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.capabilities
}

// ListTools returns available tool names
func (s *MCPServer) ListTools() []string {
	if s.capabilities == nil {
		return nil
	}
	return s.capabilities.ListTools()
}

// ListResources returns available resource URIs
func (s *MCPServer) ListResources() []string {
	if s.capabilities == nil {
		return nil
	}
	return s.capabilities.ListResources()
}

// ListPrompts returns available prompt names
func (s *MCPServer) ListPrompts() []string {
	if s.capabilities == nil {
		return nil
	}
	return s.capabilities.ListPrompts()
}

// Config returns the server config
func (s *MCPServer) Config() *ServerConfig {
	return s.config
}

// createTransportForConfig creates a transport for the given config
func createTransportForConfig(cfg *ServerConfig) (mcpsdk.Transport, error) {
	if len(cfg.Command) > 0 {
		// Parse command
		var cmd string
		var args []string

		if len(cfg.Command) == 1 {
			parts := strings.Fields(cfg.Command[0])
			if len(parts) > 0 {
				cmd = parts[0]
				args = parts[1:]
			}
		} else {
			cmd = cfg.Command[0]
			args = cfg.Command[1:]
		}

		if cmd == "" {
			return nil, fmt.Errorf("invalid command: %v", cfg.Command)
		}

		return &mcpsdk.CommandTransport{
			Command: exec.Command(cmd, args...),
		}, nil
	}

	if cfg.URL != "" {
		return &mcpsdk.StreamableClientTransport{
			Endpoint:   cfg.URL,
			HTTPClient: http.DefaultClient,
		}, nil
	}

	return nil, fmt.Errorf("either Command or URL must be specified in ServerConfig")
}

// discoverCapabilities discovers the capabilities of a connected session
func discoverCapabilities(ctx context.Context, session *mcpsdk.ClientSession, include IncludeConfig) (*ServerCapabilities, error) {
	caps := &ServerCapabilities{}

	if include.Tools {
		tools, err := session.ListTools(ctx, nil)
		if err == nil && tools != nil {
			caps.Tools = tools.Tools
		}
	}

	if include.Resources {
		resources, err := session.ListResources(ctx, nil)
		if err == nil && resources != nil {
			caps.Resources = resources.Resources
		}
		templates, err := session.ListResourceTemplates(ctx, nil)
		if err == nil && templates != nil {
			caps.ResourceTemplates = templates.ResourceTemplates
		}
	}

	if include.Prompts {
		prompts, err := session.ListPrompts(ctx, nil)
		if err == nil && prompts != nil {
			caps.Prompts = prompts.Prompts
		}
	}

	return caps, nil
}

// ConnectToCommandServer is a helper to connect to a stdio-based MCP server
func ConnectToCommandServer(ctx context.Context, name string, command string, args ...string) (*MCPServer, error) {
	mgr := NewManager()
	cfg := &ServerConfig{
		Name:    name,
		Command: append([]string{command}, args...),
	}
	return mgr.Connect(ctx, cfg)
}

// ConnectToHTTPServer is a helper to connect to an HTTP-based MCP server
func ConnectToHTTPServer(ctx context.Context, name string, url string) (*MCPServer, error) {
	mgr := NewManager()
	cfg := &ServerConfig{
		Name: name,
		URL:  url,
	}
	return mgr.Connect(ctx, cfg)
}

// NewCommand creates a ServerConfig for a command-based server
func NewCommand(name string, command string, args ...string) *ServerConfig {
	return &ServerConfig{
		Name:    name,
		Command: append([]string{command}, args...),
		Include: DefaultInclude(),
	}
}

// NewHTTP creates a ServerConfig for an HTTP-based server
func NewHTTP(name string, url string) *ServerConfig {
	return &ServerConfig{
		Name:    name,
		URL:     url,
		Include: DefaultInclude(),
	}
}

// QuickConvert converts a command-based server to a skill in one call
func QuickConvert(ctx context.Context, command string, outputDir string, args ...string) (*skill.Skill, error) {
	cfg := &ServerConfig{
		Command: append([]string{command}, args...),
		Include: DefaultInclude(),
	}
	c := NewConverter()
	return c.Convert(ctx, cfg, outputDir)
}

// QuickConvertHTTP converts an HTTP-based server to a skill in one call
func QuickConvertHTTP(ctx context.Context, url string, outputDir string) (*skill.Skill, error) {
	cfg := &ServerConfig{
		URL:     url,
		Include: DefaultInclude(),
	}
	c := NewConverter()
	return c.Convert(ctx, cfg, outputDir)
}

// QuickDiscover connects and discovers capabilities without converting
func QuickDiscover(ctx context.Context, command string, args ...string) (*ServerCapabilities, error) {
	mgr := NewManager()
	name := fmt.Sprintf("server-%d", len(mgr.ListServers()))
	cfg := &ServerConfig{
		Name:    name,
		Command: append([]string{command}, args...),
		Include: DefaultInclude(),
	}

	srv, err := mgr.Connect(ctx, cfg)
	if err != nil {
		return nil, err
	}
	defer mgr.Disconnect(ctx, name)

	return srv.Capabilities(), nil
}
