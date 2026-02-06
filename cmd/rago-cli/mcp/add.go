package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/spf13/cobra"
)

// serverConfigJSON represents the JSON structure for mcpServers.json
type serverConfigJSON struct {
	MCPServers map[string]serverConfig `json:"mcpServers"`
}

// serverConfig represents a single server configuration in JSON
type serverConfig struct {
	Type       string            `json:"type,omitempty"`
	Command    string            `json:"command,omitempty"`
	Args       []string          `json:"args,omitempty"`
	URL        string            `json:"url,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	WorkingDir string            `json:"working_dir,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
}

// mcpAddCmd adds a new MCP server configuration
var mcpAddCmd = &cobra.Command{
	Use:   "add <server-name> <command>",
	Short: "Add a new MCP server configuration",
	Long: `Add a new MCP server to the configuration.

The server will be added to mcpServers.json and can be used immediately.

Examples:
  # Add a stdio server
  rago mcp add filesystem "npx -y @modelcontextprotocol/server-filesystem /path/to/allowed"

  # Add with description
  rago mcp add my-server "python server.py" --desc "My custom server"

  # Add HTTP server
  rago mcp add http-server "" --url http://localhost:3000/mcp`,
	Args: cobra.MinimumNArgs(2),
	RunE: runMCPAdd,
}

// mcpRemoveCmd removes an MCP server configuration
var mcpRemoveCmd = &cobra.Command{
	Use:   "remove <server-name>",
	Short: "Remove an MCP server configuration",
	Long: `Remove an MCP server from the configuration.

The server will be removed from mcpServers.json.

Examples:
  rago mcp remove filesystem`,
	Args: cobra.ExactArgs(1),
	RunE: runMCPRemove,
}

var (
	addDescription string
	addURL         string
	addArgs        []string
)

func init() {
	MCPCmd.AddCommand(mcpAddCmd)
	MCPCmd.AddCommand(mcpRemoveCmd)

	// Add flags
	mcpAddCmd.Flags().StringVarP(&addDescription, "description", "d", "", "Server description")
	mcpAddCmd.Flags().StringVarP(&addURL, "url", "u", "", "HTTP URL (for HTTP-type servers)")
	mcpAddCmd.Flags().StringSliceVar(&addArgs, "arg", []string{}, "Additional command arguments")
}

func runMCPAdd(cmd *cobra.Command, args []string) error {
	if Cfg == nil {
		var err error
		Cfg, err = config.Load("")
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	serverName := args[0]
	command := args[1]

	// Determine the mcpServers.json file path
	configFile := "./mcpServers.json"
	if len(Cfg.MCP.Servers) > 0 {
		configFile = Cfg.MCP.Servers[0]
	}

	// Also check ~/.rago/ directory
	homeDir, _ := os.UserHomeDir()
	homeConfig := filepath.Join(homeDir, ".rago", "mcpServers.json")

	// Check if file exists in current directory, otherwise use home directory
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if _, err := os.Stat(homeConfig); err == nil {
			configFile = homeConfig
		}
	}

	// Load existing config
	var jsonConfig serverConfigJSON
	if data, err := os.ReadFile(configFile); err == nil {
		if err := json.Unmarshal(data, &jsonConfig); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Check if server already exists
	if _, exists := jsonConfig.MCPServers[serverName]; exists {
		return fmt.Errorf("server %s already exists. Use 'rago mcp remove %s' first", serverName, serverName)
	}

	// Create new server config
	serverCfg := serverConfig{
		Type:    "stdio",
		Command: command,
		Args:    addArgs,
	}

	// Set URL if provided
	if addURL != "" {
		serverCfg.Type = "http"
		serverCfg.URL = addURL
		serverCfg.Command = "" // Clear command for HTTP type
	}

	// Add to config
	if jsonConfig.MCPServers == nil {
		jsonConfig.MCPServers = make(map[string]serverConfig)
	}
	jsonConfig.MCPServers[serverName] = serverCfg

	// Write back to file
	data, err := json.MarshalIndent(jsonConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configFile), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("✅ Added MCP server: %s\n", serverName)
	fmt.Printf("   Type: %s\n", serverCfg.Type)
	if serverCfg.Type == "stdio" {
		fmt.Printf("   Command: %s\n", command)
		if len(addArgs) > 0 {
			fmt.Printf("   Args: %v\n", addArgs)
		}
	} else {
		fmt.Printf("   URL: %s\n", serverCfg.URL)
	}
	fmt.Printf("   Config: %s\n\n", configFile)
	fmt.Printf("💡 Test with: rago mcp discover %s\n", serverName)

	return nil
}

func runMCPRemove(cmd *cobra.Command, args []string) error {
	if Cfg == nil {
		var err error
		Cfg, err = config.Load("")
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	serverName := args[0]

	// Determine the mcpServers.json file path
	configFile := "./mcpServers.json"
	if len(Cfg.MCP.Servers) > 0 {
		configFile = Cfg.MCP.Servers[0]
	}

	// Also check ~/.rago/ directory
	homeDir, _ := os.UserHomeDir()
	homeConfig := filepath.Join(homeDir, ".rago", "mcpServers.json")

	// Check if file exists in current directory, otherwise use home directory
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if _, err := os.Stat(homeConfig); err == nil {
			configFile = homeConfig
		}
	}

	// Load existing config
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config file not found: %s", configFile)
		}
		return fmt.Errorf("failed to read config: %w", err)
	}

	var jsonConfig serverConfigJSON
	if err := json.Unmarshal(data, &jsonConfig); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Check if server exists
	if _, exists := jsonConfig.MCPServers[serverName]; !exists {
		return fmt.Errorf("server %s not found in configuration", serverName)
	}

	// Remove server
	delete(jsonConfig.MCPServers, serverName)

	// Write back to file
	data, err = json.MarshalIndent(jsonConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("✅ Removed MCP server: %s\n", serverName)
	fmt.Printf("   Config: %s\n\n", configFile)

	return nil
}
