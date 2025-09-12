package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestServerConfig(t *testing.T) {
	config := ServerConfig{
		Name:             "test-server",
		Description:      "Test MCP server",
		Command:          []string{"echo", "test"},
		Args:             []string{"arg1", "arg2"},
		WorkingDir:       "/tmp",
		Env:              map[string]string{"TEST": "value"},
		AutoStart:        true,
		RestartOnFailure: true,
		MaxRestarts:      5,
		RestartDelay:     10 * time.Second,
		Capabilities:     []string{"tools", "prompts"},
	}

	// Test basic fields
	assert.Equal(t, "test-server", config.Name)
	assert.Equal(t, "Test MCP server", config.Description)
	assert.Equal(t, []string{"echo", "test"}, config.Command)
	assert.Equal(t, []string{"arg1", "arg2"}, config.Args)
	assert.Equal(t, "/tmp", config.WorkingDir)
	assert.Equal(t, map[string]string{"TEST": "value"}, config.Env)
	assert.True(t, config.AutoStart)
	assert.True(t, config.RestartOnFailure)
	assert.Equal(t, 5, config.MaxRestarts)
	assert.Equal(t, 10*time.Second, config.RestartDelay)
	assert.Equal(t, []string{"tools", "prompts"}, config.Capabilities)
}

func TestConfig(t *testing.T) {
	config := Config{
		Enabled:               true,
		LogLevel:              "debug",
		DefaultTimeout:        45 * time.Second,
		MaxConcurrentRequests: 15,
		HealthCheckInterval:   120 * time.Second,
		Servers:               []string{"config1.json", "config2.json"},
		ServersConfigPath:     "legacy.json",
		LoadedServers:         []ServerConfig{},
	}

	// Test basic fields
	assert.True(t, config.Enabled)
	assert.Equal(t, "debug", config.LogLevel)
	assert.Equal(t, 45*time.Second, config.DefaultTimeout)
	assert.Equal(t, 15, config.MaxConcurrentRequests)
	assert.Equal(t, 120*time.Second, config.HealthCheckInterval)
	assert.Equal(t, []string{"config1.json", "config2.json"}, config.Servers)
	assert.Equal(t, "legacy.json", config.ServersConfigPath)
	assert.Empty(t, config.LoadedServers)
}

func TestSimpleServerConfig(t *testing.T) {
	config := SimpleServerConfig{
		Command:    "test-command",
		Args:       []string{"arg1", "arg2"},
		WorkingDir: "/test",
		Env:        map[string]string{"KEY": "value"},
	}

	// Test JSON marshaling/unmarshaling
	data, err := json.Marshal(config)
	require.NoError(t, err)

	var unmarshaled SimpleServerConfig
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, config, unmarshaled)
}

func TestJSONServersConfig(t *testing.T) {
	config := JSONServersConfig{
		MCPServers: map[string]SimpleServerConfig{
			"server1": {
				Command:    "cmd1",
				Args:       []string{"arg1"},
				WorkingDir: "/path1",
				Env:        map[string]string{"ENV1": "val1"},
			},
			"server2": {
				Command:    "cmd2",
				Args:       []string{"arg2", "arg3"},
				WorkingDir: "/path2",
				Env:        map[string]string{"ENV2": "val2"},
			},
		},
	}

	// Test JSON marshaling/unmarshaling
	data, err := json.Marshal(config)
	require.NoError(t, err)

	var unmarshaled JSONServersConfig
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, config, unmarshaled)
	assert.Len(t, unmarshaled.MCPServers, 2)
	assert.Contains(t, unmarshaled.MCPServers, "server1")
	assert.Contains(t, unmarshaled.MCPServers, "server2")
}

