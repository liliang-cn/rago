package agent

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/liliang-cn/agent-go/pkg/config"
	"github.com/liliang-cn/agent-go/pkg/domain"
)

func TestSanitizeDispatchText(t *testing.T) {
	got := sanitizeDispatchText("<think>internal reasoning</think>\n\nFinal answer")
	if got != "Final answer" {
		t.Fatalf("expected thinking tags to be removed, got %q", got)
	}
}

func TestNewStoreAppliesSQLitePragmas(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatalf("new store failed: %v", err)
	}
	defer store.db.Close()

	var journalMode string
	if err := store.db.QueryRow(`PRAGMA journal_mode;`).Scan(&journalMode); err != nil {
		t.Fatalf("query journal_mode failed: %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("expected WAL mode, got %q", journalMode)
	}

	var busyTimeout int
	if err := store.db.QueryRow(`PRAGMA busy_timeout;`).Scan(&busyTimeout); err != nil {
		t.Fatalf("query busy_timeout failed: %v", err)
	}
	if busyTimeout < sqliteBusyTimeoutMillis {
		t.Fatalf("expected busy_timeout >= %d, got %d", sqliteBusyTimeoutMillis, busyTimeout)
	}
}

func TestSeedDefaultMembersCreatesBuiltInsByDefault(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatalf("new store failed: %v", err)
	}
	manager := NewSquadManager(store)
	if err := manager.SeedDefaultMembers(); err != nil {
		t.Fatalf("seed default members failed: %v", err)
	}

	captain, err := manager.GetMemberByName("Captain")
	if err != nil {
		t.Fatalf("get default captain failed: %v", err)
	}
	if captain.Kind != AgentKindCaptain {
		t.Fatalf("expected captain kind, got %q", captain.Kind)
	}

	assistant, err := manager.GetAgentByName("Assistant")
	if err != nil {
		t.Fatalf("get standalone assistant failed: %v", err)
	}
	if assistant.Kind != AgentKindAgent {
		t.Fatalf("expected Assistant standalone kind, got %q", assistant.Kind)
	}
	if len(assistant.Squads) != 0 {
		t.Fatalf("expected Assistant to be standalone, got squads=%+v", assistant.Squads)
	}

	stakeholder, err := manager.GetAgentByName("Stakeholder")
	if err != nil {
		t.Fatalf("get standalone stakeholder failed: %v", err)
	}
	if stakeholder.Kind != AgentKindAgent {
		t.Fatalf("expected Stakeholder standalone kind, got %q", stakeholder.Kind)
	}
	if len(stakeholder.Squads) != 0 {
		t.Fatalf("expected Stakeholder to be standalone, got squads=%+v", stakeholder.Squads)
	}
	if stakeholder.Description != "Product/business representative for goals, scope, priorities, and acceptance criteria." {
		t.Fatalf("unexpected Stakeholder description: %q", stakeholder.Description)
	}
	if !strings.Contains(stakeholder.Instructions, "product manager or business representative") {
		t.Fatalf("expected Stakeholder prompt to include PM/business framing, got %q", stakeholder.Instructions)
	}
	if !strings.Contains(stakeholder.Instructions, "Do not write code unless the user explicitly asks you to") {
		t.Fatalf("expected Stakeholder prompt to discourage direct coding, got %q", stakeholder.Instructions)
	}
	if !strings.Contains(stakeholder.Instructions, "acceptance criteria") || !strings.Contains(stakeholder.Instructions, "risk lists") || !strings.Contains(stakeholder.Instructions, "prioritization recommendations") {
		t.Fatalf("expected Stakeholder prompt to prioritize product outputs, got %q", stakeholder.Instructions)
	}

	if _, err := manager.GetMemberByName("Coder"); err == nil {
		t.Fatal("expected default squad to not seed Coder")
	}

	if _, err := manager.GetMemberByName("FileSystemAgent"); err == nil {
		t.Fatal("expected FileSystemAgent to be removed from the default squad")
	}
}

