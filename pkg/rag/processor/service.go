package processor

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	pdf "github.com/dslipak/pdf"
	"github.com/google/uuid"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/providers"
)

type Service struct {
	embedder        domain.Embedder
	generator       domain.Generator
	chunker         domain.Chunker
	vectorStore     domain.VectorStore
	documentStore   domain.DocumentStore
	config          *config.Config
	llmService      domain.MetadataExtractor
}

func New(
	embedder domain.Embedder,
	generator domain.Generator,
	chunker domain.Chunker,
	vectorStore domain.VectorStore,
	documentStore domain.DocumentStore,
	config *config.Config,
	llmService domain.MetadataExtractor,
) *Service {
	s := &Service{
		embedder:      embedder,
		generator:     generator,
		chunker:       chunker,
		vectorStore:   vectorStore,
		documentStore: documentStore,
		config:        config,
		llmService:    llmService,
	}


	return s
}

// initializeTools sets up the tool system

func (s *Service) Ingest(ctx context.Context, req domain.IngestRequest) (domain.IngestResponse, error) {
	if err := s.validateIngestRequest(req); err != nil {
		return domain.IngestResponse{}, err
	}

	content, err := s.extractContent(req)
	if err != nil {
		return domain.IngestResponse{}, err
	}

	if content == "" {
		return domain.IngestResponse{
				Success: false,
				Message: "no content found",
			},
			nil
	}

	// Initialize metadata map if it's nil
	if req.Metadata == nil {
		req.Metadata = make(map[string]interface{})
	}

	// Automatic metadata extraction
	if s.config.Ingest.MetadataExtraction.Enable {
		log.Println("Enhanced metadata extraction enabled, calling LLM...")
		extracted, err := s.llmService.ExtractMetadata(ctx, content, s.config.Ingest.MetadataExtraction.LLMModel)
		if err != nil {
			log.Printf("Warning: metadata extraction failed, proceeding without it. Error: %v", err)
		} else {
			log.Printf("Successfully extracted metadata: %+v", extracted)
			s.mergeMetadata(req.Metadata, extracted)
		}
	}

	// Fallback for creation_date
	if _, ok := req.Metadata["creation_date"]; !ok || req.Metadata["creation_date"] == nil || req.Metadata["creation_date"] == "" {
		s.addFileCreationDate(req.FilePath, req.Metadata)
	}

	doc := domain.Document{
		ID:       uuid.New().String(),
		Path:     req.FilePath,
		URL:      req.URL,
		Content:  content, // Storing full content might be redundant, consider trade-offs
		Metadata: req.Metadata,
		Created:  time.Now(),
	}

	chunkOptions := domain.ChunkOptions{
		Size:    req.ChunkSize,
		Overlap: req.Overlap,
		Method:  "sentence",
	}

	if req.ChunkSize <= 0 {
		chunkOptions.Size = s.config.Chunker.ChunkSize
	}
	if req.Overlap < 0 {
		chunkOptions.Overlap = s.config.Chunker.Overlap
	}

	textChunks, err := s.chunker.Split(content, chunkOptions)
	if err != nil {
		return domain.IngestResponse{}, fmt.Errorf("failed to chunk text: %w", err)
	}

	var chunks []domain.Chunk
	for i, textChunk := range textChunks {
		vector, err := s.embedder.Embed(ctx, textChunk)
		if err != nil {
			return domain.IngestResponse{}, fmt.Errorf("failed to generate embedding for chunk %d: %w", i, err)
		}

		chunk := domain.Chunk{
			ID:         fmt.Sprintf("%s_%d", doc.ID, i),
			DocumentID: doc.ID,
			Content:    textChunk,
			Vector:     vector,
			Metadata:   doc.Metadata, // Pass down the combined metadata to each chunk
		}
		chunks = append(chunks, chunk)

	}

	if err := s.vectorStore.Store(ctx, chunks); err != nil {
		return domain.IngestResponse{}, fmt.Errorf("failed to store vectors: %w", err)
	}

	// Store document metadata after vectors are stored (sqvect v0.7.0 needs vectors first)
	if err := s.documentStore.Store(ctx, doc); err != nil {
		return domain.IngestResponse{}, fmt.Errorf("failed to store document: %w", err)
	}

	return domain.IngestResponse{
			Success:    true,
			DocumentID: doc.ID,
			ChunkCount: len(chunks),
			Message:    fmt.Sprintf("Successfully ingested %d chunks", len(chunks)),
		},
		nil
}

