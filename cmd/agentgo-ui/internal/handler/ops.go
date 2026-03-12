package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

const maxOpsLogs = 100

type OpsLogEntry struct {
	ID         string                 `json:"id"`
	AgentName  string                 `json:"agent_name"`
	Kind       string                 `json:"kind"`
	Status     string                 `json:"status"`
	Title      string                 `json:"title"`
	Detail     string                 `json:"detail"`
	Timestamp  time.Time              `json:"timestamp"`
	DurationMS *int64                 `json:"duration_ms,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

func (h *Handler) appendOpsLog(entry OpsLogEntry) {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	h.opsLogMu.Lock()
	defer h.opsLogMu.Unlock()

	h.opsLogs = append([]OpsLogEntry{entry}, h.opsLogs...)
	if len(h.opsLogs) > maxOpsLogs {
		h.opsLogs = h.opsLogs[:maxOpsLogs]
	}
}

func (h *Handler) listOpsLogs(limit int) []OpsLogEntry {
	h.opsLogMu.RLock()
	defer h.opsLogMu.RUnlock()

	if limit <= 0 || limit > len(h.opsLogs) {
		limit = len(h.opsLogs)
	}

	result := make([]OpsLogEntry, limit)
	copy(result, h.opsLogs[:limit])
	return result
}

func (h *Handler) HandleOpsLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	JSONResponse(w, map[string]interface{}{
		"logs": h.listOpsLogs(limit),
	})
}