func TestLoadServersFromJSON_Success(t *testing.T) {
	// Create temporary test file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-servers.json")

	testConfig := JSONServersConfig{
		MCPServers: map[string]SimpleServerConfig{
			"test-server": {
				Command:    "echo",
				Args:       []string{"hello"},
				WorkingDir: "/tmp",
				Env:        map[string]string{"TEST": "value"},
			},
			"another-server": {
				Command: "ls",
				Args:    []string{"-la"},
			},
		},
	}

	data, err := json.Marshal(testConfig)
	require.NoError(t, err)

	err = os.WriteFile(configFile, data, 0644)
	require.NoError(t, err)

	// Test loading
	config := &Config{
		Servers: []string{configFile},
	}

	err = config.LoadServersFromJSON()
	require.NoError(t, err)

	// Verify loaded servers
	assert.Len(t, config.LoadedServers, 2)

	// Check test-server
	var testServer *ServerConfig
	var anotherServer *ServerConfig
	for i, server := range config.LoadedServers {
		if server.Name == "test-server" {
			testServer = &config.LoadedServers[i]
		} else if server.Name == "another-server" {
			anotherServer = &config.LoadedServers[i]
		}
	}

	require.NotNil(t, testServer)
	assert.Equal(t, "test-server", testServer.Name)
	assert.Equal(t, "MCP server: test-server", testServer.Description)
	assert.Equal(t, []string{"echo"}, testServer.Command)
	assert.Equal(t, []string{"hello"}, testServer.Args)
	assert.Equal(t, "/tmp", testServer.WorkingDir)
	assert.Equal(t, map[string]string{"TEST": "value"}, testServer.Env)
	assert.True(t, testServer.AutoStart)
	assert.True(t, testServer.RestartOnFailure)
	assert.Equal(t, 3, testServer.MaxRestarts)
	assert.Equal(t, 5*time.Second, testServer.RestartDelay)

	require.NotNil(t, anotherServer)
	assert.Equal(t, "another-server", anotherServer.Name)
	assert.Equal(t, []string{"ls"}, anotherServer.Command)
	assert.Equal(t, []string{"-la"}, anotherServer.Args)
	assert.Empty(t, anotherServer.WorkingDir)
	assert.Empty(t, anotherServer.Env)
}

func TestLoadServersFromJSON_MultipleFiles(t *testing.T) {
	// Create temporary test files
	tempDir := t.TempDir()
	configFile1 := filepath.Join(tempDir, "servers1.json")
	configFile2 := filepath.Join(tempDir, "servers2.json")

	testConfig1 := JSONServersConfig{
		MCPServers: map[string]SimpleServerConfig{
			"server1": {Command: "cmd1"},
		},
	}

	testConfig2 := JSONServersConfig{
		MCPServers: map[string]SimpleServerConfig{
			"server2": {Command: "cmd2"},
		},
	}

	// Write files
	data1, _ := json.Marshal(testConfig1)
	data2, _ := json.Marshal(testConfig2)
	require.NoError(t, os.WriteFile(configFile1, data1, 0644))
	require.NoError(t, os.WriteFile(configFile2, data2, 0644))

	// Test loading multiple files
	config := &Config{
		Servers: []string{configFile1, configFile2},
	}

	err := config.LoadServersFromJSON()
	require.NoError(t, err)

	// Verify both servers loaded
	assert.Len(t, config.LoadedServers, 2)
	serverNames := make([]string, len(config.LoadedServers))
	for i, server := range config.LoadedServers {
		serverNames[i] = server.Name
	}
	assert.Contains(t, serverNames, "server1")
	assert.Contains(t, serverNames, "server2")
}

func TestLoadServersFromJSON_BackwardCompatibility(t *testing.T) {
	// Create temporary test file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "legacy-servers.json")

	testConfig := JSONServersConfig{
		MCPServers: map[string]SimpleServerConfig{
			"legacy-server": {Command: "legacy-cmd"},
		},
	}

	data, _ := json.Marshal(testConfig)
	require.NoError(t, os.WriteFile(configFile, data, 0644))

	// Test loading with legacy ServersConfigPath
	config := &Config{
		Servers:           []string{},
		ServersConfigPath: configFile,
	}

	err := config.LoadServersFromJSON()
	require.NoError(t, err)

	assert.Len(t, config.LoadedServers, 1)
	assert.Equal(t, "legacy-server", config.LoadedServers[0].Name)
}

