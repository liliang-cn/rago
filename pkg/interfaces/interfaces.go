package interfaces

import (
	"context"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// RAGService defines the core RAG functionality interface
type RAGService interface {
	// Core RAG operations
	Ingest(ctx context.Context, req domain.IngestRequest) (domain.IngestResponse, error)
	Query(ctx context.Context, req domain.QueryRequest) (domain.QueryResponse, error)
	StreamQuery(ctx context.Context, req domain.QueryRequest, callback func(string)) error

	// Document management
	ListDocuments(ctx context.Context) ([]domain.Document, error)
	DeleteDocument(ctx context.Context, documentID string) error
	Reset(ctx context.Context) error

	// Optional tool-enhanced operations
	QueryWithTools(ctx context.Context, req domain.QueryRequest) (domain.QueryResponse, error)
	StreamQueryWithTools(ctx context.Context, req domain.QueryRequest, callback func(string)) error
}

// MCPService defines MCP server interaction interface (optional component)
type MCPService interface {
	Initialize(ctx context.Context) error
	Close() error
	IsHealthy() bool
	GetAvailableTools() map[string]interface{}
	CallTool(ctx context.Context, name string, args map[string]interface{}) (map[string]interface{}, error)
}

// AgentService defines agent workflow execution interface (optional component)
type AgentService interface {
	// Agent lifecycle
	Initialize(ctx context.Context) error
	Close() error

	// Workflow management
	CreateWorkflow(name string, definition map[string]interface{}) error
	ExecuteWorkflow(ctx context.Context, workflowName string, input map[string]interface{}) (*WorkflowResult, error)
	ListWorkflows() ([]WorkflowInfo, error)
	DeleteWorkflow(name string) error

	// Agent management
	CreateAgent(name string, config AgentConfig) error
	ExecuteAgent(ctx context.Context, agentName string, task string) (*AgentResult, error)
	ListAgents() ([]AgentInfo, error)
}

// SchedulerService defines task scheduling interface (optional component)
type SchedulerService interface {
	// Scheduler lifecycle
	Start() error
	Stop() error
	IsRunning() bool

	// Job management
	CreateJob(job *Job) (string, error)
	UpdateJob(job *Job) error
	DeleteJob(id string) error
	GetJob(id string) (*Job, error)
	ListJobs() ([]*Job, error)

	// Job execution
	RunJobNow(jobID string) error
	EnableJob(id string) error
	DisableJob(id string) error

	// Execution history
	GetJobExecutions(jobID string, limit int) ([]JobExecution, error)
}

// LLMProvider defines the interface for language model providers
type LLMProvider interface {
	// Core operations
	Generate(ctx context.Context, prompt string, options *domain.GenerationOptions) (string, error)
	Stream(ctx context.Context, prompt string, options *domain.GenerationOptions, callback func(string)) error

	// Tool-enhanced operations (optional) - simplified return types
	GenerateWithTools(ctx context.Context, prompt string, tools []domain.ToolDefinition, options *domain.GenerationOptions) (map[string]interface{}, error)
	StreamWithTools(ctx context.Context, prompt string, tools []domain.ToolDefinition, options *domain.GenerationOptions, callback func(string)) error

	// Provider info
	Name() string
	IsAvailable(ctx context.Context) bool
}

// EmbeddingProvider defines the interface for embedding providers
type EmbeddingProvider interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	BatchEmbed(ctx context.Context, texts []string) ([][]float32, error)
	Dimension() int
	Name() string
	IsAvailable(ctx context.Context) bool
}

// Component registry for dependency injection
type ComponentRegistry interface {
	// Core components (always required)
	RegisterRAG(service RAGService)
	GetRAG() RAGService

	RegisterLLMProvider(name string, provider LLMProvider)
	GetLLMProvider(name string) LLMProvider
	GetDefaultLLMProvider() LLMProvider

	RegisterEmbeddingProvider(name string, provider EmbeddingProvider)
	GetEmbeddingProvider(name string) EmbeddingProvider
	GetDefaultEmbeddingProvider() EmbeddingProvider

	// Optional components
	RegisterMCP(service MCPService)
	GetMCP() MCPService
	HasMCP() bool

	RegisterAgents(service AgentService)
	GetAgents() AgentService
	HasAgents() bool

	RegisterScheduler(service SchedulerService)
	GetScheduler() SchedulerService
	HasScheduler() bool
}

// WorkflowResult represents the result of a workflow execution
type WorkflowResult struct {
	Success   bool                   `json:"success"`
	Output    map[string]interface{} `json:"output"`
	Error     string                 `json:"error,omitempty"`
	Duration  time.Duration          `json:"duration"`
	StepCount int                    `json:"step_count"`
}

// WorkflowInfo represents workflow metadata
type WorkflowInfo struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	StepCount   int       `json:"step_count"`
}

// AgentConfig defines agent configuration
type AgentConfig struct {
	Type        string                 `json:"type"`
	Model       string                 `json:"model"`
	Temperature float64                `json:"temperature"`
	MaxTokens   int                    `json:"max_tokens"`
	Tools       []string               `json:"tools"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// AgentResult represents the result of an agent execution
type AgentResult struct {
	Success  bool          `json:"success"`
	Response string        `json:"response"`
	Actions  []AgentAction `json:"actions,omitempty"`
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration"`
}

// AgentAction represents an action taken by an agent
type AgentAction struct {
	Type      string                 `json:"type"`
	Tool      string                 `json:"tool,omitempty"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	Result    interface{}            `json:"result,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// AgentInfo represents agent metadata
type AgentInfo struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	LastRun     time.Time `json:"last_run,omitempty"`
}

// Job represents a scheduled task
type Job struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Schedule  string                 `json:"schedule"` // Cron expression
	Enabled   bool                   `json:"enabled"`
	Payload   map[string]interface{} `json:"payload"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	LastRun   *time.Time             `json:"last_run,omitempty"`
	NextRun   *time.Time             `json:"next_run,omitempty"`
}

// JobExecution represents a single job execution
type JobExecution struct {
	ID        string    `json:"id"`
	JobID     string    `json:"job_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	Output    string    `json:"output,omitempty"`
}
