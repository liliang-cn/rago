// Package agents implements the Agent pillar.
// This pillar focuses on workflow orchestration and multi-step reasoning.
package agents

import (
	"context"
	"fmt"
	randPkg "math/rand"
	"sync"
	"time"
	
	"github.com/liliang-cn/rago/v2/pkg/agents/reasoning"
	"github.com/liliang-cn/rago/v2/pkg/agents/scheduler"
	"github.com/liliang-cn/rago/v2/pkg/agents/workflow"
	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/llm"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/rag"
)

// Service implements the Agent pillar service interface.
// This is the main entry point for all agent operations including workflow
// management and agent execution.
type Service struct {
	config   core.AgentsConfig
	mu       sync.RWMutex
	
	// Core components
	workflowEngine *workflow.Engine
	scheduler      *scheduler.Scheduler
	reasoning      *reasoning.Engine
	
	// Pillar integrations
	llmService llm.Service
	ragService rag.Service
	mcpService mcp.Service
	
	// Agent and workflow storage
	agents    map[string]*Agent
	workflows map[string]*workflow.Definition
	
	// Execution tracking
	executions map[string]*ExecutionState
	
	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewService creates a new Agent service instance.
func NewService(config core.AgentsConfig, opts ...ServiceOption) (*Service, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	service := &Service{
		config:     config,
		agents:     make(map[string]*Agent),
		workflows:  make(map[string]*workflow.Definition),
		executions: make(map[string]*ExecutionState),
		ctx:        ctx,
		cancel:     cancel,
	}
	
	// Apply options
	for _, opt := range opts {
		if err := opt(service); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}
	
	// Initialize workflow engine
	workflowEngine, err := workflow.NewEngine(workflow.EngineConfig{
		MaxConcurrentWorkflows: config.WorkflowEngine.MaxConcurrentWorkflows,
		MaxConcurrentSteps:     config.WorkflowEngine.MaxConcurrentSteps,
		DefaultTimeout:         config.WorkflowEngine.DefaultTimeout,
		EnableParallelism:      config.WorkflowEngine.EnableParallelism,
		StateStoragePath:       config.StateStorage.Path,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow engine: %w", err)
	}
	service.workflowEngine = workflowEngine
	
	// Initialize scheduler
	schedulerInstance, err := scheduler.NewScheduler(scheduler.Config{
		MaxConcurrentJobs:   config.Scheduling.MaxConcurrentJobs,
		DefaultRetryPolicy:  config.Scheduling.RetryPolicy,
		PersistencePath:     config.Scheduling.PersistencePath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler: %w", err)
	}
	service.scheduler = schedulerInstance
	
	// Initialize reasoning engine
	reasoningEngine, err := reasoning.NewEngine(reasoning.Config{
		MaxDepth:            config.ReasoningChains.MaxDepth,
		MemoryCapacity:      config.ReasoningChains.MemoryCapacity,
		EnableLearning:      config.ReasoningChains.EnableLearning,
		PersistencePath:     config.ReasoningChains.PersistencePath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create reasoning engine: %w", err)
	}
	service.reasoning = reasoningEngine
	
	// Start background workers
	service.wg.Add(1)
	go service.runMaintenanceLoop()
	
	return service, nil
}

// ===== WORKFLOW MANAGEMENT =====

// CreateWorkflow creates a new workflow definition.
func (s *Service) CreateWorkflow(definition core.WorkflowDefinition) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Validate workflow definition
	if err := s.validateWorkflowDefinition(definition); err != nil {
		return fmt.Errorf("invalid workflow definition: %w", err)
	}
	
	// Check for duplicate name
	if _, exists := s.workflows[definition.Name]; exists {
		return fmt.Errorf("workflow %s already exists", definition.Name)
	}
	
	// Convert to internal workflow definition
	workflowDef := &workflow.Definition{
		ID:          generateID(),
		Name:        definition.Name,
		Description: definition.Description,
		Steps:       s.convertWorkflowSteps(definition.Steps),
		Inputs:      definition.Inputs,
		Outputs:     definition.Outputs,
		Metadata:    definition.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	// Register with workflow engine
	if err := s.workflowEngine.RegisterWorkflow(workflowDef); err != nil {
		return fmt.Errorf("failed to register workflow: %w", err)
	}
	
	// Store workflow
	s.workflows[definition.Name] = workflowDef
	
	return nil
}

// ExecuteWorkflow executes a workflow.
func (s *Service) ExecuteWorkflow(ctx context.Context, req core.WorkflowRequest) (*core.WorkflowResponse, error) {
	s.mu.RLock()
	workflowDef, exists := s.workflows[req.WorkflowName]
	s.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("workflow %s not found", req.WorkflowName)
	}
	
	// Create execution context
	execID := generateID()
	execCtx := &workflow.ExecutionContext{
		ID:         execID,
		WorkflowID: workflowDef.ID,
		Inputs:     req.Inputs,
		Context:    req.Context,
		StartedAt:  time.Now(),
		Status:     workflow.StatusRunning,
	}
	
	// Track execution
	s.mu.Lock()
	s.executions[execID] = &ExecutionState{
		ID:           execID,
		WorkflowName: req.WorkflowName,
		Status:       "running",
		StartedAt:    execCtx.StartedAt,
	}
	s.mu.Unlock()
	
	// Execute workflow with pillar integration
	result, err := s.workflowEngine.ExecuteWorkflow(ctx, workflowDef, execCtx, workflow.ExecutionOptions{
		LLMService: s.llmService,
		RAGService: s.ragService,
		MCPService: s.mcpService,
		Reasoning:  s.reasoning,
	})
	
	// Update execution state
	s.mu.Lock()
	if exec, ok := s.executions[execID]; ok {
		exec.CompletedAt = time.Now()
		if err != nil {
			exec.Status = "failed"
			exec.Error = err.Error()
		} else {
			exec.Status = "completed"
		}
	}
	s.mu.Unlock()
	
	if err != nil {
		return nil, fmt.Errorf("workflow execution failed: %w", err)
	}
	
	// Convert to response
	return s.convertWorkflowResult(result), nil
}

// ListWorkflows lists all available workflows.
func (s *Service) ListWorkflows() []core.WorkflowInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	workflows := make([]core.WorkflowInfo, 0, len(s.workflows))
	for name, wf := range s.workflows {
		workflows = append(workflows, core.WorkflowInfo{
			Name:        name,
			Description: wf.Description,
			StepsCount:  len(wf.Steps),
			CreatedAt:   wf.CreatedAt,
			UpdatedAt:   wf.UpdatedAt,
		})
	}
	
	return workflows
}

// DeleteWorkflow deletes a workflow definition.
func (s *Service) DeleteWorkflow(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	workflowDef, exists := s.workflows[name]
	if !exists {
		return fmt.Errorf("workflow %s not found", name)
	}
	
	// Unregister from workflow engine
	if err := s.workflowEngine.UnregisterWorkflow(workflowDef.ID); err != nil {
		return fmt.Errorf("failed to unregister workflow: %w", err)
	}
	
	// Remove from storage
	delete(s.workflows, name)
	
	return nil
}

// ===== AGENT MANAGEMENT =====

// CreateAgent creates a new agent definition.
func (s *Service) CreateAgent(definition core.AgentDefinition) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Validate agent definition
	if err := s.validateAgentDefinition(definition); err != nil {
		return fmt.Errorf("invalid agent definition: %w", err)
	}
	
	// Check for duplicate name
	if _, exists := s.agents[definition.Name]; exists {
		return fmt.Errorf("agent %s already exists", definition.Name)
	}
	
	// Create agent based on type
	agent, err := s.createAgentByType(definition)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}
	
	// Store agent
	s.agents[definition.Name] = agent
	
	return nil
}

