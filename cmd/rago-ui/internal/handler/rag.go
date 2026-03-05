package handler

import (
	"encoding/json"
	"net/http"

		"github.com/liliang-cn/rago/v2/pkg/rag"
)

// RAG handlers

func (h *Handler) HandleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Query      string `json:"query"`
		Collection string `json:"collection"`
		TopK       int    `json:"top_k"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.TopK == 0 {
		req.TopK = 5
	}

	result, err := h.ragClient.Query(r.Context(), req.Query, &rag.QueryOptions{
		TopK:        req.TopK,
		Temperature: 0.7,
		ShowSources: true,
	})
	if err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, result)
}

func (h *Handler) HandleDocuments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	docs, err := h.ragClient.ListDocuments(r.Context())
	if err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	JSONResponse(w, docs)
}

func (h *Handler) HandleDocumentOperation(w http.ResponseWriter, r *http.Request) {
	docID := r.URL.Path[len("/api/documents/"):]
	if docID == "" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		doc, err := h.ragClient.GetDocument(r.Context(), docID)
		if err != nil {
			JSONError(w, err.Error(), http.StatusNotFound)
			return
		}
		JSONResponse(w, doc)
	case http.MethodDelete:
		if err := h.ragClient.DeleteDocument(r.Context(), docID); err != nil {
			JSONError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		JSONResponse(w, map[string]bool{"success": true})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) HandleCollections(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, _ := h.ragClient.GetStats(r.Context())
	JSONResponse(w, []map[string]interface{}{
		{"name": "default", "count": stats.TotalDocuments},
	})
}

func (h *Handler) HandleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Message   string `json:"message"`
		SessionID string `json:"session_id"`
		Stream    bool   `json:"stream"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	session, _ := h.ragClient.StartChat(r.Context(), map[string]interface{}{"session_id": req.SessionID})
	resp, _ := h.ragClient.Chat(r.Context(), session.ID, req.Message, &rag.QueryOptions{
		Temperature: 0.7,
		ShowSources: false,
	})

	JSONResponse(w, map[string]interface{}{
		"response":   resp.Answer,
		"session_id": session.ID,
	})
}

func (h *Handler) HandleIngest(w http.ResponseWriter, r *http.Request) {
	JSONError(w, "Use CLI to ingest files", http.StatusNotImplemented)
}
