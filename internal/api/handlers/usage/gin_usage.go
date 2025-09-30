package usage

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/pkg/usage"
)

// GinHandler handles usage-related HTTP requests for Gin
type GinHandler struct {
	usageService *usage.Service
}

// NewGinHandler creates a new usage handler for Gin
func NewGinHandler(usageService *usage.Service) *GinHandler {
	return &GinHandler{
		usageService: usageService,
	}
}

// ListConversations lists all conversations
func (h *GinHandler) ListConversations(c *gin.Context) {
	limitStr := c.Query("limit")
	offsetStr := c.Query("offset")
	
	limit := 50
	offset := 0
	
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}
	
	conversations, err := h.usageService.ListConversations(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list conversations"})
		return
	}
	
	c.JSON(http.StatusOK, conversations)
}

// CreateConversation creates a new conversation
func (h *GinHandler) CreateConversation(c *gin.Context) {
	var req struct {
		Title string `json:"title"`
	}
	
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	
	conversation, err := h.usageService.StartConversation(c.Request.Context(), req.Title)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create conversation"})
		return
	}
	
	c.JSON(http.StatusCreated, conversation)
}

// GetConversation gets a specific conversation with its messages
func (h *GinHandler) GetConversation(c *gin.Context) {
	conversationID := c.Param("id")
	
	history, err := h.usageService.GetConversationHistory(c.Request.Context(), conversationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}
	
	c.JSON(http.StatusOK, history)
}

// DeleteConversation deletes a conversation
func (h *GinHandler) DeleteConversation(c *gin.Context) {
	conversationID := c.Param("id")
	
	err := h.usageService.DeleteConversation(c.Request.Context(), conversationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete conversation"})
		return
	}
	
	c.JSON(http.StatusNoContent, nil)
}

// ExportConversation exports a conversation to JSON
func (h *GinHandler) ExportConversation(c *gin.Context) {
	conversationID := c.Param("id")
	
	data, err := h.usageService.ExportConversation(c.Request.Context(), conversationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}
	
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=\"conversation-"+conversationID+".json\"")
	c.Data(http.StatusOK, "application/json", data)
}

// AddMessage adds a message to the current conversation
func (h *GinHandler) AddMessage(c *gin.Context) {
	var req struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	
	message, err := h.usageService.AddMessage(c.Request.Context(), req.Role, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add message"})
		return
	}
	
	c.JSON(http.StatusCreated, message)
}

// GetUsageStats gets usage statistics
func (h *GinHandler) GetUsageStats(c *gin.Context) {
	filter := parseUsageFilterFromGin(c)
	
	stats, err := h.usageService.GetUsageStats(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get usage stats"})
		return
	}
	
	c.JSON(http.StatusOK, stats)
}

// GetUsageStatsByType gets usage statistics by type
func (h *GinHandler) GetUsageStatsByType(c *gin.Context) {
	filter := parseUsageFilterFromGin(c)
	
	stats, err := h.usageService.GetUsageStatsByType(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get usage stats by type"})
		return
	}
	
	c.JSON(http.StatusOK, stats)
}

// GetUsageStatsByProvider gets usage statistics by provider
func (h *GinHandler) GetUsageStatsByProvider(c *gin.Context) {
	filter := parseUsageFilterFromGin(c)
	
	stats, err := h.usageService.GetUsageStatsByProvider(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get usage stats by provider"})
		return
	}
	
	c.JSON(http.StatusOK, stats)
}

// GetDailyUsage gets daily usage statistics
func (h *GinHandler) GetDailyUsage(c *gin.Context) {
	daysStr := c.Query("days")
	days := 7
	
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}
	
	usage, err := h.usageService.GetDailyUsage(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get daily usage"})
		return
	}
	
	c.JSON(http.StatusOK, usage)
}

// GetTopModels gets the most used models
func (h *GinHandler) GetTopModels(c *gin.Context) {
	limitStr := c.Query("limit")
	limit := 10
	
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	
	models, err := h.usageService.GetTopModels(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get top models"})
		return
	}
	
	c.JSON(http.StatusOK, models)
}

// ListUsageRecords lists usage records with filtering
func (h *GinHandler) ListUsageRecords(c *gin.Context) {
	filter := parseUsageFilterFromGin(c)
	
	// Note: This method doesn't exist in the service yet
	// You may need to add it or use GetUsageStats instead
	stats, err := h.usageService.GetUsageStats(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list usage records"})
		return
	}
	
	c.JSON(http.StatusOK, stats)
}

// GetUsageRecord gets a specific usage record
func (h *GinHandler) GetUsageRecord(c *gin.Context) {
	recordID := c.Param("id")
	
	// Note: This method doesn't exist in the service yet
	// You may need to implement it
	c.JSON(http.StatusNotImplemented, gin.H{"error": "GetUsageRecord not implemented", "id": recordID})
}

