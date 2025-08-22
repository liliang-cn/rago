package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/liliang-cn/rago/internal/domain"
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
	Messages     []Message                 `json:"messages"`
	ToolCalls    []domain.ExecutedToolCall `json:"tool_calls"`
	StartTime    time.Time                 `json:"start_time"`
	LastActivity time.Time                 `json:"last_activity"`
	RoundCount   int                       `json:"round_count"`
	Completed    bool                      `json:"completed"`
	Error        string                    `json:"error,omitempty"`
	Context      *ExecutionContext         `json:"context"`
}

// Message represents a conversation message
type Message struct {
	Role       string            `json:"role"` // user, assistant, tool
	Content    string            `json:"content"`
	ToolCalls  []domain.ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
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

	// Start cleanup goroutine
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

	// Get or create conversation
	conversation := c.getOrCreateConversation(conversationID, execCtx)

	// Check round limits
	if conversation.RoundCount >= c.maxRounds {
		return nil, fmt.Errorf("conversation %s exceeded maximum rounds: %d", conversationID, c.maxRounds)
	}

	c.logger.Info("Processing %d tool calls for conversation %s", len(toolCalls), conversationID)

	// Execute tool calls concurrently
	results := make([]domain.ExecutedToolCall, len(toolCalls))
	var wg sync.WaitGroup
	errChan := make(chan error, len(toolCalls))

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

	// Wait for all executions to complete
	wg.Wait()
	close(errChan)

	// Collect any errors
	var executionErrors []error
	for err := range errChan {
		if err != nil {
			executionErrors = append(executionErrors, err)
		}
	}

	// Update conversation
	c.updateConversation(conversationID, results)

	// Return results even if some tools failed
	return results, nil
}

