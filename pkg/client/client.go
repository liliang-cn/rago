// Package client implements the unified RAGO client providing access to all four pillars:
// LLM, RAG, MCP, and Agents. This package serves as the primary interface for library users.
package client

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agents"
	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/llm"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/rag"
	"github.com/liliang-cn/rago/v2/pkg/rag/ingest"
	"github.com/liliang-cn/rago/v2/pkg/rag/storage"
)

// Client implements the core.Client interface and provides unified access to all RAGO pillars.
// It serves as the primary entry point for applications using RAGO as their AI foundation.
type Client struct {
	// Configuration
	config *Config
	
	// Pillar services - each pillar is independently accessible
	llmService   core.LLMService
	ragService   core.RAGService  
	mcpService   core.MCPService
	agentService core.AgentService
	
	// Health monitoring
	healthMonitor *HealthMonitor
	
	// Lifecycle management
	mu     sync.RWMutex
	closed bool
	ctx    context.Context
	cancel context.CancelFunc
}

// New creates a new RAGO client with the specified config file path.
// This is the primary constructor for the unified client supporting all four pillars.
func New(configPath string) (*Client, error) {
	// Load V3 configuration
	cfg, err := Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	
	return NewWithConfig(cfg)
}

// NewWithDefaults creates a new RAGO client with default configuration.
// This provides a quick start option for users who want to get started immediately.
func NewWithDefaults() (*Client, error) {
	cfg := getDefaultConfig()
	return NewWithConfig(cfg)
}

// NewWithConfig creates a new RAGO client with the provided configuration.
// This function initializes all four pillars based on the configuration.
func NewWithConfig(cfg *Config) (*Client, error) {
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	client := &Client{
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}
	
	// Initialize pillars based on configuration
	if err := client.initializePillars(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize pillars: %w", err)
	}
	
	// Initialize health monitoring
	client.healthMonitor = NewHealthMonitor(client)
	
	return client, nil
}

