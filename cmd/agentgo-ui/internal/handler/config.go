package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
)

type ConfigHandler struct {
	config      *Config
	configPath  string
}

type Config struct {
	Home           string   `json:"home"`
	MCPAllowedDirs []string `json:"mcpAllowedDirs"`
}

type UpdateConfigRequest struct {
	Home           *string  `json:"home,omitempty"`
	MCPAllowedDirs []string `json:"mcpAllowedDirs,omitempty"`
}

func NewConfigHandler(cfg *Config, configPath string) *ConfigHandler {
	return &ConfigHandler{config: cfg, configPath: configPath}
}

func (h *ConfigHandler) HandleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getConfig(w, r)
	case http.MethodPut:
		h.updateConfig(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ConfigHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	config := Config{
		Home:           h.config.Home,
		MCPAllowedDirs: h.config.MCPAllowedDirs,
	}
	JSONResponse(w, config)
}

func (h *ConfigHandler) updateConfig(w http.ResponseWriter, r *http.Request) {
	var req UpdateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Home != nil {
		h.config.Home = *req.Home
	}

	if req.MCPAllowedDirs != nil {
		h.config.MCPAllowedDirs = req.MCPAllowedDirs
	}

	// Persist to config file
	if err := h.saveConfig(); err != nil {
		JSONError(w, "Failed to save config: " + err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, map[string]bool{"success": true})
}

func (h *ConfigHandler) saveConfig() error {
	// Ensure directory exists
	dir := filepath.Dir(h.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Read existing config
	data := map[string]interface{}{}
	if content, err := os.ReadFile(h.configPath); err == nil {
		json.Unmarshal(content, &data)
	}

	// Update values
	data["home"] = h.config.Home
	data["mcp_allowed_dirs"] = h.config.MCPAllowedDirs

	// Write back
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(h.configPath, content, 0644)
}
