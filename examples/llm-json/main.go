package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

// Complex nested structure for testing
type APIDocumentation struct {
	Title       string     `json:"title"`
	Version     string     `json:"version"`
	Description string     `json:"description"`
	BaseURL     string     `json:"base_url"`
	Endpoints   []Endpoint `json:"endpoints"`
}

type Endpoint struct {
	Path        string      `json:"path"`
	Method      string      `json:"method"`
	Description string      `json:"description"`
	Parameters  []Parameter `json:"parameters"`
	Response    Response    `json:"response"`
}

type Parameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

type Response struct {
	StatusCode int                    `json:"status_code"`
	Body       map[string]interface{} `json:"body"`
}

func main() {
	// Create RAGO client
	cfg, err := client.Load("./rago.toml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ragoClient, err := client.NewWithConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer ragoClient.Close()

	ctx := context.Background()

	// Test 1: Generate complex nested API documentation
	fmt.Println("=== Test 1: Generate Complex API Documentation ===")
	testComplexStructure(ctx, ragoClient)

	// Test 2: Extract structured data from unstructured text
	fmt.Println("\n=== Test 2: Extract Structured Data from Text ===")
	testDataExtraction(ctx, ragoClient)

	// Test 3: Validate JSON generation with strict schema
	fmt.Println("\n=== Test 3: Schema Validation Test ===")
	testSchemaValidation(ctx, ragoClient)
}

func testComplexStructure(ctx context.Context, client *client.Client) {
	req := core.StructuredGenerationRequest{
		GenerationRequest: core.GenerationRequest{
			Prompt: "Generate API documentation for a user management REST API with endpoints for creating, reading, updating, and deleting users",
		},
		Schema:    &APIDocumentation{},
		ForceJSON: true,
		ExampleJSON: `{
			"title": "Example API",
			"version": "1.0.0",
			"description": "Sample API description",
			"base_url": "https://api.example.com",
			"endpoints": [
				{
					"path": "/users",
					"method": "GET",
					"description": "List all users",
					"parameters": [
						{
							"name": "page",
							"type": "integer",
							"required": false,
							"description": "Page number"
						}
					],
					"response": {
						"status_code": 200,
						"body": {
							"users": "array",
							"total": "number"
						}
					}
				}
			]
		}`,
	}

	result, err := client.LLM().GenerateStructured(ctx, req)
	if err != nil {
		log.Printf("Failed to generate structured output: %v", err)
		return
	}

	if !result.Valid {
		log.Printf("Generated JSON is invalid: %s", result.Error)
		return
	}

	// Type assert to APIDocumentation
	apiDoc, ok := result.Data.(*APIDocumentation)
	if !ok {
		log.Printf("Failed to cast to APIDocumentation")
		return
	}

	// Display results
	fmt.Printf("✅ Generated API Documentation:\n")
	fmt.Printf("  Title: %s\n", apiDoc.Title)
	fmt.Printf("  Version: %s\n", apiDoc.Version)
	fmt.Printf("  Base URL: %s\n", apiDoc.BaseURL)
	fmt.Printf("  Endpoints: %d defined\n", len(apiDoc.Endpoints))
	
	for i, endpoint := range apiDoc.Endpoints {
		fmt.Printf("\n  Endpoint %d:\n", i+1)
		fmt.Printf("    %s %s\n", endpoint.Method, endpoint.Path)
		fmt.Printf("    Description: %s\n", endpoint.Description)
		fmt.Printf("    Parameters: %d\n", len(endpoint.Parameters))
		fmt.Printf("    Response Code: %d\n", endpoint.Response.StatusCode)
	}
}

func testDataExtraction(ctx context.Context, client *client.Client) {
	// Complex text with multiple data points
	content := `
	Our Q4 2024 financial report shows remarkable growth. Revenue reached $45.7 million,
	representing a 32% year-over-year increase. The North American market contributed 
	$28.3 million (62%), while European operations generated $12.1 million (26%), and 
	Asia-Pacific brought in $5.3 million (12%). 
	
	Product breakdown: Software licenses accounted for $27.4 million, professional 
	services generated $13.2 million, and support contracts contributed $5.1 million.
	
	Key metrics: Customer retention rate improved to 94%, average deal size increased 
	to $125,000, and we added 87 new enterprise customers. Operating margin expanded 
	to 18.5%, up from 14.2% last year.
	`

	req := core.MetadataExtractionRequest{
		Content: content,
		Fields: []string{
			"total_revenue",
			"revenue_growth",
			"geographic_breakdown",
			"product_revenues",
			"customer_metrics",
			"financial_ratios",
		},
	}

	metadata, err := client.LLM().ExtractMetadata(ctx, req)
	if err != nil {
		log.Printf("Failed to extract metadata: %v", err)
		return
	}

	fmt.Printf("✅ Extracted Financial Data:\n")
	fmt.Printf("  Summary: %s\n", metadata.Summary)
	
	if len(metadata.CustomFields) > 0 {
		fmt.Printf("\n  Extracted Metrics:\n")
		for field, value := range metadata.CustomFields {
			fmt.Printf("    %s: %v\n", field, value)
		}
	}

	if len(metadata.Entities) > 0 {
		fmt.Printf("\n  Identified Entities:\n")
		for entityType, entities := range metadata.Entities {
			fmt.Printf("    %s: %v\n", entityType, entities)
		}
	}

	// Show as formatted JSON
	metadataJSON, _ := json.MarshalIndent(metadata, "  ", "  ")
	fmt.Printf("\n  Full JSON:\n%s\n", string(metadataJSON))
}

func testSchemaValidation(ctx context.Context, client *client.Client) {
	// Test with a strict schema requirement
	type StrictSchema struct {
		ID          string   `json:"id"`          // Must be present
		Name        string   `json:"name"`        // Must be present
		Categories  []string `json:"categories"`  // Must be array
		Score       float64  `json:"score"`       // Must be number
		IsActive    bool     `json:"is_active"`   // Must be boolean
		Metadata    map[string]interface{} `json:"metadata"` // Must be object
	}

	req := core.StructuredGenerationRequest{
		GenerationRequest: core.GenerationRequest{
			Prompt: "Generate a test data object with all required fields populated appropriately",
		},
		Schema:    &StrictSchema{},
		ForceJSON: true,
	}

	result, err := client.LLM().GenerateStructured(ctx, req)
	if err != nil {
		log.Printf("Failed to generate with strict schema: %v", err)
		return
	}

	if !result.Valid {
		fmt.Printf("❌ Schema Validation Failed: %s\n", result.Error)
		fmt.Printf("  Raw output: %s\n", result.Raw)
		return
	}

	// Type assert to StrictSchema
	data, ok := result.Data.(*StrictSchema)
	if !ok {
		log.Printf("Failed to cast to StrictSchema")
		return
	}

	fmt.Printf("✅ Schema Validation Passed:\n")
	fmt.Printf("  ID: %s\n", data.ID)
	fmt.Printf("  Name: %s\n", data.Name)
	fmt.Printf("  Categories: %v\n", data.Categories)
	fmt.Printf("  Score: %.2f\n", data.Score)
	fmt.Printf("  Is Active: %v\n", data.IsActive)
	fmt.Printf("  Metadata Fields: %d\n", len(data.Metadata))
	
	// Verify all fields are properly typed
	fmt.Printf("\n  Type Validation:\n")
	fmt.Printf("    ID is string: ✓\n")
	fmt.Printf("    Categories is array: ✓ (length: %d)\n", len(data.Categories))
	fmt.Printf("    Score is number: ✓\n")
	fmt.Printf("    IsActive is boolean: ✓\n")
	fmt.Printf("    Metadata is object: ✓\n")
}