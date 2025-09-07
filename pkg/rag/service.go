// Package rag implements the RAG (Retrieval-Augmented Generation) pillar for RAGO V3.
// This pillar provides document ingestion, storage, and retrieval capabilities as an independent,
// modular component that can work standalone or integrate with other RAGO pillars.
package rag

import (
	"context"
	"fmt"
	"log"
	"strings"
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
	llmService      core.LLMService // For question-answering functionality

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
func NewService(config *Config, embedder storage.Embedder, llmService core.LLMService) (*Service, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if embedder == nil {
		return nil, fmt.Errorf("embedder cannot be nil")
	}
	if llmService == nil {
		return nil, fmt.Errorf("llmService cannot be nil")
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	service := &Service{
		config:     config,
		llmService: llmService,
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

// Answer performs RAG-based question answering by retrieving relevant context and generating an answer
func (s *Service) Answer(ctx context.Context, req core.QARequest) (*core.QAResponse, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, fmt.Errorf("service is closed")
	}
	s.mu.RUnlock()

	start := time.Now()

	// Set defaults
	maxSources := req.MaxSources
	if maxSources <= 0 {
		maxSources = 5
	}
	
	minScore := req.MinScore
	if minScore <= 0 {
		minScore = 0.1 // Reasonable minimum relevance threshold
	}

	// Step 1: Retrieve relevant context using search
	searchStart := time.Now()
	searchReq := core.SearchRequest{
		Query: req.Question,
		Limit: maxSources,
	}

	searchResp, err := s.Search(ctx, searchReq)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "rag", "Answer", "context retrieval failed")
	}
	searchDuration := time.Since(searchStart)

	// Filter sources by minimum score
	var relevantSources []core.SearchResult
	var totalScore float32
	for _, result := range searchResp.Results {
		if result.Score >= minScore {
			relevantSources = append(relevantSources, result)
			totalScore += result.Score
		}
	}

	// Handle case where no relevant sources found
	if len(relevantSources) == 0 {
		return &core.QAResponse{
			Question:    req.Question,
			Answer:      "I don't have enough relevant information to answer this question.",
			Sources:     []core.SearchResult{},
			Confidence:  0.0,
			Model:       req.Model,
			Duration:    time.Since(start),
			SearchStats: core.SearchStats{
				SourcesFound:    len(searchResp.Results),
				SourcesUsed:     0,
				SearchDuration:  searchDuration,
				HighestScore:    0.0,
				AverageScore:    0.0,
			},
		}, nil
	}

	// Step 2: Compose prompt with retrieved context
	prompt := s.composeRAGPrompt(req.Question, relevantSources)

	// Step 3: Generate answer using LLM
	genReq := core.GenerationRequest{
		Prompt:      prompt,
		Model:       req.Model,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	genResp, err := s.llmService.Generate(ctx, genReq)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "rag", "Answer", "answer generation failed")
	}

	// Calculate statistics
	highestScore := relevantSources[0].Score
	avgScore := totalScore / float32(len(relevantSources))
	
	// Estimate confidence based on source quality and count
	confidence := s.calculateConfidence(relevantSources, avgScore)

	response := &core.QAResponse{
		Question:    req.Question,
		Answer:      genResp.Content,
		Sources:     relevantSources,
		Confidence:  confidence,
		Model:       genResp.Model,
		TokensUsed:  genResp.Usage.TotalTokens,
		Duration:    time.Since(start),
		SearchStats: core.SearchStats{
			SourcesFound:    len(searchResp.Results),
			SourcesUsed:     len(relevantSources),
			SearchDuration:  searchDuration,
			HighestScore:    highestScore,
			AverageScore:    avgScore,
		},
	}

	log.Printf("[RAG] Generated answer for question in %v (used %d/%d sources)", 
		response.Duration, len(relevantSources), len(searchResp.Results))

	return response, nil
}

