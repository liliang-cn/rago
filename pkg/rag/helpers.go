package rag

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/rag/ingest"
	"github.com/liliang-cn/rago/v2/pkg/rag/search"
	"github.com/liliang-cn/rago/v2/pkg/rag/storage"
	"github.com/liliang-cn/rago/v2/pkg/store"
)

// ===== INITIALIZATION HELPERS =====

// initializeStorage sets up all storage backends
func (s *Service) initializeStorage(embedder storage.Embedder) error {
	var err error

	// Initialize vector backend
	s.vectorBackend, err = storage.NewVectorBackend(s.config.VectorStore)
	if err != nil {
		return fmt.Errorf("failed to initialize vector backend: %w", err)
	}

	// Initialize keyword backend
	s.keywordBackend, err = storage.NewKeywordBackend(s.config.KeywordStore)
	if err != nil {
		return fmt.Errorf("failed to initialize keyword backend: %w", err)
	}

	// Initialize document backend (needs vector store for SQLite implementation)
	if sqliteBackend, ok := s.vectorBackend.(*storage.SQLiteVectorBackend); ok {
		// Access the underlying SQLite store
		vectorStore := extractSQLiteStore(sqliteBackend)
		s.documentBackend, err = storage.NewDocumentBackend(s.config.DocumentStore, vectorStore)
		if err != nil {
			return fmt.Errorf("failed to initialize document backend: %w", err)
		}
	} else {
		return fmt.Errorf("unsupported backend combination")
	}

	// Create storage manager
	storageConfig := &storage.Config{
		Vector:   s.config.VectorStore,
		Keyword:  s.config.KeywordStore,
		Document: s.config.DocumentStore,
	}

	s.storageManager, err = storage.NewManager(
		s.vectorBackend,
		s.keywordBackend,
		s.documentBackend,
		embedder,
		storageConfig,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize storage manager: %w", err)
	}

	return nil
}

// initializeProcessing sets up processing components
func (s *Service) initializeProcessing() error {
	var err error

	// Initialize ingestion engine
	coreRAGConfig := core.RAGConfig{
		ChunkingStrategy: core.ChunkingConfig{
			Strategy:     s.config.Ingestion.ChunkingStrategy.Strategy,
			ChunkSize:    s.config.Ingestion.ChunkingStrategy.ChunkSize,
			ChunkOverlap: s.config.Ingestion.ChunkingStrategy.ChunkOverlap,
			MinChunkSize: s.config.Ingestion.ChunkingStrategy.MinChunkSize,
		},
	}

	s.ingestionEngine, err = ingest.NewEngine(coreRAGConfig, s.storageManager)
	if err != nil {
		return fmt.Errorf("failed to initialize ingestion engine: %w", err)
	}

	// Initialize search engine
	s.searchEngine, err = search.NewEngine(
		s.vectorBackend,
		s.keywordBackend,
		s.storageManager.GetEmbedder(),
		s.config.Search,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize search engine: %w", err)
	}

	return nil
}

// backgroundOptimization runs optimization tasks periodically
func (s *Service) backgroundOptimization() {
	ticker := time.NewTicker(s.config.OptimizeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			if err := s.Optimize(ctx); err != nil {
				log.Printf("[RAG] Background optimization failed: %v", err)
			}
			cancel()
		}
	}
}

// ===== STATISTICS HELPERS =====

// updateIngestionStats updates statistics after document ingestion
func (s *Service) updateIngestionStats(response *core.IngestResponse, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stats.TotalDocuments++
	s.stats.TotalChunks += response.ChunksCount
	s.stats.StorageSize += response.StorageSize

	// Update performance metrics
	s.stats.Performance["last_ingest_duration"] = duration
	s.stats.LastHealthCheck = time.Now()
}

// updateBatchStats updates statistics after batch ingestion
func (s *Service) updateBatchStats(response *core.BatchIngestResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	totalChunks := 0
	totalSize := int64(0)

	for _, resp := range response.Responses {
		totalChunks += resp.ChunksCount
		totalSize += resp.StorageSize
	}

	s.stats.TotalDocuments += response.SuccessfulCount
	s.stats.TotalChunks += totalChunks
	s.stats.StorageSize += totalSize

	// Update performance metrics
	s.stats.Performance["last_batch_duration"] = response.Duration
	s.stats.Performance["last_batch_size"] = response.TotalDocuments
	s.stats.LastHealthCheck = time.Now()
}

// updateDeletionStats updates statistics after document deletion
func (s *Service) updateDeletionStats(docID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Note: We don't know exact chunk count or size without querying
	// In a production system, we'd want to track this more accurately
	if s.stats.TotalDocuments > 0 {
		s.stats.TotalDocuments--
	}

	s.stats.LastHealthCheck = time.Now()
}

