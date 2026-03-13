package agent

import (
	"testing"

	"github.com/liliang-cn/agent-go/pkg/domain"
)

func TestBuildConversationMessagesIncludesSessionHistory(t *testing.T) {
	session := NewSession("agent-1")
	session.AddMessage(domainMessage("user", "今天有什么新闻？"))
	session.AddMessage(domainMessage("assistant", "我已经给你做了一版摘要。"))

	svc := &Service{}
	messages := svc.buildConversationMessages(session, "筛一版", "", "", "")

	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}
	if messages[0].Role != "user" || messages[0].Content != "今天有什么新闻？" {
		t.Fatalf("unexpected first message: %+v", messages[0])
	}
	if messages[1].Role != "assistant" || messages[1].Content != "我已经给你做了一版摘要。" {
		t.Fatalf("unexpected second message: %+v", messages[1])
	}
	if messages[2].Role != "user" || messages[2].Content != "筛一版" {
		t.Fatalf("unexpected new turn message: %+v", messages[2])
	}
}

func TestBuildConversationMessagesUsesSummaryWhenHistoryEmpty(t *testing.T) {
	svc := &Service{}
	messages := svc.buildConversationMessages(NewSession("agent-1"), "继续", "", "", "之前讨论了今天新闻摘要。")

	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	if messages[0].Role != "user" {
		t.Fatalf("unexpected role: %+v", messages[0])
	}
	if messages[0].Content == "继续" {
		t.Fatalf("expected summary to be prepended, got %q", messages[0].Content)
	}
}

func domainMessage(role, content string) domain.Message {
	return domain.Message{Role: role, Content: content}
}