// ExecuteAgent executes an agent.
func (s *Service) ExecuteAgent(ctx context.Context, req core.AgentRequest) (*core.AgentResponse, error) {
	s.mu.RLock()
	agent, exists := s.agents[req.AgentName]
	s.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("agent %s not found", req.AgentName)
	}
	
	// Create execution context
	execCtx := &AgentExecutionContext{
		ID:        generateID(),
		AgentName: req.AgentName,
		Task:      req.Task,
		Context:   req.Context,
		MaxSteps:  req.MaxSteps,
		StartedAt: time.Now(),
		LLMService: s.llmService,
		RAGService: s.ragService,
		MCPService: s.mcpService,
		Reasoning:  s.reasoning,
	}
	
	if execCtx.MaxSteps == 0 {
		execCtx.MaxSteps = 10 // default max steps
	}
	
	// Execute agent with multi-step reasoning
	result, err := agent.Execute(ctx, execCtx)
	if err != nil {
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}
	
	// Convert to response
	return &core.AgentResponse{
		AgentName:   req.AgentName,
		Status:      result.Status,
		Result:      result.Result,
		Steps:       s.convertAgentSteps(result.Steps),
		Duration:    result.Duration,
		StartedAt:   result.StartedAt,
		CompletedAt: result.CompletedAt,
	}, nil
}

