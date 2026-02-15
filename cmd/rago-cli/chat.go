package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/liliang-cn/rago/v2/pkg/agent"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/spf13/cobra"
)

var (
	chatSessionID string
	chatStream     bool
	chatModel      string
)

var chatCmd = &cobra.Command{
	Use:   "chat [message]",
	Short: "Chat with RAGO Agent (conversational AI with memory)",
	Long: `Interactive chat with the RAGO Agent that maintains conversation context across messages.

The agent has access to:
- MCP Tools (external integrations)
- Skills (domain-specific capabilities)
- Memory (long-term factual memory)
- RAG (knowledge base retrieval)

Examples:
  rago chat "你好"
  rago chat --session my-session "记住：我喜欢蓝色"
  rago chat --stream "介绍一下你自己"
  rago chat  # Interactive mode`,
	RunE: runChat,
}

func init() {
	RootCmd.AddCommand(chatCmd)
	chatCmd.Flags().StringVarP(&chatSessionID, "session", "s", "", "Session ID for conversation (default: auto-generated)")
	chatCmd.Flags().BoolVarP(&chatStream, "stream", "", false, "Stream the response")
	chatCmd.Flags().StringVarP(&chatModel, "model", "m", "", "LLM model to use")
}

func runChat(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load config
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize agent service
	agentDBPath := cfg.DataDir() + "/agent.db"

	// Create agent with full capabilities
	svc, err := agent.New(&agent.AgentConfig{
		Name:         "rago-assistant",
		DBPath:       agentDBPath,
		EnableMCP:    true,
		EnableSkills: true,
		EnableRouter: true,
		EnableMemory: true,
		EnableRAG:    false, // Requires embedding pool to be enabled
		ProgressCb:   progressCallback,
	})
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}
	defer svc.Close()

	// Set session if specified
	if chatSessionID != "" {
		svc.SetSessionID(chatSessionID)
	}

	// Get current session ID
	currentSessionID := svc.CurrentSessionID()
	if currentSessionID != "" {
		fmt.Printf("📝 Session: %s\n", currentSessionID)
	}

	// Interactive mode (no arguments)
	if len(args) == 0 {
		return runInteractiveChat(ctx, svc)
	}

	// Single message mode
	message := strings.Join(args, " ")
	fmt.Printf("\n🤔 You: %s\n", message)

	result, err := svc.Chat(ctx, message)
	if err != nil {
		return fmt.Errorf("chat failed: %w", err)
	}

	if result.FinalResult != nil {
		fmt.Printf("\n🤖 RAGO: %v\n", result.FinalResult)
	}

	// Show session ID after first message
	if currentSessionID == "" {
		newSessionID := svc.CurrentSessionID()
		fmt.Printf("\n💡 Session: %s (use --session %s to continue)\n", newSessionID, newSessionID)
	}

	return nil
}

// progressCallback displays agent progress
func progressCallback(event agent.ProgressEvent) {
	switch event.Type {
	case "thinking":
		fmt.Printf("  🤔 %s\n", event.Message)
	case "tool_call":
		if event.Tool != "" {
			fmt.Printf("  🔧 %s\n", event.Message)
		}
	case "tool_result":
		fmt.Printf("  ✓ %s\n", event.Message)
	}
}

func runInteractiveChat(ctx context.Context, svc *agent.Service) error {
	fmt.Println("🤖 RAGO Chat Mode")
	fmt.Println("💡 Type 'quit' or 'exit' to end, 'clear' to reset session")
	fmt.Println()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	var wg sync.WaitGroup
	quitChan := make(chan struct{})

	// Input goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-quitChan:
				return
			default:
				fmt.Print("👤 You: ")
				var input string
				fmt.Scanln(&input)
				input = strings.TrimSpace(input)

				if input == "" {
					continue
				}

				// Exit commands
				if input == "quit" || input == "exit" || input == "q" {
					close(quitChan)
					fmt.Println("\n👋 Goodbye!")
					return
				}

				// Clear session
				if input == "clear" || input == "reset" {
					svc.ResetSession()
					fmt.Printf("✓ Session reset (new: %s)\n", svc.CurrentSessionID())
					continue
				}

				// Process message
				fmt.Printf("\n🤔 Thinking...\n")
				result, err := svc.Chat(ctx, input)
				if err != nil {
					fmt.Printf("❌ Error: %v\n\n", err)
					continue
				}

				if result.FinalResult != nil {
					fmt.Printf("\n🤖 RAGO: %v\n\n", result.FinalResult)
				} else {
					fmt.Println("\n🤖 RAGO: (empty response)")
				}
			}
		}
	}()

	// Wait for quit or interrupt
	select {
	case <-quitChan:
		// Normal exit
	case <-sigChan:
		// User pressed Ctrl+C
		close(quitChan)
		fmt.Println("\n\n👋 Interrupted. Goodbye!")
	}

	wg.Wait()
	return nil
}
