package rag

import (
	"context"
	"fmt"

	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/rag/storage"
)

// NewServiceFromCoreConfig creates a RAG service from core configuration for backward compatibility.
// This function acts as an adapter between the old core.RAGConfig and the new modular structure.
func NewServiceFromCoreConfig(coreConfig core.RAGConfig, embedder storage.Embedder) (*Service, error) {
	// Convert core config to new RAG config
	config := &Config{
		StorageBackend: coreConfig.StorageBackend,
		VectorStore: storage.VectorConfig{
			Backend:    coreConfig.VectorStore.Backend,
			DBPath:     "", // Will need to be set based on storage backend
			Dimensions: coreConfig.VectorStore.Dimensions,
			Metric:     coreConfig.VectorStore.Metric,
			IndexType:  coreConfig.VectorStore.IndexType,
		},
		KeywordStore: storage.KeywordConfig{
			Backend:   coreConfig.KeywordStore.Backend,
			IndexPath: "", // Will need to be set based on storage backend
			Analyzer:  coreConfig.KeywordStore.Analyzer,
			Languages: coreConfig.KeywordStore.Languages,
			Stemming:  coreConfig.KeywordStore.Stemming,
		},
		DocumentStore: storage.DocumentConfig{
			Backend: "sqlite", // Default to SQLite
			DBPath:  "",
		},
		BatchSize:     10, // Default values
		MaxConcurrent: 5,
	}

	// Set default paths if not specified
	if config.VectorStore.DBPath == "" {
		config.VectorStore.DBPath = "./data/vectors.db"
	}
	if config.KeywordStore.IndexPath == "" {
		config.KeywordStore.IndexPath = "./data/keyword_index"
	}
	if config.DocumentStore.DBPath == "" {
		config.DocumentStore.DBPath = "./data/documents.db"
	}

	return NewService(config, embedder)
}

// DefaultEmbedder provides a simple embedder implementation for compatibility.
// In a production system, this would be replaced with a real embedding service.
type DefaultEmbedder struct{}

// Embed generates a dummy embedding vector for compatibility.
// This is a placeholder implementation - in production, you'd use a real embedding model.
func (e *DefaultEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	if text == "" {
		return nil, fmt.Errorf("empty text provided")
	}
	
	// Generate a simple hash-based vector for demonstration
	// In production, this would call an actual embedding service
	vector := make([]float64, 384) // Common embedding dimension
	
	// Simple hash-based pseudo-embedding
	hash := simpleHash(text)
	for i := range vector {
		vector[i] = float64((hash + i) % 100) / 100.0
	}
	
	return vector, nil
}

// simpleHash creates a simple hash from text
func simpleHash(text string) int {
	hash := 0
	for _, r := range text {
		hash = hash*31 + int(r)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}