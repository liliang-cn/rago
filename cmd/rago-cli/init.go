package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/rag/store"
	"github.com/spf13/cobra"
)

var (
	forceInit    bool
	outputPath   string
	enableMCP    bool
	skipDatabase bool
)

var initCmd = &cobra.Command{
	Use:   "init [provider]",
	Short: "Initialize RAGO (optional - RAGO works without config!)",
	Long: `Initialize creates an optional configuration file.

🚀 QUICK START: RAGO works WITHOUT any configuration!
   Just run: rago status

When to use init:
  - To use a different LLM provider (OpenAI, LM Studio)
  - To change default models
  - To enable advanced features (MCP)

Providers:
  ollama    - Local Ollama (default, no config needed!)
  openai    - OpenAI API (requires API key)
  lmstudio  - LM Studio (local GUI)
  custom    - Custom OpenAI-compatible API

Examples:
  rago status                # Works immediately, no config needed!
  rago init                  # Optional: create config file
  rago init openai          # Use OpenAI instead of Ollama
  rago init --enable-mcp    # Enable advanced tool capabilities`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine config path
		configPath := outputPath
		if configPath == "" {
			// Default to ./rago.toml in current directory
			configPath = "rago.toml"
		}

		// Check if file already exists
		if !forceInit {
			if _, err := os.Stat(configPath); err == nil {
				fmt.Printf("⚠️  Configuration file already exists at %s\n", configPath)
				fmt.Println("   Use --force to overwrite or choose a different location with --output")
				return fmt.Errorf("configuration file already exists")
			}
		}

		// Determine provider type
		var providerType string
		if len(args) > 0 {
			providerType = strings.ToLower(args[0])
			if !isValidProvider(providerType) {
				return fmt.Errorf("invalid provider: %s. Supported providers: ollama, openai, lmstudio, custom", args[0])
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
			if os.IsPermission(err) {
				return fmt.Errorf("permission denied: cannot create directory %s. Try running with sudo or choose a different location", dir)
			}
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// Generate config based on provider type
		var configContent string
		var err error

		switch providerType {
		case "skip":
			// User chose to skip config creation
			fmt.Println("\n✅ No configuration needed!")
			fmt.Println("\n🚀 Get started:")
			fmt.Println("   1. Make sure Ollama is running: ollama serve")
			fmt.Println("   2. Pull models: ollama pull qwen3 && ollama pull nomic-embed-text")
			fmt.Println("   3. Test: rago status")
			fmt.Println("   4. Use: rago query \"your question\"")
			fmt.Println("\n💡 Run 'rago init' later if you need custom settings")
			return nil
		case "ollama":
			configContent, err = generateOllamaConfig(enableMCP)
		case "openai":
			configContent, err = generateOpenAIConfig(enableMCP)
		case "lmstudio":
			configContent, err = generateLMStudioConfig(enableMCP)
		case "custom":
			configContent, err = generateCustomConfig(enableMCP)
		default:
			return fmt.Errorf("unsupported provider type: %s", providerType)
		}

		if err != nil {
			return fmt.Errorf("failed to generate config for %s: %w", providerType, err)
		}

		// Write config file
		fmt.Println("\n📝 Writing configuration file...")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			return fmt.Errorf("failed to write configuration file: %w", err)
		}

		// Initialize the database and create directory structure
		// For custom providers, skip DB init by default to avoid connection hangs
		shouldInitDB := !skipDatabase && providerType != "custom"
		if shouldInitDB {
			fmt.Println("🗄️  Initializing database (this may take a moment)...")
			if err := initializeDatabase(configPath); err != nil {
				fmt.Printf("⚠️  Configuration file created but database initialization failed: %v\n", err)
				fmt.Println("   The database will be created automatically when you first run RAGO")
			} else {
				fmt.Println("✅ Database and directory structure initialized successfully")
			}
		} else if providerType == "custom" && !skipDatabase {
			fmt.Println("ℹ️  Skipping database initialization for custom provider")
			fmt.Println("   The database will be created when you first use RAGO")
		}

		// Initialize MCP servers if requested
		if enableMCP {
			if err := initializeMCPServers(filepath.Dir(configPath)); err != nil {
				fmt.Printf("⚠️  MCP server configuration failed: %v\n", err)
				fmt.Println("   You can configure MCP servers later using:")
				fmt.Println("   rago mcp install")
			} else {
				fmt.Println("✅ MCP servers configured successfully")
			}
		}

		fmt.Printf("✅ Configuration file created successfully at: %s\n", configPath)
		printProviderSpecificInstructions(providerType, configPath)

		return nil
	},
}

