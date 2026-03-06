package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// MCP handlers

func (h *Handler) HandleMCPServers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.mcpService == nil {
		JSONResponse(w, []interface{}{})
		return
	}
	JSONResponse(w, h.mcpService.ListServers())
}

func (h *Handler) HandleMCPTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.mcpService == nil {
		JSONResponse(w, []interface{}{})
		return
	}
	JSONResponse(w, h.mcpService.GetAvailableTools(r.Context()))
}

func (h *Handler) HandleMCPAddServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.mcpService == nil {
		JSONError(w, "MCP service not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Name    string   `json:"name"`
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Name == "" || req.Command == "" {
		JSONError(w, "name and command required", http.StatusBadRequest)
		return
	}

	err := h.mcpService.AddDynamicServer(r.Context(), req.Name, req.Command, req.Args)
	if err != nil {
		JSONError(w, fmt.Sprintf("failed: %v", err), http.StatusInternalServerError)
		return
	}
	JSONResponse(w, map[string]interface{}{"success": true, "name": req.Name})
}

func (h *Handler) HandleMCPCallTool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.mcpService == nil {
		JSONError(w, "MCP service not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		ToolName  string                 `json:"tool_name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	result, err := h.mcpService.CallTool(r.Context(), req.ToolName, req.Arguments)
	if err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	JSONResponse(w, result)
}