func TestCreateMemberAppliesUsefulDefaults(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatalf("new store failed: %v", err)
	}
	manager := NewSquadManager(store)
	if err := manager.SeedDefaultMembers(); err != nil {
		t.Fatalf("seed default members failed: %v", err)
	}

	writer, err := manager.CreateMember(context.Background(), &AgentModel{
		Name:         "Writer",
		Kind:         AgentKindSpecialist,
		Description:  "Writes concise docs.",
		Instructions: "Write concise docs.",
	})
	if err != nil {
		t.Fatalf("create specialist failed: %v", err)
	}
	if !writer.EnableMCP {
		t.Fatal("expected created member to enable MCP by default")
	}
	if len(writer.MCPTools) == 0 {
		t.Fatal("expected created member to receive default MCP tools")
	}

	if _, err := manager.CreateMember(context.Background(), &AgentModel{
		Name:         "DocCaptain",
		TeamID:       "docs-squad-test",
		Kind:         AgentKindCaptain,
		Description:  "Leads documentation work.",
		Instructions: "Coordinate documentation tasks.",
	}); err == nil {
		t.Fatalf("expected unknown squad creation to fail")
	}

	squad, err := manager.CreateSquad(context.Background(), &Squad{
		Name:        "Docs Squad",
		Description: "Documentation squad.",
	})
	if err != nil {
		t.Fatalf("create squad failed: %v", err)
	}

	docsMember, err := manager.CreateMember(context.Background(), &AgentModel{
		Name:         "DocWriter",
		TeamID:       squad.ID,
		Kind:         AgentKindSpecialist,
		Description:  "Writes documentation.",
		Instructions: "Write documentation.",
	})
	if err != nil {
		t.Fatalf("create specialist in new squad failed: %v", err)
	}
	if !docsMember.EnableMCP || !docsMember.EnableRAG || !docsMember.EnableMemory {
		t.Fatalf("expected squad member defaults to enable MCP/RAG/Memory, got %+v", docsMember)
	}
}

func TestCreateAgentCreatesStandaloneAgent(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatalf("new store failed: %v", err)
	}
	manager := NewSquadManager(store)
	if err := manager.SeedDefaultMembers(); err != nil {
		t.Fatalf("seed default members failed: %v", err)
	}

	model, err := manager.CreateAgent(context.Background(), &AgentModel{
		Name:         "Writer",
		Description:  "Writes standalone notes.",
		Instructions: "Write concise standalone notes.",
	})
	if err != nil {
		t.Fatalf("create agent failed: %v", err)
	}
	if model.Kind != AgentKindAgent {
		t.Fatalf("expected standalone kind agent, got %q", model.Kind)
	}
	if model.TeamID != "" {
		t.Fatalf("expected standalone agent to have no squad, got %q", model.TeamID)
	}

	members, err := manager.ListMembers()
	if err != nil {
		t.Fatalf("list members failed: %v", err)
	}
	for _, member := range members {
		if member.Name == "Writer" {
			t.Fatalf("expected standalone agent to be excluded from squad members: %+v", member)
		}
	}
}

func TestJoinAndLeaveSquadMovesStandaloneAgent(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatalf("new store failed: %v", err)
	}
	manager := NewSquadManager(store)
	if err := manager.SeedDefaultMembers(); err != nil {
		t.Fatalf("seed default members failed: %v", err)
	}

	writer, err := manager.CreateAgent(context.Background(), &AgentModel{
		Name:         "Writer",
		Description:  "Writes docs.",
		Instructions: "Write docs.",
	})
	if err != nil {
		t.Fatalf("create standalone agent failed: %v", err)
	}

	joined, err := manager.JoinSquad(context.Background(), writer.Name, defaultSquadID, AgentKindSpecialist)
	if err != nil {
		t.Fatalf("join squad failed: %v", err)
	}
	if joined.TeamID != defaultSquadID {
		t.Fatalf("expected joined squad id %q, got %q", defaultSquadID, joined.TeamID)
	}
	if joined.Kind != AgentKindSpecialist {
		t.Fatalf("expected joined kind specialist, got %q", joined.Kind)
	}
	if _, err := manager.GetMemberByName(writer.Name); err != nil {
		t.Fatalf("expected joined agent to be loadable as member: %v", err)
	}

	left, err := manager.LeaveSquad(context.Background(), writer.Name)
	if err != nil {
		t.Fatalf("leave squad failed: %v", err)
	}
	if left.TeamID != "" {
		t.Fatalf("expected agent to leave squad, got %q", left.TeamID)
	}
	if left.Kind != AgentKindAgent {
		t.Fatalf("expected standalone kind agent after leave, got %q", left.Kind)
	}
	if _, err := manager.GetMemberByName(writer.Name); err == nil {
		t.Fatal("expected standalone agent to no longer be treated as squad member")
	}
}

