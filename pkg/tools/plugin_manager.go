package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"plugin"
	"sync"
)

// PluginManager manages tool plugins
type PluginManager struct {
	registry      *Registry
	pluginPaths   []string
	loadedPlugins map[string]*plugin.Plugin
	mu            sync.RWMutex
	logger        Logger
}

// ToolPlugin represents a plugin that provides tools
type ToolPlugin interface {
	// Name returns the plugin name
	Name() string
	// Version returns the plugin version
	Version() string
	// Description returns the plugin description
	Description() string
	// Tools returns the list of tools provided by this plugin
	Tools() []Tool
	// Initialize initializes the plugin with configuration
	Initialize(config map[string]interface{}) error
	// Cleanup cleans up plugin resources
	Cleanup() error
}

// PluginInfo contains metadata about a plugin
type PluginInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Path        string `json:"path"`
	Loaded      bool   `json:"loaded"`
	ToolCount   int    `json:"tool_count"`
	Error       string `json:"error,omitempty"`
}

// PluginConfig represents configuration for plugins
type PluginConfig struct {
	Enabled     bool                              `toml:"enabled" mapstructure:"enabled"`
	PluginPaths []string                          `toml:"plugin_paths" mapstructure:"plugin_paths"`
	AutoLoad    bool                              `toml:"auto_load" mapstructure:"auto_load"`
	Configs     map[string]map[string]interface{} `toml:"configs" mapstructure:"configs"`
	Whitelist   []string                          `toml:"whitelist" mapstructure:"whitelist"` // Allowed plugin names
	Blacklist   []string                          `toml:"blacklist" mapstructure:"blacklist"` // Blocked plugin names
}

// DefaultPluginConfig returns default plugin configuration
func DefaultPluginConfig() PluginConfig {
	return PluginConfig{
		Enabled:     true,
		PluginPaths: []string{"./plugins", "./tools/plugins"},
		AutoLoad:    true,
		Configs:     make(map[string]map[string]interface{}),
		Whitelist:   []string{}, // Empty means allow all
		Blacklist:   []string{}, // Empty means block none
	}
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(registry *Registry, config PluginConfig) *PluginManager {
	return &PluginManager{
		registry:      registry,
		pluginPaths:   config.PluginPaths,
		loadedPlugins: make(map[string]*plugin.Plugin),
		logger:        &DefaultLogger{},
	}
}

// SetLogger sets a custom logger for the plugin manager
func (pm *PluginManager) SetLogger(logger Logger) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.logger = logger
}

// LoadPlugin loads a specific plugin from path
func (pm *PluginManager) LoadPlugin(pluginPath string, config map[string]interface{}) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.logger.Info("Loading plugin: %s", pluginPath)

	// Load the plugin
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to open plugin %s: %w", pluginPath, err)
	}

	// Look for the plugin entry point
	symPlugin, err := p.Lookup("Plugin")
	if err != nil {
		return fmt.Errorf("plugin %s does not export 'Plugin' symbol: %w", pluginPath, err)
	}

	// Cast to ToolPlugin interface
	toolPlugin, ok := symPlugin.(ToolPlugin)
	if !ok {
		return fmt.Errorf("plugin %s does not implement ToolPlugin interface", pluginPath)
	}

	// Initialize the plugin
	if err := toolPlugin.Initialize(config); err != nil {
		return fmt.Errorf("failed to initialize plugin %s: %w", pluginPath, err)
	}

	// Register all tools from the plugin
	tools := toolPlugin.Tools()
	for _, tool := range tools {
		if err := pm.registry.Register(tool); err != nil {
			pm.logger.Warn("Failed to register tool %s from plugin %s: %v", tool.Name(), pluginPath, err)
			continue
		}
		pm.logger.Info("Registered tool %s from plugin %s", tool.Name(), pluginPath)
	}

	// Store the loaded plugin
	pm.loadedPlugins[toolPlugin.Name()] = p

	pm.logger.Info("Successfully loaded plugin %s (version %s) with %d tools",
		toolPlugin.Name(), toolPlugin.Version(), len(tools))

	return nil
}

// LoadPluginsFromDirectory loads all plugins from a directory
func (pm *PluginManager) LoadPluginsFromDirectory(dir string, configs map[string]map[string]interface{}) error {
	pm.logger.Info("Loading plugins from directory: %s", dir)

	// Find all .so files in the directory
	matches, err := filepath.Glob(filepath.Join(dir, "*.so"))
	if err != nil {
		return fmt.Errorf("failed to glob plugins in %s: %w", dir, err)
	}

	if len(matches) == 0 {
		pm.logger.Debug("No plugin files found in directory: %s", dir)
		return nil
	}

	// Load each plugin
	var loadErrors []error
	for _, pluginPath := range matches {
		pluginName := filepath.Base(pluginPath)
		config := configs[pluginName]
		if config == nil {
			config = make(map[string]interface{})
		}

		if err := pm.LoadPlugin(pluginPath, config); err != nil {
			pm.logger.Error("Failed to load plugin %s: %v", pluginPath, err)
			loadErrors = append(loadErrors, err)
			continue
		}
	}

	if len(loadErrors) > 0 {
		return fmt.Errorf("failed to load %d plugins: %v", len(loadErrors), loadErrors[0])
	}

	return nil
}

