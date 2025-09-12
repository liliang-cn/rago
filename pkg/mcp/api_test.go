package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockMCPToolManager is a mock implementation for testing
type MockMCPToolManager struct {
	mock.Mock
}

func (m *MockMCPToolManager) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockMCPToolManager) StartWithFailures(ctx context.Context) ([]string, []string) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Get(1).([]string)
}

func (m *MockMCPToolManager) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (*MCPToolResult, error) {
	mockArgs := m.Called(ctx, toolName, args)
	return mockArgs.Get(0).(*MCPToolResult), mockArgs.Error(1)
}

func (m *MockMCPToolManager) ListTools() map[string]*MCPToolWrapper {
	args := m.Called()
	return args.Get(0).(map[string]*MCPToolWrapper)
}

func (m *MockMCPToolManager) ListToolsByServer(serverName string) map[string]*MCPToolWrapper {
	args := m.Called(serverName)
	return args.Get(0).(map[string]*MCPToolWrapper)
}

func (m *MockMCPToolManager) GetToolsForLLM() []map[string]interface{} {
	args := m.Called()
	return args.Get(0).([]map[string]interface{})
}

func (m *MockMCPToolManager) StartServer(ctx context.Context, serverName string) error {
	args := m.Called(ctx, serverName)
	return args.Error(0)
}

func (m *MockMCPToolManager) StopServer(serverName string) error {
	args := m.Called(serverName)
	return args.Error(0)
}

func (m *MockMCPToolManager) GetServerStatus() map[string]bool {
	args := m.Called()
	return args.Get(0).(map[string]bool)
}

func (m *MockMCPToolManager) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestNewMCPService(t *testing.T) {
	config := &Config{
		Enabled:  true,
		LogLevel: "debug",
	}

	service := NewMCPService(config)

	assert.NotNil(t, service)
	assert.NotNil(t, service.toolManager)
	assert.Equal(t, config, service.config)
}