func TestLoadServersFromJSON_FileNotFound(t *testing.T) {
	config := &Config{
		Servers: []string{"nonexistent-file.json"},
	}

	err := config.LoadServersFromJSON()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load server file")
}

func TestLoadServersFromJSON_InvalidJSON(t *testing.T) {
	// Create temporary file with invalid JSON
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "invalid.json")

	err := os.WriteFile(configFile, []byte("invalid json content"), 0644)
	require.NoError(t, err)

	config := &Config{
		Servers: []string{configFile},
	}

	err = config.LoadServersFromJSON()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse MCP servers config JSON")
}

func TestLoadServersFromJSON_HomeDirectory(t *testing.T) {
	// This tests the home directory path resolution logic
	// We can't easily test the actual home directory logic in unit tests
	// but we can test the relative path behavior

	config := &Config{
		Servers: []string{"nonexistent-relative-path.json"},
	}

	err := config.LoadServersFromJSON()
	assert.Error(t, err)
	// The error should mention failing to read the file
	assert.Contains(t, err.Error(), "failed to read MCP servers config file")
}

func TestLoadServerFile_ClearsPreviousServers(t *testing.T) {
	// Create temporary test file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-servers.json")

	testConfig := JSONServersConfig{
		MCPServers: map[string]SimpleServerConfig{
			"new-server": {Command: "new-cmd"},
		},
	}

	data, _ := json.Marshal(testConfig)
	require.NoError(t, os.WriteFile(configFile, data, 0644))

	// Pre-populate with some servers
	config := &Config{
		Servers: []string{configFile},
		LoadedServers: []ServerConfig{
			{Name: "old-server", Command: []string{"old-cmd"}},
		},
	}

	err := config.LoadServersFromJSON()
	require.NoError(t, err)

	// Should have only the new server, old one should be cleared
	assert.Len(t, config.LoadedServers, 1)
	assert.Equal(t, "new-server", config.LoadedServers[0].Name)
}

func TestToolInfo(t *testing.T) {
	now := time.Now()
	info := ToolInfo{
		ServerName:  "test-server",
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param": map[string]interface{}{
					"type": "string",
				},
			},
		},
		LastUsed:   now,
		UsageCount: 42,
	}

	// Test JSON marshaling/unmarshaling
	data, err := json.Marshal(info)
	require.NoError(t, err)

	var unmarshaled ToolInfo
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, info.ServerName, unmarshaled.ServerName)
	assert.Equal(t, info.Name, unmarshaled.Name)
	assert.Equal(t, info.Description, unmarshaled.Description)
	assert.Equal(t, info.InputSchema, unmarshaled.InputSchema)
	assert.Equal(t, info.UsageCount, unmarshaled.UsageCount)
	// Time might have slight precision differences in JSON round-trip
	assert.WithinDuration(t, info.LastUsed, unmarshaled.LastUsed, time.Second)
}

func TestToolResultTypes(t *testing.T) {
	t.Run("success result", func(t *testing.T) {
		result := ToolResult{
			Success: true,
			Data:    "test data",
		}

		// Test JSON marshaling/unmarshaling
		data, err := json.Marshal(result)
		require.NoError(t, err)

		var unmarshaled ToolResult
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, result, unmarshaled)
	})

	t.Run("error result", func(t *testing.T) {
		result := ToolResult{
			Success: false,
			Error:   "test error",
		}

		// Test JSON marshaling/unmarshaling
		data, err := json.Marshal(result)
		require.NoError(t, err)

		var unmarshaled ToolResult
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, result, unmarshaled)
	})

	t.Run("result with complex data", func(t *testing.T) {
		complexData := map[string]interface{}{
			"string": "value",
			"number": 42.5,
			"array":  []string{"item1", "item2"},
			"nested": map[string]interface{}{
				"key": "nested value",
			},
		}

		result := ToolResult{
			Success: true,
			Data:    complexData,
		}

		data, err := json.Marshal(result)
		require.NoError(t, err)

		var unmarshaled ToolResult
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.True(t, unmarshaled.Success)

		// Check data structure (JSON unmarshal converts string arrays to []interface{})
		resultData := unmarshaled.Data.(map[string]interface{})
		assert.Equal(t, "value", resultData["string"])
		assert.Equal(t, 42.5, resultData["number"])

		// Array becomes []interface{} after JSON unmarshal
		array := resultData["array"].([]interface{})
		assert.Len(t, array, 2)
		assert.Equal(t, "item1", array[0])
		assert.Equal(t, "item2", array[1])

		nested := resultData["nested"].(map[string]interface{})
		assert.Equal(t, "nested value", nested["key"])
	})
}

