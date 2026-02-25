package mcp

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/go-openapi/spec"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// APIFilter represents different types of API filtering
type APIFilter struct {
	// Paths to exclude from tool conversion (exact match)
	ExcludePaths []string
	
	// Path patterns to exclude (supports wildcards like /api/v1/*)
	ExcludePathPatterns []string
	
	// Operation IDs to exclude
	ExcludeOperationIDs []string
	
	// HTTP methods to exclude (e.g., ["DELETE", "PATCH"])
	ExcludeMethods []string
	
	// Tag-based filtering - exclude operations with these tags
	ExcludeTags []string
	
	// Include only specific paths (if provided, only these will be included)
	IncludeOnlyPaths []string
	
	// Include only specific operation IDs
	IncludeOnlyOperationIDs []string
}

// Config holds the configuration for the MCP server
type Config struct {
	// API configuration
	APIBaseURL string
	APIKey     string
	
	// Swagger specification
	SwaggerSpec *spec.Swagger
	SwaggerData []byte // Raw swagger data for lazy loading
	
	// Server configuration
	Name        string
	Version     string
	Description string
	
	// Transport configuration
	Transport Transport
	
	// API filtering configuration
	Filter *APIFilter
}

// Transport interface for different transport methods
type Transport interface {
	Connect(ctx context.Context, server *mcp.Server) (*mcp.ServerSession, error)
}

// StdioTransport implements stdio transport
type StdioTransport struct{}

func (t *StdioTransport) Connect(ctx context.Context, server *mcp.Server) (*mcp.ServerSession, error) {
	transport := &mcp.StdioTransport{}
	return server.Connect(ctx, transport, nil)
}

// HTTPTransport implements HTTP transport
type HTTPTransport struct {
	Port   int
	Host   string
	Path   string
	Writer io.Writer // For response output
}

func (t *HTTPTransport) Connect(ctx context.Context, server *mcp.Server) (*mcp.ServerSession, error) {
	// HTTP transport doesn't use the standard MCP session model
	// It should be handled differently in the Server.Run method
	// This is a placeholder that should not be called for HTTP transport
	return nil, fmt.Errorf("HTTP transport requires special handling, use Server.RunHTTP instead")
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Name:        "swagger-mcp-server",
		Version:     "v0.2.0",
		Description: "MCP server generated from Swagger/OpenAPI specification",
		Transport:   &StdioTransport{},
	}
}

// WithSwaggerSpec sets the swagger specification
func (c *Config) WithSwaggerSpec(swagger *spec.Swagger) *Config {
	c.SwaggerSpec = swagger
	return c
}

// WithSwaggerData sets raw swagger data for lazy loading
func (c *Config) WithSwaggerData(data []byte) *Config {
	c.SwaggerData = data
	return c
}

// WithAPIConfig sets API configuration
func (c *Config) WithAPIConfig(baseURL, apiKey string) *Config {
	c.APIBaseURL = baseURL
	c.APIKey = apiKey
	return c
}

// WithTransport sets the transport method
func (c *Config) WithTransport(transport Transport) *Config {
	c.Transport = transport
	return c
}

// WithServerInfo sets server information
func (c *Config) WithServerInfo(name, version, description string) *Config {
	c.Name = name
	c.Version = version
	c.Description = description
	return c
}

// WithAPIFilter sets the API filtering configuration
func (c *Config) WithAPIFilter(filter *APIFilter) *Config {
	c.Filter = filter
	return c
}

// WithExcludePaths sets paths to exclude from tool conversion
func (c *Config) WithExcludePaths(paths ...string) *Config {
	if c.Filter == nil {
		c.Filter = &APIFilter{}
	}
	c.Filter.ExcludePaths = append(c.Filter.ExcludePaths, paths...)
	return c
}

// WithExcludePathPatterns sets path patterns to exclude
func (c *Config) WithExcludePathPatterns(patterns ...string) *Config {
	if c.Filter == nil {
		c.Filter = &APIFilter{}
	}
	c.Filter.ExcludePathPatterns = append(c.Filter.ExcludePathPatterns, patterns...)
	return c
}

