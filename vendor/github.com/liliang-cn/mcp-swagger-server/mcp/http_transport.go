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
)

// HTTPServer wraps the MCP server for HTTP transport
type HTTPServer struct {
	server     *Server
	port       int
	host       string
	path       string
	httpServer *http.Server
}

// NewHTTPServer creates a new HTTP server wrapper
func NewHTTPServer(server *Server, port int, host, path string) *HTTPServer {
	if host == "" {
		host = "localhost"
	}
	if path == "" {
		path = "/mcp"
	}

	return &HTTPServer{
		server: server,
		port:   port,
		host:   host,
		path:   path,
	}
}

// Start starts the HTTP server
func (h *HTTPServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	
	// Add CORS middleware
	corsHandler := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			next(w, r)
		}
	}
	
	// MCP endpoint
	mux.HandleFunc(h.path, corsHandler(h.handleMCPRequest))
	
	// Health check endpoint
	mux.HandleFunc("/health", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
			log.Printf("Failed to encode health response: %v", err)
		}
	}))
	
	// Tools list endpoint
	mux.HandleFunc("/tools", corsHandler(h.handleToolsList))

	addr := fmt.Sprintf("%s:%d", h.host, h.port)
	h.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	log.Printf("Starting HTTP MCP server on %s%s", addr, h.path)
	
	go func() {
		<-ctx.Done()
		if err := h.httpServer.Shutdown(context.Background()); err != nil {
			log.Printf("Failed to shutdown HTTP server: %v", err)
		}
	}()

	if err := h.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server error: %w", err)
	}

	return nil
}

// handleMCPRequest handles MCP requests over HTTP
func (h *HTTPServer) handleMCPRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Parse MCP request
	var mcpRequest struct {
		Method string                 `json:"method"`
		Params map[string]interface{} `json:"params"`
	}

	if err := json.Unmarshal(body, &mcpRequest); err != nil {
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	// Handle different MCP methods
	var response interface{}
	var httpStatus = http.StatusOK

	switch mcpRequest.Method {
	case "tools/list":
		response = h.handleToolsListMCP()
	case "tools/call":
		response, httpStatus = h.handleToolCallMCP(mcpRequest.Params)
	default:
		response = map[string]string{"error": "Unknown method: " + mcpRequest.Method}
		httpStatus = http.StatusBadRequest
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode MCP response: %v", err)
	}
}

// handleToolsList handles GET /tools endpoint
func (h *HTTPServer) handleToolsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method allowed", http.StatusMethodNotAllowed)
		return
	}

	tools := h.getAvailableTools()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"tools": tools,
	}); err != nil {
		log.Printf("Failed to encode tools response: %v", err)
	}
}

// handleToolsListMCP returns tools list in MCP format
func (h *HTTPServer) handleToolsListMCP() interface{} {
	tools := h.getAvailableTools()
	return map[string]interface{}{
		"tools": tools,
	}
}

// handleToolCallMCP handles tool calls in MCP format
func (h *HTTPServer) handleToolCallMCP(params map[string]interface{}) (interface{}, int) {
	// Extract tool name and arguments
	toolName, ok := params["name"].(string)
	if !ok {
		return map[string]string{"error": "Missing or invalid tool name"}, http.StatusBadRequest
	}

	arguments, ok := params["arguments"].(map[string]interface{})
	if !ok {
		arguments = make(map[string]interface{})
	}

	// Get the underlying MCP server and execute the tool directly
	mcpServer := h.server.GetMCPServer()
	if mcpServer == nil {
		return map[string]string{"error": "MCP server not available"}, http.StatusInternalServerError
	}

	// Execute the API call directly using the same logic as the MCP server
	result, err := h.executeAPICall(toolName, arguments)
	if err != nil {
		return map[string]string{"error": err.Error()}, http.StatusInternalServerError
	}

	return result, http.StatusOK
}

