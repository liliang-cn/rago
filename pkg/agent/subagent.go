package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// SubAgentMode defines how the sub-agent runs
type SubAgentMode string

const (
	// SubAgentModeForeground runs blocking, returns result
	SubAgentModeForeground SubAgentMode = "foreground"
	// SubAgentModeBackground runs non-blocking, can be resumed
	SubAgentModeBackground SubAgentMode = "background"
)

// SubAgentState defines the current state of a sub-agent
type SubAgentState string

const (
	SubAgentStatePending   SubAgentState = "pending"
	SubAgentStateRunning   SubAgentState = "running"
	SubAgentStateCompleted SubAgentState = "completed"
	SubAgentStateFailed    SubAgentState = "failed"
	SubAgentStatePaused    SubAgentState = "paused"
	SubAgentStateCancelled SubAgentState = "cancelled"
	SubAgentStateTimeout   SubAgentState = "timeout"
)

// SubAgentProgress represents progress information
type SubAgentProgress struct {
	SubagentID   string        `json:"subagent_id"`
	SubagentName string        `json:"subagent_name"`
	CurrentTurn  int           `json:"current_turn"`
	MaxTurns     int           `json:"max_turns"`
	State        SubAgentState `json:"state"`
	Goal         string        `json:"goal"`
	ElapsedTime  time.Duration `json:"elapsed_time"`
	Message      string        `json:"message,omitempty"`
}

// SubAgentProgressCallback is called when progress updates
type SubAgentProgressCallback func(progress SubAgentProgress)

// SubAgentConfig configures a sub-agent execution
type SubAgentConfig struct {
	Agent           *Agent                 // Target agent to run
	Mode            SubAgentMode           // Foreground or background
	MaxTurns        int                    // Maximum turns (default: 10)
	Isolated        bool                   // Isolate context from parent (default: true)
	ToolAllowlist   []string               // Only allow these tools (nil = all)
	ToolDenylist    []string               // Deny these tools
	ParentSession   *Session               // Parent session (for context inheritance)
	Goal            string                 // Task goal
	Context         map[string]interface{} // Additional context
	Service         *Service               // Parent service for tool access
	Timeout         time.Duration          // Execution timeout (0 = no timeout)
	ProgressCb      SubAgentProgressCallback // Progress callback
	RetryOnFailure  int                    // Number of retries on failure (default: 0)
	CancelOnTimeout bool                   // Cancel execution on timeout (default: true)
}

// SubAgent represents a wrapped agent execution with independent context
type SubAgent struct {
	id       string
	config   SubAgentConfig
	state    SubAgentState
	session  *Session // Isolated session

	// State management
	mu          sync.RWMutex
	currentTurn int
	result      interface{}
	err         error
	startTime   time.Time
	endTime     *time.Time

	// Context management
	ctx        context.Context
	cancel     context.CancelFunc
	timeoutCtx context.Context
	timeoutCancel context.CancelFunc

	// Hooks reference (from parent service)
	hooks *HookRegistry

	// Progress tracking
	progressChan chan SubAgentProgress
}

// SubAgentOption configures a SubAgent
type SubAgentOption func(*SubAgentConfig)

// NewSubAgent creates a new sub-agent wrapper
func NewSubAgent(cfg SubAgentConfig, opts ...SubAgentOption) *SubAgent {
	// Apply options
	for _, opt := range opts {
		opt(&cfg)
	}

	// Set defaults
	if cfg.MaxTurns == 0 {
		cfg.MaxTurns = 10
	}
	if cfg.Mode == "" {
		cfg.Mode = SubAgentModeForeground
	}

	// Create isolated session
	session := NewSession(cfg.Agent.ID())
	if cfg.Isolated && cfg.ParentSession != nil {
		// Copy only context, not full history
		session.Context = copyMap(cfg.ParentSession.Context)
	}

	// Get hooks from service
	var hooks *HookRegistry
	if cfg.Service != nil {
		hooks = cfg.Service.GetHooks()
	} else {
		hooks = GlobalHookRegistry()
	}

	return &SubAgent{
		id:           uuid.New().String(),
		config:       cfg,
		state:        SubAgentStatePending,
		session:      session,
		hooks:        hooks,
		progressChan: make(chan SubAgentProgress, 10),
	}
}

// ID returns the sub-agent ID
func (sa *SubAgent) ID() string {
	return sa.id
}

// Name returns the agent name
func (sa *SubAgent) Name() string {
	return sa.config.Agent.Name()
}

// ProgressChan returns a channel for progress updates
func (sa *SubAgent) ProgressChan() <-chan SubAgentProgress {
	return sa.progressChan
}

