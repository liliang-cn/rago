package agent

import "strings"

const FinishOrBlockContract = `Finish-Or-Block Contract:
- Do the task now; do not end on a plan or "next steps" unless you need input. Finish with the verified result (with evidence when you used tools), or call task_blocked with the concrete blocker and what you tried.`

// defaultBlockedText is what a blocked run reports when neither the model's
// task_blocked arguments nor its assistant text said anything. Silence is the
// worst outcome here — the user gets an empty reply and cannot tell a refusal
// from a crash — so every blocked path degrades to this sentence instead.
const defaultBlockedText = "Task blocked: no specific blocker was reported."

func isTaskTerminalToolName(name string) bool {
	switch strings.TrimSpace(name) {
	case "task_complete", "task_blocked":
		return true
	default:
		return false
	}
}

func taskTerminalToolResult(toolName string, args map[string]interface{}, fallback string) string {
	if args == nil {
		return strings.TrimSpace(fallback)
	}
	switch strings.TrimSpace(toolName) {
	case "task_blocked":
		for _, key := range []string{"blocker", "reason", "result"} {
			if text, ok := args[key].(string); ok && strings.TrimSpace(text) != "" {
				return strings.TrimSpace(text)
			}
		}
	case "task_complete":
		if text, ok := args["result"].(string); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return strings.TrimSpace(fallback)
}
