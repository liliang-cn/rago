package agent

import (
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/spf13/cobra"
)

var (
	// Shared configuration
	Cfg *config.Config

	// Shared flags (passed from root)
	verbose bool
	quiet   bool

	// AgentCmd is the main agent command exported for root
	AgentCmd *cobra.Command
)

// SetSharedVariables sets the shared configuration and flags
func SetSharedVariables(cfg *config.Config, v bool, q bool) {
	Cfg = cfg
	verbose = v
	quiet = q
}

// Initialize sets up the agent command structure
func Initialize(cfg *config.Config) {
	Cfg = cfg

	// Initialize agent command if not already done
	if AgentCmd == nil {
		// Create the main agent command
		AgentCmd = &cobra.Command{
			Use:   "agent",
			Short: "Execute natural language requests using MCP tools",
			Long:  `Execute natural language requests by planning and executing MCP tools dynamically.`,
		}

		// Add run command - the main agent functionality
		AgentCmd.AddCommand(agentRunCmd)

		// Add exec command - execute saved plans
		AgentCmd.AddCommand(agentExecCmd)

		// Add do command - intelligent RAG + planning + execution
		AgentCmd.AddCommand(agentDoCmd)
	}
}
