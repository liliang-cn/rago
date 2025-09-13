package main

import (
	"fmt"
	"log"
	"os"

	"github.com/liliang-cn/rago/v2/client"
)

func main() {
	// Create a new client with default configuration
	ragClient, err := client.New("")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer ragClient.Close()

	// Example 1: Ingest a text document
	fmt.Println("📝 Ingesting text content...")
	err = ragClient.IngestText(
		"The Go programming language is an open source project to make programmers more productive. "+
			"Go is expressive, concise, clean, and efficient. Its concurrency mechanisms make it easy to "+
			"write programs that get the most out of multicore and networked machines.",
		"go-intro",
	)
	if err != nil {
		log.Fatalf("Failed to ingest text: %v", err)
	}
	fmt.Println("✅ Text ingested successfully")

	// Example 2: Ingest a file (if provided as argument)
	if len(os.Args) > 1 {
		filePath := os.Args[1]
		fmt.Printf("📁 Ingesting file: %s\n", filePath)
		err = ragClient.IngestFile(filePath)
		if err != nil {
			log.Printf("Failed to ingest file: %v", err)
		} else {
			fmt.Println("✅ File ingested successfully")
		}
	}

	// Example 3: Perform a simple query
	fmt.Println("\n🔍 Querying: What is Go programming language?")
	response, err := ragClient.Query("What is Go programming language?")
	if err != nil {
		log.Fatalf("Failed to query: %v", err)
	}
	fmt.Printf("💡 Answer: %s\n", response.Answer)

	// Example 4: Query with sources
	fmt.Println("\n🔍 Querying with sources: Tell me about Go's concurrency")
	responseWithSources, err := ragClient.QueryWithSources("Tell me about Go's concurrency", true)
	if err != nil {
		log.Fatalf("Failed to query with sources: %v", err)
	}
	fmt.Printf("💡 Answer: %s\n", responseWithSources.Answer)
	if len(responseWithSources.Sources) > 0 {
		fmt.Println("📚 Sources:")
		for i, source := range responseWithSources.Sources {
			fmt.Printf("  [%d] Score: %.2f - %s\n", i+1, source.Score, source.Content[:min(100, len(source.Content))])
		}
	}

	// Example 5: List all documents
	fmt.Println("\n📚 Listing all documents...")
	docs, err := ragClient.ListDocuments()
	if err != nil {
		log.Fatalf("Failed to list documents: %v", err)
	}
	fmt.Printf("Found %d documents\n", len(docs))
	for _, doc := range docs[:min(3, len(docs))] {
		fmt.Printf("  - ID: %s, Created: %s\n", doc.ID, doc.Created.Format("2006-01-02 15:04:05"))
	}

	// Example 6: Advanced document listing with metadata
	fmt.Println("\n📊 Getting documents with enhanced metadata...")
	docsWithInfo, err := ragClient.ListDocumentsWithInfo()
	if err != nil {
		log.Fatalf("Failed to list documents with info: %v", err)
	}
	for _, info := range docsWithInfo[:min(2, len(docsWithInfo))] {
		fmt.Printf("📄 Document: %s\n", info.ID)
		if info.Summary != "" {
			fmt.Printf("   Summary: %s\n", info.Summary)
		}
		if len(info.Keywords) > 0 {
			fmt.Printf("   Keywords: %v\n", info.Keywords)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}