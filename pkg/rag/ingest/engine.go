// Package ingest handles document processing and ingestion for the RAG pillar.
// It provides enhanced document processing with support for multiple formats,
// improved metadata extraction, batch processing, and document versioning.
package ingest

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	pdf "github.com/dslipak/pdf"

	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/rag/storage"
)

// Engine handles document ingestion operations with enhanced capabilities.
type Engine struct {
	config         core.RAGConfig
	storageManager *storage.Manager
	processors     map[string]Processor
	chunker        Chunker
	mu             sync.RWMutex
}

// Processor defines the interface for document content processors.
type Processor interface {
	// Process extracts content and metadata from a document
	Process(ctx context.Context, req core.IngestRequest) (*ProcessedDocument, error)
	// SupportedTypes returns the content types this processor supports
	SupportedTypes() []string
}

// Chunker defines the interface for text chunking strategies.
type Chunker interface {
	// Chunk splits text into chunks based on the chunking strategy
	Chunk(ctx context.Context, text string, options ChunkOptions) ([]TextChunk, error)
}

// ProcessedDocument represents a document after initial processing.
type ProcessedDocument struct {
	ID          string
	Content     string
	ContentType string
	Metadata    map[string]interface{}
	Size        int64
	ProcessedAt time.Time
}

// TextChunk represents a chunk of text with associated metadata.
type TextChunk struct {
	ID       string
	Content  string
	Metadata map[string]interface{}
	Position int
}

// ChunkOptions defines options for text chunking.
type ChunkOptions struct {
	Strategy    string // "fixed", "sentence", "paragraph", "semantic"
	ChunkSize   int
	ChunkOverlap int
	MinChunkSize int
}

// Config defines configuration for the ingestion engine.
type Config struct {
	ChunkingStrategy core.ChunkingConfig `toml:"chunking_strategy"`
	MaxConcurrency   int                 `toml:"max_concurrency"`
	BatchSize        int                 `toml:"batch_size"`
}

// NewEngine creates a new document ingestion engine.
func NewEngine(config core.RAGConfig, storageManager *storage.Manager) (*Engine, error) {
	log.Println("[INFO] Initializing ingestion engine...")

	engine := &Engine{
		config:         config,
		storageManager: storageManager,
		processors:     make(map[string]Processor),
	}

	// Initialize chunker
	chunker, err := NewChunker(config.ChunkingStrategy)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "ingest", "NewEngine", "failed to initialize chunker")
	}
	engine.chunker = chunker

	// Register default processors
	engine.registerDefaultProcessors()

	log.Println("[INFO] Ingestion engine initialized successfully")
	return engine, nil
}

// registerDefaultProcessors registers the default set of document processors.
func (e *Engine) registerDefaultProcessors() {
	// Text processor for .txt, .md files
	textProcessor := &TextProcessor{}
	for _, contentType := range textProcessor.SupportedTypes() {
		e.processors[contentType] = textProcessor
	}

	// PDF processor for .pdf files
	pdfProcessor := &PDFProcessor{}
	for _, contentType := range pdfProcessor.SupportedTypes() {
		e.processors[contentType] = pdfProcessor
	}

	// Raw content processor for direct content input
	rawProcessor := &RawProcessor{}
	for _, contentType := range rawProcessor.SupportedTypes() {
		e.processors[contentType] = rawProcessor
	}

	log.Printf("[INFO] Registered %d document processors", len(e.processors))
}

// IngestDocument processes and ingests a single document.
func (e *Engine) IngestDocument(ctx context.Context, req core.IngestRequest) (*core.IngestResponse, error) {
	start := time.Now()

	// Validate request
	if err := e.validateIngestRequest(req); err != nil {
		return nil, core.NewValidationError("ingest_request", req, err.Error())
	}

	// Determine content type if not specified
	contentType := req.ContentType
	if contentType == "" {
		contentType = e.detectContentType(req)
	}

	// Get appropriate processor
	processor, err := e.getProcessor(contentType)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "ingest", "IngestDocument", "failed to get processor")
	}

	// Process document
	processed, err := processor.Process(ctx, req)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "ingest", "IngestDocument", "document processing failed")
	}

	// Chunk the content
	chunkOptions := ChunkOptions{
		Strategy:     e.config.ChunkingStrategy.Strategy,
		ChunkSize:    e.config.ChunkingStrategy.ChunkSize,
		ChunkOverlap: e.config.ChunkingStrategy.ChunkOverlap,
		MinChunkSize: e.config.ChunkingStrategy.MinChunkSize,
	}

	chunks, err := e.chunker.Chunk(ctx, processed.Content, chunkOptions)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "ingest", "IngestDocument", "content chunking failed")
	}

	if len(chunks) == 0 {
		return &core.IngestResponse{
			DocumentID:  processed.ID,
			ChunksCount: 0,
			ProcessedAt: time.Now(),
			Duration:    time.Since(start),
		}, nil
	}

	// Convert to storage types and store
	storageDoc := convertToStorageDocument(processed)
	storageChunks := convertToStorageChunks(chunks)
	
	err = e.storageManager.StoreDocument(ctx, storageDoc, storageChunks)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "ingest", "IngestDocument", "document storage failed")
	}

	return &core.IngestResponse{
		DocumentID:  processed.ID,
		ChunksCount: len(chunks),
		ProcessedAt: time.Now(),
		Duration:    time.Since(start),
		StorageSize: processed.Size,
	}, nil
}

