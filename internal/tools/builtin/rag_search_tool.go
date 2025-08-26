package builtin

import (
	"context"
	"fmt"

	"github.com/liliang-cn/rago/internal/domain"
	"github.com/liliang-cn/rago/internal/tools"
)

// ProcessorInterface defines the methods we need from processor
type ProcessorInterface interface {
	Query(ctx context.Context, req domain.QueryRequest) (domain.QueryResponse, error)
	ListDocuments(ctx context.Context) ([]domain.Document, error)
	DeleteDocument(ctx context.Context, documentID string) error
}

// RAGSearchTool provides RAG search functionality
type RAGSearchTool struct {
	processor ProcessorInterface
}

// NewRAGSearchTool creates a new RAG search tool
func NewRAGSearchTool(processor ProcessorInterface) *RAGSearchTool {
	return &RAGSearchTool{
		processor: processor,
	}
}

// Name returns the tool name
func (t *RAGSearchTool) Name() string {
	return "rag_search"
}

// Description returns the tool description
func (t *RAGSearchTool) Description() string {
	return "Search through the knowledge base using semantic similarity and return relevant document chunks"
}

// Parameters returns the tool parameters schema
func (t *RAGSearchTool) Parameters() tools.ToolParameters {
	return tools.ToolParameters{
		Type: "object",
		Properties: map[string]tools.ToolParameter{
			"query": {
				Type:        "string",
				Description: "The search query to find relevant documents",
			},
			"top_k": {
				Type:        "integer",
				Description: "Number of top results to return (default: 5)",
				Minimum:     func() *float64 { v := float64(1); return &v }(),
				Maximum:     func() *float64 { v := float64(20); return &v }(),
				Default:     5,
			},
			"filters": {
				Type:        "object",
				Description: "Metadata filters to apply (e.g., source, document_type)",
			},
			"include_content": {
				Type:        "boolean",
				Description: "Whether to include full chunk content in results (default: true)",
				Default:     true,
			},
		},
		Required: []string{"query"},
	}
}

// Execute runs the RAG search tool
func (t *RAGSearchTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return &tools.ToolResult{
			Success: false,
			Error:   "query parameter is required",
		}, nil
	}

	// Set default top_k
	topK := 5
	if k, ok := args["top_k"]; ok {
		if kInt, ok := k.(int); ok {
			topK = kInt
		} else if kFloat, ok := k.(float64); ok {
			topK = int(kFloat)
		}
	}

	// Extract filters
	var filters map[string]interface{}
	if f, ok := args["filters"]; ok {
		if fMap, ok := f.(map[string]interface{}); ok {
			filters = fMap
		}
	}

	// Include content flag
	includeContent := true
	if inc, ok := args["include_content"].(bool); ok {
		includeContent = inc
	}

	// Create query request
	queryReq := domain.QueryRequest{
		Query:   query,
		TopK:    topK,
		Filters: filters,
	}

	// Use the processor's hybrid search directly
	response, err := t.processor.Query(ctx, queryReq)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("search failed: %v", err),
		}, nil
	}

	// Format results
	results := make([]map[string]interface{}, 0, len(response.Sources))
	for i, chunk := range response.Sources {
		result := map[string]interface{}{
			"rank":        i + 1,
			"score":       chunk.Score,
			"document_id": chunk.DocumentID,
			"chunk_id":    chunk.ID,
		}

		if includeContent {
			result["content"] = chunk.Content
		} else {
			// Just include a preview
			content := chunk.Content
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			result["preview"] = content
		}

		if chunk.Metadata != nil {
			result["metadata"] = chunk.Metadata
		}

		results = append(results, result)
	}

	return &tools.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"query":       query,
			"total_found": len(results),
			"results":     results,
			"search_time": response.Elapsed,
		},
	}, nil
}

// Validate validates the tool arguments
func (t *RAGSearchTool) Validate(args map[string]interface{}) error {
	query, ok := args["query"]
	if !ok {
		return fmt.Errorf("query parameter is required")
	}

	if queryStr, ok := query.(string); !ok || queryStr == "" {
		return fmt.Errorf("query must be a non-empty string")
	}

	// Validate top_k if provided
	if topK, ok := args["top_k"]; ok {
		var k int
		if kInt, ok := topK.(int); ok {
			k = kInt
		} else if kFloat, ok := topK.(float64); ok {
			k = int(kFloat)
		} else {
			return fmt.Errorf("top_k must be an integer")
		}

		if k < 1 || k > 20 {
			return fmt.Errorf("top_k must be between 1 and 20")
		}
	}

	// Validate filters if provided
	if filters, ok := args["filters"]; ok {
		if _, ok := filters.(map[string]interface{}); !ok {
			return fmt.Errorf("filters must be an object")
		}
	}

	return nil
}

