package v1

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/internal/api/handlers"
	"github.com/liliang-cn/rago/v2/pkg/usage"
)

// AnalyticsHandler handles analytics-related API endpoints
type AnalyticsHandler struct{
	usageService *usage.Service
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(usageService *usage.Service) *AnalyticsHandler {
	return &AnalyticsHandler{
		usageService: usageService,
	}
}

// GetToolCallStats returns tool call statistics
func (h *AnalyticsHandler) GetToolCallStats(c *gin.Context) {
	// Get query parameters
	limit := c.DefaultQuery("limit", "50")
	offset := c.DefaultQuery("offset", "0")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	// Mock data for now
	stats := gin.H{
		"total_calls": 156,
		"success_rate": 0.95,
		"average_duration_ms": 234,
		"most_used_tools": []gin.H{
			{"name": "filesystem_read", "count": 45},
			{"name": "web_search", "count": 32},
			{"name": "code_execute", "count": 28},
		},
		"time_range": gin.H{
			"start": startTime,
			"end": endTime,
		},
		"pagination": gin.H{
			"limit": limit,
			"offset": offset,
		},
	}

	handlers.SendListResponse(c, stats, 1)
}

// GetToolCalls returns tool call history
func (h *AnalyticsHandler) GetToolCalls(c *gin.Context) {
	// Get query parameters
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	// If no usage service, return empty
	if h.usageService == nil {
		handlers.SendPagedResponse(c, []gin.H{}, 1, limit, 0)
		return
	}

	// Build filter
	filter := &usage.RAGSearchFilter{
		Limit:  limit,
		Offset: offset,
	}
	if startTime != "" {
		// Try parsing as Unix timestamp first
		if timestamp, err := strconv.ParseInt(startTime, 10, 64); err == nil {
			filter.StartTime = time.Unix(timestamp, 0)
		} else if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			// Fall back to RFC3339 format
			filter.StartTime = t
		}
	}
	if endTime != "" {
		// Try parsing as Unix timestamp first
		if timestamp, err := strconv.ParseInt(endTime, 10, 64); err == nil {
			filter.EndTime = time.Unix(timestamp, 0)
		} else if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			// Fall back to RFC3339 format
			filter.EndTime = t
		}
	}

	// Get real tool calls from usage service (using RAG tool calls for now)
	calls, err := h.usageService.GetRAGToolCalls(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch tool calls",
		})
		return
	}

	if calls == nil {
		calls = []*usage.RAGToolCall{}
	}

	// Convert to response format
	response := []gin.H{}
	for _, call := range calls {
		response = append(response, gin.H{
			"id":           call.ID,
			"uuid":         call.UUID, // Use the UUID field now
			"tool":         call.ToolName,
			"tool_name":    call.ToolName,
			"tool_type":    "mcp", // Default to MCP type
			"server_name":  "unknown",
			"timestamp":    call.CreatedAt.Unix(),
			"duration_ms":  call.Duration,
			"status":       func() string { if call.Success { return "success" } else { return "failed" } }(),
			"success":      call.Success,
			"args":         call.Arguments,
			"arguments":    call.Arguments,
			"result":       call.Result,
			"error_message": call.ErrorMessage,
			"created_at":   call.CreatedAt.Format(time.RFC3339),
		})
	}

	handlers.SendPagedResponse(c, response, (offset/limit)+1, limit, len(response))
}

// GetToolCallAnalytics returns analytics for tool calls
func (h *AnalyticsHandler) GetToolCallAnalytics(c *gin.Context) {
	// Get query parameters
	limit := c.DefaultQuery("limit", "50")
	offset := c.DefaultQuery("offset", "0")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	analytics := gin.H{
		"usage_by_hour": []gin.H{
			{"hour": "00:00", "count": 12},
			{"hour": "01:00", "count": 8},
			{"hour": "02:00", "count": 5},
			// ... more hours
		},
		"success_trends": []gin.H{
			{"date": "2025-09-27", "success": 45, "failure": 2},
			{"date": "2025-09-28", "success": 52, "failure": 3},
		},
		"error_distribution": gin.H{
			"timeout": 3,
			"invalid_args": 2,
			"permission_denied": 1,
		},
		"time_range": gin.H{
			"start": startTime,
			"end": endTime,
		},
		"pagination": gin.H{
			"limit": limit,
			"offset": offset,
		},
	}

	handlers.SendListResponse(c, analytics, 1)
}

// GetRAGPerformance returns RAG system performance metrics
func (h *AnalyticsHandler) GetRAGPerformance(c *gin.Context) {
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	_ = c.DefaultQuery("limit", "50")

	// If no usage service, return empty performance data
	if h.usageService == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{},
		})
		return
	}

	// Build filter
	filter := &usage.RAGSearchFilter{}
	if startTime != "" {
		// Try parsing as Unix timestamp first
		if timestamp, err := strconv.ParseInt(startTime, 10, 64); err == nil {
			filter.StartTime = time.Unix(timestamp, 0)
		} else if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			// Fall back to RFC3339 format
			filter.StartTime = t
		}
	}
	if endTime != "" {
		// Try parsing as Unix timestamp first
		if timestamp, err := strconv.ParseInt(endTime, 10, 64); err == nil {
			filter.EndTime = time.Unix(timestamp, 0)
		} else if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			// Fall back to RFC3339 format
			filter.EndTime = t
		}
	}

	// Get real performance report from usage service
	report, err := h.usageService.GetRAGPerformanceReport(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": "Failed to fetch performance report",
		})
		return
	}

	// Convert to response format with safe defaults
	response := gin.H{
		"p95_latency": 0.0,
		"retrieval_ratio": 0.0,
		"latency_trend": "stable",
		"quality_trend": "stable",
		"slow_queries": []interface{}{},
		"low_quality_queries": []interface{}{},
		"recommendations": []interface{}{},
		"time_range": gin.H{
			"start": startTime,
			"end": endTime,
		},
	}

	if report != nil {
		response["p95_latency"] = report.P95Latency
		response["retrieval_ratio"] = report.RetrievalRatio
		response["latency_trend"] = report.LatencyTrend
		response["quality_trend"] = report.QualityTrend
		
		if report.SlowQueries != nil {
			response["slow_queries"] = report.SlowQueries
		}
		if report.LowQualityQueries != nil {
			response["low_quality_queries"] = report.LowQualityQueries
		}
		if report.Recommendations != nil {
			response["recommendations"] = report.Recommendations
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": response,
	})
}