// executeAPICall executes an API call based on tool name and arguments
func (h *HTTPServer) executeAPICall(toolName string, arguments map[string]interface{}) (interface{}, error) {
	config := h.server.GetConfig()
	if config.SwaggerSpec == nil {
		return nil, fmt.Errorf("swagger specification not available")
	}

	// Find the operation for this tool
	method, path, operation := h.findOperationByTool(toolName, config.SwaggerSpec)
	if operation == nil {
		return nil, fmt.Errorf("tool not found: %s", toolName)
	}

	// Build the API request
	url, body, err := h.buildAPIRequest(method, path, operation, arguments, config.APIBaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to build API request: %w", err)
	}

	// Execute the HTTP request
	return h.executeHTTPRequest(method, url, body, config.APIKey)
}

// findOperationByTool finds the swagger operation for a given tool name (applying filters)
func (h *HTTPServer) findOperationByTool(toolName string, swagger *spec.Swagger) (string, string, *spec.Operation) {
	config := h.server.GetConfig()
	filter := config.Filter
	
	for path, pathItem := range swagger.Paths.Paths {
		if pathItem.Get != nil && !h.shouldExcludeOperation("GET", path, pathItem.Get, filter) {
			if h.getToolName("GET", path, pathItem.Get) == toolName {
				return "GET", path, pathItem.Get
			}
		}
		if pathItem.Post != nil && !h.shouldExcludeOperation("POST", path, pathItem.Post, filter) {
			if h.getToolName("POST", path, pathItem.Post) == toolName {
				return "POST", path, pathItem.Post
			}
		}
		if pathItem.Put != nil && !h.shouldExcludeOperation("PUT", path, pathItem.Put, filter) {
			if h.getToolName("PUT", path, pathItem.Put) == toolName {
				return "PUT", path, pathItem.Put
			}
		}
		if pathItem.Delete != nil && !h.shouldExcludeOperation("DELETE", path, pathItem.Delete, filter) {
			if h.getToolName("DELETE", path, pathItem.Delete) == toolName {
				return "DELETE", path, pathItem.Delete
			}
		}
		if pathItem.Patch != nil && !h.shouldExcludeOperation("PATCH", path, pathItem.Patch, filter) {
			if h.getToolName("PATCH", path, pathItem.Patch) == toolName {
				return "PATCH", path, pathItem.Patch
			}
		}
	}
	return "", "", nil
}

// getToolName generates tool name (same logic as in server.go)
func (h *HTTPServer) getToolName(method, path string, op *spec.Operation) string {
	if op.ID != "" {
		toolName := strings.ReplaceAll(op.ID, " ", "_")
		return strings.ToLower(toolName)
	}
	
	// Create tool name from method and path
	toolName := strings.ToLower(method) + "_"
	pathName := strings.ReplaceAll(path, "/", "_")
	pathName = strings.ReplaceAll(pathName, "{", "")
	pathName = strings.ReplaceAll(pathName, "}", "")
	pathName = strings.TrimPrefix(pathName, "_")
	return toolName + pathName
}

// buildAPIRequest builds the HTTP request for the API call
func (h *HTTPServer) buildAPIRequest(method, path string, operation *spec.Operation, arguments map[string]interface{}, baseURL string) (string, io.Reader, error) {
	// Build URL with path parameters
	url := baseURL + path
	
	// Extract body parameter if present
	var bodyData interface{}
	if body, exists := arguments["body"]; exists {
		bodyData = body
		delete(arguments, "body")
	}

	// Replace path parameters
	for key, value := range arguments {
		placeholder := "{" + key + "}"
		if strings.Contains(url, placeholder) {
			url = strings.ReplaceAll(url, placeholder, fmt.Sprintf("%v", value))
			delete(arguments, key) // Remove from arguments since it's in the URL
		}
	}

	// Prepare request body
	var body io.Reader
	if method == "POST" || method == "PUT" || method == "PATCH" {
		// Use body data if available, otherwise use remaining arguments
		var dataToSend interface{}
		if bodyData != nil {
			dataToSend = bodyData
		} else if len(arguments) > 0 {
			dataToSend = arguments
		}
		
		if dataToSend != nil {
			jsonData, err := json.Marshal(dataToSend)
			if err != nil {
				return "", nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			body = bytes.NewReader(jsonData)
		}
	} else {
		// Add remaining arguments as query parameters
		if len(arguments) > 0 {
			queryParams := []string{}
			for key, value := range arguments {
				queryParams = append(queryParams, fmt.Sprintf("%s=%v", key, value))
			}
			if strings.Contains(url, "?") {
				url += "&" + strings.Join(queryParams, "&")
			} else {
				url += "?" + strings.Join(queryParams, "&")
			}
		}
	}

	return url, body, nil
}

// executeHTTPRequest executes the HTTP request and returns the response
func (h *HTTPServer) executeHTTPRequest(method, url string, body io.Reader, apiKey string) (interface{}, error) {
	// Create HTTP request
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	
	// Add API key if configured
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Failed to close response body: %v", err)
		}
	}()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode >= 400 {
		return map[string]interface{}{
			"error":   true,
			"status":  resp.StatusCode,
			"message": string(responseBody),
		}, nil
	}

	// Try to parse JSON response
	var jsonResponse interface{}
	if err := json.Unmarshal(responseBody, &jsonResponse); err == nil {
		return jsonResponse, nil
	}

	// Return as plain text if not JSON
	return map[string]interface{}{
		"content": string(responseBody),
		"type":    "text",
	}, nil
}

