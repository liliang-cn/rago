package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/rag"
	"github.com/liliang-cn/rago/v2/pkg/rag/store"
)

// Agent provides a high-level interface for planning and executing workflows
type Agent struct {
	planner    *Planner
	executor   *Executor
	config     *config.Config
	storage    *PlanStorage
	llm        domain.Generator
	ragService *rag.Service
	mcpManager *mcp.Manager
	verbose    bool
}

// NewAgent creates a new agent with planner and executor
func NewAgent(cfg *config.Config, llm domain.Generator, mcpManager *mcp.Manager) *Agent {
	return NewAgentWithEmbedder(cfg, llm, nil, mcpManager)
}

// NewAgentWithEmbedder creates a new agent with embedder for RAG support
func NewAgentWithEmbedder(cfg *config.Config, llm domain.Generator, embedder domain.EmbedderProvider, mcpManager *mcp.Manager) *Agent {
	// Get configured paths with defaults
	dataPath := ".rago/data"
	if cfg.Agents != nil && cfg.Agents.DataPath != "" {
		dataPath = cfg.Agents.DataPath
	}
	
	// Ensure data directory exists
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create data directory: %v", err))
	}
	
	// Initialize shared database storage in configured data path
	dbPath := filepath.Join(dataPath, "plans.db")
	storage, err := NewPlanStorage(dbPath)
	if err != nil {
		// Database is required now
		panic(fmt.Sprintf("Failed to initialize plan database: %v", err))
	}
	
	// Create planner and executor (they will create their own storage connections)
	planner := NewPlanner(cfg, llm, mcpManager)
	executor := NewExecutor(cfg, llm, mcpManager)
	
	// Initialize RAG service (optional - may be nil)
	var ragService *rag.Service
	if embedder != nil {
		ragDBPath := filepath.Join(dataPath, "rag.db")
		if fileExists(ragDBPath) {
			if ragStore, err := store.NewSQLiteStore(ragDBPath); err == nil {
				ragService = rag.NewService(ragStore, embedder)
			}
		}
	}
	
	return &Agent{
		planner:    planner,
		executor:   executor,
		config:     cfg,
		storage:    storage,
		llm:        llm,
		ragService: ragService,
		mcpManager: mcpManager,
		verbose:    false,
	}
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// SetVerbose enables verbose output for both planner and executor
func (a *Agent) SetVerbose(v bool) {
	a.verbose = v
	a.planner.SetVerbose(v)
	a.executor.SetVerbose(v)
}

// PlanAndExecute creates a plan and executes it immediately
func (a *Agent) PlanAndExecute(ctx context.Context, request string) (*AgentResult, error) {
	// First, create the plan
	planID, err := a.planner.Plan(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("planning failed: %w", err)
	}

	// Then execute the plan
	results, err := a.executor.ExecutePlan(ctx, planID)
	if err != nil {
		return &AgentResult{
			PlanID:  planID,
			Success: false,
			Error:   err,
		}, err
	}

	return &AgentResult{
		PlanID:  planID,
		Results: results,
		Success: true,
	}, nil
}

// PlanOnly creates and saves a plan without executing it
func (a *Agent) PlanOnly(ctx context.Context, request string) (string, error) {
	return a.planner.Plan(ctx, request)
}

// ExecuteOnly executes a saved plan by ID
func (a *Agent) ExecuteOnly(ctx context.Context, planID string) (map[string]interface{}, error) {
	return a.executor.ExecutePlan(ctx, planID)
}

// ListPlans returns a list of all saved plans from database
func (a *Agent) ListPlans() ([]PlanRecord, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("database storage not available")
	}
	// Get last 100 plans
	return a.storage.ListPlans(100, 0)
}

// GetPlan reads and returns a saved plan from database
func (a *Agent) GetPlan(planID string) (*Plan, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("database storage not available")
	}
	return a.storage.GetPlan(planID)
}

// DeletePlan removes a saved plan from database
func (a *Agent) DeletePlan(planID string) error {
	if a.storage == nil {
		return fmt.Errorf("database storage not available")
	}
	return a.storage.DeletePlan(planID)
}

// AgentResult contains the results of a plan and execute operation
type AgentResult struct {
	PlanID  string                 `json:"plan_id"`
	Results map[string]interface{} `json:"results,omitempty"`
	Success bool                   `json:"success"`
	Error   error                  `json:"error,omitempty"`
}

// Helper function for JSON parsing
func parseJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// ListPlansFromDB retrieves plans from the database
func (a *Agent) ListPlansFromDB(limit, offset int) ([]PlanRecord, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("database storage not available")
	}
	return a.storage.ListPlans(limit, offset)
}

// SearchPlans searches for plans in the database
func (a *Agent) SearchPlans(searchTerm string) ([]PlanRecord, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("database storage not available")
	}
	return a.storage.SearchPlans(searchTerm)
}

