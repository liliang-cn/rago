package main

import (
	"fmt"
	"log"
	"os"

	"github.com/liliang-cn/rago/client"
)

func main() {
	fmt.Println("🚀 RAGO Document Ingestion Example")
	
	// Create client
	c, err := client.New("")
	if err != nil {
		log.Fatal("Failed to create RAGO client:", err)
	}
	defer c.Close()

	fmt.Println("✅ Client created successfully!")

	// Example 1: Ingest text content directly
	fmt.Println("\n📝 Ingesting text content...")
	textContent := `
# Introduction to Go Programming

Go is a programming language developed by Google. It's known for:
- Simple syntax and easy to learn
- Fast compilation and execution
- Built-in concurrency support
- Strong standard library
- Great tooling and ecosystem

Go is widely used for:
- Web services and APIs
- Cloud and network services
- Command-line tools
- System programming
`
	
	err = c.IngestText(textContent, "go_programming_intro")
	if err != nil {
		log.Printf("Failed to ingest text: %v", err)
	} else {
		fmt.Println("✅ Text content ingested successfully!")
	}

	// Example 2: Ingest from file (if exists)
	fmt.Println("\n📁 Checking for sample files to ingest...")
	sampleFiles := []string{"README.md", "rago.example.toml"}
	
	for _, filename := range sampleFiles {
		if _, err := os.Stat(filename); err == nil {
			fmt.Printf("📄 Ingesting file: %s\n", filename)
			err = c.IngestFile(filename)
			if err != nil {
				log.Printf("Failed to ingest file %s: %v", filename, err)
			} else {
				fmt.Printf("✅ File %s ingested successfully!\n", filename)
			}
		}
	}

	// Example 3: Query the ingested content
	fmt.Println("\n🔍 Querying ingested content...")
	response, err := c.Query("What are the benefits of Go programming language?")
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}

	fmt.Println("🤖 Response:", response.Answer)

	// Example 4: List documents
	fmt.Println("\n📚 Listing ingested documents...")
	docs, err := c.ListDocuments()
	if err != nil {
		log.Printf("Failed to list documents: %v", err)
	} else {
		fmt.Printf("📊 Total documents: %d\n", len(docs))
		for i, doc := range docs {
			if i >= 3 { // Only show first 3
				fmt.Printf("... and %d more documents\n", len(docs)-3)
				break
			}
			fmt.Printf("  • %s (ID: %s)\n", doc.Path, doc.ID[:8])
		}
	}

	fmt.Println("\n✨ Document ingestion example completed!")
}