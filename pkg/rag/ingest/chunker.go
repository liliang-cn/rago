package ingest

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/liliang-cn/rago/v2/pkg/core"
)

// DefaultChunker implements the Chunker interface with multiple chunking strategies.
type DefaultChunker struct {
	config core.ChunkingConfig
}

// NewChunker creates a new chunker with the specified configuration.
func NewChunker(config core.ChunkingConfig) (Chunker, error) {
	return &DefaultChunker{
		config: config,
	}, nil
}

// Chunk splits text into chunks based on the configured strategy.
func (c *DefaultChunker) Chunk(ctx context.Context, text string, options ChunkOptions) ([]TextChunk, error) {
	if text == "" {
		return []TextChunk{}, nil
	}

	// Use options or fall back to config defaults
	strategy := options.Strategy
	if strategy == "" {
		strategy = c.config.Strategy
	}

	chunkSize := options.ChunkSize
	if chunkSize <= 0 {
		chunkSize = c.config.ChunkSize
	}

	chunkOverlap := options.ChunkOverlap
	if chunkOverlap < 0 {
		chunkOverlap = c.config.ChunkOverlap
	}

	minChunkSize := options.MinChunkSize
	if minChunkSize <= 0 {
		minChunkSize = c.config.MinChunkSize
	}

	switch strategy {
	case "sentence":
		return c.chunkBySentence(text, chunkSize, chunkOverlap, minChunkSize)
	case "paragraph":
		return c.chunkByParagraph(text, chunkSize, chunkOverlap, minChunkSize)
	case "fixed":
		return c.chunkByFixed(text, chunkSize, chunkOverlap, minChunkSize)
	case "semantic":
		// TODO: Implement semantic chunking using embeddings
		return c.chunkBySentence(text, chunkSize, chunkOverlap, minChunkSize)
	default:
		return nil, fmt.Errorf("unsupported chunking strategy: %s", strategy)
	}
}

// chunkBySentence splits text by sentences and combines them into appropriately sized chunks.
func (c *DefaultChunker) chunkBySentence(text string, chunkSize, chunkOverlap, minChunkSize int) ([]TextChunk, error) {
	sentences := c.splitIntoSentences(text)
	return c.combineIntoChunks(sentences, chunkSize, chunkOverlap, minChunkSize), nil
}

// chunkByParagraph splits text by paragraphs, then by sentences within paragraphs.
func (c *DefaultChunker) chunkByParagraph(text string, chunkSize, chunkOverlap, minChunkSize int) ([]TextChunk, error) {
	paragraphs := c.splitIntoParagraphs(text)
	var sentences []string
	for _, para := range paragraphs {
		sentences = append(sentences, c.splitIntoSentences(para)...)
	}
	return c.combineIntoChunks(sentences, chunkSize, chunkOverlap, minChunkSize), nil
}

// chunkByFixed splits text into fixed-size chunks by character count.
func (c *DefaultChunker) chunkByFixed(text string, chunkSize, chunkOverlap, minChunkSize int) ([]TextChunk, error) {
	var chunks []TextChunk
	textRunes := []rune(text)
	totalLength := len(textRunes)
	
	if totalLength <= chunkSize {
		// Text fits in a single chunk
		return []TextChunk{{
			ID:       "chunk_0",
			Content:  text,
			Metadata: make(map[string]interface{}),
			Position: 0,
		}}, nil
	}
	
	position := 0
	chunkIndex := 0
	
	for position < totalLength {
		// Calculate chunk end position
		chunkEnd := position + chunkSize
		if chunkEnd > totalLength {
			chunkEnd = totalLength
		}
		
		// Extract chunk content
		chunkContent := string(textRunes[position:chunkEnd])
		
		// Skip chunks that are too small (except the last one)
		if len(strings.TrimSpace(chunkContent)) >= minChunkSize || chunkEnd == totalLength {
			chunk := TextChunk{
				ID:       fmt.Sprintf("chunk_%d", chunkIndex),
				Content:  strings.TrimSpace(chunkContent),
				Metadata: map[string]interface{}{
					"chunk_index": chunkIndex,
					"start_pos":   position,
					"end_pos":     chunkEnd,
				},
				Position: chunkIndex,
			}
			chunks = append(chunks, chunk)
			chunkIndex++
		}
		
		// Move position forward, accounting for overlap
		position += chunkSize - chunkOverlap
		if position >= totalLength {
			break
		}
	}
	
	return chunks, nil
}

// splitIntoSentences splits text into individual sentences.
func (c *DefaultChunker) splitIntoSentences(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{}
	}

	var sentences []string
	var current strings.Builder

	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		current.WriteRune(r)

		if c.isSentenceEnd(r) {
			// For sentence ending punctuation, check if this is actually end of sentence
			nextCharExists := i+1 < len(runes)

			isEnd := false
			if !nextCharExists {
				isEnd = true
			} else {
				nextChar := runes[i+1]
				if unicode.IsSpace(nextChar) || unicode.IsUpper(nextChar) || c.isSentenceEnd(nextChar) {
					isEnd = true
				} else {
					// For CJK characters, consider it sentence end unless next char is punctuation
					if c.isCJK(r) || c.isCJK(nextChar) {
						isEnd = !unicode.IsPunct(nextChar) || c.isSentenceEnd(nextChar)
					}
				}
			}

			if isEnd {
				sentence := strings.TrimSpace(current.String())
				if sentence != "" {
					sentences = append(sentences, sentence)
				}
				current.Reset()

				// Skip whitespace after sentence end
				for i+1 < len(runes) && unicode.IsSpace(runes[i+1]) {
					i++
				}
			}
		}
	}

	// Handle any remaining content
	if current.Len() > 0 {
		sentence := strings.TrimSpace(current.String())
		if sentence != "" {
			sentences = append(sentences, sentence)
		}
	}

	return sentences
}

