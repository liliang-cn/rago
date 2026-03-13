package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"

	"github.com/liliang-cn/agent-go/pkg/agent"
	"github.com/liliang-cn/agent-go/pkg/config"
	"github.com/spf13/cobra"
)

func delegatedResultLooksFailed(result string) bool {
	normalized := strings.ToLower(strings.TrimSpace(result))
	return strings.HasPrefix(normalized, "code execution failed:") ||
		strings.HasPrefix(normalized, "execute_javascript failed:") ||
		strings.HasPrefix(normalized, "task failed:") ||
		strings.Contains(normalized, "\n**status:** failed")
}

var (
	chatSessionID  string
	chatStream     bool
	chatWithPTC    bool
	chatNoMemory   bool
	chatShowMemory bool
)

type delegatedTask struct {
	AgentName   string
	Instruction string
}

type chatRequest struct {
	Input string
	Done  chan struct{}
}

var agentMentionPattern = regexp.MustCompile(`^@([^\s@]+)$`)

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
	chatCmd.Flags().BoolVar(&chatWithPTC, "with-ptc", false, "Enable Programmatic Tool Calling (JS sandbox)")
	chatCmd.Flags().BoolVar(&chatNoMemory, "no-memory", false, "Disable long-term memory for this chat")
	chatCmd.Flags().BoolVar(&chatShowMemory, "show-memory", false, "Show retrieved memories in output")
}

