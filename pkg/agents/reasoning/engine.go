// Package reasoning implements multi-step reasoning with working memory.
package reasoning

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"
	"time"
)

// Engine provides multi-step reasoning capabilities with working memory.
type Engine struct {
	mu     sync.RWMutex
	config Config
	
	// Working memory
	memory *WorkingMemory
	
	// Reasoning chains
	chains map[string]*ReasoningChain
	
	// Learning data
	learningStore *LearningStore
	
	// Goal management
	goals map[string]*Goal
	
	// Execution history
	history []ExecutionRecord
}

// Config holds reasoning engine configuration.
type Config struct {
	MaxDepth        int
	MemoryCapacity  int
	EnableLearning  bool
	PersistencePath string
}

// WorkingMemory manages short-term memory for reasoning.
type WorkingMemory struct {
	mu       sync.RWMutex
	facts    map[string]Fact
	capacity int
	lru      []string
}

// Fact represents a piece of knowledge in working memory.
type Fact struct {
	ID        string
	Type      FactType
	Subject   string
	Predicate string
	Object    interface{}
	Confidence float64
	Source    string
	Timestamp time.Time
	TTL       time.Duration
}

// FactType defines the type of fact.
type FactType string

const (
	FactTypeObservation FactType = "observation"
	FactTypeInference   FactType = "inference"
	FactTypeGoal        FactType = "goal"
	FactTypeConstraint  FactType = "constraint"
)

// ReasoningChain represents a chain of reasoning steps.
type ReasoningChain struct {
	ID          string
	Name        string
	Description string
	Steps       []ReasoningStep
	CreatedAt   time.Time
}

// ReasoningStep represents a single step in reasoning.
type ReasoningStep struct {
	ID          string
	Type        StepType
	Description string
	Action      func(context.Context, *WorkingMemory) (interface{}, error)
	Conditions  []Condition
	Effects     []Effect
}

// StepType defines the type of reasoning step.
type StepType string

const (
	StepTypeObserve    StepType = "observe"
	StepTypeInfer      StepType = "infer"
	StepTypePlan       StepType = "plan"
	StepTypeExecute    StepType = "execute"
	StepTypeEvaluate   StepType = "evaluate"
	StepTypeLearn      StepType = "learn"
)

// Condition represents a condition that must be met.
type Condition struct {
	Type      ConditionType
	Subject   string
	Predicate string
	Value     interface{}
}

// ConditionType defines the type of condition.
type ConditionType string

const (
	ConditionTypeExists   ConditionType = "exists"
	ConditionTypeEquals   ConditionType = "equals"
	ConditionTypeGreater  ConditionType = "greater"
	ConditionTypeLess     ConditionType = "less"
	ConditionTypeContains ConditionType = "contains"
)

// Effect represents an effect of a reasoning step.
type Effect struct {
	Type   EffectType
	Target string
	Value  interface{}
}

// EffectType defines the type of effect.
type EffectType string

const (
	EffectTypeAdd    EffectType = "add"
	EffectTypeRemove EffectType = "remove"
	EffectTypeUpdate EffectType = "update"
)

// Goal represents a reasoning goal.
type Goal struct {
	ID          string
	Description string
	Priority    int
	Status      GoalStatus
	Conditions  []Condition
	Plan        *Plan
	CreatedAt   time.Time
	Deadline    time.Time
}

// GoalStatus represents the status of a goal.
type GoalStatus string

const (
	GoalStatusPending   GoalStatus = "pending"
	GoalStatusActive    GoalStatus = "active"
	GoalStatusAchieved  GoalStatus = "achieved"
	GoalStatusFailed    GoalStatus = "failed"
	GoalStatusAbandoned GoalStatus = "abandoned"
)

// Plan represents a plan to achieve a goal.
type Plan struct {
	ID       string
	GoalID   string
	Steps    []PlanStep
	Status   PlanStatus
	Progress float64
}

// PlanStep represents a step in a plan.
type PlanStep struct {
	ID          string
	Action      string
	Parameters  map[string]interface{}
	Preconditions []Condition
	Effects     []Effect
	Status      StepStatus
}

