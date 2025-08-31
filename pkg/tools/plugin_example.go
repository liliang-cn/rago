package tools

// PluginExample demonstrates how to create a tool plugin
// This file shows the interface that external plugins should implement

// ExamplePlugin is a sample plugin implementation
type ExamplePlugin struct {
	name        string
	version     string
	description string
	tools       []Tool
	initialized bool
}

// NewExamplePlugin creates a new example plugin
func NewExamplePlugin() *ExamplePlugin {
	return &ExamplePlugin{
		name:        "example-plugin",
		version:     "1.0.0",
		description: "Example plugin demonstrating the plugin interface",
		tools:       make([]Tool, 0),
	}
}

// Name returns the plugin name
func (p *ExamplePlugin) Name() string {
	return p.name
}

// Version returns the plugin version
func (p *ExamplePlugin) Version() string {
	return p.version
}

// Description returns the plugin description
func (p *ExamplePlugin) Description() string {
	return p.description
}

// Tools returns the list of tools provided by this plugin
func (p *ExamplePlugin) Tools() []Tool {
	if !p.initialized {
		return []Tool{}
	}
	return p.tools
}

// Initialize initializes the plugin with configuration
func (p *ExamplePlugin) Initialize(config map[string]interface{}) error {
	// Initialize your tools here
	// For example, you might create database connections, load configurations, etc.

	// Add your tools to the tools slice
	// p.tools = append(p.tools, &YourCustomTool{})

	p.initialized = true
	return nil
}

// Cleanup cleans up plugin resources
func (p *ExamplePlugin) Cleanup() error {
	// Clean up resources like database connections, files, etc.
	p.initialized = false
	p.tools = nil
	return nil
}

// Plugin is the exported symbol that the plugin manager will look for
// Each plugin .so file must export this symbol
var Plugin ToolPlugin = NewExamplePlugin()

/*
HOW TO CREATE A PLUGIN:

1. Create a Go package for your plugin:

```go
package main

import (
    "context"
    "github.com/liliang-cn/rago/pkg/tools"
)

// YourCustomTool implements the Tool interface
type YourCustomTool struct {
    config map[string]interface{}
}

func (t *YourCustomTool) Name() string {
    return "your_custom_tool"
}

func (t *YourCustomTool) Description() string {
    return "Your custom tool description"
}

func (t *YourCustomTool) Parameters() tools.ToolParameters {
    return tools.ToolParameters{
        Type: "object",
        Properties: map[string]tools.ToolParameter{
            "input": {
                Type: "string",
                Description: "Input parameter",
            },
        },
        Required: []string{"input"},
    }
}

func (t *YourCustomTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
    // Implement your tool logic here
    input, ok := args["input"].(string)
    if !ok {
        return &tools.ToolResult{
            Success: false,
            Error: "input parameter is required and must be a string",
        }, nil
    }

    // Your custom logic
    result := "Processed: " + input

    return &tools.ToolResult{
        Success: true,
        Data: result,
    }, nil
}

func (t *YourCustomTool) Validate(args map[string]interface{}) error {
    if _, ok := args["input"]; !ok {
        return fmt.Errorf("input parameter is required")
    }
    return nil
}

// YourPlugin implements the ToolPlugin interface
type YourPlugin struct {
    tools []tools.Tool
}

func (p *YourPlugin) Name() string {
    return "your-plugin"
}

func (p *YourPlugin) Version() string {
    return "1.0.0"
}

func (p *YourPlugin) Description() string {
    return "Your plugin description"
}

func (p *YourPlugin) Tools() []tools.Tool {
    return p.tools
}

func (p *YourPlugin) Initialize(config map[string]interface{}) error {
    // Initialize your tools
    customTool := &YourCustomTool{config: config}
    p.tools = []tools.Tool{customTool}
    return nil
}

func (p *YourPlugin) Cleanup() error {
    // Cleanup resources
    p.tools = nil
    return nil
}

// Plugin is the exported symbol
var Plugin tools.ToolPlugin = &YourPlugin{}
```

2. Build your plugin as a shared library:

```bash
go build -buildmode=plugin -o your_plugin.so your_plugin.go
```

3. Place the .so file in one of the configured plugin directories

4. Configure the plugin in config.toml:

```toml
[tools.plugins]
enabled = true
auto_load = true
plugin_paths = ["./plugins", "./tools/plugins"]

[tools.plugins.configs.your_plugin]
param1 = "value1"
param2 = "value2"
```

5. The plugin will be automatically loaded and its tools registered
*/
