package store

import (
	"context"
	"fmt"
	"strings"
)

// StoreFactory creates vector stores based on configuration
type StoreFactory struct {
	registeredTypes map[string]StoreCreator
}

// StoreCreator is a function that creates a VectorStore from config
type StoreCreator func(config StoreConfig) (VectorStore, error)

// NewStoreFactory creates a new store factory
func NewStoreFactory() *StoreFactory {
	f := &StoreFactory{
		registeredTypes: make(map[string]StoreCreator),
	}

	// Register default store types
	f.registerDefaults()

	return f
}

// registerDefaults registers the built-in store types
func (f *StoreFactory) registerDefaults() {
	// Register SQLite/sqvect store
	f.Register("sqvect", createSqvectStore)
	f.Register("sqlite", createSqvectStore) // Alias

	// Future store types can be registered here:
	// f.Register("pgvector", createPgVectorStore)
	// f.Register("qdrant", createQdrantStore)
	// f.Register("weaviate", createWeaviateStore)
	// f.Register("pinecone", createPineconeStore)
	// f.Register("milvus", createMilvusStore)
	// f.Register("chromadb", createChromaStore)
}

// Register adds a new store type to the factory
func (f *StoreFactory) Register(storeType string, creator StoreCreator) {
	f.registeredTypes[strings.ToLower(storeType)] = creator
}

// CreateStore creates a vector store based on configuration
func (f *StoreFactory) CreateStore(config StoreConfig) (VectorStore, error) {
	storeType := strings.ToLower(config.Type)

	creator, exists := f.registeredTypes[storeType]
	if !exists {
		return nil, fmt.Errorf("unsupported store type: %s", config.Type)
	}

	return creator(config)
}

// SupportedTypes returns a list of supported store types
func (f *StoreFactory) SupportedTypes() []string {
	types := make([]string, 0, len(f.registeredTypes))
	for t := range f.registeredTypes {
		types = append(types, t)
	}
	return types
}

// createSqvectStore creates a SQLite vector store
func createSqvectStore(config StoreConfig) (VectorStore, error) {
	// Extract parameters
	dbPath, ok := config.Parameters["db_path"].(string)
	if !ok || dbPath == "" {
		dbPath = "./.rago/data/rag.db"
	}

	dimensions := 1536 // Default for OpenAI embeddings
	if dim, ok := config.Parameters["dimensions"].(int); ok {
		dimensions = dim
	} else if dim, ok := config.Parameters["dimensions"].(float64); ok {
		dimensions = int(dim)
	}

	store := NewSqvectWrapper(dbPath, dimensions)

	// Initialize the store
	ctx := context.Background()
	if err := store.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize sqvect store: %w", err)
	}

	return store, nil
}

// Helper function to get a store with default configuration
func NewDefaultStore(storeType string) (VectorStore, error) {
	factory := NewStoreFactory()

	config := StoreConfig{
		Type: storeType,
		Parameters: map[string]interface{}{
			"db_path":    "./.rago/data/rag.db",
			"dimensions": 1536,
		},
	}

	return factory.CreateStore(config)
}

// NewSqvectStore creates a SQLite vector store with specific configuration
func NewSqvectStore(dbPath string, dimensions int) (VectorStore, error) {
	factory := NewStoreFactory()

	config := StoreConfig{
		Type: "sqvect",
		Parameters: map[string]interface{}{
			"db_path":    dbPath,
			"dimensions": dimensions,
		},
	}

	return factory.CreateStore(config)
}
