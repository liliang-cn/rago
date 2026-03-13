package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/liliang-cn/agent-go/pkg/config"
	"github.com/liliang-cn/agent-go/pkg/prompt"
)

func TestBuildSystemPromptOmitsOperationalNotesForConcierge(t *testing.T) {
	concierge := NewAgentWithConfig(BuiltInConciergeAgentName, "concierge instructions", nil)
	svc := &Service{
		agent:         concierge,
		promptManager: prompt.NewManager(),
		cfg: &config.Config{
			Tooling: config.ToolingConfig{
				WebSearch: config.WebSearchConfig{Mode: "auto"},
			},
		},
	}

	got := svc.buildSystemPrompt(context.Background(), concierge)
	if strings.Contains(got, "\nRules:\n") {
		t.Fatalf("expected concierge prompt to omit rules, got %q", got)
	}
	if strings.Contains(got, "Web search capability:") {
		t.Fatalf("expected concierge prompt to omit web search note, got %q", got)
	}
	if !strings.Contains(got, "concierge instructions") {
		t.Fatalf("expected concierge instructions in prompt, got %q", got)
	}
}

func TestBuildSystemPromptKeepsOperationalNotesForAssistant(t *testing.T) {
	assistant := NewAgentWithConfig("Assistant", "assistant instructions", nil)
	svc := &Service{
		agent:         assistant,
		promptManager: prompt.NewManager(),
		cfg: &config.Config{
			Tooling: config.ToolingConfig{
				WebSearch: config.WebSearchConfig{Mode: "auto"},
			},
		},
	}

	got := svc.buildSystemPrompt(context.Background(), assistant)
	if !strings.Contains(got, "\nRules:\n") {
		t.Fatalf("expected assistant prompt to keep rules, got %q", got)
	}
	if !strings.Contains(got, "Web search capability:") {
		t.Fatalf("expected assistant prompt to keep web search note, got %q", got)
	}
}
