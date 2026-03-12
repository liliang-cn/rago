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

func TestSeedDefaultMembersCreatesAssistantOnlyByDefault(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatalf("new store failed: %v", err)
	}
	manager := NewSquadManager(store)
	if err := manager.SeedDefaultMembers(); err != nil {
		t.Fatalf("seed default members failed: %v", err)
	}

	assistant, err := manager.GetMemberByName("Assistant")
	if err != nil {
		t.Fatalf("get default assistant failed: %v", err)
	}
	if assistant.Kind != AgentKindCaptain {
		t.Fatalf("expected assistant captain kind, got %q", assistant.Kind)
	}
	if assistant.Status != AgentStatusRunning {
		t.Fatalf("expected assistant to start enabled, got %q", assistant.Status)
	}

	if _, err := manager.GetMemberByName("Coder"); err == nil {
		t.Fatal("expected default squad to not seed Coder")
	}

	if _, err := manager.GetMemberByName("FileSystemAgent"); err == nil {
		t.Fatal("expected FileSystemAgent to be removed from the default squad")
	}
}

func TestSpecialistDoesNotSupportLifecycleState(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatalf("new store failed: %v", err)
	}
	manager := NewSquadManager(store)
	if err := manager.SeedDefaultMembers(); err != nil {
		t.Fatalf("seed default members failed: %v", err)
	}

	_, err = manager.CreateMember(context.Background(), &AgentModel{
		Name:         "Writer",
		Kind:         AgentKindSpecialist,
		Description:  "Writes files and summaries.",
		Instructions: "Write concise documents.",
		Status:       AgentStatusStopped,
	})
	if err != nil {
		t.Fatalf("create specialist failed: %v", err)
	}

	if err := manager.EnableCaptain(context.Background(), "Writer"); err == nil {
		t.Fatal("expected start specialist to fail")
	}
	if err := manager.DisableCaptain(context.Background(), "Writer"); err == nil {
		t.Fatal("expected stop specialist to fail")
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

	captain, err := manager.CreateMember(context.Background(), &AgentModel{
		Name:         "DocCaptain",
		Kind:         AgentKindCaptain,
		Description:  "Leads documentation work.",
		Instructions: "Coordinate documentation tasks.",
	})
	if err != nil {
		t.Fatalf("create captain failed: %v", err)
	}
	if !captain.EnableMCP || !captain.EnableRAG || !captain.EnableMemory {
		t.Fatalf("expected captain defaults to enable MCP/RAG/Memory, got %+v", captain)
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

	if _, err := manager.AddCaptain(context.Background(), squad.ID, "Docs Captain", "Leads docs work.", "Lead docs work."); err != nil {
		t.Fatalf("add captain failed: %v", err)
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

func TestEnqueueSharedTaskQueuesImmediately(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatalf("new store failed: %v", err)
	}
	manager := NewSquadManager(store)
	if err := manager.SeedDefaultMembers(); err != nil {
		t.Fatalf("seed default members failed: %v", err)
	}

	first, err := manager.EnqueueSharedTask(context.Background(), "Assistant", []string{"Assistant"}, "first task")
	if err != nil {
		t.Fatalf("enqueue first failed: %v", err)
	}
	second, err := manager.EnqueueSharedTask(context.Background(), "Assistant", []string{"Assistant"}, "second task")
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
		tasks := manager.ListSharedTasks("Assistant", time.Time{}, 10)
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

	tasks := manager.ListSharedTasks("Assistant", time.Time{}, 10)
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

	assistantPrompt := buildTeamMemberPrompt(&AgentModel{
		Name:         "Assistant",
		Instructions: "assistant base",
	})
	if !strings.Contains(assistantPrompt, "Delegate specialized implementation work to specialists") {
		t.Fatalf("assistant prompt missing coordinator guidance: %s", assistantPrompt)
	}
	if !strings.Contains(assistantPrompt, "Do not assume the work must go through MCP") {
		t.Fatalf("assistant prompt missing non-MCP capability rule: %s", assistantPrompt)
	}
	if !strings.Contains(assistantPrompt, "Never inspect blacklisted repository paths unless the user explicitly asks for them.") {
		t.Fatalf("assistant prompt missing repo-analysis rule: %s", assistantPrompt)
	}
	if !strings.Contains(assistantPrompt, "read those files first before calling list_directory") {
		t.Fatalf("assistant prompt missing direct-file-read rule: %s", assistantPrompt)
	}
	if strings.Contains(assistantPrompt, "MUST call a filesystem write or modify tool") {
		t.Fatalf("assistant prompt should not include coder file-writing rules: %s", assistantPrompt)
	}

	teamPrompt := buildTeamSystemPrompt(cfg, &AgentModel{Name: "Assistant"})
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
	if !strings.Contains(teamPrompt, "default captain for the squad") {
		t.Fatalf("squad prompt missing captain guidance: %s", teamPrompt)
	}
}

func TestDefaultMemberMCPTools_AssistantIncludesFilesystemAndWebTools(t *testing.T) {
	tools := defaultMemberMCPTools("Assistant")
	if len(tools) == 0 {
		t.Fatal("expected assistant default tool allowlist")
	}
	if containsStr(tools, "mcp_filesystem_tree") {
		t.Fatalf("assistant should not have tree by default: %v", tools)
	}
	if containsStr(tools, "mcp_filesystem_search_files") {
		t.Fatalf("assistant should not have broad file search by default: %v", tools)
	}
	if !containsStr(tools, "mcp_filesystem_read_file") {
		t.Fatalf("assistant should keep targeted file reads: %v", tools)
	}
	if !containsStr(tools, "mcp_filesystem_write_file") {
		t.Fatalf("assistant should have file write capability: %v", tools)
	}
	if !containsStr(tools, "mcp_websearch_websearch_ai_summary") {
		t.Fatalf("assistant should have websearch capability: %v", tools)
	}
}

func TestBuildTeamTaskEnvelope_ContainsSharedContext(t *testing.T) {
	cfg := &config.Config{Home: "/tmp/agentgo-test"}
	t.Setenv("SHELL", "/bin/test-shell")

	envelope := buildTeamTaskEnvelope(cfg, "Assistant", "summarize the repo flow")
	if !strings.Contains(envelope, "Squad task context:") {
		t.Fatalf("missing squad task header: %s", envelope)
	}
	if !strings.Contains(envelope, "Target squad member: Assistant") {
		t.Fatalf("missing target squad member: %s", envelope)
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