// emitProgress emits progress update
func (sa *SubAgent) emitProgress(message string) {
	sa.mu.RLock()
	elapsed := time.Since(sa.startTime)
	progress := SubAgentProgress{
		SubagentID:   sa.id,
		SubagentName: sa.config.Agent.Name(),
		CurrentTurn:  sa.currentTurn,
		MaxTurns:     sa.config.MaxTurns,
		State:        sa.state,
		Goal:         sa.config.Goal,
		ElapsedTime:  elapsed,
		Message:      message,
	}
	sa.mu.RUnlock()

	// Send to channel (non-blocking)
	select {
	case sa.progressChan <- progress:
	default:
	}

	// Call callback if set
	if sa.config.ProgressCb != nil {
		sa.config.ProgressCb(progress)
	}

	// Emit progress hook
	if sa.hooks != nil {
		sa.hooks.Emit(HookEventSubagentProgress, HookData{
			SubagentID:   sa.id,
			SubagentName: sa.config.Agent.Name(),
			Goal:         sa.config.Goal,
			Metadata: map[string]interface{}{
				"current_turn":  progress.CurrentTurn,
				"max_turns":     progress.MaxTurns,
				"elapsed_time":  progress.ElapsedTime.String(),
				"message":       message,
			},
		})
	}
}

// Run starts the sub-agent execution (blocking)
func (sa *SubAgent) Run(parentCtx context.Context) (interface{}, error) {
	sa.mu.Lock()
	sa.state = SubAgentStateRunning
	sa.startTime = time.Now()
	sa.ctx, sa.cancel = context.WithCancel(parentCtx)
	sa.mu.Unlock()

	// Setup timeout if configured
	if sa.config.Timeout > 0 {
		sa.timeoutCtx, sa.timeoutCancel = context.WithTimeout(sa.ctx, sa.config.Timeout)
		sa.ctx = sa.timeoutCtx
	}

	// Emit SubagentStart hook
	sa.hooks.Emit(HookEventSubagentStart, HookData{
		SubagentID:   sa.id,
		SubagentName: sa.config.Agent.Name(),
		Goal:         sa.config.Goal,
		SessionID:    sa.session.GetID(),
		AgentID:      sa.config.Agent.ID(),
		Metadata: map[string]interface{}{
			"max_turns": sa.config.MaxTurns,
			"timeout":   sa.config.Timeout.String(),
		},
	})

	defer func() {
		sa.mu.Lock()
		now := time.Now()
		sa.endTime = &now

		// Determine final state
		if sa.ctx != nil {
			if sa.ctx.Err() == context.DeadlineExceeded {
				sa.state = SubAgentStateTimeout
				sa.err = fmt.Errorf("execution timed out after %s", sa.config.Timeout)
			} else if sa.ctx.Err() == context.Canceled {
				if sa.state != SubAgentStatePaused {
					sa.state = SubAgentStateCancelled
					sa.err = fmt.Errorf("execution cancelled")
				}
			} else if sa.err != nil {
				sa.state = SubAgentStateFailed
			} else {
				sa.state = SubAgentStateCompleted
			}
		}
		sa.mu.Unlock()

		// Close progress channel
		close(sa.progressChan)

		// Cleanup timeout context
		if sa.timeoutCancel != nil {
			sa.timeoutCancel()
		}

		// Emit SubagentStop hook
		sa.hooks.Emit(HookEventSubagentStop, HookData{
			SubagentID:   sa.id,
			SubagentName: sa.config.Agent.Name(),
			Result:       sa.result,
			Error:        sa.err,
			Duration:     now.Sub(sa.startTime),
			SessionID:    sa.session.GetID(),
			Metadata: map[string]interface{}{
				"final_state": string(sa.state),
				"turns_used":  sa.currentTurn,
			},
		})
	}()

	// Execute with retry support
	var result interface{}
	var err error
	for attempt := 0; attempt <= sa.config.RetryOnFailure; attempt++ {
		result, err = sa.execute(sa.ctx)
		if err == nil {
			sa.result = result
			sa.err = nil
			return result, nil
		}

		// Don't retry on cancellation or timeout
		if sa.ctx.Err() != nil {
			break
		}

		// Retry
		if attempt < sa.config.RetryOnFailure {
			sa.emitProgress(fmt.Sprintf("Retrying (attempt %d/%d)", attempt+1, sa.config.RetryOnFailure))
		}
	}

	sa.result = result
	sa.err = err
	return result, err
}

