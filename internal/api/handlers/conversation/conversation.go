package conversation

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/liliang-cn/rago/v2/pkg/store"
)

// Handler handles conversation-related API endpoints
type Handler struct {
	store *store.ConversationStore
}

// NewHandler creates a new conversation handler
func NewHandler(convStore *store.ConversationStore) *Handler {
	return &Handler{
		store: convStore,
	}
}

// SaveConversationRequest represents a request to save a conversation
type SaveConversationRequest struct {
	ID       string                        `json:"id,omitempty"`
	Title    string                        `json:"title,omitempty"`
	Messages []store.ConversationMessage   `json:"messages"`
	Metadata map[string]interface{}        `json:"metadata,omitempty"`
}

// ConversationResponse represents a conversation in API responses
type ConversationResponse struct {
	ID        string                        `json:"id"`
	Title     string                        `json:"title"`
	Messages  []store.ConversationMessage   `json:"messages"`
	Metadata  map[string]interface{}        `json:"metadata,omitempty"`
	CreatedAt int64                         `json:"created_at"`
	UpdatedAt int64                         `json:"updated_at"`
}

// ConversationListResponse represents a list of conversations
type ConversationListResponse struct {
	Conversations []ConversationSummary `json:"conversations"`
	Total         int                   `json:"total"`
	Page          int                   `json:"page"`
	PageSize      int                   `json:"page_size"`
}

// ConversationSummary represents a conversation summary in list
type ConversationSummary struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	MessageCount int    `json:"message_count"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
}

// SaveConversation saves or updates a conversation
func (h *Handler) SaveConversation(c *gin.Context) {
	var req SaveConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
		})
		return
	}

	// Create conversation object
	conv := &store.Conversation{
		ID:       req.ID,
		Title:    req.Title,
		Messages: req.Messages,
		Metadata: req.Metadata,
	}

	// Generate ID if not provided
	if conv.ID == "" {
		conv.ID = uuid.New().String()
	}

	// Add timestamps to messages if not present
	now := time.Now().Unix()
	for i := range conv.Messages {
		if conv.Messages[i].Timestamp == 0 {
			conv.Messages[i].Timestamp = now
		}
	}

	// Save to store
	if err := h.store.SaveConversation(conv); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to save conversation",
		})
		return
	}

	// Return saved conversation
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": ConversationResponse{
			ID:        conv.ID,
			Title:     conv.Title,
			Messages:  conv.Messages,
			Metadata:  conv.Metadata,
			CreatedAt: conv.CreatedAt,
			UpdatedAt: conv.UpdatedAt,
		},
	})
}

// GetConversation retrieves a conversation by ID
func (h *Handler) GetConversation(c *gin.Context) {
	id := c.Param("id")
	
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Conversation ID is required",
		})
		return
	}

	conv, err := h.store.GetConversation(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Conversation not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": ConversationResponse{
			ID:        conv.ID,
			Title:     conv.Title,
			Messages:  conv.Messages,
			Metadata:  conv.Metadata,
			CreatedAt: conv.CreatedAt,
			UpdatedAt: conv.UpdatedAt,
		},
	})
}

// ListConversations retrieves a list of conversations
func (h *Handler) ListConversations(c *gin.Context) {
	// Parse query parameters
	page := 1
	pageSize := 20
	
	if p := c.Query("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}
	
	if ps := c.Query("page_size"); ps != "" {
		if val, err := strconv.Atoi(ps); err == nil && val > 0 && val <= 100 {
			pageSize = val
		}
	}

	offset := (page - 1) * pageSize

	// Get conversations from store
	conversations, total, err := h.store.ListConversations(pageSize, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to list conversations",
		})
		return
	}

	// Create response
	summaries := make([]ConversationSummary, len(conversations))
	for i, conv := range conversations {
		summaries[i] = ConversationSummary{
			ID:           conv.ID,
			Title:        conv.Title,
			MessageCount: len(conv.Messages),
			CreatedAt:    conv.CreatedAt,
			UpdatedAt:    conv.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": ConversationListResponse{
			Conversations: summaries,
			Total:         total,
			Page:          page,
			PageSize:      pageSize,
		},
	})
}

// DeleteConversation deletes a conversation by ID
func (h *Handler) DeleteConversation(c *gin.Context) {
	id := c.Param("id")
	
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Conversation ID is required",
		})
		return
	}

	if err := h.store.DeleteConversation(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to delete conversation",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]string{
			"message": "Conversation deleted successfully",
		},
	})
}

// SearchConversations searches conversations
func (h *Handler) SearchConversations(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Search query is required",
		})
		return
	}

	// Parse pagination parameters
	page := 1
	pageSize := 20
	
	if p := c.Query("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}
	
	if ps := c.Query("page_size"); ps != "" {
		if val, err := strconv.Atoi(ps); err == nil && val > 0 && val <= 100 {
			pageSize = val
		}
	}

	offset := (page - 1) * pageSize

	// Search conversations
	conversations, total, err := h.store.SearchConversations(query, pageSize, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to search conversations",
		})
		return
	}

	// Create response
	summaries := make([]ConversationSummary, len(conversations))
	for i, conv := range conversations {
		summaries[i] = ConversationSummary{
			ID:           conv.ID,
			Title:        conv.Title,
			MessageCount: len(conv.Messages),
			CreatedAt:    conv.CreatedAt,
			UpdatedAt:    conv.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": ConversationListResponse{
			Conversations: summaries,
			Total:         total,
			Page:          page,
			PageSize:      pageSize,
		},
	})
}

// CreateNewConversation creates a new conversation with initial ID
func (h *Handler) CreateNewConversation(c *gin.Context) {
	// Generate new conversation ID
	convID := uuid.New().String()
	
	// Return the new conversation ID
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]string{
			"id": convID,
		},
	})
}