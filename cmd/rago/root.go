package rago

import (
	"fmt"

	"github.com/liliang-cn/rago/internal/config"
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

var rootCmd = &cobra.Command{
	Use:   "rago",
	Short: "RAGO - Local RAG System",
	Long: `RAGO (Retrieval-Augmented Generation Offline) is a fully local RAG system written in Go,
integrating SQLite vector database (sqvect) and local LLM client (ollama-go),
supporting document ingestion, semantic search, and context-enhanced Q&A.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		if dbPath != "" {
			cfg.Sqvect.DBPath = dbPath
		}

		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

// SetVersion sets the version for the CLI
func SetVersion(v string) {
	version = v
	rootCmd.Version = v
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("RAGO version %s\n", version)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "configuration file path (default: ./config.toml)")
	rootCmd.PersistentFlags().StringVar(&dbPath, "db-path", "", "database path (default: ./data/rag.db)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose logging output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "quiet mode")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(ingestCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(resetCmd)
	rootCmd.AddCommand(serveCmd)
}