// isValidProvider checks if the provider type is supported
func isValidProvider(provider string) bool {
	validProviders := map[string]bool{
		"skip":     true,
		"ollama":   true,
		"openai":   true,
		"lmstudio": true,
		"custom":   true,
	}
	return validProviders[provider]
}

// promptForProvider prompts user to select a provider interactively
func promptForProvider() (string, error) {
	fmt.Println("\n✨ RAGO works without any configuration!")
	fmt.Println("   Config files are only needed for:")
	fmt.Println("   • Using providers other than Ollama")
	fmt.Println("   • Changing default models")
	fmt.Println("   • Enabling advanced features")
	fmt.Println("📝 Create config for provider (optional):")
	fmt.Println("  1) Skip - Use defaults (recommended)")
	fmt.Println("  2) OpenAI - Cloud API")
	fmt.Println("  3) LM Studio - Local GUI")
	fmt.Println("  4) Custom - Other API")
	fmt.Print("\nChoice [1]: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	switch input {
	case "", "1":
		return "skip", nil
	case "2":
		return "openai", nil
	case "3":
		return "lmstudio", nil
	case "4":
		return "custom", nil
	default:
		return "", fmt.Errorf("invalid choice: %s", input)
	}
}

// generateOllamaConfig generates configuration for Ollama provider
func generateOllamaConfig(enableMCP bool) (string, error) {
	config := `# RAGO Configuration (Optional!)
# 🚀 Quick Start: RAGO works without ANY configuration!
# Just run: rago status

# Default configuration (already built-in, uncomment to override):
# [providers]
# default_llm = "ollama"

# [providers.ollama]
# type = "ollama"                      # Provider type
# base_url = "http://localhost:11434"
# llm_model = "qwen3"                  # or llama3, mistral, etc.
# embedding_model = "nomic-embed-text" # or mxbai-embed-large
# timeout = "30s"
`

	if enableMCP {
		config += `
# MCP (Model Context Protocol) for advanced tool capabilities
[mcp]
enabled = true
`
	}

	return config, nil
}

// generateOpenAIConfig generates configuration for OpenAI provider
func generateOpenAIConfig(enableMCP bool) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\n🔑 Enter your OpenAI API key: ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		return "", fmt.Errorf("OpenAI API key is required")
	}

	config := fmt.Sprintf(`# RAGO - Minimal Configuration (OpenAI)
# Get started with just 3 lines!

[providers]
default_llm = "openai"
default_embedder = "openai"

[providers.openai]
type = "openai"
api_key = "%s"
llm_model = "gpt-4o-mini"
embedding_model = "text-embedding-3-small"
timeout = "60s"
`, apiKey)

	if enableMCP {
		config += `
# MCP (Model Context Protocol) for advanced tool capabilities
[mcp]
enabled = true
`
	}

	return config, nil
}

