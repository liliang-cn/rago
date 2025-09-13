package client

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// IngestOptions configures how content is ingested
type IngestOptions struct {
	ChunkSize          int                    // Size of text chunks
	Overlap            int                    // Overlap between chunks
	EnhancedExtraction bool                   // Enable enhanced metadata extraction
	Metadata           map[string]interface{} // Additional metadata
}

// IngestFile ingests a file from the local filesystem
func (c *Client) IngestFile(filePath string) error {
	return c.IngestFileWithOptions(filePath, nil)
}

// IngestFileWithOptions ingests a file with custom options
func (c *Client) IngestFileWithOptions(filePath string, opts *IngestOptions) error {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	ctx := context.Background()
	req := domain.IngestRequest{
		FilePath: absPath,
	}

	if opts != nil {
		req.ChunkSize = opts.ChunkSize
		req.Overlap = opts.Overlap
		req.Metadata = opts.Metadata

		// Enable enhanced extraction if requested
		if opts.EnhancedExtraction {
			// Temporarily enable metadata extraction for this request
			origConfig := c.config.Ingest.MetadataExtraction.Enable
			c.config.Ingest.MetadataExtraction.Enable = true
			defer func() {
				c.config.Ingest.MetadataExtraction.Enable = origConfig
			}()
		}
	}

	_, err = c.processor.Ingest(ctx, req)
	return err
}

// IngestText ingests text content directly
func (c *Client) IngestText(text, source string) error {
	return c.IngestTextWithOptions(text, source, nil)
}

// IngestTextWithMetadata ingests text content with additional metadata
func (c *Client) IngestTextWithMetadata(text, source string, additionalMetadata map[string]interface{}) error {
	opts := &IngestOptions{
		Metadata: additionalMetadata,
	}
	return c.IngestTextWithOptions(text, source, opts)
}

// IngestTextWithOptions ingests text content with custom options
func (c *Client) IngestTextWithOptions(text, source string, opts *IngestOptions) error {
	ctx := context.Background()

	metadata := make(map[string]interface{})
	metadata["source"] = source
	metadata["type"] = "text"

	req := domain.IngestRequest{
		Content:  text,
		Metadata: metadata,
	}

	if opts != nil {
		req.ChunkSize = opts.ChunkSize
		req.Overlap = opts.Overlap

		// Merge additional metadata
		if opts.Metadata != nil {
			for k, v := range opts.Metadata {
				req.Metadata[k] = v
			}
		}

		// Enable enhanced extraction if requested
		if opts.EnhancedExtraction {
			// Temporarily enable metadata extraction for this request
			origConfig := c.config.Ingest.MetadataExtraction.Enable
			c.config.Ingest.MetadataExtraction.Enable = true
			defer func() {
				c.config.Ingest.MetadataExtraction.Enable = origConfig
			}()
		}
	}

	_, err := c.processor.Ingest(ctx, req)
	return err
}

// Query performs a simple query and returns the response
func (c *Client) Query(query string) (domain.QueryResponse, error) {
	return c.QueryWithSources(query, false)
}

// QueryWithSources performs a query with optional source information
func (c *Client) QueryWithSources(query string, showSources bool) (domain.QueryResponse, error) {
	ctx := context.Background()
	req := domain.QueryRequest{
		Query:       query,
		TopK:        c.config.Sqvect.TopK,
		Temperature: 0.7,
		MaxTokens:   4000,
		ShowSources: showSources,
	}

	return c.processor.Query(ctx, req)
}

// QueryWithTools performs a query with tool calling enabled
func (c *Client) QueryWithTools(query string, allowedTools []string, maxToolCalls int) (domain.QueryResponse, error) {
	ctx := context.Background()
	req := domain.QueryRequest{
		Query:        query,
		TopK:         c.config.Sqvect.TopK,
		Temperature:  0.7,
		MaxTokens:    4000,
		ToolsEnabled: true,
		AllowedTools: allowedTools,
		MaxToolCalls: maxToolCalls,
	}

	return c.processor.Query(ctx, req)
}

// QueryWithFilters performs a query with metadata filters
func (c *Client) QueryWithFilters(query string, filters map[string]interface{}) (domain.QueryResponse, error) {
	ctx := context.Background()
	req := domain.QueryRequest{
		Query:       query,
		TopK:        c.config.Sqvect.TopK,
		Temperature: 0.7,
		MaxTokens:   4000,
		Filters:     filters,
	}

	return c.processor.Query(ctx, req)
}

// StreamQuery performs a streaming query
func (c *Client) StreamQuery(query string, callback func(string)) error {
	_, err := c.StreamQueryWithSources(query, callback, false)
	return err
}