// Additional comprehensive tests for DefaultConfig
func TestDefaultConfig_Values(t *testing.T) {
	config := DefaultConfig()

	// Test all default values
	assert.False(t, config.Enabled)
	assert.Equal(t, "info", config.LogLevel)
	assert.Equal(t, 30*time.Second, config.DefaultTimeout)
	assert.Equal(t, 10, config.MaxConcurrentRequests)
	assert.Equal(t, 60*time.Second, config.HealthCheckInterval)
	assert.Equal(t, []string{"./mcpServers.json"}, config.Servers)
	assert.Empty(t, config.ServersConfigPath)
	assert.Empty(t, config.LoadedServers)
}

// Test Config field validation
func TestConfig_EdgeCases(t *testing.T) {
	t.Run("zero values", func(t *testing.T) {
		config := Config{}

		// Test with all zero values
		assert.False(t, config.Enabled)
		assert.Empty(t, config.LogLevel)
		assert.Equal(t, time.Duration(0), config.DefaultTimeout)
		assert.Equal(t, 0, config.MaxConcurrentRequests)
		assert.Equal(t, time.Duration(0), config.HealthCheckInterval)
		assert.Empty(t, config.Servers)
		assert.Empty(t, config.ServersConfigPath)
		assert.Empty(t, config.LoadedServers)
	})

	t.Run("negative values", func(t *testing.T) {
		config := Config{
			DefaultTimeout:        -1 * time.Second,
			MaxConcurrentRequests: -5,
			HealthCheckInterval:   -10 * time.Second,
		}

		// These should be handled gracefully by the application
		assert.Equal(t, -1*time.Second, config.DefaultTimeout)
		assert.Equal(t, -5, config.MaxConcurrentRequests)
		assert.Equal(t, -10*time.Second, config.HealthCheckInterval)
	})
}

// Test ServerConfig edge cases
func TestServerConfig_EdgeCases(t *testing.T) {
	t.Run("empty server config", func(t *testing.T) {
		config := ServerConfig{}

		assert.Empty(t, config.Name)
		assert.Empty(t, config.Description)
		assert.Empty(t, config.Command)
		assert.Empty(t, config.Args)
		assert.Empty(t, config.WorkingDir)
		assert.Empty(t, config.Env)
		assert.False(t, config.AutoStart)
		assert.False(t, config.RestartOnFailure)
		assert.Equal(t, 0, config.MaxRestarts)
		assert.Equal(t, time.Duration(0), config.RestartDelay)
		assert.Empty(t, config.Capabilities)
	})

	t.Run("server config with special characters", func(t *testing.T) {
		config := ServerConfig{
			Name:        "test-server-ñ-unicode-测试",
			Description: "Server with special chars: !@#$%^&*()_+-={}[]|\\:;<>?,./'\"`~",
			Command:     []string{"/usr/bin/special-cmd"},
			Args:        []string{"--config", "/path/with spaces/config.json"},
			WorkingDir:  "/path/with spaces",
			Env:         map[string]string{"SPECIAL_VAR": "value with spaces & symbols!"},
		}

		// Test JSON marshaling handles special characters
		data, err := json.Marshal(config)
		require.NoError(t, err)

		var unmarshaled ServerConfig
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, config, unmarshaled)
	})
}

