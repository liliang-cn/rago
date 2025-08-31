package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/liliang-cn/rago/pkg/domain"
)

// Coordinator manages the interaction between LLM and tool execution
type Coordinator struct {
	registry        *Registry
	executor        *Executor
	pluginManager   *PluginManager
	logger          Logger
	maxToolCalls    int
	maxRounds       int
	conversationTTL time.Duration
	conversations   map[string]*Conversation
	mu              sync.RWMutex
}

// Conversation tracks a multi-turn tool calling conversation
type Conversation struct {
	ID           string                    `json:"id"`
	Messages     []domain.Message          `json:"messages"`
	ToolCalls    []domain.ExecutedToolCall `json:"tool_calls"`
	StartTime    time.Time                 `json:"start_time"`
	LastActivity time.Time                 `json:"last_activity"`
	RoundCount   int                       `json:"round_count"`
	Completed    bool                      `json:"completed"`
	Error        string                    `json:"error,omitempty"`
	Context      *ExecutionContext         `json:"context"`
}

// CoordinatorConfig contains configuration for the coordinator
type CoordinatorConfig struct {
	MaxToolCalls    int           `json:"max_tool_calls"`
	MaxRounds       int           `json:"max_rounds"`
	ConversationTTL time.Duration `json:"conversation_ttl"`
	EnableLogging   bool          `json:"enable_logging"`
}

// DefaultCoordinatorConfig returns default configuration
func DefaultCoordinatorConfig() CoordinatorConfig {
	return CoordinatorConfig{
		MaxToolCalls:    10,
		MaxRounds:       5,
		ConversationTTL: 30 * time.Minute,
		EnableLogging:   true,
	}
}

// NewCoordinator creates a new tool calling coordinator
func NewCoordinator(registry *Registry, executor *Executor, config CoordinatorConfig) *Coordinator {
	coord := &Coordinator{
		registry:        registry,
		executor:        executor,
		logger:          &DefaultLogger{},
		maxToolCalls:    config.MaxToolCalls,
		maxRounds:       config.MaxRounds,
		conversationTTL: config.ConversationTTL,
		conversations:   make(map[string]*Conversation),
	}

	go coord.cleanupExpiredConversations()

	return coord
}

// SetPluginManager sets the plugin manager for the coordinator
func (c *Coordinator) SetPluginManager(pm *PluginManager) {
	c.pluginManager = pm
}

// SetLogger sets a custom logger
func (c *Coordinator) SetLogger(logger Logger) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.logger = logger
}

// ProcessToolCalls processes tool calls from LLM and returns the results
func (c *Coordinator) ProcessToolCalls(ctx context.Context, conversationID string,
	toolCalls []domain.ToolCall, execCtx *ExecutionContext) ([]domain.ExecutedToolCall, error) {
	if len(toolCalls) == 0 {
		return nil, nil
	}

	if len(toolCalls) > c.maxToolCalls {
		return nil, fmt.Errorf("too many tool calls: %d exceeds maximum %d", len(toolCalls), c.maxToolCalls)
	}

	conversation := c.getOrCreateConversation(conversationID, execCtx)

	if conversation.RoundCount >= c.maxRounds {
		return nil, fmt.Errorf("conversation %s exceeded maximum rounds: %d", conversationID, c.maxRounds)
	}

	c.logger.Info("Processing %d tool calls for conversation %s", len(toolCalls), conversationID)

	results := make([]domain.ExecutedToolCall, len(toolCalls))
	var wg sync.WaitGroup

	for i, toolCall := range toolCalls {
		wg.Add(1)
		go func(index int, call domain.ToolCall) {
			defer wg.Done()

			startTime := time.Now()
			result, err := c.executor.Execute(ctx, execCtx, call.Function.Name, call.Function.Arguments)
			elapsed := time.Since(startTime)

			executed := domain.ExecutedToolCall{
				ToolCall: call,
				Elapsed:  elapsed.String(),
				Success:  err == nil,
			}

			if err != nil {
				executed.Error = err.Error()
				c.logger.Error("Tool execution failed for %s: %v", call.Function.Name, err)
			} else {
				executed.Result = result.Data
				c.logger.Debug("Tool execution succeeded for %s in %v", call.Function.Name, elapsed)
			}

			results[index] = executed
		}(i, toolCall)
	}

	wg.Wait()

	c.updateConversation(conversationID, results)

	return results, nil
}