// generateLMStudioConfig generates configuration for LM Studio provider
func generateLMStudioConfig(enableMCP bool) (string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("\n🖥️  Enter LM Studio server URL [http://localhost:1234]: ")
	baseURL, _ := reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)

	if baseURL == "" {
		baseURL = "http://localhost:1234"
	}

	fmt.Print("Enter LLM model name (e.g., 'qwen/qwen3-4b-2507'): ")
	llmModel, _ := reader.ReadString('\n')
	llmModel = strings.TrimSpace(llmModel)

	if llmModel == "" {
		return "", fmt.Errorf("LLM model name is required")
	}

	fmt.Print("Enter embedding model name (e.g., 'text-embedding-qwen3-embedding-4b'): ")
	embeddingModel, _ := reader.ReadString('\n')
	embeddingModel = strings.TrimSpace(embeddingModel)

	if embeddingModel == "" {
		return "", fmt.Errorf("embedding model name is required")
	}

	config := fmt.Sprintf(`# RAGO - Minimal Configuration (LM Studio)
# Get started with just 3 lines!

[providers]
default_llm = "lmstudio"
default_embedder = "lmstudio"

[providers.lmstudio]
type = "lmstudio"
base_url = "%s"
llm_model = "%s"
embedding_model = "%s"
timeout = "120s"
`, baseURL, llmModel, embeddingModel)

	if enableMCP {
		config += `
# MCP (Model Context Protocol) for advanced tool capabilities
[mcp]
enabled = true
`
	}

	return config, nil
}