// Test JSON configuration loading edge cases
func TestLoadServersFromJSON_EdgeCases(t *testing.T) {
	t.Run("empty JSON file", func(t *testing.T) {
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "empty.json")

		// Create empty JSON object
		emptyConfig := JSONServersConfig{MCPServers: map[string]SimpleServerConfig{}}
		data, _ := json.Marshal(emptyConfig)
		err := os.WriteFile(configFile, data, 0644)
		require.NoError(t, err)

		config := &Config{Servers: []string{configFile}}
		err = config.LoadServersFromJSON()
		require.NoError(t, err)

		assert.Empty(t, config.LoadedServers)
	})

	t.Run("minimal server config", func(t *testing.T) {
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "minimal.json")

		testConfig := JSONServersConfig{
			MCPServers: map[string]SimpleServerConfig{
				"minimal": {Command: "echo"},
			},
		}

		data, _ := json.Marshal(testConfig)
		err := os.WriteFile(configFile, data, 0644)
		require.NoError(t, err)

		config := &Config{Servers: []string{configFile}}
		err = config.LoadServersFromJSON()
		require.NoError(t, err)

		assert.Len(t, config.LoadedServers, 1)
		server := config.LoadedServers[0]
		assert.Equal(t, "minimal", server.Name)
		assert.Equal(t, []string{"echo"}, server.Command)
		assert.Empty(t, server.Args)
		assert.Empty(t, server.WorkingDir)
		assert.Empty(t, server.Env)
		assert.True(t, server.AutoStart)        // Should default to true
		assert.True(t, server.RestartOnFailure) // Should default to true
	})

	t.Run("large number of servers", func(t *testing.T) {
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "many-servers.json")

		// Create config with 100 servers
		servers := make(map[string]SimpleServerConfig)
		for i := 0; i < 100; i++ {
			servers[fmt.Sprintf("server-%d", i)] = SimpleServerConfig{
				Command: fmt.Sprintf("cmd-%d", i),
				Args:    []string{fmt.Sprintf("arg-%d", i)},
			}
		}

		testConfig := JSONServersConfig{MCPServers: servers}
		data, _ := json.Marshal(testConfig)
		err := os.WriteFile(configFile, data, 0644)
		require.NoError(t, err)

		config := &Config{Servers: []string{configFile}}
		err = config.LoadServersFromJSON()
		require.NoError(t, err)

		assert.Len(t, config.LoadedServers, 100)
	})

	t.Run("file permissions error", func(t *testing.T) {
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "no-read.json")

		// Create file with no read permissions
		testConfig := JSONServersConfig{MCPServers: map[string]SimpleServerConfig{"test": {Command: "echo"}}}
		data, _ := json.Marshal(testConfig)
		err := os.WriteFile(configFile, data, 0000) // No permissions
		require.NoError(t, err)

		config := &Config{Servers: []string{configFile}}
		err = config.LoadServersFromJSON()

		// Should fail to read the file
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read MCP servers config file")
	})
}

// Test concurrent access to Config
func TestConfig_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent LoadServersFromJSON", func(t *testing.T) {
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "concurrent.json")

		testConfig := JSONServersConfig{
			MCPServers: map[string]SimpleServerConfig{
				"test": {Command: "echo"},
			},
		}

		data, _ := json.Marshal(testConfig)
		err := os.WriteFile(configFile, data, 0644)
		require.NoError(t, err)

		config := &Config{Servers: []string{configFile}}

		// Run multiple goroutines concurrently
		const numGoroutines = 10
		errChan := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				errChan <- config.LoadServersFromJSON()
			}()
		}

		// Check all goroutines completed without error
		for i := 0; i < numGoroutines; i++ {
			err := <-errChan
			assert.NoError(t, err)
		}

		// Verify final state
		assert.Len(t, config.LoadedServers, 1)
	})
}

// Test Client struct edge cases
func TestClientStruct(t *testing.T) {
	t.Run("client struct fields", func(t *testing.T) {
		serverConfig := &ServerConfig{
			Name:    "test",
			Command: []string{"echo"},
		}

		client := &Client{
			config:    serverConfig,
			session:   nil,
			tools:     make(map[string]*mcp.Tool),
			connected: false,
		}

		assert.Equal(t, serverConfig, client.config)
		assert.Nil(t, client.session)
		assert.NotNil(t, client.tools)
		assert.False(t, client.connected)
	})
}