func TestMCPService_Initialize_Disabled(t *testing.T) {
	config := &Config{
		Enabled: false,
	}

	service := NewMCPService(config)
	ctx := context.Background()

	err := service.Initialize(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MCP service is disabled")
}

func TestMCPService_Initialize_Enabled(t *testing.T) {
	config := &Config{
		Enabled: true,
	}

	service := NewMCPService(config)

	// We can't easily test the actual initialization without a real MCP server
	// but we can test the error path
	ctx := context.Background()
	err := service.Initialize(ctx)
	// This will likely fail due to missing server configs, but that's expected
	// The important thing is that it doesn't fail on the "disabled" check
	// Error will be related to loading servers or connecting
	if err != nil {
		assert.NotContains(t, err.Error(), "MCP service is disabled")
	}
}

func TestMCPService_CallTool(t *testing.T) {
	config := &Config{Enabled: true}
	service := NewMCPService(config)

	ctx := context.Background()
	toolName := "test-tool"
	args := map[string]interface{}{"param": "value"}

	// This will fail because no servers are configured, but we can test the method exists
	result, err := service.CallTool(ctx, toolName, args)
	// Expect error since tool doesn't exist
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestMCPService_GetAvailableTools(t *testing.T) {
	config := &Config{Enabled: true}
	service := NewMCPService(config)

	tools := service.GetAvailableTools()
	// Should return empty map since no servers configured
	assert.NotNil(t, tools)
	assert.Empty(t, tools)
}

func TestMCPService_GetToolsByServer(t *testing.T) {
	config := &Config{Enabled: true}
	service := NewMCPService(config)

	tools := service.GetToolsByServer("nonexistent-server")
	assert.NotNil(t, tools)
	assert.Empty(t, tools)
}

func TestMCPService_GetToolsForLLM(t *testing.T) {
	config := &Config{Enabled: true}
	service := NewMCPService(config)

	llmTools := service.GetToolsForLLM()
	assert.NotNil(t, llmTools)
	assert.Empty(t, llmTools)
}

func TestMCPService_IsEnabled(t *testing.T) {
	t.Run("enabled", func(t *testing.T) {
		config := &Config{Enabled: true}
		service := NewMCPService(config)
		assert.True(t, service.IsEnabled())
	})

	t.Run("disabled", func(t *testing.T) {
		config := &Config{Enabled: false}
		service := NewMCPService(config)
		assert.False(t, service.IsEnabled())
	})
}

func TestMCPService_GetConfig(t *testing.T) {
	config := &Config{
		Enabled:  true,
		LogLevel: "info",
	}

	service := NewMCPService(config)
	returnedConfig := service.GetConfig()

	assert.Equal(t, config, returnedConfig)
}

func TestMCPService_GetServerStatus(t *testing.T) {
	config := &Config{Enabled: true}
	service := NewMCPService(config)

	status := service.GetServerStatus()
	assert.NotNil(t, status)
}

func TestMCPService_Close(t *testing.T) {
	config := &Config{Enabled: true}
	service := NewMCPService(config)

	err := service.Close()
	// Should not error even with no servers configured
	assert.NoError(t, err)
}

func TestNewMCPLibraryAPI(t *testing.T) {
	config := &Config{
		Enabled:  true,
		LogLevel: "debug",
	}

	api := NewMCPLibraryAPI(config)

	assert.NotNil(t, api)
	assert.NotNil(t, api.service)
}

func TestMCPLibraryAPI_Start(t *testing.T) {
	config := &Config{Enabled: false}
	api := NewMCPLibraryAPI(config)
	ctx := context.Background()

	err := api.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MCP service is disabled")
}

func TestMCPLibraryAPI_StartWithFailures(t *testing.T) {
	config := &Config{Enabled: false}
	api := NewMCPLibraryAPI(config)
	ctx := context.Background()

	succeeded, failed := api.StartWithFailures(ctx)
	assert.Empty(t, succeeded)
	assert.Empty(t, failed)
}

func TestMCPLibraryAPI_CallTool(t *testing.T) {
	config := &Config{Enabled: true}
	api := NewMCPLibraryAPI(config)

	ctx := context.Background()
	result, err := api.CallTool(ctx, "nonexistent-tool", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
}

func TestMCPLibraryAPI_CallToolWithTimeout(t *testing.T) {
	config := &Config{Enabled: true}
	api := NewMCPLibraryAPI(config)

	timeout := 5 * time.Second
	result, err := api.CallToolWithTimeout("nonexistent-tool", nil, timeout)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestMCPLibraryAPI_ListTools(t *testing.T) {
	config := &Config{Enabled: true}
	api := NewMCPLibraryAPI(config)

	tools := api.ListTools()
	assert.NotNil(t, tools)
	assert.Empty(t, tools)
}

func TestMCPLibraryAPI_GetToolsForLLMIntegration(t *testing.T) {
	config := &Config{Enabled: true}
	api := NewMCPLibraryAPI(config)

	llmTools := api.GetToolsForLLMIntegration()
	assert.NotNil(t, llmTools)
}

func TestMCPLibraryAPI_IsAvailable(t *testing.T) {
	t.Run("available", func(t *testing.T) {
		config := &Config{Enabled: true}
		api := NewMCPLibraryAPI(config)
		assert.True(t, api.IsAvailable())
	})

	t.Run("not available", func(t *testing.T) {
		config := &Config{Enabled: false}
		api := NewMCPLibraryAPI(config)
		assert.False(t, api.IsAvailable())
	})
}

func TestMCPLibraryAPI_GetServerStatuses(t *testing.T) {
	config := &Config{Enabled: true}
	api := NewMCPLibraryAPI(config)

	statuses := api.GetServerStatuses()
	assert.NotNil(t, statuses)
}

func TestMCPLibraryAPI_Stop(t *testing.T) {
	config := &Config{Enabled: true}
	api := NewMCPLibraryAPI(config)

	err := api.Stop()
	assert.NoError(t, err)
}

func TestMCPLibraryAPI_QuickCall(t *testing.T) {
	config := &Config{Enabled: true}
	api := NewMCPLibraryAPI(config)

	options := QuickCallOptions{
		Timeout: 5 * time.Second,
		Args:    map[string]interface{}{"param": "value"},
	}

	result, err := api.QuickCall("nonexistent-tool", options)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestMCPLibraryAPI_QuickCall_DefaultTimeout(t *testing.T) {
	config := &Config{Enabled: true}
	api := NewMCPLibraryAPI(config)

	options := QuickCallOptions{
		// No timeout specified - should use default 30s
		Args: map[string]interface{}{"param": "value"},
	}

	result, err := api.QuickCall("nonexistent-tool", options)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestMCPLibraryAPI_BatchCall(t *testing.T) {
	config := &Config{Enabled: true}
	api := NewMCPLibraryAPI(config)

	calls := []ToolCall{
		{
			ToolName: "tool1",
			Args:     map[string]interface{}{"param": "value1"},
		},
		{
			ToolName: "tool2",
			Args:     map[string]interface{}{"param": "value2"},
		},
	}

	ctx := context.Background()
	results, err := api.BatchCall(ctx, calls)

	// Should error because tools don't exist
	assert.Error(t, err)
	assert.Nil(t, results)
}

func TestMCPLibraryAPI_BatchCall_Empty(t *testing.T) {
	config := &Config{Enabled: true}
	api := NewMCPLibraryAPI(config)

	calls := []ToolCall{}
	ctx := context.Background()

	results, err := api.BatchCall(ctx, calls)
	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestToolSummary(t *testing.T) {
	summary := ToolSummary{
		Name:        "test-tool",
		Description: "A test tool",
		ServerName:  "test-server",
	}

	assert.Equal(t, "test-tool", summary.Name)
	assert.Equal(t, "A test tool", summary.Description)
	assert.Equal(t, "test-server", summary.ServerName)
}

func TestQuickCallOptions(t *testing.T) {
	options := QuickCallOptions{
		Timeout:    10 * time.Second,
		ServerName: "test-server",
		Args:       map[string]interface{}{"key": "value"},
	}

	assert.Equal(t, 10*time.Second, options.Timeout)
	assert.Equal(t, "test-server", options.ServerName)
	assert.Equal(t, map[string]interface{}{"key": "value"}, options.Args)
}

func TestToolCall(t *testing.T) {
	call := ToolCall{
		ToolName: "test-tool",
		Args:     map[string]interface{}{"param": "value"},
	}

	assert.Equal(t, "test-tool", call.ToolName)
	assert.Equal(t, map[string]interface{}{"param": "value"}, call.Args)
}

func TestExampleBasicUsage(t *testing.T) {
	t.Skip("Example function - requires real MCP server for integration test")

	// This would test the ExampleBasicUsage function
	// but it requires a real MCP server to be meaningful
	config := &Config{
		Enabled: false, // Keep disabled to avoid startup issues
	}

	err := ExampleBasicUsage(config)
	assert.Error(t, err) // Should error because MCP is disabled
}

func TestExampleLLMIntegration(t *testing.T) {
	t.Skip("Example function - requires real MCP server for integration test")

	// This would test the ExampleLLMIntegration function
	config := &Config{
		Enabled: false, // Keep disabled
	}

	tools, err := ExampleLLMIntegration(config)
	assert.Error(t, err)
	assert.Nil(t, tools)
}

// Test with mock to demonstrate how the API would work with a real tool manager
func TestMCPLibraryAPI_WithMockedManager(t *testing.T) {
	// This test demonstrates how the API works by mocking the underlying service
	config := &Config{Enabled: true}
	api := NewMCPLibraryAPI(config)

	// We can't easily replace the internal tool manager without changing the API,
	// but we can test the basic structure and ensure methods are called correctly

	// Test that tools list is initially empty
	tools := api.ListTools()
	assert.Empty(t, tools)

	// Test server statuses
	statuses := api.GetServerStatuses()
	assert.NotNil(t, statuses)

	// Test LLM integration tools
	llmTools := api.GetToolsForLLMIntegration()
	assert.NotNil(t, llmTools)
}

// Additional comprehensive tests for MCP API functionality

// Test MCPService with comprehensive scenarios
func TestMCPService_ComprehensiveScenarios(t *testing.T) {
	t.Run("service lifecycle with various configs", func(t *testing.T) {
		// Test with minimal config
		minimalConfig := &Config{Enabled: true}
		service := NewMCPService(minimalConfig)
		assert.NotNil(t, service)
		assert.Equal(t, minimalConfig, service.config)
		assert.True(t, service.IsEnabled())

		// Test with full config
		fullConfig := &Config{
			Enabled:               true,
			LogLevel:              "debug",
			DefaultTimeout:        45 * time.Second,
			MaxConcurrentRequests: 20,
			HealthCheckInterval:   90 * time.Second,
			Servers:               []string{"test1.json", "test2.json"},
		}
		service = NewMCPService(fullConfig)
		assert.Equal(t, fullConfig, service.config)
		assert.True(t, service.IsEnabled())

		// Test with disabled service
		disabledConfig := &Config{Enabled: false}
		service = NewMCPService(disabledConfig)
		assert.False(t, service.IsEnabled())

		ctx := context.Background()
		err := service.Initialize(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "MCP service is disabled")
	})

	t.Run("service operations with empty state", func(t *testing.T) {
		service := NewMCPService(&Config{Enabled: true})

		// All operations should work with empty state
		tools := service.GetAvailableTools()
		assert.Empty(t, tools)

		toolsByServer := service.GetToolsByServer("nonexistent")
		assert.Empty(t, toolsByServer)

		llmTools := service.GetToolsForLLM()
		assert.NotNil(t, llmTools)
		assert.Empty(t, llmTools)

		status := service.GetServerStatus()
		assert.NotNil(t, status)

		config := service.GetConfig()
		assert.NotNil(t, config)
		assert.True(t, config.Enabled)

		// Close should not error
		err := service.Close()
		assert.NoError(t, err)
	})

	t.Run("service error conditions", func(t *testing.T) {
		service := NewMCPService(&Config{Enabled: true})

		ctx := context.Background()

		// CallTool with nonexistent tool
		result, err := service.CallTool(ctx, "nonexistent-tool", nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not found")

		// StartServer with invalid name
		err = service.StartServer(ctx, "invalid-server-name")
		assert.Error(t, err)

		// StopServer with nonexistent server
		err = service.StopServer("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server not found")
	})
}

// Test MCPLibraryAPI with comprehensive scenarios
func TestMCPLibraryAPI_ComprehensiveScenarios(t *testing.T) {
	t.Run("API with various configurations", func(t *testing.T) {
		configs := []*Config{
			{Enabled: true, LogLevel: "debug"},
			{Enabled: true, LogLevel: "info", DefaultTimeout: 60 * time.Second},
			{Enabled: false},
			{Enabled: true, MaxConcurrentRequests: 5},
		}

		for i, config := range configs {
			t.Run(fmt.Sprintf("config_%d", i), func(t *testing.T) {
				api := NewMCPLibraryAPI(config)
				assert.NotNil(t, api)
				assert.Equal(t, config.Enabled, api.IsAvailable())

				// Test basic operations
				tools := api.ListTools()
				assert.NotNil(t, tools)

				statuses := api.GetServerStatuses()
				assert.NotNil(t, statuses)

				llmTools := api.GetToolsForLLMIntegration()
				assert.NotNil(t, llmTools)
			})
		}
	})

	t.Run("API concurrent operations", func(t *testing.T) {
		api := NewMCPLibraryAPI(&Config{Enabled: true})

		const numGoroutines = 20
		wg := sync.WaitGroup{}
		wg.Add(numGoroutines)

		// Test concurrent API calls
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()

				// Multiple operations that should be thread-safe
				tools := api.ListTools()
				_ = tools

				statuses := api.GetServerStatuses()
				_ = statuses

				llmTools := api.GetToolsForLLMIntegration()
				_ = llmTools

				available := api.IsAvailable()
				_ = available
			}()
		}
		wg.Wait()
	})

	t.Run("API batch operations", func(t *testing.T) {
		api := NewMCPLibraryAPI(&Config{Enabled: true})

		// Test empty batch call
		ctx := context.Background()
		emptyCalls := []ToolCall{}
		results, err := api.BatchCall(ctx, emptyCalls)
		assert.NoError(t, err)
		assert.Empty(t, results)

		// Test batch call with nonexistent tools
		calls := []ToolCall{
			{ToolName: "tool1", Args: map[string]interface{}{"param": "value1"}},
			{ToolName: "tool2", Args: map[string]interface{}{"param": "value2"}},
		}
		results, err = api.BatchCall(ctx, calls)
		assert.Error(t, err) // Should fail because tools don't exist
		assert.Nil(t, results)
	})

	t.Run("API timeout handling", func(t *testing.T) {
		api := NewMCPLibraryAPI(&Config{Enabled: true})

		// Test with very short timeout
		result, err := api.CallToolWithTimeout("nonexistent", nil, 1*time.Nanosecond)
		assert.Error(t, err)
		assert.Nil(t, result)

		// Test QuickCall with default timeout
		result, err = api.QuickCall("nonexistent", QuickCallOptions{
			Args: map[string]interface{}{"test": "value"},
		})
		assert.Error(t, err)
		assert.Nil(t, result)

		// Test QuickCall with custom timeout
		result, err = api.QuickCall("nonexistent", QuickCallOptions{
			Timeout: 5 * time.Second,
			Args:    map[string]interface{}{"test": "value"},
		})
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// Test data structures and serialization
func TestMCPAPI_DataStructures(t *testing.T) {
	t.Run("ToolSummary serialization", func(t *testing.T) {
		summary := ToolSummary{
			Name:        "test-tool",
			Description: "A comprehensive test tool with unicode: 测试工具",
			ServerName:  "test-server-测试",
		}

		data, err := json.Marshal(summary)
		require.NoError(t, err)

		var unmarshaled ToolSummary
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, summary, unmarshaled)
	})

	t.Run("QuickCallOptions serialization", func(t *testing.T) {
		options := QuickCallOptions{
			Timeout:    30 * time.Second,
			ServerName: "custom-server",
			Args: map[string]interface{}{
				"string":  "test",
				"number":  42,
				"boolean": true,
				"array":   []interface{}{1, 2, 3},
				"object":  map[string]interface{}{"nested": "value"},
				"unicode": "测试数据",
			},
		}

		data, err := json.Marshal(options)
		require.NoError(t, err)

		var unmarshaled QuickCallOptions
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, options.Timeout, unmarshaled.Timeout)
		assert.Equal(t, options.ServerName, unmarshaled.ServerName)

		// Check args structure (JSON unmarshal converts integers to float64)
		args := unmarshaled.Args
		assert.Equal(t, "test", args["string"])
		assert.Equal(t, float64(42), args["number"]) // JSON converts int to float64
		assert.Equal(t, true, args["boolean"])
		assert.Equal(t, "测试数据", args["unicode"])

		array := args["array"].([]interface{})
		assert.Len(t, array, 3)
		assert.Equal(t, float64(1), array[0])
		assert.Equal(t, float64(2), array[1])
		assert.Equal(t, float64(3), array[2])

		object := args["object"].(map[string]interface{})
		assert.Equal(t, "value", object["nested"])
	})

	t.Run("ToolCall serialization", func(t *testing.T) {
		call := ToolCall{
			ToolName: "complex-tool-name_with-special.chars",
			Args: map[string]interface{}{
				"nested_object": map[string]interface{}{
					"deep_array": []interface{}{
						map[string]interface{}{"id": 1, "name": "item1"},
						map[string]interface{}{"id": 2, "name": "item2"},
					},
				},
				"unicode_string": "中文测试🚀✨",
				"empty_value":    nil,
			},
		}

		data, err := json.Marshal(call)
		require.NoError(t, err)

		var unmarshaled ToolCall
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, call.ToolName, unmarshaled.ToolName)

		// Check args structure (JSON unmarshal converts integers to float64)
		args := unmarshaled.Args
		assert.Equal(t, "中文测试🚀✨", args["unicode_string"])
		assert.Nil(t, args["empty_value"])

		nestedObject := args["nested_object"].(map[string]interface{})
		deepArray := nestedObject["deep_array"].([]interface{})
		assert.Len(t, deepArray, 2)

		item1 := deepArray[0].(map[string]interface{})
		assert.Equal(t, float64(1), item1["id"]) // JSON converts int to float64
		assert.Equal(t, "item1", item1["name"])

		item2 := deepArray[1].(map[string]interface{})
		assert.Equal(t, float64(2), item2["id"]) // JSON converts int to float64
		assert.Equal(t, "item2", item2["name"])
	})
}

// Benchmark tests for API performance
func BenchmarkMCPLibraryAPI_ListTools(b *testing.B) {
	api := NewMCPLibraryAPI(&Config{Enabled: true})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tools := api.ListTools()
			_ = tools
		}
	})
}

// Test error edge cases
func TestMCPAPI_EdgeCases(t *testing.T) {
	t.Run("service operations after close", func(t *testing.T) {
		api := NewMCPLibraryAPI(&Config{Enabled: true})

		// Close the API
		err := api.Stop()
		assert.NoError(t, err)

		// Operations should still work (though they may return empty results)
		tools := api.ListTools()
		assert.NotNil(t, tools)

		statuses := api.GetServerStatuses()
		assert.NotNil(t, statuses)
	})

	t.Run("concurrent start and stop", func(t *testing.T) {
		api := NewMCPLibraryAPI(&Config{Enabled: true})

		const numGoroutines = 10
		wg := sync.WaitGroup{}
		wg.Add(numGoroutines * 2)

		// Concurrent start attempts
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				ctx := context.Background()
				_, _ = api.StartWithFailures(ctx)
			}()
		}

		// Concurrent stop attempts
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				_ = api.Stop()
			}()
		}

		wg.Wait()
		// Should not panic or corrupt state
	})
}
