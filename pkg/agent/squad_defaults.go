package agent

import (
	"context"
	"strings"
	"time"
)

const (
	defaultConciergeAgentID     = "agent-concierge-001"
	defaultConciergeAgentName   = "Concierge"
	defaultCaptainAgentID       = "agent-captain-001"
	defaultCaptainAgentName     = "Captain"
	defaultAssistantAgentID     = "agent-assistant-001"
	defaultAssistantAgentName   = "Assistant"
	defaultOperatorAgentID      = "agent-operator-001"
	defaultOperatorAgentName    = "Operator"
	defaultStakeholderAgentID   = "agent-stakeholder-001"
	defaultStakeholderAgentName = "Stakeholder"
)

const (
	BuiltInConciergeAgentName = defaultConciergeAgentName
	BuiltInCaptainAgentName   = defaultCaptainAgentName
)

func defaultBuiltInStandaloneAgents() []*AgentModel {
	return []*AgentModel{
		{
			ID:           defaultConciergeAgentID,
			Name:         defaultConciergeAgentName,
			Kind:         AgentKindAgent,
			Description:  "Always-on user entry agent for intake, status checks, and dispatching work.",
			Instructions: "You are Concierge, the always-on intake agent for AgentGo. Accept user requests, clarify ambiguous asks, answer simple questions directly, and coordinate work across squads and agents when deeper execution is needed. Prefer lightweight orchestration: inspect squad status, inspect agent status, and use submit_squad_task or submit_agent_task for longer-running work instead of doing heavy execution yourself. For repository, filesystem, code generation, or web lookup tasks, submit the work to Assistant by default unless the user names a different agent or squad. Do not use sub-agent delegation or tool-catalog searching as a substitute for proper task submission. Keep replies concise, acknowledge queued work clearly, and never pretend background work is already finished. When the user asks for progress, use get_task_status or list_session_tasks.",
			EnableMemory: true,
		},
		{
			ID:           defaultAssistantAgentID,
			Name:         defaultAssistantAgentName,
			Kind:         AgentKindAgent,
			Description:  "A general-purpose standalone assistant agent for everyday requests.",
			Instructions: "You are Assistant, a general-purpose standalone agent. Help directly, stay pragmatic, and work independently unless a squad explicitly asks for your involvement.",
			MCPTools:     defaultMemberMCPTools(defaultAssistantAgentName),
			EnableRAG:    true,
			EnableMemory: true,
			EnableMCP:    true,
		},
		{
			ID:           defaultOperatorAgentID,
			Name:         defaultOperatorAgentName,
			Kind:         AgentKindAgent,
			Description:  "An execution-focused standalone operator for file work, environment checks, and runnable validation steps.",
			Instructions: "You are Operator, a standalone execution-focused agent. Specialize in doing operational work directly: inspecting files, writing files, validating generated artifacts, running concrete verification steps, and reporting factual outcomes concisely. You can manage generic PTY-backed command sessions for interactive CLIs, send follow-up input, interrupt running sessions, and inspect their output. For coding-agent CLIs such as Claude, Gemini, Codex, and OpenCode, always prefer the dedicated coding-agent tools first (start_coding_agent_session, send_coding_agent_prompt, get_coding_agent_session, list_coding_agent_sessions, interrupt_coding_agent_session, stop_coding_agent_session, run_coding_agent_once). Do not guess shell commands for those tools when a dedicated coding-agent tool fits. Prefer direct execution and verification over ideation. If a task needs product judgment or business prioritization, hand the decision back to the requester or the appropriate planning role instead of inventing it.",
			MCPTools:     defaultMemberMCPTools(defaultOperatorAgentName),
			EnableRAG:    true,
			EnableMemory: true,
			EnableMCP:    true,
		},
		{
			ID:           defaultStakeholderAgentID,
			Name:         defaultStakeholderAgentName,
			Kind:         AgentKindAgent,
			Description:  "Product/business representative for goals, scope, priorities, and acceptance criteria.",
			Instructions: "You are Stakeholder, a standalone stakeholder-representative agent. Work like a product manager or business representative. Clarify goals, priorities, constraints, trade-offs, risks, and acceptance criteria from a user and product perspective. Prefer requirement clarification, acceptance criteria, risk lists, and prioritization recommendations. Do not write code unless the user explicitly asks you to.",
			MCPTools:     defaultMemberMCPTools(defaultStakeholderAgentName),
			EnableRAG:    true,
			EnableMemory: true,
			EnableMCP:    true,
		},
	}
}