// composeRAGPrompt creates a prompt with context for RAG-based answering
func (s *Service) composeRAGPrompt(question string, sources []core.SearchResult) string {
	var contextBuilder strings.Builder
	
	contextBuilder.WriteString("You are a helpful AI assistant. Use the following context to answer the user's question directly and concisely. ")
	contextBuilder.WriteString("Do not show your reasoning process or use thinking tags. Provide only the final answer. ")
	contextBuilder.WriteString("If the context doesn't contain enough information to answer the question, say so clearly.\n\n")
	
	contextBuilder.WriteString("Context:\n")
	for i, source := range sources {
		contextBuilder.WriteString(fmt.Sprintf("Source %d: %s\n\n", i+1, source.Content))
	}
	
	contextBuilder.WriteString(fmt.Sprintf("Question: %s\n\n", question))
	contextBuilder.WriteString("Answer based on the provided context:")
	
	return contextBuilder.String()
}

// calculateConfidence estimates answer confidence based on source quality
func (s *Service) calculateConfidence(sources []core.SearchResult, avgScore float32) float32 {
	if len(sources) == 0 {
		return 0.0
	}
	
	// Base confidence on average score and number of sources
	confidence := avgScore
	
	// Boost confidence for multiple high-quality sources
	if len(sources) > 2 && avgScore > 0.5 {
		confidence = min(confidence*1.2, 1.0)
	}
	
	// Reduce confidence for low-quality sources
	if avgScore < 0.3 {
		confidence *= 0.7
	}
	
	return confidence
}

// StreamAnswer performs RAG-based question answering with streaming response
func (s *Service) StreamAnswer(ctx context.Context, req core.QARequest, callback core.StreamCallback) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return fmt.Errorf("service is closed")
	}
	s.mu.RUnlock()

	start := time.Now()

	// Set defaults
	maxSources := req.MaxSources
	if maxSources <= 0 {
		maxSources = 5
	}
	
	minScore := req.MinScore
	if minScore <= 0 {
		minScore = 0.1
	}

	// Step 1: Retrieve relevant context using search
	searchReq := core.SearchRequest{
		Query: req.Question,
		Limit: maxSources,
	}

	searchResp, err := s.Search(ctx, searchReq)
	if err != nil {
		return core.WrapErrorWithContext(err, "rag", "StreamAnswer", "context retrieval failed")
	}

	// Filter sources by minimum score
	var relevantSources []core.SearchResult
	var totalScore float32
	for _, result := range searchResp.Results {
		if result.Score >= minScore {
			relevantSources = append(relevantSources, result)
			totalScore += result.Score
		}
	}

	// Handle case where no relevant sources found
	if len(relevantSources) == 0 {
		// Send a single chunk with "no information" message
		chunk := core.StreamChunk{
			Content:  "I don't have enough relevant information to answer this question.",
			Finished: true,
		}
		return callback(chunk)
	}

	// Step 2: Compose prompt with retrieved context
	prompt := s.composeRAGPrompt(req.Question, relevantSources)

	// Step 3: Stream answer using LLM
	genReq := core.GenerationRequest{
		Prompt:      prompt,
		Model:       req.Model,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	// Stream the LLM response directly
	err = s.llmService.Stream(ctx, genReq, callback)
	if err != nil {
		return core.WrapErrorWithContext(err, "rag", "StreamAnswer", "streaming generation failed")
	}

	// Send final chunk with timing information
	statsChunk := core.StreamChunk{
		Content:  "",
		Finished: true,
		Usage: core.TokenUsage{},
		Duration: time.Since(start),
	}
	
	err = callback(statsChunk)
	if err != nil {
		return core.WrapErrorWithContext(err, "rag", "StreamAnswer", "streaming generation failed")
	}

	log.Printf("[RAG] Streamed answer for question (used %d/%d sources)", 
		len(relevantSources), len(searchResp.Results))

	return nil
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