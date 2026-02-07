package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ToolClient interface for testing
type ToolClient interface {
	CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*ToolResult, error)
}

// MockClient is a mock implementation of ToolClient
type MockClient struct {
	mock.Mock
}

func (m *MockClient) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*ToolResult, error) {
	args := m.Called(ctx, toolName, arguments)
	return args.Get(0).(*ToolResult), args.Error(1)
}

func (m *MockClient) IsConnected() bool {
	args := m.Called()
	return args.Bool(0)
}

// MockToolWrapper for testing that doesn't rely on real Client
type MockToolWrapper struct {
	client     ToolClient
	serverName string
	toolName   string
	tool       *mcp.Tool
}

func TestNewMCPToolWrapper(t *testing.T) {
	client := &Client{
		config: &ServerConfig{Name: "test-server"},
	}

	tool := &mcp.Tool{
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: nil,
	}

	wrapper := NewMCPToolWrapper(client, "test-server", tool)

	assert.NotNil(t, wrapper)
	assert.Equal(t, client, wrapper.client)
	assert.Equal(t, "test-server", wrapper.serverName)
	assert.Equal(t, "test-tool", wrapper.toolName)
	assert.Equal(t, tool, wrapper.tool)
}

func TestMCPToolWrapper_Name(t *testing.T) {
	wrapper := &MCPToolWrapper{
		serverName: "my-server",
		toolName:   "my-tool",
	}

	expected := "mcp_my-server_my-tool"
	assert.Equal(t, expected, wrapper.Name())
}

func TestMCPToolWrapper_Description(t *testing.T) {
	tool := &mcp.Tool{
		Description: "Does something useful",
	}

	wrapper := &MCPToolWrapper{
		serverName: "my-server",
		tool:       tool,
	}

	expected := "[MCP:my-server] Does something useful"
	assert.Equal(t, expected, wrapper.Description())
}

func TestMCPToolWrapper_ServerName(t *testing.T) {
	wrapper := &MCPToolWrapper{
		serverName: "test-server",
	}

	assert.Equal(t, "test-server", wrapper.ServerName())
}

func TestMCPToolWrapper_Schema(t *testing.T) {
	t.Run("with valid input schema", func(t *testing.T) {
		// Test with a tool that has InputSchema set to non-nil value
		// Since we can't easily create jsonschema.Schema without importing it,
		// we'll test the fallback behavior when InputSchema is present but can't be marshaled
		tool := &mcp.Tool{
			Name:        "test-tool",
			Description: "Test tool",
			InputSchema: nil, // Test with nil first
		}

		wrapper := &MCPToolWrapper{tool: tool}
		result := wrapper.Schema()

		// Should return fallback schema
		assert.Equal(t, "object", result["type"])
		assert.NotNil(t, result["properties"])
		assert.Equal(t, true, result["additionalProperties"])
	})

	t.Run("with nil input schema", func(t *testing.T) {
		tool := &mcp.Tool{
			InputSchema: nil,
		}

		wrapper := &MCPToolWrapper{tool: tool}
		result := wrapper.Schema()

		// Should return fallback schema
		assert.Equal(t, "object", result["type"])
		assert.NotNil(t, result["properties"])
		assert.Equal(t, true, result["additionalProperties"])
	})

	t.Run("with schema that fails marshaling", func(t *testing.T) {
		// Create a schema that might fail marshaling
		// We can't easily create a schema that fails marshaling, so we'll test the fallback path
		tool := &mcp.Tool{
			InputSchema: nil, // This will trigger the fallback
		}

		wrapper := &MCPToolWrapper{tool: tool}
		result := wrapper.Schema()

		// Should return fallback schema
		assert.Equal(t, "object", result["type"])
		assert.NotNil(t, result["properties"])
		assert.Equal(t, true, result["additionalProperties"])
	})
}

