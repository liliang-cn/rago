package usage

import (
	"context"
	"time"
)

// Repository defines the interface for usage data persistence
type Repository interface {
	// Conversation operations
	CreateConversation(ctx context.Context, conversation *Conversation) error
	GetConversation(ctx context.Context, id string) (*Conversation, error)
	ListConversations(ctx context.Context, limit, offset int) ([]*Conversation, error)
	UpdateConversation(ctx context.Context, id string, title string) error
	DeleteConversation(ctx context.Context, id string) error

	// Message operations
	CreateMessage(ctx context.Context, message *Message) error
	GetMessage(ctx context.Context, id string) (*Message, error)
	ListMessages(ctx context.Context, conversationID string) ([]*Message, error)
	DeleteMessage(ctx context.Context, id string) error

	// Usage record operations
	CreateUsageRecord(ctx context.Context, record *UsageRecord) error
	GetUsageRecord(ctx context.Context, id string) (*UsageRecord, error)
	ListUsageRecords(ctx context.Context, filter *UsageFilter) ([]*UsageRecord, error)
	
	// Statistics operations
	GetUsageStats(ctx context.Context, filter *UsageFilter) (*UsageStats, error)
	GetUsageStatsByType(ctx context.Context, filter *UsageFilter) (UsageStatsByType, error)
	GetUsageStatsByProvider(ctx context.Context, filter *UsageFilter) (UsageStatsByProvider, error)
	GetConversationStats(ctx context.Context, conversationID string) (*UsageStats, error)
	
	// Aggregation operations
	GetDailyUsage(ctx context.Context, days int) (map[string]*UsageStats, error)
	GetTopModels(ctx context.Context, limit int) (map[string]int64, error)
	GetCostByProvider(ctx context.Context, startTime, endTime time.Time) (map[string]float64, error)
	
	// RAG operations
	CreateRAGQuery(ctx context.Context, query *RAGQueryRecord) error
	GetRAGQuery(ctx context.Context, id string) (*RAGQueryRecord, error)
	UpdateRAGQuery(ctx context.Context, query *RAGQueryRecord) error
	ListRAGQueries(ctx context.Context, filter *RAGSearchFilter) ([]*RAGQueryRecord, error)
	CreateChunkHit(ctx context.Context, hit *RAGChunkHit) error
	ListChunkHits(ctx context.Context, ragQueryID string) ([]*RAGChunkHit, error)
	CreateToolCall(ctx context.Context, toolCall *RAGToolCall) error
	ListToolCalls(ctx context.Context, ragQueryID string) ([]*RAGToolCall, error)
	ListAllToolCalls(ctx context.Context, filter *RAGSearchFilter) ([]*RAGToolCall, error)
	GetRAGVisualization(ctx context.Context, ragQueryID string) (*RAGQueryVisualization, error)
	ListRAGVisualizations(ctx context.Context, filter *RAGSearchFilter) ([]*RAGQueryVisualization, error)
	GetRAGAnalytics(ctx context.Context, filter *RAGSearchFilter) (*RAGAnalytics, error)
	
	// Database management
	Initialize(ctx context.Context) error
	Close() error
}