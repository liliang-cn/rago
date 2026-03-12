package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/liliang-cn/agent-go/pkg/config"
	"github.com/liliang-cn/agent-go/pkg/domain"
	"github.com/liliang-cn/agent-go/pkg/pool"
	"github.com/liliang-cn/agent-go/pkg/services"
)

// SquadManager handles the lifecycle, discovery, and execution routing for squad members.
type SquadManager struct {
	store         *Store
	runningAgents map[string]context.CancelFunc // Tracks running agents if they are background loopers
	services      map[string]*Service           // Cached instantiated agent services
	mu            sync.RWMutex
	queueMu       sync.Mutex
	taskQueues    map[string][]string
	sharedTasks   map[string]*SharedTask
	queueRunning  map[string]bool
}

// TeamManager is kept as a compatibility alias for older call sites.
type TeamManager = SquadManager

type SharedTaskStatus string

const (
	SharedTaskStatusQueued    SharedTaskStatus = "queued"
	SharedTaskStatusRunning   SharedTaskStatus = "running"
	SharedTaskStatusCompleted SharedTaskStatus = "completed"
	SharedTaskStatusFailed    SharedTaskStatus = "failed"
	defaultSquadID                             = "squad-default-001"
	defaultSquadName                           = "Default Squad"
)

// SharedTaskResult captures the outcome of one delegated squad member call.
type SharedTaskResult struct {
	AgentName string `json:"agent_name"`
	Text      string `json:"text,omitempty"`
	Error     string `json:"error,omitempty"`
}