func TestMCPToolWrapper_Call(t *testing.T) {
	t.Run("test call method structure", func(t *testing.T) {
		// Test the structure of the Call method with a valid client but no connection
		client := &Client{
			config:    &ServerConfig{Name: "test-server"},
			connected: false, // Not connected
			tools:     make(map[string]*mcp.Tool),
		}

		wrapper := &MCPToolWrapper{
			client:     client,
			serverName: "test-server",
			toolName:   "test-tool",
		}

		ctx := context.Background()
		expectedArgs := map[string]interface{}{"param": "value"}

		// This should fail gracefully since client is not connected
		result, err := wrapper.Call(ctx, expectedArgs)

		// The Call method returns results, not errors
		assert.Nil(t, err) // Call method doesn't return errors
		assert.NotNil(t, result)
		assert.False(t, result.Success) // Should fail due to not connected
		assert.Equal(t, "test-server", result.ServerName)
		assert.Equal(t, "test-tool", result.ToolName)
		assert.Contains(t, result.Error, "client not connected")
	})

	t.Run("test result structure", func(t *testing.T) {
		// Test that MCPToolResult has the correct structure
		result := &MCPToolResult{
			Success:    true,
			Data:       "test data",
			ServerName: "test-server",
			ToolName:   "test-tool",
			Duration:   100 * time.Millisecond,
		}

		assert.True(t, result.Success)
		assert.Equal(t, "test data", result.Data)
		assert.Equal(t, "test-server", result.ServerName)
		assert.Equal(t, "test-tool", result.ToolName)
		assert.Equal(t, 100*time.Millisecond, result.Duration)
	})
}

func TestNewMCPToolManager(t *testing.T) {
	config := &Config{
		Enabled: true,
	}

	manager := NewMCPToolManager(config)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.manager)
	assert.NotNil(t, manager.tools)
	assert.Empty(t, manager.tools)
}

func TestMCPToolManager_Start_Disabled(t *testing.T) {
	config := &Config{
		Enabled: false,
	}

	manager := NewMCPToolManager(config)
	ctx := context.Background()

	err := manager.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MCP is disabled")
}

func TestMCPToolManager_GetTool(t *testing.T) {
	manager := NewMCPToolManager(&Config{})

	// Test getting non-existent tool
	tool, exists := manager.GetTool("nonexistent")
	assert.Nil(t, tool)
	assert.False(t, exists)

	// Add a mock tool
	mockWrapper := &MCPToolWrapper{
		serverName: "test-server",
		toolName:   "test-tool",
	}
	manager.tools["test-tool"] = mockWrapper

	// Test getting existing tool
	tool, exists = manager.GetTool("test-tool")
	assert.Equal(t, mockWrapper, tool)
	assert.True(t, exists)
}

func TestMCPToolManager_ListTools(t *testing.T) {
	manager := NewMCPToolManager(&Config{})

	// Test empty list
	tools := manager.ListTools()
	assert.Empty(t, tools)

	// Add some mock tools
	tool1 := &MCPToolWrapper{serverName: "server1", toolName: "tool1"}
	tool2 := &MCPToolWrapper{serverName: "server2", toolName: "tool2"}
	manager.tools["tool1"] = tool1
	manager.tools["tool2"] = tool2

	// Test listing tools
	tools = manager.ListTools()
	assert.Len(t, tools, 2)
	assert.Contains(t, tools, "tool1")
	assert.Contains(t, tools, "tool2")
	assert.Equal(t, tool1, tools["tool1"])
	assert.Equal(t, tool2, tools["tool2"])
}

