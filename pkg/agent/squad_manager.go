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
	"github.com/liliang-cn/agent-go/pkg/services"
)

// SquadManager handles the lifecycle, discovery, and execution routing for squad agents.
type SquadManager struct {
	store          *Store
	runningAgents  map[string]context.CancelFunc // Tracks running agents if they are background loopers
	services       map[string]*Service           // Cached instantiated agent services
	mu             sync.RWMutex
	sessionMu      sync.Mutex
	memberSessions map[string]string
	queueMu        sync.Mutex
	taskQueues     map[string][]string
	sharedTasks    map[string]*SharedTask
	queueRunning   map[string]bool
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
	defaultSquadName                           = "AgentGo Squad"
	legacyDefaultSquadName                     = "Default Squad"
	defaultSquadDescription                    = "Default AgentGo squad."
)

// SharedTaskResult captures the outcome of one delegated squad agent call.
type SharedTaskResult struct {
	AgentName string `json:"agent_name"`
	Text      string `json:"text,omitempty"`
	Error     string `json:"error,omitempty"`
}

// SharedTask is a queued squad task owned by one squad lead agent.
type SharedTask struct {
	ID          string             `json:"id"`
	SquadID     string             `json:"squad_id"`
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

// SeedDefaultMembers seeds the built-in default squad and standalone agents.
func (m *SquadManager) SeedDefaultMembers() error {
	if _, err := m.ensureDefaultSquad(); err != nil {
		return err
	}

	ctx := context.Background()
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

	for _, builtin := range defaultBuiltInStandaloneAgents() {
		if err := m.ensureBuiltInStandaloneAgent(ctx, builtin); err != nil {
			return err
		}
	}
	if err := m.ensureDefaultSquadCaptain(ctx); err != nil {
		return err
	}
	if err := m.detachBuiltInStandaloneAgentsFromDefaultSquad(defaultAssistantAgentName, defaultStakeholderAgentName); err != nil {
		return err
	}
	return nil
}

func (m *SquadManager) ensureDefaultSquad() (*Squad, error) {
	squads, err := m.store.ListTeams()
	if err != nil {
		return nil, err
	}
	for _, squad := range squads {
		if squad.ID == defaultSquadID || strings.EqualFold(squad.Name, defaultSquadName) || strings.EqualFold(squad.Name, legacyDefaultSquadName) {
			updated := false
			if squad.ID != defaultSquadID {
				squad.ID = defaultSquadID
				updated = true
			}
			if !strings.EqualFold(squad.Name, defaultSquadName) {
				squad.Name = defaultSquadName
				updated = true
			}
			if strings.TrimSpace(squad.Description) == "" || strings.EqualFold(strings.TrimSpace(squad.Description), "Default workspace squad.") {
				squad.Description = defaultSquadDescription
				updated = true
			}
			if updated {
				squad.UpdatedAt = time.Now()
				if err := m.store.SaveTeam(squad); err != nil {
					return nil, err
				}
			}
			return squad, nil
		}
	}

	squad := &Squad{
		ID:          defaultSquadID,
		Name:        defaultSquadName,
		Description: defaultSquadDescription,
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
		store:          s,
		runningAgents:  make(map[string]context.CancelFunc),
		services:       make(map[string]*Service),
		memberSessions: make(map[string]string),
		taskQueues:     make(map[string][]string),
		sharedTasks:    make(map[string]*SharedTask),
		queueRunning:   make(map[string]bool),
	}
}

// NewTeamManager is kept as a compatibility alias for older call sites.
func NewTeamManager(s *Store) *TeamManager {
	return NewSquadManager(s)
}

// EnqueueSharedTask queues a squad task under one squad lead agent and returns an immediate acknowledgement.
func (m *SquadManager) EnqueueSharedTask(ctx context.Context, captainName string, agentNames []string, prompt string) (*SharedTask, error) {
	return m.EnqueueSharedTaskForSquad(ctx, "", captainName, agentNames, prompt)
}

// EnqueueSharedTaskForSquad queues a squad task for a specific squad and lead agent.
func (m *SquadManager) EnqueueSharedTaskForSquad(ctx context.Context, squadID, captainName string, agentNames []string, prompt string) (*SharedTask, error) {
	if strings.TrimSpace(prompt) == "" {
		return nil, fmt.Errorf("message required")
	}
	squad, captain, err := m.resolveSharedTaskContext(strings.TrimSpace(squadID), strings.TrimSpace(captainName))
	if err != nil {
		return nil, err
	}

	if len(agentNames) == 0 {
		agentNames = []string{captain.Name}
	}

	for _, name := range agentNames {
		member, memberErr := m.GetMemberByNameInSquad(name, squad.ID)
		if memberErr != nil {
			return nil, fmt.Errorf("cannot load squad agent %s: %w", name, memberErr)
		}
		if member.Kind == AgentKindCaptain && !strings.EqualFold(name, captain.Name) {
			return nil, fmt.Errorf("%s is also a squad lead agent and cannot be delegated from %s", name, captain.Name)
		}
	}

	now := time.Now()
	task := &SharedTask{
		ID:          uuid.New().String(),
		SquadID:     squad.ID,
		CaptainName: captain.Name,
		AgentNames:  append([]string(nil), agentNames...),
		Prompt:      strings.TrimSpace(prompt),
		Status:      SharedTaskStatusQueued,
		CreatedAt:   now,
	}

	m.queueMu.Lock()
	queuedAhead := len(m.taskQueues[squad.ID])
	if m.queueRunning[squad.ID] {
		queuedAhead++
	}
	task.QueuedAhead = queuedAhead
	task.AckMessage = buildSharedTaskAck(captain.Name, queuedAhead)
	m.sharedTasks[task.ID] = task
	m.taskQueues[squad.ID] = append(m.taskQueues[squad.ID], task.ID)
	shouldStartWorker := !m.queueRunning[squad.ID]
	if shouldStartWorker {
		m.queueRunning[squad.ID] = true
	}
	m.queueMu.Unlock()

	if shouldStartWorker {
		go m.runSharedTaskQueue(context.WithoutCancel(ctx), squad.ID)
	}

	return cloneSharedTask(task), nil
}

// ListSharedTasks returns recent queued or completed squad tasks for one captain.
func (m *SquadManager) ListSharedTasks(captainName string, since time.Time, limit int) []*SharedTask {
	return m.listSharedTasks("", captainName, since, limit)
}

// ListSharedTasksForSquad returns recent queued or completed squad tasks for one squad.
func (m *SquadManager) ListSharedTasksForSquad(squadID string, since time.Time, limit int) []*SharedTask {
	return m.listSharedTasks(squadID, "", since, limit)
}

func (m *SquadManager) listSharedTasks(squadID, captainName string, since time.Time, limit int) []*SharedTask {
	m.queueMu.Lock()
	defer m.queueMu.Unlock()

	tasks := make([]*SharedTask, 0, len(m.sharedTasks))
	for _, task := range m.sharedTasks {
		if squadID != "" && !strings.EqualFold(task.SquadID, squadID) {
			continue
		}
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

func (m *SquadManager) runSharedTaskQueue(ctx context.Context, squadID string) {
	for {
		task := m.nextQueuedTask(squadID)
		if task == nil {
			return
		}
		m.executeSharedTask(ctx, task)
	}
}

func (m *SquadManager) nextQueuedTask(squadID string) *SharedTask {
	m.queueMu.Lock()
	defer m.queueMu.Unlock()

	queue := m.taskQueues[squadID]
	if len(queue) == 0 {
		delete(m.queueRunning, squadID)
		return nil
	}

	taskID := queue[0]
	if len(queue) == 1 {
		delete(m.taskQueues, squadID)
	} else {
		m.taskQueues[squadID] = queue[1:]
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
			text, err := m.ChatWithMember(ctx, task.SquadID, agentName, task.Prompt)
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

// ListMembers returns all registered captains and specialists that belong to squads.
func (m *SquadManager) ListMembers() ([]*AgentModel, error) {
	all, err := m.store.ListAgentModels()
	if err != nil {
		return nil, err
	}
	members := make([]*AgentModel, 0, len(all))
	for _, model := range all {
		for _, membership := range model.Squads {
			members = append(members, cloneAgentForMembership(model, membership))
		}
	}
	return members, nil
}

// CreateMember persists a new squad agent configuration.
func (m *SquadManager) CreateMember(ctx context.Context, model *AgentModel) (*AgentModel, error) {
	if model == nil {
		return nil, fmt.Errorf("agent model is required")
	}
	squadID := strings.TrimSpace(model.TeamID)
	if squadID == "" {
		defaultSquad, err := m.ensureDefaultSquad()
		if err != nil {
			return nil, err
		}
		squadID = defaultSquad.ID
	}
	role := model.Kind
	if role == "" || role == AgentKindAgent {
		role = AgentKindSpecialist
	}
	model.TeamID = squadID
	model.Kind = role
	return m.CreateAgent(ctx, model)
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

	leadAgentName := defaultSquadLeadName(squad.Name)
	if existing, err := m.store.GetAgentModelByName(leadAgentName); err == nil {
		if _, err := m.JoinSquad(context.Background(), existing.Name, squad.ID, AgentKindCaptain); err != nil {
			return nil, err
		}
	} else {
		_, err := m.CreateMember(context.Background(), &AgentModel{
			ID:           uuid.New().String(),
			TeamID:       squad.ID,
			Name:         leadAgentName,
			Kind:         AgentKindCaptain,
			Description:  fmt.Sprintf("Default captain agent for %s.", squad.Name),
			Instructions: fmt.Sprintf("You are the captain agent for squad %s. Help directly when possible and coordinate specialists when useful.", squad.Name),
			MCPTools:     defaultMemberMCPTools("Captain"),
			EnableRAG:    true,
			EnableMemory: true,
			EnableMCP:    true,
		})
		if err != nil {
			return nil, err
		}
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

// GetMemberByName retrieves a persisted squad agent model by name.
func (m *SquadManager) GetMemberByName(name string) (*AgentModel, error) {
	model, err := m.store.GetAgentModelByName(strings.TrimSpace(name))
	if err != nil {
		return nil, err
	}
	if len(model.Squads) == 0 {
		return nil, fmt.Errorf("agent '%s' is not in a squad", model.Name)
	}
	if len(model.Squads) == 1 {
		return cloneAgentForMembership(model, model.Squads[0]), nil
	}
	for _, membership := range model.Squads {
		if membership.SquadID == defaultSquadID {
			return cloneAgentForMembership(model, membership), nil
		}
	}
	return cloneAgentForMembership(model, model.Squads[0]), nil
}

func (m *SquadManager) GetMemberByNameInSquad(name, squadID string) (*AgentModel, error) {
	model, err := m.store.GetAgentModelByName(strings.TrimSpace(name))
	if err != nil {
		return nil, err
	}
	squadID = strings.TrimSpace(squadID)
	for _, membership := range model.Squads {
		if membership.SquadID == squadID {
			return cloneAgentForMembership(model, membership), nil
		}
	}
	return nil, fmt.Errorf("agent '%s' is not in squad %s", model.Name, squadID)
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

	var agentgoCfg *config.Config
	builder := New(model.Name)

	if cfg, cfgErr := config.Load(""); cfgErr == nil {
		agentgoCfg = cfg
		if len(model.Squads) > 0 {
			builder.WithSystemPrompt(buildTeamSystemPrompt(cfg, model) + "\n\n" + buildTeamMemberPrompt(model))
		} else {
			builder.WithSystemPrompt(buildStandaloneAgentPrompt(cfg, model))
		}
	} else {
		if len(model.Squads) > 0 {
			builder.WithSystemPrompt(buildTeamMemberPrompt(model))
		} else {
			builder.WithSystemPrompt(strings.TrimSpace(model.Instructions))
		}
	}

	if agentgoCfg != nil {
		builder.WithConfig(agentgoCfg)

		globalPool := services.GetGlobalPoolService()
		if initErr := globalPool.Initialize(context.Background(), agentgoCfg); initErr == nil {
			if llmSvc, llmErr := globalPool.GetLLMServiceWithHint(selectionHintForAgentModel(model)); llmErr == nil {
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

	if label := configuredModelLabel(model); label != "" {
		newSvc.agent.SetModel(label)
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
	if model != nil && hasMembershipRole(model.Squads, AgentKindCaptain) {
		lines = append(lines,
			"- You are the captain for this squad.",
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
		"- Target squad agent: " + strings.TrimSpace(agentName),
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
	case "assistant", "captain", "stakeholder":
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
	_, err := m.store.GetAgentModelByName(name)
	return err
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

// DispatchTask runs the task on the target squad agent service directly.
func (m *SquadManager) DispatchTask(ctx context.Context, agentName string, instruction string) (string, error) {
	return m.dispatchTask(ctx, agentName, instruction, "")
}

// ChatWithMember runs a squad chat turn with persistent history scoped to one conversation key and squad agent.
func (m *SquadManager) ChatWithMember(ctx context.Context, conversationKey, agentName string, instruction string) (string, error) {
	conversationKey = strings.TrimSpace(conversationKey)
	if conversationKey == "" {
		return m.DispatchTask(ctx, agentName, instruction)
	}
	return m.dispatchTask(ctx, agentName, instruction, m.conversationSessionID(conversationKey, agentName))
}

func (m *SquadManager) dispatchTask(ctx context.Context, agentName string, instruction string, sessionID string) (string, error) {
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

	runOptions := dispatchRunOptions(agentName)
	if strings.TrimSpace(sessionID) != "" {
		runOptions = append(runOptions, WithSessionID(sessionID))
	}

	res, err := svc.Run(ctx, wrappedInstruction, runOptions...)
	if err != nil {
		return "", err
	}

	if text := extractDispatchText(res); text != "" {
		return text, nil
	}

	bz, _ := json.Marshal(res.FinalResult)
	return string(bz), nil
}

func (m *SquadManager) conversationSessionID(conversationKey, agentName string) string {
	key := strings.ToLower(strings.TrimSpace(conversationKey)) + "::" + strings.ToLower(strings.TrimSpace(agentName))

	m.sessionMu.Lock()
	defer m.sessionMu.Unlock()

	if sessionID, ok := m.memberSessions[key]; ok && strings.TrimSpace(sessionID) != "" {
		return sessionID
	}

	sessionID := uuid.NewString()
	m.memberSessions[key] = sessionID
	return sessionID
}

// DispatchTaskStream runs the task on the target agent and returns the raw event stream.
func (m *SquadManager) DispatchTaskStream(ctx context.Context, agentName string, instruction string) (<-chan *Event, error) {
	return m.dispatchTaskStream(ctx, agentName, instruction, "", nil)
}

// ChatWithMemberStream runs a squad chat turn with persistent history and returns the raw event stream.
func (m *SquadManager) ChatWithMemberStream(ctx context.Context, conversationKey, agentName, instruction string) (<-chan *Event, error) {
	conversationKey = strings.TrimSpace(conversationKey)
	if conversationKey == "" {
		return m.DispatchTaskStream(ctx, agentName, instruction)
	}
	return m.dispatchTaskStream(ctx, agentName, instruction, m.conversationSessionID(conversationKey, agentName), nil)
}

// DispatchTaskStreamWithOptions runs the task and returns the raw event stream with explicit run options.
func (m *SquadManager) DispatchTaskStreamWithOptions(ctx context.Context, agentName string, instruction string, opts ...RunOption) (<-chan *Event, error) {
	return m.dispatchTaskStream(ctx, agentName, instruction, "", opts)
}

// ChatWithMemberStreamWithOptions runs a squad chat turn with persistent history and explicit run options.
func (m *SquadManager) ChatWithMemberStreamWithOptions(ctx context.Context, conversationKey, agentName, instruction string, opts ...RunOption) (<-chan *Event, error) {
	conversationKey = strings.TrimSpace(conversationKey)
	if conversationKey == "" {
		return m.DispatchTaskStreamWithOptions(ctx, agentName, instruction, opts...)
	}
	return m.dispatchTaskStream(ctx, agentName, instruction, m.conversationSessionID(conversationKey, agentName), opts)
}

func (m *SquadManager) dispatchTaskStream(ctx context.Context, agentName string, instruction string, sessionID string, extraOpts []RunOption) (<-chan *Event, error) {
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

	runOptions := dispatchRunOptions(agentName)
	if strings.TrimSpace(sessionID) != "" {
		runOptions = append(runOptions, WithSessionID(sessionID))
	}
	runOptions = append(runOptions, extraOpts...)

	return svc.RunStreamWithOptions(ctx, wrappedInstruction, runOptions...)
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

// RegisterCaptainTools adds the squad management tools to the frontdesk lead agent.
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
			})
		}
		return result, nil
	}, CategoryCustom)

	// 2. delegate_task
	delegateDef := domain.ToolDefinition{
		Type: "function",
		Function: domain.ToolFunction{
			Name:        "delegate_task",
			Description: "Delegate a specific task to a squad agent.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"agent_name": map[string]interface{}{
						"type":        "string",
						"description": "The name of the target squad agent.",
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
