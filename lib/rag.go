package rago

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/liliang-cn/rago/internal/domain"
)

// IngestFile ingests a file from the local filesystem
func (c *Client) IngestFile(filePath string) error {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	ctx := context.Background()
	req := domain.IngestRequest{
		FilePath: absPath,
	}

	_, err = c.processor.Ingest(ctx, req)
	return err
}

// IngestText ingests text content directly
func (c *Client) IngestText(text, source string) error {
	return c.IngestTextWithMetadata(text, source, nil)
}

// IngestTextWithMetadata ingests text content with additional metadata
func (c *Client) IngestTextWithMetadata(text, source string, additionalMetadata map[string]interface{}) error {
	ctx := context.Background()
	req := domain.IngestRequest{
		Content:  text,
		Metadata: additionalMetadata,
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

// ListDocuments returns all documents in the knowledge base
func (c *Client) ListDocuments() ([]domain.Document, error) {
	ctx := context.Background()
	return c.vectorStore.List(ctx)
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
		MaxTokens:    1000,
		ToolsEnabled: true,
		MaxToolCalls: 3,
	}

	return c.processor.QueryWithTools(ctx, req)
}
