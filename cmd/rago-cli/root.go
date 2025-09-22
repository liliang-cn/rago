package main

import (
	"fmt"

	"github.com/liliang-cn/rago/v2/cmd/rago-cli/agent"
	"github.com/liliang-cn/rago/v2/cmd/rago-cli/mcp"
	"github.com/liliang-cn/rago/v2/cmd/rago-cli/rag"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	dbPath  string
	verbose bool
	quiet   bool
	cfg     *config.Config
	version string = "dev"
)

var RootCmd = &cobra.Command{
	Use:   "rago",
	Short: "RAGO - AI Development Platform for Go",
	Long: `RAGO is a comprehensive AI development platform for Go developers.
It provides unified access to:
  • LLM - Language model operations (generation, chat, streaming)
  • RAG - Retrieval-augmented generation (ingestion, search, Q&A)
  • Tools - External tool integration via MCP protocol
  • Agent - Autonomous task execution with natural language

Get started:
  rago init              # Initialize configuration
  rago platform demo     # Run platform demonstration
  rago platform --help   # Show platform commands`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for commands that don't need existing config
		if cmd.Name() == "init" || cmd.Name() == "version" {
			return nil
		}

		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		if dbPath != "" {
			cfg.Sqvect.DBPath = dbPath
		}

		// Pass shared variables to all packages
		rag.SetSharedVariables(cfg, verbose, quiet, version)
		mcp.SetSharedVariables(cfg, verbose, quiet)
		agent.SetSharedVariables(cfg, verbose, quiet)

		return nil
	},
}

func Execute() error {
	return RootCmd.Execute()
}

// GetRootCmd returns the root cobra command for testing purposes.
func GetRootCmd() *cobra.Command {
	return RootCmd
}

// SetVersion sets the version for the CLI
func SetVersion(v string) {
	version = v
	RootCmd.Version = v
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("RAGO version %s\n", version)
	},
}

func init() {
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "configuration file path (default: ~/.rago/rago.toml or ./rago.toml)")
	RootCmd.PersistentFlags().StringVar(&dbPath, "db-path", "", "database path (default: ./.rago/data/rag.db)")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose logging output")
	RootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "quiet mode")

	RootCmd.AddCommand(versionCmd)
	RootCmd.AddCommand(initCmd)

	// Add RAG parent command from rag package
	RootCmd.AddCommand(rag.RagCmd)

	// Add MCP parent command from mcp package
	RootCmd.AddCommand(mcp.MCPCmd)

	// Add Agent parent command from agent package
	agent.Initialize(nil)
	if agent.AgentCmd != nil {
		RootCmd.AddCommand(agent.AgentCmd)
	}

	RootCmd.AddCommand(serveCmd)
	RootCmd.AddCommand(llmCmd)
	RootCmd.AddCommand(chatCmd)
	RootCmd.AddCommand(statusCmd)
}
