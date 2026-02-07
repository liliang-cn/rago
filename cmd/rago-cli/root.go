package main

import (
	"context"
	"fmt"

	"github.com/liliang-cn/rago/v2/cmd/rago-cli/agent"
	"github.com/liliang-cn/rago/v2/cmd/rago-cli/mcp"
	"github.com/liliang-cn/rago/v2/cmd/rago-cli/rag"
	"github.com/liliang-cn/rago/v2/cmd/rago-cli/skills"
	"github.com/liliang-cn/rago/v2/pkg/config"
	ragolog "github.com/liliang-cn/rago/v2/pkg/log"
	"github.com/liliang-cn/rago/v2/pkg/services"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
	debug   bool
	quiet   bool
	cfg     *config.Config
	version string = "dev"
)

var RootCmd = &cobra.Command{
	Use:   "rago",
	Short: "RAGO - AI Agent SDK & CLI for Go developers",
	Long: `RAGO is a modular AI development platform that empowers Go applications with:
  • Agent  - Autonomous planning and execution with multi-turn reasoning
  • RAG    - Hybrid retrieval using Vector search and Knowledge Graphs
  • LLM    - Unified API for Ollama, OpenAI, DeepSeek, and more
  • MCP    - Standardized tool integration via Model Context Protocol
  • Skills - Expert capabilities via Claude-compatible markdown skills
  • Status - Real-time monitoring of provider health and system status`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for commands that don't need existing config
		if cmd.Name() == "version" || cmd.Name() == "init" {
			return nil
		}

		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Enable debug mode if CLI flag is set or config has debug=true
		if debug || cfg.Debug {
			ragolog.SetDebug(true)
		}

		// Initialize global pool service
		globalPoolService := services.GetGlobalPoolService()
		ctx := context.Background()
		if err := globalPoolService.Initialize(ctx, cfg); err != nil {
			return fmt.Errorf("failed to initialize global pool service: %w", err)
		}

		// Pass shared variables to all packages
		rag.SetSharedVariables(cfg, verbose, quiet, version)
		mcp.SetSharedVariables(cfg, verbose, quiet)
		agent.SetSharedVariables(cfg, verbose)

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
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "configuration file path (default: ./rago.toml or ~/.rago/config/rago.toml)")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose logging output")
	RootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable debug logging")
	RootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "quiet mode")

	RootCmd.AddCommand(versionCmd)

	// Add RAG parent command from rag package
	RootCmd.AddCommand(rag.RagCmd)

	// Add MCP parent command from mcp package
	RootCmd.AddCommand(mcp.MCPCmd)

	// Add Agent command
	RootCmd.AddCommand(agent.AgentCmd)

	// Add Skills command
	RootCmd.AddCommand(skills.Cmd)

	RootCmd.AddCommand(llmCmd)
	RootCmd.AddCommand(statusCmd)
}