func TestLastCaptainCannotLeaveOrBeDeleted(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatalf("new store failed: %v", err)
	}
	manager := NewSquadManager(store)
	if err := manager.SeedDefaultMembers(); err != nil {
		t.Fatalf("seed default members failed: %v", err)
	}

	if _, err := manager.LeaveSquad(context.Background(), "Captain"); err == nil {
		t.Fatal("expected last captain leave to fail")
	}
	if err := manager.DeleteAgent(context.Background(), "Captain"); err == nil {
		t.Fatal("expected last captain delete to fail")
	}
}

func TestGetSquadStatusIdleByDefault(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatalf("new store failed: %v", err)
	}
	manager := NewSquadManager(store)
	if err := manager.SeedDefaultMembers(); err != nil {
		t.Fatalf("seed default members failed: %v", err)
	}

	status, err := manager.GetSquadStatus(defaultSquadID)
	if err != nil {
		t.Fatalf("get squad status failed: %v", err)
	}
	if status.Status != "idle" {
		t.Fatalf("expected idle status, got %q", status.Status)
	}
	if status.RunningTasks != 0 || status.QueuedTasks != 0 {
		t.Fatalf("expected no active tasks, got running=%d queued=%d", status.RunningTasks, status.QueuedTasks)
	}
}

func TestGetSquadStatusAggregatesRunningAndQueuedTasks(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatalf("new store failed: %v", err)
	}
	manager := NewSquadManager(store)
	if err := manager.SeedDefaultMembers(); err != nil {
		t.Fatalf("seed default members failed: %v", err)
	}

	manager.queueMu.Lock()
	manager.sharedTasks["task-running"] = &SharedTask{
		ID:          "task-running",
		SquadID:     defaultSquadID,
		CaptainName: "Captain",
		Status:      SharedTaskStatusRunning,
	}
	manager.sharedTasks["task-queued"] = &SharedTask{
		ID:          "task-queued",
		SquadID:     defaultSquadID,
		CaptainName: "Captain",
		Status:      SharedTaskStatusQueued,
	}
	manager.queueRunning[defaultSquadID] = true
	manager.queueMu.Unlock()

	status, err := manager.GetSquadStatus(defaultSquadID)
	if err != nil {
		t.Fatalf("get squad status failed: %v", err)
	}
	if status.Status != "running" {
		t.Fatalf("expected running status, got %q", status.Status)
	}
	if status.RunningTasks != 1 {
		t.Fatalf("expected 1 running task, got %d", status.RunningTasks)
	}
	if status.QueuedTasks != 1 {
		t.Fatalf("expected 1 queued task, got %d", status.QueuedTasks)
	}
	if len(status.ActiveTaskIDs) != 1 || status.ActiveTaskIDs[0] != "task-running" {
		t.Fatalf("unexpected active task ids: %+v", status.ActiveTaskIDs)
	}
	if len(status.RunningCaptains) != 1 || status.RunningCaptains[0] != "Captain" {
		t.Fatalf("unexpected running captains: %+v", status.RunningCaptains)
	}
}

