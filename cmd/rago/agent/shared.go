package agent

import (
	"github.com/liliang-cn/rago/v2/pkg/agents"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/spf13/cobra"
)

var (
	// Shared configuration
	Cfg          *config.Config
	agentManager *agents.Manager

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
		setupCommands()

		// Add all subcommands
		AgentCmd.AddCommand(agentCreateCmd)
		AgentCmd.AddCommand(agentListCmd)
		AgentCmd.AddCommand(agentGetCmd)
		AgentCmd.AddCommand(agentExecuteCmd)
		AgentCmd.AddCommand(agentDeleteCmd)
		AgentCmd.AddCommand(agentTemplatesCmd)

		// Add generate commands
		AgentCmd.AddCommand(agentGenerateCmd)

		// Add run command
		AgentCmd.AddCommand(agentRunCmd)

		// Add workflow commands
		AgentCmd.AddCommand(workflowGenerateCmd)
		AgentCmd.AddCommand(workflowTemplateCmd)

		// Add planner commands
		AgentCmd.AddCommand(agentPlanCmd)
		AgentCmd.AddCommand(agentRunPlanCmd)
		AgentCmd.AddCommand(agentListPlansCmd)
		AgentCmd.AddCommand(agentPlanStatusCmd)
		AgentCmd.AddCommand(agentAutoRunCmd)
	}
}