// initializePillars initializes all pillar services based on configuration
func (c *Client) initializePillars() error {
	var enabledPillars []string
	
	// Initialize LLM pillar if enabled
	if !c.shouldDisableLLM() {
		llmSvc, err := llm.NewService(c.config.LLM)
		if err != nil {
			return fmt.Errorf("failed to initialize LLM service: %w", err)
		}
		c.llmService = llmSvc
		enabledPillars = append(enabledPillars, "LLM")
	}
	
	// Initialize RAG pillar if enabled  
	if !c.shouldDisableRAG() {
		// We need to create an embedder from the LLM service
		if c.llmService == nil {
			return fmt.Errorf("RAG pillar requires LLM pillar for embeddings - initialize LLM first")
		}
		
		// Create a complete RAG config using the configuration and data directory
		// Use the configured data_dir to ensure all storage backends use consistent paths
		dataDir := c.config.DataDir
		if dataDir == "" {
			dataDir = ".rago/data" // fallback default
		}
		
		ragConfig := &rag.Config{
			StorageBackend: c.config.RAG.StorageBackend,
			VectorStore: storage.VectorConfig{
				Backend:    "sqvect",
				DBPath:     filepath.Join(dataDir, "vectors.db"),
				Dimensions: 768, // Standard embedding dimension
				Metric:     "cosine",
				IndexType:  "hnsw",
				Options:    make(map[string]interface{}),
			},
			KeywordStore: storage.KeywordConfig{
				Backend:   "bleve",
				IndexPath: filepath.Join(dataDir, "keyword_index"),
				Analyzer:  "standard",
				Options:   make(map[string]interface{}),
			},
			DocumentStore: storage.DocumentConfig{
				Backend: "sqlite",
				DBPath:  filepath.Join(dataDir, "vectors.db"), // Use same DB as vector store for SQLite backend
				Options: make(map[string]interface{}),
			},
			Ingestion: ingest.Config{
				ChunkingStrategy: core.ChunkingConfig{
					Strategy:     c.config.RAG.ChunkingStrategy.Strategy,
					ChunkSize:    c.config.RAG.ChunkingStrategy.ChunkSize,
					ChunkOverlap: c.config.RAG.ChunkingStrategy.ChunkOverlap,
					MinChunkSize: 100, // Default minimum chunk size
				},
				MaxConcurrency: 4,
				BatchSize:      100,
			},
			BatchSize:     100,
			MaxConcurrent: 4,
		}
		
		// Create an embedder adapter that uses the LLM service
		embedder := &LLMEmbedder{llm: c.llmService}
		
		// Initialize REAL RAG service with the actual implementation
		ragSvc, err := rag.NewService(ragConfig, embedder, c.llmService)
		if err != nil {
			return fmt.Errorf("failed to initialize RAG service: %w", err)
		}
		c.ragService = ragSvc
		enabledPillars = append(enabledPillars, "RAG")
	}
	
	// Initialize MCP pillar if enabled
	if !c.config.Mode.DisableMCP {
		mcpSvc, err := mcp.NewService(c.config.MCP)
		if err != nil {
			// MCP initialization is optional - log warning but don't fail
			fmt.Printf("Warning: MCP service initialization failed: %v\n", err)
			c.config.Mode.DisableMCP = true // Disable MCP if it can't initialize
		} else {
			c.mcpService = mcpSvc
			
			enabledPillars = append(enabledPillars, "MCP")
		}
	}
	
	// Initialize Agents pillar if enabled
	if !c.config.Mode.DisableAgent {
		agentSvc, err := agents.NewService(c.config.Agents)
		if err != nil {
			return fmt.Errorf("failed to initialize Agents service: %w", err)
		}
		c.agentService = agentSvc
		enabledPillars = append(enabledPillars, "Agent")
	}
	
	// Wire up cross-pillar integrations
	// Connect MCP service to LLM service for tool calling
	if c.llmService != nil && c.mcpService != nil {
		// Check if LLM service has SetMCPService method
		if llmWithMCP, ok := c.llmService.(interface {
			SetMCPService(llm.MCPService)
		}); ok {
			// Create an adapter that implements llm.MCPService interface
			mcpAdapter := &mcpServiceAdapter{service: c.mcpService}
			llmWithMCP.SetMCPService(mcpAdapter)
		}
	}
	
	// Validate that at least one pillar is enabled
	if len(enabledPillars) == 0 {
		return fmt.Errorf("no pillars enabled - at least one pillar (LLM, RAG, MCP, or Agent) must be enabled")
	}
	
	return nil
}

// shouldDisableLLM checks if LLM pillar should be disabled
func (c *Client) shouldDisableLLM() bool {
	return c.config.Mode.RAGOnly
}

// shouldDisableRAG checks if RAG pillar should be disabled
func (c *Client) shouldDisableRAG() bool {
	return c.config.Mode.LLMOnly
}

// Close closes the client and releases all resources.
// This implements proper lifecycle management for all pillars.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.closed {
		return nil
	}
	
	c.cancel()
	var errs []error

	// Close all pillar services
	if c.llmService != nil {
		if closer, ok := c.llmService.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				errs = append(errs, fmt.Errorf("failed to close LLM service: %w", err))
			}
		}
	}
	
	if c.ragService != nil {
		if closer, ok := c.ragService.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				errs = append(errs, fmt.Errorf("failed to close RAG service: %w", err))
			}
		}
	}
	
	if c.mcpService != nil {
		if closer, ok := c.mcpService.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				errs = append(errs, fmt.Errorf("failed to close MCP service: %w", err))
			}
		}
	}
	
	if c.agentService != nil {
		if closer, ok := c.agentService.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				errs = append(errs, fmt.Errorf("failed to close Agent service: %w", err))
			}
		}
	}
	
	c.closed = true
	
	if len(errs) > 0 {
		return fmt.Errorf("errors closing client: %v", errs)
	}

	return nil
}

// ===== PILLAR ACCESS METHODS =====
// These methods provide access to individual pillars, enabling library users
// to use specific functionality without initializing unused pillars.

// LLM returns the LLM service for direct language model operations.
func (c *Client) LLM() core.LLMService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.llmService
}

// RAG returns the RAG service for document operations and retrieval.
func (c *Client) RAG() core.RAGService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ragService
}

// MCP returns the MCP service for tool operations and external integrations.
func (c *Client) MCP() core.MCPService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.mcpService
}

