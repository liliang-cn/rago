// Package client provides the unified client interface for all RAGO pillars.
// This package implements the four-pillar architecture allowing access to
// LLM, RAG, MCP, and Agent services through a single interface.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	
	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/llm"
	"github.com/liliang-cn/rago/v2/pkg/rag"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/agents"
)

// Client implements the unified RAGO client interface.
// It provides access to all four pillars and high-level operations.
type Client struct {
	config core.Config
	
	// Individual pillar services
	llmService    *llm.Service
	ragService    *rag.Service
	mcpService    *mcp.Service
	agentService  *agents.Service
}

// New creates a new unified RAGO client with the provided configuration.
func New(config core.Config) (*Client, error) {
	client := &Client{
		config: config,
	}
	
	var err error
	
	// Initialize LLM pillar if not disabled
	if !config.Mode.LLMOnly || !config.Mode.DisableAgent {
		client.llmService, err = llm.NewService(config.LLM)
		if err != nil {
			return nil, core.WrapErrorWithContext(err, "client", "New", "failed to initialize LLM service")
		}
	}
	
	// Initialize RAG pillar if not disabled
	if !config.Mode.RAGOnly || !config.Mode.DisableAgent {
		// Use backward compatibility adapter with default embedder
		embedder := &rag.DefaultEmbedder{}
		client.ragService, err = rag.NewServiceFromCoreConfig(config.RAG, embedder)
		if err != nil {
			return nil, core.WrapErrorWithContext(err, "client", "New", "failed to initialize RAG service")
		}
	}
	
	// Initialize MCP pillar if not disabled
	if !config.Mode.DisableMCP {
		client.mcpService, err = mcp.NewService(config.MCP)
		if err != nil {
			return nil, core.WrapErrorWithContext(err, "client", "New", "failed to initialize MCP service")
		}
	}
	
	// Initialize Agent pillar if not disabled
	if !config.Mode.DisableAgent {
		client.agentService, err = agents.NewService(config.Agents)
		if err != nil {
			return nil, core.WrapErrorWithContext(err, "client", "New", "failed to initialize Agent service")
		}
	}
	
	return client, nil
}

// ===== PILLAR ACCESS =====

// LLM returns the LLM service interface.
func (c *Client) LLM() core.LLMService {
	return c.llmService
}

// RAG returns the RAG service interface.
func (c *Client) RAG() core.RAGService {
	return c.ragService
}

// MCP returns the MCP service interface.
func (c *Client) MCP() core.MCPService {
	return c.mcpService
}

// Agents returns the Agent service interface.
func (c *Client) Agents() core.AgentService {
	return c.agentService
}

// ===== HIGH-LEVEL OPERATIONS =====

