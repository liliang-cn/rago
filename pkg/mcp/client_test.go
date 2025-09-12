package mcp

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *ServerConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &ServerConfig{
				Name:        "test-server",
				Description: "Test MCP server",
				Command:     []string{"echo", "test"},
				Args:        []string{},
				AutoStart:   false,
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "empty command",
			config: &ServerConfig{
				Name:    "test-server",
				Command: []string{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.Equal(t, tt.config, client.config)
				assert.False(t, client.connected)
				assert.NotNil(t, client.tools)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.False(t, config.Enabled) // Should start disabled
	assert.Equal(t, "info", config.LogLevel)
	assert.Equal(t, 30*time.Second, config.DefaultTimeout)
	assert.Equal(t, 10, config.MaxConcurrentRequests)
	assert.Equal(t, 60*time.Second, config.HealthCheckInterval)
	assert.Equal(t, []string{"./mcpServers.json"}, config.Servers)
	assert.Empty(t, config.LoadedServers)
}

func TestNewManager(t *testing.T) {
	t.Run("with config", func(t *testing.T) {
		config := &Config{
			Enabled:  true,
			LogLevel: "debug",
		}

		manager := NewManager(config)
		assert.NotNil(t, manager)
		assert.Equal(t, config, manager.config)
		assert.NotNil(t, manager.clients)
	})

	t.Run("with nil config", func(t *testing.T) {
		manager := NewManager(nil)
		assert.NotNil(t, manager)
		assert.NotNil(t, manager.config)
		// Should use default config
		assert.False(t, manager.config.Enabled)
		assert.Equal(t, "info", manager.config.LogLevel)
	})
}

func TestManager_GetClient_NotFound(t *testing.T) {
	manager := NewManager(nil)

	client, exists := manager.GetClient("nonexistent")
	assert.Nil(t, client)
	assert.False(t, exists)
}

func TestManager_StartServer_ConfigNotFound(t *testing.T) {
	config := &Config{
		Enabled: true,
		LoadedServers: []ServerConfig{
			{
				Name:    "existing-server",
				Command: []string{"echo"},
			},
		},
	}

	manager := NewManager(config)
	ctx := context.Background()

	_, err := manager.StartServer(ctx, "nonexistent-server")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server configuration not found")
}

func TestToolResult(t *testing.T) {
	t.Run("success result", func(t *testing.T) {
		result := &ToolResult{
			Success: true,
			Data:    "test data",
		}

		assert.True(t, result.Success)
		assert.Equal(t, "test data", result.Data)
		assert.Empty(t, result.Error)
	})

	t.Run("error result", func(t *testing.T) {
		result := &ToolResult{
			Success: false,
			Error:   "test error",
		}

		assert.False(t, result.Success)
		assert.Equal(t, "test error", result.Error)
		assert.Nil(t, result.Data)
	})
}

func TestClient_IsConnected(t *testing.T) {
	config := &ServerConfig{
		Name:    "test-server",
		Command: []string{"echo", "test"},
	}

	client, err := NewClient(config)
	require.NoError(t, err)

	// Initially not connected
	assert.False(t, client.IsConnected())

	// Set connected state manually for testing
	client.connected = true
	assert.True(t, client.IsConnected())
}

func TestClient_GetTools(t *testing.T) {
	config := &ServerConfig{
		Name:    "test-server",
		Command: []string{"echo", "test"},
	}

	client, err := NewClient(config)
	require.NoError(t, err)

	// Initially no tools
	tools := client.GetTools()
	assert.Empty(t, tools)

	// Add some mock tools
	mockTool1 := &mcp.Tool{Name: "tool1"}
	mockTool2 := &mcp.Tool{Name: "tool2"}
	client.tools = map[string]*mcp.Tool{
		"tool1": mockTool1,
		"tool2": mockTool2,
	}

	tools = client.GetTools()
	assert.Len(t, tools, 2)
	assert.Contains(t, tools, "tool1")
	assert.Contains(t, tools, "tool2")
}

func TestClient_GetServerInfo_NotConnected(t *testing.T) {
	config := &ServerConfig{
		Name:    "test-server",
		Command: []string{"echo", "test"},
	}

	client, err := NewClient(config)
	require.NoError(t, err)

	// No session, should return nil
	info := client.GetServerInfo()
	assert.Nil(t, info)
}

func TestClient_Close_NotConnected(t *testing.T) {
	config := &ServerConfig{
		Name:    "test-server",
		Command: []string{"echo", "test"},
	}

	client, err := NewClient(config)
	require.NoError(t, err)

	// Not connected, should not error
	err = client.Close()
	assert.NoError(t, err)
}

func TestClient_Close_Connected(t *testing.T) {
	config := &ServerConfig{
		Name:    "test-server",
		Command: []string{"echo", "test"},
	}

	client, err := NewClient(config)
	require.NoError(t, err)

	// Mock connected state
	client.connected = true
	client.session = nil // Can't easily mock a real session

	err = client.Close()
	// Will likely error because session is nil, but that's expected in unit test
	// The important thing is that connected gets set to false
	assert.False(t, client.connected)
	assert.Nil(t, client.session)
}

func TestManager_ListClients(t *testing.T) {
	manager := NewManager(nil)

	// Initially empty
	clients := manager.ListClients()
	assert.Empty(t, clients)

	// Add some mock clients
	client1, _ := NewClient(&ServerConfig{Name: "server1", Command: []string{"echo"}})
	client2, _ := NewClient(&ServerConfig{Name: "server2", Command: []string{"echo"}})

	manager.clients["server1"] = client1
	manager.clients["server2"] = client2

	clients = manager.ListClients()
	assert.Len(t, clients, 2)
	assert.Contains(t, clients, "server1")
	assert.Contains(t, clients, "server2")
	assert.Equal(t, client1, clients["server1"])
	assert.Equal(t, client2, clients["server2"])
}

func TestManager_StopServer_NotFound(t *testing.T) {
	manager := NewManager(nil)

	err := manager.StopServer("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server not found")
}

func TestManager_StopServer_Success(t *testing.T) {
	manager := NewManager(nil)

	// Add a client
	client, _ := NewClient(&ServerConfig{Name: "test-server", Command: []string{"echo"}})
	manager.clients["test-server"] = client

	_ = manager.StopServer("test-server")
	// May error because client.Close() fails, but client should be removed
	assert.NotContains(t, manager.clients, "test-server")
}

func TestManager_Close(t *testing.T) {
	manager := NewManager(nil)

	// Add some clients
	client1, _ := NewClient(&ServerConfig{Name: "server1", Command: []string{"echo"}})
	client2, _ := NewClient(&ServerConfig{Name: "server2", Command: []string{"echo"}})
	manager.clients["server1"] = client1
	manager.clients["server2"] = client2

	_ = manager.Close()
	// May have errors from closing individual clients, but all should be removed
	assert.Empty(t, manager.clients)
}

func TestManager_StartServer_ExistingConnected(t *testing.T) {
	config := &Config{
		Enabled: true,
		LoadedServers: []ServerConfig{
			{Name: "test-server", Command: []string{"echo"}},
		},
	}
	manager := NewManager(config)

	// Add existing connected client
	existingClient, _ := NewClient(&config.LoadedServers[0])
	existingClient.connected = true
	manager.clients["test-server"] = existingClient

	ctx := context.Background()
	client, err := manager.StartServer(ctx, "test-server")

	// Should return existing client without error
	assert.NoError(t, err)
	assert.Equal(t, existingClient, client)
}

func TestManager_StartServer_ExistingDisconnected(t *testing.T) {
	config := &Config{
		Enabled: true,
		LoadedServers: []ServerConfig{
			{Name: "test-server", Command: []string{"echo"}},
		},
	}
	manager := NewManager(config)

	// Add existing disconnected client
	existingClient, _ := NewClient(&config.LoadedServers[0])
	existingClient.connected = false
	manager.clients["test-server"] = existingClient

	ctx := context.Background()
	_, err := manager.StartServer(ctx, "test-server")

	// Should remove old client and try to create new one
	// Will likely fail because echo isn't an MCP server, but that's expected
	if err == nil {
		// If it succeeds, the old client should be replaced
		newClient := manager.clients["test-server"]
		assert.NotEqual(t, existingClient, newClient)
	} else {
		// If it fails, old client should be removed
		assert.NotContains(t, manager.clients, "test-server")
	}
}

// Integration test that requires a real MCP server would go here
// For now, we'll skip it since we don't have a test MCP server set up
func TestMCPIntegration_Skip(t *testing.T) {
	t.Skip("Integration test requires external MCP server - implement when mcp-sqlite-server is available")

	// This test would:
	// 1. Start a test MCP server (like mcp-sqlite-server)
	// 2. Create a client and connect
	// 3. List tools
	// 4. Call a tool
	// 5. Verify results
	// 6. Clean up
}

// Additional comprehensive tests for client functionality

// Test Client with more edge cases
func TestClient_EdgeCases(t *testing.T) {
	t.Run("NewClient with various config combinations", func(t *testing.T) {
		// Test with minimal config
		minimalConfig := &ServerConfig{
			Name:    "minimal",
			Command: []string{"echo"},
		}
		client, err := NewClient(minimalConfig)
		require.NoError(t, err)
		assert.Equal(t, minimalConfig, client.config)
		assert.False(t, client.connected)
		assert.NotNil(t, client.tools)
		assert.Empty(t, client.tools)

		// Test with full config
		fullConfig := &ServerConfig{
			Name:             "full-server",
			Description:      "Full MCP server",
			Command:          []string{"/usr/bin/mcp-server"},
			Args:             []string{"--config", "test.json"},
			WorkingDir:       "/tmp",
			Env:              map[string]string{"DEBUG": "1", "PORT": "8080"},
			AutoStart:        true,
			RestartOnFailure: true,
			MaxRestarts:      5,
			RestartDelay:     10 * time.Second,
			Capabilities:     []string{"tools", "prompts", "resources"},
		}
		client, err = NewClient(fullConfig)
		require.NoError(t, err)
		assert.Equal(t, fullConfig, client.config)
		assert.False(t, client.connected)
	})

	t.Run("Client method calls when not connected", func(t *testing.T) {
		config := &ServerConfig{
			Name:    "test-server",
			Command: []string{"echo"},
		}
		client, _ := NewClient(config)

		// Test CallTool when not connected
		ctx := context.Background()
		result, err := client.CallTool(ctx, "test-tool", map[string]interface{}{"arg": "value"})
		assert.Nil(t, result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "client not connected")

		// Test loadTools when not connected
		err = client.loadTools(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "client not connected")
	})

	t.Run("Client GetServerInfo when not connected", func(t *testing.T) {
		config := &ServerConfig{
			Name:    "test-server",
			Command: []string{"echo"},
		}
		client, _ := NewClient(config)

		// Test GetServerInfo when session is nil
		serverInfo := client.GetServerInfo()
		assert.Nil(t, serverInfo)
	})
}

// Test Manager with comprehensive edge cases
func TestManager_ComprehensiveEdgeCases(t *testing.T) {
	t.Run("Manager with nil and empty configs", func(t *testing.T) {
		// Test with completely nil config
		manager := NewManager(nil)
		assert.NotNil(t, manager.config)
		assert.False(t, manager.config.Enabled) // Should use default

		// Test with empty config
		emptyConfig := &Config{}
		manager = NewManager(emptyConfig)
		assert.Equal(t, emptyConfig, manager.config)
	})

	t.Run("concurrent access to Manager", func(t *testing.T) {
		manager := NewManager(&Config{Enabled: true})

		// Test concurrent GetClient calls
		const numGoroutines = 10
		wg := sync.WaitGroup{}
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				client, exists := manager.GetClient("nonexistent")
				assert.Nil(t, client)
				assert.False(t, exists)
			}(i)
		}
		wg.Wait()
	})

	t.Run("Manager with loaded servers", func(t *testing.T) {
		config := &Config{
			Enabled: true,
			LoadedServers: []ServerConfig{
				{Name: "server1", Command: []string{"cmd1"}},
				{Name: "server2", Command: []string{"cmd2"}},
				{Name: "server3", Command: []string{"cmd3"}},
			},
		}
		manager := NewManager(config)

		// Test finding servers
		ctx := context.Background()
		_, err := manager.StartServer(ctx, "server1")
		// Will likely fail due to invalid command, but should find the config
		if err != nil {
			assert.NotContains(t, err.Error(), "server configuration not found")
		}

		_, err = manager.StartServer(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server configuration not found")
	})

	t.Run("Manager multiple operations", func(t *testing.T) {
		manager := NewManager(&Config{Enabled: true})

		// Add some mock clients
		client1, _ := NewClient(&ServerConfig{Name: "client1", Command: []string{"echo"}})
		client2, _ := NewClient(&ServerConfig{Name: "client2", Command: []string{"echo"}})
		manager.clients["client1"] = client1
		manager.clients["client2"] = client2

		// Test ListClients
		clients := manager.ListClients()
		assert.Len(t, clients, 2)
		assert.Contains(t, clients, "client1")
		assert.Contains(t, clients, "client2")

		// Test GetClient
		retrieved, exists := manager.GetClient("client1")
		assert.True(t, exists)
		assert.Equal(t, client1, retrieved)

		// Test StopServer
		_ = manager.StopServer("client1")
		// May error due to session being nil, but client should be removed
		assert.NotContains(t, manager.clients, "client1")
		assert.Contains(t, manager.clients, "client2")

		// Test Close
		_ = manager.Close()
		assert.Empty(t, manager.clients)
	})
}

// Test concurrent operations
func TestClient_ConcurrentOperations(t *testing.T) {
	t.Run("concurrent GetTools calls", func(t *testing.T) {
		config := &ServerConfig{
			Name:    "concurrent-server",
			Command: []string{"echo"},
		}
		client, _ := NewClient(config)

		// Add some tools
		client.tools = map[string]*mcp.Tool{
			"tool1": {Name: "tool1", Description: "First tool"},
			"tool2": {Name: "tool2", Description: "Second tool"},
			"tool3": {Name: "tool3", Description: "Third tool"},
		}

		const numGoroutines = 20
		wg := sync.WaitGroup{}
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				tools := client.GetTools()
				assert.Len(t, tools, 3)
				assert.Contains(t, tools, "tool1")
				assert.Contains(t, tools, "tool2")
				assert.Contains(t, tools, "tool3")
			}()
		}
		wg.Wait()
	})

	t.Run("concurrent IsConnected calls", func(t *testing.T) {
		config := &ServerConfig{
			Name:    "concurrent-server",
			Command: []string{"echo"},
		}
		client, _ := NewClient(config)

		const numGoroutines = 20
		wg := sync.WaitGroup{}
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				connected := client.IsConnected()
				assert.False(t, connected) // Should be false initially
			}()
		}
		wg.Wait()

		// Change state and test again
		client.connected = true
		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				connected := client.IsConnected()
				assert.True(t, connected)
			}()
		}
		wg.Wait()
	})
}

