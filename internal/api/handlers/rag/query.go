package rag

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/rag/processor"
	"github.com/liliang-cn/rago/v2/pkg/usage"
)

type QueryHandler struct {
	processor    *processor.Service
	usageService *usage.Service
}

func NewQueryHandler(p *processor.Service) *QueryHandler {
	return &QueryHandler{processor: p}
}

func NewQueryHandlerWithUsage(p *processor.Service, u *usage.Service) *QueryHandler {
	return &QueryHandler{
		processor:    p,
		usageService: u,
	}
}

func (h *QueryHandler) Handle(c *gin.Context) {
	var req domain.QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate query before processing
	if req.Query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid input: empty query",
		})
		return
	}

	if req.Stream {
		h.handleStream(c, req)
		return
	}

	// Use QueryWithTools if tools are enabled and requested
	var resp domain.QueryResponse
	var err error
	if req.ToolsEnabled {
		resp, err = h.processor.QueryWithTools(c.Request.Context(), req)
	} else {
		resp, err = h.processor.Query(c.Request.Context(), req)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process query: " + err.Error(),
		})
		return
	}

	// 确保sources总是数组
	if resp.Sources == nil {
		resp.Sources = []domain.Chunk{}
	}

	c.JSON(http.StatusOK, resp)
}

func (h *QueryHandler) SearchOnly(c *gin.Context) {
	var req struct {
		Query   string                 `json:"query"`
		TopK    int                    `json:"top_k"`
		Filters map[string]interface{} `json:"filters,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate query before processing
	if req.Query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid input: empty query",
		})
		return
	}

	if req.TopK <= 0 {
		req.TopK = 5
	}

	ctx := c.Request.Context()

	// Create a simple search request using the processor
	queryReq := domain.QueryRequest{
		Query:   req.Query,
		TopK:    req.TopK,
		Filters: req.Filters,
	}

	// Use processor's search functionality - we'll need to add a search-only method
	resp, err := h.processor.Query(ctx, queryReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to search: " + err.Error(),
		})
		return
	}

	// 确保总是返回数组，即使为空
	sources := resp.Sources
	if sources == nil {
		sources = []domain.Chunk{}
	}

	c.JSON(http.StatusOK, sources)
}

func (h *QueryHandler) HandleStream(c *gin.Context) {
	var req domain.QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate query before processing
	if req.Query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid input: empty query",
		})
		return
	}

	// Force streaming mode
	req.Stream = true
	h.handleStream(c, req)
}

func (h *QueryHandler) handleStream(c *gin.Context, req domain.QueryRequest) {
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Streaming not supported",
		})
		return
	}

	// Handle conversation tracking if usage service is available
	var conversationID string
	
	// Extract conversation ID from request if available
	if req.ConversationID != "" {
		conversationID = req.ConversationID
	}

	// Store the user message if usage service is available
	if h.usageService != nil {
		// If no conversation ID provided, start a new conversation
		if conversationID == "" {
			conversation, err := h.usageService.StartConversation(ctx, "Chat about documents")
			if err != nil {
				log.Printf("Warning: failed to start conversation: %v", err)
			} else {
				conversationID = conversation.ID
			}
		} else {
			// Set the current conversation
			if err := h.usageService.SetCurrentConversation(ctx, conversationID); err != nil {
				log.Printf("Warning: failed to set current conversation: %v", err)
			}
		}

		// Store the user's message
		if conversationID != "" {
			_, err := h.usageService.AddMessage(ctx, "user", req.Query)
			if err != nil {
				log.Printf("Warning: failed to store user message: %v", err)
			}
		}
	}

	// Collect the response for storage
	var responseContent strings.Builder

	// Use StreamQueryWithTools if tools are enabled and requested
	var err error
	if req.ToolsEnabled {
		err = h.processor.StreamQueryWithTools(ctx, req, func(token string) {
			// Write token to client
			if _, writeErr := fmt.Fprint(c.Writer, token); writeErr != nil {
				log.Printf("Error writing token: %v", writeErr)
			}
			flusher.Flush()
			
			// Collect token for storage
			responseContent.WriteString(token)
		})
	} else {
		err = h.processor.StreamQuery(ctx, req, func(token string) {
			// Write token to client
			if _, writeErr := fmt.Fprint(c.Writer, token); writeErr != nil {
				log.Printf("Error writing token: %v", writeErr)
			}
			flusher.Flush()
			
			// Collect token for storage
			responseContent.WriteString(token)
		})
	}

	if err != nil {
		if _, writeErr := fmt.Fprintf(c.Writer, "\n\nError: %v", err); writeErr != nil {
			log.Printf("Error writing error message: %v", writeErr)
		}
		flusher.Flush()
		
		// Add error to response content
		responseContent.WriteString(fmt.Sprintf("\n\nError: %v", err))
	}

	// Store the assistant's response if usage service is available
	if h.usageService != nil && conversationID != "" && responseContent.Len() > 0 {
		_, err := h.usageService.AddMessage(ctx, "assistant", responseContent.String())
		if err != nil {
			log.Printf("Warning: failed to store assistant message: %v", err)
		}
	}
}