// IngestBatch processes and ingests multiple documents in parallel.
func (e *Engine) IngestBatch(ctx context.Context, requests []core.IngestRequest) (*core.BatchIngestResponse, error) {
	start := time.Now()
	totalDocs := len(requests)

	log.Printf("[INFO] Starting batch ingestion of %d documents", totalDocs)

	// Create response channels
	responses := make([]core.IngestResponse, totalDocs)
	errChan := make(chan error, totalDocs)
	
	// Control concurrency
	maxConcurrency := 5 // TODO: Make this configurable
	semaphore := make(chan struct{}, maxConcurrency)
	
	var wg sync.WaitGroup
	var successCount, failedCount int
	var mu sync.Mutex

	// Process documents concurrently
	for i, req := range requests {
		wg.Add(1)
		go func(index int, request core.IngestRequest) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			resp, err := e.IngestDocument(ctx, request)
			
			mu.Lock()
			if err != nil {
				failedCount++
				errChan <- fmt.Errorf("document %d (%s): %w", index, request.DocumentID, err)
				responses[index] = core.IngestResponse{
					DocumentID:  request.DocumentID,
					ProcessedAt: time.Now(),
				}
			} else {
				successCount++
				responses[index] = *resp
			}
			mu.Unlock()
		}(i, req)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errs []string
	for err := range errChan {
		errs = append(errs, err.Error())
	}

	batchResp := &core.BatchIngestResponse{
		Responses:       responses,
		TotalDocuments:  totalDocs,
		SuccessfulCount: successCount,
		FailedCount:     failedCount,
		Duration:        time.Since(start),
	}

	if failedCount > 0 {
		log.Printf("[WARN] Batch ingestion completed with %d failures: %v", failedCount, errs)
	}

	log.Printf("[INFO] Batch ingestion completed: %d/%d successful in %v", successCount, totalDocs, batchResp.Duration)
	return batchResp, nil
}

// validateIngestRequest validates an ingestion request.
func (e *Engine) validateIngestRequest(req core.IngestRequest) error {
	// Check that we have at least one content source
	hasContent := req.Content != ""
	hasFilePath := req.FilePath != ""
	hasURL := req.URL != ""

	contentSources := 0
	if hasContent {
		contentSources++
	}
	if hasFilePath {
		contentSources++
	}
	if hasURL {
		contentSources++
	}

	if contentSources == 0 {
		return fmt.Errorf("no content source provided (content, file_path, or url required)")
	}
	if contentSources > 1 {
		return fmt.Errorf("multiple content sources provided (only one of content, file_path, or url allowed)")
	}

	// Generate document ID if not provided
	if req.DocumentID == "" {
		// This should be handled by caller, but we can provide a fallback
		if hasFilePath {
			req.DocumentID = filepath.Base(req.FilePath)
		} else {
			req.DocumentID = uuid.New().String()
		}
	}

	return nil
}

// detectContentType attempts to detect the content type from the request.
func (e *Engine) detectContentType(req core.IngestRequest) string {
	if req.FilePath != "" {
		ext := strings.ToLower(filepath.Ext(req.FilePath))
		switch ext {
		case ".txt":
			return "text/plain"
		case ".md", ".markdown":
			return "text/markdown"
		case ".pdf":
			return "application/pdf"
		default:
			return "text/plain" // Default fallback
		}
	}
	if req.URL != "" {
		return "text/html" // Assume web content
	}
	return "text/plain" // Raw content default
}

// getProcessor returns the appropriate processor for a content type.
func (e *Engine) getProcessor(contentType string) (Processor, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	processor, exists := e.processors[contentType]
	if !exists {
		// Try fallback processors
		if strings.HasPrefix(contentType, "text/") {
			if fallback, ok := e.processors["text/plain"]; ok {
				return fallback, nil
			}
		}
		return nil, fmt.Errorf("no processor available for content type: %s", contentType)
	}

	return processor, nil
}

// Close closes the ingestion engine and cleans up resources.
func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Close any processors that need cleanup
	for _, processor := range e.processors {
		if closable, ok := processor.(interface{ Close() error }); ok {
			if err := closable.Close(); err != nil {
				log.Printf("[WARN] Error closing processor: %v", err)
			}
		}
	}

	log.Println("[INFO] Ingestion engine closed")
	return nil
}

