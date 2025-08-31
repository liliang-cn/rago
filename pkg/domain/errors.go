package domain

import "errors"

var (
	ErrDocumentNotFound    = errors.New("document not found")
	ErrInvalidInput        = errors.New("invalid input")
	ErrEmbeddingFailed     = errors.New("embedding generation failed")
	ErrGenerationFailed    = errors.New("text generation failed")
	ErrChunkingFailed      = errors.New("text chunking failed")
	ErrVectorStoreFailed   = errors.New("vector store operation failed")
	ErrDocumentStoreFailed = errors.New("document store operation failed")
	ErrConfigurationError  = errors.New("configuration error")
	ErrServiceUnavailable  = errors.New("service unavailable")
)
