package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/liliang-cn/rago/internal/config"
	"github.com/liliang-cn/rago/internal/domain"
	"github.com/liliang-cn/rago/internal/embedder"
	"github.com/liliang-cn/rago/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFullIntegration(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "rago-integration-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test configuration
	cfg := &config.Config{
		Ollama: config.OllamaConfig{
			BaseURL:        "http://localhost:11434",
			EmbeddingModel: "nomic-embed-text",
			LLMModel:      "qwen2.5",
		},
		Sqvect: config.SqvectConfig{
			DBPath: filepath.Join(tempDir, "test.db"),
		},
		Ingest: config.IngestConfig{
			MetadataExtraction: config.MetadataExtractionConfig{
				Enable:   true,
				LLMModel: "qwen2.5",
			},
		},
	}

	// Initialize components
	embedderClient, err := embedder.NewOllamaService(cfg.Ollama.BaseURL, cfg.Ollama.EmbeddingModel)
	require.NoError(t, err)
	
	vectorStore, err := store.NewSQLiteStore(cfg.Sqvect.DBPath, 768, 10, 100)
	require.NoError(t, err)
	defer vectorStore.Close()

	docStore := store.NewDocumentStore(vectorStore.GetSqvectStore())

	ctx := context.Background()

	// Test documents
	testDocs := []domain.Document{
		{
			ID:      "doc1",
			Path:    filepath.Join(tempDir, "go-tutorial.md"),
			Content: "Go is a programming language developed by Google. It's fast, statically typed, and compiled. Go is excellent for backend services and system programming.",
			Created: time.Now(),
			Metadata: map[string]interface{}{
				"category":      "tutorial",
				"language":      "english",
				"difficulty":    "beginner",
				"topic":         "programming",
				"document_type": "Technical Guide",
			},
		},
		{
			ID:      "doc2", 
			Path:    filepath.Join(tempDir, "python-basics.md"),
			Content: "Python is a high-level programming language. It's interpreted, dynamically typed, and very readable. Python is great for data science, web development, and automation.",
			Created: time.Now(),
			Metadata: map[string]interface{}{
				"category":      "tutorial",
				"language":      "english", 
				"difficulty":    "beginner",
				"topic":         "programming",
				"document_type": "Technical Guide",
			},
		},
		{
			ID:      "doc3",
			Path:    filepath.Join(tempDir, "rust-advanced.md"),
			Content: "Rust is a systems programming language focused on safety and performance. It prevents memory errors and has zero-cost abstractions. Rust is perfect for operating systems and embedded systems.",
			Created: time.Now(),
			Metadata: map[string]interface{}{
				"category":      "tutorial",
				"language":      "english",
				"difficulty":    "advanced",
				"topic":         "programming", 
				"document_type": "Technical Manual",
			},
		},
		{
			ID:      "doc4",
			Path:    filepath.Join(tempDir, "database-design.md"),
			Content: "Database design involves creating a detailed data model of a database. Good design reduces data redundancy and improves data integrity. PostgreSQL and MySQL are popular relational databases.",
			Created: time.Now(),
			Metadata: map[string]interface{}{
				"category":      "tutorial",
				"language":      "english",
				"difficulty":    "intermediate", 
				"topic":         "database",
				"document_type": "Technical Guide",
			},
		},
		{
			ID:      "doc5",
			Path:    filepath.Join(tempDir, "api-documentation.md"),
			Content: "RESTful APIs follow REST architectural principles. They use HTTP methods like GET, POST, PUT, DELETE. Proper API design includes versioning, authentication, and clear documentation.",
			Created: time.Now(),
			Metadata: map[string]interface{}{
				"category":      "documentation",
				"language":      "english",
				"difficulty":    "intermediate",
				"topic":         "api",
				"document_type": "API Reference",
			},
		},
	}

	t.Run("Document Ingestion", func(t *testing.T) {
		// Process all test documents directly using store
		for _, doc := range testDocs {
			// Store document first
			err := docStore.Store(ctx, doc)
			require.NoError(t, err, "Failed to store document %s", doc.ID)
			
			// Create chunks and embed them
			text := doc.Content
			if len(text) > 500 {
				text = text[:500] // Truncate for testing
			}
			
			// Get embedding
			vector, err := embedderClient.Embed(ctx, text)
			require.NoError(t, err, "Failed to embed content for %s", doc.ID)
			
			// Create chunk
			chunk := domain.Chunk{
				ID:         doc.ID + "_chunk_0",
				DocumentID: doc.ID,
				Content:    text,
				Vector:     vector,
				Metadata:   doc.Metadata,
			}
			
			// Store chunk
			err = vectorStore.Store(ctx, []domain.Chunk{chunk})
			require.NoError(t, err, "Failed to store chunk for %s", doc.ID)
		}

		// Verify documents were stored
		docs, err := docStore.List(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(docs), 5, "Should have at least 5 documents")
	})

	t.Run("Vector Search", func(t *testing.T) {
		// Test basic search
		queryVector, err := embedderClient.Embed(ctx, "programming languages comparison")
		require.NoError(t, err)

		results, err := vectorStore.Search(ctx, queryVector, 5)
		require.NoError(t, err)
		assert.Greater(t, len(results), 0, "Should return search results")

		// Check that results are relevant to programming
		foundProgrammingDoc := false
		for _, result := range results {
			if result.Metadata["topic"] == "programming" {
				foundProgrammingDoc = true
				break
			}
		}
		assert.True(t, foundProgrammingDoc, "Should find programming-related documents")
	})

	t.Run("Filtered Search", func(t *testing.T) {
		// Test search with filters
		queryVector, err := embedderClient.Embed(ctx, "beginner programming tutorial")
		require.NoError(t, err)

		// Filter for beginner level documents
		filters := map[string]interface{}{
			"difficulty": "beginner",
		}

		results, err := vectorStore.SearchWithFilters(ctx, queryVector, 10, filters)
		require.NoError(t, err)
		assert.Greater(t, len(results), 0, "Should return filtered results")

		// Verify all results match filter
		for _, result := range results {
			assert.Equal(t, "beginner", result.Metadata["difficulty"], 
				"All results should have beginner difficulty")
		}
	})

	t.Run("Multi-Filter Search", func(t *testing.T) {
		// Test search with multiple filters
		queryVector, err := embedderClient.Embed(ctx, "programming tutorial")
		require.NoError(t, err)

		// Filter for tutorial category AND programming topic
		filters := map[string]interface{}{
			"category": "tutorial",
			"topic":    "programming",
		}

		results, err := vectorStore.SearchWithFilters(ctx, queryVector, 10, filters)
		require.NoError(t, err)
		assert.Greater(t, len(results), 0, "Should return multi-filtered results")

		// Verify all results match both filters
		for _, result := range results {
			assert.Equal(t, "tutorial", result.Metadata["category"], 
				"All results should be tutorials")
			assert.Equal(t, "programming", result.Metadata["topic"], 
				"All results should be about programming")
		}
	})

	t.Run("Document Type Filter", func(t *testing.T) {
		// Test filtering by document type
		queryVector, err := embedderClient.Embed(ctx, "technical guide")
		require.NoError(t, err)

		filters := map[string]interface{}{
			"document_type": "Technical Guide",
		}

		results, err := vectorStore.SearchWithFilters(ctx, queryVector, 10, filters)
		require.NoError(t, err)
		assert.Greater(t, len(results), 0, "Should return Technical Guide documents")

		// Count different document types
		techGuideCount := 0
		for _, result := range results {
			if result.Metadata["document_type"] == "Technical Guide" {
				techGuideCount++
			}
		}
		assert.Equal(t, len(results), techGuideCount, "All results should be Technical Guides")
	})

	t.Run("Empty Filter Behavior", func(t *testing.T) {
		// Test that empty filters return all results
		queryVector, err := embedderClient.Embed(ctx, "tutorial")
		require.NoError(t, err)

		emptyFilters := map[string]interface{}{}
		allResults, err := vectorStore.SearchWithFilters(ctx, queryVector, 10, emptyFilters)
		require.NoError(t, err)

		noFilterResults, err := vectorStore.Search(ctx, queryVector, 10)
		require.NoError(t, err)

		assert.Equal(t, len(noFilterResults), len(allResults), 
			"Empty filters should return same as no filters")
	})

	t.Run("Document Deletion", func(t *testing.T) {
		// Delete one document
		err := vectorStore.Delete(ctx, "doc1")
		require.NoError(t, err)

		// Verify document is gone
		docs, err := docStore.List(ctx)
		require.NoError(t, err)
		
		foundDoc1 := false
		for _, doc := range docs {
			if doc.ID == "doc1" {
				foundDoc1 = true
				break
			}
		}
		assert.False(t, foundDoc1, "Document doc1 should be deleted")
	})

	t.Run("Bulk Operations", func(t *testing.T) {
		// Test processing multiple documents in sequence
		bulkDocs := []domain.Document{
			{
				ID:      "bulk1",
				Content: "Machine learning is a subset of artificial intelligence that enables computers to learn without explicit programming.",
				Created: time.Now(),
				Metadata: map[string]interface{}{
					"category":      "tutorial",
					"topic":         "machine-learning",
					"difficulty":    "intermediate",
					"document_type": "Technical Guide",
				},
			},
			{
				ID:      "bulk2",
				Content: "Deep learning uses neural networks with multiple layers to model and understand complex patterns in data.",
				Created: time.Now(),
				Metadata: map[string]interface{}{
					"category":      "tutorial", 
					"topic":         "machine-learning",
					"difficulty":    "advanced",
					"document_type": "Technical Manual",
				},
			},
		}

			// Process bulk documents
			for _, doc := range bulkDocs {
				// Store document
				err := docStore.Store(ctx, doc)
				require.NoError(t, err, "Failed to store bulk document %s", doc.ID)
				
				// Get embedding and store chunk
				vector, err := embedderClient.Embed(ctx, doc.Content)
				require.NoError(t, err, "Failed to embed bulk content for %s", doc.ID)
				
				chunk := domain.Chunk{
					ID:         doc.ID + "_chunk_0",
					DocumentID: doc.ID,
					Content:    doc.Content,
					Vector:     vector,
					Metadata:   doc.Metadata,
				}
				
				err = vectorStore.Store(ctx, []domain.Chunk{chunk})
				require.NoError(t, err, "Failed to store chunk for bulk %s", doc.ID)
			}

		// Search for machine learning content
		queryVector, err := embedderClient.Embed(ctx, "artificial intelligence neural networks")
		require.NoError(t, err)

		filters := map[string]interface{}{
			"topic": "machine-learning",
		}

		results, err := vectorStore.SearchWithFilters(ctx, queryVector, 10, filters)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 2, "Should find both bulk documents")
	})

	t.Run("Statistics and Cleanup", func(t *testing.T) {
		// Get final document count
		docs, err := docStore.List(ctx)
		require.NoError(t, err)
		
		fmt.Printf("Final document count: %d\n", len(docs))
		assert.Greater(t, len(docs), 0, "Should have documents remaining")

		// Test vector store search one more time
		queryVector, err := embedderClient.Embed(ctx, "programming")
		require.NoError(t, err)

		results, err := vectorStore.Search(ctx, queryVector, 3)
		require.NoError(t, err)
		assert.Greater(t, len(results), 0, "Should still be able to search")

		// Print some statistics
		for i, result := range results {
			fmt.Printf("Result %d: Score=%.4f, Content=%.50s...\n", 
				i+1, result.Score, result.Content)
		}
	})
}