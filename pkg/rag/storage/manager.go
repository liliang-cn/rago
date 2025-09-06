package storage

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
)

// Manager orchestrates storage operations across multiple backends.
type Manager struct {
	vectorBackend   VectorBackend
	keywordBackend  KeywordBackend
	documentBackend DocumentBackend
	embedder        Embedder // For generating vectors during ingestion
	config          *Config
	mu              sync.RWMutex
}

// Config defines the storage manager configuration.
type Config struct {
	Vector   VectorConfig   `toml:"vector"`
	Keyword  KeywordConfig  `toml:"keyword"`
	Document DocumentConfig `toml:"document"`
}

// NewManager creates a new storage manager with the specified backends and configuration.
func NewManager(
	vectorBackend VectorBackend,
	keywordBackend KeywordBackend,
	documentBackend DocumentBackend,
	embedder Embedder,
	config *Config,
) (*Manager, error) {
	if vectorBackend == nil {
		return nil, fmt.Errorf("vector backend cannot be nil")
	}
	if keywordBackend == nil {
		return nil, fmt.Errorf("keyword backend cannot be nil")
	}
	if documentBackend == nil {
		return nil, fmt.Errorf("document backend cannot be nil")
	}
	if embedder == nil {
		return nil, fmt.Errorf("embedder cannot be nil")
	}

	return &Manager{
		vectorBackend:   vectorBackend,
		keywordBackend:  keywordBackend,
		documentBackend: documentBackend,
		embedder:        embedder,
		config:          config,
	}, nil
}

// StoreDocument processes and stores a document with its chunks across all backends.
func (m *Manager) StoreDocument(ctx context.Context, doc *Document, chunks []TextChunk) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	docID := doc.ID
	
	log.Printf("[STORAGE] Storing document %s with %d chunks", docID, len(chunks))
	start := time.Now()

	// Store document metadata first
	if err := m.documentBackend.StoreDocument(ctx, doc); err != nil {
		return core.WrapErrorWithContext(err, "storage", "StoreDocument", "failed to store document metadata")
	}

	// Process chunks in parallel
	var vectorChunks []VectorChunk
	var keywordChunks []KeywordChunk

	// Generate embeddings and prepare chunks
	for _, chunk := range chunks {
		// Generate vector embedding
		vector, err := m.embedder.Embed(ctx, chunk.Content)
		if err != nil {
			return core.WrapErrorWithContext(err, "storage", "StoreDocument", "failed to generate embedding")
		}

		// Create vector chunk
		vectorChunk := ConvertToVectorChunk(chunk, docID, vector)
		vectorChunks = append(vectorChunks, vectorChunk)

		// Create keyword chunk
		keywordChunk := ConvertToKeywordChunk(chunk, docID)
		keywordChunks = append(keywordChunks, keywordChunk)
	}

	// Store in both vector and keyword backends concurrently
	var wg sync.WaitGroup
	errors := make(chan error, 2)

	// Store vector embeddings
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := m.vectorBackend.StoreVectors(ctx, docID, vectorChunks); err != nil {
			errors <- core.WrapErrorWithContext(err, "storage", "StoreDocument", "vector storage failed")
		}
	}()

	// Store keyword index
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := m.keywordBackend.IndexChunks(ctx, docID, keywordChunks); err != nil {
			errors <- core.WrapErrorWithContext(err, "storage", "StoreDocument", "keyword indexing failed")
		}
	}()

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		// If storage fails, we should clean up what was stored
		m.cleanupFailedDocument(ctx, docID)
		return err
	}

	duration := time.Since(start)
	log.Printf("[STORAGE] Document %s stored successfully in %v", docID, duration)

	return nil
}

// GetDocument retrieves a document by ID from the document backend.
func (m *Manager) GetDocument(ctx context.Context, docID string) (*Document, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.documentBackend.GetDocument(ctx, docID)
}

// ListDocuments retrieves documents with optional filtering.
func (m *Manager) ListDocuments(ctx context.Context, filter DocumentFilter) ([]Document, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.documentBackend.ListDocuments(ctx, filter)
}