// HandleToolCallingConversation manages a complete tool calling conversation
func (c *Coordinator) HandleToolCallingConversation(ctx context.Context,
	generator domain.Generator, prompt string, tools []domain.ToolDefinition,
	opts *domain.GenerationOptions, execCtx *ExecutionContext) (*domain.QueryResponse, error) {

	conversationID := uuid.New().String()
	if execCtx == nil {
		execCtx = &ExecutionContext{
			RequestID: conversationID,
		}
	}

	c.logger.Info("Starting tool calling conversation %s", conversationID)

	conversation := c.getOrCreateConversation(conversationID, execCtx)
	allToolCalls := make([]domain.ExecutedToolCall, 0)
	toolsUsed := make(map[string]bool)

	// Initial user message
	messages := []Message{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	for round := 0; round < c.maxRounds; round++ {
		conversation.RoundCount = round + 1

		c.logger.Debug("Tool calling conversation %s round %d", conversationID, round+1)

		// Generate response with tools
		result, err := generator.GenerateWithTools(ctx, c.buildPromptFromMessages(messages), tools, opts)
		if err != nil {
			c.markConversationError(conversationID, err)
			return nil, fmt.Errorf("generation failed in round %d: %w", round+1, err)
		}

		// Add assistant message
		assistantMsg := Message{
			Role:    "assistant",
			Content: result.Content,
		}

		// If no tool calls, we're done
		if len(result.ToolCalls) == 0 {
			messages = append(messages, assistantMsg)
			break
		}

		// Add tool calls to message
		assistantMsg.ToolCalls = result.ToolCalls
		messages = append(messages, assistantMsg)

		// Execute tool calls
		executedCalls, err := c.ProcessToolCalls(ctx, conversationID, result.ToolCalls, execCtx)
		if err != nil {
			c.logger.Error("Tool execution failed: %v", err)
			// Continue with partial results rather than failing completely
		}

		// Track executed tool calls and used tools
		allToolCalls = append(allToolCalls, executedCalls...)
		for _, call := range executedCalls {
			toolsUsed[call.Function.Name] = true
		}

		// Add tool result messages
		for _, executed := range executedCalls {
			toolMsg := Message{
				Role:       "tool",
				Content:    c.formatToolResult(executed),
				ToolCallID: executed.ID,
			}
			messages = append(messages, toolMsg)
		}
	}

	// Mark conversation as completed
	c.markConversationCompleted(conversationID)

	// Build final response
	usedToolsList := make([]string, 0, len(toolsUsed))
	for tool := range toolsUsed {
		usedToolsList = append(usedToolsList, tool)
	}

	// Get final content from the last assistant message
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
		execCtx = &ExecutionContext{
			RequestID: conversationID,
		}
	}

	c.logger.Info("Starting streaming tool calling conversation %s", conversationID)

	_ = c.getOrCreateConversation(conversationID, execCtx) // Create conversation for tracking
	allToolCalls := make([]domain.ExecutedToolCall, 0)

	messages := []Message{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	streamCallback := func(chunk string, toolCalls []domain.ToolCall) error {
		// If we have tool calls, execute them
		if len(toolCalls) > 0 {
			executedCalls, err := c.ProcessToolCalls(ctx, conversationID, toolCalls, execCtx)
			if err != nil {
				c.logger.Error("Tool execution failed in streaming mode: %v", err)
				return callback("", executedCalls, false)
			}

			allToolCalls = append(allToolCalls, executedCalls...)

			// Send tool execution results
			return callback("", executedCalls, false)
		}

		// Regular content chunk
		return callback(chunk, nil, false)
	}

	// Start streaming generation
	err := generator.StreamWithTools(ctx, c.buildPromptFromMessages(messages), tools, opts, streamCallback)
	if err != nil {
		c.markConversationError(conversationID, err)
		return err
	}

	// Mark as completed and send final callback
	c.markConversationCompleted(conversationID)
	return callback("", allToolCalls, true)
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
		Messages:     make([]Message, 0),
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

func (c *Coordinator) buildPromptFromMessages(messages []Message) string {
	// Build a proper conversation prompt that includes all messages and tool results
	var promptParts []string
	
	// Find the original user question
	var originalQuestion string
	for _, msg := range messages {
		if msg.Role == "user" && msg.Content != "" {
			originalQuestion = msg.Content
			break
		}
	}
	
	if originalQuestion == "" {
		return ""
	}
	
	// Check if we have tool results to incorporate
	var toolResults []string
	for _, msg := range messages {
		if msg.Role == "tool" && msg.Content != "" {
			toolResults = append(toolResults, msg.Content)
		}
	}
	
	// If we have tool results, build a prompt that asks the LLM to use them
	if len(toolResults) > 0 {
		promptParts = append(promptParts, "User question: "+originalQuestion)
		promptParts = append(promptParts, "\nTool execution results:")
		for i, result := range toolResults {
			promptParts = append(promptParts, fmt.Sprintf("Tool %d result: %s", i+1, result))
		}
		promptParts = append(promptParts, "\nPlease use BOTH the knowledge base documents AND the tool execution results above to provide a complete and accurate answer to the user's question. If the question involves current information (like time, date, or file status), prioritize the tool results. If it involves stored knowledge, use the relevant documents.")
		return strings.Join(promptParts, "\n")
	}
	
	// If no tool results yet, return the original question
	return originalQuestion
}

func (c *Coordinator) formatToolResult(executed domain.ExecutedToolCall) string {
	if executed.Success {
		// Format the result more comprehensively
		if executed.Result != nil {
			if dataMap, ok := executed.Result.(map[string]interface{}); ok {
				// Format structured data nicely for specific tools
				switch executed.Function.Name {
				case "datetime":
					if datetime, ok := dataMap["datetime"]; ok {
						if iso8601, ok := dataMap["iso8601"]; ok {
							return fmt.Sprintf("Tool %s executed successfully. Current time: %v (%v)", 
								executed.Function.Name, datetime, iso8601)
						}
						return fmt.Sprintf("Tool %s executed successfully. Current time: %v", 
							executed.Function.Name, datetime)
					}
				case "file_operations":
					if action, ok := dataMap["path"]; ok {
						if count, ok := dataMap["count"]; ok {
							return fmt.Sprintf("Tool %s executed successfully. Found %v items in %v", 
								executed.Function.Name, count, action)
						}
					}
				}
				
				// Generic structured data formatting
				var parts []string
				for key, value := range dataMap {
					parts = append(parts, fmt.Sprintf("%s: %v", key, value))
				}
				return fmt.Sprintf("Tool %s executed successfully. Results: %s", 
					executed.Function.Name, strings.Join(parts, ", "))
			}
			return fmt.Sprintf("Tool %s executed successfully. Result: %v", 
				executed.Function.Name, executed.Result)
		}
		return fmt.Sprintf("Tool %s executed successfully", executed.Function.Name)
	}
	return fmt.Sprintf("Tool %s failed: %s", executed.Function.Name, executed.Error)
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
