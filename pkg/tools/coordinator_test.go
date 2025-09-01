package tools

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTool is a mock implementation of the Tool interface
type MockTool struct {
	mock.Mock
}

func (m *MockTool) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTool) Description() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTool) Parameters() ToolParameters {
	args := m.Called()
	return args.Get(0).(ToolParameters)
}

func (m *MockTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	mockArgs := m.Called(ctx, args)
	if mockArgs.Get(0) == nil {
		return nil, mockArgs.Error(1)
	}
	return mockArgs.Get(0).(*ToolResult), mockArgs.Error(1)
}

func (m *MockTool) Validate(args map[string]interface{}) error {
	mockArgs := m.Called(args)
	return mockArgs.Error(0)
}

// MockGenerator is a mock implementation of the Generator interface
type MockGenerator struct {
	mock.Mock
}

func (m *MockGenerator) Generate(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	args := m.Called(ctx, prompt, opts)
	return args.String(0), args.Error(1)
}

func (m *MockGenerator) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
	args := m.Called(ctx, prompt, opts, callback)
	return args.Error(0)
}

func (m *MockGenerator) GenerateWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
	args := m.Called(ctx, messages, tools, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.GenerationResult), args.Error(1)
}

func (m *MockGenerator) StreamWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error {
	args := m.Called(ctx, messages, tools, opts, callback)
	return args.Error(0)
}

func (m *MockGenerator) GenerateStructured(ctx context.Context, prompt string, schema interface{}, opts *domain.GenerationOptions) (*domain.StructuredResult, error) {
	args := m.Called(ctx, prompt, schema, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.StructuredResult), args.Error(1)
}

func setupCoordinator(t *testing.T) (*Coordinator, *Registry, *Executor) {
	config := DefaultToolConfig()
	// Enable test tools for testing
	config.EnabledTools = []string{"test_tool"}
	// Built-in tools are no longer supported - use MCP servers

	registry := NewRegistry(&config)

	executorConfig := &ExecutorConfig{
		MaxConcurrency: 3,
		DefaultTimeout: 10 * time.Second,
		EnableLogging:  false,
	}
	executor := NewExecutor(registry, executorConfig)

	coordConfig := DefaultCoordinatorConfig()
	coordConfig.ConversationTTL = time.Minute // Short TTL for testing
	coordinator := NewCoordinator(registry, executor, coordConfig)

	return coordinator, registry, executor
}

func TestNewCoordinator(t *testing.T) {
	coordinator, _, _ := setupCoordinator(t)

	assert.NotNil(t, coordinator)
	assert.Equal(t, 10, coordinator.maxToolCalls)
	assert.Equal(t, 5, coordinator.maxRounds)
	assert.NotNil(t, coordinator.conversations)
}

func TestCoordinator_ProcessToolCalls_Success(t *testing.T) {
	coordinator, registry, _ := setupCoordinator(t)

	// Setup mock tool - only mock what's actually called
	mockTool := &MockTool{}
	mockTool.On("Name").Return("test_tool")
	mockTool.On("Parameters").Return(ToolParameters{
		Type: "object",
		Properties: map[string]ToolParameter{
			"input": {
				Type:        "string",
				Description: "Test input",
			},
		},
		Required: []string{"input"},
	})
	mockTool.On("Validate", mock.Anything).Return(nil)
	mockTool.On("Execute", mock.Anything, mock.Anything).Return(&ToolResult{
		Success: true,
		Data:    "test result",
	}, nil)

	err := registry.Register(mockTool)
	assert.NoError(t, err)

	// Test tool calls
	toolCalls := []domain.ToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: domain.FunctionCall{
				Name:      "test_tool",
				Arguments: map[string]interface{}{"input": "test"},
			},
		},
	}

	ctx := context.Background()
	execCtx := &ExecutionContext{RequestID: "test"}

	results, err := coordinator.ProcessToolCalls(ctx, "conv_1", toolCalls, execCtx)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.True(t, results[0].Success)
	assert.Equal(t, "test result", results[0].Result)

	mockTool.AssertExpectations(t)
}

