// Package mcp provides the core functionality for converting Swagger/OpenAPI
// specifications into Model Context Protocol (MCP) tools.
//
// This package handles:
//   - Parsing Swagger/OpenAPI specifications from JSON or YAML
//   - Converting API endpoints to MCP tool definitions
//   - Building parameter schemas from OpenAPI parameters
//   - Executing HTTP requests when MCP tools are invoked
//   - Managing authentication and error handling
//
// # Main Components
//
// SwaggerMCPServer is the main server type that:
//   - Loads and parses Swagger specifications
//   - Registers MCP tools for each API endpoint
//   - Handles tool invocations by making HTTP requests
//
// # Usage Example
//
//	// Load a Swagger specification
//	specData, err := os.ReadFile("api.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Parse the specification
//	swagger, err := ParseSwaggerSpec(specData)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Create MCP server
//	server := NewSwaggerMCPServer("https://api.example.com", swagger, "api-key")
//
//	// Run the server
//	ctx := context.Background()
//	if err := server.Run(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
// # Swagger Parsing
//
// The package supports both JSON and YAML formats for Swagger 2.0 and
// OpenAPI 3.0 specifications:
//
//	// From JSON
//	swagger, err := ParseSwaggerSpec(jsonData)
//
//	// From URL
//	data, err := FetchSwaggerFromURL("https://api.example.com/swagger.json")
//	swagger, err := ParseSwaggerSpec(data)
//
// # Parameter Schema Building
//
// The package automatically converts OpenAPI parameters to JSON Schema
// for MCP tool definitions:
//
//   - Path parameters become required string properties
//   - Query parameters become optional properties with appropriate types
//   - Body parameters preserve their schema structure
//   - Arrays and nested objects are fully supported
//
// # HTTP Request Handling
//
// When an MCP tool is invoked, the package:
//  1. Extracts parameters from the tool invocation
//  2. Builds the appropriate HTTP request
//  3. Handles authentication if configured
//  4. Executes the request and returns the response
//
// # Error Handling
//
// All errors are properly propagated through the MCP protocol:
//   - Parse errors for invalid specifications
//   - Network errors for failed requests
//   - HTTP errors (4xx, 5xx) with status codes and messages
//
// # Testing
//
// The package includes comprehensive unit and integration tests with
// 88%+ code coverage. Run tests with:
//
//	go test -v -cover ./mcp
package mcp