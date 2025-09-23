// Example: RAG Operations
// This example demonstrates Retrieval-Augmented Generation operations

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

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

	// Example 1: Create sample documents
	fmt.Println("=== Example 1: Preparing Sample Documents ===")

	// Create a temporary directory for test documents
	tempDir, err := ioutil.TempDir("", "rago-example")
	if err != nil {
		log.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create sample documents
	doc1 := filepath.Join(tempDir, "doc1.txt")
	content1 := `RAGO Platform Overview
	
RAGO is a comprehensive AI development platform for Go developers. 
It provides unified access to Language Models (LLM), Retrieval-Augmented Generation (RAG),
external tools via MCP protocol, and autonomous task execution through agents.

Key Features:
- Multi-provider LLM support (Ollama, OpenAI, etc.)
- Vector-based document search with semantic understanding
- Tool integration through Model Context Protocol
- Intelligent agent automation for complex tasks`

	if err := ioutil.WriteFile(doc1, []byte(content1), 0644); err != nil {
		log.Fatalf("Failed to write doc1: %v", err)
	}

	doc2 := filepath.Join(tempDir, "doc2.txt")
	content2 := `Getting Started with RAGO

Installation:
1. Install Go 1.21 or later
2. Run: go get github.com/liliang-cn/rago/v2
3. Initialize configuration: rago init

Basic Usage:
- Create a client with client.New()
- Use client.LLM for language model operations
- Use client.RAG for document search and Q&A
- Use client.Tools for external tool integration
- Use client.Agent for task automation`

	if err := ioutil.WriteFile(doc2, []byte(content2), 0644); err != nil {
		log.Fatalf("Failed to write doc2: %v", err)
	}

	// Example 2: Ingest documents
	fmt.Println("\n=== Example 2: Document Ingestion ===")
	if c.RAG != nil {
		// Ingest first document
		req1 := client.IngestRequest{
			Path:      doc1,
			ChunkSize: 200,
			Overlap:   50,
			Metadata: map[string]string{
				"type":   "overview",
				"source": "doc1.txt",
			},
		}

		resp1, err := c.Ingest(ctx, req1)
		if err != nil {
			log.Printf("Failed to ingest doc1: %v\n", err)
		} else {
			fmt.Printf("✓ Ingested doc1: %v\n", resp1)
		}

		// Ingest second document
		req2 := client.IngestRequest{
			Path:      doc2,
			ChunkSize: 200,
			Overlap:   50,
			Metadata: map[string]string{
				"type":   "tutorial",
				"source": "doc2.txt",
			},
		}

		resp2, err := c.Ingest(ctx, req2)
		if err != nil {
			log.Printf("Failed to ingest doc2: %v\n", err)
		} else {
			fmt.Printf("✓ Ingested doc2: %v\n", resp2)
		}

		// Simple text ingestion
		err = c.RAG.Ingest("RAGO supports multiple embedding models including OpenAI and Ollama.")
		if err != nil {
			log.Printf("Failed to ingest text: %v\n", err)
		} else {
			fmt.Println("✓ Ingested additional text")
		}
	} else {
		fmt.Println("RAG not initialized")
	}

	// Example 3: Query the knowledge base
	fmt.Println("\n=== Example 3: RAG Query ===")
	if c.RAG != nil {
		queries := []string{
			"What is RAGO?",
			"How do I install RAGO?",
			"What are the key features?",
		}

		for _, query := range queries {
			fmt.Printf("\nQ: %s\n", query)

			// Simple query
			answer, err := c.RAG.Query(query)
			if err != nil {
				log.Printf("Query error: %v\n", err)
			} else {
				fmt.Printf("A: %s\n", answer)
			}
		}
	}

	// Example 4: Query with options and sources
	fmt.Println("\n=== Example 4: Query with Sources ===")
	queryReq := client.QueryRequest{
		Query:       "What tools does RAGO support?",
		TopK:        3,
		Temperature: 0.7,
		MaxTokens:   200,
		ShowSources: true,
	}

	queryResp, err := c.Query(ctx, queryReq)
	if err != nil {
		log.Printf("Query error: %v\n", err)
	} else {
		fmt.Printf("Q: %s\n", queryReq.Query)
		fmt.Printf("A: %s\n\n", queryResp.Answer)

		if len(queryResp.Sources) > 0 {
			fmt.Println("Sources:")
			for i, source := range queryResp.Sources {
				fmt.Printf("  [%d] Score: %.2f\n", i+1, source.Score)
				fmt.Printf("      Content: %.100s...\n", source.Content)
				if source.Source != "" {
					fmt.Printf("      Source: %s\n", source.Source)
				}
			}
		}
	}

	// Example 5: Semantic Search
	fmt.Println("\n=== Example 5: Semantic Search ===")
	searchOpts := client.DefaultSearchOptions()
	searchOpts.TopK = 5

	results, err := c.Search(ctx, "installation steps", searchOpts)
	if err != nil {
		log.Printf("Search error: %v\n", err)
	} else {
		fmt.Printf("Found %d results for 'installation steps':\n", len(results))
		for i, result := range results {
			fmt.Printf("  [%d] Score: %.2f\n", i+1, result.Score)
			fmt.Printf("      Content: %.100s...\n", result.Content)
		}
	}

	// Example 6: RAG with specific options
	fmt.Println("\n=== Example 6: RAG with Custom Options ===")
	if c.RAG != nil {
		queryOpts := &client.QueryOptions{
			TopK:        5,
			Temperature: 0.5,
			MaxTokens:   300,
			ShowSources: true,
			Filters: map[string]string{
				"type": "tutorial", // Filter by metadata
			},
		}

		resp, err := c.RAG.QueryWithOptions(ctx, "How to use RAGO?", queryOpts)
		if err != nil {
			log.Printf("Query error: %v\n", err)
		} else {
			fmt.Printf("Answer: %s\n", resp.Answer)
			fmt.Printf("Sources found: %d\n", len(resp.Sources))
		}
	}

	// Example 7: Search with context (generates contextual answer)
	fmt.Println("\n=== Example 7: Search with Context ===")
	answer, sources, err := c.SearchWithContext(ctx, "What protocols does RAGO use?", nil)
	if err != nil {
		log.Printf("Search with context error: %v\n", err)
	} else {
		fmt.Printf("Contextual Answer: %s\n", answer)
		fmt.Printf("Based on %d sources\n", len(sources))
	}

	fmt.Println("\n=== RAG Operations Complete ===")
}
