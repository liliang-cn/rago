package v1

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/internal/api/handlers"
	"github.com/liliang-cn/rago/v2/pkg/usage"
)

// UsageHandler handles usage-related API endpoints
type UsageHandler struct {
	usageService *usage.Service
}

// NewUsageHandler creates a new usage handler
func NewUsageHandler(usageService *usage.Service) *UsageHandler {
	return &UsageHandler{
		usageService: usageService,
	}
}

// GetConversations returns conversation history from real database
func (h *UsageHandler) GetConversations(c *gin.Context) {
	limit := c.DefaultQuery("limit", "50")
	offset := c.DefaultQuery("offset", "0")

	limitInt, _ := strconv.Atoi(limit)
	offsetInt, _ := strconv.Atoi(offset)

	// Get usage service from context or initialize it
	usageService := c.MustGet("usageService").(*usage.Service)
	
	// Get real conversations from database
	realConversations, err := usageService.ListConversations(c.Request.Context(), limitInt, offsetInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": "Failed to fetch conversations: " + err.Error(),
		})
		return
	}

	// Convert to response format
	conversations := []gin.H{}
	for _, conv := range realConversations {
		conversations = append(conversations, gin.H{
			"id":            conv.ID,
			"title":         conv.Title,
			"created_at":    conv.CreatedAt.Unix(),
			"updated_at":    conv.UpdatedAt.Unix(),
			"message_count": 0, // We'll calculate this if needed
			"last_message":  "", // We'll get this if needed
		})
	}

	handlers.SendPagedResponse(c, conversations, offsetInt/limitInt+1, limitInt, len(conversations))
}

// GetConversation returns details for a specific conversation from real database
func (h *UsageHandler) GetConversation(c *gin.Context) {
	conversationID := c.Param("id")
	
	// Get usage service from context
	usageService := c.MustGet("usageService").(*usage.Service)
	
	// Get real conversation history from database
	conversationHistory, err := usageService.GetConversationHistory(c.Request.Context(), conversationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": "Conversation not found: " + err.Error(),
		})
		return
	}

	// Convert messages to response format
	messages := []gin.H{}
	for _, msg := range conversationHistory.Messages {
		messages = append(messages, gin.H{
			"id":          msg.ID,
			"role":        msg.Role,
			"content":     msg.Content,
			"timestamp":   msg.CreatedAt.Unix(),
			"token_count": msg.TokenCount,
		})
	}

	conversation := gin.H{
		"id":            conversationHistory.Conversation.ID,
		"title":         conversationHistory.Conversation.Title,
		"created_at":    conversationHistory.Conversation.CreatedAt.Unix(),
		"updated_at":    conversationHistory.Conversation.UpdatedAt.Unix(),
		"messages":      messages,
		"message_count": len(messages),
		"total_tokens":  conversationHistory.TokenUsage,
		"last_message":  "", // Can get from last message if needed
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": conversation,
	})
}

// GetUsageStats returns usage statistics
func (h *UsageHandler) GetUsageStats(c *gin.Context) {
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	// Create filter from query parameters
	filter := &usage.UsageFilter{}
	
	// Parse time parameters if provided
	if startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filter.StartTime = startTime
		}
	}
	if endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filter.EndTime = endTime
		}
	}

	// Get real usage statistics from database
	stats, err := h.usageService.GetUsageStats(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": "Failed to fetch usage statistics: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": stats,
	})
}

// GetUsageStatsByType returns usage statistics by type
func (h *UsageHandler) GetUsageStatsByType(c *gin.Context) {
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	// Create filter from query parameters
	filter := &usage.UsageFilter{}
	
	// Parse time parameters if provided
	if startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filter.StartTime = startTime
		}
	}
	if endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filter.EndTime = endTime
		}
	}

	// Get real usage statistics by type from database
	statsByType, err := h.usageService.GetUsageStatsByType(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": "Failed to fetch usage statistics by type: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": statsByType,
	})
}

// GetUsageStatsByProvider returns usage statistics by provider
func (h *UsageHandler) GetUsageStatsByProvider(c *gin.Context) {
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	// Create filter from query parameters
	filter := &usage.UsageFilter{}
	
	// Parse time parameters if provided
	if startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filter.StartTime = startTime
		}
	}
	if endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filter.EndTime = endTime
		}
	}

	// Get real usage statistics by provider from database
	statsByProvider, err := h.usageService.GetUsageStatsByProvider(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": "Failed to fetch usage statistics by provider: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": statsByProvider,
	})
}

// GetUsageStatsByModel returns usage statistics for top models
func (h *UsageHandler) GetUsageStatsByModel(c *gin.Context) {
	limit := c.DefaultQuery("limit", "10")

	limitInt, _ := strconv.Atoi(limit)

	// Get real top models data from database
	topModels, err := h.usageService.GetTopModels(c.Request.Context(), limitInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": "Failed to fetch top models: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": topModels,
	})
}

// GetDailyUsageStats returns daily usage statistics
func (h *UsageHandler) GetDailyUsageStats(c *gin.Context) {
	days := c.DefaultQuery("days", "7")

	daysInt, _ := strconv.Atoi(days)

	// Get real daily usage data from database
	dailyUsage, err := h.usageService.GetDailyUsage(c.Request.Context(), daysInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": "Failed to fetch daily usage: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": dailyUsage,
	})
}

// GetUsageCost returns cost breakdown
func (h *UsageHandler) GetUsageCost(c *gin.Context) {
	// Get current month stats
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	currentMonthFilter := &usage.UsageFilter{
		StartTime: startOfMonth,
		EndTime:   now,
	}

	currentMonthByProvider, err := h.usageService.GetUsageStatsByProvider(c.Request.Context(), currentMonthFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": "Failed to fetch current month by provider: " + err.Error(),
		})
		return
	}

	// Format provider costs - return simple cost map for frontend
	costs := gin.H{}
	for provider, stats := range currentMonthByProvider {
		costs[provider] = stats.TotalCost
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": costs,
	})
}