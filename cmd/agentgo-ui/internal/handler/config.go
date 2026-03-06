package handler

import (
	"encoding/json"
	"net/http"
)

type ConfigHandler struct {
	config *Config
}

type Config struct {
	Home          string   `json:"home"`
	MCPAllowedDirs []string `json:"mcpAllowedDirs"`
}

type UpdateConfigRequest struct {
	Home          *string  `json:"home,omitempty"`
	MCPAllowedDirs []string `json:"mcpAllowedDirs,omitempty"`
}

func NewConfigHandler(cfg *Config) *ConfigHandler {
	return &ConfigHandler{config: cfg}
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
		Home:          h.config.Home,
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

	JSONResponse(w, map[string]bool{"success": true})
}