// printProviderSpecificInstructions prints provider-specific setup instructions
func printProviderSpecificInstructions(providerType, configPath string) {
	fmt.Println("\n📁 Data Directory:")
	fmt.Println("   • ~/.rago/ - Main data directory")
	fmt.Println("   • ~/.rago/rag.db - Vector database")
	fmt.Println("   • ~/.rago/keyword.bleve/ - Keyword search index")
	if enableMCP {
		fmt.Println("   • ~/.rago/mcpServers.json - MCP server configuration")
	}

	fmt.Printf("\n📝 Configuration Summary:\n")
	fmt.Printf("   • Provider: %s\n", strings.ToUpper(string(providerType[0]))+providerType[1:])
	fmt.Printf("   • Config file: %s\n", configPath)
	fmt.Println("   • RAG: Enabled (dual storage: vector + keyword)")
	fmt.Println("   • Web UI: Enabled on port 7127")
	if enableMCP {
		fmt.Println("   • MCP Tools: Enabled")
	}

	fmt.Println("\n🚀 Next Steps:")

	switch providerType {
	case "ollama":
		fmt.Println("   1. Make sure Ollama is running:")
		fmt.Println("      ollama serve")
		fmt.Println("   2. Pull required models (or choose your own):")
		fmt.Println("      ollama pull qwen3           # LLM model")
		fmt.Println("      ollama pull nomic-embed-text # Embedding model")
		fmt.Println("   3. Test your configuration:")
		fmt.Printf("      rago --config %s status\n", configPath)
		fmt.Println("   4. Start RAGO server:")
		fmt.Printf("      rago --config %s serve\n", configPath)
		fmt.Println("   5. Open http://localhost:7127 in your browser")
		fmt.Println("\n   💡 Tip: Use 'ollama list' to see available models")
	case "openai":
		fmt.Println("   1. Verify your OpenAI API key has sufficient credits")
		fmt.Println("   2. Test your configuration:")
		fmt.Printf("      rago --config %s status\n", configPath)
		fmt.Println("   3. Start RAGO server:")
		fmt.Printf("      rago --config %s serve\n", configPath)
		fmt.Println("   4. Open http://localhost:7127 in your browser")
		fmt.Println("\n   💡 Tip: Monitor usage at https://platform.openai.com/usage")
	case "lmstudio":
		fmt.Println("   1. Start LM Studio and load your models")
		fmt.Println("   2. Start the LM Studio server (usually on port 1234)")
		fmt.Println("   3. Test your configuration:")
		fmt.Printf("      rago --config %s status\n", configPath)
		fmt.Println("   4. Start RAGO server:")
		fmt.Printf("      rago --config %s serve\n", configPath)
		fmt.Println("   5. Open http://localhost:7127 in your browser")
		fmt.Println("\n   💡 Tip: Check LM Studio's server logs for connection issues")
	case "custom":
		fmt.Println("   1. Ensure your custom API server is running")
		fmt.Println("   2. Test your configuration:")
		fmt.Printf("      rago --config %s status\n", configPath)
		fmt.Println("   3. Start RAGO server:")
		fmt.Printf("      rago --config %s serve\n", configPath)
		fmt.Println("   4. Open http://localhost:7127 in your browser")
		fmt.Println("\n   💡 Tip: Use 'rago status --verbose' for detailed connectivity info")
	}

	fmt.Println("\n📚 Quick Start Commands:")
	fmt.Println("   • Ingest documents:  rago ingest document.pdf")
	fmt.Println("   • Query knowledge:   rago query \"What is this about?\"")
	fmt.Println("   • Check status:      rago status")
	if enableMCP {
		fmt.Println("   • MCP status:        rago mcp status")
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

// generateCustomConfig generates configuration for a custom OpenAI-compatible provider
func generateCustomConfig(enableMCP bool) (string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n🔧 Configure custom OpenAI-compatible provider")

	fmt.Print("Provider name (e.g., 'localai', 'groq'): ")
	providerName, _ := reader.ReadString('\n')
	providerName = strings.TrimSpace(providerName)
	if providerName == "" {
		providerName = "custom"
	}

	fmt.Print("API Base URL (e.g., http://localhost:8080/v1): ")
	baseURL, _ := reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "", fmt.Errorf("API base URL is required")
	}

	fmt.Print("API Key (press Enter if not required): ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)

	// Show note about API key requirement
	if apiKey == "" {
		fmt.Println("   ℹ️  Note: Using placeholder API key for compatibility")
	}

	fmt.Print("LLM model name: ")
	llmModel, _ := reader.ReadString('\n')
	llmModel = strings.TrimSpace(llmModel)
	if llmModel == "" {
		return "", fmt.Errorf("LLM model name is required")
	}

	fmt.Print("Embedding model name: ")
	embeddingModel, _ := reader.ReadString('\n')
	embeddingModel = strings.TrimSpace(embeddingModel)
	if embeddingModel == "" {
		return "", fmt.Errorf("embedding model name is required")
	}

	// For custom providers, we always need an API key (even if placeholder)
	if apiKey == "" {
		apiKey = "not-needed" // Placeholder for servers that don't require auth
	}

	config := fmt.Sprintf(`# RAGO - Minimal Configuration
# Custom OpenAI-compatible provider (%s)

# Specify which provider to use by default
[providers]
default_llm = "openai"
default_embedder = "openai"

# Configure OpenAI provider to use your custom server
[providers.openai]
type = "openai"
api_key = "%s"
base_url = "%s"
llm_model = "%s"
embedding_model = "%s"
timeout = "30s"
`, providerName, apiKey, baseURL, llmModel, embeddingModel)

	if enableMCP {
		config += `
# MCP (Model Context Protocol) for advanced tool capabilities
[mcp]
enabled = true
`
	}

	return config, nil
}

// initializeMCPServers creates the MCP servers configuration
func initializeMCPServers(configDir string) error {
	mcpConfigPath := filepath.Join(configDir, "mcpServers.json")

	// Check if MCP config already exists
	if _, err := os.Stat(mcpConfigPath); err == nil {
		return nil // Already exists, skip
	}

	// Create a basic MCP servers configuration
	mcpConfig := `{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/home", "/tmp"],
      "enabled": true
    },
    "git": {
      "command": "uvx",
      "args": ["mcp-server-git", "--repository", "."],
      "enabled": true
    }
  }
}
`

	if err := os.WriteFile(mcpConfigPath, []byte(mcpConfig), 0644); err != nil {
		return fmt.Errorf("failed to create MCP servers configuration: %w", err)
	}

	return nil
}

func init() {
	initCmd.Flags().BoolVarP(&forceInit, "force", "f", false, "overwrite existing configuration file")
	initCmd.Flags().StringVarP(&outputPath, "output", "o", "", "output path for configuration file (default: ./rago.toml)")
	initCmd.Flags().BoolVar(&enableMCP, "enable-mcp", false, "enable MCP servers configuration")
	initCmd.Flags().BoolVar(&skipDatabase, "skip-db", false, "skip database initialization")
}
