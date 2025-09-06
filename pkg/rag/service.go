// Package rag implements the RAG (Retrieval-Augmented Generation) pillar for RAGO V3.
// This pillar provides document ingestion, storage, and retrieval capabilities as an independent,
// modular component that can work standalone or integrate with other RAGO pillars.
package rag

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/rag/ingest"
	"github.com/liliang-cn/rago/v2/pkg/rag/search"
	"github.com/liliang-cn/rago/v2/pkg/rag/storage"
)

// Service implements the RAGService interface and provides the main orchestrator
// for all RAG operations including document ingestion, storage management, and retrieval.
type Service struct {
	// Core components
	ingestionEngine *ingest.Engine
	searchEngine    *search.DefaultEngine
	storageManager  *storage.Manager

	// Storage backends
	vectorBackend   storage.VectorBackend
	keywordBackend  storage.KeywordBackend
	documentBackend storage.DocumentBackend

	// Configuration and state
	config *Config
	stats  *RAGStats
	mu     sync.RWMutex

	// Lifecycle management
	closed bool
}

// Config defines configuration options for the RAG service
type Config struct {
	// Storage configuration
	StorageBackend string                    `toml:"storage_backend"`
	VectorStore    storage.VectorConfig     `toml:"vector_store"`
	KeywordStore   storage.KeywordConfig    `toml:"keyword_store"`
	DocumentStore  storage.DocumentConfig   `toml:"document_store"`

	// Ingestion configuration
	Ingestion ingest.Config `toml:"ingestion"`

	// Search configuration
	Search search.Config `toml:"search"`

	// Performance tuning
	BatchSize        int           `toml:"batch_size"`
	MaxConcurrent    int           `toml:"max_concurrent"`
	OptimizeInterval time.Duration `toml:"optimize_interval"`
}

// RAGStats tracks comprehensive statistics about the RAG system
type RAGStats struct {
	// Document metrics
	TotalDocuments int                    `json:"total_documents"`
	TotalChunks    int                    `json:"total_chunks"`
	StorageSize    int64                  `json:"storage_size"`
	IndexSize      int64                  `json:"index_size"`
	ByContentType  map[string]int         `json:"by_content_type"`

	// Performance metrics
	LastOptimized time.Time              `json:"last_optimized"`
	Performance   map[string]interface{} `json:"performance"`

	// Health metrics
	LastHealthCheck time.Time `json:"last_health_check"`
	HealthStatus    string    `json:"health_status"`
}

// NewService creates a new RAG service instance with the provided configuration
func NewService(config *Config, embedder storage.Embedder) (*Service, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if embedder == nil {
		return nil, fmt.Errorf("embedder cannot be nil")
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	service := &Service{
		config: config,
		stats: &RAGStats{
			ByContentType: make(map[string]int),
			Performance:   make(map[string]interface{}),
			HealthStatus:  "initializing",
		},
	}

	// Initialize storage backends
	if err := service.initializeStorage(embedder); err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize processing components
	if err := service.initializeProcessing(); err != nil {
		return nil, fmt.Errorf("failed to initialize processing components: %w", err)
	}

	// Start background optimization if configured
	if config.OptimizeInterval > 0 {
		go service.backgroundOptimization()
	}

	service.stats.HealthStatus = "healthy"
	service.stats.LastHealthCheck = time.Now()

	log.Printf("[RAG] Service initialized successfully with backend: %s", config.StorageBackend)
	return service, nil
}

// ===== DOCUMENT OPERATIONS =====

// IngestDocument processes a single document and stores it in the RAG system
func (s *Service) IngestDocument(ctx context.Context, req core.IngestRequest) (*core.IngestResponse, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, fmt.Errorf("service is closed")
	}
	s.mu.RUnlock()

	start := time.Now()
	
	// Use the ingestion engine to process the document
	response, err := s.ingestionEngine.IngestDocument(ctx, req)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "rag", "IngestDocument", "ingestion failed")
	}

	// Update statistics
	s.updateIngestionStats(response, time.Since(start))

	log.Printf("[RAG] Successfully ingested document %s (%d chunks)", response.DocumentID, response.ChunksCount)
	return response, nil
}

