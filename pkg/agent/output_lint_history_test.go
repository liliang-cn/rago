package agent

import (
	"testing"

	"github.com/liliang-cn/agent-go/v3/pkg/domain"
)

func TestLintRetryLeavesNoTraceInTheConversation(t *testing.T) {
	nudge := domain.Message{Role: "user", Content: FormatLintFeedback(&LintViolation{LintName: "requested_action_contract", Reason: "call memory_save"})}
	in := []domain.Message{
		{Role: "user", Content: "remember my work"},
		{Role: "assistant", Content: "Done, remembered."}, // the rejected draft
		nudge,
		{Role: "assistant", ToolCalls: []domain.ToolCall{{ID: "c1"}}},
		{Role: "tool", ToolCallID: "c1", Content: "saved"},
		{Role: "assistant", Content: "Saved it."},
	}
	got := conversationMessages(in)
	if len(got) != 4 {
		t.Fatalf("kept %d messages, want 4: %+v", len(got), got)
	}
	for _, m := range got {
		if isLintFeedback(m) || m.Content == "Done, remembered." {
			t.Errorf("kept %q", m.Content)
		}
	}
	if !isLintFeedback(nudge) || isLintFeedback(domain.Message{Role: "user", Content: "hello"}) {
		t.Error("isLintFeedback misjudged")
	}
}
