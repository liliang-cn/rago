package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const baseURL = "http://localhost:7127/api/mcp"

// MCPClient is a simple HTTP client for MCP API
type MCPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewMCPClient creates a new MCP HTTP client
func NewMCPClient(baseURL string) *MCPClient {
	return &MCPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ListTools lists all available MCP tools
func (c *MCPClient) ListTools() error {
	resp, err := c.httpClient.Get(c.baseURL + "/tools")
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Println("Available MCP Tools:")
	fmt.Println(string(body))
	return nil
}

// GetTool gets details of a specific tool
func (c *MCPClient) GetTool(toolName string) error {
	resp, err := c.httpClient.Get(c.baseURL + "/tools/" + toolName)
	if err != nil {
		return fmt.Errorf("failed to get tool: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Printf("Tool Details for '%s':\n", toolName)
	fmt.Println(string(body))
	return nil
}

// CallTool calls an MCP tool with arguments
func (c *MCPClient) CallTool(toolName string, args map[string]interface{}) error {
	reqBody := map[string]interface{}{
		"tool_name": toolName,
		"args":      args,
		"timeout":   30,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/tools/call",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to call tool: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Printf("Tool Call Result for '%s':\n", toolName)
	fmt.Println(string(body))
	return nil
}

// BatchCallTools calls multiple tools in parallel
func (c *MCPClient) BatchCallTools(calls []map[string]interface{}) error {
	reqBody := map[string]interface{}{
		"calls":   calls,
		"timeout": 60,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/tools/batch",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to batch call tools: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Println("Batch Call Results:")
	fmt.Println(string(body))
	return nil
}

// GetServerStatus gets the status of all MCP servers
func (c *MCPClient) GetServerStatus() error {
	resp, err := c.httpClient.Get(c.baseURL + "/servers")
	if err != nil {
		return fmt.Errorf("failed to get server status: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Println("MCP Server Status:")
	fmt.Println(string(body))
	return nil
}

// StartServer starts a specific MCP server
func (c *MCPClient) StartServer(serverName string) error {
	reqBody := map[string]interface{}{
		"server_name": serverName,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/servers/start",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Printf("Start Server '%s' Result:\n", serverName)
	fmt.Println(string(body))
	return nil
}

// StopServer stops a specific MCP server
func (c *MCPClient) StopServer(serverName string) error {
	reqBody := map[string]interface{}{
		"server_name": serverName,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/servers/stop",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to stop server: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Printf("Stop Server '%s' Result:\n", serverName)
	fmt.Println(string(body))
	return nil
}

// GetLLMTools gets tools formatted for LLM integration
func (c *MCPClient) GetLLMTools() error {
	resp, err := c.httpClient.Get(c.baseURL + "/llm/tools")
	if err != nil {
		return fmt.Errorf("failed to get LLM tools: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Println("Tools for LLM Integration:")
	fmt.Println(string(body))
	return nil
}

func main() {
	// Create MCP client
	client := NewMCPClient(baseURL)

	fmt.Println("=== MCP HTTP API Client Example ===\n")

	// 1. Get server status
	fmt.Println("1. Getting server status...")
	if err := client.GetServerStatus(); err != nil {
		log.Printf("Error: %v\n", err)
	}
	fmt.Println()

	// 2. List all available tools
	fmt.Println("2. Listing all available tools...")
	if err := client.ListTools(); err != nil {
		log.Printf("Error: %v\n", err)
	}
	fmt.Println()

	// 3. Get details of a specific tool (example: if filesystem tool exists)
	fmt.Println("3. Getting tool details...")
	if err := client.GetTool("mcp_filesystem_readFile"); err != nil {
		log.Printf("Error: %v\n", err)
	}
	fmt.Println()

	// 4. Call a tool (example: read a file)
	fmt.Println("4. Calling a tool...")
	if err := client.CallTool("mcp_filesystem_readFile", map[string]interface{}{
		"path": "/tmp/test.txt",
	}); err != nil {
		log.Printf("Error: %v\n", err)
	}
	fmt.Println()

	// 5. Batch call multiple tools
	fmt.Println("5. Batch calling tools...")
	calls := []map[string]interface{}{
		{
			"tool_name": "mcp_filesystem_listDirectory",
			"args": map[string]interface{}{
				"path": "/tmp",
			},
		},
		{
			"tool_name": "mcp_filesystem_getFileInfo",
			"args": map[string]interface{}{
				"path": "/tmp/test.txt",
			},
		},
	}
	if err := client.BatchCallTools(calls); err != nil {
		log.Printf("Error: %v\n", err)
	}
	fmt.Println()

	// 6. Get tools formatted for LLM
	fmt.Println("6. Getting tools for LLM integration...")
	if err := client.GetLLMTools(); err != nil {
		log.Printf("Error: %v\n", err)
	}
	fmt.Println()

	// 7. Start/stop server examples (commented out to avoid actual operations)
	// fmt.Println("7. Starting a server...")
	// if err := client.StartServer("filesystem"); err != nil {
	//     log.Printf("Error: %v\n", err)
	// }
	// 
	// fmt.Println("8. Stopping a server...")
	// if err := client.StopServer("filesystem"); err != nil {
	//     log.Printf("Error: %v\n", err)
	// }

	fmt.Println("=== Example completed ===")
}