// GetExecutionHistory retrieves execution history for a plan
func (a *Agent) GetExecutionHistory(planID string) ([]ExecutionRecord, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("database storage not available")
	}
	return a.storage.GetExecutionHistory(planID)
}

// GetPlanFromDB retrieves a plan from the database by ID
func (a *Agent) GetPlanFromDB(planID string) (*Plan, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("database storage not available")
	}
	return a.storage.GetPlan(planID)
}

// DeletePlanFromDB removes a plan from the database
func (a *Agent) DeletePlanFromDB(planID string) error {
	if a.storage == nil {
		return fmt.Errorf("database storage not available")
	}
	return a.storage.DeletePlan(planID)
}

// Close closes the database connection
func (a *Agent) Close() error {
	if a.storage != nil {
		return a.storage.Close()
	}
	return nil
}

// Do combines RAG retrieval, planning, and execution in one intelligent flow
func (a *Agent) Do(ctx context.Context, request string) (*DoResult, error) {
	result := &DoResult{
		Request: request,
	}
	
	// Step 1: RAG Retrieval - Get relevant context from knowledge base
	var ragContext string
	var relevantDocsCount int
	if a.ragService != nil && a.ragService.IsAvailable() {
		if a.verbose {
			fmt.Println("🔍 Searching knowledge base for relevant context...")
		}
		
		// Use RAG service to get relevant context
		context, docCount, err := a.ragService.GetRelevantContext(ctx, request, 5)
		if err == nil {
			ragContext = context
			relevantDocsCount = docCount
			result.RAGContext = ragContext
			
			if a.verbose {
				if relevantDocsCount > 0 {
					fmt.Printf("📚 Found %d relevant documents from knowledge base\n", relevantDocsCount)
				} else {
					fmt.Println("📚 No relevant documents found in knowledge base")
				}
			}
		}
	}
	
	// Step 2: Intent Recognition - Understand what the user wants
	var intent *domain.IntentResult
	if a.llm != nil {
		if a.verbose {
			fmt.Println("🎯 Recognizing user intent...")
		}
		intent, _ = a.llm.RecognizeIntent(ctx, request)
		if intent != nil {
			result.Intent = intent
			if a.verbose {
				fmt.Printf("   Intent: %s (confidence: %.2f)\n", intent.Intent, intent.Confidence)
				if intent.NeedsTools {
					fmt.Println("   Requires tools: Yes")
				} else {
					fmt.Println("   Requires tools: No")
				}
			}
		}
	}
	
	// Step 3: Enhanced Request with RAG Context
	enhancedRequest := request
	if ragContext != "" {
		// Use LLM to synthesize the context with the request
		synthesis, err := a.synthesizeWithContext(ctx, request, ragContext)
		if err == nil {
			enhancedRequest = synthesis
			result.EnhancedRequest = enhancedRequest
			
			if a.verbose {
				fmt.Println("🧠 Enhanced request with knowledge base context")
			}
		}
	}
	
	// Step 4: Smart Decision - Use intent to help determine if we need tools
	var needsTools bool
	var directAnswer string
	var err error
	
	// If intent recognition suggests we don't need tools and it's a simple question/calculation
	if intent != nil && !intent.NeedsTools && 
	   (intent.Intent == domain.IntentQuestion || intent.Intent == domain.IntentCalculation) {
		// Try to answer directly first
		needsTools, directAnswer, err = a.determineApproach(ctx, enhancedRequest, ragContext)
	} else {
		// Otherwise, do full approach determination
		needsTools, directAnswer, err = a.determineApproach(ctx, enhancedRequest, ragContext)
	}
	
	if err != nil {
		return result, fmt.Errorf("failed to determine approach: %w", err)
	}
	
	result.NeedsTools = needsTools
	
	// Step 5: Execute based on approach
	if !needsTools && directAnswer != "" {
		// Can answer directly from RAG + LLM
		result.DirectAnswer = directAnswer
		result.Success = true
		
		if a.verbose {
			fmt.Println("💡 Answering directly from knowledge base")
		}
		
		return result, nil
	}
	
	// Step 6: Need tools - Plan and Execute
	if a.verbose {
		fmt.Println("🔧 Request requires tool execution")
	}
	
	// Create plan with enhanced request
	planID, err := a.planner.Plan(ctx, enhancedRequest)
	if err != nil {
		return result, fmt.Errorf("planning failed: %w", err)
	}
	result.PlanID = planID
	
	// Execute the plan
	execResults, err := a.executor.ExecutePlan(ctx, planID)
	if err != nil {
		result.Error = err
		return result, err
	}
	
	result.ExecutionResults = execResults
	
	// Step 7: Synthesize final answer combining execution results and RAG context
	finalAnswer, err := a.synthesizeFinalAnswer(ctx, request, ragContext, execResults)
	if err == nil {
		result.FinalAnswer = finalAnswer
	}
	
	result.Success = true
	return result, nil
}

