package domain

import (
	"context"
	"time"
)

// MemoryType represents different types of long-term memories
type MemoryType string

const (
	MemoryTypeFact      MemoryType = "fact"
	MemoryTypeSkill     MemoryType = "skill"
	MemoryTypePattern   MemoryType = "pattern"
	MemoryTypeContext   MemoryType = "context"
	MemoryTypePreference MemoryType = "preference"
)

// Memory represents a single long-term memory
type Memory struct {
	ID          string                 `json:"id"`
	SessionID   string                 `json:"session_id,omitempty"` // Associated session, empty means global memory
	Type        MemoryType             `json:"type"`
	Content     string                 `json:"content"`
	Vector      []float64              `json:"vector,omitempty"`
	Importance  float64                `json:"importance"`  // 0-1, used for sorting/priority
	AccessCount int                    `json:"access_count"`
	LastAccessed time.Time             `json:"last_accessed"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// MemoryWithScore represents a memory with its similarity score
type MemoryWithScore struct {
	*Memory
	Score float64 `json:"score"`
}

// MemoryRetrieveResult represents the result of a memory retrieval
type MemoryRetrieveResult struct {
	Memories    []*MemoryWithScore `json:"memories"`
	Query       string             `json:"query"`
	Threshold   float64            `json:"threshold"`   // Minimum relevance score
	HasRelevant bool               `json:"has_relevant"` // Whether relevant memories were found
}

// MemoryStoreRequest is a request to store memories after task completion
type MemoryStoreRequest struct {
	SessionID    string                 `json:"session_id"`
	TaskGoal     string                 `json:"task_goal"`
	TaskResult   string                 `json:"task_result"`
	ExecutionLog string                 `json:"execution_log,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// MemorySummaryResult is the LLM-extracted memory summary
type MemorySummaryResult struct {
	ShouldStore bool         `json:"should_store"`
	Memories    []MemoryItem `json:"memories"`
	Reasoning   string       `json:"reasoning"`
}

// MemoryItem represents a single memory item extracted by LLM
type MemoryItem struct {
	Type       MemoryType `json:"type"`
	Content    string     `json:"content"`
	Importance float64    `json:"importance"`
	Tags       []string   `json:"tags,omitempty"`
	Entities   []string   `json:"entities,omitempty"`
}

// MemoryStore defines the interface for memory persistence
type MemoryStore interface {
	// Store saves a new memory
	Store(ctx context.Context, memory *Memory) error

	// Search performs vector search for related memories
	Search(ctx context.Context, vector []float64, topK int, minScore float64) ([]*MemoryWithScore, error)

	// SearchBySession searches memories within a specific session
	SearchBySession(ctx context.Context, sessionID string, vector []float64, topK int) ([]*MemoryWithScore, error)

	// Get retrieves a memory by ID
	Get(ctx context.Context, id string) (*Memory, error)

	// Update updates an existing memory
	Update(ctx context.Context, memory *Memory) error

	// IncrementAccess increments the access count and updates last_accessed
	IncrementAccess(ctx context.Context, id string) error

	// GetByType retrieves memories by type
	GetByType(ctx context.Context, memoryType MemoryType, limit int) ([]*Memory, error)

	// List lists all memories with optional pagination
	List(ctx context.Context, limit, offset int) ([]*Memory, int, error)

	// Delete removes a memory
	Delete(ctx context.Context, id string) error

	// DeleteBySession removes all memories for a session
	DeleteBySession(ctx context.Context, sessionID string) error

	// InitSchema initializes the memory tables
	InitSchema(ctx context.Context) error
}

// MemoryService defines the interface for memory management
type MemoryService interface {
	// RetrieveAndInject searches relevant memories and formats them for LLM context
	RetrieveAndInject(ctx context.Context, query string, sessionID string) (string, []*MemoryWithScore, error)

	// StoreIfWorthwhile analyzes task completion and decides what to store
	StoreIfWorthwhile(ctx context.Context, req *MemoryStoreRequest) error

	// Add directly adds a memory
	Add(ctx context.Context, memory *Memory) error

	// Update updates a memory's content (LLM-driven)
	Update(ctx context.Context, id string, content string) error

	// Search searches memories by query
	Search(ctx context.Context, query string, topK int) ([]*MemoryWithScore, error)

	// Get retrieves a memory by ID
	Get(ctx context.Context, id string) (*Memory, error)

	// List lists memories
	List(ctx context.Context, limit, offset int) ([]*Memory, int, error)

	// Delete removes a memory
	Delete(ctx context.Context, id string) error
}
