package agent

import (
	"fmt"
	"strings"
	"time"
)

func (m *SquadManager) resolveSharedTaskContext(squadID, captainName string) (*Squad, *AgentModel, error) {
	if squadID != "" {
		squad, err := m.store.GetTeam(squadID)
		if err != nil {
			return nil, nil, err
		}
		leadAgent, err := m.GetLeadAgentForSquad(squadID)
		if err != nil {
			return nil, nil, err
		}
		if captainName != "" && !strings.EqualFold(captainName, leadAgent.Name) {
			return nil, nil, fmt.Errorf("%s is not the lead agent for squad %s", captainName, squad.Name)
		}
		return squad, leadAgent, nil
	}

	if captainName == "" {
		captainName = defaultCaptainAgentName
	}

	model, err := m.store.GetAgentModelByName(captainName)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot load squad lead agent %s: %w", captainName, err)
	}

	leadMemberships := leadMemberships(model.Squads)
	if len(leadMemberships) == 0 {
		return nil, nil, fmt.Errorf("%s is not a squad lead agent", captainName)
	}

	var membership SquadMembership
	switch {
	case len(leadMemberships) == 1:
		membership = leadMemberships[0]
	default:
		for _, candidate := range leadMemberships {
			if candidate.SquadID == defaultSquadID {
				membership = candidate
				break
			}
		}
		if strings.TrimSpace(membership.SquadID) == "" {
			return nil, nil, fmt.Errorf("%s leads multiple squads; specify squad id", captainName)
		}
	}

	squad, err := m.store.GetTeam(membership.SquadID)
	if err != nil {
		return nil, nil, err
	}
	return squad, cloneAgentForMembership(model, membership), nil
}

func (m *SquadManager) isSquadTaskActive(squadID string) bool {
	m.queueMu.Lock()
	defer m.queueMu.Unlock()

	for _, task := range m.sharedTasks {
		if task.SquadID != squadID {
			continue
		}
		if task.Status == SharedTaskStatusQueued || task.Status == SharedTaskStatusRunning {
			return true
		}
	}
	return false
}

func pruneSharedTaskResults(tasks []*SharedTask, since time.Time) []*SharedTask {
	if since.IsZero() {
		return tasks
	}
	out := make([]*SharedTask, 0, len(tasks))
	for _, task := range tasks {
		if task == nil {
			continue
		}
		if task.CreatedAt.After(since) || (task.FinishedAt != nil && task.FinishedAt.After(since)) {
			out = append(out, task)
		}
	}
	return out
}
