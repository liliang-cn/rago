package rag

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// Service provides RAG (Retrieval-Augmented Generation) functionality
type Service struct {
	store         domain.VectorStore
	embedder      domain.EmbedderProvider
	relevanceThreshold float64 // Minimum similarity score for relevance
}

// NewService creates a new RAG service
func NewService(store domain.VectorStore, embedder domain.EmbedderProvider) *Service {
	return &Service{
		store:         store,
		embedder:      embedder,
		relevanceThreshold: 0.7, // Default threshold - documents below this are considered irrelevant
	}
}

// NewServiceWithThreshold creates a new RAG service with custom relevance threshold
func NewServiceWithThreshold(store domain.VectorStore, embedder domain.EmbedderProvider, threshold float64) *Service {
	return &Service{
		store:         store,
		embedder:      embedder,
		relevanceThreshold: threshold,
	}
}

// SearchResult represents a search result with relevance information
type SearchResult struct {
	Chunk      *domain.Chunk
	Score      float64 // Similarity score (0-1, higher is better)
	IsRelevant bool    // Whether this result meets the relevance threshold
}

// SearchOptions configures the search behavior
type SearchOptions struct {
	MaxResults         int     // Maximum number of results to return
	MinRelevanceScore  float64 // Minimum score for a result to be considered relevant
	IncludeIrrelevant  bool    // If true, return all results even if below threshold
}

// DefaultSearchOptions returns sensible defaults
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		MaxResults:        5,
		MinRelevanceScore: 0.7,
		IncludeIrrelevant: false,
	}
}

// SearchWithRelevance performs semantic search and filters by relevance
func (s *Service) SearchWithRelevance(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error) {
	if s.store == nil || s.embedder == nil {
		return nil, fmt.Errorf("RAG service not properly initialized")
	}

	// Generate embeddings for the query
	embeddings, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Search for chunks
	chunks, err := s.store.Search(ctx, embeddings, opts.MaxResults*2) // Get more to filter
	if err != nil {
		return nil, fmt.Errorf("failed to search chunks: %w", err)
	}

	// Calculate similarity scores and filter by relevance
	var results []SearchResult
	for _, chunk := range chunks {
		// Calculate cosine similarity if we have chunk embeddings
		// For now, we'll use a simple heuristic based on content matching
		score := s.calculateRelevanceScore(query, &chunk)
		
		isRelevant := score >= opts.MinRelevanceScore
		
		if isRelevant || opts.IncludeIrrelevant {
			results = append(results, SearchResult{
				Chunk:      &chunk,
				Score:      score,
				IsRelevant: isRelevant,
			})
		}
	}

	// Sort by score (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit to MaxResults
	if len(results) > opts.MaxResults {
		results = results[:opts.MaxResults]
	}

	return results, nil
}

// GetRelevantContext retrieves relevant context for a query
func (s *Service) GetRelevantContext(ctx context.Context, query string, maxDocs int) (string, int, error) {
	opts := SearchOptions{
		MaxResults:        maxDocs,
		MinRelevanceScore: s.relevanceThreshold,
		IncludeIrrelevant: false,
	}

	results, err := s.SearchWithRelevance(ctx, query, opts)
	if err != nil {
		return "", 0, err
	}

	// Only include relevant results
	var relevantDocs []SearchResult
	for _, r := range results {
		if r.IsRelevant {
			relevantDocs = append(relevantDocs, r)
		}
	}

	if len(relevantDocs) == 0 {
		return "", 0, nil // No relevant context found
	}

	// Build context string
	var contexts []string
	for _, result := range relevantDocs {
		source := "unknown"
		if result.Chunk.Metadata != nil && result.Chunk.Metadata["source"] != nil {
			source = fmt.Sprintf("%v", result.Chunk.Metadata["source"])
		}
		
		// Include score in context for transparency
		contexts = append(contexts, fmt.Sprintf(
			"[Source: %s | Relevance: %.2f]\n%s",
			source,
			result.Score,
			result.Chunk.Content,
		))
	}

	contextStr := strings.Join(contexts, "\n\n---\n\n")
	return contextStr, len(relevantDocs), nil
}

// calculateRelevanceScore calculates a simple relevance score between query and chunk
// This is a placeholder - in production, you'd use proper vector similarity
func (s *Service) calculateRelevanceScore(query string, chunk *domain.Chunk) float64 {
	// Convert to lowercase for comparison
	queryLower := strings.ToLower(query)
	contentLower := strings.ToLower(chunk.Content)
	
	// Split query into words
	queryWords := strings.Fields(queryLower)
	if len(queryWords) == 0 {
		return 0.0
	}
	
	// Count matching words
	matchCount := 0
	for _, word := range queryWords {
		if len(word) > 2 && strings.Contains(contentLower, word) {
			matchCount++
		}
	}
	
	// Calculate score based on match ratio
	score := float64(matchCount) / float64(len(queryWords))
	
	// Boost score if query appears as exact substring
	if strings.Contains(contentLower, queryLower) {
		score = math.Min(score*1.5, 1.0)
	}
	
	// Add small random component to simulate vector similarity variance
	// In production, this would be actual cosine similarity from embeddings
	score = score * 0.9 + 0.1*0.5 // Adding 0.05 base score
	
	return math.Min(score, 1.0)
}

// SetRelevanceThreshold updates the relevance threshold
func (s *Service) SetRelevanceThreshold(threshold float64) {
	s.relevanceThreshold = math.Max(0.0, math.Min(1.0, threshold))
}

// IsAvailable checks if RAG service is available
func (s *Service) IsAvailable() bool {
	return s.store != nil && s.embedder != nil
}