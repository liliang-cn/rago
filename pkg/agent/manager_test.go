package agent

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
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

	concierge, err := manager.GetAgentByName("Concierge")
	if err != nil {
		t.Fatalf("get standalone concierge failed: %v", err)
	}
	if concierge.Kind != AgentKindAgent {
		t.Fatalf("expected Concierge standalone kind, got %q", concierge.Kind)
	}
	if len(concierge.Squads) != 0 {
		t.Fatalf("expected Concierge to be standalone, got squads=%+v", concierge.Squads)
	}
	if concierge.Description != "Always-on user entry agent for intake, status checks, and dispatching work." {
		t.Fatalf("unexpected Concierge description: %q", concierge.Description)
	}
	if !strings.Contains(concierge.Instructions, "inspect squad status") || !strings.Contains(concierge.Instructions, "submit squad work") {
		t.Fatalf("expected Concierge prompt to focus on orchestration, got %q", concierge.Instructions)
	}
	if concierge.EnableMCP || len(concierge.MCPTools) != 0 {
		t.Fatalf("expected Concierge to stay lightweight without default MCP tools, got enable_mcp=%v tools=%v", concierge.EnableMCP, concierge.MCPTools)
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
