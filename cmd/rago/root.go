package rago

import (
	"fmt"

	"github.com/liliang-cn/rago/v2/cmd/rago/agent"
	"github.com/liliang-cn/rago/v2/cmd/rago/mcp"
	"github.com/liliang-cn/rago/v2/cmd/rago/rag"
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
	Short: "RAGO - Local RAG System",
	Long: `RAGO (Retrieval-Augmented Generation Offline) is a fully local RAG system written in Go,
integrating SQLite vector database (sqvect) and local LLM client (ollama-go),
supporting document ingestion, semantic search, and context-enhanced Q&A.`,
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
}