// StreamQueryWithSources performs a streaming query with optional source information
func (c *Client) StreamQueryWithSources(query string, callback func(string), showSources bool) ([]domain.Chunk, error) {
	ctx := context.Background()
	req := domain.QueryRequest{
		Query:        query,
		TopK:         c.config.Sqvect.TopK,
		Temperature:  0.7,
		MaxTokens:    4000,
		Stream:       true,
		ShowSources:  showSources,
		ShowThinking: false,
	}

	// Get sources first if requested
	var sources []domain.Chunk
	if showSources {
		resp, err := c.processor.Query(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve sources: %w", err)
		}
		sources = resp.Sources
	}

	// Perform streaming query
	err := c.processor.StreamQuery(ctx, req, callback)
	if err != nil {
		return nil, fmt.Errorf("streaming query failed: %w", err)
	}

	return sources, nil
}

// StreamQueryWithFilters performs a streaming query with metadata filters
func (c *Client) StreamQueryWithFilters(query string, filters map[string]interface{}, callback func(string)) error {
	ctx := context.Background()
	req := domain.QueryRequest{
		Query:       query,
		TopK:        c.config.Sqvect.TopK,
		Temperature: 0.7,
		MaxTokens:   4000,
		Stream:      true,
		Filters:     filters,
	}

	return c.processor.StreamQuery(ctx, req, callback)
}

// DocumentInfo contains document information with formatted metadata
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

// ListDocuments returns all documents in the knowledge base
func (c *Client) ListDocuments() ([]domain.Document, error) {
	ctx := context.Background()
	return c.vectorStore.List(ctx)
}

// ListDocumentsWithInfo returns documents with parsed metadata information
func (c *Client) ListDocumentsWithInfo() ([]DocumentInfo, error) {
	ctx := context.Background()
	docs, err := c.vectorStore.List(ctx)
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
				for category, items := range entities {
					if itemList, ok := items.([]interface{}); ok {
						for _, item := range itemList {
							if str, ok := item.(string); ok {
								info.Entities[category] = append(info.Entities[category], str)
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

// parseMapString parses a string like "map[key:value key2:value2]" into a map
func parseMapString(s string) map[string]string {
	result := make(map[string]string)

	// Remove "map[" prefix and "]" suffix
	if strings.HasPrefix(s, "map[") {
		s = strings.TrimPrefix(s, "map[")
		s = strings.TrimSuffix(s, "]")

		// Split by spaces and parse key:value pairs
		pairs := strings.Fields(s)
		for _, pair := range pairs {
			parts := strings.SplitN(pair, ":", 2)
			if len(parts) == 2 {
				result[parts[0]] = parts[1]
			}
		}
	}

	return result
}

// DeleteDocument removes a document and all its chunks from the knowledge base
func (c *Client) DeleteDocument(documentID string) error {
	ctx := context.Background()
	return c.processor.DeleteDocument(ctx, documentID)
}

// Reset clears all data from the knowledge base
func (c *Client) Reset() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return c.processor.Reset(ctx)
}

// QueryWithMCP performs a query using MCP tools for enhanced functionality
func (c *Client) QueryWithMCP(query string) (domain.QueryResponse, error) {
	if !c.IsMCPEnabled() {
		return domain.QueryResponse{}, fmt.Errorf("MCP is not enabled")
	}

	// Get MCP tools for LLM integration
	tools, err := c.ListMCPTools()
	if err != nil {
		return domain.QueryResponse{}, fmt.Errorf("failed to get MCP tools: %w", err)
	}

	if len(tools) == 0 {
		return domain.QueryResponse{}, fmt.Errorf("no MCP tools available")
	}

	// Format tools for the prompt
	var toolDescriptions []string
	for _, tool := range tools {
		toolDescriptions = append(toolDescriptions,
			fmt.Sprintf("- %s_%s (%s): %s", tool.ServerName, tool.Name, tool.ServerName, tool.Description))
	}

	// Create enhanced prompt that includes tool information
	systemPrompt := fmt.Sprintf(`You are an AI assistant with access to MCP tools. You can use the following tools to answer questions:

Available MCP Tools:
%s

User Question: %s

Please analyze the question and determine which MCP tool(s) to use to answer it effectively. Call the appropriate tools with the necessary parameters.`,
		strings.Join(toolDescriptions, "\n"),
		query,
	)

	ctx := context.Background()
	req := domain.QueryRequest{
		Query:        systemPrompt,
		TopK:         c.config.Sqvect.TopK,
		Temperature:  0.7,
		MaxTokens:    30000,
		ToolsEnabled: true,
		MaxToolCalls: 3,
	}

	return c.processor.QueryWithTools(ctx, req)
}