// Agents returns the Agent service for workflow and task automation.
func (c *Client) Agents() core.AgentService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.agentService
}

// GetConfig returns the client configuration
func (c *Client) GetConfig() *Config {
	return c.config
}

// ===== HIGH-LEVEL OPERATIONS =====
// These operations combine multiple pillars to provide high-level functionality
// that showcases the power of the four-pillar architecture.

// Chat performs an intelligent chat operation using multiple pillars.
// It can combine LLM generation, RAG context, and MCP tools based on the request.
func (c *Client) Chat(ctx context.Context, req core.ChatRequest) (*core.ChatResponse, error) {
	start := time.Now()
	
	response := &core.ChatResponse{
		Context: req.Context,
		Duration: 0, // Will be set at the end
	}
	
	// Build generation request
	genReq := core.GenerationRequest{
		Prompt:     req.Message,
		Parameters: req.Parameters,
	}
	
	// Add context messages if provided
	if len(req.Context) > 0 {
		genReq.Context = req.Context
	}
	
	// Enhance with RAG if requested and available
	if req.UseRAG && c.ragService != nil {
		searchReq := core.SearchRequest{
			Query: req.Message,
			Limit: 5, // Default to top 5 results
		}
		
		searchResp, err := c.ragService.Search(ctx, searchReq)
		if err == nil && len(searchResp.Results) > 0 {
			response.Sources = searchResp.Results
			
			// Build context from search results
			var contextParts []string
			for _, result := range searchResp.Results {
				contextParts = append(contextParts, result.Content)
			}
			
			// Enhance prompt with context
			contextStr := fmt.Sprintf("Context information:\n%s\n\nUser question: %s", 
				string(contextParts[0]), req.Message) // Simplified for now
			genReq.Prompt = contextStr
		}
	}
	
	// Generate response using LLM
	if c.llmService == nil {
		return nil, fmt.Errorf("LLM service not available")
	}
	
	var genResp *core.GenerationResponse
	var err error
	
	if req.UseTools && c.mcpService != nil {
		// Get available tools
		tools := c.mcpService.GetTools()
		toolGenReq := core.ToolGenerationRequest{
			GenerationRequest: genReq,
			Tools:             tools,
			MaxToolCalls:      3, // Default limit
		}
		
		toolResp, err := c.llmService.GenerateWithTools(ctx, toolGenReq)
		if err != nil {
			return nil, fmt.Errorf("tool generation failed: %w", err)
		}
		
		genResp = &toolResp.GenerationResponse
		response.ToolCalls = toolResp.ToolCalls
	} else {
		genResp, err = c.llmService.Generate(ctx, genReq)
		if err != nil {
			return nil, fmt.Errorf("generation failed: %w", err)
		}
	}
	
	response.Response = genResp.Content
	response.Usage = genResp.Usage
	
	// Update context with the new message
	response.Context = append(response.Context, core.Message{
		Role:    "assistant",
		Content: genResp.Content,
	})
	
	response.Duration = time.Since(start)
	return response, nil
}

// StreamChat performs streaming chat with real-time response delivery.
func (c *Client) StreamChat(ctx context.Context, req core.ChatRequest, callback core.StreamCallback) error {
	if c.llmService == nil {
		return fmt.Errorf("LLM service not available")
	}
	
	// Build generation request (similar to Chat but for streaming)
	genReq := core.GenerationRequest{
		Prompt:     req.Message,
		Parameters: req.Parameters,
		Context:    req.Context,
	}
	
	// Enhance with RAG if requested
	if req.UseRAG && c.ragService != nil {
		// Same logic as Chat method for context enhancement
		searchReq := core.SearchRequest{
			Query: req.Message,
			Limit: 5,
		}
		
		if searchResp, err := c.ragService.Search(ctx, searchReq); err == nil && len(searchResp.Results) > 0 {
			var contextParts []string
			for _, result := range searchResp.Results {
				contextParts = append(contextParts, result.Content)
			}
			contextStr := fmt.Sprintf("Context information:\n%s\n\nUser question: %s", 
				string(contextParts[0]), req.Message)
			genReq.Prompt = contextStr
		}
	}
	
	return c.llmService.Stream(ctx, genReq, callback)
}