// mergeMetadata merges the extracted metadata into the request's metadata map.
func (s *Service) mergeMetadata(base map[string]interface{}, extracted *domain.ExtractedMetadata) {
	if extracted.Summary != "" {
		base["summary"] = extracted.Summary
	}
	if len(extracted.Keywords) > 0 {
		base["keywords"] = extracted.Keywords
	}
	if extracted.DocumentType != "" {
		base["document_type"] = extracted.DocumentType
	}
	if extracted.CreationDate != "" {
		base["creation_date"] = extracted.CreationDate
	}
	if extracted.Collection != "" {
		base["collection"] = extracted.Collection
	}
	
	// Merge enhanced fields
	if len(extracted.TemporalRefs) > 0 {
		base["temporal_refs"] = extracted.TemporalRefs
	}
	if len(extracted.Entities) > 0 {
		base["entities"] = extracted.Entities
	}
	if len(extracted.Events) > 0 {
		base["events"] = extracted.Events
	}
	if len(extracted.CustomMeta) > 0 {
		for k, v := range extracted.CustomMeta {
			base[k] = v
		}
	}
}

// addFileCreationDate adds the file's modification time as a fallback creation date.
func (s *Service) addFileCreationDate(filePath string, metadata map[string]interface{}) {
	if filePath == "" {
		return
	}
	fileInfo, err := os.Stat(filePath)
	if err == nil {
		metadata["creation_date"] = fileInfo.ModTime().Format("2006-01-02")
	}
}

func (s *Service) Query(ctx context.Context, req domain.QueryRequest) (domain.QueryResponse, error) {
	start := time.Now()

	if req.Query == "" {
		return domain.QueryResponse{}, fmt.Errorf("%w: empty query", domain.ErrInvalidInput)
	}

	chunks, err := s.hybridSearch(ctx, req)
	if err != nil {
		return domain.QueryResponse{}, err
	}

	if len(chunks) == 0 {
		return domain.QueryResponse{
				Answer:  "很抱歉，我在知识库中找不到相关信息来回答您的问题。",
				Sources: []domain.Chunk{},
				Elapsed: time.Since(start).String(),
			},
			nil
	}

	prompt := providers.ComposePrompt(chunks, req.Query)

	genOpts := &domain.GenerationOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	if genOpts.Temperature <= 0 {
		genOpts.Temperature = 0.7
	}
	if genOpts.MaxTokens <= 0 {
		genOpts.MaxTokens = 25000
	}

	answer, err := s.generator.Generate(ctx, prompt, genOpts)
	if err != nil {
		return domain.QueryResponse{}, fmt.Errorf("failed to generate answer: %w", err)
	}

	// Clean up internal thinking tags from the answer only if ShowThinking is false
	if !req.ShowThinking {
		answer = s.cleanThinkingTags(answer)
	}

	// Prepare sources based on ShowSources flag
	var sources []domain.Chunk
	if req.ShowSources {
		sources = chunks
	} else {
		sources = []domain.Chunk{}
	}

	return domain.QueryResponse{
			Answer:  answer,
			Sources: sources,
			Elapsed: time.Since(start).String(),
		},
		nil
}

// QueryWithTools processes a query with tool calling support
// QueryWithTools - deprecated, use MCP instead
func (s *Service) QueryWithTools(ctx context.Context, req domain.QueryRequest) (domain.QueryResponse, error) {
	return s.Query(ctx, req)
}

// StreamQueryWithTools processes a streaming query with tool calling support
// StreamQueryWithTools - deprecated, use MCP instead
func (s *Service) StreamQueryWithTools(ctx context.Context, req domain.QueryRequest, callback func(string)) error {
	return s.StreamQuery(ctx, req, callback)
}

// getAvailableTools returns the list of available tools based on allowed list
// getAvailableTools - deprecated
func (s *Service) getAvailableTools(allowedTools []string) []interface{} {
	return nil
}

