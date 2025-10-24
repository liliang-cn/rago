package main

import (
	"context"
	"fmt"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/providers"
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
	if cfg.Providers.ProviderConfigs.OpenAI != nil {
		return checkProviderStatus(ctx, factory, cfg)
	}

	// No provider configured
	fmt.Println("⚠️  No provider configuration found")
	fmt.Println("💡 Please configure an OpenAI-compatible provider in your config file")
	fmt.Println("   Example:")
	fmt.Println("   [providers.provider_configs.openai]")
	fmt.Println("   base_url = \"http://localhost:11434/v1\"")
	fmt.Println("   api_key = \"not-required-for-local-llm\"")
	fmt.Println("   llm_model = \"qwen3:latest\"")
	fmt.Println("   embedding_model = \"text-embedding-ada-002\"")
	return nil
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
				}
			}
		}
	}

	return nil
}
