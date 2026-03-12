package agent

import "context"

const currentAgentKey contextKey = "current_agent"

func withCurrentAgent(ctx context.Context, agent *Agent) context.Context {
	if agent == nil {
		return ctx
	}
	return context.WithValue(ctx, currentAgentKey, agent)
}

func getCurrentAgent(ctx context.Context) *Agent {
	if ctx == nil {
		return nil
	}
	if agent, ok := ctx.Value(currentAgentKey).(*Agent); ok {
		return agent
	}
	return nil
}
