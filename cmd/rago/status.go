package rago

import (
	"context"
	"fmt"
	"time"

	"github.com/liliang-cn/rago/internal/llm"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the status of Ollama connection",
	Long:  `Check if Ollama service is available and can be connected to.`,
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create Ollama service
	ollamaService, err := llm.NewOllamaService(cfg.Ollama.BaseURL, cfg.Ollama.LLMModel)

	if err != nil {
		fmt.Printf("❌ Failed to create Ollama client: %v\n", err)
		return nil
	}

	// Check Ollama health
	fmt.Printf("🔍 Checking Ollama connection to %s...\n", cfg.Ollama.BaseURL)

	if err := ollamaService.Health(ctx); err != nil {
		fmt.Printf("❌ Ollama connection failed: %v\n", err)
		return nil
	}

	fmt.Printf("✅ Ollama is available at %s\n", cfg.Ollama.BaseURL)
	fmt.Printf("📋 Configuration:\n")
	fmt.Printf("   • LLM Model: %s\n", cfg.Ollama.LLMModel)
	fmt.Printf("   • Embedding Model: %s\n", cfg.Ollama.EmbeddingModel)
	fmt.Printf("   • Timeout: %s\n", cfg.Ollama.Timeout)

	return nil
}

func init() {
	RootCmd.AddCommand(statusCmd)
}
