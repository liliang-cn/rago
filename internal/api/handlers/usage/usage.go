package usage

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/liliang-cn/rago/v2/pkg/usage"
)

// Handler handles usage-related HTTP requests
type Handler struct {
	usageService *usage.Service
}

// NewHandler creates a new usage handler
func NewHandler(usageService *usage.Service) *Handler {
	return &Handler{
		usageService: usageService,
	}
}

// RegisterRoutes registers the usage API routes
func (h *Handler) RegisterRoutes(router *mux.Router) {
	// Conversation routes
	router.HandleFunc("/api/v1/conversations", h.ListConversations).Methods("GET")
	router.HandleFunc("/api/v1/conversations", h.CreateConversation).Methods("POST")
	router.HandleFunc("/api/v1/conversations/{id}", h.GetConversation).Methods("GET")
	router.HandleFunc("/api/v1/conversations/{id}", h.DeleteConversation).Methods("DELETE")
	router.HandleFunc("/api/v1/conversations/{id}/messages", h.ListMessages).Methods("GET")
	router.HandleFunc("/api/v1/conversations/{id}/export", h.ExportConversation).Methods("GET")
	
	// Usage statistics routes
	router.HandleFunc("/api/v1/usage/stats", h.GetUsageStats).Methods("GET")
	router.HandleFunc("/api/v1/usage/stats/type", h.GetUsageStatsByType).Methods("GET")
	router.HandleFunc("/api/v1/usage/stats/provider", h.GetUsageStatsByProvider).Methods("GET")
	router.HandleFunc("/api/v1/usage/stats/daily", h.GetDailyUsage).Methods("GET")
	router.HandleFunc("/api/v1/usage/stats/models", h.GetTopModels).Methods("GET")
	router.HandleFunc("/api/v1/usage/stats/cost", h.GetCostByProvider).Methods("GET")
	
	// Usage records routes
	router.HandleFunc("/api/v1/usage/records", h.ListUsageRecords).Methods("GET")
	router.HandleFunc("/api/v1/usage/records/{id}", h.GetUsageRecord).Methods("GET")
	
	// RAG visualization routes
	router.HandleFunc("/api/v1/rag/queries", h.ListRAGQueries).Methods("GET")
	router.HandleFunc("/api/v1/rag/queries/{id}", h.GetRAGQuery).Methods("GET")
	router.HandleFunc("/api/v1/rag/queries/{id}/visualization", h.GetRAGVisualization).Methods("GET")
	router.HandleFunc("/api/v1/rag/analytics", h.GetRAGAnalytics).Methods("GET")
	router.HandleFunc("/api/v1/rag/performance", h.GetRAGPerformanceReport).Methods("GET")
}

