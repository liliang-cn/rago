package usage

import (
	"context"
	"time"
)

// UsageTracker defines the interface for tracking usage
type UsageTracker interface {
	// TrackLLMCall tracks an LLM API call
	TrackLLMCall(ctx context.Context, provider, model string, input, output string, startTime time.Time) (*UsageRecord, error)
	
	// TrackLLMCallWithTokens tracks an LLM call with known token counts
	TrackLLMCallWithTokens(ctx context.Context, provider, model string, inputTokens, outputTokens int, startTime time.Time) (*UsageRecord, error)
	
	// TrackMCPCall tracks an MCP tool call
	TrackMCPCall(ctx context.Context, toolName string, params interface{}, startTime time.Time) (*UsageRecord, error)
	
	// TrackRAGCall tracks a RAG pipeline call
	TrackRAGCall(ctx context.Context, operation string, query string, resultCount int, startTime time.Time) (*UsageRecord, error)
	
	// TrackError tracks an error in any type of call
	TrackError(ctx context.Context, callType CallType, provider, model string, errorMsg string, startTime time.Time) (*UsageRecord, error)
	
	// AddMessage adds a message to the current conversation
	AddMessage(ctx context.Context, role, content string) (*Message, error)
}