// IngestBatch processes multiple documents in batch for improved efficiency
func (s *Service) IngestBatch(ctx context.Context, requests []core.IngestRequest) (*core.BatchIngestResponse, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, fmt.Errorf("service is closed")
	}
	s.mu.RUnlock()

	// Use the ingestion engine for batch processing
	response, err := s.ingestionEngine.IngestBatch(ctx, requests)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "rag", "IngestBatch", "batch ingestion failed")
	}

	// Update statistics
	s.updateBatchStats(response)

	log.Printf("[RAG] Batch ingestion completed: %d success, %d failed", response.SuccessfulCount, response.FailedCount)
	return response, nil
}

// DeleteDocument removes a document and all its associated chunks from the RAG system
func (s *Service) DeleteDocument(ctx context.Context, docID string) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return fmt.Errorf("service is closed")
	}
	s.mu.RUnlock()

	if docID == "" {
		return fmt.Errorf("document ID cannot be empty")
	}

	// Use storage manager to delete from all backends
	err := s.storageManager.DeleteDocument(ctx, docID)
	if err != nil {
		return core.WrapErrorWithContext(err, "rag", "DeleteDocument", "deletion failed")
	}

	// Update statistics
	s.updateDeletionStats(docID)

	log.Printf("[RAG] Successfully deleted document %s", docID)
	return nil
}

// ListDocuments retrieves a list of documents with optional filtering
func (s *Service) ListDocuments(ctx context.Context, filter core.DocumentFilter) ([]core.Document, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, fmt.Errorf("service is closed")
	}
	s.mu.RUnlock()

	// Convert core filter to storage filter
	storageFilter := convertDocumentFilter(filter)

	// Use storage manager to list documents
	docs, err := s.storageManager.ListDocuments(ctx, storageFilter)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "rag", "ListDocuments", "failed to list documents")
	}

	// Convert back to core format
	coreDocs := make([]core.Document, len(docs))
	for i, doc := range docs {
		coreDocs[i] = storage.ConvertToCoreDocument(doc)
	}

	return coreDocs, nil
}

// ===== SEARCH OPERATIONS =====

// Search performs basic vector or keyword search depending on the request
func (s *Service) Search(ctx context.Context, req core.SearchRequest) (*core.SearchResponse, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, fmt.Errorf("service is closed")
	}
	s.mu.RUnlock()

	start := time.Now()

	// Convert to search request format
	searchReq := convertSearchRequest(req)

	// Generate query vector from text
	if req.Query != "" {
		// For now, perform keyword search (TODO: add vector search support)
		keywordReq := search.KeywordSearchRequest{
			Query:   req.Query,
			Limit:   searchReq.Limit,
			Offset:  searchReq.Offset,
			Filter:  searchReq.Filter,
			Options: searchReq.Options,
		}

		result, err := s.searchEngine.KeywordSearch(ctx, keywordReq)
		if err != nil {
			return nil, core.WrapErrorWithContext(err, "rag", "Search", "search failed")
		}

		// Convert to core response
		response := &core.SearchResponse{
			Results:  search.ConvertToCoreSearchResults(result.Results),
			Total:    result.Total,
			Duration: time.Since(start),
			Query:    req.Query,
		}

		s.updateSearchStats(req, len(response.Results), time.Since(start))
		return response, nil
	}

	return &core.SearchResponse{
		Results:  []core.SearchResult{},
		Total:    0,
		Duration: time.Since(start),
		Query:    req.Query,
	}, nil
}

