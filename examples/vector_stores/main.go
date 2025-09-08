package main

import (
	"context"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"github.com/liliang-cn/rago/v2/pkg/store"
)

func main() {
	ctx := context.Background()

	// Example 1: Using the factory with configuration
	fmt.Println("=== Example 1: Factory Pattern ===")
	
	factory := store.NewStoreFactory()
	
	// Create a SQLite/sqvect store
	sqliteConfig := store.StoreConfig{
		Type: "sqvect",
		Parameters: map[string]interface{}{
			"db_path":    "./data/example.db",
			"dimensions": 1536,
		},
	}
	
	sqliteStore, err := factory.CreateStore(sqliteConfig)
	if err != nil {
		log.Fatal("Failed to create SQLite store:", err)
	}
	defer sqliteStore.Close()
	
	// Use the store
	doc := &store.Document{
		ID:        "doc1",
		Content:   "This is an example document about vector databases.",
		Embedding: generateMockEmbedding(1536),
		Source:    "example.txt",
		Metadata: map[string]interface{}{
			"category": "database",
			"author":   "example",
		},
	}
	
	if err := sqliteStore.Store(ctx, doc); err != nil {
		log.Fatal("Failed to store document:", err)
	}
	
	fmt.Println("✅ Document stored successfully")
	
	// Search for similar documents
	searchQuery := store.SearchQuery{
		Embedding:       generateMockEmbedding(1536),
		TopK:            5,
		Threshold:       0.7,
		IncludeMetadata: true,
	}
	
	results, err := sqliteStore.Search(ctx, searchQuery)
	if err != nil {
		log.Fatal("Failed to search:", err)
	}
	
	fmt.Printf("📊 Found %d similar documents in %v\n", results.TotalCount, results.QueryTime)
	
	// Example 2: Future store types (when implemented)
	fmt.Println("\n=== Example 2: Future Store Types ===")
	
	// Show how different stores would be configured
	configs := []store.StoreConfig{
		{
			Type: "pgvector",
			Parameters: map[string]interface{}{
				"connection_string": "postgresql://user:pass@localhost/vectordb",
				"dimensions":        1536,
				"table_name":        "documents",
			},
		},
		{
			Type: "qdrant",
			Parameters: map[string]interface{}{
				"url":             "http://localhost:6333",
				"api_key":         "your-api-key",
				"collection_name": "documents",
				"dimensions":      1536,
			},
		},
		{
			Type: "pinecone",
			Parameters: map[string]interface{}{
				"api_key":     "your-api-key",
				"environment": "us-west1-gcp",
				"index_name":  "documents",
				"dimensions":  1536,
			},
		},
		{
			Type: "weaviate",
			Parameters: map[string]interface{}{
				"url":        "http://localhost:8080",
				"api_key":    "your-api-key",
				"class_name": "Document",
				"dimensions": 1536,
			},
		},
		{
			Type: "milvus",
			Parameters: map[string]interface{}{
				"host":            "localhost",
				"port":            19530,
				"collection_name": "documents",
				"dimensions":      1536,
			},
		},
		{
			Type: "chromadb",
			Parameters: map[string]interface{}{
				"path":            "./data/chroma",
				"collection_name": "documents",
				"dimensions":      1536,
			},
		},
	}
	
	fmt.Println("Supported store types (when implemented):")
	for _, config := range configs {
		fmt.Printf("  - %s: %v\n", config.Type, config.Parameters)
	}
	
	// Example 3: Store abstraction benefits
	fmt.Println("\n=== Example 3: Store Abstraction Benefits ===")
	
	// The same code works with any store implementation
	var vectorStore store.VectorStore
	
	// Could switch between stores without changing application code
	useStore := "sqvect" // Could be "pgvector", "qdrant", etc.
	
	switch useStore {
	case "sqvect":
		vectorStore, err = store.NewSqvectStore("./data/app.db", 1536)
	// Future implementations:
	// case "pgvector":
	//     vectorStore, err = store.NewPgVectorStore(connectionString, 1536)
	// case "qdrant":
	//     vectorStore, err = store.NewQdrantStore(url, apiKey, 1536)
	default:
		vectorStore, err = store.NewDefaultStore("sqvect")
	}
	
	if err != nil {
		log.Fatal("Failed to create store:", err)
	}
	
	// Application code remains the same regardless of store type
	err = vectorStore.Initialize(ctx)
	if err != nil {
		log.Fatal("Failed to initialize store:", err)
	}
	defer vectorStore.Close()
	
	// Store documents
	docs := []*store.Document{
		{
			ID:        "doc2",
			Content:   "Vector databases enable semantic search.",
			Embedding: generateMockEmbedding(1536),
			Source:    "vectors.txt",
		},
		{
			ID:        "doc3",
			Content:   "RAG systems combine retrieval with generation.",
			Embedding: generateMockEmbedding(1536),
			Source:    "rag.txt",
		},
	}
	
	if err := vectorStore.StoreBatch(ctx, docs); err != nil {
		log.Fatal("Failed to store batch:", err)
	}
	
	fmt.Printf("✅ Stored %d documents\n", len(docs))
	
	// Count documents
	count, err := vectorStore.Count(ctx)
	if err != nil {
		log.Fatal("Failed to count:", err)
	}
	
	fmt.Printf("📚 Total documents in store: %d\n", count)
	
	// Example 4: Configuration from file
	fmt.Println("\n=== Example 4: Configuration Management ===")
	
	// In production, configuration would come from config file
	// e.g., from rago.toml:
	configExample := `
[store]
type = "sqvect"  # Easy to change to "pgvector", "qdrant", etc.

[store.parameters]
db_path = "./data/production.db"
dimensions = 1536

# Future: Just change the type and parameters
# [store]
# type = "pgvector"
# 
# [store.parameters]
# connection_string = "postgresql://user:pass@localhost/vectordb"
# dimensions = 1536
`
	
	fmt.Println("Example configuration:")
	fmt.Println(configExample)
	
	fmt.Println("\n✨ Benefits of abstraction:")
	fmt.Println("1. Switch vector stores without changing application code")
	fmt.Println("2. Test with SQLite locally, use Pgvector/Qdrant in production")
	fmt.Println("3. Compare performance across different backends")
	fmt.Println("4. Gradual migration between storage systems")
	fmt.Println("5. Plugin architecture for custom implementations")
}

// generateMockEmbedding creates a mock embedding vector
func generateMockEmbedding(dimensions int) []float32 {
	embedding := make([]float32, dimensions)
	for i := range embedding {
		embedding[i] = float32(i) / float32(dimensions)
	}
	return embedding
}