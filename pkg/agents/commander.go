package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
)

// Commander orchestrates multiple agents working in parallel
type Commander struct {
	config          *config.Config
	llm             domain.Generator
	mcpManager      *mcp.Manager
	agentPool       *AgentPool
	missions        map[string]*Mission
	missionMutex    sync.RWMutex
	progressStorage *ProgressStorage
	verbose         bool
}

// Mission represents a complex task that requires multiple agents
type Mission struct {
	ID          string                 `json:"id"`
	Goal        string                 `json:"goal"`
	Strategy    *MissionStrategy       `json:"strategy"`
	Agents      map[string]*AgentTask  `json:"agents"`
	Status      MissionStatus          `json:"status"`
	Results     map[string]interface{} `json:"results"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     *time.Time             `json:"end_time,omitempty"`
	Errors      []string               `json:"errors,omitempty"`
}

// MissionStrategy defines how to approach a complex task
type MissionStrategy struct {
	Type           StrategyType           `json:"type"`
	Decomposition  []TaskDecomposition    `json:"decomposition"`
	Dependencies   map[string][]string    `json:"dependencies"`
	MaxParallel    int                    `json:"max_parallel"`
	TimeoutMinutes int                    `json:"timeout_minutes"`
}

// StrategyType defines the execution strategy
type StrategyType string

const (
	StrategyParallel   StrategyType = "parallel"
	StrategySequential StrategyType = "sequential"
	StrategyPipeline   StrategyType = "pipeline"
	StrategyMapReduce  StrategyType = "map_reduce"
)

// TaskDecomposition represents a sub-task
type TaskDecomposition struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description"`
	Type        TaskType               `json:"type"`
	Input       map[string]interface{} `json:"input"`
	Priority    int                    `json:"priority"`
}

// TaskType defines the type of task
type TaskType string

const (
	TaskTypeResearch   TaskType = "research"
	TaskTypeAnalysis   TaskType = "analysis"
	TaskTypeExecution  TaskType = "execution"
	TaskTypeSynthesis  TaskType = "synthesis"
	TaskTypeValidation TaskType = "validation"
)