// DocumentInfoTool provides information about documents in the knowledge base
type DocumentInfoTool struct {
	processor ProcessorInterface
}

// NewDocumentInfoTool creates a new document info tool
func NewDocumentInfoTool(processor ProcessorInterface) *DocumentInfoTool {
	return &DocumentInfoTool{
		processor: processor,
	}
}

// Name returns the tool name
func (t *DocumentInfoTool) Name() string {
	return "document_info"
}

// Description returns the tool description
func (t *DocumentInfoTool) Description() string {
	return "Get information about documents in the knowledge base including count, list, and metadata"
}

// Parameters returns the tool parameters schema
func (t *DocumentInfoTool) Parameters() tools.ToolParameters {
	return tools.ToolParameters{
		Type: "object",
		Properties: map[string]tools.ToolParameter{
			"action": {
				Type:        "string",
				Description: "The action to perform",
				Enum:        []string{"count", "list", "get"},
			},
			"document_id": {
				Type:        "string",
				Description: "Document ID for 'get' action",
			},
			"include_metadata": {
				Type:        "boolean",
				Description: "Whether to include metadata in results (default: true)",
				Default:     true,
			},
		},
		Required: []string{"action"},
	}
}

// Execute runs the document info tool
func (t *DocumentInfoTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	action, ok := args["action"].(string)
	if !ok {
		return &tools.ToolResult{
			Success: false,
			Error:   "action parameter is required",
		}, nil
	}

	includeMetadata := true
	if inc, ok := args["include_metadata"].(bool); ok {
		includeMetadata = inc
	}

	switch action {
	case "count":
		return t.getDocumentCount(ctx)
	case "list":
		return t.listDocuments(ctx, includeMetadata)
	case "get":
		docID, ok := args["document_id"].(string)
		if !ok || docID == "" {
			return &tools.ToolResult{
				Success: false,
				Error:   "document_id parameter is required for 'get' action",
			}, nil
		}
		return t.getDocument(ctx, docID, includeMetadata)
	default:
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("unknown action: %s", action),
		}, nil
	}
}

// Validate validates the tool arguments
func (t *DocumentInfoTool) Validate(args map[string]interface{}) error {
	action, ok := args["action"]
	if !ok {
		return fmt.Errorf("action parameter is required")
	}

	actionStr, ok := action.(string)
	if !ok {
		return fmt.Errorf("action must be a string")
	}

	validActions := []string{"count", "list", "get"}
	valid := false
	for _, v := range validActions {
		if actionStr == v {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid action: %s", actionStr)
	}

	if actionStr == "get" {
		if _, ok := args["document_id"]; !ok {
			return fmt.Errorf("document_id parameter is required for 'get' action")
		}
	}

	return nil
}

func (t *DocumentInfoTool) getDocumentCount(ctx context.Context) (*tools.ToolResult, error) {
	documents, err := t.processor.ListDocuments(ctx)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to get document count: %v", err),
		}, nil
	}

	return &tools.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"count": len(documents),
		},
	}, nil
}

func (t *DocumentInfoTool) listDocuments(ctx context.Context, includeMetadata bool) (*tools.ToolResult, error) {
	documents, err := t.processor.ListDocuments(ctx)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to list documents: %v", err),
		}, nil
	}

	results := make([]map[string]interface{}, 0, len(documents))
	for _, doc := range documents {
		result := map[string]interface{}{
			"id":      doc.ID,
			"path":    doc.Path,
			"url":     doc.URL,
			"created": doc.Created.Format("2006-01-02 15:04:05"),
		}

		if includeMetadata && doc.Metadata != nil {
			result["metadata"] = doc.Metadata
		}

		results = append(results, result)
	}

	return &tools.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"total":     len(documents),
			"documents": results,
		},
	}, nil
}

func (t *DocumentInfoTool) getDocument(ctx context.Context, docID string, includeMetadata bool) (*tools.ToolResult, error) {
	documents, err := t.processor.ListDocuments(ctx)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to get documents: %v", err),
		}, nil
	}

	for _, doc := range documents {
		if doc.ID == docID {
			result := map[string]interface{}{
				"id":      doc.ID,
				"path":    doc.Path,
				"url":     doc.URL,
				"created": doc.Created.Format("2006-01-02 15:04:05"),
			}

			if includeMetadata && doc.Metadata != nil {
				result["metadata"] = doc.Metadata
			}

			return &tools.ToolResult{
				Success: true,
				Data:    result,
			}, nil
		}
	}

	return &tools.ToolResult{
		Success: false,
		Error:   fmt.Sprintf("document with ID %s not found", docID),
	}, nil
}
