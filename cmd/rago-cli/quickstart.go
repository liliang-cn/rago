package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/liliang-cn/rago/v2/client"
	"github.com/spf13/cobra"
)

var quickstartCmd = &cobra.Command{
	Use:   "quickstart",
	Short: "Quick start guide for RAGO platform",
	Long: `Interactive quick start guide to get you up and running with RAGO.
This command will help you:
  1. Check your configuration
  2. Test LLM connectivity
  3. Try basic operations
  4. Learn about platform capabilities`,
	RunE: runQuickstart,
}

func runQuickstart(cmd *cobra.Command, args []string) error {
	fmt.Println("=== RAGO Platform Quick Start ===")
	fmt.Println()

	// Step 1: Check configuration
	fmt.Println("Step 1: Checking configuration...")
	configPath := cfgFile
	if configPath == "" {
		home, _ := os.UserHomeDir()
		defaultPath := home + "/.rago/rago.toml"
		if _, err := os.Stat(defaultPath); err == nil {
			configPath = defaultPath
			fmt.Printf("✓ Found configuration at: %s\n", configPath)
		} else {
			fmt.Println("✗ No configuration found")
			fmt.Println("  Run 'rago init' to create one")
			return nil
		}
	}

	// Step 2: Initialize client
	fmt.Println("\nStep 2: Initializing RAGO platform...")
	rago, err := client.New(configPath)
	if err != nil {
		fmt.Printf("✗ Failed to initialize: %v\n", err)
		fmt.Println("  Check your configuration and provider settings")
		return nil
	}
	defer rago.Close()
	fmt.Println("✓ Platform initialized")

	// Step 3: Test LLM
	fmt.Println("\nStep 3: Testing LLM connectivity...")
	response, err := rago.LLM.Generate("Say 'Hello RAGO!' if you can hear me")
	if err != nil {
		fmt.Printf("✗ LLM test failed: %v\n", err)
		fmt.Println("  Make sure your LLM provider is running (e.g., 'ollama serve')")
		return nil
	}
	fmt.Printf("✓ LLM responds: %s\n", response)

	// Step 4: Show capabilities
	fmt.Println("\n=== Platform Capabilities ===")
	fmt.Println()

	capabilities := []struct {
		name    string
		command string
		desc    string
	}{
		{
			"LLM (Language Models)",
			"rago platform generate \"Your prompt here\"",
			"Generate text, chat, and stream responses",
		},
		{
			"RAG (Knowledge Base)",
			"rago rag ingest ./docs && rago platform query \"Your question\"",
			"Build knowledge bases and query them",
		},
		{
			"Tools (MCP Integration)",
			"rago platform tools",
			"Use external tools via MCP protocol",
		},
		{
			"Agent (Automation)",
			"rago platform run \"Your task description\"",
			"Execute complex tasks autonomously",
		},
	}

	for i, cap := range capabilities {
		fmt.Printf("%d. %s\n", i+1, cap.name)
		fmt.Printf("   %s\n", cap.desc)
		fmt.Printf("   Try: %s\n\n", cap.command)
	}

	// Step 5: Interactive demo
	fmt.Println("=== Try It Now ===")
	fmt.Println("Would you like to try a simple example? (y/n)")

	var input string
	fmt.Scanln(&input)

	if strings.ToLower(input) == "y" {
		fmt.Println("\nLet's generate a simple haiku about coding:")
		fmt.Println("----------------------------------------")

		haiku, err := rago.LLM.GenerateWithOptions(
			context.Background(),
			"Write a haiku about programming in Go",
			&client.GenerateOptions{
				Temperature: 0.9,
				MaxTokens:   100,
			},
		)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Println(haiku)
		}
		fmt.Println("----------------------------------------")
	}

	// Final guidance
	fmt.Println("\n=== Next Steps ===")
	fmt.Println("1. Explore examples: cd examples && ls")
	fmt.Println("2. Read documentation: cat README.md")
	fmt.Println("3. Run full demo: rago platform demo")
	fmt.Println("4. Get help: rago --help")
	fmt.Println("\nHappy coding with RAGO! 🚀")

	return nil
}

func init() {
	RootCmd.AddCommand(quickstartCmd)
}