// Chat provides a high-level chat interface using multiple pillars.
// This operation combines RAG for context retrieval, LLM for generation, and MCP tools as needed.
func (c *Client) Chat(ctx context.Context, req core.ChatRequest) (*core.ChatResponse, error) {
	response := &core.ChatResponse{
		Context: make([]core.Message, 0),
	}
	
	// Step 1: Check if RAG context is enabled and retrieve relevant knowledge
	if req.UseRAG && c.ragService != nil {
		// Extract limit and threshold from parameters or use defaults
		limit := 5
		threshold := float32(0.7)
		if l, ok := req.Parameters["rag_limit"].(int); ok {
			limit = l
		}
		if t, ok := req.Parameters["rag_threshold"].(float32); ok {
			threshold = t
		}
		
		searchReq := core.SearchRequest{
			Query:     req.Message,
			Limit:     limit,
			Threshold: threshold,
		}
		
		searchResults, err := c.ragService.Search(ctx, searchReq)
		if err == nil && searchResults != nil && len(searchResults.Results) > 0 {
			// Store search results for response
			response.Sources = searchResults.Results
			
			// Add RAG context as system messages
			for _, result := range searchResults.Results {
				response.Context = append(response.Context, core.Message{
					Role:    "system",
					Content: fmt.Sprintf("Context (relevance: %.2f): %s", result.Score, result.Content),
				})
			}
		}
	}
	
	// Step 2: Check if MCP tools are needed
	if req.UseTools && c.mcpService != nil {
		// Analyze the message to determine if tools are needed
		toolsNeeded := c.analyzeToolNeeds(req.Message)
		
		if len(toolsNeeded) > 0 {
			response.ToolCalls = make([]core.ToolCall, 0)
			
			for _, toolName := range toolsNeeded {
				toolReq := core.ToolCallRequest{
					ToolName:  toolName,
					Arguments: req.Parameters,
				}
				
				toolResp, err := c.mcpService.CallTool(ctx, toolReq)
				if err != nil {
					response.ToolCalls = append(response.ToolCalls, core.ToolCall{
						ID:   fmt.Sprintf("tool_%d", len(response.ToolCalls)),
						Name: toolName,
					})
				} else {
					response.ToolCalls = append(response.ToolCalls, core.ToolCall{
						ID:         fmt.Sprintf("tool_%d", len(response.ToolCalls)),
						Name:       toolName,
						Parameters: toolResp.Metadata,
					})
					
					// Add tool result as context
					response.Context = append(response.Context, core.Message{
						Role:    "tool",
						Content: fmt.Sprintf("Tool %s result: %v", toolName, toolResp.Result),
					})
				}
			}
		}
	}
	
	// Step 3: Generate response using LLM with enriched context
	if c.llmService != nil {
		// Build messages including context
		messages := append(req.Context, response.Context...)
		messages = append(messages, core.Message{
			Role:    "user",
			Content: req.Message,
		})
		
		// Extract generation parameters
		temperature := float32(0.7)
		maxTokens := 1000
		if t, ok := req.Parameters["temperature"].(float32); ok {
			temperature = t
		}
		if m, ok := req.Parameters["max_tokens"].(int); ok {
			maxTokens = m
		}
		
		// Build prompt from messages
		var promptBuilder strings.Builder
		for _, msg := range messages {
			promptBuilder.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
		}
		
		genReq := core.GenerationRequest{
			Prompt:      promptBuilder.String(),
			Context:     messages,
			Temperature: temperature,
			MaxTokens:   maxTokens,
		}
		
		genResp, err := c.llmService.Generate(ctx, genReq)
		if err != nil {
			return nil, core.WrapErrorWithContext(err, "client", "Chat", "LLM generation failed")
		}
		
		response.Response = genResp.Content
		response.Usage = genResp.Usage
	} else {
		return nil, core.WrapErrorWithContext(core.ErrServiceUnavailable, "client", "Chat", "LLM service not available")
	}
	
	return response, nil
}

// StreamChat provides a streaming chat interface using multiple pillars.
// Similar to Chat but streams the LLM response in real-time.
func (c *Client) StreamChat(ctx context.Context, req core.ChatRequest, callback core.StreamCallback) error {
	// Step 1: Prepare context (same as Chat)
	var contextMessages []core.Message
	
	// Retrieve RAG context if enabled
	if req.UseRAG && c.ragService != nil {
		// Extract limit and threshold from parameters or use defaults
		limit := 5
		threshold := float32(0.7)
		if l, ok := req.Parameters["rag_limit"].(int); ok {
			limit = l
		}
		if t, ok := req.Parameters["rag_threshold"].(float32); ok {
			threshold = t
		}
		
		searchReq := core.SearchRequest{
			Query:     req.Message,
			Limit:     limit,
			Threshold: threshold,
		}
		
		searchResults, err := c.ragService.Search(ctx, searchReq)
		if err == nil && searchResults != nil && len(searchResults.Results) > 0 {
			for _, result := range searchResults.Results {
				contextMessages = append(contextMessages, core.Message{
					Role:    "system",
					Content: fmt.Sprintf("Context (relevance: %.2f): %s", result.Score, result.Content),
				})
			}
		}
	}
	
	// Execute MCP tools if needed
	if req.UseTools && c.mcpService != nil {
		toolsNeeded := c.analyzeToolNeeds(req.Message)
		
		if len(toolsNeeded) > 0 {
			for _, toolName := range toolsNeeded {
				toolReq := core.ToolCallRequest{
					ToolName:  toolName,
					Arguments: req.Parameters,
				}
				
				toolResp, err := c.mcpService.CallTool(ctx, toolReq)
				if err == nil {
					contextMessages = append(contextMessages, core.Message{
						Role:    "tool",
						Content: fmt.Sprintf("Tool %s result: %v", toolName, toolResp.Result),
					})
				}
			}
		}
	}
	
	// Step 2: Stream LLM response with enriched context
	if c.llmService != nil {
		// Build messages including context
		messages := append(req.Context, contextMessages...)
		messages = append(messages, core.Message{
			Role:    "user",
			Content: req.Message,
		})
		
		// Extract generation parameters
		temperature := float32(0.7)
		maxTokens := 1000
		if t, ok := req.Parameters["temperature"].(float32); ok {
			temperature = t
		}
		if m, ok := req.Parameters["max_tokens"].(int); ok {
			maxTokens = m
		}
		
		// Build prompt from messages
		var promptBuilder strings.Builder
		for _, msg := range messages {
			promptBuilder.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
		}
		
		streamReq := core.GenerationRequest{
			Prompt:      promptBuilder.String(),
			Context:     messages,
			Temperature: temperature,
			MaxTokens:   maxTokens,
		}
		
		return c.llmService.Stream(ctx, streamReq, callback)
	}
	
	return core.WrapErrorWithContext(core.ErrServiceUnavailable, "client", "StreamChat", "LLM service not available")
}

