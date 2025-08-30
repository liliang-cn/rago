package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	assert.Empty(t, config.Servers)
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
		Servers: []ServerConfig{
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