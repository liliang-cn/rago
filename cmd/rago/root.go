package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	dbPath  string
	verbose bool
	quiet   bool
	cfg     *client.Config
	version string = "dev"
)

var RootCmd = &cobra.Command{
	Use:   "rago",
	Short: "RAGO - Local-First AI Foundation",
	Long: `RAGO is a comprehensive local-first, privacy-first AI foundation with four equal pillars:
- LLM: Multi-provider language model management with load balancing
- RAG: Document ingestion, vectorization, and semantic retrieval
- MCP: Model Context Protocol for tool integration
- Agents: Autonomous workflows combining all capabilities

RAGO can be used as a library in your Go applications or as a standalone CLI/server.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Configure logging based on verbose flag
		if verbose {
			log.SetOutput(os.Stderr)
		} else {
			log.SetOutput(io.Discard)
		}
		
		// Skip config loading for commands that don't need existing config
		if cmd.Name() == "init" || cmd.Name() == "version" {
			return nil
		}

		var err error
		cfg, err = client.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		if dbPath != "" {
			// TODO: Map database path to new config structure
			// Legacy cfg.Sqvect.DBPath is no longer available
			// This may need to be handled through DataDir or similar
			cfg.DataDir = dbPath
		}

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
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "configuration file path (default search order: ./rago.toml → ./.rago/rago.toml → ~/.rago/rago.toml)")
	RootCmd.PersistentFlags().StringVar(&dbPath, "db-path", "", "database path (default: ./data/rag.db)")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose logging output")
	RootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "quiet mode")

	// Core commands
	RootCmd.AddCommand(versionCmd)
	RootCmd.AddCommand(statusCmd)     // Status command
	RootCmd.AddCommand(serveCmd)      // Serve command
	
	// Four-pillar commands
	RootCmd.AddCommand(llmCmd)        // LLM pillar
	RootCmd.AddCommand(ragCmd)        // RAG pillar  
	RootCmd.AddCommand(mcpCmd)        // MCP pillar
	RootCmd.AddCommand(agentCmd)      // Agent pillar
	
	// Multi-pillar commands
	RootCmd.AddCommand(ingestCmd)     // RAG ingest command
}