// RunAsync starts the sub-agent in background
func (sa *SubAgent) RunAsync(parentCtx context.Context) <-chan *Event {
	eventChan := make(chan *Event, 50)

	go func() {
		defer close(eventChan)

		// Forward progress events
		go func() {
			for progress := range sa.progressChan {
				eventChan <- &Event{
					ID:        uuid.New().String(),
					Type:      EventTypeStateUpdate,
					AgentID:   sa.config.Agent.ID(),
					AgentName: sa.config.Agent.Name(),
					Content:   fmt.Sprintf("Turn %d/%d: %s", progress.CurrentTurn, progress.MaxTurns, progress.Message),
					Timestamp: time.Now(),
				}
			}
		}()

		result, err := sa.Run(parentCtx)

		// Send completion event
		eventChan <- &Event{
			ID:        uuid.New().String(),
			Type:      EventTypeComplete,
			AgentID:   sa.config.Agent.ID(),
			AgentName: sa.config.Agent.Name(),
			Content:   fmt.Sprintf("%v", result),
			Timestamp: time.Now(),
		}

		if err != nil {
			eventChan <- &Event{
				ID:        uuid.New().String(),
				Type:      EventTypeError,
				AgentID:   sa.config.Agent.ID(),
				AgentName: sa.config.Agent.Name(),
				Content:   err.Error(),
				Timestamp: time.Now(),
			}
		}
	}()

	return eventChan
}

// Cancel forcefully cancels the sub-agent execution
func (sa *SubAgent) Cancel() error {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	if sa.state != SubAgentStateRunning {
		return fmt.Errorf("subagent not running (current: %s)", sa.state)
	}

	sa.state = SubAgentStateCancelled

	// Cancel context
	if sa.cancel != nil {
		sa.cancel()
	}

	// Emit cancel hook
	sa.hooks.Emit(HookEventSubagentCancel, HookData{
		SubagentID:   sa.id,
		SubagentName: sa.config.Agent.Name(),
		Goal:         sa.config.Goal,
		Metadata: map[string]interface{}{
			"current_turn": sa.currentTurn,
		},
	})

	return nil
}

// Stop gracefully stops the sub-agent (alias for Cancel for clarity)
func (sa *SubAgent) Stop() error {
	return sa.Cancel()
}

// Wait waits for the sub-agent to complete and returns the result
func (sa *SubAgent) Wait(ctx context.Context) (interface{}, error) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			sa.mu.RLock()
			state := sa.state
			result := sa.result
			err := sa.err
			sa.mu.RUnlock()

			if state == SubAgentStateCompleted ||
				state == SubAgentStateFailed ||
				state == SubAgentStateCancelled ||
				state == SubAgentStateTimeout {
				return result, err
			}
		}
	}
}

// IsTerminal returns true if the sub-agent is in a terminal state
func (sa *SubAgent) IsTerminal() bool {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	return sa.state == SubAgentStateCompleted ||
		sa.state == SubAgentStateFailed ||
		sa.state == SubAgentStateCancelled ||
		sa.state == SubAgentStateTimeout
}

// execute runs the agent with tool filtering
func (sa *SubAgent) execute(ctx context.Context) (interface{}, error) {
	if sa.config.Service == nil {
		return nil, fmt.Errorf("service not configured for sub-agent")
	}

	// Collect tools with filtering
	tools := sa.collectFilteredTools(ctx, sa.config.Agent)

	// Build messages
	messages := []domain.Message{
		{Role: "user", Content: sa.config.Goal},
	}

	// Add context if provided
	if len(sa.config.Context) > 0 {
		contextStr := formatContext(sa.config.Context)
		messages[0].Content += "\n\n--- Context ---\n" + contextStr
	}

	sa.emitProgress("Starting execution")

	// Execute with turn limit
	for round := 0; round < sa.config.MaxTurns; round++ {
		// Check cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		sa.mu.Lock()
		sa.currentTurn = round + 1
		sa.mu.Unlock()

		sa.emitProgress(fmt.Sprintf("Turn %d/%d", sa.currentTurn, sa.config.MaxTurns))

		// Build system prompt
		systemMsg := sa.config.Service.buildSystemPrompt(sa.config.Agent)
		genMessages := append([]domain.Message{{Role: "system", Content: systemMsg}}, messages...)

		// LLM call
		result, err := sa.config.Service.llmService.GenerateWithTools(ctx, genMessages, tools, &domain.GenerationOptions{
			Temperature: 0.3,
			MaxTokens:   2000,
		})
		if err != nil {
			return nil, fmt.Errorf("LLM error: %w", err)
		}

		// Handle tool calls
		if len(result.ToolCalls) > 0 {
			// Add assistant message
			messages = append(messages, domain.Message{
				Role:      "assistant",
				Content:   result.Content,
				ToolCalls: result.ToolCalls,
			})

			// Execute tools
			for _, tc := range result.ToolCalls {
				sa.emitProgress(fmt.Sprintf("Executing tool: %s", tc.Function.Name))

				toolResult, toolErr, _ := sa.executeTool(ctx, tc)

				var content string
				if toolErr != nil {
					content = fmt.Sprintf("Error: %v", toolErr)
				} else {
					content = fmt.Sprintf("%v", toolResult)
				}

				messages = append(messages, domain.Message{
					Role:       "tool",
					ToolCallID: tc.ID,
					Content:    content,
				})
			}
			continue
		}

		// No tool calls - done
		sa.emitProgress("Execution completed")
		return result.Content, nil
	}

	return nil, fmt.Errorf("exceeded maximum turns (%d)", sa.config.MaxTurns)
}