// ProcessDocument performs comprehensive document processing using multiple pillars.
func (c *Client) ProcessDocument(ctx context.Context, req core.DocumentRequest) (*core.DocumentResponse, error) {
	start := time.Now()
	
	response := &core.DocumentResponse{
		Action:     req.Action,
		DocumentID: req.DocumentID,
	}
	
	switch req.Action {
	case "analyze":
		if c.llmService == nil {
			return nil, fmt.Errorf("LLM service required for document analysis")
		}
		
		// Get document content
		var content string
		if req.Content != "" {
			content = req.Content
		} else if req.DocumentID != "" && c.ragService != nil {
			// Retrieve document from RAG
			docs, err := c.ragService.ListDocuments(ctx, core.DocumentFilter{
				Limit:  1,
				Offset: 0, // This would need proper filtering by ID
			})
			if err != nil || len(docs) == 0 {
				return nil, fmt.Errorf("document not found: %s", req.DocumentID)
			}
			content = docs[0].Content
			response.DocumentID = docs[0].ID
		} else {
			return nil, fmt.Errorf("no content provided for analysis")
		}
		
		// Generate analysis
		prompt := fmt.Sprintf("Please analyze the following document and provide a comprehensive summary:\n\n%s", content)
		genReq := core.GenerationRequest{
			Prompt:     prompt,
			Parameters: req.Parameters,
		}
		
		genResp, err := c.llmService.Generate(ctx, genReq)
		if err != nil {
			return nil, fmt.Errorf("analysis generation failed: %w", err)
		}
		
		response.Result = genResp.Content
		response.Usage = genResp.Usage
		
	case "ingest":
		if c.ragService == nil {
			return nil, fmt.Errorf("RAG service required for document ingestion")
		}
		
		// Ingest document
		ingestReq := core.IngestRequest{
			DocumentID:  req.DocumentID,
			Content:     req.Content,
			ContentType: "text/plain", // Default, could be enhanced
		}
		
		ingestResp, err := c.ragService.IngestDocument(ctx, ingestReq)
		if err != nil {
			return nil, fmt.Errorf("document ingestion failed: %w", err)
		}
		
		response.DocumentID = ingestResp.DocumentID
		response.Result = fmt.Sprintf("Successfully ingested document with %d chunks", ingestResp.ChunksCount)
		
	default:
		return nil, fmt.Errorf("unsupported document action: %s", req.Action)
	}
	
	response.Duration = time.Since(start)
	return response, nil
}

// ExecuteTask performs complex task execution using agent workflows.
func (c *Client) ExecuteTask(ctx context.Context, req core.TaskRequest) (*core.TaskResponse, error) {
	start := time.Now()
	
	response := &core.TaskResponse{
		Task: req.Task,
	}
	
	// If a specific agent is requested
	if req.Agent != "" && c.agentService != nil {
		agentReq := core.AgentRequest{
			AgentName: req.Agent,
			Task:      req.Task,
			Context:   req.Context,
		}
		
		agentResp, err := c.agentService.ExecuteAgent(ctx, agentReq)
		if err != nil {
			return nil, fmt.Errorf("agent execution failed: %w", err)
		}
		
		response.Result = agentResp.Result
		// Convert agent steps to task steps
		for _, step := range agentResp.Steps {
			response.Steps = append(response.Steps, core.StepResult{
				StepID:   fmt.Sprintf("step-%d", step.StepNumber),
				Status:   "completed",
				Output:   step.Output,
				Duration: step.Duration,
			})
		}
		
	} else if req.Workflow != "" && c.agentService != nil {
		// Execute specific workflow
		workflowReq := core.WorkflowRequest{
			WorkflowName: req.Workflow,
			Inputs:       req.Parameters,
			Context:      req.Context,
		}
		
		workflowResp, err := c.agentService.ExecuteWorkflow(ctx, workflowReq)
		if err != nil {
			return nil, fmt.Errorf("workflow execution failed: %w", err)
		}
		
		response.Result = fmt.Sprintf("Workflow '%s' completed with status: %s", 
			workflowResp.WorkflowName, workflowResp.Status)
		response.Steps = workflowResp.Steps
		
	} else {
		// Fallback to LLM-based task execution
		if c.llmService == nil {
			return nil, fmt.Errorf("no execution method available - need either agents or LLM service")
		}
		
		genReq := core.GenerationRequest{
			Prompt:     fmt.Sprintf("Please help me with this task: %s", req.Task),
			Parameters: req.Parameters,
		}
		
		genResp, err := c.llmService.Generate(ctx, genReq)
		if err != nil {
			return nil, fmt.Errorf("task execution failed: %w", err)
		}
		
		response.Result = genResp.Content
		response.Usage = genResp.Usage
	}
	
	response.Duration = time.Since(start)
	return response, nil
}

