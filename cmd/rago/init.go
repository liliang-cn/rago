package rago

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/liliang-cn/rago/internal/config"
	"github.com/liliang-cn/rago/internal/store"
	"github.com/spf13/cobra"
)

var (
	forceInit  bool
	outputPath string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new RAGO configuration file with Ollama as default provider",
	Long: `Initialize creates a new RAGO configuration file with default settings using Ollama as the default provider.

This command will create a config.toml file in the current directory with:
- Ollama configured as the default LLM and embedding provider
- Modern provider-based architecture
- Tool calling functionality enabled
- All default configuration values

The generated configuration includes commented OpenAI provider settings
that you can uncomment and customize if you want to use OpenAI instead.

After initialization, you can customize the configuration as needed and start RAGO.`,
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

		// Initialize the database and create directory structure
		if err := initializeDatabase(configPath); err != nil {
			fmt.Printf("⚠️  Configuration file created but initialization failed: %v\n", err)
			fmt.Println("   You can try initializing later by running:")
			fmt.Printf("   rago --config %s serve\n", configPath)
		} else {
			fmt.Println("✅ Database and directory structure initialized successfully")
		}

		fmt.Printf("✅ Configuration file created successfully at: %s\n", configPath)
		fmt.Println("\n📁 Directory Structure Created:")
		fmt.Println("   • ./data/ - Main data directory")
		fmt.Println("   • ./data/rag.db - Vector database")
		fmt.Println("   • ./data/keyword.bleve/ - Keyword search index")
		fmt.Println("   • ./data/documents/ - Document storage")
		fmt.Println("   • ./data/exports/ - Export files")
		fmt.Println("   • ./data/imports/ - Import files")
		fmt.Println("   • ./data/backups/ - Backup files")
		fmt.Println("   • .gitignore - Git ignore file (if not exists)")
		fmt.Println("\n📝 Configuration Summary:")
		fmt.Println("   • Provider: Ollama (default)")
		fmt.Println("   • LLM Model: qwen3")
		fmt.Println("   • Embedding Model: nomic-embed-text")
		fmt.Println("   • Tools: Enabled")
		fmt.Println("   • Web UI: Enabled")
		fmt.Println("\n🚀 Next Steps:")
		fmt.Println("   1. Make sure Ollama is running with your models:")
		fmt.Println("      ollama pull qwen3")
		fmt.Println("      ollama pull nomic-embed-text")
		fmt.Println("   2. Start RAGO:")
		fmt.Printf("      rago --config %s serve\n", configPath)
		fmt.Println("   3. Open http://localhost:7127 in your browser")
		fmt.Println("\n💡 To use OpenAI instead, uncomment and configure the [providers.openai] section in the config file.")

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
enable_ui = true
cors_origins = ["*"]

[providers]
# Default providers - using Ollama as default
default_llm = "ollama"
default_embedder = "ollama"

# Ollama Provider Configuration
[providers.ollama]
type = "ollama"
base_url = "http://localhost:11434"
llm_model = "qwen3"                    # LLM model for text generation
embedding_model = "nomic-embed-text"     # Model for embeddings
timeout = "120s"                         # Request timeout

# OpenAI Provider Configuration (optional)
# Uncomment and configure to use OpenAI instead
# [providers.openai]
# type = "openai"
# api_key = "your-openai-api-key-here"
# base_url = "https://api.openai.com/v1"
# llm_model = "gpt-4o-mini"
# embedding_model = "text-embedding-3-small"
# timeout = "60s"

[sqvect]
# SQLite vector database configuration
db_path = "./data/rag.db"
vector_dim = 768                         # 768 for nomic-embed-text, 1536 for OpenAI
max_conns = 10
batch_size = 100
top_k = 5
threshold = 0.0

[keyword]
# Bleve keyword index configuration
index_path = "./data/keyword.bleve"

[chunker]
# Document chunking configuration
chunk_size = 500
overlap = 50
method = "sentence"                      # options: sentence, paragraph, token

[ingest]
# Document ingestion configuration

[ingest.metadata_extraction]
# Automatic metadata extraction configuration
enable = false                           # Enable automatic metadata extraction via LLM
llm_model = "qwen3"                    # Model to use for metadata extraction

[tools]
# Tool calling configuration
enabled = true                           # Enable tool calling functionality
max_concurrent_calls = 5                 # Maximum concurrent tool calls
call_timeout = "30s"                     # Timeout for individual tool calls
security_level = "normal"                # Security level: strict, normal, relaxed
log_level = "info"                       # Log level: debug, info, warn, error

# List of enabled tools
enabled_tools = [
    "datetime",                          # Date and time operations
    "rag_search",                        # Search in RAG database
    "document_info",                     # Document information queries
    "open_url",                       # HTTP web requests
    "web_search",                     # Google search functionality
    "file_operations"                    # File system operations
]

# Built-in tool configurations
[tools.builtin]

[tools.builtin.web_search]
enabled = true
# api_key = "your-google-api-key"       # Optional: for better rate limits
# search_engine_id = "your-cse-id"     # Optional: for custom search

[tools.builtin.open_url]
enabled = true
timeout = "10s"
max_redirects = 5
user_agent = "RAGO/1.3.1"

[tools.builtin.file_operations]
enabled = true
max_file_size = "10MB"
allowed_extensions = [".txt", ".md", ".json", ".csv", ".log"]
base_directory = "./data"
`
}

func initializeDatabase(configPath string) error {
	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create main data directory
	dataDir := "./data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory %s: %w", dataDir, err)
	}

	// Create specific directories for different data types
	dbDir := filepath.Dir(cfg.Sqvect.DBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory %s: %w", dbDir, err)
	}

	keywordDir := filepath.Dir(cfg.Keyword.IndexPath)
	if err := os.MkdirAll(keywordDir, 0755); err != nil {
		return fmt.Errorf("failed to create keyword index directory %s: %w", keywordDir, err)
	}

	// Create additional useful directories
	additionalDirs := []string{
		"./data/documents", // For document storage
		"./data/exports",   // For export files
		"./data/imports",   // For import files
		"./data/backups",   // For backup files
	}

	for _, dir := range additionalDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Initialize SQLite vector store
	vectorStore, err := store.NewSQLiteStore(
		cfg.Sqvect.DBPath,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize vector store: %w", err)
	}
	defer vectorStore.Close()

	// Initialize Bleve keyword store
	keywordStore, err := store.NewKeywordStore(cfg.Keyword.IndexPath)
	if err != nil {
		return fmt.Errorf("failed to initialize keyword store: %w", err)
	}
	defer keywordStore.Close()

	// Create a .gitignore file if it doesn't exist
	gitignorePath := "./.gitignore"
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		gitignoreContent := `# RAGO Data Files
data/rag.db
data/keyword.bleve/
data/documents/
data/exports/
data/imports/
data/backups/

# Configuration with sensitive data (if using API keys)
# config.toml

# Logs
*.log

# OS generated files
.DS_Store
Thumbs.db
`
		if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
			// Not critical, just log the issue but don't fail
			fmt.Printf("Warning: Could not create .gitignore file: %v\n", err)
		}
	}

	return nil
}

func init() {
	initCmd.Flags().BoolVarP(&forceInit, "force", "f", false, "overwrite existing configuration file")
	initCmd.Flags().StringVarP(&outputPath, "output", "o", "", "output path for configuration file (default: config.toml)")
}