// synthesizeWithContext uses LLM to combine user request with RAG context
func (a *Agent) synthesizeWithContext(ctx context.Context, request, ragContext string) (string, error) {
	prompt := fmt.Sprintf(`Given the following context from the knowledge base and user request, 
create an enhanced request that incorporates relevant information from the context.

KNOWLEDGE BASE CONTEXT:
%s

USER REQUEST:
%s

ENHANCED REQUEST (be specific and include relevant details from context):`, ragContext, request)

	opts := &domain.GenerationOptions{
		Temperature: 0.3,
		MaxTokens:   500,
	}
	
	response, err := a.llm.Generate(ctx, prompt, opts)
	if err != nil {
		return request, err // Fallback to original request
	}
	
	return strings.TrimSpace(response), nil
}

// determineApproach decides if we need tools or can answer directly
func (a *Agent) determineApproach(ctx context.Context, request, ragContext string) (needsTools bool, directAnswer string, err error) {
	prompt := fmt.Sprintf(`Analyze this request and determine if it requires tool execution or can be answered directly.

REQUEST: %s

AVAILABLE CONTEXT:
%s

AVAILABLE TOOLS:
%s

Respond with JSON:
{
  "needs_tools": true/false,
  "reason": "why tools are needed or not",
  "direct_answer": "answer if no tools needed, empty string otherwise"
}`, request, ragContext, a.getAvailableToolsSummary())

	opts := &domain.GenerationOptions{
		Temperature: 0.2,
		MaxTokens:   1000,
	}
	
	response, err := a.llm.Generate(ctx, prompt, opts)
	if err != nil {
		return true, "", nil // Default to needing tools
	}
	
	// Parse JSON response
	var decision struct {
		NeedsTools   bool   `json:"needs_tools"`
		Reason       string `json:"reason"`
		DirectAnswer string `json:"direct_answer"`
	}
	
	// Extract JSON from response
	jsonStr := extractJSON(response)
	if err := json.Unmarshal([]byte(jsonStr), &decision); err != nil {
		return true, "", nil // Default to needing tools
	}
	
	return decision.NeedsTools, decision.DirectAnswer, nil
}

// synthesizeFinalAnswer creates a comprehensive answer from all sources
func (a *Agent) synthesizeFinalAnswer(ctx context.Context, request, ragContext string, execResults map[string]interface{}) (string, error) {
	resultsJSON, _ := json.MarshalIndent(execResults, "", "  ")
	
	prompt := fmt.Sprintf(`Create a comprehensive answer to the user's request based on all available information.

USER REQUEST:
%s

KNOWLEDGE BASE CONTEXT:
%s

TOOL EXECUTION RESULTS:
%s

Provide a clear, concise answer that combines insights from all sources:`, request, ragContext, string(resultsJSON))

	opts := &domain.GenerationOptions{
		Temperature: 0.4,
		MaxTokens:   1000,
	}
	
	response, err := a.llm.Generate(ctx, prompt, opts)
	if err != nil {
		return "", err
	}
	
	return strings.TrimSpace(response), nil
}

// getAvailableToolsSummary returns a summary of available MCP tools
func (a *Agent) getAvailableToolsSummary() string {
	if a.mcpManager == nil {
		return "No tools available"
	}
	
	// Use the new MCP package method
	return a.mcpManager.GetToolsDescription(context.Background())
}

// extractJSON is a helper to extract JSON from text
func extractJSON(content string) string {
	// Remove thinking tags if present
	content = strings.ReplaceAll(content, "<think>", "")
	content = strings.ReplaceAll(content, "</think>", "")
	
	// Find JSON content between ```json and ```
	if strings.Contains(content, "```json") {
		start := strings.Index(content, "```json") + 7
		end := strings.Index(content[start:], "```")
		if end > 0 {
			return strings.TrimSpace(content[start : start+end])
		}
	}
	
	// Try to find JSON object directly
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start >= 0 && end > start {
		return content[start : end+1]
	}
	
	return content
}

// DoResult contains the comprehensive result of the Do operation
type DoResult struct {
	Request          string                 `json:"request"`
	Intent           *domain.IntentResult   `json:"intent,omitempty"`
	RAGContext       string                 `json:"rag_context,omitempty"`
	EnhancedRequest  string                 `json:"enhanced_request,omitempty"`
	NeedsTools       bool                   `json:"needs_tools"`
	DirectAnswer     string                 `json:"direct_answer,omitempty"`
	PlanID           string                 `json:"plan_id,omitempty"`
	ExecutionResults map[string]interface{} `json:"execution_results,omitempty"`
	FinalAnswer      string                 `json:"final_answer,omitempty"`
	Success          bool                   `json:"success"`
	Error            error                  `json:"error,omitempty"`
}