func TestSquadHelpersExposeSquadStyleAPI(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatalf("new store failed: %v", err)
	}
	manager := NewSquadManager(store)
	if err := manager.SeedDefaultMembers(); err != nil {
		t.Fatalf("seed default members failed: %v", err)
	}

	squad, err := manager.CreateSquad(context.Background(), &Squad{
		Name:        "Docs Squad",
		Description: "Handles documentation tasks.",
	})
	if err != nil {
		t.Fatalf("create squad failed: %v", err)
	}

	loaded, err := manager.GetSquadByName("docs squad")
	if err != nil {
		t.Fatalf("get squad by name failed: %v", err)
	}
	if loaded.ID != squad.ID {
		t.Fatalf("expected same squad ID, got %s want %s", loaded.ID, squad.ID)
	}

	if _, err := manager.AddSpecialist(context.Background(), squad.ID, "Writer", "Writes docs.", "Write docs."); err != nil {
		t.Fatalf("add specialist failed: %v", err)
	}

	captains, err := manager.ListCaptains()
	if err != nil {
		t.Fatalf("list captains failed: %v", err)
	}
	specialists, err := manager.ListSpecialists()
	if err != nil {
		t.Fatalf("list specialists failed: %v", err)
	}
	if len(captains) < 2 {
		t.Fatalf("expected at least 2 captains, got %d", len(captains))
	}
	if len(specialists) < 1 {
		t.Fatalf("expected at least 1 specialist, got %d", len(specialists))
	}
}

func TestConversationSessionIDIsStablePerConversationAndMember(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatalf("new store failed: %v", err)
	}
	manager := NewSquadManager(store)

	first := manager.conversationSessionID("squad-chat-1", "Captain")
	second := manager.conversationSessionID("squad-chat-1", "Captain")
	otherMember := manager.conversationSessionID("squad-chat-1", "Writer")
	otherConversation := manager.conversationSessionID("squad-chat-2", "Captain")

	if first == "" {
		t.Fatal("expected non-empty session id")
	}
	if first != second {
		t.Fatalf("expected stable session id, got %q and %q", first, second)
	}
	if first == otherMember {
		t.Fatal("expected different members to get different session ids")
	}
	if first == otherConversation {
		t.Fatal("expected different conversations to get different session ids")
	}
}

func TestEnqueueSharedTaskQueuesImmediately(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatalf("new store failed: %v", err)
	}
	manager := NewSquadManager(store)
	if err := manager.SeedDefaultMembers(); err != nil {
		t.Fatalf("seed default members failed: %v", err)
	}

	first, err := manager.EnqueueSharedTask(context.Background(), "Captain", []string{"Captain"}, "first task")
	if err != nil {
		t.Fatalf("enqueue first failed: %v", err)
	}
	second, err := manager.EnqueueSharedTask(context.Background(), "Captain", []string{"Captain"}, "second task")
	if err != nil {
		t.Fatalf("enqueue second failed: %v", err)
	}

	if first.Status != SharedTaskStatusQueued && first.Status != SharedTaskStatusRunning {
		t.Fatalf("expected first task queued or running, got %q", first.Status)
	}
	if second.QueuedAhead < 1 {
		t.Fatalf("expected second task to see queue ahead, got %d", second.QueuedAhead)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		tasks := manager.ListSharedTasks("Captain", time.Time{}, 10)
		var completed int
		for _, task := range tasks {
			if task.Status == SharedTaskStatusCompleted || task.Status == SharedTaskStatusFailed {
				completed++
			}
		}
		if completed >= 2 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	tasks := manager.ListSharedTasks("Captain", time.Time{}, 10)
	t.Fatalf("expected queued tasks to complete, got %+v", tasks)
}

