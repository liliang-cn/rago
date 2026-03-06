package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	agentgolog "github.com/liliang-cn/agent-go/pkg/log"
)

// Agent handlers

func (h *Handler) HandleAgentRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Message string `json:"message"`
		Debug   bool   `json:"debug"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Message == "" {
		JSONError(w, "Message required", http.StatusBadRequest)
		return
	}
	if h.agentService == nil {
		JSONError(w, "Agent service unavailable", http.StatusServiceUnavailable)
		return
	}

	start := time.Now()
	result, err := h.agentService.Run(r.Context(), req.Message)
	dur := time.Since(start).Milliseconds()

	if err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	JSONResponse(w, map[string]interface{}{
		"response":    result.Text(),
		"duration_ms": dur,
	})
}

func (h *Handler) HandleAgentStream(w http.ResponseWriter, r *http.Request) {
	agentgolog.Infof("Agent stream request from %s", r.RemoteAddr)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Message   string `json:"message"`
		Debug     bool   `json:"debug"`
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		JSONError(w, "Message required", http.StatusBadRequest)
		return
	}

	svc := h.agentService
	if svc == nil {
		JSONError(w, "Agent service unavailable", http.StatusServiceUnavailable)
		return
	}

	if req.Debug {
		svc.SetDebug(true)
		defer svc.SetDebug(false)
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		JSONError(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	events, err := svc.RunStream(r.Context(), req.Message)
	if err != nil {
		data, _ := json.Marshal(map[string]string{"type": "error", "content": err.Error()})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return
	}

	for evt := range events {
		select {
		case <-r.Context().Done():
			return
		default:
		}

		payload := map[string]interface{}{
			"type":       string(evt.Type),
			"content":    evt.Content,
			"agent_name": evt.AgentName,
		}
		if evt.ToolName != "" {
			payload["tool_name"] = evt.ToolName
		}
		if evt.ToolArgs != nil {
			payload["tool_args"] = evt.ToolArgs
		}
		if evt.ToolResult != nil {
			payload["tool_result"] = evt.ToolResult
		}
		if evt.Round > 0 {
			payload["round"] = evt.Round
			payload["debug_type"] = evt.DebugType
		}

		data, _ := json.Marshal(payload)
		if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
			return
		}
		flusher.Flush()
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}
