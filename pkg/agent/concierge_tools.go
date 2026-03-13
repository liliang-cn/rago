package agent

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (m *SquadManager) RegisterConciergeTools(concierge *Service) {
	if concierge == nil {
		return
	}

	register := func(name, description string, parameters map[string]interface{}, handler func(context.Context, map[string]interface{}) (interface{}, error)) {
		if concierge.toolRegistry != nil && concierge.toolRegistry.Has(name) {
			return
		}
		concierge.AddTool(name, description, parameters, handler)
	}

	register("list_squads", "List all squads with their current runtime status.", map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		statuses, err := m.ListSquadStatuses()
		if err != nil {
			return nil, err
		}
		out := make([]map[string]interface{}, 0, len(statuses))
		for _, status := range statuses {
			out = append(out, map[string]interface{}{
				"squad_id":        status.SquadID,
				"name":            status.Name,
				"description":     status.Description,
				"status":          status.Status,
				"agent_count":     status.AgentCount,
				"captains":        append([]string(nil), status.CaptainNames...),
				"running_tasks":   status.RunningTasks,
				"queued_tasks":    status.QueuedTasks,
				"active_task_ids": append([]string(nil), status.ActiveTaskIDs...),
			})
		}
		return out, nil
	})

	register("get_squad_status", "Get the runtime status of one squad.", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"squad_id":   map[string]interface{}{"type": "string"},
			"squad_name": map[string]interface{}{"type": "string"},
		},
	}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		squad, err := m.resolveSquadRef(getStringArg(args, "squad_id"), getStringArg(args, "squad_name"))
		if err != nil {
			return nil, err
		}
		return m.GetSquadStatus(squad.ID)
	})

	register("list_agents", "List all known agents with their runtime status.", map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		statuses, err := m.ListAgentStatuses()
		if err != nil {
			return nil, err
		}
		out := make([]map[string]interface{}, 0, len(statuses))
		for _, status := range statuses {
			out = append(out, map[string]interface{}{
				"agent_id":           status.AgentID,
				"name":               status.Name,
				"kind":               status.Kind,
				"status":             status.Status,
				"built_in":           status.BuiltIn,
				"preferred_provider": status.PreferredProvider,
				"preferred_model":    status.PreferredModel,
				"configured_model":   status.ConfiguredModel,
				"running_tasks":      status.RunningTaskCount,
				"queued_tasks":       status.QueuedTaskCount,
				"squads":             append([]SquadMembership(nil), status.Squads...),
			})
		}
		return out, nil
	})

	register("get_agent_status", "Get the runtime status of one agent.", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"agent_name": map[string]interface{}{
				"type":        "string",
				"description": "The target agent name.",
			},
		},
		"required": []string{"agent_name"},
	}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		agentName := getStringArg(args, "agent_name")
		if agentName == "" {
			return nil, fmt.Errorf("agent_name is required")
		}
		return m.GetAgentStatus(agentName)
	})

	register("submit_squad_task", "Queue a task for a squad and return an immediate acknowledgement.", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"squad_id":   map[string]interface{}{"type": "string"},
			"squad_name": map[string]interface{}{"type": "string"},
			"prompt": map[string]interface{}{
				"type":        "string",
				"description": "The task prompt to queue.",
			},
			"agent_names": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Optional target agent names. Defaults to the squad captain.",
			},
		},
		"required": []string{"prompt"},
	}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		squad, err := m.resolveSquadRef(getStringArg(args, "squad_id"), getStringArg(args, "squad_name"))
		if err != nil {
			return nil, err
		}
		lead, err := m.GetLeadAgentForSquad(squad.ID)
		if err != nil {
			return nil, err
		}

		prompt := getStringArg(args, "prompt")
		if prompt == "" {
			return nil, fmt.Errorf("prompt is required")
		}

		agentNames := getStringSliceArg(args, "agent_names")
		if len(agentNames) == 0 {
			agentNames = []string{lead.Name}
		}

		task, err := m.EnqueueSharedTaskForSquad(ctx, squad.ID, lead.Name, agentNames, prompt)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"task_id":      task.ID,
			"squad_id":     task.SquadID,
			"squad_name":   squad.Name,
			"captain_name": task.CaptainName,
			"agent_names":  append([]string(nil), task.AgentNames...),
			"ack_message":  task.AckMessage,
			"status":       task.Status,
			"queued_ahead": task.QueuedAhead,
		}, nil
	})

	register("list_squad_tasks", "List recent tasks for a squad.", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"squad_id":   map[string]interface{}{"type": "string"},
			"squad_name": map[string]interface{}{"type": "string"},
			"limit":      map[string]interface{}{"type": "number"},
		},
	}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		squad, err := m.resolveSquadRef(getStringArg(args, "squad_id"), getStringArg(args, "squad_name"))
		if err != nil {
			return nil, err
		}
		limit := getIntArg(args, "limit", 10)
		tasks := m.ListSharedTasksForSquad(squad.ID, time.Time{}, limit)
		out := make([]map[string]interface{}, 0, len(tasks))
		for _, task := range tasks {
			out = append(out, map[string]interface{}{
				"task_id":      task.ID,
				"captain_name": task.CaptainName,
				"agent_names":  append([]string(nil), task.AgentNames...),
				"prompt":       task.Prompt,
				"status":       task.Status,
				"ack_message":  task.AckMessage,
				"result_text":  task.ResultText,
				"created_at":   task.CreatedAt,
				"started_at":   task.StartedAt,
				"finished_at":  task.FinishedAt,
			})
		}
		return out, nil
	})
}

func (m *SquadManager) resolveSquadRef(squadID, squadName string) (*Squad, error) {
	squadID = strings.TrimSpace(squadID)
	squadName = strings.TrimSpace(squadName)
	switch {
	case squadID != "":
		return m.store.GetTeam(squadID)
	case squadName != "":
		return m.store.GetTeamByName(squadName)
	default:
		return m.store.GetTeam(defaultSquadID)
	}
}

func getStringArg(args map[string]interface{}, key string) string {
	if args == nil {
		return ""
	}
	if value, ok := args[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func getStringSliceArg(args map[string]interface{}, key string) []string {
	if args == nil {
		return nil
	}
	raw, ok := args[key]
	if !ok || raw == nil {
		return nil
	}
	switch values := raw.(type) {
	case []string:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if value = strings.TrimSpace(value); value != "" {
				out = append(out, value)
			}
		}
		return out
	case []interface{}:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if item, ok := value.(string); ok {
				item = strings.TrimSpace(item)
				if item != "" {
					out = append(out, item)
				}
			}
		}
		return out
	default:
		return nil
	}
}

func getIntArg(args map[string]interface{}, key string, fallback int) int {
	if args == nil {
		return fallback
	}
	switch value := args[key].(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return fallback
	}
}