// ListRAGQueries lists RAG queries with filtering
func (h *GinHandler) ListRAGQueries(c *gin.Context) {
	filter := parseRAGSearchFilterFromGin(c)
	
	queries, err := h.usageService.ListRAGQueries(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list RAG queries"})
		return
	}
	
	c.JSON(http.StatusOK, queries)
}

// GetRAGQuery gets a specific RAG query
func (h *GinHandler) GetRAGQuery(c *gin.Context) {
	queryID := c.Param("id")
	
	query, err := h.usageService.GetRAGQuery(c.Request.Context(), queryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "RAG query not found"})
		return
	}
	
	c.JSON(http.StatusOK, query)
}

// GetRAGVisualization gets comprehensive visualization data for a RAG query
func (h *GinHandler) GetRAGVisualization(c *gin.Context) {
	queryID := c.Param("id")
	
	visualization, err := h.usageService.GetRAGVisualization(c.Request.Context(), queryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "RAG query visualization not found"})
		return
	}
	
	c.JSON(http.StatusOK, visualization)
}

// GetRAGAnalytics gets comprehensive analytics for RAG queries
func (h *GinHandler) GetRAGAnalytics(c *gin.Context) {
	filter := parseRAGSearchFilterFromGin(c)
	
	analytics, err := h.usageService.GetRAGAnalytics(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get RAG analytics"})
		return
	}
	
	c.JSON(http.StatusOK, analytics)
}

// GetRAGPerformanceReport gets comprehensive performance report for RAG queries
func (h *GinHandler) GetRAGPerformanceReport(c *gin.Context) {
	filter := parseRAGSearchFilterFromGin(c)
	
	report, err := h.usageService.GetRAGPerformanceReport(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get RAG performance report"})
		return
	}
	
	c.JSON(http.StatusOK, report)
}

// GetToolCallVisualization gets visualization data for a specific tool call
func (h *GinHandler) GetToolCallVisualization(c *gin.Context) {
	toolCallID := c.Param("id")
	
	// For now, we'll return the tool call details from the RAG tool calls
	// In a real implementation, you would have a dedicated method for this
	filter := &usage.RAGSearchFilter{}
	toolCalls, err := h.usageService.GetRAGToolCalls(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tool call not found"})
		return
	}
	
	// Find the specific tool call
	var foundCall *usage.RAGToolCall
	for _, call := range toolCalls {
		if call.ID == toolCallID {
			foundCall = call
			break
		}
	}
	
	if foundCall == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tool call not found"})
		return
	}
	
	// Create a visualization response with the tool call and related metrics
	visualization := gin.H{
		"tool_call": foundCall,
		"metrics": gin.H{
			"duration_ms": foundCall.Duration,
			"success": foundCall.Success,
			"tool_name": foundCall.ToolName,
		},
		"context": gin.H{
			"rag_query_id": foundCall.RAGQueryID,
			"created_at": foundCall.CreatedAt,
		},
	}
	
	c.JSON(http.StatusOK, visualization)
}

// Helper functions

func parseUsageFilterFromGin(c *gin.Context) *usage.UsageFilter {
	filter := &usage.UsageFilter{}
	
	if conversationID := c.Query("conversation_id"); conversationID != "" {
		filter.ConversationID = conversationID
	}
	if provider := c.Query("provider"); provider != "" {
		filter.Provider = provider
	}
	if model := c.Query("model"); model != "" {
		filter.Model = model
	}
	if callType := c.Query("call_type"); callType != "" {
		filter.CallType = usage.CallType(callType)
	}
	
	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filter.StartTime = t
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filter.EndTime = t
		}
	}
	
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}
	
	return filter
}

func parseRAGSearchFilterFromGin(c *gin.Context) *usage.RAGSearchFilter {
	filter := &usage.RAGSearchFilter{}
	
	if conversationID := c.Query("conversation_id"); conversationID != "" {
		filter.ConversationID = conversationID
	}
	if query := c.Query("query"); query != "" {
		filter.Query = query
	}
	
	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filter.StartTime = t
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filter.EndTime = t
		}
	}
	
	// These fields may not exist in RAGSearchFilter yet
	// Keeping them commented for future implementation
	// if onlySuccessful := c.Query("only_successful"); onlySuccessful == "true" {
	// 	filter.OnlySuccessful = true
	// }
	// if withToolCalls := c.Query("with_tool_calls"); withToolCalls == "true" {
	// 	filter.WithToolCalls = true
	// }
	
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}
	
	// OrderBy field may not exist in RAGSearchFilter yet
	// Keeping it commented for future implementation
	// if orderBy := c.Query("order_by"); orderBy != "" {
	// 	filter.OrderBy = orderBy
	// }
	
	return filter
}