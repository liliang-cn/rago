package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/services"
	"github.com/spf13/cobra"
)

var (
	llmProvider string
	llmModel    string
	llmPrompt   string
	llmStream   bool
)

// llmCmd represents the LLM command group
var llmCmd = &cobra.Command{
	Use:   "llm",
	Short: "LLM operations - language model interactions",
	Long:  `Commands for interacting with language models through various providers.`,
}

// llmChatCmd handles chat interactions
var llmChatCmd = &cobra.Command{
	Use:   "chat [message]",
	Short: "Chat with an LLM",
	Long:  `Send a message to an LLM and get a response.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		message := strings.Join(args, " ")

		ctx := context.Background()

		// Use the global cfg which is of type *config.Config
		if cfg == nil {
			return fmt.Errorf("configuration not loaded")
		}

		// Override provider if specified
		if llmProvider != "" {
			cfg.Providers.DefaultLLM = llmProvider
		}

		// Get global LLM service
		llmService, err := services.GetGlobalLLM()
		if err != nil {
			return fmt.Errorf("failed to get global LLM service: %w", err)
		}

		// Create generation options
		opts := &domain.GenerationOptions{
			MaxTokens:   30000,
			Temperature: 0.7,
		}

		fmt.Println("🤖 LLM Response:")
		fmt.Println("================")

		// Execute generation
		resp, err := llmService.Generate(ctx, message, opts)
		if err != nil {
			return fmt.Errorf("generation failed: %w", err)
		}
		fmt.Println(resp)

		return nil
	},
}

// llmListCmd lists available models
var llmListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available LLM models",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Use the global cfg which is of type *config.Config
		if cfg == nil {
			return fmt.Errorf("configuration not loaded")
		}

		fmt.Println("🤖 Available LLM Providers and Models")
		fmt.Println("=====================================")

		// List OpenAI provider if configured (compatible with all OpenAI-format LLMs)
		if cfg.Providers.ProviderConfigs.OpenAI != nil {
			fmt.Printf("\n📦 OpenAI (Compatible Format)\n")
			fmt.Printf("   URL: %s\n", cfg.Providers.ProviderConfigs.OpenAI.BaseURL)
			fmt.Printf("   LLM Model: %s\n", cfg.Providers.ProviderConfigs.OpenAI.LLMModel)
			fmt.Printf("   Embedding Model: %s\n", cfg.Providers.ProviderConfigs.OpenAI.EmbeddingModel)
		}

		fmt.Printf("\n⭐ Default LLM Provider: %s\n", cfg.Providers.DefaultLLM)
		fmt.Printf("⭐ Default Embedder Provider: %s\n", cfg.Providers.DefaultEmbedder)

		return nil
	},
}

func init() {
	// Add subcommands
	llmCmd.AddCommand(llmChatCmd)
	llmCmd.AddCommand(llmListCmd)

	// Chat flags
	llmChatCmd.Flags().StringVarP(&llmProvider, "provider", "p", "", "LLM provider to use")
	llmChatCmd.Flags().StringVarP(&llmModel, "model", "m", "", "Model to use")
	llmChatCmd.Flags().BoolVarP(&llmStream, "stream", "s", false, "Stream the response")
}
