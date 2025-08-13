package processor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/liliang-cn/rago/internal/domain"
	"github.com/liliang-cn/rago/internal/llm"
)

type Service struct {
	embedder      domain.Embedder
	generator     domain.Generator
	chunker       domain.Chunker
	vectorStore   domain.VectorStore
	documentStore domain.DocumentStore
}

func New(
	embedder domain.Embedder,
	generator domain.Generator,
	chunker domain.Chunker,
	vectorStore domain.VectorStore,
	documentStore domain.DocumentStore,
) *Service {
	return &Service{
		embedder:      embedder,
		generator:     generator,
		chunker:       chunker,
		vectorStore:   vectorStore,
		documentStore: documentStore,
	}
}

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
		}, nil
	}

	doc := domain.Document{
		ID:       uuid.New().String(),
		Path:     req.FilePath,
		URL:      req.URL,
		Content:  content,
		Metadata: req.Metadata,
		Created:  time.Now(),
	}

	if err := s.documentStore.Store(ctx, doc); err != nil {
		return domain.IngestResponse{}, fmt.Errorf("failed to store document: %w", err)
	}

	chunkOptions := domain.ChunkOptions{
		Size:    req.ChunkSize,
		Overlap: req.Overlap,
		Method:  "sentence",
	}

	if req.ChunkSize <= 0 {
		chunkOptions.Size = 300
	}
	if req.Overlap < 0 {
		chunkOptions.Overlap = 50
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
			Metadata:   req.Metadata,
		}
		chunks = append(chunks, chunk)
	}

	if err := s.vectorStore.Store(ctx, chunks); err != nil {
		return domain.IngestResponse{}, fmt.Errorf("failed to store vectors: %w", err)
	}

	return domain.IngestResponse{
		Success:    true,
		DocumentID: doc.ID,
		ChunkCount: len(chunks),
		Message:    fmt.Sprintf("Successfully ingested %d chunks", len(chunks)),
	}, nil
}

func (s *Service) Query(ctx context.Context, req domain.QueryRequest) (domain.QueryResponse, error) {
	start := time.Now()

	if req.Query == "" {
		return domain.QueryResponse{}, fmt.Errorf("%w: empty query", domain.ErrInvalidInput)
	}

	if req.TopK <= 0 {
		req.TopK = 5
	}

	queryVector, err := s.embedder.Embed(ctx, req.Query)
	if err != nil {
		return domain.QueryResponse{}, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	var chunks []domain.Chunk
	if len(req.Filters) > 0 {
		chunks, err = s.vectorStore.SearchWithFilters(ctx, queryVector, req.TopK, req.Filters)
	} else {
		chunks, err = s.vectorStore.Search(ctx, queryVector, req.TopK)
	}
	
	if err != nil {
		return domain.QueryResponse{}, fmt.Errorf("failed to search vectors: %w", err)
	}

	if len(chunks) == 0 {
		return domain.QueryResponse{
			Answer:  "很抱歉，我在知识库中找不到相关信息来回答您的问题。",
			Sources: []domain.Chunk{},
			Elapsed: time.Since(start).String(),
		}, nil
	}

	prompt := llm.ComposePrompt(chunks, req.Query)

	genOpts := &domain.GenerationOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	if genOpts.Temperature <= 0 {
		genOpts.Temperature = 0.7
	}
	if genOpts.MaxTokens <= 0 {
		genOpts.MaxTokens = 500
	}

	answer, err := s.generator.Generate(ctx, prompt, genOpts)
	if err != nil {
		return domain.QueryResponse{}, fmt.Errorf("failed to generate answer: %w", err)
	}

	return domain.QueryResponse{
		Answer:  answer,
		Sources: chunks,
		Elapsed: time.Since(start).String(),
	}, nil
}

func (s *Service) StreamQuery(ctx context.Context, req domain.QueryRequest, callback func(string)) error {
	if req.Query == "" {
		return fmt.Errorf("%w: empty query", domain.ErrInvalidInput)
	}

	if callback == nil {
		return fmt.Errorf("%w: nil callback", domain.ErrInvalidInput)
	}

	if req.TopK <= 0 {
		req.TopK = 5
	}

	queryVector, err := s.embedder.Embed(ctx, req.Query)
	if err != nil {
		return fmt.Errorf("failed to generate query embedding: %w", err)
	}

	var chunks []domain.Chunk
	if len(req.Filters) > 0 {
		chunks, err = s.vectorStore.SearchWithFilters(ctx, queryVector, req.TopK, req.Filters)
	} else {
		chunks, err = s.vectorStore.Search(ctx, queryVector, req.TopK)
	}
	
	if err != nil {
		return fmt.Errorf("failed to search vectors: %w", err)
	}

	if len(chunks) == 0 {
		callback("很抱歉，我在知识库中找不到相关信息来回答您的问题。")
		return nil
	}

	prompt := llm.ComposePrompt(chunks, req.Query)

	genOpts := &domain.GenerationOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	if genOpts.Temperature <= 0 {
		genOpts.Temperature = 0.7
	}
	if genOpts.MaxTokens <= 0 {
		genOpts.MaxTokens = 500
	}

	return s.generator.Stream(ctx, prompt, genOpts, callback)
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
	if err := s.vectorStore.Reset(ctx); err != nil {
		return fmt.Errorf("failed to reset vector store: %w", err)
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
		return "", fmt.Errorf("PDF files not yet supported")
		
	default:
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
}