// PlanStatus represents the status of a plan.
type PlanStatus string

const (
	PlanStatusDraft     PlanStatus = "draft"
	PlanStatusReady     PlanStatus = "ready"
	PlanStatusExecuting PlanStatus = "executing"
	PlanStatusCompleted PlanStatus = "completed"
	PlanStatusFailed    PlanStatus = "failed"
)

// StepStatus represents the status of a plan step.
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
)

// LearningStore manages learned patterns and optimizations.
type LearningStore struct {
	mu       sync.RWMutex
	patterns map[string]*Pattern
	path     string
}

// Pattern represents a learned pattern.
type Pattern struct {
	ID          string
	Type        string
	Trigger     []Condition
	Actions     []string
	SuccessRate float64
	UsageCount  int
	LastUsed    time.Time
}

// ExecutionRecord tracks reasoning execution history.
type ExecutionRecord struct {
	ID        string
	ChainID   string
	StartTime time.Time
	EndTime   time.Time
	Success   bool
	Facts     []Fact
	Result    interface{}
	Error     error
}

// NewEngine creates a new reasoning engine.
func NewEngine(config Config) (*Engine, error) {
	engine := &Engine{
		config: config,
		memory: NewWorkingMemory(config.MemoryCapacity),
		chains: make(map[string]*ReasoningChain),
		goals:  make(map[string]*Goal),
		history: make([]ExecutionRecord, 0),
	}
	
	if config.EnableLearning {
		learningStore, err := NewLearningStore(config.PersistencePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create learning store: %w", err)
		}
		engine.learningStore = learningStore
	}
	
	return engine, nil
}

// NewWorkingMemory creates a new working memory instance.
func NewWorkingMemory(capacity int) *WorkingMemory {
	return &WorkingMemory{
		facts:    make(map[string]Fact),
		capacity: capacity,
		lru:      make([]string, 0, capacity),
	}
}

// NewLearningStore creates a new learning store.
func NewLearningStore(path string) (*LearningStore, error) {
	store := &LearningStore{
		patterns: make(map[string]*Pattern),
		path:     path,
	}
	
	// Load existing patterns
	if err := store.Load(); err != nil {
		return nil, err
	}
	
	return store, nil
}

// Reason performs multi-step reasoning for a given task.
func (e *Engine) Reason(ctx context.Context, task string, inputs map[string]interface{}) (*ReasoningResult, error) {
	// Create execution record
	record := ExecutionRecord{
		ID:        generateID(),
		StartTime: time.Now(),
	}
	
	// Initialize working memory with inputs
	for key, value := range inputs {
		e.memory.AddFact(Fact{
			ID:        generateID(),
			Type:      FactTypeObservation,
			Subject:   key,
			Predicate: "has_value",
			Object:    value,
			Confidence: 1.0,
			Source:    "input",
			Timestamp: time.Now(),
		})
	}
	
	// Parse task to identify goal
	goal := e.parseGoal(task)
	e.goals[goal.ID] = goal
	
	// Create plan to achieve goal
	plan, err := e.createPlan(ctx, goal)
	if err != nil {
		record.EndTime = time.Now()
		record.Success = false
		record.Error = err
		e.history = append(e.history, record)
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}
	
	goal.Plan = plan
	
	// Execute plan
	result, err := e.executePlan(ctx, plan)
	if err != nil {
		record.EndTime = time.Now()
		record.Success = false
		record.Error = err
		e.history = append(e.history, record)
		return nil, fmt.Errorf("plan execution failed: %w", err)
	}
	
	// Learn from execution if enabled
	if e.config.EnableLearning {
		e.learn(goal, plan, result)
	}
	
	// Update execution record
	record.EndTime = time.Now()
	record.Success = true
	record.Result = result
	record.Facts = e.memory.GetFacts()
	e.history = append(e.history, record)
	
	return &ReasoningResult{
		Goal:     goal,
		Plan:     plan,
		Result:   result,
		Facts:    e.memory.GetFacts(),
		Duration: record.EndTime.Sub(record.StartTime),
	}, nil
}