// Health returns comprehensive health information for all pillars.
func (c *Client) Health() core.HealthReport {
	if c.healthMonitor == nil {
		return core.HealthReport{
			Overall:   core.HealthStatusUnknown,
			LastCheck: time.Now(),
			Details:   map[string]interface{}{"error": "health monitor not initialized"},
		}
	}
	
	return c.healthMonitor.GetHealthReport()
}

// TriggerHealthCheck triggers an immediate health check (optimized for LLM only by default)
func (c *Client) TriggerHealthCheck() {
	if c.healthMonitor != nil {
		c.healthMonitor.TriggerHealthCheck()
	}
}

// TriggerFullHealthCheck triggers a comprehensive health check on ALL pillars
// This is slower but more thorough than the default TriggerHealthCheck
func (c *Client) TriggerFullHealthCheck() {
	if c.healthMonitor != nil {
		c.healthMonitor.TriggerFullHealthCheck()
	}
}

// Helper functions for REAL RAG service initialization

// LLMEmbedder adapts LLMService to work as an embedder for RAG storage
type LLMEmbedder struct {
	llm core.LLMService
}

// Embed implements the embedder interface using LLM providers
func (e *LLMEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	// Try to get embeddings from the LLM service if it has the GenerateEmbedding method
	if embedder, ok := e.llm.(interface {
		GenerateEmbedding(ctx context.Context, text string) ([]float64, error)
	}); ok {
		// Try to use the embeddings if the LLM service supports it
		embedding, err := embedder.GenerateEmbedding(ctx, text)
		if err == nil && len(embedding) > 0 {
			return embedding, nil
		}
		// If it fails, continue to fallback
	}
	
	// Log warning about using fallback embeddings
	fmt.Printf("Warning: No embedding provider available, using fallback hash-based embeddings. Install an embedding model (e.g., nomic-embed-text) for better RAG performance.\n")
	
	// Fallback to hash-based embedding if no real embeddings available
	return createHashEmbedding(text), nil
}


// Utility functions
func contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || 
		(len(str) > len(substr) && 
		(str[:len(substr)] == substr || 
		 str[len(str)-len(substr):] == substr ||
		 strings.Contains(str, substr))))
}

func createHashEmbedding(text string) []float64 {
	// Create a deterministic hash-based embedding
	embedding := make([]float64, 768)
	hash := 0
	for _, char := range text {
		hash = hash*31 + int(char)
	}
	
	for i := range embedding {
		embedding[i] = float64((hash+i)%1000) / 1000.0
	}
	
	return embedding
}

// Helper functions to create default configurations
func createDefaultIngestionConfig() interface{} {
	// Return a basic ingestion config with chunking strategy
	return map[string]interface{}{
		"chunking_strategy": "recursive",
		"chunk_size":        1000,
		"chunk_overlap":     200,
		"max_chunks":        100,
	}
}

func createDefaultSearchConfig() interface{} {
	// Return a basic search config
	return map[string]interface{}{
		"hybrid_alpha":    0.5,
		"rerank":          true,
		"max_results":     20,
		"similarity_threshold": 0.7,
	}
}

// mcpServiceAdapter adapts core.MCPService to llm.MCPService interface
type mcpServiceAdapter struct {
	service core.MCPService
}

func (a *mcpServiceAdapter) GetTools() []core.ToolInfo {
	return a.service.GetTools()
}

func (a *mcpServiceAdapter) GetToolsForLLM() []core.ToolInfo {
	return a.service.GetToolsForLLM()
}
