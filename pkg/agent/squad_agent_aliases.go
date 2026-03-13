package agent

import "context"

// AddLeadAgent is the generic public API for adding a lead-role agent directly into a squad.
func (m *SquadManager) AddLeadAgent(ctx context.Context, squadID, name, description, instructions string) (*AgentModel, error) {
	return m.AddCaptain(ctx, squadID, name, description, instructions)
}

// ListLeadAgents is the generic public API for listing all lead-role squad agents.
func (m *SquadManager) ListLeadAgents() ([]*AgentModel, error) {
	return m.ListCaptains()
}

// AddSquadAgent is the generic public API for adding an agent directly into a squad with a role.
func (m *SquadManager) AddSquadAgent(ctx context.Context, squadID, name string, role AgentKind, description, instructions string) (*AgentModel, error) {
	if role == "" {
		role = AgentKindSpecialist
	}
	return m.CreateMember(ctx, &AgentModel{
		TeamID:       squadID,
		Name:         name,
		Kind:         role,
		Description:  description,
		Instructions: instructions,
	})
}

// CreateSquadAgent is the public alias for creating an agent directly inside a squad.
func (m *SquadManager) CreateSquadAgent(ctx context.Context, model *AgentModel) (*AgentModel, error) {
	return m.CreateMember(ctx, model)
}

// ListSquadAgents is the public alias for listing all agents that currently belong to squads.
func (m *SquadManager) ListSquadAgents() ([]*AgentModel, error) {
	return m.ListMembers()
}

// GetSquadAgentByName is the public alias for resolving a squad agent by name.
func (m *SquadManager) GetSquadAgentByName(name string) (*AgentModel, error) {
	return m.GetMemberByName(name)
}