// Test error conditions
func TestClient_ErrorConditions(t *testing.T) {
	t.Run("CallTool with nonexistent tool when connected", func(t *testing.T) {
		config := &ServerConfig{
			Name:    "test-server",
			Command: []string{"echo"},
		}
		client, _ := NewClient(config)
		client.connected = true
		// Set session to nil to test error handling
		client.session = nil

		ctx := context.Background()
		result, err := client.CallTool(ctx, "nonexistent-tool", nil)
		assert.Nil(t, result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "client not connected")
	})

	t.Run("various invalid server configurations", func(t *testing.T) {
		testCases := []struct {
			name    string
			config  *ServerConfig
			wantErr bool
			errMsg  string
		}{
			{
				name:    "nil config",
				config:  nil,
				wantErr: true,
				errMsg:  "server config cannot be nil",
			},
			{
				name:    "empty command array",
				config:  &ServerConfig{Name: "test", Command: []string{}},
				wantErr: true,
				errMsg:  "server command cannot be empty",
			},
			{
				name:    "nil command array",
				config:  &ServerConfig{Name: "test", Command: nil},
				wantErr: true,
				errMsg:  "server command cannot be empty",
			},
			{
				name:    "valid config",
				config:  &ServerConfig{Name: "test", Command: []string{"echo"}},
				wantErr: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				client, err := NewClient(tc.config)
				if tc.wantErr {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tc.errMsg)
					assert.Nil(t, client)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, client)
				}
			})
		}
	})
}