func TestBuildTeamMemberPrompt_IsAgentSpecific(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/agentgo-test"}
	t.Setenv("SHELL", "/bin/test-shell")

	coderPrompt := buildTeamMemberPrompt(&AgentModel{
		Name:         "Coder",
		Instructions: "coder base",
	})
	if !strings.Contains(coderPrompt, "MUST call a filesystem write or modify tool") {
		t.Fatalf("coder prompt missing file-writing rule: %s", coderPrompt)
	}
	if !strings.Contains(coderPrompt, "they are not the only possible path") {
		t.Fatalf("coder prompt missing current-runtime capability rule: %s", coderPrompt)
	}
	if strings.Contains(coderPrompt, "Never recursively inspect large dependency or build directories") {
		t.Fatalf("coder prompt should not include researcher repo-analysis rules: %s", coderPrompt)
	}
	if strings.Contains(coderPrompt, "Shared writable workspace") {
		t.Fatalf("coder member prompt should not include squad workspace context: %s", coderPrompt)
	}

	captainPrompt := buildTeamMemberPrompt(&AgentModel{
		Name:         "Captain",
		Instructions: "captain base",
	})
	if !strings.Contains(captainPrompt, "Delegate specialized implementation work to specialists") {
		t.Fatalf("captain prompt missing coordinator guidance: %s", captainPrompt)
	}
	if !strings.Contains(captainPrompt, "Do not assume the work must go through MCP") {
		t.Fatalf("captain prompt missing non-MCP capability rule: %s", captainPrompt)
	}
	if !strings.Contains(captainPrompt, "Never inspect blacklisted repository paths unless the user explicitly asks for them.") {
		t.Fatalf("captain prompt missing repo-analysis rule: %s", captainPrompt)
	}
	if !strings.Contains(captainPrompt, "read those files first before calling list_directory") {
		t.Fatalf("captain prompt missing direct-file-read rule: %s", captainPrompt)
	}
	if strings.Contains(captainPrompt, "MUST call a filesystem write or modify tool") {
		t.Fatalf("captain prompt should not include coder file-writing rules: %s", captainPrompt)
	}

	teamPrompt := buildTeamSystemPrompt(cfg, &AgentModel{Name: "Captain", Squads: []SquadMembership{{SquadID: defaultSquadID, Role: AgentKindCaptain}}})
	if !strings.Contains(teamPrompt, "Shared writable workspace") {
		t.Fatalf("squad prompt missing workspace context: %s", teamPrompt)
	}
	if !strings.Contains(teamPrompt, "Active project root") {
		t.Fatalf("squad prompt missing project root context: %s", teamPrompt)
	}
	if !strings.Contains(teamPrompt, "Runtime OS/arch: "+runtime.GOOS+"/"+runtime.GOARCH) {
		t.Fatalf("squad prompt missing runtime context: %s", teamPrompt)
	}
	if !strings.Contains(teamPrompt, "Interactive shell: /bin/test-shell") {
		t.Fatalf("squad prompt missing shell context: %s", teamPrompt)
	}
	if !strings.Contains(teamPrompt, "captain for this squad") {
		t.Fatalf("squad prompt missing captain guidance: %s", teamPrompt)
	}
}

func TestDefaultMemberMCPTools_CaptainIncludesFilesystemAndWebTools(t *testing.T) {
	tools := defaultMemberMCPTools("Captain")
	if len(tools) == 0 {
		t.Fatal("expected captain default tool allowlist")
	}
	if containsStr(tools, "mcp_filesystem_tree") {
		t.Fatalf("captain should not have tree by default: %v", tools)
	}
	if containsStr(tools, "mcp_filesystem_search_files") {
		t.Fatalf("captain should not have broad file search by default: %v", tools)
	}
	if !containsStr(tools, "mcp_filesystem_read_file") {
		t.Fatalf("captain should keep targeted file reads: %v", tools)
	}
	if !containsStr(tools, "mcp_filesystem_write_file") {
		t.Fatalf("captain should have file write capability: %v", tools)
	}
	if !containsStr(tools, "mcp_websearch_websearch_ai_summary") {
		t.Fatalf("captain should have websearch capability: %v", tools)
	}
}