// CreateChain creates a new reasoning chain.
func (e *Engine) CreateChain(chain *ReasoningChain) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if _, exists := e.chains[chain.ID]; exists {
		return fmt.Errorf("chain %s already exists", chain.ID)
	}
	
	e.chains[chain.ID] = chain
	return nil
}

// SetGoal sets a new goal for the reasoning engine.
func (e *Engine) SetGoal(goal *Goal) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	e.goals[goal.ID] = goal
	return nil
}

// GetMemory returns the current working memory state.
func (e *Engine) GetMemory() []Fact {
	return e.memory.GetFacts()
}

// Close closes the reasoning engine.
func (e *Engine) Close() error {
	if e.learningStore != nil {
		return e.learningStore.Save()
	}
	return nil
}

// parseGoal parses a task string to create a goal.
func (e *Engine) parseGoal(task string) *Goal {
	return &Goal{
		ID:          generateID(),
		Description: task,
		Priority:    1,
		Status:      GoalStatusPending,
		CreatedAt:   time.Now(),
		Deadline:    time.Now().Add(1 * time.Hour),
	}
}

// createPlan creates a plan to achieve a goal.
func (e *Engine) createPlan(ctx context.Context, goal *Goal) (*Plan, error) {
	// Check if we have a learned pattern for this type of goal
	if e.learningStore != nil {
		if pattern := e.learningStore.FindPattern(goal); pattern != nil {
			return e.createPlanFromPattern(goal, pattern), nil
		}
	}
	
	// Create a new plan using backward chaining
	plan := &Plan{
		ID:     generateID(),
		GoalID: goal.ID,
		Status: PlanStatusDraft,
		Steps:  make([]PlanStep, 0),
	}
	
	// Simple planning: create basic steps
	// In a real implementation, this would use more sophisticated planning
	plan.Steps = append(plan.Steps, PlanStep{
		ID:     generateID(),
		Action: "analyze",
		Parameters: map[string]interface{}{
			"task": goal.Description,
		},
		Status: StepStatusPending,
	})
	
	plan.Steps = append(plan.Steps, PlanStep{
		ID:     generateID(),
		Action: "execute",
		Parameters: map[string]interface{}{
			"goal": goal.ID,
		},
		Status: StepStatusPending,
	})
	
	plan.Status = PlanStatusReady
	return plan, nil
}

// createPlanFromPattern creates a plan from a learned pattern.
func (e *Engine) createPlanFromPattern(goal *Goal, pattern *Pattern) *Plan {
	plan := &Plan{
		ID:     generateID(),
		GoalID: goal.ID,
		Status: PlanStatusReady,
		Steps:  make([]PlanStep, 0),
	}
	
	for _, action := range pattern.Actions {
		plan.Steps = append(plan.Steps, PlanStep{
			ID:     generateID(),
			Action: action,
			Status: StepStatusPending,
		})
	}
	
	return plan
}

// executePlan executes a plan.
func (e *Engine) executePlan(ctx context.Context, plan *Plan) (interface{}, error) {
	plan.Status = PlanStatusExecuting
	
	var result interface{}
	for i, step := range plan.Steps {
		step.Status = StepStatusRunning
		
		// Execute step action
		stepResult, err := e.executeStep(ctx, &step)
		if err != nil {
			step.Status = StepStatusFailed
			plan.Status = PlanStatusFailed
			return nil, fmt.Errorf("step %s failed: %w", step.ID, err)
		}
		
		step.Status = StepStatusCompleted
		result = stepResult
		
		// Update progress
		plan.Progress = float64(i+1) / float64(len(plan.Steps))
	}
	
	plan.Status = PlanStatusCompleted
	return result, nil
}

// executeStep executes a single plan step.
func (e *Engine) executeStep(ctx context.Context, step *PlanStep) (interface{}, error) {
	// Execute based on action type
	switch step.Action {
	case "analyze":
		return e.analyzeTask(ctx, step.Parameters)
	case "execute":
		return e.executeGoal(ctx, step.Parameters)
	default:
		return nil, fmt.Errorf("unknown action: %s", step.Action)
	}
}