// HybridSearch performs advanced search combining vector and keyword approaches with fusion
func (s *Service) HybridSearch(ctx context.Context, req core.HybridSearchRequest) (*core.HybridSearchResponse, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, fmt.Errorf("service is closed")
	}
	s.mu.RUnlock()

	start := time.Now()

	// Convert to hybrid search request
	hybridReq := convertHybridSearchRequest(req)

	// Execute hybrid search through search engine
	results, err := s.searchEngine.HybridSearch(ctx, hybridReq)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "rag", "HybridSearch", "hybrid search failed")
	}

	// Convert results to core format
	response := &core.HybridSearchResponse{
		SearchResponse: core.SearchResponse{
			Results:  search.ConvertToCoreSearchResults(results.FusedResults),
			Total:    results.Total,
			Duration: time.Since(start),
			Query:    req.Query,
		},
		VectorResults:  search.ConvertToCoreSearchResults(results.VectorResults),
		KeywordResults: search.ConvertToCoreSearchResults(results.KeywordResults),
		FusionMethod:   results.FusionMethod,
	}

	// Update search statistics
	s.updateSearchStats(req.SearchRequest, len(response.Results), time.Since(start))

	return response, nil
}

// ===== MANAGEMENT OPERATIONS =====

// GetStats returns comprehensive statistics about the RAG system
func (s *Service) GetStats(ctx context.Context) (*core.RAGStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, fmt.Errorf("service is closed")
	}

	// Refresh stats from storage
	if err := s.refreshStats(ctx); err != nil {
		log.Printf("[RAG] Warning: failed to refresh stats: %v", err)
	}

	// Convert to core stats format
	coreStats := &core.RAGStats{
		TotalDocuments: s.stats.TotalDocuments,
		TotalChunks:    s.stats.TotalChunks,
		StorageSize:    s.stats.StorageSize,
		IndexSize:      s.stats.IndexSize,
		ByContentType:  s.stats.ByContentType,
		LastOptimized:  s.stats.LastOptimized,
		Performance:    s.stats.Performance,
	}

	return coreStats, nil
}

// Optimize performs comprehensive optimization of all storage backends and indices
func (s *Service) Optimize(ctx context.Context) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return fmt.Errorf("service is closed")
	}
	s.mu.RUnlock()

	log.Printf("[RAG] Starting optimization process...")
	start := time.Now()

	// Use storage manager to optimize all backends
	err := s.storageManager.Optimize(ctx)
	if err != nil {
		return core.WrapErrorWithContext(err, "rag", "Optimize", "optimization failed")
	}

	// Update optimization timestamp
	s.mu.Lock()
	s.stats.LastOptimized = time.Now()
	s.mu.Unlock()

	duration := time.Since(start)
	log.Printf("[RAG] Optimization completed successfully in %v", duration)

	return nil
}

// Reset clears all data from the RAG system
func (s *Service) Reset(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("service is closed")
	}

	log.Printf("[RAG] Resetting all data...")

	// Use storage manager to reset all backends
	err := s.storageManager.Reset(ctx)
	if err != nil {
		return core.WrapErrorWithContext(err, "rag", "Reset", "reset failed")
	}

	// Reset statistics
	s.stats = &RAGStats{
		ByContentType:   make(map[string]int),
		Performance:     make(map[string]interface{}),
		HealthStatus:    "healthy",
		LastHealthCheck: time.Now(),
	}

	log.Printf("[RAG] Reset completed successfully")
	return nil
}

// Close gracefully shuts down the RAG service and closes all resources
func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	log.Printf("[RAG] Shutting down service...")

	var errors []error

	// Close processing components
	if s.ingestionEngine != nil {
		if err := s.ingestionEngine.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close ingestion engine: %w", err))
		}
	}

	if s.searchEngine != nil {
		if err := s.searchEngine.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close search engine: %w", err))
		}
	}

	// Close storage manager
	if s.storageManager != nil {
		if err := s.storageManager.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close storage manager: %w", err))
		}
	}

	s.closed = true

	if len(errors) > 0 {
		return fmt.Errorf("shutdown partially failed: %v", errors)
	}

	log.Printf("[RAG] Service shutdown completed successfully")
	return nil
}