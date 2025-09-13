package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/liliang-cn/rago/v2/client"
)

func main() {
	// Create a new client
	ragClient, err := client.New("")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer ragClient.Close()

	ctx := context.Background()

	// Prepare diverse test data
	fmt.Println("📚 Preparing knowledge base with diverse content...")
	testData := []struct {
		content string
		source  string
		metadata map[string]interface{}
	}{
		{
			content: "Go is a statically typed, compiled language designed at Google. It provides excellent support for concurrent programming through goroutines and channels.",
			source:  "go-intro",
			metadata: map[string]interface{}{
				"category": "programming",
				"language": "go",
				"topic":    "introduction",
				"year":     2009,
			},
		},
		{
			content: "Python is a high-level, interpreted programming language known for its simplicity and readability. It's widely used in data science, web development, and automation.",
			source:  "python-intro",
			metadata: map[string]interface{}{
				"category": "programming",
				"language": "python",
				"topic":    "introduction",
				"year":     1991,
			},
		},
		{
			content: "Machine learning is a subset of artificial intelligence that enables systems to learn and improve from experience without being explicitly programmed.",
			source:  "ml-basics",
			metadata: map[string]interface{}{
				"category": "ai",
				"topic":    "machine-learning",
				"level":    "beginner",
				"year":     2024,
			},
		},
		{
			content: "Docker containers package applications with their dependencies, ensuring consistency across different computing environments.",
			source:  "docker-guide",
			metadata: map[string]interface{}{
				"category": "devops",
				"tool":     "docker",
				"topic":    "containers",
				"year":     2023,
			},
		},
		{
			content: "Kubernetes orchestrates containerized applications, providing automated deployment, scaling, and management of container clusters.",
			source:  "k8s-overview",
			metadata: map[string]interface{}{
				"category": "devops",
				"tool":     "kubernetes",
				"topic":    "orchestration",
				"year":     2023,
			},
		},
	}

	// Ingest all test data
	for _, data := range testData {
		err = ragClient.IngestTextWithMetadata(data.content, data.source, data.metadata)
		if err != nil {
			log.Printf("Failed to ingest %s: %v", data.source, err)
		}
	}
	fmt.Println("✅ Knowledge base prepared")

	// Example 1: Basic search
	fmt.Println("\n🔍 Basic Search")
	fmt.Println("===============")
	searchOpts := client.DefaultSearchOptions()
	searchOpts.TopK = 3
	
	results, err := ragClient.Search(ctx, "programming languages", searchOpts)
	if err != nil {
		log.Printf("Search failed: %v", err)
	} else {
		fmt.Printf("Found %d results:\n", len(results))
		for i, result := range results {
			fmt.Printf("[%d] Score: %.3f\n", i+1, result.Score)
			fmt.Printf("    Content: %s\n", result.Content[:min(100, len(result.Content))])
			fmt.Printf("    Source: %s\n", result.Source)
		}
	}

	// Example 2: Search with context
	fmt.Println("\n📝 Search with Context")
	fmt.Println("=====================")
	contextStr, contextResults, err := ragClient.SearchWithContext(
		ctx,
		"container orchestration",
		&client.SearchOptions{
			TopK:            2,
			IncludeMetadata: true,
			HybridSearch:    true,
			VectorWeight:    0.8,
		},
	)
	if err != nil {
		log.Printf("Search with context failed: %v", err)
	} else {
		fmt.Println("Generated Context:")
		fmt.Println(contextStr)
		fmt.Printf("\nFound %d results with metadata:\n", len(contextResults))
		for _, result := range contextResults {
			if result.Metadata != nil {
				fmt.Printf("  - Category: %v, Tool: %v\n", 
					result.Metadata["category"], 
					result.Metadata["tool"])
			}
		}
	}

	// Example 3: Similarity search with score threshold
	fmt.Println("\n🎯 Similarity Search with Score Threshold")
	fmt.Println("=========================================")
	similarityOpts := &client.SearchOptions{
		TopK:           5,
		ScoreThreshold: 0.7, // Only return results with score > 0.7
		HybridSearch:   false, // Pure vector search
		IncludeMetadata: true,
	}
	
	highScoreResults, err := ragClient.Search(ctx, "artificial intelligence machine learning", similarityOpts)
	if err != nil {
		log.Printf("Similarity search failed: %v", err)
	} else {
		fmt.Printf("High-score results (>0.7):\n")
		for _, result := range highScoreResults {
			fmt.Printf("  Score: %.3f - %s\n", result.Score, result.Source)
		}
	}

	// Example 4: Hybrid search (combining vector and keyword search)
	fmt.Println("\n🔄 Hybrid Search (Vector + Keyword)")
	fmt.Println("====================================")
	hybridOpts := &client.SearchOptions{
		TopK:         3,
		HybridSearch: true,
		VectorWeight: 0.5, // 50% vector, 50% keyword
	}
	
	hybridResults, err := ragClient.Search(ctx, "Google programming concurrent", hybridOpts)
	if err != nil {
		log.Printf("Hybrid search failed: %v", err)
	} else {
		fmt.Printf("Hybrid search results:\n")
		for i, result := range hybridResults {
			fmt.Printf("[%d] Score: %.3f - Source: %s\n", i+1, result.Score, result.Source)
			if result.Metadata != nil {
				fmt.Printf("    Language: %v, Year: %v\n", 
					result.Metadata["language"], 
					result.Metadata["year"])
			}
		}
	}

	// Example 5: Query with filtered search
	fmt.Println("\n🔍 Query with Filtered Search")
	fmt.Println("=============================")
	
	// Note: Filter functionality would need to be implemented in the RAG layer
	// This demonstrates the pattern for filtered queries
	queryResponse, err := ragClient.QueryWithFilters(
		"Tell me about container technologies",
		map[string]interface{}{
			"category": "devops",
		},
	)
	if err != nil {
		log.Printf("Filtered query failed: %v", err)
	} else {
		fmt.Printf("Answer: %s\n", queryResponse.Answer)
		if len(queryResponse.Sources) > 0 {
			fmt.Println("Sources used:")
			for _, source := range queryResponse.Sources {
				fmt.Printf("  - %s (Score: %.3f)\n", source.DocumentID, source.Score)
			}
		}
	}

	// Example 6: Time-based search patterns
	fmt.Println("\n⏰ Recent Content Search")
	fmt.Println("========================")
	
	// Search for recent content (2023-2024)
	recentOpts := &client.SearchOptions{
		TopK:            5,
		IncludeMetadata: true,
	}
	
	recentResults, err := ragClient.Search(ctx, "latest technologies 2023 2024", recentOpts)
	if err != nil {
		log.Printf("Recent search failed: %v", err)
	} else {
		fmt.Println("Recent content found:")
		for _, result := range recentResults {
			if result.Metadata != nil && result.Metadata["year"] != nil {
				year := result.Metadata["year"]
				if y, ok := year.(float64); ok && y >= 2023 {
					fmt.Printf("  [%v] %s - Score: %.3f\n", 
						year, result.Source, result.Score)
				}
			}
		}
	}

	// Example 7: Performance comparison
	fmt.Println("\n⚡ Search Performance Comparison")
	fmt.Println("================================")
	
	queries := []string{
		"programming",
		"containers and orchestration",
		"machine learning AI",
	}
	
	for _, query := range queries {
		// Vector search
		start := time.Now()
		_, err := ragClient.Search(ctx, query, &client.SearchOptions{
			TopK:         3,
			HybridSearch: false,
		})
		vectorTime := time.Since(start)
		
		// Hybrid search
		start = time.Now()
		_, err = ragClient.Search(ctx, query, &client.SearchOptions{
			TopK:         3,
			HybridSearch: true,
		})
		hybridTime := time.Since(start)
		
		if err == nil {
			fmt.Printf("Query: '%s'\n", query)
			fmt.Printf("  Vector search: %v\n", vectorTime)
			fmt.Printf("  Hybrid search: %v\n", hybridTime)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}