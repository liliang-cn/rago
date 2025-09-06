package rag

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/rag/ingest"
	"github.com/liliang-cn/rago/v2/pkg/rag/storage"
)

// TestRAGServiceEndToEnd tests the complete RAG pillar functionality
func TestRAGServiceEndToEnd(t *testing.T) {
	// Create temporary directory for test data
	tempDir, err := os.MkdirTemp("", "rago_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test configuration
	config := &Config{
		StorageBackend: "sqlite",
		VectorStore: storage.VectorConfig{
			Backend:    "sqlite",
			DBPath:     filepath.Join(tempDir, "vectors.db"),
			Dimensions: 384,
			Metric:     "cosine",
			IndexType:  "flat",
		},
		KeywordStore: storage.KeywordConfig{
			Backend:   "bleve",
			IndexPath: filepath.Join(tempDir, "keyword_index"),
			Analyzer:  "standard",
			Languages: []string{"en"},
			Stemming:  true,
		},
		DocumentStore: storage.DocumentConfig{
			Backend: "sqlite",
			DBPath:  filepath.Join(tempDir, "documents.db"),
		},
		Ingestion: ingest.Config{
			ChunkingStrategy: core.ChunkingConfig{
				Strategy:     "sentence",
				ChunkSize:    500,
				ChunkOverlap: 50,
				MinChunkSize: 100,
			},
		},
		BatchSize:     10,
		MaxConcurrent: 5,
	}

	// Create embedder (using the default embedder from adapter.go)
	embedder := &DefaultEmbedder{}

	// Create RAG service
	service, err := NewService(config, embedder)
	if err != nil {
		t.Fatalf("Failed to create RAG service: %v", err)
	}
	defer service.Close()

	ctx := context.Background()

	t.Run("Document Ingestion", func(t *testing.T) {
		// Test single document ingestion
		req := core.IngestRequest{
			Content:     "This is a test document about artificial intelligence and machine learning.",
			ContentType: "text/plain",
			Metadata: map[string]interface{}{
				"category": "test",
				"author":   "test_user",
				"filename": "test_doc.txt",
			},
		}

		response, err := service.IngestDocument(ctx, req)
		if err != nil {
			t.Fatalf("Failed to ingest document: %v", err)
		}

		if response.DocumentID == "" {
			t.Error("Expected non-empty document ID")
		}
		if response.ChunksCount <= 0 {
			t.Error("Expected at least one chunk")
		}

		t.Logf("Ingested document %s with %d chunks", response.DocumentID, response.ChunksCount)
	})

	t.Run("Batch Document Ingestion", func(t *testing.T) {
		// Test batch ingestion
		requests := []core.IngestRequest{
			{
				Content:     "Document about natural language processing and neural networks.",
				ContentType: "text/plain",
				Metadata:    map[string]interface{}{"category": "nlp", "filename": "nlp_doc.txt"},
			},
			{
				Content:     "Deep learning models for computer vision applications.",
				ContentType: "text/plain",
				Metadata:    map[string]interface{}{"category": "computer_vision", "filename": "cv_doc.txt"},
			},
		}

		response, err := service.IngestBatch(ctx, requests)
		if err != nil {
			t.Fatalf("Failed to ingest batch: %v", err)
		}

		if response.SuccessfulCount != 2 {
			t.Errorf("Expected 2 successful ingestions, got %d", response.SuccessfulCount)
		}
		if response.FailedCount != 0 {
			t.Errorf("Expected 0 failed ingestions, got %d", response.FailedCount)
		}

		t.Logf("Batch ingestion: %d successful, %d failed", response.SuccessfulCount, response.FailedCount)
	})

	t.Run("Basic Search", func(t *testing.T) {
		// Wait a bit for indexing to complete
		time.Sleep(100 * time.Millisecond)

		// Test basic search
		searchReq := core.SearchRequest{
			Query:  "artificial intelligence",
			Limit:  10,
			Offset: 0,
		}

		response, err := service.Search(ctx, searchReq)
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}

		if len(response.Results) == 0 {
			t.Error("Expected at least one search result")
		}

		t.Logf("Search returned %d results in %v", len(response.Results), response.Duration)

		// Verify result structure
		for i, result := range response.Results {
			if result.DocumentID == "" {
				t.Errorf("Result %d missing document ID", i)
			}
			if result.Content == "" {
				t.Errorf("Result %d missing content", i)
			}
			if result.Score <= 0 {
				t.Errorf("Result %d has invalid score: %f", i, result.Score)
			}
		}
	})

	t.Run("Hybrid Search", func(t *testing.T) {
		// Test hybrid search with vector and keyword combination
		hybridReq := core.HybridSearchRequest{
			SearchRequest: core.SearchRequest{
				Query:  "machine learning neural networks",
				Limit:  5,
				Offset: 0,
			},
			VectorWeight:  0.6,
			KeywordWeight: 0.4,
			RRFParams: core.RRFParams{
				K: 60,
			},
		}

		response, err := service.HybridSearch(ctx, hybridReq)
		if err != nil {
			t.Fatalf("Failed to perform hybrid search: %v", err)
		}

		if len(response.Results) == 0 {
			t.Error("Expected at least one hybrid search result")
		}

		t.Logf("Hybrid search returned %d results with fusion method: %s", 
			len(response.Results), response.FusionMethod)

		// Verify fusion results exist
		if len(response.VectorResults) == 0 && len(response.KeywordResults) == 0 {
			t.Error("Expected either vector or keyword results")
		}
	})

	t.Run("Document Management", func(t *testing.T) {
		// Test listing documents
		filter := core.DocumentFilter{
			Limit:  10,
			Offset: 0,
		}

		docs, err := service.ListDocuments(ctx, filter)
		if err != nil {
			t.Fatalf("Failed to list documents: %v", err)
		}

		if len(docs) == 0 {
			t.Error("Expected at least one document")
		}

		t.Logf("Found %d documents", len(docs))

		// Test document deletion
		if len(docs) > 0 {
			docID := docs[0].ID
			err := service.DeleteDocument(ctx, docID)
			if err != nil {
				t.Fatalf("Failed to delete document: %v", err)
			}

			t.Logf("Successfully deleted document %s", docID)

			// Verify deletion
			updatedDocs, err := service.ListDocuments(ctx, filter)
			if err != nil {
				t.Fatalf("Failed to list documents after deletion: %v", err)
			}

			if len(updatedDocs) >= len(docs) {
				t.Error("Expected fewer documents after deletion")
			}
		}
	})

	t.Run("Statistics and Management", func(t *testing.T) {
		// Test getting statistics
		stats, err := service.GetStats(ctx)
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}

		if stats.TotalDocuments < 0 {
			t.Error("Expected non-negative total documents")
		}
		if stats.TotalChunks < 0 {
			t.Error("Expected non-negative total chunks")
		}

		t.Logf("Stats: %d documents, %d chunks, %d bytes storage", 
			stats.TotalDocuments, stats.TotalChunks, stats.StorageSize)

		// Test optimization
		err = service.Optimize(ctx)
		if err != nil {
			t.Fatalf("Failed to optimize: %v", err)
		}

		t.Log("Optimization completed successfully")

		// Test reset (commented out to preserve test data for verification)
		// err = service.Reset(ctx)
		// if err != nil {
		// 	t.Fatalf("Failed to reset: %v", err)
		// }
		// t.Log("Reset completed successfully")
	})
}