// ListAgents lists all available agents.
func (s *Service) ListAgents() []core.AgentInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	agents := make([]core.AgentInfo, 0, len(s.agents))
	for name, agent := range s.agents {
		agents = append(agents, core.AgentInfo{
			Name:        name,
			Type:        agent.Type,
			Description: agent.Description,
			ToolsCount:  len(agent.Tools),
			CreatedAt:   agent.CreatedAt,
			UpdatedAt:   agent.UpdatedAt,
		})
	}
	
	return agents
}

// DeleteAgent deletes an agent definition.
func (s *Service) DeleteAgent(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.agents[name]; !exists {
		return fmt.Errorf("agent %s not found", name)
	}
	
	// Remove from storage
	delete(s.agents, name)
	
	return nil
}

// ===== SCHEDULING =====

// ScheduleWorkflow schedules a workflow for execution.
func (s *Service) ScheduleWorkflow(name string, schedule core.ScheduleConfig) error {
	s.mu.RLock()
	workflowDef, exists := s.workflows[name]
	s.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("workflow %s not found", name)
	}
	
	// Create scheduled job
	job := &scheduler.Job{
		ID:           generateID(),
		Name:         fmt.Sprintf("workflow-%s", name),
		Type:         scheduler.JobTypeWorkflow,
		Schedule:     s.convertScheduleConfig(schedule),
		Payload:      map[string]interface{}{"workflow_id": workflowDef.ID},
		Handler:      s.createWorkflowHandler(name),
	}
	
	// Register with scheduler
	if err := s.scheduler.ScheduleJob(job); err != nil {
		return fmt.Errorf("failed to schedule workflow: %w", err)
	}
	
	return nil
}

// GetScheduledTasks gets all scheduled tasks.
func (s *Service) GetScheduledTasks() []core.ScheduledTask {
	jobs := s.scheduler.ListJobs()
	
	tasks := make([]core.ScheduledTask, 0, len(jobs))
	for _, job := range jobs {
		// Extract workflow name from job name
		workflowName := ""
		if job.Type == scheduler.JobTypeWorkflow {
			if wfID, ok := job.Payload["workflow_id"].(string); ok {
				for name, wf := range s.workflows {
					if wf.ID == wfID {
						workflowName = name
						break
					}
				}
			}
		}
		
		tasks = append(tasks, core.ScheduledTask{
			ID:           job.ID,
			WorkflowName: workflowName,
			Schedule:     s.convertSchedulerConfig(job.Schedule),
			NextRun:      job.NextRun,
			LastRun:      job.LastRun,
			Status:       string(job.Status),
		})
	}
	
	return tasks
}

// Close closes the Agent service and cleans up resources.
func (s *Service) Close() error {
	// Signal shutdown
	s.cancel()
	
	// Stop scheduler
	if s.scheduler != nil {
		if err := s.scheduler.Stop(); err != nil {
			return fmt.Errorf("failed to stop scheduler: %w", err)
		}
	}
	
	// Stop workflow engine
	if s.workflowEngine != nil {
		if err := s.workflowEngine.Stop(); err != nil {
			return fmt.Errorf("failed to stop workflow engine: %w", err)
		}
	}
	
	// Stop reasoning engine
	if s.reasoning != nil {
		if err := s.reasoning.Close(); err != nil {
			return fmt.Errorf("failed to close reasoning engine: %w", err)
		}
	}
	
	// Wait for background tasks
	s.wg.Wait()
	
	return nil
}

// ===== HELPER TYPES AND FUNCTIONS =====