// ProcessDocument provides high-level document processing using multiple pillars.
// This operation uses RAG for ingestion, Agents for processing workflows, and LLM for analysis.
func (c *Client) ProcessDocument(ctx context.Context, req core.DocumentRequest) (*core.DocumentResponse, error) {
	response := &core.DocumentResponse{
		DocumentID: req.DocumentID,
		Action:     req.Action,
	}
	
	// Step 1: Ingest document into RAG if action requires it
	if (req.Action == "ingest" || req.Action == "analyze") && c.ragService != nil {
		ingestReq := core.IngestRequest{
			DocumentID: req.DocumentID,
			Content:    req.Content,
			Metadata:   req.Parameters,
		}
		
		_, err := c.ragService.IngestDocument(ctx, ingestReq)
		if err != nil {
			return nil, core.WrapErrorWithContext(err, "client", "ProcessDocument", "document ingestion failed")
		}
	}
	
	// Step 2: Perform the requested action
	switch req.Action {
	case "analyze":
		if c.llmService != nil {
			// Extract analysis prompt from parameters
			prompt := "Analyze this document and provide insights"
			if p, ok := req.Parameters["prompt"].(string); ok {
				prompt = p
			}
			
			analysisReq := core.GenerationRequest{
				Prompt:      fmt.Sprintf("%s\n\nDocument Content:\n%s", prompt, req.Content),
				Temperature: 0.3, // Lower temperature for analysis
				MaxTokens:   1000,
			}
			
			analysisResp, err := c.llmService.Generate(ctx, analysisReq)
			if err != nil {
				return nil, core.WrapErrorWithContext(err, "client", "ProcessDocument", "analysis failed")
			}
			
			response.Result = analysisResp.Content
			response.Usage = analysisResp.Usage
		}
		
	case "summarize":
		if c.llmService != nil {
			summarizeReq := core.GenerationRequest{
				Prompt:      fmt.Sprintf("Summarize the following document:\n\n%s", req.Content),
				Temperature: 0.3,
				MaxTokens:   500,
			}
			
			summarizeResp, err := c.llmService.Generate(ctx, summarizeReq)
			if err != nil {
				return nil, core.WrapErrorWithContext(err, "client", "ProcessDocument", "summarization failed")
			}
			
			response.Result = summarizeResp.Content
			response.Usage = summarizeResp.Usage
		}
		
	case "extract":
		if c.mcpService != nil {
			// Use MCP tools for extraction if available
			toolReq := core.ToolCallRequest{
				ToolName: "entity_extraction",
				Arguments: map[string]interface{}{
					"text":  req.Content,
					"types": req.Parameters["entity_types"],
				},
			}
			
			toolResp, err := c.mcpService.CallTool(ctx, toolReq)
			if err != nil {
				return nil, core.WrapErrorWithContext(err, "client", "ProcessDocument", "extraction failed")
			}
			
			resultStr, _ := json.Marshal(toolResp.Result)
			response.Result = string(resultStr)
		} else if c.llmService != nil {
			// Fallback to LLM for extraction
			extractReq := core.GenerationRequest{
				Prompt:      fmt.Sprintf("Extract key information from the following document:\n\n%s", req.Content),
				Temperature: 0.2,
				MaxTokens:   800,
			}
			
			extractResp, err := c.llmService.Generate(ctx, extractReq)
			if err != nil {
				return nil, core.WrapErrorWithContext(err, "client", "ProcessDocument", "extraction failed")
			}
			
			response.Result = extractResp.Content
			response.Usage = extractResp.Usage
		}
		
	default:
		response.Result = "Unknown action: " + req.Action
	}
	return response, nil
}