// buildPromptWithContext builds a prompt with RAG context
func (s *Service) buildPromptWithContext(query string, chunks []domain.Chunk) string {
	if len(chunks) == 0 {
		return fmt.Sprintf(`Please answer the user's question: %s

If you need current information (like time, date, weather, file contents, web data, etc.), use the available tools to get accurate and up-to-date information.`, query)
	}

	var contextParts []string
	for i, chunk := range chunks {
		contextParts = append(contextParts, fmt.Sprintf("[Document %d]\n%s", i+1, chunk.Content))
	}

	context := strings.Join(contextParts, "\n\n")

	return fmt.Sprintf(`Please answer the user's question using the following context AND any available tools when needed.

IMPORTANT INSTRUCTIONS:
1. For questions about current information (time, date, weather, file contents, web data, etc.), always use the appropriate tools to get accurate and up-to-date information.
2. For questions about stored knowledge, use the provided context documents.
3. If both context and tools are relevant, combine information from both sources.

Context Documents:
%s

User Question: %s

Please provide a comprehensive answer using both the context documents and tools as appropriate.`, context, query)
}

func (s *Service) StreamQuery(ctx context.Context, req domain.QueryRequest, callback func(string)) error {
	if req.Query == "" {
		return fmt.Errorf("%w: empty query", domain.ErrInvalidInput)
	}

	if callback == nil {
		return fmt.Errorf("%w: nil callback", domain.ErrInvalidInput)
	}

	chunks, err := s.hybridSearch(ctx, req)
	if err != nil {
		return err
	}

	if len(chunks) == 0 {
		callback("很抱歉，我在知识库中找不到相关信息来回答您的问题。")
		return nil
	}

	prompt := providers.ComposePrompt(chunks, req.Query)

	genOpts := &domain.GenerationOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	if genOpts.Temperature <= 0 {
		genOpts.Temperature = 0.7
	}
	if genOpts.MaxTokens <= 0 {
		genOpts.MaxTokens = 25000
	}

	return s.generator.Stream(ctx, prompt, genOpts, s.wrapCallbackForThinking(callback, req.ShowThinking))
}

func (s *Service) hybridSearch(ctx context.Context, req domain.QueryRequest) ([]domain.Chunk, error) {
	if req.TopK <= 0 {
		req.TopK = 5
	}

	// Simple vector search only
	queryVector, err := s.embedder.Embed(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	var chunks []domain.Chunk
	if len(req.Filters) > 0 {
		chunks, err = s.vectorStore.SearchWithFilters(ctx, queryVector, req.TopK, req.Filters)
	} else {
		chunks, err = s.vectorStore.Search(ctx, queryVector, req.TopK)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to search vectors: %w", err)
	}

	return s.deduplicateChunks(chunks), nil
}

func (s *Service) ListDocuments(ctx context.Context) ([]domain.Document, error) {
	return s.documentStore.List(ctx)
}

func (s *Service) DeleteDocument(ctx context.Context, documentID string) error {
	if documentID == "" {
		return fmt.Errorf("%w: empty document ID", domain.ErrInvalidInput)
	}

	if err := s.vectorStore.Delete(ctx, documentID); err != nil {
		return fmt.Errorf("failed to delete from vector store: %w", err)
	}

	if err := s.documentStore.Delete(ctx, documentID); err != nil {
		return fmt.Errorf("failed to delete from document store: %w", err)
	}

	return nil
}

func (s *Service) Reset(ctx context.Context) error {
	// Reset vector store (Qdrant)
	if err := s.vectorStore.Reset(ctx); err != nil {
		return fmt.Errorf("failed to reset vector store: %w", err)
	}

	// Reset document store (SQLite) if it has a Reset method
	if s.documentStore != nil {
		if resetter, ok := s.documentStore.(interface{ Reset(context.Context) error }); ok {
			if err := resetter.Reset(ctx); err != nil {
				return fmt.Errorf("failed to reset document store: %w", err)
			}
		}
	}

	return nil
}

func (s *Service) validateIngestRequest(req domain.IngestRequest) error {
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
		return fmt.Errorf("%w: no content source provided", domain.ErrInvalidInput)
	}

	if contentSources > 1 {
		return fmt.Errorf("%w: multiple content sources provided", domain.ErrInvalidInput)
	}

	return nil
}