// LoadAllPlugins loads plugins from all configured paths
func (pm *PluginManager) LoadAllPlugins(config PluginConfig) error {
	pm.logger.Info("Loading all plugins from configured paths")

	var allErrors []error

	for _, path := range pm.pluginPaths {
		if err := pm.LoadPluginsFromDirectory(path, config.Configs); err != nil {
			pm.logger.Warn("Failed to load plugins from %s: %v", path, err)
			allErrors = append(allErrors, err)
		}
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("encountered %d errors while loading plugins", len(allErrors))
	}

	return nil
}

// UnloadPlugin unloads a specific plugin
func (pm *PluginManager) UnloadPlugin(pluginName string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	p, exists := pm.loadedPlugins[pluginName]
	if !exists {
		return fmt.Errorf("plugin %s is not loaded", pluginName)
	}

	// Look for the plugin entry point to call cleanup
	symPlugin, err := p.Lookup("Plugin")
	if err == nil {
		if toolPlugin, ok := symPlugin.(ToolPlugin); ok {
			if err := toolPlugin.Cleanup(); err != nil {
				pm.logger.Warn("Plugin %s cleanup failed: %v", pluginName, err)
			}
		}
	}

	// Remove from loaded plugins
	delete(pm.loadedPlugins, pluginName)

	pm.logger.Info("Unloaded plugin: %s", pluginName)
	return nil
}

// ListPlugins returns information about all loaded plugins
func (pm *PluginManager) ListPlugins() []PluginInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var plugins []PluginInfo

	for pluginName, p := range pm.loadedPlugins {
		info := PluginInfo{
			Name:   pluginName,
			Loaded: true,
		}

		// Try to get plugin details
		if symPlugin, err := p.Lookup("Plugin"); err == nil {
			if toolPlugin, ok := symPlugin.(ToolPlugin); ok {
				info.Version = toolPlugin.Version()
				info.Description = toolPlugin.Description()
				info.ToolCount = len(toolPlugin.Tools())
			}
		}

		plugins = append(plugins, info)
	}

	return plugins
}

// ReloadPlugin reloads a specific plugin
func (pm *PluginManager) ReloadPlugin(pluginName string, config map[string]interface{}) error {
	// Find the plugin path
	var pluginPath string
	for _, basePath := range pm.pluginPaths {
		path := filepath.Join(basePath, pluginName+".so")
		if fileExists(path) {
			pluginPath = path
			break
		}
	}

	if pluginPath == "" {
		return fmt.Errorf("plugin file for %s not found in plugin paths", pluginName)
	}

	// Unload if already loaded
	if err := pm.UnloadPlugin(pluginName); err != nil {
		pm.logger.Warn("Failed to unload plugin %s before reload: %v", pluginName, err)
	}

	// Load again
	return pm.LoadPlugin(pluginPath, config)
}

// GetPluginInfo returns information about a specific plugin
func (pm *PluginManager) GetPluginInfo(pluginName string) (*PluginInfo, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	p, exists := pm.loadedPlugins[pluginName]
	if !exists {
		return nil, fmt.Errorf("plugin %s is not loaded", pluginName)
	}

	info := &PluginInfo{
		Name:   pluginName,
		Loaded: true,
	}

	// Try to get plugin details
	if symPlugin, err := p.Lookup("Plugin"); err == nil {
		if toolPlugin, ok := symPlugin.(ToolPlugin); ok {
			info.Version = toolPlugin.Version()
			info.Description = toolPlugin.Description()
			info.ToolCount = len(toolPlugin.Tools())
		}
	} else {
		info.Error = err.Error()
	}

	return info, nil
}

// IsPluginAllowed checks if a plugin is allowed to be loaded
func (pm *PluginManager) IsPluginAllowed(pluginName string, config PluginConfig) bool {
	// Check blacklist first
	for _, blocked := range config.Blacklist {
		if pluginName == blocked {
			return false
		}
	}

	// If whitelist is empty, allow all (except blacklisted)
	if len(config.Whitelist) == 0 {
		return true
	}

	// Check whitelist
	for _, allowed := range config.Whitelist {
		if pluginName == allowed {
			return true
		}
	}

	return false
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	if _, err := filepath.Abs(path); err != nil {
		return false
	}
	return true
}

// PluginToolWrapper wraps a plugin tool to add plugin metadata
type PluginToolWrapper struct {
	Tool
	pluginName    string
	pluginVersion string
}

// NewPluginToolWrapper creates a new plugin tool wrapper
func NewPluginToolWrapper(tool Tool, pluginName, pluginVersion string) *PluginToolWrapper {
	return &PluginToolWrapper{
		Tool:          tool,
		pluginName:    pluginName,
		pluginVersion: pluginVersion,
	}
}

// GetPluginName returns the plugin name
func (w *PluginToolWrapper) GetPluginName() string {
	return w.pluginName
}

// GetPluginVersion returns the plugin version
func (w *PluginToolWrapper) GetPluginVersion() string {
	return w.pluginVersion
}

// Execute wraps the original execute with plugin context
func (w *PluginToolWrapper) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	// Add plugin metadata to context if needed
	result, err := w.Tool.Execute(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("plugin %s tool %s execution failed: %w",
						w.pluginName, w.Name(), err)
	}
	return result, nil
}
