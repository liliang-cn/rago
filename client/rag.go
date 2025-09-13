package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/rag"
)

// IngestOptions configures how content is ingested - kept for client compatibility
type IngestOptions struct {
	ChunkSize          int                    // Size of text chunks
	Overlap            int                    // Overlap between chunks
	EnhancedExtraction bool                   // Enable enhanced metadata extraction
	Metadata           map[string]interface{} // Additional metadata
}

// QueryOptions configures query parameters - kept for advanced features
type QueryOptions struct {
	TopK         int                    // Number of top results to retrieve
	Temperature  float64                // Generation temperature
	MaxTokens    int                    // Maximum tokens in response
	ShowSources  bool                   // Include source documents
	ShowThinking bool                   // Show thinking process
	Filters      map[string]interface{} // Metadata filters
}

// ========================================
// Basic RAG Operations (delegated to RAG client)
// ========================================

// IngestFile ingests a file from the local filesystem
func (c *Client) IngestFile(filePath string) error {
	ctx := context.Background()
	_, err := c.ragClient.IngestFile(ctx, filePath, nil)
	return err
}

// IngestFileWithOptions ingests a file with custom options
func (c *Client) IngestFileWithOptions(filePath string, opts *IngestOptions) error {
	ctx := context.Background()
	var ragOpts *rag.IngestOptions
	if opts != nil {
		ragOpts = &rag.IngestOptions{
			ChunkSize:          opts.ChunkSize,
			Overlap:            opts.Overlap,
			Metadata:           opts.Metadata,
			EnhancedExtraction: opts.EnhancedExtraction,
		}
	}
	_, err := c.ragClient.IngestFile(ctx, filePath, ragOpts)
	return err
}

// IngestText ingests text content directly
func (c *Client) IngestText(text, source string) error {
	ctx := context.Background()
	_, err := c.ragClient.IngestText(ctx, text, source, nil)
	return err
}

// IngestTextWithMetadata ingests text content with additional metadata
func (c *Client) IngestTextWithMetadata(text, source string, additionalMetadata map[string]interface{}) error {
	ctx := context.Background()
	ragOpts := &rag.IngestOptions{
		Metadata: additionalMetadata,
	}
	_, err := c.ragClient.IngestText(ctx, text, source, ragOpts)
	return err
}

// IngestTextWithOptions ingests text content with custom options
func (c *Client) IngestTextWithOptions(text, source string, opts *IngestOptions) error {
	ctx := context.Background()
	var ragOpts *rag.IngestOptions
	if opts != nil {
		ragOpts = &rag.IngestOptions{
			ChunkSize:          opts.ChunkSize,
			Overlap:            opts.Overlap,
			Metadata:           opts.Metadata,
			EnhancedExtraction: opts.EnhancedExtraction,
		}
	}
	_, err := c.ragClient.IngestText(ctx, text, source, ragOpts)
	return err
}

// Query performs a simple query and returns the response
func (c *Client) Query(query string) (domain.QueryResponse, error) {
	ctx := context.Background()
	resp, err := c.ragClient.Query(ctx, query, nil)
	if err != nil {
		return domain.QueryResponse{}, err
	}
	return *resp, nil
}

// QueryWithSources performs a query with optional source information
func (c *Client) QueryWithSources(query string, showSources bool) (domain.QueryResponse, error) {
	ctx := context.Background()
	ragOpts := &rag.QueryOptions{
		ShowSources: showSources,
	}
	resp, err := c.ragClient.Query(ctx, query, ragOpts)
	if err != nil {
		return domain.QueryResponse{}, err
	}
	return *resp, nil
}

// QueryWithFilters performs a query with metadata filters
func (c *Client) QueryWithFilters(query string, filters map[string]interface{}) (domain.QueryResponse, error) {
	ctx := context.Background()
	// For now, filters need to be handled at the query level
	// This would require an enhancement to the RAG client
	resp, err := c.ragClient.Query(ctx, query, nil)
	if err != nil {
		return domain.QueryResponse{}, err
	}
	return *resp, nil
}

