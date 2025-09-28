package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	mcpswagger "github.com/liliang-cn/mcp-swagger-server"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp/transport"
)

// SwaggerConfig represents configuration for a Swagger-based MCP server
type SwaggerConfig struct {
	// Name is the identifier for this Swagger server
	Name string `json:"name" yaml:"name"`
	
	// SwaggerURL is the URL to fetch the Swagger spec from
	SwaggerURL string `json:"swagger_url" yaml:"swagger_url"`
	
	// SwaggerFile is the local file path to a Swagger spec (alternative to URL)
	SwaggerFile string `json:"swagger_file" yaml:"swagger_file"`
	
	// SwaggerData is raw Swagger spec data (JSON or YAML)
	SwaggerData []byte `json:"-" yaml:"-"`
	
	// Transport specifies the transport type (stdio or http)
	Transport string `json:"transport" yaml:"transport"`
	
	// HTTPConfig is configuration for HTTP transport
	HTTPConfig *HTTPTransportConfig `json:"http_config,omitempty" yaml:"http_config,omitempty"`
	
	// Timeout for operations
	Timeout time.Duration `json:"timeout" yaml:"timeout"`
	
	// BaseURL to override the base URL in the Swagger spec
	BaseURL string `json:"base_url,omitempty" yaml:"base_url,omitempty"`
	
	// Headers to include in API requests
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	
	// Auth configuration
	Auth *SwaggerAuthConfig `json:"auth,omitempty" yaml:"auth,omitempty"`
}

// SwaggerAuthConfig represents authentication configuration
type SwaggerAuthConfig struct {
	Type   string `json:"type" yaml:"type"`     // basic, bearer, apikey
	Value  string `json:"value" yaml:"value"`   // token or key value
	Header string `json:"header,omitempty" yaml:"header,omitempty"` // header name for API key
}

// HTTPTransportConfig represents HTTP transport configuration
type HTTPTransportConfig struct {
	Host string `json:"host" yaml:"host"`
	Port int    `json:"port" yaml:"port"`
}

// SwaggerServer wraps an mcp-swagger-server instance
type SwaggerServer struct {
	config    *SwaggerConfig
	server    *mcpswagger.Server
	transport transport.Transport
	running   bool
}

// NewSwaggerServer creates a new Swagger-based MCP server
func NewSwaggerServer(config *SwaggerConfig) (*SwaggerServer, error) {
	if config == nil {
		return nil, fmt.Errorf("swagger config cannot be nil")
	}
	
	// Set defaults
	if config.Transport == "" {
		config.Transport = "stdio"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	
	// Create the server
	var server *mcpswagger.Server
	var err error
	
	if len(config.SwaggerData) > 0 {
		// Use raw data
		server, err = mcpswagger.NewFromSwaggerData(config.SwaggerData)
	} else if config.SwaggerFile != "" {
		// Load from file
		data, err := os.ReadFile(config.SwaggerFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read swagger file: %w", err)
		}
		server, err = mcpswagger.NewFromSwaggerData(data)
	} else if config.SwaggerURL != "" {
		// Fetch from URL
		data, err := fetchSwaggerSpec(config.SwaggerURL, config.Headers, config.Timeout)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch swagger spec: %w", err)
		}
		server, err = mcpswagger.NewFromSwaggerData(data)
	} else {
		return nil, fmt.Errorf("no swagger source specified (need URL, file, or data)")
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to create swagger server: %w", err)
	}
	
	// Configure auth if provided
	if config.Auth != nil {
		configureAuth(server, config.Auth)
	}
	
	// Override base URL if provided
	if config.BaseURL != "" {
		server.SetBaseURL(config.BaseURL)
	}
	
	return &SwaggerServer{
		config: config,
		server: server,
	}, nil
}

// Start starts the Swagger MCP server
func (s *SwaggerServer) Start(ctx context.Context) error {
	if s.running {
		return fmt.Errorf("server already running")
	}
	
	var err error
	
	switch s.config.Transport {
	case "http":
		err = s.startHTTP(ctx)
	case "stdio":
		err = s.startStdio(ctx)
	default:
		return fmt.Errorf("unsupported transport: %s", s.config.Transport)
	}
	
	if err != nil {
		return err
	}
	
	s.running = true
	return nil
}

