package main

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/liliang-cn/agent-go/cmd/agentgo-cli/internal/cliui"
	"github.com/liliang-cn/agent-go/pkg/agent"
)

type chatTaskFollower struct {
	manager *agent.SquadManager
	mu      sync.Mutex
	seen    map[string]struct{}
}

func newChatTaskFollower(manager *agent.SquadManager) *chatTaskFollower {
	if manager == nil {
		return nil
	}
	return &chatTaskFollower{
		manager: manager,
		seen:    make(map[string]struct{}),
	}
}

func (f *chatTaskFollower) StartSessionTasks(ctx context.Context, sessionID string) {
	if f == nil || f.manager == nil {
		return
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}

	tasks := f.manager.ListSessionTasks(sessionID, 20)
	for _, task := range tasks {
		if task == nil {
			continue
		}
		f.mu.Lock()
		_, exists := f.seen[task.ID]
		if !exists {
			f.seen[task.ID] = struct{}{}
		}
		f.mu.Unlock()
		if exists {
			continue
		}

		if isTerminalTask(task.Status) {
			printChatTaskSnapshot(task)
			continue
		}
		go f.followTask(ctx, task.ID)
	}
}

func (f *chatTaskFollower) followTask(ctx context.Context, taskID string) {
	events, unsubscribe, err := f.manager.SubscribeTask(taskID)
	if err != nil {
		fmt.Printf("\n%s Task follow failed for %s: %v\n%s", cliui.Error, taskID, err, cliui.UserPrompt)
		return
	}
	defer unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-events:
			if !ok {
				return
			}
			renderChatTaskEvent(evt)
		}
	}
}

func renderChatTaskEvent(evt *agent.TaskEvent) {
	if evt == nil {
		return
	}

	taskLabel := shortTaskID(evt.TaskID)
	switch evt.Type {
	case agent.TaskEventTypeCreated:
		fmt.Printf("\n%s [%s] %s%s", cliui.TaskCreated, taskLabel, firstNonEmpty(evt.Message, "Task created."), chatPromptRestore())
	case agent.TaskEventTypeStarted:
		fmt.Printf("\n%s [%s] %s%s", cliui.TaskStarted, taskLabel, firstNonEmpty(evt.Message, "Task started."), chatPromptRestore())
	case agent.TaskEventTypeRuntime:
		if shouldRenderChatTaskRuntimeEvent() {
			renderRuntimeTaskEvent(taskLabel, evt.Runtime)
		}
	case agent.TaskEventTypeCompleted:
		fmt.Printf("\n%s [%s] Task completed", cliui.Success, taskLabel)
		if evt.AgentName != "" {
			fmt.Printf(" by @%s", evt.AgentName)
		}
		fmt.Println()
		if text := strings.TrimSpace(evt.Message); text != "" {
			fmt.Println(text)
		}
		fmt.Print(chatPromptRestore())
	case agent.TaskEventTypeFailed:
		fmt.Printf("\n%s [%s] Task failed", cliui.Error, taskLabel)
		if evt.AgentName != "" {
			fmt.Printf(" in @%s", evt.AgentName)
		}
		if text := strings.TrimSpace(evt.Message); text != "" {
			fmt.Printf(": %s", text)
		}
		fmt.Print(chatPromptRestore())
	}
}

func renderRuntimeTaskEvent(taskLabel string, evt *agent.Event) {
	if evt == nil {
		return
	}

	switch evt.Type {
	case agent.EventTypeStart, agent.EventTypeStateUpdate:
		if msg := summarizeChatTaskStatus(strings.TrimSpace(evt.Content)); msg != "" {
			fmt.Printf("\n… [%s] @%s %s%s", taskLabel, evt.AgentName, msg, chatPromptRestore())
		}
	case agent.EventTypeToolCall:
		fmt.Printf("\n%s [%s] @%s %s%s", cliui.Tool, taskLabel, evt.AgentName, formatChatTaskToolCall(evt.ToolName), chatPromptRestore())
	case agent.EventTypeToolResult:
		if strings.TrimSpace(evt.Content) != "" {
			fmt.Printf("\n%s [%s] @%s %s: %s%s", cliui.Error, taskLabel, evt.AgentName, evt.ToolName, strings.TrimSpace(evt.Content), chatPromptRestore())
		} else {
			fmt.Printf("\n%s [%s] @%s %s done%s", cliui.ToolDone, taskLabel, evt.AgentName, evt.ToolName, chatPromptRestore())
		}
	}
}

func shouldRenderChatTaskRuntimeEvent() bool {
	return debug || verbose
}

func printChatTaskSnapshot(task *agent.AsyncTask) {
	if task == nil {
		return
	}

	taskLabel := shortTaskID(task.ID)
	switch task.Status {
	case agent.AsyncTaskStatusCompleted:
		fmt.Printf("\n%s [%s] Task completed\n", cliui.Success, taskLabel)
		if text := strings.TrimSpace(task.ResultText); text != "" {
			fmt.Println(text)
		}
		fmt.Print(chatPromptRestore())
	case agent.AsyncTaskStatusFailed:
		fmt.Printf("\n%s [%s] Task failed", cliui.Error, taskLabel)
		if text := strings.TrimSpace(task.Error); text != "" {
			fmt.Printf(": %s", text)
		}
		fmt.Print(chatPromptRestore())
	}
}

func shortTaskID(taskID string) string {
	taskID = strings.TrimSpace(taskID)
	if len(taskID) <= 8 {
		return taskID
	}
	return taskID[:8]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if text := strings.TrimSpace(value); text != "" {
			return text
		}
	}
	return ""
}

func isTerminalTask(status agent.AsyncTaskStatus) bool {
	switch status {
	case agent.AsyncTaskStatusCompleted, agent.AsyncTaskStatusFailed, agent.AsyncTaskStatusCancelled:
		return true
	default:
		return false
	}
}

func chatPromptRestore() string {
	if isInteractive() {
		return "\n" + cliui.UserPrompt
	}
	return "\n"
}

func formatChatTaskToolCall(name string) string {
	if strings.TrimSpace(name) == "" {
		return "starting tool"
	}
	return fmt.Sprintf("using %s", name)
}

func summarizeChatTaskStatus(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return ""
	}
	if strings.HasPrefix(msg, "Starting task:") {
		return "Starting task"
	}
	if strings.HasPrefix(msg, "Starting sub-agent goal:") {
		return "Starting delegated step"
	}
	if strings.Contains(msg, "Executing specific tool:") {
		return ""
	}
	if strings.EqualFold(msg, "Delegated step completed") {
		return ""
	}
	if idx := strings.Index(msg, "\n"); idx >= 0 {
		msg = msg[:idx]
	}
	if len(msg) > 96 {
		return msg[:96] + "..."
	}
	return msg
}