// ExecuteTask provides high-level task execution using multiple pillars.
// This operation orchestrates Agents with MCP tools, LLM reasoning, and RAG knowledge.
func (c *Client) ExecuteTask(ctx context.Context, req core.TaskRequest) (*core.TaskResponse, error) {
	response := &core.TaskResponse{
		Task:  req.Task,
		Steps: make([]core.StepResult, 0),
	}
	
	// Step 1: Try to use Agent if available and specified
	if req.Agent != "" && c.agentService != nil {
		// Execute through a specific agent
		agentReq := core.AgentRequest{
			AgentName: req.Agent,
			Task:      req.Task,
			Context:   req.Context,
		}
		
		agentResp, err := c.agentService.ExecuteAgent(ctx, agentReq)
		if err != nil {
			return nil, core.WrapErrorWithContext(err, "client", "ExecuteTask", "agent execution failed")
		}
		
		// Convert agent steps to task steps
		for i, step := range agentResp.Steps {
			response.Steps = append(response.Steps, core.StepResult{
				StepID: fmt.Sprintf("step_%d", i+1),
				Output: step.Output,
				Status: "completed",
			})
		}
		
		response.Result = agentResp.Result
		
	} else if req.Workflow != "" && c.agentService != nil {
		// Execute through a workflow
		workflowReq := core.WorkflowRequest{
			WorkflowName: req.Workflow,
			Inputs:       req.Context,
		}
		
		workflowResp, err := c.agentService.ExecuteWorkflow(ctx, workflowReq)
		if err != nil {
			return nil, core.WrapErrorWithContext(err, "client", "ExecuteTask", "workflow execution failed")
		}
		
		response.Result = fmt.Sprintf("%v", workflowResp.Outputs)
		response.Duration = workflowResp.Duration
		
		// Add workflow steps
		for _, step := range workflowResp.Steps {
			response.Steps = append(response.Steps, core.StepResult{
				StepID: step.StepID,
				Output: step.Output,
				Status: step.Status,
			})
		}
		
	} else if c.llmService != nil {
		// Fallback to LLM-based task execution
		genReq := core.GenerationRequest{
			Prompt:      fmt.Sprintf("Execute the following task: %s\n\nContext: %v", req.Task, req.Context),
			Temperature: 0.7,
			MaxTokens:   2000,
		}
		
		genResp, err := c.llmService.Generate(ctx, genReq)
		if err != nil {
			return nil, core.WrapErrorWithContext(err, "client", "ExecuteTask", "LLM execution failed")
		}
		
		response.Result = genResp.Content
		response.Usage = genResp.Usage
		
		// Add a single step for LLM execution
		response.Steps = append(response.Steps, core.StepResult{
			StepID: "llm_execution",
			Output: "Task completed using LLM",
			Status: "success",
		})
		
	} else {
		return nil, core.WrapErrorWithContext(core.ErrServiceUnavailable, "client", "ExecuteTask", "no execution service available")
	}
	
	return response, nil
}

// ===== HELPER METHODS =====

// analyzeToolNeeds analyzes a message to determine which MCP tools might be needed
func (c *Client) analyzeToolNeeds(message string) []string {
	tools := make([]string, 0)
	
	// Simple keyword-based analysis (can be enhanced with LLM classification)
	keywords := map[string][]string{
		"filesystem": {"file", "directory", "folder", "read", "write", "create", "delete"},
		"web":        {"website", "url", "fetch", "download", "api", "http"},
		"database":   {"query", "sql", "database", "table", "record"},
		"calculator": {"calculate", "compute", "math", "sum", "average"},
	}
	
	messageLower := strings.ToLower(message)
	for tool, words := range keywords {
		for _, word := range words {
			if strings.Contains(messageLower, word) {
				tools = append(tools, tool)
				break
			}
		}
	}
	
	return tools
}


// ===== CONVENIENCE CONSTRUCTORS =====

// NewFromPath creates a client from a configuration file path.
func NewFromPath(configPath string) (core.Client, error) {
	config, err := LoadCoreConfigFromPath(configPath)
	if err != nil {
		return nil, core.WrapError(err, "failed to load configuration")
	}
	return New(*config)
}

// NewDefault creates a client from default configuration locations.
func NewDefault() (core.Client, error) {
	config, err := LoadDefaultCoreConfig()
	if err != nil {
		return nil, core.WrapError(err, "failed to load default configuration")
	}
	return New(*config)
}

// ===== CLIENT MANAGEMENT =====

// Close closes all pillar services and cleans up resources.
func (c *Client) Close() error {
	var lastErr error
	
	if c.llmService != nil {
		if err := c.llmService.Close(); err != nil {
			lastErr = err
		}
	}
	
	if c.ragService != nil {
		if err := c.ragService.Close(); err != nil {
			lastErr = err
		}
	}
	
	if c.mcpService != nil {
		if err := c.mcpService.Close(); err != nil {
			lastErr = err
		}
	}
	
	if c.agentService != nil {
		if err := c.agentService.Close(); err != nil {
			lastErr = err
		}
	}
	
	return lastErr
}