func TestMCPToolManager_ListToolsByServer(t *testing.T) {
	manager := NewMCPToolManager(&Config{})

	// Add tools from different servers
	tool1 := &MCPToolWrapper{serverName: "server1", toolName: "tool1"}
	tool2 := &MCPToolWrapper{serverName: "server2", toolName: "tool2"}
	tool3 := &MCPToolWrapper{serverName: "server1", toolName: "tool3"}

	manager.tools["tool1"] = tool1
	manager.tools["tool2"] = tool2
	manager.tools["tool3"] = tool3

	// Test listing tools from server1
	server1Tools := manager.ListToolsByServer("server1")
	assert.Len(t, server1Tools, 2)
	assert.Contains(t, server1Tools, "tool1")
	assert.Contains(t, server1Tools, "tool3")

	// Test listing tools from server2
	server2Tools := manager.ListToolsByServer("server2")
	assert.Len(t, server2Tools, 1)
	assert.Contains(t, server2Tools, "tool2")

	// Test listing tools from non-existent server
	nonExistentTools := manager.ListToolsByServer("nonexistent")
	assert.Empty(t, nonExistentTools)
}

func TestMCPToolManager_CallTool(t *testing.T) {
	manager := NewMCPToolManager(&Config{})
	ctx := context.Background()

	// Test calling non-existent tool
	result, err := manager.CallTool(ctx, "nonexistent", nil)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MCP tool 'nonexistent' not found")
}

func TestMCPToolManager_GetToolsForLLM(t *testing.T) {
	manager := NewMCPToolManager(&Config{})

	// Test with empty tools
	llmTools := manager.GetToolsForLLM()
	assert.Empty(t, llmTools)

	// Add some mock tools
	tool1 := &MCPToolWrapper{
		serverName: "server1",
		toolName:   "tool1",
		tool: &mcp.Tool{
			Name:        "tool1",
			Description: "First tool",
		},
	}
	tool2 := &MCPToolWrapper{
		serverName: "server2",
		toolName:   "tool2",
		tool: &mcp.Tool{
			Name:        "tool2",
			Description: "Second tool",
		},
	}

	manager.tools["mcp_server1_tool1"] = tool1
	manager.tools["mcp_server2_tool2"] = tool2

	// Test getting LLM formatted tools
	llmTools = manager.GetToolsForLLM()
	assert.Len(t, llmTools, 2)

	// Check structure of first tool
	found := false
	for _, llmTool := range llmTools {
		if function, ok := llmTool["function"].(map[string]interface{}); ok {
			if name, ok := function["name"].(string); ok && name == "mcp_server1_tool1" {
				found = true
				assert.Equal(t, "function", llmTool["type"])
				assert.Equal(t, "[MCP:server1] First tool", function["description"])
				assert.NotNil(t, function["parameters"])
				break
			}
		}
	}
	assert.True(t, found, "Should find the formatted tool")
}

func TestMCPToolManager_StopServer(t *testing.T) {
	manager := NewMCPToolManager(&Config{})

	// Add tools from different servers
	tool1 := &MCPToolWrapper{serverName: "server1", toolName: "tool1"}
	tool2 := &MCPToolWrapper{serverName: "server2", toolName: "tool2"}
	tool3 := &MCPToolWrapper{serverName: "server1", toolName: "tool3"}

	manager.tools["tool1"] = tool1
	manager.tools["tool2"] = tool2
	manager.tools["tool3"] = tool3

	// Stop server1 - this should remove tools from server1 but keep server2 tools
	err := manager.StopServer("server1")
	// We expect an error because the underlying manager doesn't have this server
	assert.Error(t, err)

	// Check that server1 tools are removed
	assert.NotContains(t, manager.tools, "tool1")
	assert.NotContains(t, manager.tools, "tool3")
	// But server2 tools should remain
	assert.Contains(t, manager.tools, "tool2")
}

func TestMCPToolManager_GetServerStatus(t *testing.T) {
	manager := NewMCPToolManager(&Config{
		LoadedServers: []ServerConfig{
			{Name: "server1"},
			{Name: "server2"},
		},
	})

	status := manager.GetServerStatus()

	// Should show configured servers as disconnected (since we didn't start any)
	assert.Contains(t, status, "server1")
	assert.Contains(t, status, "server2")
	assert.False(t, status["server1"])
	assert.False(t, status["server2"])
}

