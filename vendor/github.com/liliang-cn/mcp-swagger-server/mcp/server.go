package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/go-openapi/spec"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type SwaggerMCPServer struct {
	server     *mcp.Server
	apiBaseURL string
	swagger    *spec.Swagger
	apiKey     string
	filter     *APIFilter
}

// NewSwaggerMCPServer creates a new MCP server from Swagger spec
func NewSwaggerMCPServer(apiBaseURL string, swaggerSpec *spec.Swagger, apiKey string) *SwaggerMCPServer {
	return NewSwaggerMCPServerWithFilter(apiBaseURL, swaggerSpec, apiKey, nil)
}

// NewSwaggerMCPServerWithFilter creates a new MCP server from Swagger spec with filtering
func NewSwaggerMCPServerWithFilter(apiBaseURL string, swaggerSpec *spec.Swagger, apiKey string, filter *APIFilter) *SwaggerMCPServer {
	// Create MCP server with Implementation
	implementation := &mcp.Implementation{
		Name:    "swagger-mcp-server",
		Version: "v0.1.0",
	}

	server := mcp.NewServer(implementation, nil)

	// Create converter
	converter := &SwaggerMCPServer{
		server:     server,
		apiBaseURL: apiBaseURL,
		swagger:    swaggerSpec,
		apiKey:     apiKey,
		filter:     filter,
	}

	// Register tools from Swagger
	converter.RegisterTools()

	return converter
}

// Run starts the MCP server with stdio transport
func (s *SwaggerMCPServer) Run(ctx context.Context) error {
	// Create stdio transport
	transport := &mcp.StdioTransport{}

	log.Println("Starting MCP server from Swagger...")

	// Connect and run the server
	session, err := s.server.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("failed to connect MCP server: %w", err)
	}

	// Wait for the session to end
	_ = session.Wait()
	return nil
}

// RegisterTools creates MCP tools from Swagger endpoints
func (s *SwaggerMCPServer) RegisterTools() {
	for path, pathItem := range s.swagger.Paths.Paths {
		s.registerPathTools(path, pathItem)
	}
}

func (s *SwaggerMCPServer) registerPathTools(path string, pathItem spec.PathItem) {
	// Register GET endpoints
	if pathItem.Get != nil {
		s.registerOperation("GET", path, pathItem.Get)
	}

	// Register POST endpoints
	if pathItem.Post != nil {
		s.registerOperation("POST", path, pathItem.Post)
	}

	// Register PUT endpoints
	if pathItem.Put != nil {
		s.registerOperation("PUT", path, pathItem.Put)
	}

	// Register DELETE endpoints
	if pathItem.Delete != nil {
		s.registerOperation("DELETE", path, pathItem.Delete)
	}

	// Register PATCH endpoints
	if pathItem.Patch != nil {
		s.registerOperation("PATCH", path, pathItem.Patch)
	}
}

func (s *SwaggerMCPServer) registerOperation(method, path string, op *spec.Operation) {
	// Check if this operation should be excluded
	if s.filter != nil && s.filter.ShouldExcludeOperation(method, path, op) {
		return // Skip this operation
	}

	// Generate tool name
	toolName := ""
	if op.ID != "" {
		toolName = strings.ReplaceAll(op.ID, " ", "_")
		toolName = strings.ToLower(toolName)
	} else {
		// Create tool name from method and path
		toolName = strings.ToLower(method) + "_"
		pathName := strings.ReplaceAll(path, "/", "_")
		pathName = strings.ReplaceAll(pathName, "{", "")
		pathName = strings.ReplaceAll(pathName, "}", "")
		pathName = strings.TrimPrefix(pathName, "_")
		toolName += pathName
	}

	// Build description
	description := op.Summary
	if description == "" {
		description = op.Description
	}
	if description == "" {
		description = fmt.Sprintf("%s %s", method, path)
	}

	// Build parameters schema
	schema := s.buildParametersSchema(op.Parameters)

	// Create tool
	tool := &mcp.Tool{
		Name:        toolName,
		Description: description,
		InputSchema: schema,
	}

	// Create handler function that wraps our logic
	handler := s.createHandler(method, path, op)

	// Register the tool
	s.server.AddTool(tool, handler)
}