func TestBuildTeamTaskEnvelope_ContainsSharedContext(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/agentgo-test"}
	t.Setenv("SHELL", "/bin/test-shell")

	envelope := buildTeamTaskEnvelope(cfg, "Assistant", "summarize the repo flow")
	if !strings.Contains(envelope, "Squad task context:") {
		t.Fatalf("missing squad task header: %s", envelope)
	}
	if !strings.Contains(envelope, "Target squad agent: Assistant") {
		t.Fatalf("missing target squad agent: %s", envelope)
	}
	if !strings.Contains(envelope, "Shared writable workspace: /tmp/agentgo-test/workspace") {
		t.Fatalf("missing workspace context: %s", envelope)
	}
	if !strings.Contains(envelope, "Active project root:") {
		t.Fatalf("missing project root context: %s", envelope)
	}
	if !strings.Contains(envelope, "Runtime OS/arch: "+runtime.GOOS+"/"+runtime.GOARCH) {
		t.Fatalf("missing runtime context: %s", envelope)
	}
	if !strings.Contains(envelope, "Interactive shell: /bin/test-shell") {
		t.Fatalf("missing shell context: %s", envelope)
	}
	if !strings.Contains(envelope, "The bullets above are context, not the requested action.") {
		t.Fatalf("missing context-vs-task guardrail: %s", envelope)
	}
	if !strings.Contains(envelope, "Ignore blacklisted repository paths unless the user explicitly asks for them:") {
		t.Fatalf("missing blacklist guardrail: %s", envelope)
	}
	if !strings.Contains(envelope, "\nTask:\nsummarize the repo flow") {
		t.Fatalf("missing original task body: %s", envelope)
	}
}

func TestFilterToolDefinitionsForAgent_RespectsAllowlist(t *testing.T) {
	svc := &Service{}
	currentAgent := NewAgentWithConfig("Assistant", "assistant", nil)
	currentAgent.SetAllowedMCPTools([]string{
		"mcp_filesystem_read_file",
		"mcp_filesystem_list_directory",
	})
	currentAgent.SetAllowedSkills([]string{"repo_map"})

	defs := []domain.ToolDefinition{
		{Function: domain.ToolFunction{Name: "mcp_filesystem_read_file"}},
		{Function: domain.ToolFunction{Name: "mcp_filesystem_tree"}},
		{Function: domain.ToolFunction{Name: "skill_repo_map"}},
		{Function: domain.ToolFunction{Name: "skill_shell"}},
		{Function: domain.ToolFunction{Name: "search_available_tools"}},
	}

	filtered := svc.filterToolDefinitionsForAgent(currentAgent, defs)
	names := make([]string, 0, len(filtered))
	for _, def := range filtered {
		names = append(names, def.Function.Name)
	}

	if !containsStr(names, "mcp_filesystem_read_file") {
		t.Fatalf("expected allowed mcp tool to remain: %v", names)
	}
	if containsStr(names, "mcp_filesystem_tree") {
		t.Fatalf("expected disallowed mcp tool to be filtered: %v", names)
	}
	if !containsStr(names, "skill_repo_map") {
		t.Fatalf("expected allowed skill to remain: %v", names)
	}
	if containsStr(names, "skill_shell") {
		t.Fatalf("expected disallowed skill to be filtered: %v", names)
	}
	if !containsStr(names, "search_available_tools") {
		t.Fatalf("expected search_available_tools to remain: %v", names)
	}
}

func TestIsBlacklistedRepositoryPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{path: "/repo/ui/node_modules/react/index.js", want: true},
		{path: "/repo/rust/target/debug/app", want: true},
		{path: "/repo/.git/config", want: true},
		{path: "/repo/pkg/agent/team_manager.go", want: false},
		{path: "/repo/ui/src/App.tsx", want: false},
	}

	for _, tc := range cases {
		if got := IsBlacklistedRepositoryPath(tc.path); got != tc.want {
			t.Fatalf("IsBlacklistedRepositoryPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestDispatchRunOptions(t *testing.T) {
	coder := dispatchRunOptions("Coder")
	if len(coder) != 2 {
		t.Fatalf("expected coder-specific options, got %d", len(coder))
	}

	assistant := dispatchRunOptions("Assistant")
	if len(assistant) != 2 {
		t.Fatalf("expected tuned dispatch options for assistant, got %d", len(assistant))
	}
}