func TestCoordinator_ProcessToolCalls_ToolNotFound(t *testing.T) {
	coordinator, _, _ := setupCoordinator(t)

	toolCalls := []domain.ToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: domain.FunctionCall{
				Name:      "nonexistent_tool",
				Arguments: map[string]interface{}{},
			},
		},
	}

	ctx := context.Background()
	execCtx := &ExecutionContext{RequestID: "test"}

	results, err := coordinator.ProcessToolCalls(ctx, "conv_1", toolCalls, execCtx)

	// Should not return error but results should indicate failure
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.False(t, results[0].Success)
	assert.Contains(t, results[0].Error, "not found")
}

func TestCoordinator_ProcessToolCalls_TooManyToolCalls(t *testing.T) {
	coordinator, _, _ := setupCoordinator(t)

	// Create more tool calls than the limit
	toolCalls := make([]domain.ToolCall, 15) // More than maxToolCalls (10)
	for i := range toolCalls {
		toolCalls[i] = domain.ToolCall{
			ID:   "call_" + string(rune(i)),
			Type: "function",
			Function: domain.FunctionCall{
				Name:      "test_tool",
				Arguments: map[string]interface{}{},
			},
		}
	}

	ctx := context.Background()
	execCtx := &ExecutionContext{RequestID: "test"}

	_, err := coordinator.ProcessToolCalls(ctx, "conv_1", toolCalls, execCtx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many tool calls")
}

func TestCoordinator_HandleToolCallingConversation_NoToolCalls(t *testing.T) {
	coordinator, _, _ := setupCoordinator(t)

	// Setup mock generator that returns no tool calls
	mockGenerator := &MockGenerator{}
	mockGenerator.On("GenerateWithTools", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&domain.GenerationResult{
		Content:   "Simple response without tool calls",
		ToolCalls: []domain.ToolCall{},
		Finished:  true,
	}, nil)

	ctx := context.Background()
	tools := []domain.ToolDefinition{}
	opts := &domain.GenerationOptions{}
	execCtx := &ExecutionContext{RequestID: "test"}

	response, err := coordinator.HandleToolCallingConversation(ctx, mockGenerator, "test prompt", tools, opts, execCtx)

	assert.NoError(t, err)
	assert.Equal(t, "Simple response without tool calls", response.Answer)
	assert.Len(t, response.ToolCalls, 0)
	assert.Len(t, response.ToolsUsed, 0)

	mockGenerator.AssertExpectations(t)
}

func TestCoordinator_HandleToolCallingConversation_WithToolCalls(t *testing.T) {
	coordinator, registry, _ := setupCoordinator(t)

	// Setup mock tool - only mock what's actually called
	mockTool := &MockTool{}
	mockTool.On("Name").Return("test_tool")
	mockTool.On("Parameters").Return(ToolParameters{
		Type: "object",
		Properties: map[string]ToolParameter{
			"input": {
				Type:        "string",
				Description: "Test input",
			},
		},
		Required: []string{"input"},
	})
	mockTool.On("Validate", mock.Anything).Return(nil)
	mockTool.On("Execute", mock.Anything, mock.Anything).Return(&ToolResult{
		Success: true,
		Data:    "tool result",
	}, nil)

	err := registry.Register(mockTool)
	assert.NoError(t, err)

	// Setup mock generator that returns tool calls first, then final response
	mockGenerator := &MockGenerator{}

	// First call returns tool calls
	mockGenerator.On("GenerateWithTools", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&domain.GenerationResult{
		Content: "I need to use a tool",
		ToolCalls: []domain.ToolCall{
			{
				ID:   "call_1",
				Type: "function",
				Function: domain.FunctionCall{
					Name:      "test_tool",
					Arguments: map[string]interface{}{"input": "test"},
				},
			},
		},
		Finished: false,
	}, nil).Once()

	// Second call returns final response
	mockGenerator.On("GenerateWithTools", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&domain.GenerationResult{
		Content:   "Final response with tool result",
		ToolCalls: []domain.ToolCall{},
		Finished:  true,
	}, nil).Once()

	ctx := context.Background()
	tools := []domain.ToolDefinition{}
	opts := &domain.GenerationOptions{}
	execCtx := &ExecutionContext{RequestID: "test"}

	response, err := coordinator.HandleToolCallingConversation(ctx, mockGenerator, "test prompt", tools, opts, execCtx)

	assert.NoError(t, err)
		assert.Equal(t, "I need to use a tool", response.Answer)
	assert.Len(t, response.ToolCalls, 1)
	assert.Contains(t, response.ToolsUsed, "test_tool")

	mockTool.AssertExpectations(t)
	mockGenerator.AssertExpectations(t)
}

func TestCoordinator_HandleToolCallingConversation_MaxRoundsExceeded(t *testing.T) {
	coordinator, registry, _ := setupCoordinator(t)
	coordinator.maxRounds = 2 // Set low limit but allow at least one tool call

	// Setup mock tool - only mock what's actually called
	mockTool := &MockTool{}
	mockTool.On("Name").Return("test_tool")
	mockTool.On("Parameters").Return(ToolParameters{
		Type: "object",
		Properties: map[string]ToolParameter{
			"input": {
				Type:        "string",
				Description: "Test input",
			},
		},
		Required: []string{},
	})
	mockTool.On("Validate", mock.Anything).Return(nil)
	mockTool.On("Execute", mock.Anything, mock.Anything).Return(&ToolResult{
		Success: true,
		Data:    "tool result",
	}, nil)

	err := registry.Register(mockTool)
	assert.NoError(t, err)

	// Setup mock generator that returns tool calls for the first call,
	// then tool calls again for the second call (which will hit the limit)
	mockGenerator := &MockGenerator{}
	mockGenerator.On("GenerateWithTools", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&domain.GenerationResult{
		Content: "I need to use a tool",
		ToolCalls: []domain.ToolCall{
			{
				ID:   "call_1",
				Type: "function",
				Function: domain.FunctionCall{
					Name:      "test_tool",
					Arguments: map[string]interface{}{},
				},
			},
		},
		Finished: false,
	}, nil).Once()

	// Second call also returns tool calls but will exceed max rounds
	mockGenerator.On("GenerateWithTools", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&domain.GenerationResult{
		Content: "I need to use another tool",
		ToolCalls: []domain.ToolCall{
			{
				ID:   "call_2",
				Type: "function",
				Function: domain.FunctionCall{
					Name:      "test_tool",
					Arguments: map[string]interface{}{},
				},
			},
		},
		Finished: false,
	}, nil).Once()

	ctx := context.Background()
	tools := []domain.ToolDefinition{}
	opts := &domain.GenerationOptions{}
	execCtx := &ExecutionContext{RequestID: "test"}

	// Should complete after maxRounds and have executed 1 tool call
	response, err := coordinator.HandleToolCallingConversation(ctx, mockGenerator, "test prompt", tools, opts, execCtx)

	assert.NoError(t, err)
	assert.NotEmpty(t, response.Answer)
	assert.Len(t, response.ToolCalls, 1) // Should have made 1 tool call before hitting the limit
}

func TestCoordinator_GetConversation(t *testing.T) {
	coordinator, _, _ := setupCoordinator(t)

	execCtx := &ExecutionContext{RequestID: "test"}

	// Create a conversation
	conv := coordinator.getOrCreateConversation("conv_1", execCtx)
	assert.NotNil(t, conv)
	assert.Equal(t, "conv_1", conv.ID)

	// Get the same conversation
	retrievedConv, exists := coordinator.GetConversation("conv_1")
	assert.True(t, exists)
	assert.Equal(t, conv.ID, retrievedConv.ID)

	// Try to get non-existent conversation
	_, exists = coordinator.GetConversation("nonexistent")
	assert.False(t, exists)
}

func TestCoordinator_ListConversations(t *testing.T) {
	coordinator, _, _ := setupCoordinator(t)

	execCtx := &ExecutionContext{RequestID: "test"}

	// Initially empty
	conversations := coordinator.ListConversations()
	assert.Len(t, conversations, 0)

	// Create some conversations
	coordinator.getOrCreateConversation("conv_1", execCtx)
	coordinator.getOrCreateConversation("conv_2", execCtx)

	conversations = coordinator.ListConversations()
	assert.Len(t, conversations, 2)
}

func TestCoordinator_GetStats(t *testing.T) {
	coordinator, _, _ := setupCoordinator(t)

	execCtx := &ExecutionContext{RequestID: "test"}

	// Initially all zeros
	stats := coordinator.GetStats()
	assert.Equal(t, 0, stats["total_conversations"])
	assert.Equal(t, 0, stats["completed_count"])
	assert.Equal(t, 0, stats["error_count"])
	assert.Equal(t, 0, stats["active_count"])

	// Create and modify conversations
	conv1 := coordinator.getOrCreateConversation("conv_1", execCtx)
	conv2 := coordinator.getOrCreateConversation("conv_2", execCtx)

	coordinator.markConversationCompleted("conv_1")
	coordinator.markConversationError("conv_2", errors.New("test error"))

	stats = coordinator.GetStats()
	assert.Equal(t, 2, stats["total_conversations"])
	assert.Equal(t, 1, stats["completed_count"])
	assert.Equal(t, 1, stats["error_count"])
	assert.Equal(t, 0, stats["active_count"])

	// Verify the conversations were modified correctly
	assert.True(t, conv1.Completed)
	assert.Equal(t, "test error", conv2.Error)
}

func TestCoordinator_CleanupExpiredConversations(t *testing.T) {
	// Create coordinator with very short TTL
	config := DefaultCoordinatorConfig()
	config.ConversationTTL = 100 * time.Millisecond // Very short for testing

	_, registry, executor := setupCoordinator(t)
	coordinator := NewCoordinator(registry, executor, config)

	execCtx := &ExecutionContext{RequestID: "test"}

	// Create a conversation
	conv := coordinator.getOrCreateConversation("conv_1", execCtx)
	assert.NotNil(t, conv)

	// Should exist initially
	_, exists := coordinator.GetConversation("conv_1")
	assert.True(t, exists)

	// Wait for conversation to expire
	time.Sleep(150 * time.Millisecond)

	// Manually trigger cleanup since the goroutine runs every 5 minutes
	coordinator.mu.Lock()
	now := time.Now()
	for id, conv := range coordinator.conversations {
		if now.Sub(conv.LastActivity) > coordinator.conversationTTL {
			delete(coordinator.conversations, id)
			coordinator.logger.Debug("Cleaned up expired conversation %s", id)
		}
	}
	coordinator.mu.Unlock()

	// Should be cleaned up now
	_, exists = coordinator.GetConversation("conv_1")
	assert.False(t, exists)
}

// Benchmark tests
func BenchmarkCoordinator_ProcessToolCalls(b *testing.B) {
	coordinator, registry, _ := setupCoordinator(&testing.T{})

	// Setup mock tool - only mock what's actually called
	mockTool := &MockTool{}
	mockTool.On("Name").Return("test_tool")
	mockTool.On("Parameters").Return(ToolParameters{
		Type: "object",
		Properties: map[string]ToolParameter{
			"input": {
				Type:        "string",
				Description: "Test input",
			},
		},
		Required: []string{"input"},
	})
	mockTool.On("Validate", mock.Anything).Return(nil)
	mockTool.On("Execute", mock.Anything, mock.Anything).Return(&ToolResult{
		Success: true,
		Data:    "result",
	}, nil)

		_ = registry.Register(mockTool)

	toolCalls := []domain.ToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: domain.FunctionCall{
				Name:      "test_tool",
				Arguments: map[string]interface{}{"input": "test"},
			},
		},
	}

	ctx := context.Background()
	execCtx := &ExecutionContext{RequestID: "bench"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = coordinator.ProcessToolCalls(ctx, "conv_bench", toolCalls, execCtx)
	}
}