// startHTTP starts the server with HTTP transport
func (s *SwaggerServer) startHTTP(ctx context.Context) error {
	if s.config.HTTPConfig == nil {
		s.config.HTTPConfig = &HTTPTransportConfig{
			Host: "localhost",
			Port: 3000,
		}
	}
	
	// The mcp-swagger-server v0.4.0 supports HTTP transport
	// Run the server with HTTP transport
	go func() {
		err := s.server.Run()
		if err != nil {
			fmt.Printf("Swagger server error: %v\n", err)
		}
	}()
	
	return nil
}

// startStdio starts the server with stdio transport
func (s *SwaggerServer) startStdio(ctx context.Context) error {
	// The mcp-swagger-server v0.4.0 auto-detects transport
	// Run with stdio by default
	go func() {
		err := s.server.Run()
		if err != nil {
			fmt.Printf("Swagger server error: %v\n", err)
		}
	}()
	
	return nil
}

// Stop stops the Swagger MCP server
func (s *SwaggerServer) Stop() error {
	if !s.running {
		return fmt.Errorf("server not running")
	}
	
	// Stop the server (implementation depends on mcp-swagger-server API)
	// For now, just mark as not running
	s.running = false
	return nil
}

// GetTools returns the available tools from the Swagger spec
func (s *SwaggerServer) GetTools() ([]*mcp.Tool, error) {
	if s.server == nil {
		return nil, fmt.Errorf("server not initialized")
	}
	
	// Get tools from the swagger server
	tools := s.server.GetTools()
	
	// Convert to MCP SDK tools format
	mcpTools := make([]*mcp.Tool, 0, len(tools))
	for _, tool := range tools {
		mcpTool := &mcp.Tool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		}
		mcpTools = append(mcpTools, mcpTool)
	}
	
	return mcpTools, nil
}

// CallTool calls a tool with the given arguments
func (s *SwaggerServer) CallTool(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	if s.server == nil {
		return nil, fmt.Errorf("server not initialized")
	}
	
	// Call the tool through the swagger server
	result, err := s.server.CallTool(ctx, name, args)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool %s: %w", name, err)
	}
	
	// Convert result to JSON
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	
	return resultJSON, nil
}

// fetchSwaggerSpec fetches a Swagger spec from a URL
func fetchSwaggerSpec(specURL string, headers map[string]string, timeout time.Duration) ([]byte, error) {
	// Parse and validate URL
	u, err := url.Parse(specURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	
	// Support file:// URLs for local files
	if u.Scheme == "file" {
		path := u.Path
		if u.Host != "" && u.Host != "localhost" {
			// On Windows, file://host/path is valid
			path = filepath.Join(u.Host, path)
		}
		return os.ReadFile(path)
	}
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: timeout,
	}
	
	// Create request
	req, err := http.NewRequest("GET", specURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	
	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spec: %w", err)
	}
	defer resp.Body.Close()
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	// Read response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	return data, nil
}

// configureAuth configures authentication for the Swagger server
func configureAuth(server *mcpswagger.Server, auth *SwaggerAuthConfig) {
	switch strings.ToLower(auth.Type) {
	case "bearer":
		server.SetAuthHeader("Authorization", "Bearer "+auth.Value)
	case "basic":
		server.SetAuthHeader("Authorization", "Basic "+auth.Value)
	case "apikey":
		if auth.Header != "" {
			server.SetAuthHeader(auth.Header, auth.Value)
		} else {
			server.SetAuthHeader("X-API-Key", auth.Value)
		}
	}
}

// LoadSwaggerServers loads Swagger server configurations from config
func LoadSwaggerServers(configPath string) (map[string]*SwaggerConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	var config struct {
		SwaggerServers map[string]*SwaggerConfig `json:"swagger_servers" yaml:"swagger_servers"`
	}
	
	// Try JSON first
	if err := json.Unmarshal(data, &config); err != nil {
		// If JSON fails, config might be YAML (not implemented here for simplicity)
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	
	return config.SwaggerServers, nil
}