// ListConversations lists all conversations
func (h *Handler) ListConversations(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	
	offsetStr := r.URL.Query().Get("offset")
	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}
	
	conversations, err := h.usageService.ListConversations(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Create response without user_id
	response := map[string]interface{}{
		"success": true,
		"data":    []map[string]interface{}{},
		"page":    1,
		"per_page": limit,
		"total":   len(conversations),
		"has_more": false,
	}
	
	data := make([]map[string]interface{}, len(conversations))
	for i, conv := range conversations {
		data[i] = map[string]interface{}{
			"id":            conv.ID,
			"title":         conv.Title,
			"created_at":    conv.CreatedAt.Unix(),
			"updated_at":    conv.UpdatedAt.Unix(),
			"message_count": 0,
			"last_message":  "",
		}
	}
	response["data"] = data
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreateConversation creates a new conversation
func (h *Handler) CreateConversation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title  string `json:"title"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if req.Title == "" {
		req.Title = "New Conversation"
	}
	
	conversation, err := h.usageService.StartConversation(r.Context(), req.Title)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(conversation)
}

// GetConversation gets a conversation with its messages
func (h *Handler) GetConversation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	conversation, err := h.usageService.GetConversationHistory(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	
	// Create response without user_id
	messages := make([]map[string]interface{}, len(conversation.Messages))
	for i, msg := range conversation.Messages {
		messages[i] = map[string]interface{}{
			"id":          msg.ID,
			"role":        msg.Role,
			"content":     msg.Content,
			"timestamp":   msg.CreatedAt.Unix(),
			"token_count": msg.TokenCount,
		}
	}
	
	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"id":            conversation.Conversation.ID,
			"title":         conversation.Conversation.Title,
			"created_at":    conversation.Conversation.CreatedAt.Unix(),
			"updated_at":    conversation.Conversation.UpdatedAt.Unix(),
			"messages":      messages,
			"message_count": len(messages),
			"total_tokens":  conversation.TokenUsage,
			"last_message":  "",
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeleteConversation deletes a conversation
func (h *Handler) DeleteConversation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	if err := h.usageService.DeleteConversation(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

// ListMessages lists messages in a conversation
func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	conversationID := vars["id"]
	
	conversation, err := h.usageService.GetConversationHistory(r.Context(), conversationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conversation.Messages)
}

// ExportConversation exports a conversation as JSON
func (h *Handler) ExportConversation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	data, err := h.usageService.ExportConversation(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="conversation-`+id+`.json"`)
	w.Write(data)
}