// getAvailableTools returns a list of available tools (applying filters)
func (h *HTTPServer) getAvailableTools() []map[string]interface{} {
	config := h.server.GetConfig()
	if config.SwaggerSpec == nil {
		return []map[string]interface{}{}
	}

	tools := []map[string]interface{}{}
	for path, pathItem := range config.SwaggerSpec.Paths.Paths {
		if pathItem.Get != nil && !h.shouldExcludeOperation("GET", path, pathItem.Get, config.Filter) {
			tools = append(tools, h.createToolInfo("GET", path, pathItem.Get))
		}
		if pathItem.Post != nil && !h.shouldExcludeOperation("POST", path, pathItem.Post, config.Filter) {
			tools = append(tools, h.createToolInfo("POST", path, pathItem.Post))
		}
		if pathItem.Put != nil && !h.shouldExcludeOperation("PUT", path, pathItem.Put, config.Filter) {
			tools = append(tools, h.createToolInfo("PUT", path, pathItem.Put))
		}
		if pathItem.Delete != nil && !h.shouldExcludeOperation("DELETE", path, pathItem.Delete, config.Filter) {
			tools = append(tools, h.createToolInfo("DELETE", path, pathItem.Delete))
		}
		if pathItem.Patch != nil && !h.shouldExcludeOperation("PATCH", path, pathItem.Patch, config.Filter) {
			tools = append(tools, h.createToolInfo("PATCH", path, pathItem.Patch))
		}
	}

	return tools
}

// shouldExcludeOperation checks if an operation should be excluded based on filters
func (h *HTTPServer) shouldExcludeOperation(method, path string, operation *spec.Operation, filter *APIFilter) bool {
	if filter == nil {
		return false
	}
	return filter.ShouldExcludeOperation(method, path, operation)
}

// createToolInfo creates tool information from swagger operation
func (h *HTTPServer) createToolInfo(method, path string, op *spec.Operation) map[string]interface{} {
	toolName := h.getToolName(method, path, op)
	
	description := op.Summary
	if description == "" {
		description = op.Description
	}
	if description == "" {
		description = fmt.Sprintf("%s %s", method, path)
	}

	// Build parameter schema
	parameters := []map[string]interface{}{}
	for _, param := range op.Parameters {
		paramInfo := map[string]interface{}{
			"name":        param.Name,
			"in":          param.In,
			"required":    param.Required,
			"description": param.Description,
		}
		
		if param.Type != "" {
			paramInfo["type"] = param.Type
		}
		if param.Format != "" {
			paramInfo["format"] = param.Format
		}
		
		parameters = append(parameters, paramInfo)
	}

	return map[string]interface{}{
		"name":        toolName,
		"description": description,
		"method":      method,
		"path":        path,
		"parameters":  parameters,
		"operationId": op.ID,
	}
}


// RunHTTP runs the server with HTTP transport
func (s *Server) RunHTTP(ctx context.Context, port int) error {
	httpServer := NewHTTPServer(s, port, "", "")
	return httpServer.Start(ctx)
}

// WithHTTPTransport configures the server to use HTTP transport
func (c *Config) WithHTTPTransport(port int, host, path string) *Config {
	c.Transport = &HTTPTransport{
		Port: port,
		Host: host,
		Path: path,
	}
	return c
}