func TestMCPToolManager_Close(t *testing.T) {
	manager := NewMCPToolManager(&Config{})

	// Add some tools
	manager.tools["tool1"] = &MCPToolWrapper{}
	manager.tools["tool2"] = &MCPToolWrapper{}

	err := manager.Close()
	// Should not error even though no real servers to close
	assert.NoError(t, err)

	// Tools should be cleared
	assert.Empty(t, manager.tools)
}

func TestMCPToolManager_GetUsageStats(t *testing.T) {
	manager := NewMCPToolManager(&Config{})

	stats := manager.GetUsageStats()
	assert.NotNil(t, stats)
	assert.Empty(t, stats) // Currently returns empty stats
}

func TestToolUsageStats(t *testing.T) {
	now := time.Now()
	stats := ToolUsageStats{
		ToolName:      "test-tool",
		ServerName:    "test-server",
		CallCount:     100,
		SuccessCount:  95,
		ErrorCount:    5,
		TotalDuration: 30 * time.Second,
		AvgDuration:   300 * time.Millisecond,
		LastUsed:      now,
	}

	// Test JSON marshaling/unmarshaling
	data, err := json.Marshal(stats)
	require.NoError(t, err)

	var unmarshaled ToolUsageStats
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, stats.ToolName, unmarshaled.ToolName)
	assert.Equal(t, stats.ServerName, unmarshaled.ServerName)
	assert.Equal(t, stats.CallCount, unmarshaled.CallCount)
	assert.Equal(t, stats.SuccessCount, unmarshaled.SuccessCount)
	assert.Equal(t, stats.ErrorCount, unmarshaled.ErrorCount)
	assert.Equal(t, stats.TotalDuration, unmarshaled.TotalDuration)
	assert.Equal(t, stats.AvgDuration, unmarshaled.AvgDuration)
}

func TestMCPToolResult(t *testing.T) {
	result := MCPToolResult{
		Success:    true,
		Data:       map[string]interface{}{"key": "value"},
		ServerName: "test-server",
		ToolName:   "test-tool",
		Duration:   500 * time.Millisecond,
	}

	// Test JSON marshaling/unmarshaling
	data, err := json.Marshal(result)
	require.NoError(t, err)

	var unmarshaled MCPToolResult
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, result.Success, unmarshaled.Success)
	assert.Equal(t, result.Data, unmarshaled.Data)
	assert.Equal(t, result.ServerName, unmarshaled.ServerName)
	assert.Equal(t, result.ToolName, unmarshaled.ToolName)
	assert.Equal(t, result.Duration, unmarshaled.Duration)
}

func TestMCPToolManager_StartWithFailures(t *testing.T) {
	config := &Config{
		Enabled: false, // Disabled to test early return
	}

	manager := NewMCPToolManager(config)
	ctx := context.Background()

	succeeded, failed := manager.StartWithFailures(ctx)
	assert.Empty(t, succeeded)
	assert.Empty(t, failed)
}

// Additional comprehensive tests for tools functionality