// HandleToolCallingConversation manages a complete tool calling conversation
func (c *Coordinator) HandleToolCallingConversation(ctx context.Context,
	generator domain.Generator, prompt string, tools []domain.ToolDefinition,
	opts *domain.GenerationOptions, execCtx *ExecutionContext) (*domain.QueryResponse, error) {

	conversationID := uuid.New().String()
	if execCtx == nil {
		execCtx = &ExecutionContext{RequestID: conversationID}
	}

	c.logger.Info("Starting tool calling conversation %s", conversationID)

	conversation := c.getOrCreateConversation(conversationID, execCtx)
	allToolCalls := make([]domain.ExecutedToolCall, 0)
	toolsUsed := make(map[string]bool)

	messages := []domain.Message{
		{Role: "user", Content: prompt},
	}

	for round := 0; round < c.maxRounds; round++ {
		conversation.RoundCount = round + 1
		c.logger.Debug("Tool calling conversation %s round %d", conversationID, round+1)

		result, err := generator.GenerateWithTools(ctx, messages, tools, opts)
		if err != nil {
			c.markConversationError(conversationID, err)
			return nil, fmt.Errorf("generation failed in round %d: %w", round+1, err)
		}

		assistantMsg := domain.Message{
			Role:    "assistant",
			Content: result.Content,
		}

		if len(result.ToolCalls) == 0 {
			messages = append(messages, assistantMsg)
			break
		}

		assistantMsg.ToolCalls = result.ToolCalls
		messages = append(messages, assistantMsg)

		executedCalls, _ := c.ProcessToolCalls(ctx, conversationID, result.ToolCalls, execCtx)

		allToolCalls = append(allToolCalls, executedCalls...)
		for _, call := range executedCalls {
			toolsUsed[call.Function.Name] = true
		}

		for _, executed := range executedCalls {
			toolMsg := domain.Message{
				Role:       "tool",
				Content:    c.formatToolResult(executed),
				ToolCallID: executed.ID,
			}
			messages = append(messages, toolMsg)
		}
	}

	c.markConversationCompleted(conversationID)

	usedToolsList := make([]string, 0, len(toolsUsed))
	for tool := range toolsUsed {
		usedToolsList = append(usedToolsList, tool)
	}

	finalContent := ""
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "assistant" && messages[i].Content != "" {
			finalContent = messages[i].Content
			break
		}
	}

	return &domain.QueryResponse{
		Answer:    finalContent,
		ToolCalls: allToolCalls,
		ToolsUsed: usedToolsList,
	}, nil
}

// StreamToolCallingConversation handles streaming tool calling conversation
func (c *Coordinator) StreamToolCallingConversation(ctx context.Context,
	generator domain.Generator, prompt string, tools []domain.ToolDefinition,
	opts *domain.GenerationOptions, execCtx *ExecutionContext,
	callback func(chunk string, toolCalls []domain.ExecutedToolCall, finished bool) error) error {

	conversationID := uuid.New().String()
	if execCtx == nil {
		execCtx = &ExecutionContext{RequestID: conversationID}
	}

	c.logger.Info("Starting streaming tool calling conversation %s", conversationID)

	conversation := c.getOrCreateConversation(conversationID, execCtx)

	messages := []domain.Message{
		{Role: "user", Content: prompt},
	}

	for round := 0; round < c.maxRounds; round++ {
		conversation.RoundCount = round + 1
		c.logger.Debug("Streaming tool calling conversation %s round %d", conversationID, round+1)

		roundToolCalls := make([]domain.ToolCall, 0)
		roundContent := strings.Builder{}

		streamCallback := func(chunk string, toolCalls []domain.ToolCall) error {
			if chunk != "" {
				roundContent.WriteString(chunk)
				return callback(chunk, nil, false)
			}
			if len(toolCalls) > 0 {
				roundToolCalls = append(roundToolCalls, toolCalls...)
			}
			return nil
		}

		err := generator.StreamWithTools(ctx, messages, tools, opts, streamCallback)
		if err != nil {
			c.markConversationError(conversationID, err)
			return fmt.Errorf("generation failed in streaming round %d: %w", round+1, err)
		}

		assistantMsg := domain.Message{
			Role:    "assistant",
			Content: roundContent.String(),
		}

		if len(roundToolCalls) == 0 {
			messages = append(messages, assistantMsg)
			break
		}

		assistantMsg.ToolCalls = roundToolCalls
		messages = append(messages, assistantMsg)

		executedCalls, _ := c.ProcessToolCalls(ctx, conversationID, roundToolCalls, execCtx)

		if err := callback("", executedCalls, false); err != nil {
			return fmt.Errorf("callback failed for tool results in round %d: %w", round+1, err)
		}

		for _, executed := range executedCalls {
			toolMsg := domain.Message{
				Role:       "tool",
				Content:    c.formatToolResult(executed),
				ToolCallID: executed.ID,
			}
			messages = append(messages, toolMsg)
		}

		if round == c.maxRounds-1 {
			c.logger.Warn("Streaming conversation %s reached maximum rounds %d", conversationID, c.maxRounds)
			break
		}
	}

	c.markConversationCompleted(conversationID)
	return callback("", nil, true)
}