func runChat(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	chatCfg := cfg
	if chatCfg == nil {
		var err error
		chatCfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	agentDBPath := chatCfg.DataDir() + "/agent.db"

	var agentManager *agent.SquadManager
	agentStore, storeErr := agent.NewStore(agentDBPath)
	if storeErr == nil {
		agentManager = agent.NewTeamManager(agentStore)
		if err := agentManager.SeedDefaultMembers(); err != nil {
			agentManager = nil
		}
	}

	svc, err := buildChatConciergeService(chatCfg, agentDBPath, agentManager)
	if err != nil {
		return fmt.Errorf("failed to build concierge service: %w", err)
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
		return runInteractiveChat(ctx, svc, agentManager)
	}

	// Single message mode
	message := strings.Join(args, " ")
	fmt.Printf("\n🤔 You: %s\n", message)

	if agentManager != nil {
		tasks, parseErr := parseDelegatedTasks(message, func(name string) bool {
			_, err := agentManager.GetAgentByName(name)
			return err == nil
		})
		if parseErr != nil {
			return parseErr
		}
		if len(tasks) > 0 {
			return runDelegatedTaskChain(context.Background(), agentManager, tasks, false)
		}
	}

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

func buildChatConciergeService(chatCfg *config.Config, agentDBPath string, manager *agent.SquadManager) (*agent.Service, error) {
	if manager != nil && !chatNoMemory && !chatWithPTC {
		if svc, err := manager.GetAgentService(agent.BuiltInConciergeAgentName); err == nil {
			svc.SetDebug(debug)
			svc.SetProgressCallback(progressCallback)
			manager.RegisterConciergeTools(svc)
			return svc, nil
		}
	}

	systemPrompt := "You are Concierge, the always-on intake agent for AgentGo. Accept user requests, clarify ambiguous asks, answer simple questions directly, inspect squad or agent status, and submit squad work when deeper execution is needed. Prefer lightweight orchestration over doing heavy work yourself, acknowledge queued work clearly, and never pretend background work is already finished."
	if manager != nil {
		if model, err := manager.GetAgentByName(agent.BuiltInConciergeAgentName); err == nil && strings.TrimSpace(model.Instructions) != "" {
			systemPrompt = strings.TrimSpace(model.Instructions)
		}
	}

	builder := agent.New(agent.BuiltInConciergeAgentName).
		WithConfig(chatCfg).
		WithSystemPrompt(systemPrompt).
		WithDBPath(agentDBPath).
		WithDebug(debug).
		WithProgress(progressCallback)

	if !chatNoMemory {
		builder.WithMemory()
	}
	if chatWithPTC {
		builder.WithPTC()
	}

	svc, err := builder.Build()
	if err != nil {
		return nil, err
	}
	if manager != nil {
		manager.RegisterConciergeTools(svc)
	}
	return svc, nil
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
		fmt.Printf("\n🛎 Concierge (PTC Mode):\n%s\n", result.PTCResult.FormatForLLM())
	} else if result.FinalResult != nil {
		fmt.Printf("\n🛎 Concierge: %v\n", result.FinalResult)
	} else {
		fmt.Println("\n🛎 Concierge: (empty response)")
	}

	if result != nil {
		if result.StartedAt != nil {
			fmt.Printf("Started: %s\n", result.StartedAt.Format("2006-01-02 15:04:05"))
		}
		if result.CompletedAt != nil {
			fmt.Printf("Completed: %s\n", result.CompletedAt.Format("2006-01-02 15:04:05"))
		}
		if result.EstimatedTokens > 0 {
			fmt.Printf("Estimated tokens: %d\n", result.EstimatedTokens)
		}
		if result.ToolCalls > 0 {
			fmt.Printf("Tool calls: %d\n", result.ToolCalls)
		}
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
		fmt.Printf("  … %s\n", event.Message)
	case "tool_call":
		if event.Tool != "" {
			fmt.Printf("  🛠 %s\n", event.Message)
		}
	case "tool_result":
		fmt.Printf("  ✓ %s\n", event.Message)
	}
}

func runInteractiveChat(ctx context.Context, svc *agent.Service, manager *agent.SquadManager) error {
	fmt.Println("🛎 AgentGo Concierge Chat")
	if chatWithPTC {
		fmt.Println("⚡ PTC Mode: Enabled (JS sandbox for complex logic)")
	}
	if chatNoMemory {
		fmt.Println("🔇 Memory Mode: Disabled")
	} else if chatShowMemory || verbose {
		fmt.Println("🧠 Memory Mode: Enabled (Showing retrievals)")
	}
	fmt.Println("💡 This chat talks to Concierge, the always-on intake agent for AgentGo.")
	fmt.Println("💡 Type 'quit' or 'exit' to end, 'clear' to reset session")
	fmt.Println("💡 Tip: Use '@AgentName <instruction>' to run a saved agent in the background")
	fmt.Println()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	quitChan := make(chan struct{})
	requests := make(chan chatRequest)

	// Use scanner for multi-word input
	scanner := bufio.NewScanner(os.Stdin)

	go func() {
		for {
			select {
			case <-quitChan:
				return
			case req, ok := <-requests:
				if !ok {
					return
				}
				fmt.Printf("\n🤔 Thinking...\n")
				result, err := svc.Chat(ctx, req.Input)
				if err != nil {
					fmt.Printf("❌ Error: %v\n\n", err)
				} else {
					displayResult(result)
					fmt.Println()
				}
				if req.Done != nil {
					close(req.Done)
				}
			}
		}
	}()

	// Input goroutine
	go func() {
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

			if manager != nil {
				tasks, parseErr := parseDelegatedTasks(input, func(name string) bool {
					_, err := manager.GetAgentByName(name)
					return err == nil
				})
				if parseErr != nil {
					fmt.Printf("❌ %v\n\n", parseErr)
					continue
				}
				if len(tasks) > 0 {
					fmt.Printf("\n🚀 Delegating %d task(s) in background...\n", len(tasks))

					go func(parsedTasks []delegatedTask) {
						if err := runDelegatedTaskChain(context.Background(), manager, parsedTasks, true); err != nil {
							fmt.Printf("\n❌ %v\n\n", err)
						}
					}(tasks)

					continue
				}
			}

			// Process message normally (Concierge handling)
			done := make(chan struct{})
			requests <- chatRequest{Input: input, Done: done}
			<-done
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

func parseDelegatedTasks(input string, isKnownAgent func(name string) bool) ([]delegatedTask, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, nil
	}

	words := strings.Fields(trimmed)
	if len(words) == 0 {
		return nil, nil
	}

	firstName, ok := parseMentionedAgent(words[0])
	if !ok {
		return nil, nil
	}
	if isKnownAgent != nil && !isKnownAgent(firstName) {
		return nil, fmt.Errorf("unknown agent: %s", firstName)
	}

	tasks := make([]delegatedTask, 0, 2)
	current := delegatedTask{AgentName: firstName}

	for _, word := range words[1:] {
		if nextName, isMention := parseMentionedAgent(word); isMention {
			if isKnownAgent != nil && isKnownAgent(nextName) {
				current.Instruction = strings.TrimSpace(current.Instruction)
				if current.Instruction == "" {
					return nil, fmt.Errorf("please provide an instruction for %s", current.AgentName)
				}
				tasks = append(tasks, current)
				current = delegatedTask{AgentName: nextName}
				continue
			}
		}

		if current.Instruction == "" {
			current.Instruction = word
		} else {
			current.Instruction += " " + word
		}
	}

	current.Instruction = strings.TrimSpace(current.Instruction)
	if current.Instruction == "" {
		return nil, fmt.Errorf("please provide an instruction for %s", current.AgentName)
	}
	tasks = append(tasks, current)

	return tasks, nil
}

func parseMentionedAgent(word string) (string, bool) {
	matches := agentMentionPattern.FindStringSubmatch(word)
	if len(matches) != 2 {
		return "", false
	}
	return matches[1], true
}

func runDelegatedTaskChain(ctx context.Context, manager *agent.SquadManager, tasks []delegatedTask, background bool) error {
	if manager == nil {
		return fmt.Errorf("agent manager is not initialized")
	}
	if len(tasks) == 0 {
		return nil
	}

	var previousResult string
	for idx, task := range tasks {
		if background {
			fmt.Printf("\n🚀 Background delegation %d/%d -> %s...\n", idx+1, len(tasks), task.AgentName)
		} else {
			fmt.Printf("\n🚀 Delegating %d/%d to %s...\n", idx+1, len(tasks), task.AgentName)
		}

		instruction := task.Instruction
		if previousResult != "" {
			instruction = fmt.Sprintf(
				"Previous result from @%s:\n%s\n\nYour task:\n%s",
				tasks[idx-1].AgentName,
				previousResult,
				task.Instruction,
			)
		}

		res, err := manager.DispatchTask(ctx, task.AgentName, instruction)
		if err != nil {
			if background {
				fmt.Printf("\n❌ Background task failed for @%s: %v\n\n", task.AgentName, err)
				return nil
			}
			return fmt.Errorf("background task failed for @%s: %w", task.AgentName, err)
		}

		if delegatedResultLooksFailed(res) {
			if background {
				fmt.Printf("\n❌ Background task failed for @%s:\n%v\n\n", task.AgentName, res)
				return nil
			}
			fmt.Printf("\n❌ Task failed for @%s:\n%v\n\n", task.AgentName, res)
			return nil
		}

		if background {
			fmt.Printf("\n✅ Background task completed by @%s:\n%v\n", task.AgentName, res)
		} else {
			fmt.Printf("\n✅ Task completed by @%s:\n%v\n", task.AgentName, res)
		}

		previousResult = res
	}

	if !background {
		fmt.Println()
	}

	return nil
}
