package main

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/liliang-cn/rago/v2/client"
)

func main() {
	// Get config path - try relative to current directory
	configPath := filepath.Join(".", "rago.toml")
	
	// Create a new client
	ragClient, err := client.New(configPath)
	if err != nil {
		// Try without config path (will use default search)
		ragClient, err = client.New("")
		if err != nil {
			log.Fatal("Failed to create client:", err)
		}
	}
	defer ragClient.Close()

	// Example 1: Ingest text with enhanced metadata extraction
	fmt.Println("=== Example 1: Ingesting with Enhanced Metadata ===")
	
	opts := &client.IngestOptions{
		EnhancedExtraction: true,
		ChunkSize:          300,
		Overlap:            50,
	}

	texts := []struct {
		content string
		source  string
	}{
		{
			content: "Meeting scheduled tomorrow at 3pm with Dr. Johnson at Stanford Medical Center to discuss the Q4 surgical procedures and patient outcomes.",
			source:  "meeting-notes-demo",
		},
		{
			content: "Yesterday we reviewed the test results from last week. The patient John Smith showed significant improvement. Follow-up appointment scheduled for next Monday.",
			source:  "patient-records-demo",
		},
	}

	for _, text := range texts {
		err = ragClient.IngestTextWithOptions(text.content, text.source, opts)
		if err != nil {
			log.Printf("Failed to ingest %s: %v", text.source, err)
			continue
		}
		fmt.Printf("✓ Ingested: %s\n", text.source)
	}

	// Example 2: List documents with metadata
	fmt.Println("\n=== Example 2: Listing Documents with Metadata ===")
	
	docs, err := ragClient.ListDocumentsWithInfo()
	if err != nil {
		log.Fatal("Failed to list documents:", err)
	}

	fmt.Printf("\nFound %d documents:\n", len(docs))
	
	// Show last 2 documents (our newly ingested ones)
	startIdx := 0
	if len(docs) > 2 {
		startIdx = len(docs) - 2
	}
	
	for i := startIdx; i < len(docs); i++ {
		doc := docs[i]
		fmt.Printf("\nDocument: %s\n", doc.ID[:8]+"...")
		fmt.Printf("  Source: %s\n", doc.Source)
		fmt.Printf("  Created: %s\n", doc.Created.Format("2006-01-02 15:04:05"))
		
		if doc.Summary != "" {
			fmt.Printf("  Summary: %s\n", doc.Summary)
		}
		
		if len(doc.Keywords) > 0 {
			fmt.Printf("  Keywords: %s\n", strings.Join(doc.Keywords, ", "))
		}
		
		if doc.DocumentType != "" {
			fmt.Printf("  Type: %s\n", doc.DocumentType)
		}
		
		if len(doc.TemporalRefs) > 0 {
			fmt.Println("  Temporal References:")
			for term, date := range doc.TemporalRefs {
				fmt.Printf("    %s: %s\n", term, date)
			}
		}
	}

	// Example 3: Query the ingested content
	fmt.Println("\n=== Example 3: Querying ===")
	
	query := "What medical meetings are scheduled?"
	fmt.Printf("\nQuery: %s\n", query)
	
	resp, err := ragClient.Query(query)
	if err != nil {
		log.Printf("Query failed: %v", err)
	} else {
		fmt.Printf("Answer: %s\n", resp.Answer)
	}

	fmt.Println("\n=== Example Complete ===")
}