// WithExcludeOperationIDs sets operation IDs to exclude
func (c *Config) WithExcludeOperationIDs(ids ...string) *Config {
	if c.Filter == nil {
		c.Filter = &APIFilter{}
	}
	c.Filter.ExcludeOperationIDs = append(c.Filter.ExcludeOperationIDs, ids...)
	return c
}

// WithExcludeMethods sets HTTP methods to exclude
func (c *Config) WithExcludeMethods(methods ...string) *Config {
	if c.Filter == nil {
		c.Filter = &APIFilter{}
	}
	c.Filter.ExcludeMethods = append(c.Filter.ExcludeMethods, methods...)
	return c
}

// WithExcludeTags sets tags to exclude
func (c *Config) WithExcludeTags(tags ...string) *Config {
	if c.Filter == nil {
		c.Filter = &APIFilter{}
	}
	c.Filter.ExcludeTags = append(c.Filter.ExcludeTags, tags...)
	return c
}

// WithIncludeOnlyPaths sets paths to include exclusively
func (c *Config) WithIncludeOnlyPaths(paths ...string) *Config {
	if c.Filter == nil {
		c.Filter = &APIFilter{}
	}
	c.Filter.IncludeOnlyPaths = append(c.Filter.IncludeOnlyPaths, paths...)
	return c
}

// WithIncludeOnlyOperationIDs sets operation IDs to include exclusively
func (c *Config) WithIncludeOnlyOperationIDs(ids ...string) *Config {
	if c.Filter == nil {
		c.Filter = &APIFilter{}
	}
	c.Filter.IncludeOnlyOperationIDs = append(c.Filter.IncludeOnlyOperationIDs, ids...)
	return c
}

// ShouldExcludeOperation checks if an operation should be excluded from tool conversion
func (f *APIFilter) ShouldExcludeOperation(method, path string, operation *spec.Operation) bool {
	if f == nil {
		return false
	}

	// Check include-only filters first (if any are set, only those should be included)
	if len(f.IncludeOnlyPaths) > 0 {
		found := false
		for _, includePath := range f.IncludeOnlyPaths {
			if path == includePath {
				found = true
				break
			}
		}
		if !found {
			return true // Exclude if not in include-only list
		}
	}

	if len(f.IncludeOnlyOperationIDs) > 0 && operation.ID != "" {
		found := false
		for _, includeID := range f.IncludeOnlyOperationIDs {
			if operation.ID == includeID {
				found = true
				break
			}
		}
		if !found {
			return true // Exclude if not in include-only list
		}
	}

	// Check exclude filters
	
	// Exclude by exact path match
	for _, excludePath := range f.ExcludePaths {
		if path == excludePath {
			return true
		}
	}

	// Exclude by path pattern match
	for _, pattern := range f.ExcludePathPatterns {
		if matchesPattern(path, pattern) {
			return true
		}
	}

	// Exclude by operation ID
	if operation.ID != "" {
		for _, excludeID := range f.ExcludeOperationIDs {
			if operation.ID == excludeID {
				return true
			}
		}
	}

	// Exclude by HTTP method
	for _, excludeMethod := range f.ExcludeMethods {
		if strings.EqualFold(method, excludeMethod) {
			return true
		}
	}

	// Exclude by tags
	if len(f.ExcludeTags) > 0 && len(operation.Tags) > 0 {
		for _, opTag := range operation.Tags {
			for _, excludeTag := range f.ExcludeTags {
				if strings.EqualFold(opTag, excludeTag) {
					return true
				}
			}
		}
	}

	return false
}

// matchesPattern checks if a path matches a pattern with wildcard support
func matchesPattern(path, pattern string) bool {
	// Simple wildcard matching using filepath.Match
	// Convert API path pattern to filepath pattern
	// Replace {param} with * for wildcard matching
	convertedPattern := pattern
	
	// Handle path parameters in patterns
	if strings.Contains(convertedPattern, "{") {
		// Replace {anything} with *
		parts := strings.Split(convertedPattern, "/")
		for i, part := range parts {
			if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
				parts[i] = "*"
			}
		}
		convertedPattern = strings.Join(parts, "/")
	}
	
	matched, _ := filepath.Match(convertedPattern, path)
	return matched
}