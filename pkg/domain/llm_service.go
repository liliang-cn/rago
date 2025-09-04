package domain

import "context"

// MetadataExtractor defines the interface for metadata extraction functionality
type MetadataExtractor interface {
	ExtractMetadata(ctx context.Context, content string, model string) (*ExtractedMetadata, error)
}

// LLMService combines the Generator interface with metadata extraction capabilities
type LLMService interface {
	Generator
	MetadataExtractor
}
