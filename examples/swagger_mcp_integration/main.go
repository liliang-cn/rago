package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	mcpswagger "github.com/liliang-cn/mcp-swagger-server/mcp"
)

func main() {
	// Example 1: Create server from Petstore Swagger spec URL
	fmt.Println("=== Example 1: Petstore API from URL ===")
	if err := runPetstoreExample(); err != nil {
		log.Printf("Petstore example failed: %v", err)
	}

	// Example 2: Create server from local Swagger file
	fmt.Println("\n=== Example 2: Local Swagger File ===")
	if err := runLocalFileExample(); err != nil {
		log.Printf("Local file example failed: %v", err)
	}

	// Example 3: HTTP transport server
	fmt.Println("\n=== Example 3: HTTP Transport Server ===")
	if err := runHTTPServerExample(); err != nil {
		log.Printf("HTTP server example failed: %v", err)
	}

	// Example 4: Using filter to exclude certain operations
	fmt.Println("\n=== Example 4: With API Filter ===")
	if err := runFilterExample(); err != nil {
		log.Printf("Filter example failed: %v", err)
	}
}

func runPetstoreExample() error {
	// Create server from Swagger URL
	server, err := mcpswagger.NewFromSwaggerURL(
		"https://petstore.swagger.io/v2/swagger.json",
		"", // apiBaseURL will be inferred from swagger
		"", // no API key needed for Petstore
	)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Get the server config to display info
	config := server.GetConfig()
	fmt.Printf("Server: %s %s\n", config.Name, config.Version)
	fmt.Printf("API Base URL: %s\n", config.APIBaseURL)
	
	// Note: The actual tool listing would need to be implemented
	// by extending the underlying MCP server to expose its tools
	fmt.Println("Server created successfully from Petstore API")

	return nil
}

func runLocalFileExample() error {
	// Create a sample Swagger spec file
	swaggerSpec := `{
		"swagger": "2.0",
		"info": {
			"title": "Sample API",
			"version": "1.0.0"
		},
		"host": "api.example.com",
		"basePath": "/v1",
		"schemes": ["https"],
		"paths": {
			"/hello": {
				"get": {
					"operationId": "getHello",
					"summary": "Say hello",
					"responses": {
						"200": {
							"description": "Success"
						}
					}
				}
			},
			"/users/{id}": {
				"get": {
					"operationId": "getUser",
					"summary": "Get user by ID",
					"parameters": [
						{
							"name": "id",
							"in": "path",
							"required": true,
							"type": "string"
						}
					],
					"responses": {
						"200": {
							"description": "User found"
						}
					}
				}
			}
		}
	}`

	// Write to temp file
	tmpFile := "/tmp/sample-swagger.json"
	if err := os.WriteFile(tmpFile, []byte(swaggerSpec), 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	defer os.Remove(tmpFile)

	// Create server from file
	server, err := mcpswagger.NewFromSwaggerFile(
		tmpFile,
		"https://api.example.com/v1", // explicit base URL
		"",                            // no API key
	)
	if err != nil {
		return fmt.Errorf("failed to create server from file: %w", err)
	}

	// Get config info
	config := server.GetConfig()
	fmt.Printf("Server created: %s\n", config.Name)
	fmt.Printf("API Base URL: %s\n", config.APIBaseURL)

	return nil
}

func runHTTPServerExample() error {
	// Create server with HTTP transport
	server, err := mcpswagger.NewFromSwaggerURL(
		"https://petstore.swagger.io/v2/swagger.json",
		"",
		"",
	)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Configure HTTP transport
	httpTransport := &mcpswagger.HTTPTransport{
		Port: 8888,
		Host: "localhost",
		Path: "/mcp",
	}
	
	// Update config with HTTP transport
	config := server.GetConfig()
	config.Transport = httpTransport

	fmt.Printf("Starting HTTP server on http://%s:%d\n", httpTransport.Host, httpTransport.Port)
	fmt.Println("Available endpoints:")
	fmt.Println("  - /health - Health check")
	fmt.Println("  - /tools - List available tools")
	fmt.Println("  - /mcp - MCP protocol endpoint")
	fmt.Println()
	fmt.Println("To test: curl http://localhost:8888/tools")
	fmt.Println("Running for 5 seconds...")

	// Start server in background
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		if err := server.RunHTTP(ctx, httpTransport.Port); err != nil {
			log.Printf("Server stopped: %v", err)
		}
	}()

	// Wait for timeout
	<-ctx.Done()
	fmt.Println("\nHTTP server stopped")

	return nil
}

func runFilterExample() error {
	// Create a filter to exclude certain operations
	filter := &mcpswagger.APIFilter{
		ExcludeMethods:      []string{"DELETE"},             // Don't expose DELETE operations
		ExcludePaths:        []string{"/store/order/{orderId}"}, // Exclude specific path
		ExcludeOperationIDs: []string{"uploadFile"},         // Exclude specific operation
		ExcludeTags:         []string{"store"},              // Exclude all store-related operations
	}

	// Create config with filter
	config := mcpswagger.DefaultConfig()
	data, err := mcpswagger.FetchSwaggerFromURL("https://petstore.swagger.io/v2/swagger.json")
	if err != nil {
		return fmt.Errorf("failed to fetch swagger: %w", err)
	}
	
	config.SwaggerData = data
	config.Filter = filter

	// Create server with filtered operations
	server, err := mcpswagger.New(config)
	if err != nil {
		return fmt.Errorf("failed to create server with filter: %w", err)
	}

	fmt.Println("Server created with filtered operations:")
	fmt.Println("  - Excluded DELETE methods")
	fmt.Println("  - Excluded /store/order/{orderId} path")
	fmt.Println("  - Excluded uploadFile operation")
	fmt.Println("  - Excluded all store-related operations")

	_ = server
	return nil
}

// Example of using with authentication
func exampleWithAuth() {
	// Create config with authentication
	config := mcpswagger.DefaultConfig()
	config.APIKey = "YOUR_API_TOKEN"
	config.APIBaseURL = "https://api.example.com"
	
	// You would also need to set swagger data
	// config.SwaggerData = ...
	
	server, err := mcpswagger.New(config)
	if err != nil {
		log.Printf("Failed to create server with auth: %v", err)
		return
	}

	// Use the server...
	_ = server
}