// ===== DEFAULT PROCESSORS =====

// TextProcessor handles plain text and markdown files.
type TextProcessor struct{}

func (p *TextProcessor) Process(ctx context.Context, req core.IngestRequest) (*ProcessedDocument, error) {
	var content string
	var size int64

	if req.Content != "" {
		content = req.Content
		size = int64(len(content))
	} else if req.FilePath != "" {
		data, err := os.ReadFile(req.FilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", req.FilePath, err)
		}
		content = string(data)
		size = int64(len(data))
	} else {
		return nil, fmt.Errorf("no content source available")
	}

	// Initialize metadata
	metadata := make(map[string]interface{})
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			metadata[k] = v
		}
	}

	// Add file-based metadata if available
	if req.FilePath != "" {
		if fileInfo, err := os.Stat(req.FilePath); err == nil {
			metadata["file_name"] = filepath.Base(req.FilePath)
			metadata["file_size"] = fileInfo.Size()
			metadata["file_modified"] = fileInfo.ModTime()
		}
	}

	return &ProcessedDocument{
		ID:          req.DocumentID,
		Content:     content,
		ContentType: req.ContentType,
		Metadata:    metadata,
		Size:        size,
		ProcessedAt: time.Now(),
	}, nil
}

func (p *TextProcessor) SupportedTypes() []string {
	return []string{"text/plain", "text/markdown"}
}

// PDFProcessor handles PDF documents.
type PDFProcessor struct{}

func (p *PDFProcessor) Process(ctx context.Context, req core.IngestRequest) (*ProcessedDocument, error) {
	if req.FilePath == "" {
		return nil, fmt.Errorf("PDF processing requires file path")
	}

	r, err := pdf.Open(req.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF %s: %w", req.FilePath, err)
	}

	var content strings.Builder
	for i := 1; i <= r.NumPage(); i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			log.Printf("[WARN] Failed to extract text from page %d of %s: %v", i, req.FilePath, err)
			continue
		}
		content.WriteString(text)
		content.WriteString("\n")
	}

	// Get file info
	fileInfo, err := os.Stat(req.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Initialize metadata
	metadata := make(map[string]interface{})
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			metadata[k] = v
		}
	}

	// Add PDF-specific metadata
	metadata["file_name"] = filepath.Base(req.FilePath)
	metadata["file_size"] = fileInfo.Size()
	metadata["file_modified"] = fileInfo.ModTime()
	metadata["page_count"] = r.NumPage()

	return &ProcessedDocument{
		ID:          req.DocumentID,
		Content:     content.String(),
		ContentType: req.ContentType,
		Metadata:    metadata,
		Size:        fileInfo.Size(),
		ProcessedAt: time.Now(),
	}, nil
}

func (p *PDFProcessor) SupportedTypes() []string {
	return []string{"application/pdf"}
}

// RawProcessor handles raw content input.
type RawProcessor struct{}

func (p *RawProcessor) Process(ctx context.Context, req core.IngestRequest) (*ProcessedDocument, error) {
	if req.Content == "" {
		return nil, fmt.Errorf("raw processing requires content")
	}

	// Initialize metadata
	metadata := make(map[string]interface{})
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			metadata[k] = v
		}
	}

	return &ProcessedDocument{
		ID:          req.DocumentID,
		Content:     req.Content,
		ContentType: req.ContentType,
		Metadata:    metadata,
		Size:        int64(len(req.Content)),
		ProcessedAt: time.Now(),
	}, nil
}

func (p *RawProcessor) SupportedTypes() []string {
	return []string{"text/raw", "application/text"}
}

// ===== TYPE CONVERTERS =====

// convertToStorageDocument converts from ingest ProcessedDocument to storage Document.
func convertToStorageDocument(processed *ProcessedDocument) *storage.Document {
	return &storage.Document{
		ID:          processed.ID,
		Content:     processed.Content,
		ContentType: processed.ContentType,
		Metadata:    processed.Metadata,
		Size:        processed.Size,
		CreatedAt:   processed.ProcessedAt,
		UpdatedAt:   processed.ProcessedAt,
		Version:     1,
	}
}

// convertToStorageChunks converts from ingest TextChunks to storage TextChunks.
func convertToStorageChunks(chunks []TextChunk) []storage.TextChunk {
	storageChunks := make([]storage.TextChunk, len(chunks))
	for i, chunk := range chunks {
		storageChunks[i] = storage.TextChunk{
			ID:       chunk.ID,
			Content:  chunk.Content,
			Metadata: chunk.Metadata,
			Position: chunk.Position,
		}
	}
	return storageChunks
}