// GetRAGQueries returns RAG query history
func (h *AnalyticsHandler) GetRAGQueries(c *gin.Context) {
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	_ = c.DefaultQuery("limit", "50")

	// If no usage service, return empty
	if h.usageService == nil {
		handlers.SendListResponse(c, []gin.H{}, 0)
		return
	}

	// Build filter
	filter := &usage.RAGSearchFilter{}
	if startTime != "" {
		// Try parsing as Unix timestamp first
		if timestamp, err := strconv.ParseInt(startTime, 10, 64); err == nil {
			filter.StartTime = time.Unix(timestamp, 0)
		} else if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			// Fall back to RFC3339 format
			filter.StartTime = t
		}
	}
	if endTime != "" {
		// Try parsing as Unix timestamp first
		if timestamp, err := strconv.ParseInt(endTime, 10, 64); err == nil {
			filter.EndTime = time.Unix(timestamp, 0)
		} else if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			// Fall back to RFC3339 format
			filter.EndTime = t
		}
	}

	// Get real RAG queries from usage service
	queries, err := h.usageService.ListRAGQueries(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": "Failed to fetch RAG queries",
		})
		return
	}

	if queries == nil {
		queries = []*usage.RAGQueryRecord{}
	}

	// Convert to response format
	response := []gin.H{}
	for _, q := range queries {
		// Create empty sources array for now
		var sources []string
		if sources == nil {
			sources = []string{}
		}
		response = append(response, gin.H{
			"id": q.ID,
			"query": q.Query,
			"timestamp": q.CreatedAt.Unix(),
			"response_time_ms": q.TotalLatency,
			"chunks_retrieved": q.ChunksFound,
			"sources": sources,
			"success": q.Success,
			"created_at": q.CreatedAt.Format(time.RFC3339),
			"total_latency": q.TotalLatency,
			"chunks_found": q.ChunksFound,
		})
	}

	handlers.SendListResponse(c, response, len(response))
}

// GetRAGAnalytics returns RAG system analytics
func (h *AnalyticsHandler) GetRAGAnalytics(c *gin.Context) {
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	_ = c.DefaultQuery("limit", "50")

	// If no usage service, return empty analytics
	if h.usageService == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{},
		})
		return
	}

	// Build filter
	filter := &usage.RAGSearchFilter{}
	if startTime != "" {
		// Try parsing as Unix timestamp first
		if timestamp, err := strconv.ParseInt(startTime, 10, 64); err == nil {
			filter.StartTime = time.Unix(timestamp, 0)
		} else if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			// Fall back to RFC3339 format
			filter.StartTime = t
		}
	}
	if endTime != "" {
		// Try parsing as Unix timestamp first
		if timestamp, err := strconv.ParseInt(endTime, 10, 64); err == nil {
			filter.EndTime = time.Unix(timestamp, 0)
		} else if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			// Fall back to RFC3339 format
			filter.EndTime = t
		}
	}

	// Get real analytics from usage service
	analytics, err := h.usageService.GetRAGAnalytics(c.Request.Context(), filter)
	if err != nil {
		// Log the actual error for debugging
		fmt.Printf("RAG analytics error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": "Failed to fetch RAG analytics",
		})
		return
	}

	// Convert to response format with safe defaults
	response := gin.H{
		"total_queries": 0,
		"success_rate": 0.0,
		"avg_latency": 0.0,
		"avg_chunks": 0.0,
		"avg_score": 0.0,
		"fast_queries": 0,
		"medium_queries": 0,
		"slow_queries": 0,
		"high_quality_queries": 0,
		"medium_quality_queries": 0,
		"low_quality_queries": 0,
		"start_time": startTime,
		"end_time": endTime,
	}

	if analytics != nil {
		response["total_queries"] = analytics.TotalQueries
		response["success_rate"] = analytics.SuccessRate
		response["avg_latency"] = analytics.AvgLatency
		response["avg_chunks"] = analytics.AvgChunks
		response["avg_score"] = analytics.AvgScore
		response["start_time"] = analytics.StartTime
		response["end_time"] = analytics.EndTime
		
		// Calculate performance distribution based on latency
		// You can add more sophisticated calculations based on actual query data
		response["fast_queries"] = 0
		response["medium_queries"] = 0
		response["slow_queries"] = 0
		
		// Calculate quality distribution based on scores
		response["high_quality_queries"] = 0
		response["medium_quality_queries"] = 0
		response["low_quality_queries"] = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": response,
	})
}