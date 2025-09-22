// Example: Complete Platform Integration
// This example demonstrates using all RAGO features together

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/liliang-cn/rago/v2/client"
)

func main() {
	// Initialize client
	configPath := filepath.Join(os.Getenv("HOME"), ".rago", "rago.toml")
	c, err := client.New(configPath)
	if err != nil {
		log.Fatalf("Failed to initialize client: %v", err)
	}
	defer c.Close()

	ctx := context.Background()

	fmt.Println("=== RAGO Complete Platform Demo ===")
	fmt.Println("This example demonstrates all major RAGO features working together\n")

	// Step 1: Check system status
	fmt.Println("Step 1: System Status Check")
	fmt.Println("---------------------------")
	
	cfg := c.GetConfig()
	fmt.Printf("✓ Configuration loaded\n")
	fmt.Printf("  Default LLM: %s\n", cfg.Providers.DefaultLLM)
	fmt.Printf("  Default Embedder: %s\n", cfg.Providers.DefaultEmbedder)
	
	// Check available components
	components := []struct {
		name      string
		available bool
	}{
		{"LLM", c.LLM != nil},
		{"RAG", c.RAG != nil},
		{"Tools", c.Tools != nil},
		{"Agent", c.Agent != nil},
		{"MCP", c.GetMCPClient() != nil},
	}
	
	fmt.Println("\nComponents:")
	for _, comp := range components {
		status := "✗"
		if comp.available {
			status = "✓"
		}
		fmt.Printf("  %s %s\n", status, comp.name)
	}

	// Step 2: Create knowledge base
	fmt.Println("\nStep 2: Building Knowledge Base")
	fmt.Println("--------------------------------")
	
	// Create sample documents
	tempDir, err := ioutil.TempDir("", "rago-demo")
	if err != nil {
		log.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Document 1: Project overview
	doc1 := filepath.Join(tempDir, "project_overview.txt")
	content1 := `Project: E-Commerce Platform
Architecture: Microservices with Go
Database: PostgreSQL with Redis caching
Message Queue: RabbitMQ
API: REST with GraphQL gateway
Authentication: JWT with refresh tokens
Deployment: Kubernetes on AWS`
	
	if err := ioutil.WriteFile(doc1, []byte(content1), 0644); err != nil {
		log.Fatalf("Failed to write doc1: %v", err)
	}
	
	// Document 2: API documentation
	doc2 := filepath.Join(tempDir, "api_docs.txt")
	content2 := `API Endpoints:
- POST /api/auth/login - User authentication
- GET /api/products - List products with pagination
- POST /api/orders - Create new order
- GET /api/orders/{id} - Get order details
- PUT /api/users/profile - Update user profile`
	
	if err := ioutil.WriteFile(doc2, []byte(content2), 0644); err != nil {
		log.Fatalf("Failed to write doc2: %v", err)
	}
	
	// Ingest documents
	if c.RAG != nil {
		req1 := client.IngestRequest{
			Path:      doc1,
			ChunkSize: 100,
			Overlap:   20,
		}
		
		if _, err := c.Ingest(ctx, req1); err == nil {
			fmt.Println("✓ Ingested project overview")
		}
		
		req2 := client.IngestRequest{
			Path:      doc2,
			ChunkSize: 100,
			Overlap:   20,
		}
		
		if _, err := c.Ingest(ctx, req2); err == nil {
			fmt.Println("✓ Ingested API documentation")
		}
	}

	// Step 3: LLM-powered code generation
	fmt.Println("\nStep 3: Code Generation")
	fmt.Println("-----------------------")
	
	if c.LLM != nil {
		prompt := "Generate a Go struct for a Product with fields: ID, Name, Price, Stock"
		
		opts := &client.GenerateOptions{
			Temperature: 0.3,
			MaxTokens:   200,
		}
		
		response, err := c.LLM.GenerateWithOptions(ctx, prompt, opts)
		if err != nil {
			log.Printf("Generation error: %v\n", err)
		} else {
			fmt.Println("Generated code:")
			fmt.Println(response)
		}
	}

	// Step 4: RAG-based Q&A
	fmt.Println("\nStep 4: Knowledge-Based Q&A")
	fmt.Println("---------------------------")
	
	if c.RAG != nil {
		questions := []string{
			"What database does the project use?",
			"What authentication method is used?",
			"List the API endpoints",
		}
		
		for _, q := range questions {
			fmt.Printf("\nQ: %s\n", q)
			
			queryReq := client.QueryRequest{
				Query:       q,
				TopK:        3,
				ShowSources: true,
			}
			
			resp, err := c.Query(ctx, queryReq)
			if err != nil {
				log.Printf("Query error: %v\n", err)
			} else {
				fmt.Printf("A: %s\n", resp.Answer)
				if len(resp.Sources) > 0 {
					fmt.Printf("   (Based on %d sources)\n", len(resp.Sources))
				}
			}
		}
	}

	// Step 5: Tool integration
	fmt.Println("\nStep 5: Tool Integration")
	fmt.Println("------------------------")
	
	if c.Tools != nil {
		tools, err := c.Tools.List()
		if err == nil && len(tools) > 0 {
			fmt.Printf("✓ Found %d available tools\n", len(tools))
			
			// Show first 3 tools
			max := 3
			if len(tools) < max {
				max = len(tools)
			}
			
			fmt.Println("Sample tools:")
			for i := 0; i < max; i++ {
				fmt.Printf("  - %s: %s\n", tools[i].Name, tools[i].Description)
			}
		} else {
			fmt.Println("No tools available (configure MCP servers to enable)")
		}
	}

	// Step 6: Agent automation
	fmt.Println("\nStep 6: Agent Automation")
	fmt.Println("------------------------")
	
	if c.Agent != nil {
		// Create a complex task
		task := "Based on the project architecture in our knowledge base, suggest 3 improvements"
		
		opts := &client.AgentOptions{
			Verbose: true,
			Timeout: 30,
		}
		
		fmt.Printf("Task: %s\n", task)
		fmt.Println("Processing...")
		
		result, err := c.Agent.RunWithOptions(ctx, task, opts)
		if err != nil {
			log.Printf("Agent error: %v\n", err)
		} else {
			fmt.Printf("Agent response:\n")
			fmt.Printf("  Success: %v\n", result.Success)
			if result.Output != nil {
				fmt.Printf("  Output: %v\n", result.Output)
			}
		}
	}

	// Step 7: Interactive chat with context
	fmt.Println("\nStep 7: Contextual Chat")
	fmt.Println("-----------------------")
	
	if c.LLM != nil {
		history := client.NewConversationHistory("You are a helpful technical assistant with knowledge about our e-commerce platform.", 10)
		
		// First message
		response, err := c.ChatWithHistory(ctx, "What's our tech stack?", history, nil)
		if err == nil {
			fmt.Printf("User: What's our tech stack?\n")
			fmt.Printf("Assistant: %s\n", response)
		}
		
		// Follow-up
		response, err = c.ChatWithHistory(ctx, "What would you recommend for monitoring?", history, nil)
		if err == nil {
			fmt.Printf("\nUser: What would you recommend for monitoring?\n")
			fmt.Printf("Assistant: %s\n", response)
		}
	}

	// Step 8: Performance metrics
	fmt.Println("\nStep 8: Performance Metrics")
	fmt.Println("---------------------------")
	
	// Simulate some operations and measure time
	operations := []struct {
		name string
		fn   func() error
	}{
		{
			"LLM Generation",
			func() error {
				if c.LLM != nil {
					_, err := c.LLM.Generate("Test")
					return err
				}
				return nil
			},
		},
		{
			"RAG Query",
			func() error {
				if c.RAG != nil {
					_, err := c.RAG.Query("Test query")
					return err
				}
				return nil
			},
		},
		{
			"Search",
			func() error {
				_, err := c.Search(ctx, "test", nil)
				return err
			},
		},
	}
	
	for _, op := range operations {
		start := time.Now()
		err := op.fn()
		elapsed := time.Since(start)
		
		status := "✓"
		if err != nil {
			status = "✗"
		}
		fmt.Printf("%s %s: %v\n", status, op.name, elapsed)
	}

	// Step 9: Advanced search
	fmt.Println("\nStep 9: Semantic Search")
	fmt.Println("-----------------------")
	
	searchQueries := []string{
		"authentication security",
		"database performance",
		"API endpoints",
	}
	
	for _, query := range searchQueries {
		opts := client.DefaultSearchOptions()
		opts.TopK = 2
		
		results, err := c.Search(ctx, query, opts)
		if err != nil {
			log.Printf("Search error: %v\n", err)
		} else {
			fmt.Printf("\nSearch: '%s'\n", query)
			fmt.Printf("Found %d results\n", len(results))
			for i, r := range results {
				preview := r.Content
				if len(preview) > 50 {
					preview = preview[:50] + "..."
				}
				fmt.Printf("  [%d] Score %.2f: %s\n", i+1, r.Score, preview)
			}
		}
	}

	// Summary
	fmt.Println("\n=== Platform Integration Complete ===")
	fmt.Println("\nThis example demonstrated:")
	fmt.Println("✓ Client initialization and configuration")
	fmt.Println("✓ Document ingestion and RAG setup")
	fmt.Println("✓ LLM-powered code generation")
	fmt.Println("✓ Knowledge-based Q&A")
	fmt.Println("✓ Tool integration (when available)")
	fmt.Println("✓ Agent automation")
	fmt.Println("✓ Contextual conversations")
	fmt.Println("✓ Performance monitoring")
	fmt.Println("✓ Semantic search")
	fmt.Println("\nRAGO provides a complete AI development platform for Go!")
}