// ServiceOption configures the Agent service.
type ServiceOption func(*Service) error

// WithLLMService sets the LLM service for agent operations.
func WithLLMService(llmService llm.Service) ServiceOption {
	return func(s *Service) error {
		s.llmService = llmService
		return nil
	}
}

// WithRAGService sets the RAG service for agent operations.
func WithRAGService(ragService rag.Service) ServiceOption {
	return func(s *Service) error {
		s.ragService = ragService
		return nil
	}
}

// WithMCPService sets the MCP service for agent operations.
func WithMCPService(mcpService mcp.Service) ServiceOption {
	return func(s *Service) error {
		s.mcpService = mcpService
		return nil
	}
}

// Agent represents an agent instance.
type Agent struct {
	ID          string
	Name        string
	Type        string
	Description string
	Instructions string
	Tools       []string
	Parameters  map[string]interface{}
	CreatedAt   time.Time
	UpdatedAt   time.Time
	executor    AgentExecutor
}

// Execute executes the agent with the given context.
func (a *Agent) Execute(ctx context.Context, execCtx *AgentExecutionContext) (*AgentExecutionResult, error) {
	if a.executor == nil {
		return nil, fmt.Errorf("agent executor not initialized")
	}
	return a.executor.Execute(ctx, execCtx)
}

// AgentExecutor defines the interface for agent execution.
type AgentExecutor interface {
	Execute(ctx context.Context, execCtx *AgentExecutionContext) (*AgentExecutionResult, error)
}

// AgentExecutionContext provides context for agent execution.
type AgentExecutionContext struct {
	ID         string
	AgentName  string
	Task       string
	Context    map[string]interface{}
	MaxSteps   int
	StartedAt  time.Time
	LLMService interface{}
	RAGService interface{}
	MCPService interface{}
	Reasoning  interface{}
}

// AgentExecutionResult holds the result of agent execution.
type AgentExecutionResult struct {
	Status      string
	Result      string
	Steps       []AgentExecutionStep
	Duration    time.Duration
	StartedAt   time.Time
	CompletedAt time.Time
}

// AgentExecutionStep represents a single step in agent execution.
type AgentExecutionStep struct {
	StepNumber  int
	Action      string
	Input       interface{}
	Output      interface{}
	Duration    time.Duration
}

// ExecutionState tracks the state of a workflow or agent execution.
type ExecutionState struct {
	ID           string
	WorkflowName string
	Status       string
	StartedAt    time.Time
	CompletedAt  time.Time
	Error        string
}

// Helper functions

func (s *Service) runMaintenanceLoop() {
	defer s.wg.Done()
	
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.cleanupCompletedExecutions()
		}
	}
}

func (s *Service) cleanupCompletedExecutions() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	cutoff := time.Now().Add(-1 * time.Hour)
	for id, exec := range s.executions {
		if exec.CompletedAt.Before(cutoff) {
			delete(s.executions, id)
		}
	}
}

func (s *Service) validateWorkflowDefinition(def core.WorkflowDefinition) error {
	if def.Name == "" {
		return fmt.Errorf("workflow name is required")
	}
	if len(def.Steps) == 0 {
		return fmt.Errorf("workflow must have at least one step")
	}
	// Check for circular dependencies
	if err := s.checkCircularDependencies(def.Steps); err != nil {
		return err
	}
	return nil
}

func (s *Service) checkCircularDependencies(steps []core.WorkflowStep) error {
	// Build dependency graph
	deps := make(map[string][]string)
	for _, step := range steps {
		deps[step.ID] = step.Dependencies
	}
	
	// Check for cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	
	var hasCycle func(node string) bool
	hasCycle = func(node string) bool {
		visited[node] = true
		recStack[node] = true
		
		for _, dep := range deps[node] {
			if !visited[dep] {
				if hasCycle(dep) {
					return true
				}
			} else if recStack[dep] {
				return true
			}
		}
		
		recStack[node] = false
		return false
	}
	
	for _, step := range steps {
		if !visited[step.ID] {
			if hasCycle(step.ID) {
				return fmt.Errorf("circular dependency detected in workflow")
			}
		}
	}
	
	return nil
}