// SharedTask is a queued squad task owned by one captain.
type SharedTask struct {
	ID          string             `json:"id"`
	CaptainName string             `json:"captain_name"`
	AgentNames  []string           `json:"agent_names"`
	Prompt      string             `json:"prompt"`
	AckMessage  string             `json:"ack_message"`
	Status      SharedTaskStatus   `json:"status"`
	QueuedAhead int                `json:"queued_ahead"`
	ResultText  string             `json:"result_text,omitempty"`
	Results     []SharedTaskResult `json:"results,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	StartedAt   *time.Time         `json:"started_at,omitempty"`
	FinishedAt  *time.Time         `json:"finished_at,omitempty"`
}

// SeedDefaultMembers seeds a default Assistant captain and reusable specialists if none exist.
func (m *SquadManager) SeedDefaultMembers() error {
	if _, err := m.ensureDefaultSquad(); err != nil {
		return err
	}

	agents, err := m.store.ListAgentModels()
	if err != nil {
		return err
	}
	for _, member := range agents {
		if strings.EqualFold(member.Name, "Researcher") || strings.EqualFold(member.Name, "FileSystemAgent") || strings.EqualFold(member.Name, "Coder") {
			if err := m.store.DeleteAgentModel(member.ID); err != nil {
				return err
			}
		}
	}

	agents, err = m.store.ListAgentModels()
	if err != nil {
		return err
	}
	existing := make(map[string]struct{}, len(agents))
	for _, member := range agents {
		if strings.EqualFold(member.Name, "Assistant") {
			if member.TeamID != defaultSquadID || member.Kind != AgentKindCaptain || member.Status != AgentStatusRunning {
				member.TeamID = defaultSquadID
				member.Kind = AgentKindCaptain
				member.Status = AgentStatusRunning
				member.Description = "A general-purpose assistant for the squad. Can coordinate tasks and handle everyday requests."
				member.Instructions = "You are Assistant, the default general-purpose captain for this squad. Help the user directly when possible, and delegate to specialists only when it makes the task better."
				if len(member.MCPTools) == 0 {
					member.MCPTools = defaultMemberMCPTools("Assistant")
				}
				member.EnableMCP = true
				member.EnableRAG = true
				member.EnableMemory = true
				member.UpdatedAt = time.Now()
				if err := m.store.SaveAgentModel(member); err != nil {
					return err
				}
			}
		}
		existing[strings.ToLower(member.Name)] = struct{}{}
	}

	squads, err := m.store.ListTeams()
	if err != nil {
		return err
	}
	for _, squad := range squads {
		if strings.EqualFold(strings.TrimSpace(squad.Name), "Default Team") {
			if err := m.store.DeleteTeam(squad.ID); err != nil {
				return err
			}
		}
	}

	defaults := []*AgentModel{
		{
			ID:           "agent-assistant-001",
			TeamID:       defaultSquadID,
			Name:         "Assistant",
			Kind:         AgentKindCaptain,
			Description:  "A general-purpose assistant for the squad. Can coordinate tasks and handle everyday requests.",
			Instructions: "You are Assistant, the default general-purpose captain for this squad. Help the user directly when possible, and delegate to specialists only when it makes the task better.",
			Status:       AgentStatusRunning,
			MCPTools:     defaultMemberMCPTools("Assistant"),
			EnableRAG:    true,
			EnableMemory: true,
			EnableMCP:    true,
		},
	}

	for _, a := range defaults {
		if _, ok := existing[strings.ToLower(a.Name)]; ok {
			continue
		}
		if err := m.store.SaveAgentModel(a); err != nil {
			return err
		}
	}
	return nil
}

func (m *SquadManager) ensureDefaultSquad() (*Squad, error) {
	squads, err := m.store.ListTeams()
	if err != nil {
		return nil, err
	}
	for _, squad := range squads {
		if squad.ID == defaultSquadID || strings.EqualFold(squad.Name, defaultSquadName) {
			return squad, nil
		}
	}

	squad := &Squad{
		ID:          defaultSquadID,
		Name:        defaultSquadName,
		Description: "Default workspace squad.",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := m.store.SaveTeam(squad); err != nil {
		return nil, err
	}
	return squad, nil
}

// NewSquadManager creates a new squad manager based on a store.
func NewSquadManager(s *Store) *SquadManager {
	return &SquadManager{
		store:         s,
		runningAgents: make(map[string]context.CancelFunc),
		services:      make(map[string]*Service),
		taskQueues:    make(map[string][]string),
		sharedTasks:   make(map[string]*SharedTask),
		queueRunning:  make(map[string]bool),
	}
}

// NewTeamManager is kept as a compatibility alias for older call sites.
func NewTeamManager(s *Store) *TeamManager {
	return NewSquadManager(s)
}

// EnqueueSharedTask queues a squad task under one captain and returns an immediate acknowledgement.
func (m *SquadManager) EnqueueSharedTask(ctx context.Context, captainName string, agentNames []string, prompt string) (*SharedTask, error) {
	if strings.TrimSpace(prompt) == "" {
		return nil, fmt.Errorf("message required")
	}
	if strings.TrimSpace(captainName) == "" {
		captainName = "Assistant"
	}

	captain, err := m.GetMemberByName(captainName)
	if err != nil {
		return nil, fmt.Errorf("cannot load captain %s: %w", captainName, err)
	}
	if captain.Kind != AgentKindCaptain {
		return nil, fmt.Errorf("%s is not a captain", captainName)
	}

	if len(agentNames) == 0 {
		agentNames = []string{captain.Name}
	}

	for _, name := range agentNames {
		member, memberErr := m.GetMemberByName(name)
		if memberErr != nil {
			return nil, fmt.Errorf("cannot load squad member %s: %w", name, memberErr)
		}
		if member.Kind == AgentKindCaptain && !strings.EqualFold(name, captain.Name) {
			return nil, fmt.Errorf("%s is a captain and cannot be delegated from %s", name, captain.Name)
		}
	}

	now := time.Now()
	task := &SharedTask{
		ID:          uuid.New().String(),
		CaptainName: captain.Name,
		AgentNames:  append([]string(nil), agentNames...),
		Prompt:      strings.TrimSpace(prompt),
		Status:      SharedTaskStatusQueued,
		CreatedAt:   now,
	}

	m.queueMu.Lock()
	queuedAhead := len(m.taskQueues[captain.Name])
	if m.queueRunning[captain.Name] {
		queuedAhead++
	}
	task.QueuedAhead = queuedAhead
	task.AckMessage = buildSharedTaskAck(captain.Name, queuedAhead)
	m.sharedTasks[task.ID] = task
	m.taskQueues[captain.Name] = append(m.taskQueues[captain.Name], task.ID)
	shouldStartWorker := !m.queueRunning[captain.Name]
	if shouldStartWorker {
		m.queueRunning[captain.Name] = true
	}
	m.queueMu.Unlock()

	if shouldStartWorker {
		go m.runSharedTaskQueue(context.WithoutCancel(ctx), captain.Name)
	}

	return cloneSharedTask(task), nil
}

// ListSharedTasks returns recent queued or completed squad tasks for one captain.
func (m *SquadManager) ListSharedTasks(captainName string, since time.Time, limit int) []*SharedTask {
	m.queueMu.Lock()
	defer m.queueMu.Unlock()

	tasks := make([]*SharedTask, 0, len(m.sharedTasks))
	for _, task := range m.sharedTasks {
		if captainName != "" && !strings.EqualFold(task.CaptainName, captainName) {
			continue
		}
		if !since.IsZero() && task.CreatedAt.Before(since) && (task.FinishedAt == nil || task.FinishedAt.Before(since)) {
			continue
		}
		tasks = append(tasks, cloneSharedTask(task))
	}

	slices.SortFunc(tasks, func(a, b *SharedTask) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})
	if limit > 0 && len(tasks) > limit {
		tasks = tasks[len(tasks)-limit:]
	}
	return tasks
}

func (m *SquadManager) runSharedTaskQueue(ctx context.Context, captainName string) {
	for {
		task := m.nextQueuedTask(captainName)
		if task == nil {
			return
		}
		m.executeSharedTask(ctx, task)
	}
}

func (m *SquadManager) nextQueuedTask(captainName string) *SharedTask {
	m.queueMu.Lock()
	defer m.queueMu.Unlock()

	queue := m.taskQueues[captainName]
	if len(queue) == 0 {
		delete(m.queueRunning, captainName)
		return nil
	}

	taskID := queue[0]
	if len(queue) == 1 {
		delete(m.taskQueues, captainName)
	} else {
		m.taskQueues[captainName] = queue[1:]
	}

	task := m.sharedTasks[taskID]
	if task == nil {
		return nil
	}
	now := time.Now()
	task.Status = SharedTaskStatusRunning
	task.StartedAt = &now
	task.QueuedAhead = 0
	return cloneSharedTask(task)
}

func (m *SquadManager) executeSharedTask(ctx context.Context, task *SharedTask) {
	type dispatchResult struct {
		AgentName string
		Text      string
		Err       error
	}

	results := make([]SharedTaskResult, 0, len(task.AgentNames))
	resultTextParts := make([]string, 0, len(task.AgentNames))
	resultCh := make(chan dispatchResult, len(task.AgentNames))
	var wg sync.WaitGroup

	for _, agentName := range task.AgentNames {
		agentName := agentName
		wg.Add(1)
		go func() {
			defer wg.Done()
			text, err := m.DispatchTask(ctx, agentName, task.Prompt)
			resultCh <- dispatchResult{AgentName: agentName, Text: strings.TrimSpace(text), Err: err}
		}()
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	failed := false
	ordered := make(map[string]dispatchResult, len(task.AgentNames))
	for result := range resultCh {
		ordered[result.AgentName] = result
		if result.Err != nil {
			failed = true
		}
	}

	for _, agentName := range task.AgentNames {
		result := ordered[agentName]
		item := SharedTaskResult{AgentName: agentName, Text: result.Text}
		if result.Err != nil {
			item.Error = result.Err.Error()
			resultTextParts = append(resultTextParts, fmt.Sprintf("## %s\nError: %s", agentName, result.Err))
		} else {
			text := result.Text
			if text == "" {
				text = "No response returned."
			}
			resultTextParts = append(resultTextParts, fmt.Sprintf("## %s\n%s", agentName, text))
		}
		results = append(results, item)
	}

	now := time.Now()
	m.queueMu.Lock()
	defer m.queueMu.Unlock()
	stored := m.sharedTasks[task.ID]
	if stored == nil {
		return
	}
	stored.Results = results
	stored.ResultText = strings.Join(resultTextParts, "\n\n")
	stored.FinishedAt = &now
	if failed {
		stored.Status = SharedTaskStatusFailed
	} else {
		stored.Status = SharedTaskStatusCompleted
	}
}

func buildSharedTaskAck(captainName string, queuedAhead int) string {
	if queuedAhead > 0 {
		return fmt.Sprintf("%s received that. It is queued behind %d task(s).", captainName, queuedAhead)
	}
	return fmt.Sprintf("%s received that. Starting it now.", captainName)
}

func cloneSharedTask(task *SharedTask) *SharedTask {
	if task == nil {
		return nil
	}
	cloned := *task
	cloned.AgentNames = append([]string(nil), task.AgentNames...)
	cloned.Results = append([]SharedTaskResult(nil), task.Results...)
	if task.StartedAt != nil {
		startedAt := *task.StartedAt
		cloned.StartedAt = &startedAt
	}
	if task.FinishedAt != nil {
		finishedAt := *task.FinishedAt
		cloned.FinishedAt = &finishedAt
	}
	return &cloned
}

// EnableCaptain marks a captain as enabled in the database.
func (m *SquadManager) EnableCaptain(ctx context.Context, name string) error {
	model, err := m.store.GetAgentModelByName(name)
	if err != nil {
		return fmt.Errorf("failed to load agent %s: %w", name, err)
	}
	if model.Kind != AgentKindCaptain {
		return fmt.Errorf("agent '%s' is a specialist and does not support persistent lifecycle state", name)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Optionally start a background task queue poller here for async mode.
	// For now, we mark it as running and cache the builder.

	err = m.store.UpdateAgentStatus(model.ID, AgentStatusRunning)
	if err != nil {
		return err
	}
	model.Status = AgentStatusRunning

	return nil
}

// DisableCaptain marks a captain as disabled.
func (m *SquadManager) DisableCaptain(ctx context.Context, name string) error {
	model, err := m.store.GetAgentModelByName(name)
	if err != nil {
		return err
	}
	if model.Kind != AgentKindCaptain {
		return fmt.Errorf("agent '%s' is a specialist and does not support persistent lifecycle state", name)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if cancel, exists := m.runningAgents[name]; exists {
		cancel()
		delete(m.runningAgents, name)
	}
	delete(m.services, name)

	return m.store.UpdateAgentStatus(model.ID, AgentStatusStopped)
}

// ListActiveCaptains returns a list of enabled captains.
func (m *SquadManager) ListActiveCaptains() ([]*AgentModel, error) {
	all, err := m.store.ListAgentModels()
	if err != nil {
		return nil, err
	}
	var running []*AgentModel
	for _, a := range all {
		if a.Status == AgentStatusRunning {
			running = append(running, a)
		}
	}
	return running, nil
}

// ListMembers returns all registered captains and specialists.
func (m *SquadManager) ListMembers() ([]*AgentModel, error) {
	return m.store.ListAgentModels()
}

// CreateMember persists a new squad member configuration.
func (m *SquadManager) CreateMember(_ context.Context, model *AgentModel) (*AgentModel, error) {
	if model == nil {
		return nil, fmt.Errorf("agent model is required")
	}

	now := time.Now()
	if model.ID == "" {
		model.ID = uuid.New().String()
	}
	if model.Name == "" {
		return nil, fmt.Errorf("agent name is required")
	}
	if model.Description == "" {
		model.Description = model.Name
	}
	if model.Instructions == "" {
		model.Instructions = model.Description
	}
	if model.Status == "" {
		model.Status = AgentStatusStopped
	}
	if model.Kind == "" {
		model.Kind = AgentKindCaptain
	}
	if strings.TrimSpace(model.TeamID) == "" {
		defaultSquad, err := m.ensureDefaultSquad()
		if err != nil {
			return nil, err
		}
		model.TeamID = defaultSquad.ID
	}
	if len(model.MCPTools) == 0 {
		model.MCPTools = defaultMemberMCPTools(model.Name)
	}
	if len(model.MCPTools) > 0 {
		model.EnableMCP = true
	}
	if model.Kind == AgentKindCaptain {
		model.EnableRAG = true
		model.EnableMemory = true
	}
	if model.RequiredLLMCapability < 0 {
		model.RequiredLLMCapability = 0
	}
	if model.CreatedAt.IsZero() {
		model.CreatedAt = now
	}
	model.UpdatedAt = now

	if err := m.store.SaveAgentModel(model); err != nil {
		return nil, err
	}

	return m.store.GetAgentModel(model.ID)
}

func (m *SquadManager) CreateSquad(_ context.Context, squad *Squad) (*Squad, error) {
	if squad == nil {
		return nil, fmt.Errorf("squad is required")
	}
	if strings.TrimSpace(squad.Name) == "" {
		return nil, fmt.Errorf("squad name is required")
	}
	if strings.TrimSpace(squad.ID) == "" {
		squad.ID = uuid.New().String()
	}
	if strings.TrimSpace(squad.Description) == "" {
		squad.Description = squad.Name
	}
	now := time.Now()
	if squad.CreatedAt.IsZero() {
		squad.CreatedAt = now
	}
	squad.UpdatedAt = now

	if err := m.store.SaveTeam(squad); err != nil {
		return nil, err
	}

	captainName := fmt.Sprintf("%s Assistant", squad.Name)
	_, err := m.CreateMember(context.Background(), &AgentModel{
		ID:           uuid.New().String(),
		TeamID:       squad.ID,
		Name:         captainName,
		Kind:         AgentKindCaptain,
		Description:  fmt.Sprintf("Default captain agent for %s.", squad.Name),
		Instructions: fmt.Sprintf("You are the captain agent for squad %s. Help directly when possible and coordinate specialists when useful.", squad.Name),
		Status:       AgentStatusRunning,
		MCPTools:     defaultMemberMCPTools("Assistant"),
		EnableRAG:    true,
		EnableMemory: true,
		EnableMCP:    true,
	})
	if err != nil {
		return nil, err
	}

	return m.store.GetTeam(squad.ID)
}

func (m *SquadManager) ListSquads() ([]*Squad, error) {
	return m.store.ListTeams()
}

func (m *SquadManager) GetSquadByName(name string) (*Squad, error) {
	return m.store.GetTeamByName(strings.TrimSpace(name))
}

func (m *SquadManager) AddCaptain(ctx context.Context, squadID, name, description, instructions string) (*AgentModel, error) {
	return m.CreateMember(ctx, &AgentModel{
		TeamID:       strings.TrimSpace(squadID),
		Name:         strings.TrimSpace(name),
		Kind:         AgentKindCaptain,
		Description:  strings.TrimSpace(description),
		Instructions: strings.TrimSpace(instructions),
	})
}

func (m *SquadManager) AddSpecialist(ctx context.Context, squadID, name, description, instructions string) (*AgentModel, error) {
	return m.CreateMember(ctx, &AgentModel{
		TeamID:       strings.TrimSpace(squadID),
		Name:         strings.TrimSpace(name),
		Kind:         AgentKindSpecialist,
		Description:  strings.TrimSpace(description),
		Instructions: strings.TrimSpace(instructions),
	})
}

func (m *SquadManager) ListCaptains() ([]*AgentModel, error) {
	members, err := m.ListMembers()
	if err != nil {
		return nil, err
	}
	captains := make([]*AgentModel, 0, len(members))
	for _, member := range members {
		if member.Kind == AgentKindCaptain {
			captains = append(captains, member)
		}
	}
	return captains, nil
}

func (m *SquadManager) ListSpecialists() ([]*AgentModel, error) {
	members, err := m.ListMembers()
	if err != nil {
		return nil, err
	}
	specialists := make([]*AgentModel, 0, len(members))
	for _, member := range members {
		if member.Kind == AgentKindSpecialist {
			specialists = append(specialists, member)
		}
	}
	return specialists, nil
}

func (m *SquadManager) CreateTeam(ctx context.Context, team *Team) (*Team, error) {
	return m.CreateSquad(ctx, team)
}

func (m *SquadManager) ListTeams() ([]*Team, error) {
	return m.ListSquads()
}

func (m *SquadManager) EnableCommander(ctx context.Context, name string) error {
	return m.EnableCaptain(ctx, name)
}

func (m *SquadManager) DisableCommander(ctx context.Context, name string) error {
	return m.DisableCaptain(ctx, name)
}

func (m *SquadManager) ListActiveCommanders() ([]*AgentModel, error) {
	return m.ListActiveCaptains()
}

// GetMemberByName retrieves a persisted team member model by name.
func (m *SquadManager) GetMemberByName(name string) (*AgentModel, error) {
	return m.store.GetAgentModelByName(name)
}

// getOrBuildService returns a cached service or builds a new one for the agent model.
func (m *SquadManager) getOrBuildService(name string) (*Service, error) {
	m.mu.RLock()
	svc, exists := m.services[name]
	m.mu.RUnlock()

	if exists {
		return svc, nil
	}

	model, err := m.store.GetAgentModelByName(name)
	if err != nil {
		return nil, err
	}

	if model.Kind == AgentKindCaptain && model.Status != AgentStatusRunning {
		return nil, fmt.Errorf("agent '%s' is not running", name)
	}

	var agentgoCfg *config.Config
	builder := New(model.Name)

	if cfg, cfgErr := config.Load(""); cfgErr == nil {
		agentgoCfg = cfg
		builder.WithSystemPrompt(buildTeamSystemPrompt(cfg, model) + "\n\n" + buildTeamMemberPrompt(model))
	} else {
		builder.WithSystemPrompt(buildTeamMemberPrompt(model))
	}

	if agentgoCfg != nil {
		builder.WithConfig(agentgoCfg)

		globalPool := services.GetGlobalPoolService()
		if initErr := globalPool.Initialize(context.Background(), agentgoCfg); initErr == nil {
			if llmSvc, llmErr := globalPool.GetLLMServiceWithHint(pool.SelectionHint{
				PreferredProvider: model.Model,
				PreferredModel:    model.Model,
				MinCapability:     model.RequiredLLMCapability,
			}); llmErr == nil {
				builder.WithLLM(llmSvc)
			}
		}
	}

	if model.EnableRAG {
		builder.WithRAG()
	}
	if model.EnableMemory {
		builder.WithMemory()
	}
	if model.EnablePTC {
		builder.WithPTC()
	}
	if model.EnableMCP {
		builder.WithMCP()
	}

	// If the model specifies an LLM model string, this logic would require pool support to select specifically.
	// For now, relies on the default or global pool inside Build().

	if len(model.Skills) > 0 {
		builder.WithSkills()
	}

	newSvc, err := builder.Build()
	if err != nil {
		return nil, err
	}

	// Apply tool filters to the agent
	allowedMCPTools := model.MCPTools
	if len(allowedMCPTools) == 0 {
		allowedMCPTools = defaultMemberMCPTools(model.Name)
	}
	if len(allowedMCPTools) > 0 {
		newSvc.agent.SetAllowedMCPTools(allowedMCPTools)
	} else {
		newSvc.agent.SetAllowedMCPTools([]string{}) // none allowed if empty
	}

	if len(model.Skills) > 0 {
		newSvc.agent.SetAllowedSkills(model.Skills)
	} else {
		newSvc.agent.SetAllowedSkills([]string{}) // none allowed if empty
	}

	if model.Model != "" {
		newSvc.agent.SetModel(model.Model)
	}

	m.mu.Lock()
	m.services[name] = newSvc
	m.mu.Unlock()

	return newSvc, nil
}

func buildTeamSystemPrompt(cfg *config.Config, model *AgentModel) string {
	if cfg == nil {
		return "You are working as part of an AgentGo squad."
	}

	workspace := strings.TrimSpace(cfg.WorkspaceDir())
	projectRoot := ""
	if cwd, err := os.Getwd(); err == nil {
		projectRoot = strings.TrimSpace(cwd)
	}

	lines := []string{
		"You are working as part of an AgentGo squad.",
		"The squad shares one workspace and one project context.",
	}
	lines = append(lines, buildRuntimeContextLines()...)
	if workspace != "" {
		lines = append(lines, "- Shared writable workspace: "+workspace)
	}
	if projectRoot != "" {
		lines = append(lines, "- Active project root: "+projectRoot)
		lines = append(lines, "- Stay inside the active project root unless the user explicitly asks for another location.")
	}
	if model != nil && strings.EqualFold(strings.TrimSpace(model.Name), "assistant") {
		lines = append(lines,
			"- You are the default captain for the squad.",
			"- Handle direct user requests when possible and delegate specialist work only when that improves the result.",
		)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func buildTeamMemberPrompt(model *AgentModel) string {
	if model == nil {
		return ""
	}

	base := strings.TrimSpace(model.Instructions)
	extras := make([]string, 0, 8)

	switch strings.ToLower(strings.TrimSpace(model.Name)) {
	case "coder":
		extras = append(extras,
			"- Use whichever capabilities are currently available in this runtime to complete the task. MCP filesystem tools are common here, but they are not the only possible path.",
			"- If the task says create/write/save/update a file, you MUST call a filesystem write or modify tool and actually change the file.",
			"- Do not stop after listing files when the task clearly asks you to write or edit a file.",
			"- Directory listing is only for confirmation or discovery. It is not a valid final action for a file-writing task.",
			"- After writing a file, briefly state which file was changed and what was written.",
			"- Return the exact file path(s) you changed.",
		)
	default:
		extras = append(extras,
			"- Use the capabilities that are actually available in the current runtime. Do not assume the work must go through MCP if another available path fits better.",
			"- For repository or filesystem questions, prefer targeted file reads over broad directory traversal.",
			"- If the task already names specific files such as Makefile, package.json, App.tsx, or main.go, read those files first before calling list_directory.",
			"- Never inspect blacklisted repository paths unless the user explicitly asks for them. Blacklist: "+FormatRepositoryIgnoreList(),
			"- Avoid full repository tree scans. Use directory listing only when you need quick structure confirmation, and do it on one narrow path at a time.",
			"- Delegate specialized implementation work to specialists instead of carrying their detailed operating rules yourself.",
		)
	}

	if len(extras) == 0 {
		return base
	}
	return strings.TrimSpace(base + "\n\n" + strings.Join(extras, "\n"))
}

func buildTeamTaskEnvelope(cfg *config.Config, agentName, instruction string) string {
	instruction = strings.TrimSpace(instruction)
	if instruction == "" {
		return ""
	}

	lines := []string{
		"Squad task context:",
		"- Target squad member: " + strings.TrimSpace(agentName),
	}
	lines = append(lines, buildRuntimeContextLines()...)
	if cfg != nil {
		if workspace := strings.TrimSpace(cfg.WorkspaceDir()); workspace != "" {
			lines = append(lines, "- Shared writable workspace: "+workspace)
		}
	}
	if projectRoot, err := os.Getwd(); err == nil && strings.TrimSpace(projectRoot) != "" {
		lines = append(lines, "- Active project root: "+strings.TrimSpace(projectRoot))
		lines = append(lines, "- Stay inside the active project root unless the user explicitly asks for another location.")
	}
	lines = append(lines,
		"- The bullets above are context, not the requested action.",
		"- Execute only the work described in the Task section below.",
		"- Keep your response focused on your own role.",
		"- Ignore blacklisted repository paths unless the user explicitly asks for them: "+FormatRepositoryIgnoreList(),
		"",
		"Task:",
		instruction,
	)
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func buildRuntimeContextLines() []string {
	lines := []string{
		"- Runtime OS/arch: " + runtime.GOOS + "/" + runtime.GOARCH,
	}
	if shell := strings.TrimSpace(os.Getenv("SHELL")); shell != "" {
		lines = append(lines, "- Interactive shell: "+shell)
		lines = append(lines, "- If you provide shell commands or scripts, prefer compatibility with this shell when practical.")
	}
	return lines
}

func defaultMemberMCPTools(name string) []string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "assistant":
		return []string{
			"mcp_filesystem_list_allowed_directories",
			"mcp_filesystem_list_directory",
			"mcp_filesystem_read_file",
			"mcp_filesystem_read_multiple_files",
			"mcp_filesystem_search_within_files",
			"mcp_filesystem_get_file_info",
			"mcp_filesystem_create_directory",
			"mcp_filesystem_write_file",
			"mcp_filesystem_modify_file",
			"mcp_filesystem_move_file",
			"mcp_filesystem_copy_file",
			"mcp_filesystem_delete_file",
			"mcp_websearch_websearch_ai_summary",
			"mcp_websearch_fetch_page_content",
			"mcp_websearch_deep_read_page",
		}
	case "coder":
		return []string{
			"mcp_filesystem_list_allowed_directories",
			"mcp_filesystem_list_directory",
			"mcp_filesystem_read_file",
			"mcp_filesystem_read_multiple_files",
			"mcp_filesystem_search_within_files",
			"mcp_filesystem_get_file_info",
			"mcp_filesystem_create_directory",
			"mcp_filesystem_write_file",
			"mcp_filesystem_modify_file",
			"mcp_filesystem_move_file",
			"mcp_filesystem_copy_file",
			"mcp_filesystem_delete_file",
		}
	default:
		return []string{
			"mcp_filesystem_list_allowed_directories",
			"mcp_filesystem_list_directory",
			"mcp_filesystem_read_file",
			"mcp_filesystem_read_multiple_files",
			"mcp_filesystem_search_within_files",
			"mcp_filesystem_get_file_info",
			"mcp_filesystem_create_directory",
			"mcp_filesystem_write_file",
			"mcp_filesystem_modify_file",
			"mcp_filesystem_move_file",
			"mcp_filesystem_copy_file",
			"mcp_filesystem_delete_file",
			"mcp_websearch_websearch_ai_summary",
			"mcp_websearch_fetch_page_content",
			"mcp_websearch_deep_read_page",
		}
	}
}

func (m *SquadManager) ensureAgentRunning(ctx context.Context, name string) error {
	model, err := m.store.GetAgentModelByName(name)
	if err != nil {
		return err
	}
	if model.Kind == AgentKindSpecialist {
		return nil
	}
	if model.Status == AgentStatusRunning {
		return nil
	}
	return m.EnableCaptain(ctx, name)
}

func extractDispatchText(res *ExecutionResult) string {
	if res == nil {
		return ""
	}

	if res.PTCResult != nil && res.PTCResult.Type != PTCResultTypeText {
		text := strings.TrimSpace(res.PTCResult.FormatForLLM())
		if isMeaningfulDispatchText(text) {
			return text
		}
	}

	textCandidates := []string{
		res.Text(),
	}

	if s, ok := res.Metadata["dispatch_result"].(string); ok {
		textCandidates = append(textCandidates, s)
	}
	if s, ok := res.Metadata["final_text"].(string); ok {
		textCandidates = append(textCandidates, s)
	}

	for _, candidate := range textCandidates {
		candidate = sanitizeDispatchText(candidate)
		if isMeaningfulDispatchText(candidate) {
			return candidate
		}
	}

	if res.FinalResult != nil {
		if bz, err := json.Marshal(res.FinalResult); err == nil {
			candidate := strings.TrimSpace(string(bz))
			if candidate != "" && candidate != "null" {
				return candidate
			}
		}
	}

	for _, candidate := range textCandidates {
		candidate = sanitizeDispatchText(candidate)
		if candidate != "" {
			return candidate
		}
	}

	return ""
}

var thinkBlockRe = regexp.MustCompile(`(?s)<think>.*?</think>`)

func sanitizeDispatchText(text string) string {
	text = thinkBlockRe.ReplaceAllString(text, "")
	return strings.TrimSpace(text)
}

func isMeaningfulDispatchText(text string) bool {
	if text == "" {
		return false
	}

	normalized := strings.ToLower(strings.TrimSpace(text))
	generic := map[string]struct{}{
		"task complete":                {},
		"the task has been completed.": {},
		"the task has been completed":  {},
		"the task has been completed. the information has been saved to memory.": {},
		"the information has been saved to memory.":                              {},
		"done": {},
	}

	_, isGeneric := generic[normalized]
	return !isGeneric
}

// DispatchTask runs the task on the target squad member service directly.
func (m *SquadManager) DispatchTask(ctx context.Context, agentName string, instruction string) (string, error) {
	if err := m.ensureAgentRunning(ctx, agentName); err != nil {
		return "", fmt.Errorf("cannot start agent %s: %w", agentName, err)
	}

	svc, err := m.getOrBuildService(agentName)
	if err != nil {
		return "", fmt.Errorf("cannot dispatch to agent %s: %w", agentName, err)
	}

	wrappedInstruction := instruction
	if cfg, cfgErr := config.Load(""); cfgErr == nil {
		wrappedInstruction = buildTeamTaskEnvelope(cfg, agentName, instruction)
	}

	// For dispatch, we create a temporary sub-agent flow or run directly
	// Let's run a single session Run
	res, err := svc.Run(ctx, wrappedInstruction, dispatchRunOptions(agentName)...)
	if err != nil {
		return "", err
	}

	if text := extractDispatchText(res); text != "" {
		return text, nil
	}

	bz, _ := json.Marshal(res.FinalResult)
	return string(bz), nil
}

// DispatchTaskStream runs the task on the target agent and returns the raw event stream.
func (m *SquadManager) DispatchTaskStream(ctx context.Context, agentName string, instruction string) (<-chan *Event, error) {
	if err := m.ensureAgentRunning(ctx, agentName); err != nil {
		return nil, fmt.Errorf("cannot start agent %s: %w", agentName, err)
	}

	svc, err := m.getOrBuildService(agentName)
	if err != nil {
		return nil, fmt.Errorf("cannot dispatch to agent %s: %w", agentName, err)
	}

	wrappedInstruction := instruction
	if cfg, cfgErr := config.Load(""); cfgErr == nil {
		wrappedInstruction = buildTeamTaskEnvelope(cfg, agentName, instruction)
	}

	return svc.RunStream(ctx, wrappedInstruction)
}

func dispatchRunOptions(agentName string) []RunOption {
	name := strings.ToLower(strings.TrimSpace(agentName))
	switch name {
	case "coder":
		return []RunOption{WithMaxTurns(10), WithTemperature(0.1)}
	default:
		return []RunOption{WithMaxTurns(14), WithTemperature(0.1)}
	}
}

// RegisterCaptainTools adds the squad management tools to the frontdesk captain agent.
func (m *SquadManager) RegisterCaptainTools(captain *Service) {
	// 1. discover_agents
	discoverDef := domain.ToolDefinition{
		Type: "function",
		Function: domain.ToolFunction{
			Name:        "discover_agents",
			Description: "Discover all available specialized agents in the system and their descriptions.",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}
	captain.toolRegistry.Register(discoverDef, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		agents, err := m.ListMembers()
		if err != nil {
			return nil, err
		}
		var result []map[string]interface{}
		for _, a := range agents {
			result = append(result, map[string]interface{}{
				"name":        a.Name,
				"description": a.Description,
				"status":      a.Status,
			})
		}
		return result, nil
	}, CategoryCustom)

	// 2. start_agent
	startDef := domain.ToolDefinition{
		Type: "function",
		Function: domain.ToolFunction{
			Name:        "start_agent",
			Description: "Start/wake up a specific agent by name.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "The name of the agent to start.",
					},
				},
				"required": []string{"name"},
			},
		},
	}
	captain.toolRegistry.Register(startDef, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		name, _ := args["name"].(string)
		err := m.EnableCaptain(ctx, name)
		if err != nil {
			return nil, err
		}
		return fmt.Sprintf("Agent '%s' started successfully.", name), nil
	}, CategoryCustom)

	// 3. stop_agent
	stopDef := domain.ToolDefinition{
		Type: "function",
		Function: domain.ToolFunction{
			Name:        "stop_agent",
			Description: "Stop a currently running agent.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "The name of the agent to stop.",
					},
				},
				"required": []string{"name"},
			},
		},
	}
	captain.toolRegistry.Register(stopDef, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		name, _ := args["name"].(string)
		err := m.DisableCaptain(ctx, name)
		if err != nil {
			return nil, err
		}
		return fmt.Sprintf("Agent '%s' stopped successfully.", name), nil
	}, CategoryCustom)

	// 4. delegate_task
	delegateDef := domain.ToolDefinition{
		Type: "function",
		Function: domain.ToolFunction{
			Name:        "delegate_task",
			Description: "Delegate a specific task to a running agent.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"agent_name": map[string]interface{}{
						"type":        "string",
						"description": "The name of the running agent.",
					},
					"instruction": map[string]interface{}{
						"type":        "string",
						"description": "The full prompt/instruction for the task.",
					},
				},
				"required": []string{"agent_name", "instruction"},
			},
		},
	}
	captain.toolRegistry.Register(delegateDef, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		agentName, _ := args["agent_name"].(string)
		instruction, _ := args["instruction"].(string)
		return m.DispatchTask(ctx, agentName, instruction)
	}, CategoryCustom)
}

// RegisterCommanderTools is kept as a compatibility alias for older call sites.
func (m *SquadManager) RegisterCommanderTools(commander *Service) {
	m.RegisterCaptainTools(commander)
}
