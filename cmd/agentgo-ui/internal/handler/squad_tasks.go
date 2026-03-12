package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (h *Handler) HandleSquadTasks(w http.ResponseWriter, r *http.Request) {
	if h.squadManager == nil {
		JSONError(w, "Squad manager unavailable", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		captainName := strings.TrimSpace(r.URL.Query().Get("captain_name"))
		afterRaw := strings.TrimSpace(r.URL.Query().Get("after"))
		limit := 20
		if limitRaw := strings.TrimSpace(r.URL.Query().Get("limit")); limitRaw != "" {
			if parsed, err := strconv.Atoi(limitRaw); err == nil && parsed > 0 && parsed <= 200 {
				limit = parsed
			}
		}

		var after time.Time
		if afterRaw != "" {
			if unixMillis, err := strconv.ParseInt(afterRaw, 10, 64); err == nil {
				after = time.UnixMilli(unixMillis)
			}
		}

		JSONResponse(w, map[string]any{
			"tasks": h.squadManager.ListSharedTasks(captainName, after, limit),
		})
	case http.MethodPost:
		var req struct {
			CaptainName string   `json:"captain_name"`
			Message     string   `json:"message"`
			AgentNames  []string `json:"agent_names"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		captainName := strings.TrimSpace(req.CaptainName)
		task, err := h.squadManager.EnqueueSharedTask(r.Context(), captainName, req.AgentNames, strings.TrimSpace(req.Message))
		if err != nil {
			JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		JSONResponse(w, map[string]any{
			"task":        task,
			"ack_message": task.AckMessage,
		})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
