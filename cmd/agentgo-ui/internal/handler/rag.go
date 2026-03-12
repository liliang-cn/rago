package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/liliang-cn/agent-go/pkg/agent"
	"github.com/liliang-cn/agent-go/pkg/rag"
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

	var raw map[string]any
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if _, ok := raw["messages"]; ok {
		h.handleAISDKChat(w, r, raw)
		return
	}

	var req struct {
		Message   string `json:"message"`
		SessionID string `json:"session_id"`
		Stream    bool   `json:"stream"`
	}
	payload, _ := json.Marshal(raw)
	if err := json.Unmarshal(payload, &req); err != nil {
		JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Message) == "" {
		JSONError(w, "Message required", http.StatusBadRequest)
		return
	}

	session, err := h.ragClient.StartChat(r.Context(), map[string]interface{}{"session_id": req.SessionID})
	if err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp, err := h.ragClient.Chat(r.Context(), session.ID, req.Message, &rag.QueryOptions{
		Temperature: 0.7,
		ShowSources: false,
	})
	if err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if req.Stream {
		streamLegacyChat(w, r, resp.Answer)
		return
	}

	JSONResponse(w, map[string]interface{}{
		"response":   resp.Answer,
		"session_id": session.ID,
	})
}

func (h *Handler) handleAISDKChat(w http.ResponseWriter, r *http.Request, raw map[string]any) {
	prompt := extractLastUserMessage(raw["messages"])
	if strings.TrimSpace(prompt) == "" {
		JSONError(w, "Message required", http.StatusBadRequest)
		return
	}

	mode := strings.TrimSpace(strings.ToLower(stringValue(raw["mode"])))
	if mode == "agent" || stringValue(raw["agent_name"]) != "" {
		h.handleAISDKAgentChat(w, r, raw, prompt)
		return
	}

	externalID, _ := raw["id"].(string)
	sessionID, err := h.getOrCreateAISDKSession(r.Context(), externalID)
	if err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := h.ragClient.Chat(r.Context(), sessionID, prompt, &rag.QueryOptions{
		Temperature: 0.7,
		ShowSources: false,
	})
	if err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	streamAISDKChat(w, r, externalID, resp.Answer)
}

func (h *Handler) handleAISDKAgentChat(w http.ResponseWriter, r *http.Request, raw map[string]any, prompt string) {
	if h.squadManager == nil {
		JSONError(w, "Agent manager unavailable", http.StatusServiceUnavailable)
		return
	}

	agentName := strings.TrimSpace(stringValue(raw["agent_name"]))
	if agentName == "" {
		agentName = "Assistant"
	}

	events, err := h.squadManager.DispatchTaskStream(r.Context(), agentName, prompt)
	if err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	chatID := strings.TrimSpace(stringValue(raw["id"]))
	streamAISDKAgentChat(w, r, chatID, agentName, events)
}

func (h *Handler) getOrCreateAISDKSession(ctx context.Context, externalID string) (string, error) {
	if externalID == "" {
		session, err := h.ragClient.StartChat(ctx, map[string]interface{}{
			"created_via": "ai-sdk",
		})
		if err != nil {
			return "", err
		}
		return session.ID, nil
	}

	h.aiChatMu.RLock()
	if sessionID := h.aiChatSessions[externalID]; sessionID != "" {
		h.aiChatMu.RUnlock()
		return sessionID, nil
	}
	h.aiChatMu.RUnlock()

	session, err := h.ragClient.StartChat(ctx, map[string]interface{}{
		"external_chat_id": externalID,
		"created_via":      "ai-sdk",
	})
	if err != nil {
		return "", err
	}

	h.aiChatMu.Lock()
	h.aiChatSessions[externalID] = session.ID
	h.aiChatMu.Unlock()
	return session.ID, nil
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}