func (s *SwaggerMCPServer) buildParametersSchema(params []spec.Parameter) interface{} {
	properties := make(map[string]interface{})
	required := []string{}

	for _, param := range params {
		// Skip header and cookie params
		if param.In == "header" && !strings.EqualFold(param.Name, "content-type") {
			continue
		}
		if param.In == "cookie" {
			continue
		}

		// Create parameter schema based on type
		paramSchema := make(map[string]interface{})
		
		if param.Type != "" {
			paramSchema["type"] = getJSONType(param.Type)
		} else if param.Schema != nil {
			// Handle body parameters with schema
			if len(param.Schema.Type) > 0 {
				paramSchema["type"] = param.Schema.Type[0]
			} else {
				paramSchema["type"] = "object"
			}
			
			// Add properties if available
			if param.Schema.Properties != nil {
				props := make(map[string]interface{})
				for name, prop := range param.Schema.Properties {
					propSchema := make(map[string]interface{})
					if len(prop.Type) > 0 {
						propSchema["type"] = prop.Type[0]
					}
					if prop.Description != "" {
						propSchema["description"] = prop.Description
					}
					props[name] = propSchema
				}
				paramSchema["properties"] = props
			}
		}

		if param.Description != "" {
			paramSchema["description"] = param.Description
		}

		// Add format if specified
		if param.Format != "" {
			paramSchema["format"] = param.Format
		}

		// Handle array items
		if param.Type == "array" && param.Items != nil {
			itemSchema := make(map[string]interface{})
			if param.Items.Type != "" {
				itemSchema["type"] = getJSONType(param.Items.Type)
			}
			paramSchema["items"] = itemSchema
		}

		// Add to properties
		paramName := param.Name
		if param.In == "body" {
			// For body parameters, use "body" as the key
			paramName = "body"
		}
		properties[paramName] = paramSchema

		// Add to required if necessary
		if param.Required {
			required = append(required, paramName)
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

// Create a handler function that works as a basic ToolHandler
func (s *SwaggerMCPServer) createHandler(method, path string, op *spec.Operation) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters from the request arguments
		var params map[string]interface{}
		if req.Params.Arguments != nil {
			if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
				return nil, fmt.Errorf("failed to parse arguments: %w", err)
			}
		}
		if params == nil {
			params = make(map[string]interface{})
		}

		// Build URL with path parameters
		url := s.apiBaseURL + path
		
		// Extract body parameter if present
		var bodyData interface{}
		if body, exists := params["body"]; exists {
			bodyData = body
			delete(params, "body")
		}

		// Replace path parameters
		for key, value := range params {
			placeholder := "{" + key + "}"
			if strings.Contains(url, placeholder) {
				url = strings.ReplaceAll(url, placeholder, fmt.Sprintf("%v", value))
				delete(params, key) // Remove from params since it's in the URL
			}
		}

		// Prepare request
		var body io.Reader
		if method == "POST" || method == "PUT" || method == "PATCH" {
			// Use body data if available, otherwise use remaining params
			var dataToSend interface{}
			if bodyData != nil {
				dataToSend = bodyData
			} else if len(params) > 0 {
				dataToSend = params
			}
			
			if dataToSend != nil {
				jsonData, err := json.Marshal(dataToSend)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal request body: %w", err)
				}
				body = bytes.NewReader(jsonData)
			}
		} else {
			// Add remaining params as query parameters
			if len(params) > 0 {
				queryParams := []string{}
				for key, value := range params {
					queryParams = append(queryParams, fmt.Sprintf("%s=%v", key, value))
				}
				if strings.Contains(url, "?") {
					url += "&" + strings.Join(queryParams, "&")
				} else {
					url += "?" + strings.Join(queryParams, "&")
				}
			}
		}

		// Create HTTP request
		httpReq, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		if body != nil {
			httpReq.Header.Set("Content-Type", "application/json")
		}
		httpReq.Header.Set("Accept", "application/json")
		
		// Add API key if configured
		if s.apiKey != "" {
			// Try different common API key header formats
			httpReq.Header.Set("X-API-Key", s.apiKey)
			httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)
		}

		// Execute request
		client := &http.Client{}
		resp, err := client.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		// Read response
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		// Check status code
		if resp.StatusCode >= 400 {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: fmt.Sprintf("API error %d: %s", resp.StatusCode, string(responseBody)),
					},
				},
				IsError: true,
			}, nil
		}

		// Try to format JSON response
		var jsonResponse interface{}
		if err := json.Unmarshal(responseBody, &jsonResponse); err == nil {
			// Successfully parsed as JSON, format it
			formattedJSON, _ := json.MarshalIndent(jsonResponse, "", "  ")
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(formattedJSON),
					},
				},
			}, nil
		}

		// Return as plain text if not JSON
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: string(responseBody),
				},
			},
		}, nil
	}
}

func getJSONType(swaggerType string) string {
	switch swaggerType {
	case "integer":
		return "number"
	case "number":
		return "number"
	case "boolean":
		return "boolean"
	case "array":
		return "array"
	case "object":
		return "object"
	default:
		return "string"
	}
}