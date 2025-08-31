package rago

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/liliang-cn/rago/pkg/config"
	"github.com/liliang-cn/rago/pkg/store"
	"github.com/spf13/cobra"
)

var (
	forceInit  bool
	outputPath string
)

var initCmd = &cobra.Command{
	Use:   "init [provider]",
	Short: "Initialize a new RAGO configuration file",
	Long: `Initialize creates a new RAGO configuration file with default settings.

Supported providers:
  ollama    - Use local Ollama server (default)
  openai    - Use OpenAI API
  lmstudio  - Use LM Studio server

Examples:
  rago init            # Interactive mode
  rago init ollama     # Initialize with Ollama
  rago init openai     # Initialize with OpenAI
  rago init lmstudio   # Initialize with LM Studio

The configuration will use ~/.rago/ directory for data storage by default.
After initialization, you can customize the configuration as needed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine config path
		configPath := outputPath
		if configPath == "" {
			// Use ~/.rago/rago.toml as default
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get user home directory: %w", err)
			}
			configPath = filepath.Join(homeDir, ".rago", "rago.toml")
		}

		// Check if file already exists
		if !forceInit {
			if _, err := os.Stat(configPath); err == nil {
				return fmt.Errorf("configuration file already exists at %s (use --force to overwrite)", configPath)
			}
		}

		// Determine provider type
		var providerType string
		if len(args) > 0 {
			providerType = strings.ToLower(args[0])
			if !isValidProvider(providerType) {
				return fmt.Errorf("invalid provider: %s. Supported providers: ollama, openai, lmstudio", args[0])
			}
		} else {
			// Interactive mode
			var err error
			providerType, err = promptForProvider()
			if err != nil {
				return err
			}
		}

		// Create directory if it doesn't exist
		dir := filepath.Dir(configPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// Generate config based on provider type
		var configContent string
		var err error
		
		switch providerType {
		case "ollama":
			configContent, err = generateOllamaConfig()
		case "openai":
			configContent, err = generateOpenAIConfig()
		case "lmstudio":
			configContent, err = generateLMStudioConfig()
		default:
			return fmt.Errorf("unsupported provider type: %s", providerType)
		}
		
		if err != nil {
			return fmt.Errorf("failed to generate config for %s: %w", providerType, err)
		}

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
		printProviderSpecificInstructions(providerType, configPath)

		return nil
	},
}

// isValidProvider checks if the provider type is supported
func isValidProvider(provider string) bool {
	validProviders := map[string]bool{
		"ollama":   true,
		"openai":   true,
		"lmstudio": true,
	}
	return validProviders[provider]
}

// promptForProvider prompts user to select a provider interactively
func promptForProvider() (string, error) {
	fmt.Println("\n🤖 Choose your AI provider:")
	fmt.Println("  1) Ollama (Local, recommended for privacy)")
	fmt.Println("  2) OpenAI (Cloud, best performance)")
	fmt.Println("  3) LM Studio (Local, GUI-based)")
	fmt.Print("\nEnter your choice (1-3) [default: 1]: ")
	
	var input string
	_, _ = fmt.Scanln(&input)
	
	switch strings.TrimSpace(input) {
	case "", "1":
		return "ollama", nil
	case "2":
		return "openai", nil
	case "3":
		return "lmstudio", nil
	default:
		return "", fmt.Errorf("invalid choice: %s", input)
	}
}

// generateOllamaConfig generates configuration for Ollama provider
func generateOllamaConfig() (string, error) {
	return `# RAGO Configuration File - Ollama
# This file configures RAGO to use local Ollama server

[server]
port = 7127
host = "0.0.0.0"
enable_ui = true
cors_origins = ["*"]

[providers]
default_llm = "ollama"
default_embedder = "ollama"

[providers.ollama]
type = "ollama"
base_url = "http://localhost:11434"
llm_model = "qwen3"
embedding_model = "nomic-embed-text"
timeout = "120s"

[sqvect]
db_path = "~/.rago/rag.db"
max_conns = 10
batch_size = 100
top_k = 5
threshold = 0.0

[keyword]
index_path = "~/.rago/keyword.bleve"

[chunker]
chunk_size = 500
overlap = 50
method = "sentence"

[rrf]
k = 10
relevance_threshold = 0.05

[ingest]
[ingest.metadata_extraction]
enable = false

[tools]
enabled = true
max_concurrent_calls = 5
call_timeout = "30s"
security_level = "normal"
log_level = "info"
enabled_tools = []

[tools.rate_limit]
calls_per_minute = 100
calls_per_hour = 1000
burst_size = 10

[tools.builtin]
[tools.builtin.file_operations]
enabled = true

[tools.builtin.http_client]
enabled = true

[tools.builtin.web_search]
enabled = false

[tools.plugins]
enabled = false
plugin_paths = ["./plugins"]
auto_load = false

[mcp]
enabled = false
log_level = "info"
default_timeout = "30s"
max_concurrent_requests = 10
health_check_interval = "60s"
`, nil
}

// generateOpenAIConfig generates configuration for OpenAI provider
func generateOpenAIConfig() (string, error) {
	fmt.Print("\n🔑 Enter your OpenAI API key: ")
	var apiKey string
	_, _ = fmt.Scanln(&apiKey)
	
	if strings.TrimSpace(apiKey) == "" {
		return "", fmt.Errorf("OpenAI API key is required")
	}

	return fmt.Sprintf(`# RAGO Configuration File - OpenAI
# This file configures RAGO to use OpenAI API

[server]
port = 7127
host = "0.0.0.0"
enable_ui = true
cors_origins = ["*"]

[providers]
default_llm = "openai"
default_embedder = "openai"

[providers.openai]
type = "openai"
api_key = "%s"
base_url = "https://api.openai.com/v1"
llm_model = "gpt-4o-mini"
embedding_model = "text-embedding-3-small"
timeout = "60s"

[sqvect]
db_path = "~/.rago/rag.db"
max_conns = 10
batch_size = 100
top_k = 5
threshold = 0.0

[keyword]
index_path = "~/.rago/keyword.bleve"

[chunker]
chunk_size = 500
overlap = 50
method = "sentence"

[rrf]
k = 10
relevance_threshold = 0.05

[ingest]
[ingest.metadata_extraction]
enable = false

[tools]
enabled = true
max_concurrent_calls = 5
call_timeout = "30s"
security_level = "normal"
log_level = "info"
enabled_tools = []

[tools.rate_limit]
calls_per_minute = 100
calls_per_hour = 1000
burst_size = 10

[tools.builtin]
[tools.builtin.file_operations]
enabled = true

[tools.builtin.http_client]
enabled = true

[tools.builtin.web_search]
enabled = false

[tools.plugins]
enabled = false
plugin_paths = ["./plugins"]
auto_load = false

[mcp]
enabled = false
log_level = "info"
default_timeout = "30s"
max_concurrent_requests = 10
health_check_interval = "60s"
`, apiKey), nil
}

// generateLMStudioConfig generates configuration for LM Studio provider
func generateLMStudioConfig() (string, error) {
	fmt.Print("\n🖥️  Enter LM Studio server URL [http://localhost:1234]: ")
	var baseURL string
	_, _ = fmt.Scanln(&baseURL)
	
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "http://localhost:1234"
	}

	fmt.Print("Enter LLM model name: ")
	var llmModel string
	_, _ = fmt.Scanln(&llmModel)
	
	if strings.TrimSpace(llmModel) == "" {
		return "", fmt.Errorf("LLM model name is required")
	}

	fmt.Print("Enter embedding model name: ")
	var embeddingModel string
	_, _ = fmt.Scanln(&embeddingModel)
	
	if strings.TrimSpace(embeddingModel) == "" {
		return "", fmt.Errorf("embedding model name is required")
	}

	return fmt.Sprintf(`# RAGO Configuration File - LM Studio
# This file configures RAGO to use LM Studio server

[server]
port = 7127
host = "0.0.0.0"
enable_ui = true
cors_origins = ["*"]

[providers]
default_llm = "lmstudio"
default_embedder = "lmstudio"

[providers.lmstudio]
type = "lmstudio"
base_url = "%s"
llm_model = "%s"
embedding_model = "%s"
timeout = "120s"

[sqvect]
db_path = "~/.rago/rag.db"
max_conns = 10
batch_size = 100
top_k = 5
threshold = 0.0

[keyword]
index_path = "~/.rago/keyword.bleve"

[chunker]
chunk_size = 500
overlap = 50
method = "sentence"

[rrf]
k = 10
relevance_threshold = 0.05

[ingest]
[ingest.metadata_extraction]
enable = false

[tools]
enabled = true
max_concurrent_calls = 5
call_timeout = "30s"
security_level = "normal"
log_level = "info"
enabled_tools = []

[tools.rate_limit]
calls_per_minute = 100
calls_per_hour = 1000
burst_size = 10

[tools.builtin]
[tools.builtin.file_operations]
enabled = true

[tools.builtin.http_client]
enabled = true

[tools.builtin.web_search]
enabled = false

[tools.plugins]
enabled = false
plugin_paths = ["./plugins"]
auto_load = false

[mcp]
enabled = false
log_level = "info"
default_timeout = "30s"
max_concurrent_requests = 10
health_check_interval = "60s"
`, baseURL, llmModel, embeddingModel), nil
}

// printProviderSpecificInstructions prints provider-specific setup instructions
func printProviderSpecificInstructions(providerType, configPath string) {
	fmt.Println("\n📁 Data Directory:")
	fmt.Println("   • ~/.rago/ - Main data directory")
	fmt.Println("   • ~/.rago/rag.db - Vector database")
	fmt.Println("   • ~/.rago/keyword.bleve/ - Keyword search index")
	
	fmt.Printf("\n📝 Configuration Summary:\n")
	fmt.Printf("   • Provider: %s\n", strings.ToUpper(string(providerType[0]))+providerType[1:])
	fmt.Printf("   • Config file: %s\n", configPath)
	fmt.Println("   • Tools: Enabled")
	fmt.Println("   • Web UI: Enabled")

	fmt.Println("\n🚀 Next Steps:")
	
	switch providerType {
	case "ollama":
		fmt.Println("   1. Make sure Ollama is running:")
		fmt.Println("      ollama serve")
		fmt.Println("   2. Pull required models:")
		fmt.Println("      ollama pull qwen3")
		fmt.Println("      ollama pull nomic-embed-text")
		fmt.Printf("   3. Start RAGO:\n      rago --config %s serve\n", configPath)
		fmt.Println("   4. Open http://localhost:7127 in your browser")
	case "openai":
		fmt.Println("   1. Ensure your OpenAI API key has sufficient credits")
		fmt.Printf("   2. Start RAGO:\n      rago --config %s serve\n", configPath)
		fmt.Println("   3. Open http://localhost:7127 in your browser")
	case "lmstudio":
		fmt.Println("   1. Make sure LM Studio server is running")
		fmt.Println("   2. Load your models in LM Studio")
		fmt.Printf("   3. Start RAGO:\n      rago --config %s serve\n", configPath)
		fmt.Println("   4. Open http://localhost:7127 in your browser")
	}
}

func initializeDatabase(configPath string) error {
	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
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

	// Initialize SQLite vector store
	vectorStore, err := store.NewSQLiteStore(cfg.Sqvect.DBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize vector store: %w", err)
	}
	defer func() {
		if err := vectorStore.Close(); err != nil {
			fmt.Printf("Warning: failed to close vector store: %v\n", err)
		}
	}()

	// Initialize Bleve keyword store
	keywordStore, err := store.NewKeywordStore(cfg.Keyword.IndexPath)
	if err != nil {
		return fmt.Errorf("failed to initialize keyword store: %w", err)
	}
	defer func() {
		if err := keywordStore.Close(); err != nil {
			fmt.Printf("Warning: failed to close keyword store: %v\n", err)
		}
	}()

	// Create a .gitignore file if it doesn't exist
	gitignorePath := "./.gitignore"
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		gitignoreContent := `# RAGO Data Files
~/.rago/

# Configuration with sensitive data (if using API keys)
# ~/.rago/rago.toml

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
	initCmd.Flags().StringVarP(&outputPath, "output", "o", "", "output path for configuration file (default: ~/.rago/rago.toml)")
}
