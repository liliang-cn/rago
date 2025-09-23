package client

import (
	"context"
	"fmt"
)

// AgentWrapper wraps agent functionality
type AgentWrapper struct {
	client *BaseClient
}

// NewAgentWrapper creates a new agent wrapper
func NewAgentWrapper(client *BaseClient) *AgentWrapper {
	return &AgentWrapper{client: client}
}

// Run runs a task (simplified method)
func (a *AgentWrapper) Run(task string) (*TaskResponse, error) {
	ctx := context.Background()
	opts := &AgentOptions{
		Verbose: false,
	}
	return a.RunWithOptions(ctx, task, opts)
}

// RunWithOptions runs a task with specific options
func (a *AgentWrapper) RunWithOptions(ctx context.Context, task string, opts *AgentOptions) (*TaskResponse, error) {
	if a.client == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	// Use the RunTask method from BaseClient
	req := TaskRequest{
		Task:    task,
		Verbose: opts != nil && opts.Verbose,
		Timeout: 0,
	}

	if opts != nil && opts.Timeout > 0 {
		req.Timeout = opts.Timeout
	}

	return a.client.RunTask(ctx, req)
}

// PlanWithOptions creates a plan for a task
func (a *AgentWrapper) PlanWithOptions(ctx context.Context, task string, opts *AgentOptions) (*PlanResponse, error) {
	if a.client == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	// For now, return a simple plan
	// This would be replaced with actual planning logic
	plan := &PlanResponse{
		Task: task,
		Steps: []PlanStep{
			{Name: "Analyze", Description: "Analyze task requirements"},
			{Name: "Execute", Description: "Execute the task"},
			{Name: "Return", Description: "Return results"},
		},
	}

	return plan, nil
}
