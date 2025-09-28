package usage

import (
	"time"

	"github.com/google/uuid"
)

// CallType represents the type of API call
type CallType string

const (
	CallTypeLLM CallType = "llm"
	CallTypeMCP CallType = "mcp"
	CallTypeRAG CallType = "rag"
)

// Conversation represents a conversation session
type Conversation struct {
	ID        string    `json:"id" db:"id"`
	Title     string    `json:"title" db:"title"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	Metadata  string    `json:"metadata" db:"metadata"` // JSON string for additional data
}

// Message represents a single message in a conversation
type Message struct {
	ID             string    `json:"id" db:"id"`
	ConversationID string    `json:"conversation_id" db:"conversation_id"`
	Role           string    `json:"role" db:"role"` // user, assistant, system
	Content        string    `json:"content" db:"content"`
	TokenCount     int       `json:"token_count" db:"token_count"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	Metadata       string    `json:"metadata" db:"metadata"` // JSON string for additional data
}

// UsageRecord represents token usage for a single API call
type UsageRecord struct {
	ID               string    `json:"id" db:"id"`
	ConversationID   string    `json:"conversation_id" db:"conversation_id"`
	MessageID        string    `json:"message_id" db:"message_id"`
	CallType         CallType  `json:"call_type" db:"call_type"`
	Provider         string    `json:"provider" db:"provider"`
	Model            string    `json:"model" db:"model"`
	InputTokens      int       `json:"input_tokens" db:"input_tokens"`
	OutputTokens     int       `json:"output_tokens" db:"output_tokens"`
	TotalTokens      int       `json:"total_tokens" db:"total_tokens"`
	Cost             float64   `json:"cost" db:"cost"`
	Latency          int64     `json:"latency" db:"latency"` // milliseconds
	Success          bool      `json:"success" db:"success"`
	ErrorMessage     string    `json:"error_message" db:"error_message"`
	RequestMetadata  string    `json:"request_metadata" db:"request_metadata"`   // JSON string
	ResponseMetadata string    `json:"response_metadata" db:"response_metadata"` // JSON string
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// UsageStats represents aggregated usage statistics
type UsageStats struct {
	TotalCalls       int64   `json:"total_calls"`
	TotalInputTokens int64   `json:"total_input_tokens"`
	TotalOutputTokens int64   `json:"total_output_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	TotalCost        float64 `json:"total_cost"`
	AverageLatency   float64 `json:"average_latency"`
	SuccessRate      float64 `json:"success_rate"`
}

// UsageStatsByType represents usage statistics grouped by call type
type UsageStatsByType map[CallType]*UsageStats

// UsageStatsByProvider represents usage statistics grouped by provider
type UsageStatsByProvider map[string]*UsageStats

// ConversationWithMessages represents a conversation with its messages
type ConversationWithMessages struct {
	Conversation Conversation `json:"conversation"`
	Messages     []Message    `json:"messages"`
	TokenUsage   int          `json:"token_usage"`
}

// UsageFilter represents filters for querying usage records
type UsageFilter struct {
	ConversationID string    `json:"conversation_id,omitempty"`
	CallType       CallType  `json:"call_type,omitempty"`
	Provider       string    `json:"provider,omitempty"`
	Model          string    `json:"model,omitempty"`
	StartTime      time.Time `json:"start_time,omitempty"`
	EndTime        time.Time `json:"end_time,omitempty"`
	Limit          int       `json:"limit,omitempty"`
	Offset         int       `json:"offset,omitempty"`
}

// NewConversation creates a new conversation with a UUID
func NewConversation(title string) *Conversation {
	now := time.Now()
	return &Conversation{
		ID:        uuid.New().String(),
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewMessage creates a new message with a UUID
func NewMessage(conversationID, role, content string, tokenCount int) *Message {
	return &Message{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		Role:           role,
		Content:        content,
		TokenCount:     tokenCount,
		CreatedAt:      time.Now(),
	}
}

// NewUsageRecord creates a new usage record with a UUID
func NewUsageRecord(conversationID, messageID string, callType CallType) *UsageRecord {
	return &UsageRecord{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		MessageID:      messageID,
		CallType:       callType,
		CreatedAt:      time.Now(),
	}
}