// executeTool executes a single tool call with filtering
func (sa *SubAgent) executeTool(ctx context.Context, tc domain.ToolCall) (interface{}, error, bool) {
	// Check tool allowlist
	if len(sa.config.ToolAllowlist) > 0 && !containsStr(sa.config.ToolAllowlist, tc.Function.Name) {
		return nil, fmt.Errorf("tool %s not in allowlist", tc.Function.Name), false
	}

	// Check tool denylist
	if containsStr(sa.config.ToolDenylist, tc.Function.Name) {
		return nil, fmt.Errorf("tool %s is denied", tc.Function.Name), false
	}

	// Execute via service's tool execution logic
	rt := &Runtime{
		svc:          sa.config.Service,
		currentAgent: sa.config.Agent,
		session:      sa.session,
	}
	return rt.executeToolOrHandoff(ctx, tc)
}

// collectFilteredTools collects and filters tools for this sub-agent
func (sa *SubAgent) collectFilteredTools(ctx context.Context, agent *Agent) []domain.ToolDefinition {
	if sa.config.Service == nil {
		return nil
	}

	allTools := sa.config.Service.collectAllAvailableTools(ctx, agent)
	return filterTools(allTools, sa.config.ToolAllowlist, sa.config.ToolDenylist)
}

// filterTools filters tools based on allowlist and denylist
func filterTools(tools []domain.ToolDefinition, allowlist, denylist []string) []domain.ToolDefinition {
	if len(allowlist) == 0 && len(denylist) == 0 {
		return tools
	}

	denySet := make(map[string]bool)
	for _, name := range denylist {
		denySet[name] = true
	}

	var result []domain.ToolDefinition

	if len(allowlist) > 0 {
		allowSet := make(map[string]bool)
		for _, name := range allowlist {
			allowSet[name] = true
		}

		for _, tool := range tools {
			name := tool.Function.Name
			if allowSet[name] && !denySet[name] {
				result = append(result, tool)
			}
		}
	} else {
		for _, tool := range tools {
			name := tool.Function.Name
			if !denySet[name] {
				result = append(result, tool)
			}
		}
	}

	return result
}

// Resume resumes a paused/background sub-agent
func (sa *SubAgent) Resume(ctx context.Context, newGoal string) (interface{}, error) {
	sa.mu.Lock()
	if sa.state != SubAgentStatePaused {
		sa.mu.Unlock()
		return nil, fmt.Errorf("subagent not in paused state (current: %s)", sa.state)
	}
	sa.state = SubAgentStateRunning
	sa.config.Goal = newGoal
	sa.progressChan = make(chan SubAgentProgress, 10)
	sa.mu.Unlock()

	return sa.Run(ctx)
}

// Pause pauses a running sub-agent
func (sa *SubAgent) Pause() error {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	if sa.state != SubAgentStateRunning {
		return fmt.Errorf("subagent not running (current: %s)", sa.state)
	}

	if sa.cancel != nil {
		sa.cancel()
	}

	sa.state = SubAgentStatePaused
	return nil
}

// GetState returns current state
func (sa *SubAgent) GetState() SubAgentState {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	return sa.state
}

// GetResult returns the result (if completed)
func (sa *SubAgent) GetResult() (interface{}, error) {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	return sa.result, sa.err
}

// GetCurrentTurn returns the current turn number
func (sa *SubAgent) GetCurrentTurn() int {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	return sa.currentTurn
}

// GetSession returns the isolated session
func (sa *SubAgent) GetSession() *Session {
	return sa.session
}

// GetDuration returns the execution duration
func (sa *SubAgent) GetDuration() time.Duration {
	sa.mu.RLock()
	defer sa.mu.RUnlock()

	if sa.endTime != nil {
		return sa.endTime.Sub(sa.startTime)
	}
	if !sa.startTime.IsZero() {
		return time.Since(sa.startTime)
	}
	return 0
}

// Helper functions

func copyMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = v
	}
	return result
}

func formatContext(ctx map[string]interface{}) string {
	var result string
	for k, v := range ctx {
		result += fmt.Sprintf("- %s: %v\n", k, v)
	}
	return result
}