func extractLastUserMessage(messages any) string {
	items, ok := messages.([]any)
	if !ok {
		return ""
	}

	for i := len(items) - 1; i >= 0; i-- {
		message, ok := items[i].(map[string]any)
		if !ok {
			continue
		}
		role, _ := message["role"].(string)
		if role != "user" {
			continue
		}
		if content, ok := message["content"].(string); ok && strings.TrimSpace(content) != "" {
			return content
		}
		parts, ok := message["parts"].([]any)
		if !ok {
			continue
		}
		var textParts []string
		for _, part := range parts {
			partMap, ok := part.(map[string]any)
			if !ok {
				continue
			}
			partType, _ := partMap["type"].(string)
			if partType != "text" {
				continue
			}
			if text, ok := partMap["text"].(string); ok && strings.TrimSpace(text) != "" {
				textParts = append(textParts, text)
			}
		}
		if len(textParts) > 0 {
			return strings.Join(textParts, "\n")
		}
	}

	return ""
}

func streamLegacyChat(w http.ResponseWriter, r *http.Request, text string) {
	setSSEHeaders(w)
	flusher, ok := w.(http.Flusher)
	if !ok {
		JSONError(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	for _, chunk := range chunkText(text, 96) {
		select {
		case <-r.Context().Done():
			return
		default:
		}
		data, _ := json.Marshal(map[string]string{"content": chunk})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func streamAISDKChat(w http.ResponseWriter, r *http.Request, chatID, text string) {
	setSSEHeaders(w)
	flusher, ok := w.(http.Flusher)
	if !ok {
		JSONError(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	messageID := chatID
	if messageID == "" {
		messageID = uuid.New().String()
	}
	textPartID := "text-0"

	writeSSEChunk(w, flusher, map[string]any{
		"type":      "start",
		"messageId": messageID,
	})
	writeSSEChunk(w, flusher, map[string]any{
		"type": "text-start",
		"id":   textPartID,
	})

	for _, chunk := range chunkText(text, 96) {
		select {
		case <-r.Context().Done():
			return
		default:
		}
		writeSSEChunk(w, flusher, map[string]any{
			"type":  "text-delta",
			"id":    textPartID,
			"delta": chunk,
		})
	}

	writeSSEChunk(w, flusher, map[string]any{
		"type": "text-end",
		"id":   textPartID,
	})
	writeSSEChunk(w, flusher, map[string]any{
		"type":         "finish",
		"finishReason": "stop",
		"usage": map[string]int{
			"inputTokens":  0,
			"outputTokens": 0,
			"totalTokens":  0,
		},
	})
}

func streamAISDKAgentChat(w http.ResponseWriter, r *http.Request, chatID, agentName string, events <-chan *agent.Event) {
	setSSEHeaders(w)
	flusher, ok := w.(http.Flusher)
	if !ok {
		JSONError(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	messageID := chatID
	if messageID == "" {
		messageID = uuid.New().String()
	}
	textPartID := "text-0"
	textStarted := false
	partialSeen := false
	toolCalls := make(map[string][]string)
	finishReason := "stop"

	writeSSEChunk(w, flusher, map[string]any{
		"type":      "start",
		"messageId": messageID,
		"messageMetadata": map[string]any{
			"mode":       "agent",
			"agent_name": agentName,
		},
	})

	for evt := range events {
		select {
		case <-r.Context().Done():
			return
		default:
		}

		switch evt.Type {
		case agent.EventTypeThinking, agent.EventTypeDebug, agent.EventTypeHandoff, agent.EventTypeStart:
			writeSSEChunk(w, flusher, map[string]any{
				"type":      "data-agent-event",
				"transient": true,
				"data": map[string]any{
					"event_type":   string(evt.Type),
					"content":      evt.Content,
					"agent_name":   evt.AgentName,
					"tool_name":    evt.ToolName,
					"tool_args":    evt.ToolArgs,
					"debug_type":   evt.DebugType,
					"round":        evt.Round,
					"timestamp":    evt.Timestamp,
					"message_mode": "agent",
				},
			})
		case agent.EventTypePartial:
			if !textStarted {
				writeSSEChunk(w, flusher, map[string]any{
					"type": "text-start",
					"id":   textPartID,
				})
				textStarted = true
			}
			partialSeen = true
			if evt.Content != "" {
				writeSSEChunk(w, flusher, map[string]any{
					"type":  "text-delta",
					"id":    textPartID,
					"delta": evt.Content,
				})
			}
		case agent.EventTypeToolCall:
			callID := uuid.New().String()
			toolCalls[evt.ToolName] = append(toolCalls[evt.ToolName], callID)
			writeSSEChunk(w, flusher, map[string]any{
				"type":       "tool-input-start",
				"toolCallId": callID,
				"toolName":   evt.ToolName,
			})
			inputText := stringifyJSON(evt.ToolArgs)
			if inputText != "" {
				writeSSEChunk(w, flusher, map[string]any{
					"type":           "tool-input-delta",
					"toolCallId":     callID,
					"inputTextDelta": inputText,
				})
			}
			writeSSEChunk(w, flusher, map[string]any{
				"type":       "tool-input-available",
				"toolCallId": callID,
				"toolName":   evt.ToolName,
				"input":      evt.ToolArgs,
			})
		case agent.EventTypeToolResult:
			callID := dequeueToolCallID(toolCalls, evt.ToolName)
			if callID == "" {
				callID = uuid.New().String()
			}
			writeSSEChunk(w, flusher, map[string]any{
				"type":       "tool-output-available",
				"toolCallId": callID,
				"output":     evt.ToolResult,
			})
		case agent.EventTypeComplete:
			if evt.Content != "" && !partialSeen {
				if !textStarted {
					writeSSEChunk(w, flusher, map[string]any{
						"type": "text-start",
						"id":   textPartID,
					})
					textStarted = true
				}
				writeSSEChunk(w, flusher, map[string]any{
					"type":  "text-delta",
					"id":    textPartID,
					"delta": evt.Content,
				})
			}
		case agent.EventTypeError:
			finishReason = "error"
			writeSSEChunk(w, flusher, map[string]any{
				"type":      "error",
				"errorText": evt.Content,
			})
		}
	}

	if textStarted {
		writeSSEChunk(w, flusher, map[string]any{
			"type": "text-end",
			"id":   textPartID,
		})
	}
	writeSSEChunk(w, flusher, map[string]any{
		"type":         "finish",
		"finishReason": finishReason,
		"messageMetadata": map[string]any{
			"mode":       "agent",
			"agent_name": agentName,
		},
	})
}

func setSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
}

func writeSSEChunk(w http.ResponseWriter, flusher http.Flusher, payload map[string]any) {
	data, _ := json.Marshal(payload)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func stringifyJSON(v any) string {
	if v == nil {
		return ""
	}

	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

func dequeueToolCallID(queue map[string][]string, toolName string) string {
	items := queue[toolName]
	if len(items) == 0 {
		return ""
	}

	callID := items[0]
	if len(items) == 1 {
		delete(queue, toolName)
		return callID
	}

	queue[toolName] = items[1:]
	return callID
}

func chunkText(text string, size int) []string {
	if text == "" {
		return []string{""}
	}

	runes := []rune(text)
	if len(runes) <= size {
		return []string{text}
	}

	chunks := make([]string, 0, (len(runes)/size)+1)
	for start := 0; start < len(runes); start += size {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
	}
	return chunks
}

func (h *Handler) HandleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		JSONError(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		JSONError(w, "No file uploaded: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create temp file
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, "agentgo-upload-"+filepath.Base(header.Filename))
	out, err := os.Create(tempFile)
	if err != nil {
		JSONError(w, "Failed to create temp file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer out.Close()

	// Copy uploaded file to temp
	if _, err := io.Copy(out, file); err != nil {
		os.Remove(tempFile)
		JSONError(w, "Failed to save file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	out.Close()

	// Ingest the file
	resp, err := h.ragClient.IngestFile(r.Context(), tempFile, &rag.IngestOptions{
		ChunkSize: 1000,
		Overlap:   200,
	})

	// Clean up temp file
	os.Remove(tempFile)

	if err != nil {
		JSONError(w, "Failed to ingest file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, map[string]interface{}{
		"success":    resp.Success,
		"document":   header.Filename,
		"documentId": resp.DocumentID,
		"chunks":     resp.ChunkCount,
		"message":    resp.Message,
	})
}
