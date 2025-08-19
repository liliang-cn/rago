package rago

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	forceInit  bool
	outputPath string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new RAGO configuration file",
	Long: `Initialize creates a new RAGO configuration file with default settings.
This command will create a config.toml file in the current directory with
all default configuration values, which you can then customize as needed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := outputPath
		if configPath == "" {
			configPath = "config.toml"
		}

		// Check if file already exists
		if !forceInit {
			if _, err := os.Stat(configPath); err == nil {
				return fmt.Errorf("configuration file already exists at %s (use --force to overwrite)", configPath)
			}
		}

		// Create directory if it doesn't exist
		dir := filepath.Dir(configPath)
		if dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
		}

		// Generate default config content
		configContent := generateDefaultConfig()

		// Write config file
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			return fmt.Errorf("failed to write configuration file: %w", err)
		}

		fmt.Printf("✅ Configuration file created successfully at: %s\n", configPath)
		fmt.Println("\n📝 You can now customize the configuration and start using RAGO:")
		fmt.Printf("   rago --config %s serve\n", configPath)

		return nil
	},
}

func generateDefaultConfig() string {
	return `# RAGO Configuration File
# This file contains all configuration options for RAGO (Retrieval-Augmented Generation Offline)

[server]
# Server configuration
port = 7127
host = "0.0.0.0"
enable_ui = false
cors_origins = ["*"]

[ollama]
# Ollama LLM configuration
embedding_model = "nomic-embed-text"
llm_model = "qwen3"
base_url = "http://localhost:11434"
timeout = "30s"

[sqvect]
# SQLite vector database configuration
db_path = "./data/rag.db"
vector_dim = 1536
max_conns = 10
batch_size = 100
top_k = 5
threshold = 0.0

[chunker]
# Document chunking configuration
chunk_size = 300
overlap = 50
method = "sentence"  # options: sentence, paragraph, token
`
}

func init() {
	initCmd.Flags().BoolVarP(&forceInit, "force", "f", false, "overwrite existing configuration file")
	initCmd.Flags().StringVarP(&outputPath, "output", "o", "", "output path for configuration file (default: config.toml)")
}