// GetUsageStats gets overall usage statistics
func (h *Handler) GetUsageStats(w http.ResponseWriter, r *http.Request) {
	filter := parseUsageFilter(r)
	
	stats, err := h.usageService.GetUsageStats(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetUsageStatsByType gets usage statistics grouped by call type
func (h *Handler) GetUsageStatsByType(w http.ResponseWriter, r *http.Request) {
	filter := parseUsageFilter(r)
	
	stats, err := h.usageService.GetUsageStatsByType(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetUsageStatsByProvider gets usage statistics grouped by provider
func (h *Handler) GetUsageStatsByProvider(w http.ResponseWriter, r *http.Request) {
	filter := parseUsageFilter(r)
	
	stats, err := h.usageService.GetUsageStatsByProvider(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetDailyUsage gets daily usage statistics
func (h *Handler) GetDailyUsage(w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	days := 30
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}
	
	stats, err := h.usageService.GetDailyUsage(r.Context(), days)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetTopModels gets the most used models
func (h *Handler) GetTopModels(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	
	models, err := h.usageService.GetTopModels(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}

// GetCostByProvider gets total cost grouped by provider
func (h *Handler) GetCostByProvider(w http.ResponseWriter, r *http.Request) {
	var startTime, endTime time.Time
	
	if start := r.URL.Query().Get("start_time"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			startTime = t
		}
	}
	
	if end := r.URL.Query().Get("end_time"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			endTime = t
		}
	}
	
	// Default to last 30 days if no time range specified
	if startTime.IsZero() && endTime.IsZero() {
		endTime = time.Now()
		startTime = endTime.AddDate(0, 0, -30)
	}
	
	// Note: This method doesn't exist in the repository yet,
	// but I'm including it for completeness
	costs := make(map[string]float64)
	filter := &usage.UsageFilter{
		StartTime: startTime,
		EndTime:   endTime,
	}
	
	stats, err := h.usageService.GetUsageStatsByProvider(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	for provider, stat := range stats {
		costs[provider] = stat.TotalCost
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(costs)
}

// ListUsageRecords lists usage records
func (h *Handler) ListUsageRecords(w http.ResponseWriter, r *http.Request) {
	_ = parseUsageFilter(r) // TODO: Use filter when service method is implemented
	
	// Note: This requires adding a method to the service
	// For now, returning empty array
	records := []usage.UsageRecord{}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// GetUsageRecord gets a specific usage record
func (h *Handler) GetUsageRecord(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	// Note: This requires adding a method to the service
	// For now, returning not found
	http.Error(w, "Usage record not found", http.StatusNotFound)
	_ = id // Suppress unused variable warning
}

// parseUsageFilter parses usage filter from request query parameters
func parseUsageFilter(r *http.Request) *usage.UsageFilter {
	filter := &usage.UsageFilter{}
	
	if conversationID := r.URL.Query().Get("conversation_id"); conversationID != "" {
		filter.ConversationID = conversationID
	}
	
	if callType := r.URL.Query().Get("call_type"); callType != "" {
		filter.CallType = usage.CallType(callType)
	}
	
	if provider := r.URL.Query().Get("provider"); provider != "" {
		filter.Provider = provider
	}
	
	if model := r.URL.Query().Get("model"); model != "" {
		filter.Model = model
	}
	
	if start := r.URL.Query().Get("start_time"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			filter.StartTime = t
		}
	}
	
	if end := r.URL.Query().Get("end_time"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			filter.EndTime = t
		}
	}
	
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}
	
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}
	
	return filter
}

// RAG Visualization Handlers

// ListRAGQueries lists RAG queries with filtering support
func (h *Handler) ListRAGQueries(w http.ResponseWriter, r *http.Request) {
	filter := parseRAGSearchFilter(r)
	
	queries, err := h.usageService.ListRAGQueries(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(queries)
}

// GetRAGQuery gets a specific RAG query by ID
func (h *Handler) GetRAGQuery(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	queryID := vars["id"]
	
	query, err := h.usageService.GetRAGQuery(r.Context(), queryID)
	if err != nil {
		http.Error(w, "RAG query not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(query)
}

// GetRAGVisualization gets comprehensive visualization data for a RAG query
func (h *Handler) GetRAGVisualization(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	queryID := vars["id"]
	
	visualization, err := h.usageService.GetRAGVisualization(r.Context(), queryID)
	if err != nil {
		http.Error(w, "RAG query visualization not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(visualization)
}

// GetRAGAnalytics gets comprehensive analytics for RAG queries
func (h *Handler) GetRAGAnalytics(w http.ResponseWriter, r *http.Request) {
	filter := parseRAGSearchFilter(r)
	
	analytics, err := h.usageService.GetRAGAnalytics(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}

// GetRAGPerformanceReport gets comprehensive performance analysis for RAG queries
func (h *Handler) GetRAGPerformanceReport(w http.ResponseWriter, r *http.Request) {
	filter := parseRAGSearchFilter(r)
	
	report, err := h.usageService.GetRAGPerformanceReport(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// parseRAGSearchFilter parses RAG search filter from request query parameters
func parseRAGSearchFilter(r *http.Request) *usage.RAGSearchFilter {
	filter := &usage.RAGSearchFilter{}
	
	if conversationID := r.URL.Query().Get("conversation_id"); conversationID != "" {
		filter.ConversationID = conversationID
	}
	
	if query := r.URL.Query().Get("query"); query != "" {
		filter.Query = query
	}
	
	if start := r.URL.Query().Get("start_time"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			filter.StartTime = t
		}
	}
	
	if end := r.URL.Query().Get("end_time"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			filter.EndTime = t
		}
	}
	
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}
	
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}
	
	// Parse min/max score if provided
	if minScoreStr := r.URL.Query().Get("min_score"); minScoreStr != "" {
		if minScore, err := strconv.ParseFloat(minScoreStr, 64); err == nil {
			filter.MinScore = minScore
		}
	}
	
	if maxScoreStr := r.URL.Query().Get("max_score"); maxScoreStr != "" {
		if maxScore, err := strconv.ParseFloat(maxScoreStr, 64); err == nil {
			filter.MaxScore = maxScore
		}
	}
	
	// Parse tools used filter
	if toolsUsed := r.URL.Query().Get("tools_used"); toolsUsed != "" {
		filter.ToolsUsed = []string{toolsUsed}
	}
	
	return filter
}