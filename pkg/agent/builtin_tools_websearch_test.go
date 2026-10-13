package agent

import (
	"context"
	"testing"

	"github.com/liliang-cn/agent-go/v3/pkg/domain"
)

func TestBuildWebSearchBody(t *testing.T) {
	b := buildWebSearchBody(WebSearchConfig{Model: "qwen-plus"}, "今天 SpaceX 新闻")
	if b["enable_search"] != true {
		t.Fatalf("expected enable_search=true")
	}
	if _, ok := b["web_search_options"]; !ok {
		t.Fatalf("expected web_search_options present")
	}
	if b["model"] != "qwen-plus" {
		t.Fatalf("model not set")
	}
	msgs, ok := b["messages"].([]map[string]string)
	if !ok || len(msgs) != 1 || msgs[0]["role"] != "user" {
		t.Fatalf("bad messages: %#v", b["messages"])
	}
}

func TestBuildWebSearchBodyDeclaredGoogle(t *testing.T) {
	b := buildWebSearchBody(WebSearchConfig{Model: "gemini", Native: domain.NativeWebSearchGoogle, ReasoningEffort: "low"}, "成都天气")
	if _, ok := b["enable_search"]; ok {
		t.Fatal("google_search declared, enable_search still sent")
	}
	if _, ok := b["web_search_options"]; ok {
		t.Fatal("google_search declared, web_search_options still sent")
	}
	tools, ok := b["tools"].([]map[string]any)
	if !ok || len(tools) != 1 || tools[0]["google_search"] == nil {
		t.Fatalf("tools: %#v", b["tools"])
	}
	if b["reasoning_effort"] != "low" {
		t.Fatalf("reasoning_effort: %#v", b["reasoning_effort"])
	}
}

func TestWebSearchNotConfigured(t *testing.T) {
	if _, err := WebSearch(context.Background(), WebSearchConfig{}, "q"); err == nil {
		t.Fatal("expected error when not configured")
	}
}