// DeleteDocument removes a document from all storage backends.
func (m *Manager) DeleteDocument(ctx context.Context, docID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Printf("[STORAGE] Deleting document %s", docID)

	var errors []error

	// Delete from all backends concurrently
	var wg sync.WaitGroup
	errorsChan := make(chan error, 3)

	// Delete from vector backend
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := m.vectorBackend.DeleteDocument(ctx, docID); err != nil {
			errorsChan <- core.WrapErrorWithContext(err, "storage", "DeleteDocument", "vector deletion failed")
		}
	}()

	// Delete from keyword backend
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := m.keywordBackend.DeleteDocument(ctx, docID); err != nil {
			errorsChan <- core.WrapErrorWithContext(err, "storage", "DeleteDocument", "keyword deletion failed")
		}
	}()

	// Delete from document backend
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := m.documentBackend.DeleteDocument(ctx, docID); err != nil {
			errorsChan <- core.WrapErrorWithContext(err, "storage", "DeleteDocument", "document deletion failed")
		}
	}()

	wg.Wait()
	close(errorsChan)

	// Collect errors
	for err := range errorsChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("deletion partially failed: %v", errors)
	}

	log.Printf("[STORAGE] Document %s deleted successfully", docID)
	return nil
}

// GetStats returns combined statistics from all storage backends.
func (m *Manager) GetStats(ctx context.Context) (*Stats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get stats from all backends concurrently
	var wg sync.WaitGroup
	var vectorStats *VectorStats
	var keywordStats *KeywordStats
	var documentStats *DocumentStats
	var errors []error
	var mu sync.Mutex

	// Vector stats
	wg.Add(1)
	go func() {
		defer wg.Done()
		stats, err := m.vectorBackend.GetStats(ctx)
		mu.Lock()
		if err != nil {
			errors = append(errors, err)
		} else {
			vectorStats = stats
		}
		mu.Unlock()
	}()

	// Keyword stats
	wg.Add(1)
	go func() {
		defer wg.Done()
		stats, err := m.keywordBackend.GetStats(ctx)
		mu.Lock()
		if err != nil {
			errors = append(errors, err)
		} else {
			keywordStats = stats
		}
		mu.Unlock()
	}()

	// Document stats
	wg.Add(1)
	go func() {
		defer wg.Done()
		stats, err := m.documentBackend.GetStats(ctx)
		mu.Lock()
		if err != nil {
			errors = append(errors, err)
		} else {
			documentStats = stats
		}
		mu.Unlock()
	}()

	wg.Wait()

	if len(errors) > 0 {
		return nil, fmt.Errorf("failed to collect stats: %v", errors)
	}

	return &Stats{
		Vector:   *vectorStats,
		Keyword:  *keywordStats,
		Document: *documentStats,
	}, nil
}

// Optimize performs optimization on all storage backends.
func (m *Manager) Optimize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Printf("[STORAGE] Starting optimization for all backends")
	start := time.Now()

	// Optimize all backends concurrently
	var wg sync.WaitGroup
	errors := make(chan error, 3)

	// Optimize vector backend
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := m.vectorBackend.Optimize(ctx); err != nil {
			errors <- core.WrapErrorWithContext(err, "storage", "Optimize", "vector optimization failed")
		}
	}()

	// Optimize keyword backend
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := m.keywordBackend.Optimize(ctx); err != nil {
			errors <- core.WrapErrorWithContext(err, "storage", "Optimize", "keyword optimization failed")
		}
	}()

	// Optimize document backend
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := m.documentBackend.Optimize(ctx); err != nil {
			errors <- core.WrapErrorWithContext(err, "storage", "Optimize", "document optimization failed")
		}
	}()

	wg.Wait()
	close(errors)

	// Check for errors
	var optimizationErrors []error
	for err := range errors {
		optimizationErrors = append(optimizationErrors, err)
	}

	if len(optimizationErrors) > 0 {
		return fmt.Errorf("optimization partially failed: %v", optimizationErrors)
	}

	duration := time.Since(start)
	log.Printf("[STORAGE] Optimization completed in %v", duration)

	return nil
}