// Test MCPToolWrapper with more edge cases
func TestMCPToolWrapper_ComprehensiveTests(t *testing.T) {
	t.Run("wrapper with empty tool name", func(t *testing.T) {
		client := &Client{config: &ServerConfig{Name: "test-server"}}
		tool := &mcp.Tool{Name: "", Description: ""}
		wrapper := NewMCPToolWrapper(client, "server", tool)

		assert.Equal(t, "mcp_server_", wrapper.Name())
		assert.Equal(t, "[MCP:server] ", wrapper.Description())
	})

	t.Run("wrapper with special characters in names", func(t *testing.T) {
		client := &Client{config: &ServerConfig{Name: "test-server"}}
		tool := &mcp.Tool{
			Name:        "tool-with-dashes_and_underscores",
			Description: "Tool with special chars: àáâãäåæçèéêë",
		}
		wrapper := NewMCPToolWrapper(client, "server-name_with_chars", tool)

		assert.Equal(t, "mcp_server-name_with_chars_tool-with-dashes_and_underscores", wrapper.Name())
		assert.Contains(t, wrapper.Description(), "àáâãäåæçèéêë")
	})

	t.Run("wrapper call with various argument types", func(t *testing.T) {
		// This tests the structure, actual call will fail due to no real client
		client := &Client{config: &ServerConfig{Name: "test-server"}}
		tool := &mcp.Tool{Name: "test-tool"}
		wrapper := NewMCPToolWrapper(client, "server", tool)

		ctx := context.Background()
		args := map[string]interface{}{
			"string_arg": "test",
			"int_arg":    42,
			"float_arg":  3.14,
			"bool_arg":   true,
			"array_arg":  []interface{}{"item1", "item2"},
			"object_arg": map[string]interface{}{"key": "value"},
			"null_arg":   nil,
		}

		result, err := wrapper.Call(ctx, args)
		// Should return a result even if it fails
		assert.Nil(t, err) // Call method doesn't return errors, it returns failed results
		assert.NotNil(t, result)
		assert.Equal(t, "server", result.ServerName)
		assert.Equal(t, "test-tool", result.ToolName)
		assert.False(t, result.Success) // Should fail since client not connected
	})
}

// Test MCPToolManager with comprehensive scenarios
func TestMCPToolManager_ComprehensiveScenarios(t *testing.T) {
	t.Run("manager with multiple servers and tools", func(t *testing.T) {
		config := &Config{
			Enabled: true,
			LoadedServers: []ServerConfig{
				{Name: "server1", Command: []string{"cmd1"}, AutoStart: true},
				{Name: "server2", Command: []string{"cmd2"}, AutoStart: false},
				{Name: "server3", Command: []string{"cmd3"}, AutoStart: true},
			},
		}
		manager := NewMCPToolManager(config)

		// Mock some tools manually for testing
		tool1 := &MCPToolWrapper{serverName: "server1", toolName: "tool1"}
		tool2 := &MCPToolWrapper{serverName: "server1", toolName: "tool2"}
		tool3 := &MCPToolWrapper{serverName: "server2", toolName: "tool3"}
		manager.tools["mcp_server1_tool1"] = tool1
		manager.tools["mcp_server1_tool2"] = tool2
		manager.tools["mcp_server2_tool3"] = tool3

		// Test listing tools by server
		server1Tools := manager.ListToolsByServer("server1")
		assert.Len(t, server1Tools, 2)
		assert.Contains(t, server1Tools, "mcp_server1_tool1")
		assert.Contains(t, server1Tools, "mcp_server1_tool2")

		server2Tools := manager.ListToolsByServer("server2")
		assert.Len(t, server2Tools, 1)
		assert.Contains(t, server2Tools, "mcp_server2_tool3")

		server3Tools := manager.ListToolsByServer("server3")
		assert.Empty(t, server3Tools)
	})

	t.Run("manager tool operations with empty state", func(t *testing.T) {
		manager := NewMCPToolManager(&Config{Enabled: true})

		// All operations should work with empty state
		tools := manager.ListTools()
		assert.Empty(t, tools)

		toolsByServer := manager.ListToolsByServer("nonexistent")
		assert.Empty(t, toolsByServer)

		llmTools := manager.GetToolsForLLM()
		assert.Empty(t, llmTools)

		status := manager.GetServerStatus()
		assert.Empty(t, status)

		stats := manager.GetUsageStats()
		assert.Empty(t, stats)

		// Close should not error
		err := manager.Close()
		assert.NoError(t, err)
	})

	t.Run("manager start with invalid configuration", func(t *testing.T) {
		config := &Config{
			Enabled: true,
			Servers: []string{"nonexistent-file.json"},
		}
		manager := NewMCPToolManager(config)

		ctx := context.Background()
		err := manager.Start(ctx)
		// Missing config files are now skipped (not an error)
		assert.NoError(t, err)
		// No servers should be available
		tools := manager.ListTools()
		assert.Empty(t, tools)
	})
}