func (s *Service) extractContent(req domain.IngestRequest) (string, error) {
	if req.Content != "" {
		return req.Content, nil
	}

	if req.FilePath != "" {
		return s.readFile(req.FilePath)
	}

	if req.URL != "" {
		return "", fmt.Errorf("URL content extraction not yet implemented")
	}

	return "", fmt.Errorf("%w: no content source", domain.ErrInvalidInput)
}

func (s *Service) readFile(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".txt", ".md", ".markdown":
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
		}
		return string(data), nil

	case ".pdf":
		r, err := pdf.Open(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to open PDF %s: %w", filePath, err)
		}
		var buf strings.Builder
		for i := 1; i <= r.NumPage(); i++ {
			p := r.Page(i)
			if p.V.IsNull() {
				continue
			}
			text, err := p.GetPlainText(nil)
			if err != nil {
				// Log a warning but continue processing other pages
				log.Printf("Warning: failed to get text from page %d of %s: %v", i, filePath, err)
				continue
			}
			buf.WriteString(text)
			buf.WriteString("\n") // Add a newline between pages
		}
		return buf.String(), nil

	default:
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
}

// deduplicateChunks removes duplicate chunks by content to avoid confusion
func (s *Service) deduplicateChunks(chunks []domain.Chunk) []domain.Chunk {
	seen := make(map[string]bool)
	result := make([]domain.Chunk, 0, len(chunks))

	for _, chunk := range chunks {
		// Use content as the key for deduplication
		if !seen[chunk.Content] {
			seen[chunk.Content] = true
			result = append(result, chunk)
		}
	}

	return result
}

// cleanThinkingTags removes internal thinking tags from LLM responses
func (s *Service) cleanThinkingTags(answer string) string {
	// Remove <think>...</think> blocks and their contents
	re := strings.NewReplacer("<think>", "", "</think>", "")
	cleaned := re.Replace(answer)

	// Also handle the case where thinking tags might span multiple lines
	if strings.Contains(cleaned, "<think") || strings.Contains(cleaned, "</think") {
		// Use regex for more complex cases
		lines := strings.Split(cleaned, "\n")
		var filtered []string
		inThinking := false

		for _, line := range lines {
			if strings.Contains(line, "<think") {
				inThinking = true
				continue
			}
			if strings.Contains(line, "</think") {
				inThinking = false
				continue
			}
			if !inThinking {
				filtered = append(filtered, line)
			}
		}
		cleaned = strings.Join(filtered, "\n")
	}

	// Trim any extra whitespace
	return strings.TrimSpace(cleaned)
}

// wrapCallbackForThinking wraps the callback to filter thinking tags in streaming mode
func (s *Service) wrapCallbackForThinking(callback func(string), showThinking bool) func(string) {
	if showThinking {
		// If showing thinking, just pass through
		return callback
	}

	// If not showing thinking, filter out thinking content
	var buffer strings.Builder
	inThinking := false

	return func(token string) {
		buffer.WriteString(token)
		content := buffer.String()

		// Process complete thinking blocks
		for {
			if !inThinking {
				// Look for start of thinking block
				if idx := strings.Index(content, "<think>"); idx != -1 {
					// Send content before thinking block
					if idx > 0 {
						callback(content[:idx])
					}
					inThinking = true
					content = content[idx+7:] // Skip "<think>"
					buffer.Reset()
					buffer.WriteString(content)
					continue
				} else {
					// No thinking block start, send everything
					if content != "" {
						callback(content)
						buffer.Reset()
					}
					break
				}
			} else {
				// Look for end of thinking block
				if idx := strings.Index(content, "</think>"); idx != -1 {
					inThinking = false
					content = content[idx+8:] // Skip "</think>"
					buffer.Reset()
					buffer.WriteString(content)
					continue
				} else {
					// Still in thinking block, don't send anything
					break
				}
			}
		}
	}
}

// GetToolRegistry returns the tool registry if tools are enabled
// GetToolRegistry - deprecated
func (s *Service) GetToolRegistry() interface{} {
	return nil
}

// GetToolExecutor returns the tool executor if tools are enabled
// GetToolExecutor - deprecated
func (s *Service) GetToolExecutor() interface{} {
	return nil
}

// RegisterMCPTools registers MCP tools with the processor
// RegisterMCPTools - deprecated
func (s *Service) RegisterMCPTools(mcpService interface{}) error {
	return fmt.Errorf("tools have been removed - use MCP servers directly")
}

