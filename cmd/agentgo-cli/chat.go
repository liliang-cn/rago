package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/liliang-cn/agent-go/pkg/agent"
	"github.com/liliang-cn/agent-go/pkg/config"
	"github.com/spf13/cobra"
)

var (
	chatSessionID  string
	chatStream     bool
	chatModel      string
	chatWithPTC    bool
	chatNoMemory   bool
	chatShowMemory bool
)

var chatCmd = &cobra.Command{
	Use:   "chat [message]",
	Short: "Chat with AgentGo Agent (conversational AI with memory)",
	Long: `Interactive chat with the AgentGo Agent that maintains conversation context across messages.

The agent has access to:
- MCP Tools (external integrations)
- Skills (domain-specific capabilities)
- Memory (long-term factual memory)
- RAG (knowledge base retrieval)
- PTC (Programmatic Tool Calling - JS sandbox)

Examples:
  agentgo chat "你好"
  agentgo chat --with-ptc "比较三个城市的旅行预算"
  agentgo chat --show-memory "我之前说过我喜欢什么颜色？"
  agentgo chat --no-memory "临时不要记得这次对话内容"
  agentgo chat  # Interactive mode`,
	RunE: runChat,
}

func init() {
	RootCmd.AddCommand(chatCmd)
	chatCmd.Flags().StringVarP(&chatSessionID, "session", "s", "", "Session ID for conversation (default: auto-generated)")
	chatCmd.Flags().BoolVarP(&chatStream, "stream", "", false, "Stream the response")
	chatCmd.Flags().StringVarP(&chatModel, "model", "m", "", "LLM model to use")
	chatCmd.Flags().BoolVar(&chatWithPTC, "with-ptc", false, "Enable Programmatic Tool Calling (JS sandbox)")
	chatCmd.Flags().BoolVar(&chatNoMemory, "no-memory", false, "Disable long-term memory for this chat")
	chatCmd.Flags().BoolVar(&chatShowMemory, "show-memory", false, "Show retrieved memories in output")
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
	builder := agent.New("agentgo-assistant").
		WithDBPath(agentDBPath).
		WithMCP().
		WithSkills().
		WithRouter().
		WithDebug(debug).
		WithProgress(progressCallback)

	// Memory is enabled by default unless --no-memory is set
	if !chatNoMemory {
		builder.WithMemory()
	}

	if chatWithPTC {
		builder.WithPTC()
	}

	svc, err := builder.Build()
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

	displayResult(result)

	// Show session ID after first message
	if currentSessionID == "" {
		newSessionID := svc.CurrentSessionID()
		fmt.Printf("\n💡 Session: %s (use --session %s to continue)\n", newSessionID, newSessionID)
	}

	return nil
}

func displayResult(result *agent.ExecutionResult) {
	// Show memories if requested
	if (chatShowMemory || verbose) && len(result.Memories) > 0 {
		fmt.Printf("\n🧠 Retrieved Memories (%d):\n", len(result.Memories))
		for i, mem := range result.Memories {
			sourceTag := ""
			if mem.SourceType != "" {
				sourceTag = fmt.Sprintf(" src:%s", mem.SourceType)
			}
			confTag := ""
			if mem.Confidence > 0 {
				confTag = fmt.Sprintf(" conf:%.2f", mem.Confidence)
			}
			evTag := ""
			if len(mem.EvidenceIDs) > 0 {
				evTag = fmt.Sprintf(" evidence:%d", len(mem.EvidenceIDs))
			}
			fmt.Printf("  %d. [%s%s%s%s] %s (score: %.2f)\n",
				i+1, mem.Type, sourceTag, confTag, evTag,
				truncateString(mem.Content, 100), mem.Score)
		}
		if result.MemoryLogic != "" {
			fmt.Printf("  💡 Navigator reasoning: %s\n", truncateString(result.MemoryLogic, 200))
		}
	}

	if result.PTCResult != nil && result.PTCResult.Type != agent.PTCResultTypeText {
		fmt.Printf("\n🤖 AgentGo (PTC Mode):\n%s\n", result.PTCResult.FormatForLLM())
	} else if result.FinalResult != nil {
		fmt.Printf("\n🤖 AgentGo: %v\n", result.FinalResult)
	} else {
		fmt.Println("\n🤖 AgentGo: (empty response)")
	}
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
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
	fmt.Println("🤖 AgentGo Chat Mode")
	if chatWithPTC {
		fmt.Println("⚡ PTC Mode: Enabled (JS sandbox for complex logic)")
	}
	if chatNoMemory {
		fmt.Println("🔇 Memory Mode: Disabled")
	} else if chatShowMemory || verbose {
		fmt.Println("🧠 Memory Mode: Enabled (Showing retrievals)")
	}
	fmt.Println("💡 Type 'quit' or 'exit' to end, 'clear' to reset session")
	fmt.Println()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	var wg sync.WaitGroup
	quitChan := make(chan struct{})

	// Use scanner for multi-word input
	scanner := bufio.NewScanner(os.Stdin)

	// Input goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			fmt.Print("👤 You: ")
			if !scanner.Scan() {
				return
			}
			input := strings.TrimSpace(scanner.Text())

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

			displayResult(result)
			fmt.Println()
		}
	}()

	// Wait for quit or interrupt
	select {
	case <-quitChan:
		// Normal exit
	case <-sigChan:
		// User pressed Ctrl+C
		fmt.Println("\n\n👋 Interrupted. Goodbye!")
	}

	return nil
}
