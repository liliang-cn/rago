package mcp

import (
	"context"
	"time"
)

// UsageTracker defines a minimal interface for tracking usage to avoid circular imports
type UsageTracker interface {
	TrackMCPCall(ctx context.Context, toolName string, params interface{}, startTime time.Time) (interface{}, error)
	TrackError(ctx context.Context, callType string, provider, model string, errorMsg string, startTime time.Time) (interface{}, error)
}

// TrackedMCPToolManager wraps MCPToolManager with usage tracking
type TrackedMCPToolManager struct {
	*MCPToolManager
	usageTracker UsageTracker
}

// NewTrackedMCPToolManager creates a new tracked MCP tool manager
func NewTrackedMCPToolManager(config *Config, usageTracker UsageTracker) *TrackedMCPToolManager {
	baseManager := NewMCPToolManager(config)
	
	return &TrackedMCPToolManager{
		MCPToolManager: baseManager,
		usageTracker:   usageTracker,
	}
}

// CallTool calls an MCP tool with usage tracking
func (tm *TrackedMCPToolManager) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (*MCPToolResult, error) {
	startTime := time.Now()
	
	// Call the underlying tool
	result, err := tm.MCPToolManager.CallTool(ctx, toolName, args)
	
	// Track the usage
	if tm.usageTracker != nil {
		if err != nil {
			// Track error
			_, _ = tm.usageTracker.TrackError(ctx, "mcp", "mcp", toolName, err.Error(), startTime)
		} else {
			// Track successful call
			_, _ = tm.usageTracker.TrackMCPCall(ctx, toolName, args, startTime)
		}
	}
	
	return result, err
}

// WithUsageTracking wraps any MCP component with usage tracking
func WithUsageTracking(component interface{}, usageTracker UsageTracker) interface{} {
	if usageTracker == nil {
		return component
	}
	
	switch c := component.(type) {
	case *MCPToolManager:
		return &TrackedMCPToolManager{
			MCPToolManager: c,
			usageTracker:   usageTracker,
		}
	default:
		// Return the original component if we don't know how to wrap it
		return component
	}
}