func (s *Service) validateAgentDefinition(def core.AgentDefinition) error {
	if def.Name == "" {
		return fmt.Errorf("agent name is required")
	}
	if def.Type == "" {
		return fmt.Errorf("agent type is required")
	}
	// Validate agent type
	validTypes := []string{"research", "workflow", "monitoring", "custom"}
	valid := false
	for _, t := range validTypes {
		if def.Type == t {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid agent type: %s", def.Type)
	}
	return nil
}

func (s *Service) convertWorkflowSteps(steps []core.WorkflowStep) []workflow.Step {
	result := make([]workflow.Step, len(steps))
	for i, step := range steps {
		result[i] = workflow.Step{
			ID:           step.ID,
			Name:         step.Name,
			Type:         step.Type,
			Parameters:   step.Parameters,
			Dependencies: step.Dependencies,
			Condition:    step.Condition,
		}
	}
	return result
}

func (s *Service) convertWorkflowResult(result *workflow.ExecutionResult) *core.WorkflowResponse {
	stepResults := make([]core.StepResult, len(result.Steps))
	for i, step := range result.Steps {
		stepResults[i] = core.StepResult{
			StepID:   step.ID,
			Status:   step.Status,
			Output:   step.Output,
			Error:    step.Error,
			Duration: step.Duration,
		}
	}
	
	return &core.WorkflowResponse{
		WorkflowName: result.WorkflowName,
		Status:       result.Status,
		Outputs:      result.Outputs,
		Steps:        stepResults,
		Duration:     result.Duration,
		StartedAt:    result.StartedAt,
		CompletedAt:  result.CompletedAt,
	}
}

func (s *Service) convertAgentSteps(steps []AgentExecutionStep) []core.AgentStep {
	result := make([]core.AgentStep, len(steps))
	for i, step := range steps {
		result[i] = core.AgentStep{
			StepNumber: step.StepNumber,
			Action:     step.Action,
			Input:      step.Input,
			Output:     step.Output,
			Duration:   step.Duration,
		}
	}
	return result
}

func (s *Service) createAgentByType(def core.AgentDefinition) (*Agent, error) {
	agent := &Agent{
		ID:           generateID(),
		Name:         def.Name,
		Type:         def.Type,
		Description:  def.Description,
		Instructions: def.Instructions,
		Tools:        def.Tools,
		Parameters:   def.Parameters,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	// Create executor based on type
	switch def.Type {
	case "research":
		agent.executor = NewResearchAgentExecutor(agent, s.llmService, s.ragService)
	case "workflow":
		agent.executor = NewWorkflowAgentExecutor(agent, s.workflowEngine)
	case "monitoring":
		agent.executor = NewMonitoringAgentExecutor(agent, s.mcpService)
	case "custom":
		agent.executor = NewCustomAgentExecutor(agent, s.llmService, s.ragService, s.mcpService)
	default:
		return nil, fmt.Errorf("unsupported agent type: %s", def.Type)
	}
	
	return agent, nil
}

func (s *Service) createWorkflowHandler(name string) scheduler.JobHandler {
	return func(ctx context.Context, job *scheduler.Job) error {
		// Execute workflow
		result, err := s.ExecuteWorkflow(ctx, core.WorkflowRequest{
			WorkflowName: name,
			Inputs:       job.Payload,
		})
		
		if err != nil {
			return fmt.Errorf("scheduled workflow execution failed: %w", err)
		}
		
		// Log result
		fmt.Printf("Scheduled workflow %s completed with status: %s\n", name, result.Status)
		return nil
	}
}

func (s *Service) convertScheduleConfig(config core.ScheduleConfig) scheduler.Schedule {
	return scheduler.Schedule{
		Type:       scheduler.ScheduleType(config.Type),
		Expression: config.Expression,
		Timezone:   config.Timezone,
		Metadata:   config.Metadata,
	}
}

func (s *Service) convertSchedulerConfig(schedule scheduler.Schedule) core.ScheduleConfig {
	return core.ScheduleConfig{
		Type:       string(schedule.Type),
		Expression: schedule.Expression,
		Timezone:   schedule.Timezone,
		Metadata:   schedule.Metadata,
	}
}

func generateID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Int63())
}

// Import for rand
var rand = randPkg.New(randPkg.NewSource(time.Now().UnixNano()))