// analyzeTask analyzes a task.
func (e *Engine) analyzeTask(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	task, ok := params["task"].(string)
	if !ok {
		return nil, fmt.Errorf("task parameter required")
	}
	
	// Add analysis facts to memory
	e.memory.AddFact(Fact{
		ID:        generateID(),
		Type:      FactTypeInference,
		Subject:   "task",
		Predicate: "analyzed",
		Object:    task,
		Confidence: 0.9,
		Source:    "analyzer",
		Timestamp: time.Now(),
	})
	
	return map[string]interface{}{
		"analysis": fmt.Sprintf("Task '%s' has been analyzed", task),
	}, nil
}

// executeGoal executes a goal.
func (e *Engine) executeGoal(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	goalID, ok := params["goal"].(string)
	if !ok {
		return nil, fmt.Errorf("goal parameter required")
	}
	
	goal, exists := e.goals[goalID]
	if !exists {
		return nil, fmt.Errorf("goal %s not found", goalID)
	}
	
	// Mark goal as achieved
	goal.Status = GoalStatusAchieved
	
	return map[string]interface{}{
		"result": fmt.Sprintf("Goal '%s' achieved", goal.Description),
	}, nil
}

// learn learns from execution results.
func (e *Engine) learn(goal *Goal, plan *Plan, result interface{}) {
	if e.learningStore == nil {
		return
	}
	
	// Create pattern from successful execution
	pattern := &Pattern{
		ID:          generateID(),
		Type:        "goal_achievement",
		Trigger:     goal.Conditions,
		Actions:     make([]string, len(plan.Steps)),
		SuccessRate: 1.0,
		UsageCount:  1,
		LastUsed:    time.Now(),
	}
	
	for i, step := range plan.Steps {
		pattern.Actions[i] = step.Action
	}
	
	e.learningStore.AddPattern(pattern)
}

// WorkingMemory methods

// AddFact adds a fact to working memory.
func (wm *WorkingMemory) AddFact(fact Fact) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	
	// Check capacity and evict if needed
	if len(wm.facts) >= wm.capacity {
		wm.evictOldest()
	}
	
	wm.facts[fact.ID] = fact
	wm.lru = append(wm.lru, fact.ID)
}

// GetFacts returns all facts in working memory.
func (wm *WorkingMemory) GetFacts() []Fact {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	
	facts := make([]Fact, 0, len(wm.facts))
	for _, fact := range wm.facts {
		facts = append(facts, fact)
	}
	
	return facts
}

// evictOldest removes the oldest fact from memory.
func (wm *WorkingMemory) evictOldest() {
	if len(wm.lru) > 0 {
		oldest := wm.lru[0]
		delete(wm.facts, oldest)
		wm.lru = wm.lru[1:]
	}
}

// LearningStore methods

// AddPattern adds a pattern to the learning store.
func (ls *LearningStore) AddPattern(pattern *Pattern) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	
	ls.patterns[pattern.ID] = pattern
}

// FindPattern finds a pattern matching the goal.
func (ls *LearningStore) FindPattern(goal *Goal) *Pattern {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	
	// Simple pattern matching
	// In a real implementation, this would use more sophisticated matching
	for _, pattern := range ls.patterns {
		if pattern.Type == "goal_achievement" {
			return pattern
		}
	}
	
	return nil
}

// Load loads patterns from disk.
func (ls *LearningStore) Load() error {
	if ls.path == "" {
		return nil
	}
	
	file := filepath.Join(ls.path, "patterns.json")
	data, err := ioutil.ReadFile(file)
	if err != nil {
		// File doesn't exist yet
		return nil
	}
	
	return json.Unmarshal(data, &ls.patterns)
}

// Save saves patterns to disk.
func (ls *LearningStore) Save() error {
	if ls.path == "" {
		return nil
	}
	
	data, err := json.MarshalIndent(ls.patterns, "", "  ")
	if err != nil {
		return err
	}
	
	file := filepath.Join(ls.path, "patterns.json")
	return ioutil.WriteFile(file, data, 0644)
}

// ReasoningResult holds the result of reasoning.
type ReasoningResult struct {
	Goal     *Goal
	Plan     *Plan
	Result   interface{}
	Facts    []Fact
	Duration time.Duration
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}