// Test suite for complex scenarios
type ClientTestSuite struct {
	suite.Suite
	manager *Manager
	config  *Config
}

func (suite *ClientTestSuite) SetupTest() {
	suite.config = &Config{
		Enabled:               true,
		LogLevel:              "debug",
		DefaultTimeout:        30 * time.Second,
		MaxConcurrentRequests: 10,
		HealthCheckInterval:   60 * time.Second,
		LoadedServers: []ServerConfig{
			{Name: "test-server-1", Command: []string{"echo"}, AutoStart: true},
			{Name: "test-server-2", Command: []string{"ls"}, AutoStart: false},
			{Name: "test-server-3", Command: []string{"pwd"}, AutoStart: true},
		},
	}
	suite.manager = NewManager(suite.config)
}

func (suite *ClientTestSuite) TearDownTest() {
	if suite.manager != nil {
		_ = suite.manager.Close()
	}
}

func (suite *ClientTestSuite) TestManagerLifecycle() {
	// Test manager initialization
	suite.NotNil(suite.manager)
	suite.Equal(suite.config, suite.manager.config)
	suite.Empty(suite.manager.clients)

	// Test finding server configs
	ctx := context.Background()
	_, err := suite.manager.StartServer(ctx, "test-server-1")
	// Expected to fail because echo is not an MCP server, but config should be found
	if err != nil {
		suite.NotContains(err.Error(), "server configuration not found")
	}

	// Test with non-existent server
	_, err = suite.manager.StartServer(ctx, "non-existent")
	suite.Error(err)
	suite.Contains(err.Error(), "server configuration not found")
}