// AgentTask represents work assigned to a specific agent
type AgentTask struct {
	AgentID     string                 `json:"agent_id"`
	TaskID      string                 `json:"task_id"`
	Description string                 `json:"description"`
	Status      AgentStatus            `json:"status"`
	Result      interface{}            `json:"result,omitempty"`
	Error       error                  `json:"error,omitempty"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     *time.Time             `json:"end_time,omitempty"`
	Metrics     *TaskMetrics           `json:"metrics,omitempty"`
}

// TaskMetrics tracks performance metrics
type TaskMetrics struct {
	TokensUsed     int           `json:"tokens_used"`
	ToolCalls      int           `json:"tool_calls"`
	ExecutionTime  time.Duration `json:"execution_time"`
	RetryCount     int           `json:"retry_count"`
}

// AgentStatus represents the status of an agent's task
type AgentStatus string

const (
	AgentStatusIdle       AgentStatus = "idle"
	AgentStatusAssigned   AgentStatus = "assigned"
	AgentStatusWorking    AgentStatus = "working"
	AgentStatusCompleted  AgentStatus = "completed"
	AgentStatusFailed     AgentStatus = "failed"
	AgentStatusRetrying   AgentStatus = "retrying"
)

// MissionStatus represents the overall mission status
type MissionStatus string

const (
	MissionStatusPlanning   MissionStatus = "planning"
	MissionStatusExecuting  MissionStatus = "executing"
	MissionStatusCompleted  MissionStatus = "completed"
	MissionStatusFailed     MissionStatus = "failed"
	MissionStatusCancelled  MissionStatus = "cancelled"
)

// NewCommander creates a new multi-agent commander
func NewCommander(cfg *config.Config, llm domain.Generator, mcpManager *mcp.Manager) *Commander {
	// Get configured paths with defaults
	dataPath := ".rago/data"
	if cfg.Agents != nil && cfg.Agents.DataPath != "" {
		dataPath = cfg.Agents.DataPath
	}
	
	// Initialize progress storage
	progressDBPath := filepath.Join(dataPath, "progress.db")
	progressStorage, err := NewProgressStorage(progressDBPath)
	if err != nil {
		// Log warning but continue without progress tracking
		fmt.Printf("Warning: Failed to initialize progress storage: %v\n", err)
	}
	
	return &Commander{
		config:          cfg,
		llm:             llm,
		mcpManager:      mcpManager,
		agentPool:       NewAgentPool(cfg, llm, mcpManager),
		missions:        make(map[string]*Mission),
		progressStorage: progressStorage,
		verbose:         false,
	}
}

// SetVerbose enables verbose output
func (c *Commander) SetVerbose(v bool) {
	c.verbose = v
	c.agentPool.SetVerbose(v)
}

// ExecuteMission plans and executes a complex mission using multiple agents
func (c *Commander) ExecuteMission(ctx context.Context, goal string) (*Mission, error) {
	// Create mission
	mission := &Mission{
		ID:        uuid.New().String(),
		Goal:      goal,
		Status:    MissionStatusPlanning,
		Agents:    make(map[string]*AgentTask),
		Results:   make(map[string]interface{}),
		StartTime: time.Now(),
	}

	// Store mission
	c.missionMutex.Lock()
	c.missions[mission.ID] = mission
	c.missionMutex.Unlock()
	
	// Save initial progress to database
	if c.progressStorage != nil {
		if err := c.progressStorage.SaveMissionProgress(mission); err != nil {
			if c.verbose {
				fmt.Printf("⚠️  Failed to save mission progress: %v\n", err)
			}
		}
	}

	if c.verbose {
		fmt.Printf("🎯 Mission %s: %s\n", mission.ID[:8], goal)
	}

	// Step 1: Analyze and create strategy
	strategy, err := c.createStrategy(ctx, goal)
	if err != nil {
		mission.Status = MissionStatusFailed
		mission.Errors = append(mission.Errors, err.Error())
		return mission, fmt.Errorf("strategy creation failed: %w", err)
	}
	mission.Strategy = strategy

	if c.verbose {
		fmt.Printf("📋 Strategy: %s with %d tasks\n", strategy.Type, len(strategy.Decomposition))
	}

	// Step 2: Execute based on strategy
	mission.Status = MissionStatusExecuting
	
	switch strategy.Type {
	case StrategyParallel:
		err = c.executeParallel(ctx, mission)
	case StrategySequential:
		err = c.executeSequential(ctx, mission)
	case StrategyPipeline:
		err = c.executePipeline(ctx, mission)
	case StrategyMapReduce:
		err = c.executeMapReduce(ctx, mission)
	default:
		err = c.executeParallel(ctx, mission) // Default to parallel
	}

	// Update mission status
	now := time.Now()
	mission.EndTime = &now
	
	if err != nil {
		mission.Status = MissionStatusFailed
		mission.Errors = append(mission.Errors, err.Error())
		return mission, err
	}

	mission.Status = MissionStatusCompleted
	
	if c.verbose {
		fmt.Printf("✅ Mission %s completed in %v\n", mission.ID[:8], mission.EndTime.Sub(mission.StartTime))
	}

	return mission, nil
}

// createStrategy analyzes the goal and creates an execution strategy
func (c *Commander) createStrategy(ctx context.Context, goal string) (*MissionStrategy, error) {
	prompt := fmt.Sprintf(`Analyze this goal and create a multi-agent execution strategy.

GOAL: %s

Create a strategy that decomposes the goal into sub-tasks that can be executed by specialized agents.

Return JSON in this format:
{
  "type": "parallel|sequential|pipeline|map_reduce",
  "decomposition": [
    {
      "id": "task_1",
      "description": "specific task description",
      "type": "research|analysis|execution|synthesis|validation",
      "input": {},
      "priority": 1
    }
  ],
  "dependencies": {
    "task_2": ["task_1"]
  },
  "max_parallel": 3,
  "timeout_minutes": 30
}

Choose strategy type based on:
- parallel: Independent tasks that can run simultaneously
- sequential: Tasks that must run one after another
- pipeline: Data flows from one task to the next
- map_reduce: Split work, process in parallel, then combine results`, goal)

	opts := &domain.GenerationOptions{
		Temperature: 0.3,
		MaxTokens:   2000,
	}

	response, err := c.llm.Generate(ctx, prompt, opts)
	if err != nil {
		return nil, err
	}

	// Parse strategy
	var strategy MissionStrategy
	jsonStr := extractJSON(response)
	if err := json.Unmarshal([]byte(jsonStr), &strategy); err != nil {
		return nil, fmt.Errorf("failed to parse strategy: %w", err)
	}

	// Set defaults
	if strategy.MaxParallel == 0 {
		strategy.MaxParallel = 3
	}
	if strategy.TimeoutMinutes == 0 {
		strategy.TimeoutMinutes = 30
	}

	return &strategy, nil
}

// executeParallel runs tasks in parallel with concurrency limit
func (c *Commander) executeParallel(ctx context.Context, mission *Mission) error {
	tasks := mission.Strategy.Decomposition
	maxParallel := mission.Strategy.MaxParallel
	
	// Create context with timeout
	timeout := time.Duration(mission.Strategy.TimeoutMinutes) * time.Minute
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create work channel and result channel
	workCh := make(chan TaskDecomposition, len(tasks))
	resultCh := make(chan *AgentTask, len(tasks))
	
	// Create worker pool
	var wg sync.WaitGroup
	for i := 0; i < maxParallel; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			c.worker(ctx, workerID, mission, workCh, resultCh)
		}(i)
	}

	// Queue all tasks
	for _, task := range tasks {
		workCh <- task
	}
	close(workCh)

	// Wait for workers to finish
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results
	for agentTask := range resultCh {
		mission.Agents[agentTask.TaskID] = agentTask
		if agentTask.Error == nil {
			mission.Results[agentTask.TaskID] = agentTask.Result
		}
	}

	return nil
}

// worker processes tasks from the work channel
func (c *Commander) worker(ctx context.Context, workerID int, mission *Mission, workCh <-chan TaskDecomposition, resultCh chan<- *AgentTask) {
	for task := range workCh {
		select {
		case <-ctx.Done():
			return
		default:
			if c.verbose {
				fmt.Printf("🔧 Worker %d: Processing task %s\n", workerID, task.ID)
			}

			// Get an agent from the pool
			agent := c.agentPool.GetAgent()
			
			// Create agent task
			agentTask := &AgentTask{
				AgentID:     fmt.Sprintf("agent_%d", workerID),
				TaskID:      task.ID,
				Description: task.Description,
				Status:      AgentStatusWorking,
				StartTime:   time.Now(),
				Metrics:     &TaskMetrics{},
			}
			
			// Save initial step progress
			if c.progressStorage != nil {
				c.progressStorage.UpdateStepProgress(
					mission.ID, task.ID, agentTask.AgentID,
					task.Priority, "working", 0.0, nil,
				)
				c.progressStorage.LogAgentActivity(
					agentTask.AgentID, mission.ID, task.ID,
					"task_started", task.Description, nil,
				)
			}

			// Execute the task with progress tracking
			result, err := c.executeTaskWithProgress(ctx, agent, task, mission.ID)
			
			// Update task status
			now := time.Now()
			agentTask.EndTime = &now
			agentTask.Metrics.ExecutionTime = now.Sub(agentTask.StartTime)
			
			if err != nil {
				agentTask.Status = AgentStatusFailed
				agentTask.Error = err
				if c.verbose {
					fmt.Printf("❌ Task %s failed: %v\n", task.ID, err)
				}
				// Save failure to progress
				if c.progressStorage != nil {
					c.progressStorage.SaveStepResult(
						mission.ID, task.ID, task.Priority,
						nil, agentTask.Metrics, err,
					)
				}
			} else {
				agentTask.Status = AgentStatusCompleted
				agentTask.Result = result
				if c.verbose {
					fmt.Printf("✅ Task %s completed\n", task.ID)
				}
				// Save success to progress
				if c.progressStorage != nil {
					c.progressStorage.SaveStepResult(
						mission.ID, task.ID, task.Priority,
						result, agentTask.Metrics, nil,
					)
				}
			}

			// Return agent to pool
			c.agentPool.ReturnAgent(agent)
			
			// Send result
			resultCh <- agentTask
		}
	}
}

// executeTaskWithProgress executes a task with progress tracking
func (c *Commander) executeTaskWithProgress(ctx context.Context, agent *Agent, task TaskDecomposition, missionID string) (interface{}, error) {
	// Update progress at different stages
	if c.progressStorage != nil {
		c.progressStorage.UpdateStepProgress(
			missionID, task.ID, "agent",
			task.Priority, "executing", 25.0, nil,
		)
	}
	
	result, err := c.executeTask(ctx, agent, task)
	
	if err == nil && c.progressStorage != nil {
		c.progressStorage.UpdateStepProgress(
			missionID, task.ID, "agent",
			task.Priority, "completed", 100.0, result,
		)
	}
	
	return result, err
}

// executeTask executes a single task using an agent
func (c *Commander) executeTask(ctx context.Context, agent *Agent, task TaskDecomposition) (interface{}, error) {
	// Build task prompt based on type
	var prompt string
	switch task.Type {
	case TaskTypeResearch:
		prompt = fmt.Sprintf("Research: %s", task.Description)
	case TaskTypeAnalysis:
		prompt = fmt.Sprintf("Analyze: %s", task.Description)
	case TaskTypeExecution:
		prompt = fmt.Sprintf("Execute: %s", task.Description)
	case TaskTypeSynthesis:
		prompt = fmt.Sprintf("Synthesize: %s", task.Description)
	case TaskTypeValidation:
		prompt = fmt.Sprintf("Validate: %s", task.Description)
	default:
		prompt = task.Description
	}

	// Add input context if available
	if len(task.Input) > 0 {
		inputJSON, _ := json.MarshalIndent(task.Input, "", "  ")
		prompt = fmt.Sprintf("%s\n\nInput data:\n%s", prompt, string(inputJSON))
	}

	// Execute using agent's Do method for intelligent execution
	result, err := agent.Do(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Return appropriate result based on what was executed
	if result.DirectAnswer != "" {
		return result.DirectAnswer, nil
	}
	if result.FinalAnswer != "" {
		return result.FinalAnswer, nil
	}
	if result.ExecutionResults != nil {
		return result.ExecutionResults, nil
	}

	return result, nil
}

// executeSequential runs tasks one after another
func (c *Commander) executeSequential(ctx context.Context, mission *Mission) error {
	tasks := mission.Strategy.Decomposition
	agent := c.agentPool.GetAgent()
	defer c.agentPool.ReturnAgent(agent)

	for _, task := range tasks {
		if c.verbose {
			fmt.Printf("➡️  Executing task %s: %s\n", task.ID, task.Description)
		}

		// Add previous results as input
		if len(mission.Results) > 0 {
			task.Input["previous_results"] = mission.Results
		}

		result, err := c.executeTaskWithProgress(ctx, agent, task, mission.ID)
		if err != nil {
			return fmt.Errorf("task %s failed: %w", task.ID, err)
		}

		mission.Results[task.ID] = result
	}

	return nil
}

// executePipeline runs tasks in a pipeline fashion
func (c *Commander) executePipeline(ctx context.Context, mission *Mission) error {
	tasks := mission.Strategy.Decomposition
	var pipelineData interface{}

	for i, task := range tasks {
		if c.verbose {
			fmt.Printf("🔀 Pipeline stage %d/%d: %s\n", i+1, len(tasks), task.Description)
		}

		// Add pipeline data from previous stage
		if pipelineData != nil {
			task.Input["pipeline_input"] = pipelineData
		}

		agent := c.agentPool.GetAgent()
		result, err := c.executeTaskWithProgress(ctx, agent, task, mission.ID)
		c.agentPool.ReturnAgent(agent)

		if err != nil {
			return fmt.Errorf("pipeline stage %s failed: %w", task.ID, err)
		}

		pipelineData = result
		mission.Results[task.ID] = result
	}

	mission.Results["final_output"] = pipelineData
	return nil
}

// executeMapReduce splits work, processes in parallel, then combines
func (c *Commander) executeMapReduce(ctx context.Context, mission *Mission) error {
	tasks := mission.Strategy.Decomposition
	
	// Separate map and reduce tasks
	var mapTasks []TaskDecomposition
	var reduceTask *TaskDecomposition
	
	for _, task := range tasks {
		if task.Type == TaskTypeSynthesis {
			reduceTask = &task
		} else {
			mapTasks = append(mapTasks, task)
		}
	}

	// Map phase - parallel execution
	mapResults := make(map[string]interface{})
	resultCh := make(chan struct {
		ID     string
		Result interface{}
		Error  error
	}, len(mapTasks))

	var wg sync.WaitGroup
	for _, task := range mapTasks {
		wg.Add(1)
		go func(t TaskDecomposition) {
			defer wg.Done()
			
			agent := c.agentPool.GetAgent()
			defer c.agentPool.ReturnAgent(agent)
			
			result, err := c.executeTaskWithProgress(ctx, agent, t, mission.ID)
			resultCh <- struct {
				ID     string
				Result interface{}
				Error  error
			}{t.ID, result, err}
		}(task)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect map results
	for res := range resultCh {
		if res.Error != nil {
			return fmt.Errorf("map task %s failed: %w", res.ID, res.Error)
		}
		mapResults[res.ID] = res.Result
	}

	// Reduce phase
	if reduceTask != nil {
		if c.verbose {
			fmt.Printf("🔄 Reducing %d results\n", len(mapResults))
		}

		reduceTask.Input["map_results"] = mapResults
		
		agent := c.agentPool.GetAgent()
		finalResult, err := c.executeTaskWithProgress(ctx, agent, *reduceTask, mission.ID)
		c.agentPool.ReturnAgent(agent)

		if err != nil {
			return fmt.Errorf("reduce phase failed: %w", err)
		}

		mission.Results["final"] = finalResult
	}

	mission.Results["map_results"] = mapResults
	return nil
}

// GetMission returns information about a specific mission
func (c *Commander) GetMission(missionID string) (*Mission, error) {
	c.missionMutex.RLock()
	defer c.missionMutex.RUnlock()

	mission, exists := c.missions[missionID]
	if !exists {
		return nil, fmt.Errorf("mission %s not found", missionID)
	}

	return mission, nil
}

// ListMissions returns all missions
func (c *Commander) ListMissions() []*Mission {
	c.missionMutex.RLock()
	defer c.missionMutex.RUnlock()

	missions := make([]*Mission, 0, len(c.missions))
	for _, mission := range c.missions {
		missions = append(missions, mission)
	}

	return missions
}

// CancelMission cancels an ongoing mission
func (c *Commander) CancelMission(missionID string) error {
	c.missionMutex.Lock()
	defer c.missionMutex.Unlock()

	mission, exists := c.missions[missionID]
	if !exists {
		return fmt.Errorf("mission %s not found", missionID)
	}

	if mission.Status == MissionStatusCompleted || mission.Status == MissionStatusFailed {
		return fmt.Errorf("mission %s already finished", missionID)
	}

	mission.Status = MissionStatusCancelled
	now := time.Now()
	mission.EndTime = &now

	return nil
}

// GetMissionProgress retrieves detailed progress for a mission from database
func (c *Commander) GetMissionProgress(missionID string) (*MissionProgress, error) {
	if c.progressStorage == nil {
		return nil, fmt.Errorf("progress storage not available")
	}
	return c.progressStorage.GetMissionProgress(missionID)
}

// ResumeMission resumes a previously interrupted mission
func (c *Commander) ResumeMission(ctx context.Context, missionID string) (*Mission, error) {
	// Check if mission exists in memory
	c.missionMutex.RLock()
	mission, exists := c.missions[missionID]
	c.missionMutex.RUnlock()
	
	if !exists && c.progressStorage != nil {
		// Try to load from database
		progress, err := c.progressStorage.GetMissionProgress(missionID)
		if err != nil {
			return nil, fmt.Errorf("mission %s not found", missionID)
		}
		
		// Reconstruct mission from progress
		mission = &Mission{
			ID:     progress.MissionID,
			Goal:   progress.Goal,
			Status: MissionStatus(progress.Status),
			Results: make(map[string]interface{}),
			StartTime: progress.StartedAt,
		}
		
		// Get latest checkpoint
		checkpoint, _ := c.progressStorage.GetLatestCheckpoint(missionID)
		if checkpoint != nil {
			// Restore mission state from checkpoint
			if strategyData, ok := checkpoint["strategy"]; ok {
				if strategyJSON, err := json.Marshal(strategyData); err == nil {
					json.Unmarshal(strategyJSON, &mission.Strategy)
				}
			}
			if resultsData, ok := checkpoint["results"]; ok {
				if resultsMap, ok := resultsData.(map[string]interface{}); ok {
					mission.Results = resultsMap
				}
			}
		}
		
		// Add to memory cache
		c.missionMutex.Lock()
		c.missions[missionID] = mission
		c.missionMutex.Unlock()
	}
	
	if mission == nil {
		return nil, fmt.Errorf("mission %s not found", missionID)
	}
	
	// Check if mission is already completed
	if mission.Status == MissionStatusCompleted || mission.Status == MissionStatusFailed {
		return mission, fmt.Errorf("mission %s already finished with status: %s", missionID, mission.Status)
	}
	
	if c.verbose {
		fmt.Printf("📂 Resuming mission %s from checkpoint\n", missionID[:8])
	}
	
	// Find incomplete tasks from progress storage
	if c.progressStorage != nil {
		progress, err := c.progressStorage.GetMissionProgress(missionID)
		if err == nil {
			// Resume execution based on strategy type
			mission.Status = MissionStatusExecuting
			
			// Create a list of incomplete tasks
			incompleteTasks := []TaskDecomposition{}
			for _, step := range progress.Steps {
				if step.Status != "completed" {
					// Find the corresponding task in strategy
					for _, task := range mission.Strategy.Decomposition {
						if task.ID == step.TaskID {
							incompleteTasks = append(incompleteTasks, task)
							break
						}
					}
				}
			}
			
			if len(incompleteTasks) > 0 {
				// Resume execution with incomplete tasks
				tempStrategy := *mission.Strategy
				tempStrategy.Decomposition = incompleteTasks
				mission.Strategy = &tempStrategy
				
				// Continue execution based on strategy type
				var err error
				switch mission.Strategy.Type {
				case StrategyParallel:
					err = c.executeParallel(ctx, mission)
				case StrategySequential:
					err = c.executeSequential(ctx, mission)
				case StrategyPipeline:
					err = c.executePipeline(ctx, mission)
				case StrategyMapReduce:
					err = c.executeMapReduce(ctx, mission)
				default:
					err = c.executeParallel(ctx, mission)
				}
				
				if err != nil {
					mission.Status = MissionStatusFailed
					mission.Errors = append(mission.Errors, err.Error())
					return mission, err
				}
			}
			
			mission.Status = MissionStatusCompleted
		}
	}
	
	now := time.Now()
	mission.EndTime = &now
	
	if c.verbose {
		fmt.Printf("✅ Mission %s resumed and completed\n", missionID[:8])
	}
	
	return mission, nil
}

// SaveCheckpoint saves a checkpoint for mission resumability
func (c *Commander) SaveCheckpoint(missionID string) error {
	c.missionMutex.RLock()
	mission, exists := c.missions[missionID]
	c.missionMutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("mission %s not found", missionID)
	}
	
	if c.progressStorage == nil {
		return fmt.Errorf("progress storage not available")
	}
	
	checkpointData := map[string]interface{}{
		"strategy": mission.Strategy,
		"results":  mission.Results,
		"agents":   mission.Agents,
		"status":   mission.Status,
	}
	
	return c.progressStorage.SaveCheckpoint(missionID, "manual_checkpoint", checkpointData)
}

// GetRecentProgressEvents retrieves recent progress events for a mission
func (c *Commander) GetRecentProgressEvents(missionID string, limit int) ([]ProgressEvent, error) {
	if c.progressStorage == nil {
		return nil, fmt.Errorf("progress storage not available")
	}
	return c.progressStorage.GetRecentEvents(missionID, limit)
}

// GetPerformanceMetrics retrieves performance metrics for a mission
func (c *Commander) GetPerformanceMetrics(missionID string) (map[string][]MetricPoint, error) {
	if c.progressStorage == nil {
		return nil, fmt.Errorf("progress storage not available")
	}
	return c.progressStorage.GetMetrics(missionID)
}

// GetMetrics returns performance metrics for all missions
func (c *Commander) GetMetrics() map[string]interface{} {
	c.missionMutex.RLock()
	defer c.missionMutex.RUnlock()

	completed := 0
	failed := 0
	executing := 0
	totalTime := time.Duration(0)

	for _, mission := range c.missions {
		switch mission.Status {
		case MissionStatusCompleted:
			completed++
			if mission.EndTime != nil {
				totalTime += mission.EndTime.Sub(mission.StartTime)
			}
		case MissionStatusFailed:
			failed++
		case MissionStatusExecuting:
			executing++
		}
	}

	avgTime := time.Duration(0)
	if completed > 0 {
		avgTime = totalTime / time.Duration(completed)
	}

	return map[string]interface{}{
		"total_missions":     len(c.missions),
		"completed":          completed,
		"failed":             failed,
		"executing":          executing,
		"success_rate":       float64(completed) / float64(completed+failed),
		"average_time":       avgTime.String(),
		"agent_pool_size":    c.agentPool.Size(),
		"agents_available":   c.agentPool.Available(),
	}
}