// TestRAGServiceConfiguration tests various configuration scenarios
func TestRAGServiceConfiguration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "rago_config_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	embedder := &DefaultEmbedder{}

	t.Run("Invalid Configuration", func(t *testing.T) {
		// Test with nil config
		_, err := NewService(nil, embedder)
		if err == nil {
			t.Error("Expected error with nil config")
		}

		// Test with nil embedder
		config := &Config{StorageBackend: "sqlite"}
		_, err = NewService(config, nil)
		if err == nil {
			t.Error("Expected error with nil embedder")
		}

		// Test with empty storage backend
		config = &Config{}
		_, err = NewService(config, embedder)
		if err == nil {
			t.Error("Expected error with empty storage backend")
		}
	})

	t.Run("Valid Configuration", func(t *testing.T) {
		config := &Config{
			StorageBackend: "sqlite",
			VectorStore: storage.VectorConfig{
				Backend:    "sqlite",
				DBPath:     filepath.Join(tempDir, "test_vectors.db"),
				Dimensions: 384,
			},
			KeywordStore: storage.KeywordConfig{
				Backend:   "bleve",
				IndexPath: filepath.Join(tempDir, "test_keyword"),
			},
			DocumentStore: storage.DocumentConfig{
				Backend: "sqlite",
				DBPath:  filepath.Join(tempDir, "test_documents.db"),
			},
			Ingestion: ingest.Config{
				ChunkingStrategy: core.ChunkingConfig{
					Strategy:     "sentence",
					ChunkSize:    500,
					ChunkOverlap: 50,
					MinChunkSize: 100,
				},
			},
			BatchSize:     5,
			MaxConcurrent: 2,
		}

		service, err := NewService(config, embedder)
		if err != nil {
			t.Fatalf("Failed to create service with valid config: %v", err)
		}
		defer service.Close()

		// Verify service is operational
		ctx := context.Background()
		stats, err := service.GetStats(ctx)
		if err != nil {
			t.Fatalf("Failed to get stats from new service: %v", err)
		}

		if stats == nil {
			t.Error("Expected non-nil stats")
		}
	})
}

// TestRAGServiceBackwardCompatibility tests the backward compatibility adapter
func TestRAGServiceBackwardCompatibility(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "rago_compat_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create old-style core RAG config with proper paths
	coreConfig := core.RAGConfig{
		StorageBackend: "sqlite",
		VectorStore: core.VectorStoreConfig{
			Backend:    "sqlite",
			Dimensions: 384,
			Metric:     "cosine",
			IndexType:  "flat",
		},
		KeywordStore: core.KeywordStoreConfig{
			Backend:   "bleve",
			Analyzer:  "standard",
			Languages: []string{"en"},
			Stemming:  true,
		},
	}

	embedder := &DefaultEmbedder{}

	// Ensure the data directory exists for the default paths
	if err := os.MkdirAll("./data", 0755); err != nil {
		t.Fatalf("Failed to create data directory: %v", err)
	}

	// Test backward compatibility adapter
	service, err := NewServiceFromCoreConfig(coreConfig, embedder)
	if err != nil {
		t.Fatalf("Failed to create service from core config: %v", err)
	}
	defer service.Close()
	
	// Note: The adapter creates default paths in ./data/ directory
	// In a real application, this would be configured properly
	// For this test, we'll clean up the ./data directory if it was created
	defer func() {
		if _, err := os.Stat("./data"); err == nil {
			os.RemoveAll("./data")
		}
	}()

	// Verify it's a working RAG service
	ctx := context.Background()
	stats, err := service.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats from backward compatible service: %v", err)
	}

	if stats == nil {
		t.Error("Expected non-nil stats from backward compatible service")
	}

	t.Log("Backward compatibility adapter working correctly")
}