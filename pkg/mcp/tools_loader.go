// Package mcp implements tool loading for LLM tool calling.
// These are tool definitions passed to the LLM for function calling.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
	
	"github.com/liliang-cn/rago/v2/pkg/core"
)

// LoadToolsFromMCP loads MCP tool definitions by connecting to MCP servers.
func LoadToolsFromMCP(serversPath string) ([]core.ToolInfo, error) {
	// Read the mcpServers.json file
	data, err := os.ReadFile(serversPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read servers file: %w", err)
	}
	
	// Parse the MCP servers JSON format
	var mcpFile struct {
		MCPServers map[string]struct {
			Command     string            `json:"command"`
			Args        []string          `json:"args"`
			Description string            `json:"description"`
			Env         map[string]string `json:"env"`
		} `json:"mcpServers"`
	}
	
	if err := json.Unmarshal(data, &mcpFile); err != nil {
		return nil, fmt.Errorf("failed to parse servers JSON: %w", err)
	}
	
	// Collect tools from all servers concurrently
	var (
		allTools []core.ToolInfo
		mu       sync.Mutex
		wg       sync.WaitGroup
	)
	
	for serverName, serverDef := range mcpFile.MCPServers {
		wg.Add(1)
		go func(name, command string, args []string, description string, env map[string]string) {
			defer wg.Done()
			
			serverConfig := &ServerConfig{
				Name:        name,
				Command:     []string{command},
				Args:        args,
				Description: description,
				Env:         env,
			}
			
			// Try to connect and get tools
			tools, err := getToolsFromServer(serverConfig)
			if err != nil {
				// Log the error but don't fail - server might not be running
				log.Printf("[MCP] Could not get tools from %s: %v\n", name, err)
				return
			}
			
			mu.Lock()
			allTools = append(allTools, tools...)
			mu.Unlock()
			
			log.Printf("[MCP] Loaded %d tools from %s\n", len(tools), name)
		}(serverName, serverDef.Command, serverDef.Args, serverDef.Description, serverDef.Env)
	}
	
	wg.Wait()
	
	return allTools, nil
}

// getToolsFromServer connects to an MCP server and retrieves its tools
func getToolsFromServer(config *ServerConfig) ([]core.ToolInfo, error) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Create MCP client
	client, err := NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()
	
	// Connect to server
	if err := client.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	
	// Get tools from server
	tools := client.GetTools()
	var toolInfos []core.ToolInfo
	
	for _, tool := range tools {
		toolInfo := core.ToolInfo{
			Name:        tool.Name,
			ServerName:  config.Name,
			Description: tool.Description,
			Category:    "mcp", // Default category
		}
		
		// Convert input schema if available
		if tool.InputSchema != nil {
			// Marshal the schema to JSON then unmarshal to map
			if data, err := json.Marshal(tool.InputSchema); err == nil {
				var schemaMap map[string]interface{}
				if json.Unmarshal(data, &schemaMap) == nil {
					// Ensure the schema has proper structure for OpenAI function calling
					toolInfo.InputSchema = ensureProperFunctionSchema(schemaMap)
				}
			}
		}
		
		// If no input schema, create a minimal one
		if toolInfo.InputSchema == nil {
			toolInfo.InputSchema = map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{},
				"additionalProperties": true,
			}
		}
		
		toolInfos = append(toolInfos, toolInfo)
	}
	
	return toolInfos, nil
}

// ensureProperFunctionSchema ensures the schema has the required structure for OpenAI function calling
func ensureProperFunctionSchema(schema map[string]interface{}) map[string]interface{} {
	// Make a copy to avoid modifying the original
	result := make(map[string]interface{})
	for k, v := range schema {
		result[k] = v
	}
	
	// Ensure it has a type
	if _, ok := result["type"]; !ok {
		result["type"] = "object"
	}
	
	// Ensure it has properties field (required by OpenAI API)
	if _, ok := result["properties"]; !ok {
		result["properties"] = map[string]interface{}{}
	}
	
	// Ensure properties is a map
	if props, ok := result["properties"].(map[string]interface{}); !ok {
		result["properties"] = map[string]interface{}{}
	} else {
		// Validate each property has proper structure
		for _, propDef := range props {
			if propMap, ok := propDef.(map[string]interface{}); ok {
				// Ensure each property has a type
				if _, hasType := propMap["type"]; !hasType {
					propMap["type"] = "string" // Default type
				}
			}
		}
	}
	
	// Add additionalProperties if not present
	if _, ok := result["additionalProperties"]; !ok {
		result["additionalProperties"] = false
	}
	
	return result
}