// splitIntoParagraphs splits text into paragraphs.
func (c *DefaultChunker) splitIntoParagraphs(text string) []string {
	// Split by double newlines (common paragraph separator)
	paragraphs := strings.Split(text, "\n\n")
	
	var result []string
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para != "" {
			result = append(result, para)
		}
	}
	
	return result
}

// combineIntoChunks combines sentences into chunks of appropriate size.
func (c *DefaultChunker) combineIntoChunks(sentences []string, chunkSize, chunkOverlap, minChunkSize int) []TextChunk {
	if len(sentences) == 0 {
		return []TextChunk{}
	}

	var chunks []TextChunk
	var currentChunk strings.Builder
	var chunkSentences []string
	chunkIndex := 0

	for i, sentence := range sentences {
		// Check if adding this sentence would exceed the chunk size
		potentialLength := currentChunk.Len()
		if potentialLength > 0 {
			potentialLength++ // for space
		}
		potentialLength += len(sentence)

		if potentialLength > chunkSize && currentChunk.Len() > 0 {
			// Current chunk is full, create it
			content := strings.TrimSpace(currentChunk.String())
			if len(content) >= minChunkSize {
				chunk := TextChunk{
					ID:       fmt.Sprintf("chunk_%d", chunkIndex),
					Content:  content,
					Metadata: map[string]interface{}{
						"chunk_index":     chunkIndex,
						"sentence_count":  len(chunkSentences),
						"sentence_start":  i - len(chunkSentences),
						"sentence_end":    i - 1,
					},
					Position: chunkIndex,
				}
				chunks = append(chunks, chunk)
				chunkIndex++
			}

			// Start new chunk with overlap
			currentChunk.Reset()
			chunkSentences = []string{}

			// Add overlap sentences
			overlapSentences := c.calculateOverlapSentences(sentences, i, chunkOverlap)
			for j, overlapSent := range overlapSentences {
				if j > 0 {
					currentChunk.WriteString(" ")
				}
				currentChunk.WriteString(overlapSent)
				chunkSentences = append(chunkSentences, overlapSent)
			}
		}

		// Add current sentence
		if currentChunk.Len() > 0 {
			currentChunk.WriteString(" ")
		}
		currentChunk.WriteString(sentence)
		chunkSentences = append(chunkSentences, sentence)
	}

	// Handle the last chunk
	if currentChunk.Len() > 0 {
		content := strings.TrimSpace(currentChunk.String())
		if len(content) >= minChunkSize {
			chunk := TextChunk{
				ID:       fmt.Sprintf("chunk_%d", chunkIndex),
				Content:  content,
				Metadata: map[string]interface{}{
					"chunk_index":     chunkIndex,
					"sentence_count":  len(chunkSentences),
					"sentence_start":  len(sentences) - len(chunkSentences),
					"sentence_end":    len(sentences) - 1,
				},
				Position: chunkIndex,
			}
			chunks = append(chunks, chunk)
		}
	}

	return chunks
}

// calculateOverlapSentences determines which sentences to include for overlap.
func (c *DefaultChunker) calculateOverlapSentences(sentences []string, currentIndex, overlapSize int) []string {
	if overlapSize <= 0 || currentIndex <= 0 {
		return []string{}
	}

	// Calculate how many sentences to include based on character overlap
	var overlap []string
	overlapLength := 0
	
	for i := currentIndex - 1; i >= 0 && overlapLength < overlapSize; i-- {
		sentence := sentences[i]
		if overlapLength + len(sentence) <= overlapSize {
			overlap = append([]string{sentence}, overlap...)
			overlapLength += len(sentence)
		} else {
			break
		}
	}
	
	return overlap
}

// isSentenceEnd checks if a rune is a sentence-ending punctuation.
func (c *DefaultChunker) isSentenceEnd(r rune) bool {
	return r == '.' || r == '!' || r == '?' || r == '。' || r == '！' || r == '？'
}

// isCJK checks if a rune is a CJK (Chinese, Japanese, Korean) character.
func (c *DefaultChunker) isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
		(r >= 0x3400 && r <= 0x4DBF) || // CJK Extension A
		(r >= 0x20000 && r <= 0x2A6DF) || // CJK Extension B
		(r >= 0x2A700 && r <= 0x2B73F) || // CJK Extension C
		(r >= 0x2B740 && r <= 0x2B81F) || // CJK Extension D
		(r >= 0x3040 && r <= 0x309F) || // Hiragana
		(r >= 0x30A0 && r <= 0x30FF) || // Katakana
		(r >= 0xAC00 && r <= 0xD7AF) // Hangul Syllables
}