// Health returns the health status of all pillars.
func (c *Client) Health() core.HealthReport {
	report := core.HealthReport{
		Overall: core.HealthStatusHealthy,
		Pillars: make(map[string]core.HealthStatus),
		Providers: make(map[string]core.HealthStatus),
		Servers: make(map[string]core.HealthStatus),
		Details: make(map[string]interface{}),
	}
	
	// Check LLM pillar health
	if c.llmService != nil {
		// LLM service doesn't have a Health method, so we check providers
		providerHealth := c.llmService.GetProviderHealth()
		// Determine overall LLM health from providers
		llmHealthy := false
		for name, status := range providerHealth {
			report.Providers["llm_"+name] = status
			if status == core.HealthStatusHealthy {
				llmHealthy = true
			}
		}
		
		if llmHealthy {
			report.Pillars["llm"] = core.HealthStatusHealthy
		} else if len(providerHealth) > 0 {
			report.Pillars["llm"] = core.HealthStatusDegraded
			report.Overall = core.HealthStatusDegraded
		} else {
			report.Pillars["llm"] = core.HealthStatusUnknown
			report.Overall = core.HealthStatusDegraded
		}
		
		// Store detailed metrics
		report.Details["llm"] = map[string]interface{}{
			"providers": providerHealth,
		}
	} else {
		report.Pillars["llm"] = core.HealthStatusUnknown
	}
	
	// Check RAG pillar health
	if c.ragService != nil {
		// RAG service health is assumed healthy if it exists
		report.Pillars["rag"] = core.HealthStatusHealthy
		
		// Store detailed metrics
		report.Details["rag"] = map[string]interface{}{
			"status": "operational",
		}
	} else {
		report.Pillars["rag"] = core.HealthStatusUnknown
	}
	
	// Check MCP pillar health
	if c.mcpService != nil {
		// Get MCP server health
		servers := c.mcpService.ListServers()
		healthyServers := 0
		for _, server := range servers {
			health := c.mcpService.GetServerHealth(server.Name)
			report.Servers[server.Name] = health
			if health == core.HealthStatusHealthy {
				healthyServers++
			}
		}
		
		if healthyServers > 0 {
			report.Pillars["mcp"] = core.HealthStatusHealthy
		} else if len(servers) > 0 {
			report.Pillars["mcp"] = core.HealthStatusDegraded
			report.Overall = core.HealthStatusDegraded
		} else {
			report.Pillars["mcp"] = core.HealthStatusUnknown
		}
		
		// Store detailed metrics
		report.Details["mcp"] = map[string]interface{}{
			"servers":     servers,
			"server_count": len(servers),
		}
	} else {
		report.Pillars["mcp"] = core.HealthStatusUnknown
	}
	
	// Check Agent pillar health
	if c.agentService != nil {
		// Agent service health is assumed healthy if it exists
		report.Pillars["agents"] = core.HealthStatusHealthy
		
		// Store detailed metrics
		report.Details["agents"] = map[string]interface{}{
			"status": "operational",
		}
	} else {
		report.Pillars["agents"] = core.HealthStatusUnknown
	}
	
	// Final status check - if any pillar is unhealthy, mark overall as unhealthy
	unhealthyCount := 0
	for _, status := range report.Pillars {
		if status == core.HealthStatusUnhealthy {
			unhealthyCount++
		}
	}
	
	if unhealthyCount > 0 {
		if unhealthyCount == len(report.Pillars) {
			report.Overall = core.HealthStatusUnhealthy
		} else {
			report.Overall = core.HealthStatusDegraded
		}
	}
	
	// Add summary to details
	report.Details["summary"] = map[string]interface{}{
		"healthy_pillars":   c.countHealthyPillars(report.Pillars),
		"degraded_pillars":  c.countDegradedPillars(report.Pillars),
		"unhealthy_pillars": unhealthyCount,
		"total_pillars":     len(report.Pillars),
	}
	
	return report
}

// countHealthyPillars counts the number of healthy pillars
func (c *Client) countHealthyPillars(pillars map[string]core.HealthStatus) int {
	count := 0
	for _, status := range pillars {
		if status == core.HealthStatusHealthy {
			count++
		}
	}
	return count
}

// countDegradedPillars counts the number of degraded pillars
func (c *Client) countDegradedPillars(pillars map[string]core.HealthStatus) int {
	count := 0
	for _, status := range pillars {
		if status == core.HealthStatusDegraded {
			count++
		}
	}
	return count
}