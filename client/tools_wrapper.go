package client

import (
	"context"
	"fmt"

	"github.com/liliang-cn/rago/v2/pkg/mcp"
)

// ToolsWrapper wraps the MCP service to provide tools functionality
type ToolsWrapper struct {
	service *mcp.Service
}

// NewToolsWrapper creates a new tools wrapper
func NewToolsWrapper(service *mcp.Service) *ToolsWrapper {
	return &ToolsWrapper{service: service}
}

// List lists available tools (simplified method)
func (t *ToolsWrapper) List() ([]ToolInfo, error) {
	ctx := context.Background()
	return t.ListWithOptions(ctx)
}

// ListWithOptions lists available tools
func (t *ToolsWrapper) ListWithOptions(ctx context.Context) ([]ToolInfo, error) {
	if t.service == nil {
		return nil, fmt.Errorf("MCP service not initialized")
	}

	tools := t.service.GetAvailableTools(ctx)

	// Convert to ToolInfo format
	result := make([]ToolInfo, 0, len(tools))
	for _, tool := range tools {
		// Convert parameter names to a map structure for compatibility
		params := make(map[string]interface{})
		for _, paramName := range tool.Parameters {
			params[paramName] = map[string]string{"type": "string"}
		}

		toolInfo := ToolInfo{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  params,
		}
		result = append(result, toolInfo)
	}

	return result, nil
}

// CallWithOptions calls a tool with specific arguments
func (t *ToolsWrapper) CallWithOptions(ctx context.Context, name string, args map[string]interface{}) (map[string]interface{}, error) {
	if t.service == nil {
		return nil, fmt.Errorf("MCP service not initialized")
	}

	result, err := t.service.CallTool(ctx, name, args)
	if err != nil {
		return nil, err
	}

	// Return the result as a map
	return map[string]interface{}{
		"result": result,
	}, nil
}
