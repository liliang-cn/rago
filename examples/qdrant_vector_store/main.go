package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/rag/store"
)

func main() {
	// Configuration for Qdrant
	config := store.StoreConfig{
		Type: "qdrant",
		Parameters: map[string]interface{}{
			"url":        "localhost:6334", // Default Qdrant gRPC port
			"collection": "test_documents",
		},
	}

	// Alternative configuration with Docker container using m.daocloud.io mirror
	// config := store.StoreConfig{
	//     Type: "qdrant",
	//     Parameters: map[string]interface{}{
	//         "url":        "localhost:6334",  // Map to container's port
	//         "collection": "rago_documents",
	//     },
	// }
	//
	// To run Qdrant with the daocloud mirror:
	// docker run -p 6333:6333 -p 6334:6334 \
	//     -v $(pwd)/qdrant_storage:/qdrant/storage:z \
	//     m.daocloud.io/docker.io/qdrant/qdrant:latest

	// Create Qdrant vector store
	vectorStore, err := store.NewVectorStore(config)
	if err != nil {
		log.Fatalf("Failed to create Qdrant store: %v", err)
	}

	// Ensure proper cleanup
	if qdrantStore, ok := vectorStore.(*store.QdrantStore); ok {
		defer qdrantStore.Close()
	}

	ctx := context.Background()

	// Sample documents
	documents := []struct {
		content string
		vector  []float64
	}{
		{
			content: "Qdrant is a vector database for semantic search and neural network applications.",
			vector:  generateRandomVector(1536),
		},
		{
			content: "RAGO supports multiple vector stores including SQLite and now Qdrant.",
			vector:  generateRandomVector(1536),
		},
		{
			content: "Vector databases enable efficient similarity search for AI applications.",
			vector:  generateRandomVector(1536),
		},
	}

	fmt.Println("=== Qdrant Vector Store Example ===")
	fmt.Println()

	// 1. Store documents
	fmt.Println("1. Storing documents...")
	chunks := make([]domain.Chunk, 0, len(documents))
	for i, doc := range documents {
		chunk := domain.Chunk{
			ID:         uuid.New().String(),
			DocumentID: fmt.Sprintf("doc_%d", i+1),
			Content:    doc.content,
			Vector:     doc.vector,
			Metadata: map[string]interface{}{
				"timestamp": time.Now().Unix(),
				"source":    "example",
			},
		}
		chunks = append(chunks, chunk)
	}

	err = vectorStore.Store(ctx, chunks)
	if err != nil {
		log.Fatalf("Failed to store chunks: %v", err)
	}
	fmt.Printf("✓ Stored %d documents\n", len(chunks))
	fmt.Println()

	// 2. Search for similar documents
	fmt.Println("2. Searching for similar documents...")
	queryVector := generateRandomVector(1536)
	
	results, err := vectorStore.Search(ctx, queryVector, 3)
	if err != nil {
		log.Fatalf("Failed to search: %v", err)
	}

	fmt.Printf("✓ Found %d results:\n", len(results))
	for i, result := range results {
		fmt.Printf("   %d. Score: %.4f\n", i+1, result.Score)
		fmt.Printf("      Content: %s\n", result.Content)
		fmt.Printf("      Document ID: %s\n", result.DocumentID)
		fmt.Println()
	}

	// 3. Search with filters
	fmt.Println("3. Searching with filters...")
	filteredResults, err := vectorStore.SearchWithFilters(ctx, queryVector, 2, map[string]interface{}{
		"source": "example",
	})
	if err != nil {
		log.Fatalf("Failed to search with filters: %v", err)
	}

	fmt.Printf("✓ Found %d filtered results:\n", len(filteredResults))
	for i, result := range filteredResults {
		fmt.Printf("   %d. Score: %.4f, Doc ID: %s\n", i+1, result.Score, result.DocumentID)
	}
	fmt.Println()

	// 4. Delete a document
	fmt.Println("4. Deleting a document...")
	err = vectorStore.Delete(ctx, "doc_1")
	if err != nil {
		log.Fatalf("Failed to delete document: %v", err)
	}
	fmt.Println("✓ Document deleted")
	fmt.Println()

	// 5. Verify deletion
	fmt.Println("5. Verifying deletion...")
	remainingResults, err := vectorStore.Search(ctx, queryVector, 10)
	if err != nil {
		log.Fatalf("Failed to search after deletion: %v", err)
	}
	fmt.Printf("✓ Remaining documents: %d\n", len(remainingResults))
	fmt.Println()

	// 6. Reset (clear all data)
	fmt.Println("6. Resetting vector store...")
	fmt.Print("Are you sure you want to clear all data? (y/N): ")
	var confirm string
	fmt.Scanln(&confirm)
	if confirm == "y" || confirm == "Y" {
		err = vectorStore.Reset(ctx)
		if err != nil {
			log.Fatalf("Failed to reset: %v", err)
		}
		fmt.Println("✓ Vector store reset")
	} else {
		fmt.Println("✗ Reset cancelled")
	}

	fmt.Println()
	fmt.Println("=== Example completed successfully ===")
	fmt.Println()
	fmt.Println("To use with Docker:")
	fmt.Println("1. Run Qdrant container:")
	fmt.Println("   docker run -p 6333:6333 -p 6334:6334 \\")
	fmt.Println("     -v $(pwd)/qdrant_storage:/qdrant/storage:z \\")
	fmt.Println("     m.daocloud.io/docker.io/qdrant/qdrant:latest")
	fmt.Println()
	fmt.Println("2. Access Qdrant UI at http://localhost:6333")
	fmt.Println()
	fmt.Println("3. Update your RAGO config to use Qdrant:")
	fmt.Println("   vector_store:")
	fmt.Println("     type: qdrant")
	fmt.Println("     parameters:")
	fmt.Println("       url: localhost:6334")
	fmt.Println("       collection: rago_documents")
}

// generateRandomVector generates a random vector for demonstration
func generateRandomVector(size int) []float64 {
	vector := make([]float64, size)
	for i := range vector {
		vector[i] = float64(i%10) / 10.0
	}
	return vector
}

func init() {
	// Check if Qdrant is running
	if os.Getenv("SKIP_QDRANT_CHECK") != "true" {
		fmt.Println("Note: This example requires Qdrant to be running.")
		fmt.Println("If Qdrant is not running, the example will fail.")
		fmt.Println("Set SKIP_QDRANT_CHECK=true to skip this check.")
		fmt.Println()
	}
}