// StreamQuery performs a streaming query
func (c *Client) StreamQuery(query string, callback func(string)) error {
	ctx := context.Background()
	// StreamQuery not yet implemented in RAG client
	// Fall back to regular query and stream the response
	resp, err := c.ragClient.Query(ctx, query, nil)
	if err != nil {
		return err
	}
	// Stream the response in chunks
	words := strings.Fields(resp.Answer)
	for _, word := range words {
		callback(word + " ")
	}
	return nil
}

// StreamQueryWithSources performs a streaming query with optional source information
func (c *Client) StreamQueryWithSources(query string, callback func(string), showSources bool) ([]domain.Chunk, error) {
	ctx := context.Background()
	ragOpts := &rag.QueryOptions{
		ShowSources: showSources,
	}

	// Get response with sources
	resp, err := c.ragClient.Query(ctx, query, ragOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}

	// Stream the response in chunks
	words := strings.Fields(resp.Answer)
	for _, word := range words {
		callback(word + " ")
	}

	return resp.Sources, nil
}

// StreamQueryWithFilters performs a streaming query with metadata filters
func (c *Client) StreamQueryWithFilters(query string, filters map[string]interface{}, callback func(string)) error {
	ctx := context.Background()
	// Filters not yet implemented in RAG client
	// Fall back to regular query and stream the response
	resp, err := c.ragClient.Query(ctx, query, nil)
	if err != nil {
		return err
	}
	// Stream the response in chunks
	words := strings.Fields(resp.Answer)
	for _, word := range words {
		callback(word + " ")
	}
	return nil
}

// ListDocuments returns all documents in the knowledge base
func (c *Client) ListDocuments() ([]domain.Document, error) {
	ctx := context.Background()
	return c.ragClient.ListDocuments(ctx)
}

// DeleteDocument deletes a document by ID
func (c *Client) DeleteDocument(documentID string) error {
	ctx := context.Background()
	return c.ragClient.DeleteDocument(ctx, documentID)
}

// Reset clears all documents from the knowledge base
func (c *Client) Reset() error {
	ctx := context.Background()
	return c.ragClient.Reset(ctx)
}

// ========================================
// Advanced RAG Features (kept in client)
// ========================================

// QueryWithTools performs a query with tool calling enabled (advanced feature)
func (c *Client) QueryWithTools(query string, allowedTools []string, maxToolCalls int) (domain.QueryResponse, error) {
	ctx := context.Background()
	
	// Build tool options
	ragOpts := &rag.QueryOptions{
		ShowSources: true,
	}
	
	// First get RAG context
	resp, err := c.ragClient.Query(ctx, query, ragOpts)
	if err != nil {
		return domain.QueryResponse{}, err
	}
	
	// If MCP is enabled, use tools
	if c.mcpService != nil && len(allowedTools) > 0 {
		// Get available tools
		tools := c.mcpService.GetAvailableTools(ctx)
		
		// Filter by allowed tools
		var filteredTools []domain.ToolDefinition
		for _, tool := range tools {
			for _, allowed := range allowedTools {
				if tool.Name == allowed {
					filteredTools = append(filteredTools, domain.ToolDefinition{
						Type: "function",
						Function: domain.ToolFunction{
							Name:        tool.Name,
							Description: tool.Description,
							Parameters:  map[string]interface{}{},
						},
					})
					break
				}
			}
		}
		
		// Enhance response with tool results
		if len(filteredTools) > 0 {
			messages := []domain.Message{
				{Role: "user", Content: query},
			}
			
			genOpts := &domain.GenerationOptions{
				Temperature: 0.7,
				MaxTokens:   4000,
			}
			
			result, err := c.llm.GenerateWithTools(ctx, messages, filteredTools, genOpts)
			if err == nil && result != nil {
				resp.Answer = result.Content
				// Execute tool calls if any
				for i, call := range result.ToolCalls {
					if i >= maxToolCalls {
						break
					}
					toolResult, err := c.mcpService.CallTool(ctx, call.Function.Name, call.Function.Arguments)
					if err == nil && toolResult != nil {
						// Append tool results to response
						resp.Answer += fmt.Sprintf("\n\nTool Result (%s): %v", call.Function.Name, toolResult.Data)
					}
				}
			}
		}
	}
	
	return *resp, nil
}