// Helper methods
func (c *Coordinator) getOrCreateConversation(id string, execCtx *ExecutionContext) *Conversation {
	c.mu.Lock()
	defer c.mu.Unlock()

	if conv, exists := c.conversations[id]; exists {
		conv.LastActivity = time.Now()
		return conv
	}

	conv := &Conversation{
		ID:           id,
		Messages:     make([]domain.Message, 0),
		ToolCalls:    make([]domain.ExecutedToolCall, 0),
		StartTime:    time.Now(),
		LastActivity: time.Now(),
		RoundCount:   0,
		Completed:    false,
		Context:      execCtx,
	}

	c.conversations[id] = conv
	return conv
}

func (c *Coordinator) updateConversation(id string, toolCalls []domain.ExecutedToolCall) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if conv, exists := c.conversations[id]; exists {
		conv.ToolCalls = append(conv.ToolCalls, toolCalls...)
		conv.LastActivity = time.Now()
	}
}

func (c *Coordinator) markConversationCompleted(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if conv, exists := c.conversations[id]; exists {
		conv.Completed = true
		conv.LastActivity = time.Now()
	}
}

func (c *Coordinator) markConversationError(id string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if conv, exists := c.conversations[id]; exists {
		conv.Error = err.Error()
		conv.LastActivity = time.Now()
	}
}

func (c *Coordinator) formatToolResult(executed domain.ExecutedToolCall) string {
	if !executed.Success {
		return fmt.Sprintf("Tool %s failed: %s", executed.Function.Name, executed.Error)
	}
	if executed.Result != nil {
		// Try to marshal result to JSON for a clean, structured representation
		jsonData, err := json.Marshal(executed.Result)
		if err == nil {
			return string(jsonData)
		}
		// Fallback to string representation
		return fmt.Sprintf("%v", executed.Result)
	}
	return "Tool executed successfully with no return value."
}

func (c *Coordinator) cleanupExpiredConversations() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for id, conv := range c.conversations {
			if now.Sub(conv.LastActivity) > c.conversationTTL {
				delete(c.conversations, id)
				c.logger.Debug("Cleaned up expired conversation %s", id)
			}
		}
		c.mu.Unlock()
	}
}

// GetConversation returns conversation by ID
func (c *Coordinator) GetConversation(id string) (*Conversation, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	conv, exists := c.conversations[id]
	return conv, exists
}

// ListConversations returns all active conversations
func (c *Coordinator) ListConversations() []*Conversation {
	c.mu.RLock()
	defer c.mu.RUnlock()

	conversations := make([]*Conversation, 0, len(c.conversations))
	for _, conv := range c.conversations {
		conversations = append(conversations, conv)
	}
	return conversations
}

// GetStats returns coordinator statistics
func (c *Coordinator) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := map[string]interface{}{
		"total_conversations": len(c.conversations),
		"completed_count":     0,
		"error_count":         0,
		"active_count":        0,
	}

	for _, conv := range c.conversations {
		if conv.Completed {
			stats["completed_count"] = stats["completed_count"].(int) + 1
		} else if conv.Error != "" {
			stats["error_count"] = stats["error_count"].(int) + 1
		} else {
			stats["active_count"] = stats["active_count"].(int) + 1
		}
	}

	return stats
}