package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAddStdioServer(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcpServers.json")

	opts := &AddServerOptions{ConfigFilePath: configPath}

	result, err := AddStdioServer("test-server", "node", []string{"server.js"}, opts)
	if err != nil {
		t.Fatalf("AddStdioServer failed: %v", err)
	}

	if result.ServerName != "test-server" {
		t.Errorf("Expected server name 'test-server', got '%s'", result.ServerName)
	}

	if result.Config.Type != ServerTypeStdio {
		t.Errorf("Expected type 'stdio', got '%s'", result.Config.Type)
	}

	if result.Overwritten {
		t.Error("Should not be overwritten on first add")
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}

func TestAddHTTPServer(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcpServers.json")

	opts := &AddServerOptions{ConfigFilePath: configPath}

	headers := map[string]string{
		"Authorization": "Bearer test123",
	}

	result, err := AddHTTPServer("github", "https://api.github.com/mcp", headers, opts)
	if err != nil {
		t.Fatalf("AddHTTPServer failed: %v", err)
	}

	if result.ServerName != "github" {
		t.Errorf("Expected server name 'github', got '%s'", result.ServerName)
	}

	if result.Config.Type != ServerTypeHTTP {
		t.Errorf("Expected type 'http', got '%s'", result.Config.Type)
	}

	if result.Config.URL != "https://api.github.com/mcp" {
		t.Errorf("Unexpected URL: %s", result.Config.URL)
	}

	if result.Config.Headers["Authorization"] != "Bearer test123" {
		t.Error("Headers not set correctly")
	}
}

func TestAddServerFromJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcpServers.json")

	opts := &AddServerOptions{ConfigFilePath: configPath}

	jsonConfig := `{"type":"http","url":"https://api.example.com/mcp","headers":{"X-API-Key":"secret"}}`

	result, err := AddServerFromJSON("example", jsonConfig, opts)
	if err != nil {
		t.Fatalf("AddServerFromJSON failed: %v", err)
	}

	if result.Config.Type != ServerTypeHTTP {
		t.Errorf("Expected type 'http', got '%s'", result.Config.Type)
	}

	if result.Config.Headers["X-API-Key"] != "secret" {
		t.Error("Headers not parsed correctly")
	}
}

func TestAddServerFromJSON_Stdio(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcpServers.json")

	opts := &AddServerOptions{ConfigFilePath: configPath}

	jsonConfig := `{"command":"node","args":["server.js"],"env":{"DEBUG":"true"}}`

	result, err := AddServerFromJSON("myserver", jsonConfig, opts)
	if err != nil {
		t.Fatalf("AddServerFromJSON failed: %v", err)
	}

	if result.Config.Type != ServerTypeStdio {
		t.Errorf("Expected type 'stdio', got '%s'", result.Config.Type)
	}

	if len(result.Config.Args) != 1 || result.Config.Args[0] != "server.js" {
		t.Errorf("Args not parsed correctly: %v", result.Config.Args)
	}

	if result.Config.Env["DEBUG"] != "true" {
		t.Error("Env not parsed correctly")
	}
}

func TestAddServerFromJSON_Invalid(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcpServers.json")
	opts := &AddServerOptions{ConfigFilePath: configPath}

	// Invalid JSON
	_, err := AddServerFromJSON("test", `{invalid}`, opts)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	// HTTP without URL
	_, err = AddServerFromJSON("test", `{"type":"http"}`, opts)
	if err == nil {
		t.Error("Expected error for HTTP without URL")
	}

	// Stdio without command
	_, err = AddServerFromJSON("test", `{"type":"stdio"}`, opts)
	if err == nil {
		t.Error("Expected error for stdio without command")
	}
}

func TestRemoveServer(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcpServers.json")
	opts := &AddServerOptions{ConfigFilePath: configPath}

	// Add a server first
	_, err := AddStdioServer("to-remove", "node", nil, opts)
	if err != nil {
		t.Fatalf("AddStdioServer failed: %v", err)
	}

	// Remove it
	err = RemoveServer("to-remove", opts)
	if err != nil {
		t.Fatalf("RemoveServer failed: %v", err)
	}

	// Verify it's gone
	servers, err := ListServers(opts)
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}

	if _, exists := servers["to-remove"]; exists {
		t.Error("Server should have been removed")
	}
}