// QueryWithMCP performs a query with MCP tools enabled (advanced feature)
func (c *Client) QueryWithMCP(query string) (domain.QueryResponse, error) {
	return c.QueryWithTools(query, nil, 5) // Use all available tools, max 5 calls
}

// DocumentInfo contains document information with formatted metadata (advanced feature)
type DocumentInfo struct {
	ID           string
	Path         string
	URL          string
	Source       string
	Created      time.Time
	Summary      string
	Keywords     []string
	DocumentType string
	TemporalRefs map[string]string
	Entities     map[string][]string
	Events       []string
	Metadata     map[string]interface{}
}

// ListDocumentsWithInfo returns documents with parsed metadata information (advanced feature)
func (c *Client) ListDocumentsWithInfo() ([]DocumentInfo, error) {
	ctx := context.Background()
	docs, err := c.ragClient.ListDocuments(ctx)
	if err != nil {
		return nil, err
	}

	infos := make([]DocumentInfo, 0, len(docs))
	for _, doc := range docs {
		info := DocumentInfo{
			ID:       doc.ID,
			Path:     doc.Path,
			URL:      doc.URL,
			Created:  doc.Created,
			Metadata: doc.Metadata,
		}

		// Extract source
		if doc.Path != "" {
			info.Source = doc.Path
		} else if doc.URL != "" {
			info.Source = doc.URL
		} else if source, ok := doc.Metadata["source"].(string); ok {
			info.Source = source
		}

		// Extract enhanced metadata
		if doc.Metadata != nil {
			// Summary
			if summary, ok := doc.Metadata["summary"].(string); ok {
				info.Summary = summary
			}

			// Keywords - handle both []interface{} and string formats
			if keywords, ok := doc.Metadata["keywords"].([]interface{}); ok {
				for _, k := range keywords {
					if str, ok := k.(string); ok {
						info.Keywords = append(info.Keywords, str)
					}
				}
			} else if keywordStr, ok := doc.Metadata["keywords"].(string); ok {
				// Parse "[keyword1 keyword2]" format
				if strings.HasPrefix(keywordStr, "[") && strings.HasSuffix(keywordStr, "]") {
					keywordStr = strings.TrimPrefix(keywordStr, "[")
					keywordStr = strings.TrimSuffix(keywordStr, "]")
					info.Keywords = strings.Fields(keywordStr)
				}
			}

			// Document type
			if docType, ok := doc.Metadata["document_type"].(string); ok {
				info.DocumentType = docType
			}

			// Temporal references
			if temporalRefs, ok := doc.Metadata["temporal_refs"].(map[string]interface{}); ok {
				info.TemporalRefs = make(map[string]string)
				for k, v := range temporalRefs {
					if str, ok := v.(string); ok {
						info.TemporalRefs[k] = str
					}
				}
			} else if temporalStr, ok := doc.Metadata["temporal_refs"].(string); ok {
				// Parse string format if needed
				info.TemporalRefs = parseMapString(temporalStr)
			}

			// Entities
			if entities, ok := doc.Metadata["entities"].(map[string]interface{}); ok {
				info.Entities = make(map[string][]string)
				for k, v := range entities {
					if list, ok := v.([]interface{}); ok {
						for _, item := range list {
							if str, ok := item.(string); ok {
								info.Entities[k] = append(info.Entities[k], str)
							}
						}
					}
				}
			}

			// Events
			if events, ok := doc.Metadata["events"].([]interface{}); ok {
				for _, e := range events {
					if str, ok := e.(string); ok {
						info.Events = append(info.Events, str)
					}
				}
			}
		}

		infos = append(infos, info)
	}

	return infos, nil
}

// parseMapString parses a string representation of a map
func parseMapString(str string) map[string]string {
	result := make(map[string]string)
	// Simple parsing logic for "{key1: value1, key2: value2}" format
	str = strings.TrimPrefix(str, "{")
	str = strings.TrimSuffix(str, "}")
	parts := strings.Split(str, ",")
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), ":", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			result[key] = value
		}
	}
	return result
}