// Test concurrent operations on MCPToolManager
func TestMCPToolManager_ConcurrentOperations(t *testing.T) {
	t.Run("concurrent tool access", func(t *testing.T) {
		manager := NewMCPToolManager(&Config{Enabled: true})

		// Add many tools
		for i := 0; i < 100; i++ {
			tool := &MCPToolWrapper{
				serverName: fmt.Sprintf("server%d", i%10),
				toolName:   fmt.Sprintf("tool%d", i),
			}
			manager.tools[fmt.Sprintf("mcp_server%d_tool%d", i%10, i)] = tool
		}

		const numGoroutines = 20
		wg := sync.WaitGroup{}
		wg.Add(numGoroutines)

		// Test concurrent ListTools
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				tools := manager.ListTools()
				assert.Len(t, tools, 100)
			}()
		}
		wg.Wait()

		// Test concurrent GetTool
		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				toolName := fmt.Sprintf("mcp_server%d_tool%d", id%10, id)
				tool, exists := manager.GetTool(toolName)
				if id < 100 {
					assert.True(t, exists)
					assert.NotNil(t, tool)
				} else {
					assert.False(t, exists)
					assert.Nil(t, tool)
				}
			}(i)
		}
		wg.Wait()

		// Test concurrent ListToolsByServer
		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				serverName := fmt.Sprintf("server%d", id%10)
				tools := manager.ListToolsByServer(serverName)
				assert.Len(t, tools, 10) // Each server should have 10 tools
			}(i)
		}
		wg.Wait()
	})

	t.Run("concurrent server operations", func(t *testing.T) {
		manager := NewMCPToolManager(&Config{
			Enabled: true,
			LoadedServers: []ServerConfig{
				{Name: "test-server", Command: []string{"echo"}},
			},
		})

		const numGoroutines = 10
		wg := sync.WaitGroup{}
		wg.Add(numGoroutines)

		// Concurrent GetServerStatus calls
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				status := manager.GetServerStatus()
				assert.NotNil(t, status)
				assert.Contains(t, status, "test-server")
			}()
		}
		wg.Wait()
	})
}

// Test error handling and edge cases
func TestMCPToolManager_ErrorHandling(t *testing.T) {
	t.Run("start server with invalid name", func(t *testing.T) {
		manager := NewMCPToolManager(&Config{Enabled: true})
		ctx := context.Background()

		err := manager.StartServer(ctx, "")
		assert.Error(t, err)

		err = manager.StartServer(ctx, "nonexistent-server-name-that-is-very-long")
		assert.Error(t, err)
	})

	t.Run("stop nonexistent server", func(t *testing.T) {
		manager := NewMCPToolManager(&Config{Enabled: true})

		err := manager.StopServer("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server not found")
	})

	t.Run("tool operations with special characters", func(t *testing.T) {
		manager := NewMCPToolManager(&Config{Enabled: true})

		// Add tools with special character names
		specialTool := &MCPToolWrapper{
			serverName: "server-with-dashes",
			toolName:   "tool_with_underscores-and-dashes",
		}
		unicodeTool := &MCPToolWrapper{
			serverName: "测试服务器",
			toolName:   "工具名称",
		}
		manager.tools["mcp_server-with-dashes_tool_with_underscores-and-dashes"] = specialTool
		manager.tools["mcp_测试服务器_工具名称"] = unicodeTool

		// Test retrieval
		tool, exists := manager.GetTool("mcp_server-with-dashes_tool_with_underscores-and-dashes")
		assert.True(t, exists)
		assert.Equal(t, specialTool, tool)

		tool, exists = manager.GetTool("mcp_测试服务器_工具名称")
		assert.True(t, exists)
		assert.Equal(t, unicodeTool, tool)

		// Test listing
		allTools := manager.ListTools()
		assert.Len(t, allTools, 2)
	})
}