func TestRemoveServer_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcpServers.json")
	opts := &AddServerOptions{ConfigFilePath: configPath}

	err := RemoveServer("nonexistent", opts)
	if err == nil {
		t.Error("Expected error for non-existent server")
	}
}

func TestListServers(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcpServers.json")
	opts := &AddServerOptions{ConfigFilePath: configPath}

	// Add multiple servers
	_, _ = AddStdioServer("server1", "node", nil, opts)
	_, _ = AddHTTPServer("server2", "http://example.com", nil, opts)

	servers, err := ListServers(opts)
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}

	if len(servers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(servers))
	}

	if _, ok := servers["server1"]; !ok {
		t.Error("server1 not found")
	}

	if _, ok := servers["server2"]; !ok {
		t.Error("server2 not found")
	}
}

func TestGetServer(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcpServers.json")
	opts := &AddServerOptions{ConfigFilePath: configPath}

	// Add a server
	_, _ = AddStdioServer("myserver", "python", []string{"-m", "server"}, opts)

	// Get it
	cfg, err := GetServer("myserver", opts)
	if err != nil {
		t.Fatalf("GetServer failed: %v", err)
	}

	if cfg.Command != "python" {
		t.Errorf("Expected command 'python', got '%s'", cfg.Command)
	}
}

func TestGetServer_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcpServers.json")
	opts := &AddServerOptions{ConfigFilePath: configPath}

	_, err := GetServer("nonexistent", opts)
	if err == nil {
		t.Error("Expected error for non-existent server")
	}
}

func TestOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcpServers.json")
	opts := &AddServerOptions{ConfigFilePath: configPath}

	// Add first
	result1, err := AddStdioServer("test", "node", nil, opts)
	if err != nil {
		t.Fatalf("First add failed: %v", err)
	}
	if result1.Overwritten {
		t.Error("First add should not be overwrite")
	}

	// Add again (overwrite)
	result2, err := AddStdioServer("test", "python", nil, opts)
	if err != nil {
		t.Fatalf("Second add failed: %v", err)
	}
	if !result2.Overwritten {
		t.Error("Second add should be overwrite")
	}

	// Verify content
	data, _ := os.ReadFile(configPath)
	var config JSONServersConfig
	json.Unmarshal(data, &config)

	if config.MCPServers["test"].Command != "python" {
		t.Error("Server should have been updated to python")
	}
}

func TestServerConfigToSimple(t *testing.T) {
	cfg := &ServerConfig{
		Name:        "test",
		Type:        ServerTypeStdio,
		Command:     []string{"node", "extra"},
		Args:        []string{"arg1"},
		URL:         "",
		Headers:     nil,
		WorkingDir:  "/tmp",
		Env:         map[string]string{"DEBUG": "true"},
	}

	simple := ServerConfigToSimple(cfg)

	if simple.Type != "stdio" {
		t.Errorf("Expected type 'stdio', got '%s'", simple.Type)
	}

	if simple.Command != "node" {
		t.Errorf("Expected command 'node', got '%s'", simple.Command)
	}

	if simple.WorkingDir != "/tmp" {
		t.Errorf("Expected working dir '/tmp', got '%s'", simple.WorkingDir)
	}

	if simple.Env["DEBUG"] != "true" {
		t.Error("Env not copied")
	}
}

func TestSimpleToServerConfig(t *testing.T) {
	simple := SimpleServerConfig{
		Type:       "http",
		Command:    "curl",
		Args:       []string{"-v"},
		URL:        "https://api.example.com",
		Headers:    map[string]string{"Auth": "token"},
		WorkingDir: "/home",
		Env:        map[string]string{"ENV": "test"},
	}

	cfg := SimpleToServerConfig("test", simple)

	if cfg.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", cfg.Name)
	}

	if cfg.Type != ServerTypeHTTP {
		t.Errorf("Expected type 'http', got '%s'", cfg.Type)
	}

	if cfg.URL != "https://api.example.com" {
		t.Errorf("Unexpected URL: %s", cfg.URL)
	}

	if cfg.Headers["Auth"] != "token" {
		t.Error("Headers not copied")
	}
}