func defaultBuiltInCaptain() *AgentModel {
	return &AgentModel{
		ID:           defaultCaptainAgentID,
		Name:         defaultCaptainAgentName,
		Kind:         AgentKindAgent,
		Description:  "The built-in captain agent for AgentGo Squad. Coordinates squad work and handles shared tasks.",
		Instructions: "You are Captain, the built-in captain agent for AgentGo Squad. Handle direct squad requests when possible and coordinate specialists when that improves the result.",
		MCPTools:     defaultMemberMCPTools(defaultCaptainAgentName),
		EnableRAG:    true,
		EnableMemory: true,
		EnableMCP:    true,
	}
}

func defaultSquadLeadName(squadName string) string {
	name := strings.TrimSpace(squadName)
	if name == "" {
		return "Captain"
	}
	return name + " Captain"
}

func isAutoGeneratedSquadLeadName(squadName, agentName string) bool {
	squadName = strings.TrimSpace(squadName)
	agentName = strings.TrimSpace(agentName)
	if squadName == "" || agentName == "" {
		return false
	}
	return agentName == squadName+" Captain" || agentName == squadName+" Assistant"
}

func isBuiltInStandaloneAgentName(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case strings.ToLower(defaultConciergeAgentName), strings.ToLower(defaultAssistantAgentName), strings.ToLower(defaultOperatorAgentName), strings.ToLower(defaultStakeholderAgentName):
		return true
	default:
		return false
	}
}

func (m *SquadManager) ensureBuiltInStandaloneAgent(ctx context.Context, builtin *AgentModel) error {
	if builtin == nil {
		return nil
	}

	existing, err := m.store.GetAgentModelByName(builtin.Name)
	if err == nil {
		existing.Kind = AgentKindAgent
		existing.TeamID = ""
		existing.Description = builtin.Description
		existing.Instructions = builtin.Instructions
		existing.MCPTools = append([]string(nil), builtin.MCPTools...)
		existing.EnableRAG = builtin.EnableRAG
		existing.EnableMemory = builtin.EnableMemory
		existing.EnableMCP = builtin.EnableMCP
		existing.UpdatedAt = time.Now()
		if err := m.store.SaveAgentModel(existing); err != nil {
			return err
		}
		m.clearCachedAgent(existing.Name)
		return nil
	}

	_, err = m.CreateAgent(ctx, &AgentModel{
		ID:           builtin.ID,
		Name:         builtin.Name,
		Kind:         AgentKindAgent,
		Description:  builtin.Description,
		Instructions: builtin.Instructions,
		MCPTools:     append([]string(nil), builtin.MCPTools...),
		EnableRAG:    builtin.EnableRAG,
		EnableMemory: builtin.EnableMemory,
		EnableMCP:    builtin.EnableMCP,
	})
	return err
}

func (m *SquadManager) ensureDefaultSquadCaptain(ctx context.Context) error {
	captainBuiltin := defaultBuiltInCaptain()

	if err := m.ensureBuiltInStandaloneAgent(ctx, captainBuiltin); err != nil {
		return err
	}

	captain, err := m.store.GetAgentModelByName(defaultCaptainAgentName)
	if err != nil {
		return err
	}

	if err := m.store.SaveSquadMembership(&SquadMembership{
		AgentID: captain.ID,
		SquadID: defaultSquadID,
		Role:    AgentKindCaptain,
	}); err != nil {
		return err
	}

	m.clearCachedAgent(captain.Name)
	return nil
}

func (m *SquadManager) detachBuiltInStandaloneAgentsFromDefaultSquad(names ...string) error {
	for _, name := range names {
		model, err := m.store.GetAgentModelByName(strings.TrimSpace(name))
		if err != nil {
			continue
		}
		if err := m.store.DeleteSquadMembership(model.ID, defaultSquadID); err != nil {
			return err
		}
		m.clearCachedAgent(model.Name)
	}
	return nil
}