// Test ToolInfo with edge cases
func TestToolInfo_EdgeCases(t *testing.T) {
	t.Run("empty tool info", func(t *testing.T) {
		info := ToolInfo{}

		data, err := json.Marshal(info)
		require.NoError(t, err)

		var unmarshaled ToolInfo
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, info, unmarshaled)
	})

	t.Run("tool info with nil schema", func(t *testing.T) {
		info := ToolInfo{
			ServerName:  "server",
			Name:        "tool",
			Description: "desc",
			InputSchema: nil, // nil schema
			UsageCount:  0,
		}

		data, err := json.Marshal(info)
		require.NoError(t, err)

		var unmarshaled ToolInfo
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, info, unmarshaled)
		assert.Nil(t, unmarshaled.InputSchema)
	})

	t.Run("tool info with complex schema", func(t *testing.T) {
		complexSchema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"nested": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"deep": map[string]interface{}{
							"type":        "array",
							"items":       map[string]interface{}{"type": "string"},
							"minItems":    1,
							"maxItems":    10,
							"description": "Deep nested array property",
						},
					},
				},
				"simple": map[string]interface{}{
					"type":    "string",
					"pattern": "^[a-zA-Z0-9]+$",
				},
			},
			"required": []string{"nested", "simple"},
		}

		info := ToolInfo{
			ServerName:  "complex-server",
			Name:        "complex-tool",
			Description: "Tool with complex schema",
			InputSchema: complexSchema,
			LastUsed:    time.Now(),
			UsageCount:  999,
		}

		data, err := json.Marshal(info)
		require.NoError(t, err)

		var unmarshaled ToolInfo
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, info.ServerName, unmarshaled.ServerName)
		assert.Equal(t, info.Name, unmarshaled.Name)
		assert.Equal(t, info.Description, unmarshaled.Description)
		assert.Equal(t, info.UsageCount, unmarshaled.UsageCount)

		// Check schema structure (JSON unmarshal converts numbers to float64 and string arrays to []interface{})
		schema := unmarshaled.InputSchema
		assert.Equal(t, "object", schema["type"])

		properties := schema["properties"].(map[string]interface{})
		nested := properties["nested"].(map[string]interface{})
		nestedProps := nested["properties"].(map[string]interface{})
		deep := nestedProps["deep"].(map[string]interface{})

		assert.Equal(t, "array", deep["type"])
		assert.Equal(t, float64(1), deep["minItems"])  // JSON unmarshal converts to float64
		assert.Equal(t, float64(10), deep["maxItems"]) // JSON unmarshal converts to float64

		required := schema["required"].([]interface{}) // JSON unmarshal converts to []interface{}
		assert.Len(t, required, 2)
		assert.Contains(t, required, "nested")
		assert.Contains(t, required, "simple")
	})
}

// Test suite for comprehensive edge cases
type TypesTestSuite struct {
	suite.Suite
	tempDir string
}

func (suite *TypesTestSuite) SetupTest() {
	suite.tempDir = suite.T().TempDir()
}

func (suite *TypesTestSuite) TestLoadServerFile_AbsolutePath() {
	configFile := filepath.Join(suite.tempDir, "absolute.json")

	testConfig := JSONServersConfig{
		MCPServers: map[string]SimpleServerConfig{
			"absolute-server": {Command: "absolute-cmd"},
		},
	}

	data, _ := json.Marshal(testConfig)
	err := os.WriteFile(configFile, data, 0644)
	suite.Require().NoError(err)

	config := &Config{}
	err = config.loadServerFile(configFile) // Use absolute path
	suite.Require().NoError(err)

	suite.Len(config.LoadedServers, 1)
	suite.Equal("absolute-server", config.LoadedServers[0].Name)
}

func (suite *TypesTestSuite) TestLoadServerFile_RelativePath() {
	// Test relative path handling - this is harder to test reliably
	// because it depends on current working directory
	config := &Config{}
	err := config.loadServerFile("definitely-nonexistent-file.json")
	suite.Error(err)
	suite.Contains(err.Error(), "failed to read MCP servers config file")
}

func TestTypesTestSuite(t *testing.T) {
	suite.Run(t, new(TypesTestSuite))
}