// Reset clears all data from all storage backends.
func (m *Manager) Reset(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Printf("[STORAGE] Resetting all backends")

	// Reset all backends concurrently
	var wg sync.WaitGroup
	errors := make(chan error, 3)

	// Reset vector backend
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := m.vectorBackend.Reset(ctx); err != nil {
			errors <- core.WrapErrorWithContext(err, "storage", "Reset", "vector reset failed")
		}
	}()

	// Reset keyword backend
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := m.keywordBackend.Reset(ctx); err != nil {
			errors <- core.WrapErrorWithContext(err, "storage", "Reset", "keyword reset failed")
		}
	}()

	// Reset document backend
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := m.documentBackend.Reset(ctx); err != nil {
			errors <- core.WrapErrorWithContext(err, "storage", "Reset", "document reset failed")
		}
	}()

	wg.Wait()
	close(errors)

	// Check for errors
	var resetErrors []error
	for err := range errors {
		resetErrors = append(resetErrors, err)
	}

	if len(resetErrors) > 0 {
		return fmt.Errorf("reset partially failed: %v", resetErrors)
	}

	log.Printf("[STORAGE] All backends reset successfully")
	return nil
}

// Close closes all storage backends.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errors []error

	if err := m.vectorBackend.Close(); err != nil {
		errors = append(errors, fmt.Errorf("vector backend close failed: %w", err))
	}

	if err := m.keywordBackend.Close(); err != nil {
		errors = append(errors, fmt.Errorf("keyword backend close failed: %w", err))
	}

	if err := m.documentBackend.Close(); err != nil {
		errors = append(errors, fmt.Errorf("document backend close failed: %w", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("close partially failed: %v", errors)
	}

	log.Printf("[STORAGE] Manager closed successfully")
	return nil
}

// GetVectorBackend returns the vector backend for direct access.
func (m *Manager) GetVectorBackend() VectorBackend {
	return m.vectorBackend
}

// GetKeywordBackend returns the keyword backend for direct access.
func (m *Manager) GetKeywordBackend() KeywordBackend {
	return m.keywordBackend
}

// GetDocumentBackend returns the document backend for direct access.
func (m *Manager) GetDocumentBackend() DocumentBackend {
	return m.documentBackend
}

// GetEmbedder returns the embedder for direct access.
func (m *Manager) GetEmbedder() Embedder {
	return m.embedder
}

// cleanupFailedDocument attempts to clean up a document if storage failed.
func (m *Manager) cleanupFailedDocument(ctx context.Context, docID string) {
	log.Printf("[STORAGE] Cleaning up failed document %s", docID)

	// Best effort cleanup - log errors but don't fail
	if err := m.vectorBackend.DeleteDocument(ctx, docID); err != nil {
		log.Printf("[STORAGE] Failed to cleanup vector data for document %s: %v", docID, err)
	}

	if err := m.keywordBackend.DeleteDocument(ctx, docID); err != nil {
		log.Printf("[STORAGE] Failed to cleanup keyword data for document %s: %v", docID, err)
	}

	if err := m.documentBackend.DeleteDocument(ctx, docID); err != nil {
		log.Printf("[STORAGE] Failed to cleanup document metadata for document %s: %v", docID, err)
	}
}

// Health checks the health of all storage backends.
func (m *Manager) Health(ctx context.Context) map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	health := make(map[string]string)

	// Check vector backend health
	if _, err := m.vectorBackend.GetStats(ctx); err != nil {
		health["vector"] = fmt.Sprintf("unhealthy: %v", err)
	} else {
		health["vector"] = "healthy"
	}

	// Check keyword backend health
	if _, err := m.keywordBackend.GetStats(ctx); err != nil {
		health["keyword"] = fmt.Sprintf("unhealthy: %v", err)
	} else {
		health["keyword"] = "healthy"
	}

	// Check document backend health
	if _, err := m.documentBackend.GetStats(ctx); err != nil {
		health["document"] = fmt.Sprintf("unhealthy: %v", err)
	} else {
		health["document"] = "healthy"
	}

	return health
}