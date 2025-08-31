package rago

import (
	"context"
	"fmt"
	"time"

	"github.com/liliang-cn/rago/pkg/config"
	"github.com/liliang-cn/rago/pkg/domain"
	"github.com/liliang-cn/rago/pkg/embedder"
	"github.com/liliang-cn/rago/pkg/llm"
	"github.com/liliang-cn/rago/pkg/providers"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the status of LLM provider connections",
	Long:  `Check if configured LLM providers are available and can be connected to.`,
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize provider factory
	factory := providers.NewFactory()

	// Check if using new provider configuration
	if cfg.Providers.ProviderConfigs.Ollama != nil || cfg.Providers.ProviderConfigs.OpenAI != nil {
		return checkProviderStatus(ctx, factory, cfg)
	}

	// Fallback to legacy configuration
	return checkLegacyStatus(ctx, cfg)
}

func checkProviderStatus(ctx context.Context, factory *providers.Factory, cfg *config.Config) error {
	fmt.Println("🔍 Checking provider status...")

	// Check LLM provider
	if cfg.Providers.DefaultLLM != "" {
		fmt.Printf("\n📝 LLM Provider (%s):\n", cfg.Providers.DefaultLLM)
		
		llmConfig, err := providers.GetLLMProviderConfig(&cfg.Providers.ProviderConfigs, cfg.Providers.DefaultLLM)
		if err != nil {
			fmt.Printf("❌ Failed to get LLM config: %v\n", err)
		} else {
			provider, err := factory.CreateLLMProvider(ctx, llmConfig)
			if err != nil {
				fmt.Printf("❌ Failed to create LLM provider: %v\n", err)
			} else {
				if err := provider.Health(ctx); err != nil {
					fmt.Printf("❌ LLM provider health check failed: %v\n", err)
				} else {
					fmt.Printf("✅ LLM provider is healthy\n")
					printProviderConfig(cfg.Providers.DefaultLLM, llmConfig)
				}
			}
		}
	}

	// Check Embedder provider
	if cfg.Providers.DefaultEmbedder != "" {
		fmt.Printf("\n🔢 Embedder Provider (%s):\n", cfg.Providers.DefaultEmbedder)
		
		embedderConfig, err := providers.GetEmbedderProviderConfig(&cfg.Providers.ProviderConfigs, cfg.Providers.DefaultEmbedder)
		if err != nil {
			fmt.Printf("❌ Failed to get embedder config: %v\n", err)
		} else {
			provider, err := factory.CreateEmbedderProvider(ctx, embedderConfig)
			if err != nil {
				fmt.Printf("❌ Failed to create embedder provider: %v\n", err)
			} else {
				if err := provider.Health(ctx); err != nil {
					fmt.Printf("❌ Embedder provider health check failed: %v\n", err)
				} else {
					fmt.Printf("✅ Embedder provider is healthy\n")
					printProviderConfig(cfg.Providers.DefaultEmbedder, embedderConfig)
				}
			}
		}
	}

	return nil
}

func printProviderConfig(providerType string, config interface{}) {
	switch providerType {
	case "ollama":
		if ollamaConfig, ok := config.(*domain.OllamaProviderConfig); ok {
			fmt.Printf("   • Base URL: %s\n", ollamaConfig.BaseURL)
			fmt.Printf("   • LLM Model: %s\n", ollamaConfig.LLMModel)
			fmt.Printf("   • Embedding Model: %s\n", ollamaConfig.EmbeddingModel)
			fmt.Printf("   • Timeout: %s\n", ollamaConfig.Timeout)
		}
	case "openai":
		if openaiConfig, ok := config.(*domain.OpenAIProviderConfig); ok {
			fmt.Printf("   • Base URL: %s\n", openaiConfig.BaseURL)
			fmt.Printf("   • LLM Model: %s\n", openaiConfig.LLMModel)
			fmt.Printf("   • Embedding Model: %s\n", openaiConfig.EmbeddingModel)
			fmt.Printf("   • API Key: %s\n", maskAPIKey(openaiConfig.APIKey))
			fmt.Printf("   • Timeout: %s\n", openaiConfig.Timeout)
		}
	}
}

func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "****"
	}
	return apiKey[:4] + "****" + apiKey[len(apiKey)-4:]
}

func checkLegacyStatus(ctx context.Context, cfg *config.Config) error {
	fmt.Println("🔍 Checking legacy Ollama configuration...")
	fmt.Printf("🔍 Checking Ollama connection to %s...\n", cfg.Ollama.BaseURL)

	// Create legacy Ollama service using new provider system
	factory := providers.NewFactory()
	
	legacyConfig := &domain.OllamaProviderConfig{
		BaseProviderConfig: domain.BaseProviderConfig{
			Type:    domain.ProviderOllama,
			Timeout: cfg.Ollama.Timeout,
		},
		BaseURL:        cfg.Ollama.BaseURL,
		LLMModel:       cfg.Ollama.LLMModel,
		EmbeddingModel: cfg.Ollama.EmbeddingModel,
	}

	// Check LLM provider
	llmProvider, err := factory.CreateLLMProvider(ctx, legacyConfig)
	if err != nil {
		fmt.Printf("❌ Failed to create Ollama LLM provider: %v\n", err)
		return nil
	}

	ollamaService := llm.NewService(llmProvider)
	
	// Check Ollama health
	if err := ollamaService.Health(ctx); err != nil {
		fmt.Printf("❌ Ollama connection failed: %v\n", err)
		return nil
	}

	fmt.Printf("✅ Ollama is available at %s\n", cfg.Ollama.BaseURL)
	fmt.Printf("📋 Configuration:\n")
	fmt.Printf("   • LLM Model: %s\n", cfg.Ollama.LLMModel)
	fmt.Printf("   • Embedding Model: %s\n", cfg.Ollama.EmbeddingModel)
	fmt.Printf("   • Timeout: %s\n", cfg.Ollama.Timeout)

	// Check embedder
	embedderProvider, err := factory.CreateEmbedderProvider(ctx, legacyConfig)
	if err != nil {
		fmt.Printf("⚠️  Failed to create embedder provider: %v\n", err)
		return nil
	}

	embedService := embedder.NewService(embedderProvider)
	
	if err := embedService.Health(ctx); err != nil {
		fmt.Printf("⚠️  Embedder health check failed: %v\n", err)
	} else {
		fmt.Printf("✅ Embedder is healthy\n")
	}

	return nil
}

func init() {
	RootCmd.AddCommand(statusCmd)
}