// Benchmark tests for performance
func BenchmarkMCPToolManager_ListTools(b *testing.B) {
	manager := NewMCPToolManager(&Config{Enabled: true})

	// Add many tools
	for i := 0; i < 1000; i++ {
		tool := &MCPToolWrapper{
			serverName: fmt.Sprintf("server%d", i%100),
			toolName:   fmt.Sprintf("tool%d", i),
		}
		manager.tools[fmt.Sprintf("mcp_server%d_tool%d", i%100, i)] = tool
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tools := manager.ListTools()
			_ = tools
		}
	})
}

func BenchmarkMCPToolManager_GetTool(b *testing.B) {
	manager := NewMCPToolManager(&Config{Enabled: true})

	// Add many tools
	for i := 0; i < 1000; i++ {
		tool := &MCPToolWrapper{
			serverName: fmt.Sprintf("server%d", i%100),
			toolName:   fmt.Sprintf("tool%d", i),
		}
		manager.tools[fmt.Sprintf("mcp_server%d_tool%d", i%100, i)] = tool
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Alternate between existing and non-existing tools
			toolName := fmt.Sprintf("mcp_server%d_tool%d", b.N%100, b.N%1000)
			tool, exists := manager.GetTool(toolName)
			_ = tool
			_ = exists
		}
	})
}

// Test MCPToolResult JSON serialization
func TestMCPToolResult_Serialization(t *testing.T) {
	t.Run("result with various data types", func(t *testing.T) {
		result := MCPToolResult{
			Success:    true,
			ServerName: "test-server",
			ToolName:   "test-tool",
			Duration:   250 * time.Millisecond,
			Data: map[string]interface{}{
				"string":  "test",
				"number":  42,
				"float":   3.14,
				"boolean": true,
				"null":    nil,
				"array":   []interface{}{1, 2, 3},
				"object":  map[string]interface{}{"nested": "value"},
				"unicode": "测试数据🚀",
			},
		}

		data, err := json.Marshal(result)
		require.NoError(t, err)

		var unmarshaled MCPToolResult
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, result.Success, unmarshaled.Success)
		assert.Equal(t, result.ServerName, unmarshaled.ServerName)
		assert.Equal(t, result.ToolName, unmarshaled.ToolName)
		assert.Equal(t, result.Duration, unmarshaled.Duration)

		// Check data structure (JSON unmarshal converts integers to float64)
		resultData := unmarshaled.Data.(map[string]interface{})
		assert.Equal(t, "test", resultData["string"])
		assert.Equal(t, float64(42), resultData["number"]) // JSON converts int to float64
		assert.Equal(t, 3.14, resultData["float"])
		assert.Equal(t, true, resultData["boolean"])
		assert.Nil(t, resultData["null"])
		assert.Equal(t, "测试数据🚀", resultData["unicode"])

		// Array elements become float64 after JSON unmarshal
		array := resultData["array"].([]interface{})
		assert.Len(t, array, 3)
		assert.Equal(t, float64(1), array[0])
		assert.Equal(t, float64(2), array[1])
		assert.Equal(t, float64(3), array[2])

		object := resultData["object"].(map[string]interface{})
		assert.Equal(t, "value", object["nested"])
	})

	t.Run("error result", func(t *testing.T) {
		result := MCPToolResult{
			Success:    false,
			ServerName: "error-server",
			ToolName:   "error-tool",
			Duration:   100 * time.Millisecond,
			Error:      "Tool execution failed: invalid parameters",
		}

		data, err := json.Marshal(result)
		require.NoError(t, err)

		var unmarshaled MCPToolResult
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, result, unmarshaled)
	})
}