func (suite *ClientTestSuite) TestManagerConcurrentOperations() {
	const numWorkers = 10
	ctx := context.Background()

	// Test concurrent StartServer calls
	wg := sync.WaitGroup{}
	wg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func(id int) {
			defer wg.Done()
			// Try to start the same server concurrently
			_, err := suite.manager.StartServer(ctx, "test-server-1")
			// May succeed or fail, but shouldn't panic or corrupt state
			_ = err
		}(i)
	}
	wg.Wait()

	// Verify manager state is still consistent
	clients := suite.manager.ListClients()
	suite.NotNil(clients)
}

func (suite *ClientTestSuite) TestManagerStateConsistency() {
	// Add some mock clients to test state management
	client1, _ := NewClient(&ServerConfig{Name: "mock1", Command: []string{"echo"}})
	client2, _ := NewClient(&ServerConfig{Name: "mock2", Command: []string{"echo"}})

	suite.manager.clients["mock1"] = client1
	suite.manager.clients["mock2"] = client2

	// Test ListClients
	clients := suite.manager.ListClients()
	suite.Len(clients, 2)

	// Test GetClient
	retrieved, exists := suite.manager.GetClient("mock1")
	suite.True(exists)
	suite.Equal(client1, retrieved)

	// Test StopServer
	_ = suite.manager.StopServer("mock1")
	// May error, but client should be removed from map
	clients = suite.manager.ListClients()
	suite.Len(clients, 1)
	suite.NotContains(clients, "mock1")
	suite.Contains(clients, "mock2")
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

// Benchmark tests for performance
func BenchmarkClient_GetTools(b *testing.B) {
	config := &ServerConfig{
		Name:    "bench-server",
		Command: []string{"echo"},
	}
	client, _ := NewClient(config)

	// Add many tools for realistic benchmark
	for i := 0; i < 1000; i++ {
		client.tools[fmt.Sprintf("tool%d", i)] = &mcp.Tool{
			Name:        fmt.Sprintf("tool%d", i),
			Description: fmt.Sprintf("Tool number %d", i),
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tools := client.GetTools()
			_ = tools
		}
	})
}

func BenchmarkManager_GetClient(b *testing.B) {
	manager := NewManager(&Config{Enabled: true})

	// Add many clients
	for i := 0; i < 100; i++ {
		client, _ := NewClient(&ServerConfig{
			Name:    fmt.Sprintf("client%d", i),
			Command: []string{"echo"},
		})
		manager.clients[fmt.Sprintf("client%d", i)] = client
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Alternate between existing and non-existing clients
			if b.N%2 == 0 {
				client, exists := manager.GetClient("client50")
				_ = client
				_ = exists
			} else {
				client, exists := manager.GetClient("nonexistent")
				_ = client
				_ = exists
			}
		}
	})
}