// updateSearchStats updates search performance statistics
func (s *Service) updateSearchStats(req core.SearchRequest, resultCount int, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stats.Performance["last_search_duration"] = duration
	s.stats.Performance["last_search_results"] = resultCount
	s.stats.LastHealthCheck = time.Now()
}

// refreshStats updates statistics from storage backends
func (s *Service) refreshStats(ctx context.Context) error {
	storageStats, err := s.storageManager.GetStats(ctx)
	if err != nil {
		return err
	}

	s.stats.TotalDocuments = int(storageStats.Document.TotalDocuments)
	s.stats.TotalChunks = int(storageStats.Vector.TotalVectors)
	s.stats.StorageSize = storageStats.Document.TotalSize
	s.stats.IndexSize = storageStats.Vector.IndexSize

	// Update by content type
	s.stats.ByContentType = make(map[string]int)
	for contentType, count := range storageStats.Document.ByContentType {
		s.stats.ByContentType[contentType] = int(count)
	}

	return nil
}

// ===== CONVERSION HELPERS =====

// convertDocumentFilter converts core DocumentFilter to storage DocumentFilter
func convertDocumentFilter(coreFilter core.DocumentFilter) storage.DocumentFilter {
	var contentTypes []string
	if coreFilter.ContentType != "" {
		contentTypes = []string{coreFilter.ContentType}
	}

	var createdAfter, createdBefore *time.Time
	if !coreFilter.CreatedAfter.IsZero() {
		createdAfter = &coreFilter.CreatedAfter
	}
	if !coreFilter.CreatedBefore.IsZero() {
		createdBefore = &coreFilter.CreatedBefore
	}

	return storage.DocumentFilter{
		ContentTypes:  contentTypes,
		Metadata:      coreFilter.Metadata,
		CreatedAfter:  createdAfter,
		CreatedBefore: createdBefore,
		Limit:         coreFilter.Limit,
		Offset:        coreFilter.Offset,
	}
}

// convertSearchRequest converts core SearchRequest to internal search request data
func convertSearchRequest(coreReq core.SearchRequest) struct {
	Query   string
	Limit   int
	Offset  int
	Filter  map[string]interface{}
	Options search.SearchOptions
} {
	return struct {
		Query   string
		Limit   int
		Offset  int
		Filter  map[string]interface{}
		Options search.SearchOptions
	}{
		Query:  coreReq.Query,
		Limit:  coreReq.Limit,
		Offset: coreReq.Offset,
		Filter: coreReq.Filter,
		Options: search.SearchOptions{
			IncludeContent:    true,
			IncludeMetadata:   true,
			IncludeHighlights: true,
			ContextLength:     200,
			Timeout:           30,
		},
	}
}

// convertHybridSearchRequest converts core HybridSearchRequest to search HybridSearchRequest
func convertHybridSearchRequest(coreReq core.HybridSearchRequest) search.HybridSearchRequest {
	return search.HybridSearchRequest{
		Query:   coreReq.Query,
		Limit:   coreReq.Limit,
		Offset:  coreReq.Offset,
		Filter:  coreReq.Filter,
		FusionOptions: search.FusionOptions{
			Method:        "rrf",
			VectorWeight:  float64(coreReq.VectorWeight),
			KeywordWeight: float64(coreReq.KeywordWeight),
			RRFConstant:   float64(coreReq.RRFParams.K),
			Normalize:     true,
		},
		Options: search.SearchOptions{
			IncludeContent:    true,
			IncludeMetadata:   true,
			IncludeHighlights: true,
			ContextLength:     200,
			Timeout:           30,
		},
	}
}

// ===== CONFIGURATION HELPERS =====

// validateConfig validates the RAG service configuration
func validateConfig(config *Config) error {
	if config.StorageBackend == "" {
		return fmt.Errorf("storage backend must be specified")
	}

	if config.VectorStore.Backend == "" {
		return fmt.Errorf("vector store backend must be specified")
	}

	if config.KeywordStore.Backend == "" {
		return fmt.Errorf("keyword store backend must be specified")
	}

	if config.DocumentStore.Backend == "" {
		return fmt.Errorf("document store backend must be specified")
	}

	if config.BatchSize <= 0 {
		config.BatchSize = 10
	}

	if config.MaxConcurrent <= 0 {
		config.MaxConcurrent = 5
	}

	return nil
}

// ===== UTILITY HELPERS =====

// extractSQLiteStore extracts the underlying SQLite store from a vector backend
// This allows the document backend to share the same SQLite database
func extractSQLiteStore(backend storage.VectorBackend) *store.SQLiteStore {
	if sqliteBackend, ok := backend.(*storage.SQLiteVectorBackend); ok {
		return sqliteBackend.GetStore()
	}
	return nil
}

