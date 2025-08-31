package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"

	"github.com/liliang-cn/rago/pkg/config"
	"github.com/liliang-cn/rago/pkg/mcp"
)

func main() {
	// Load config
	cfg, err := config.Load("../../config.toml")
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Create MCP tool manager
	mcpManager := mcp.NewMCPToolManager(&cfg.MCP)

	ctx := context.Background()
	err = mcpManager.Start(ctx)
	if err != nil {
		log.Fatal("Failed to start MCP:", err)
	}
	defer mcpManager.Close()

	// Get tools
	tools := mcpManager.ListTools()

	// Find an execute tool to examine its schema
	for name, tool := range tools {
		if name == "mcp_sqlite_execute" {
			fmt.Printf("=== Tool: %s ===\n", name)
			fmt.Printf("Description: %s\n", tool.Description())

			// Use reflection to examine the tool structure
			toolValue := reflect.ValueOf(tool).Elem()
			toolField := toolValue.FieldByName("tool")
			if toolField.IsValid() {
				mcpTool := toolField.Interface()
				fmt.Printf("Tool type: %T\n", mcpTool)

				mcpToolValue := reflect.ValueOf(mcpTool).Elem()
				inputSchemaField := mcpToolValue.FieldByName("InputSchema")
				if inputSchemaField.IsValid() && !inputSchemaField.IsNil() {
					schema := inputSchemaField.Interface()
					fmt.Printf("InputSchema type: %T\n", schema)

					// Try to marshal to JSON to see the structure
					if schemaBytes, err := json.MarshalIndent(schema, "", "  "); err == nil {
						fmt.Printf("InputSchema JSON:\n%s\n", string(schemaBytes))
					} else {
						fmt.Printf("Failed to marshal schema: %v\n", err)
					}

					// Use reflection to explore the schema structure
					schemaValue := reflect.ValueOf(schema)
					if schemaValue.Kind() == reflect.Ptr {
						schemaValue = schemaValue.Elem()
					}

					fmt.Printf("Schema struct fields:\n")
					schemaType := schemaValue.Type()
					for i := 0; i < schemaValue.NumField(); i++ {
						field := schemaType.Field(i)
						value := schemaValue.Field(i)
						fmt.Printf("  %s: %s = %v\n", field.Name, field.Type, value.Interface())
					}
				}
			}
			break
		}
	}
}
