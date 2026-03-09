package agents

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/liliang-cn/agent-go/pkg/agent"
	"github.com/liliang-cn/agent-go/pkg/config"
	"github.com/spf13/cobra"
)

var AgentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "Manage dynamic specialized agents",
	Long:  `Create, list, start, stop, and configure specialized agents in the system.`,
}

func init() {
	AgentsCmd.AddCommand(listCmd)
	AgentsCmd.AddCommand(startCmd)
	AgentsCmd.AddCommand(stopCmd)
}

func getManager() (*agent.AgentManager, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, err
	}
	agentDBPath := cfg.DataDir() + "/agent.db"
	store, err := agent.NewStore(agentDBPath)
	if err != nil {
		return nil, err
	}
	manager := agent.NewAgentManager(store)
	_ = manager.SeedDefaultAgents()
	return manager, nil
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all specialized agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := getManager()
		if err != nil {
			return err
		}
		agentsList, err := manager.DiscoverAgents()
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tSTATUS\tDESCRIPTION")
		for _, a := range agentsList {
			fmt.Fprintf(w, "%s\t%s\t%s\n", a.Name, a.Status, a.Description)
		}
		w.Flush()
		return nil
	},
}

var startCmd = &cobra.Command{
	Use:   "start [name]",
	Short: "Start a specific agent by name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := getManager()
		if err != nil {
			return err
		}
		name := args[0]
		err = manager.StartAgent(context.Background(), name)
		if err != nil {
			return err
		}
		fmt.Printf("✅ Agent '%s' started successfully.\n", name)
		return nil
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop [name]",
	Short: "Stop a specific agent by name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := getManager()
		if err != nil {
			return err
		}
		name := args[0]
		err = manager.StopAgent(context.Background(), name)
		if err != nil {
			return err
		}
		fmt.Printf("✅ Agent '%s' stopped successfully.\n", name)
		return nil
	},
}
