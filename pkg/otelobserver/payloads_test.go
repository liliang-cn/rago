package otelobserver

import (
	"context"
	"strings"
	"testing"

	"github.com/liliang-cn/agent-go/v3/pkg/agent"
	"github.com/liliang-cn/agent-go/v3/pkg/domain"
)

// With a Request on ModelInfo the LLM span carries the OpenInference message
// and tool attributes Phoenix renders as the prompt; without one it carries
// exactly what it did before.
func TestLLMSpanCarriesRequestWhenGiven(t *testing.T) {
	sr, tp := newRecorder()
	obs := New(tp)
	ctx := context.Background()

	req := &agent.ModelRequest{
		Messages: []domain.Message{
			{Role: "system", Content: "You are terse."},
			{Role: "user", Content: "ping please"},
			{Role: "assistant", ToolCalls: []domain.ToolCall{{ID: "c1", Function: domain.FunctionCall{Name: "ping", Arguments: map[string]any{"msg": "hi"}}}}},
			{Role: "tool", ToolCallID: "c1", Content: `{"ok":true}`},
		},
		Tools: []domain.ToolDefinition{{Type: "function", Function: domain.ToolFunction{Name: "ping", Description: "Ping back"}}},
	}
	mi := agent.ModelInfo{TaskID: "t1", SessionID: "s1", AgentName: "a", Round: 2, SpanID: "span-req", Purpose: "turn", Request: req}
	obs.OnModelStart(ctx, mi)
	obs.OnModelEnd(ctx, mi, &agent.ModelResult{Content: "all done"}, nil)

	bare := agent.ModelInfo{TaskID: "t1", SessionID: "s1", AgentName: "a", Round: 3, SpanID: "span-bare", Purpose: "turn"}
	obs.OnModelStart(ctx, bare)
	obs.OnModelEnd(ctx, bare, &agent.ModelResult{Content: "x"}, nil)

	structured := agent.ModelInfo{TaskID: "t1", SessionID: "s1", AgentName: "a", SpanID: "span-prompt", Purpose: "constraints", Request: &agent.ModelRequest{Prompt: "classify this"}}
	obs.OnModelStart(ctx, structured)
	obs.OnModelEnd(ctx, structured, &agent.ModelResult{Content: "{}"}, nil)

	byName := map[string]map[string]string{}
	for _, s := range sr.Ended() {
		m := map[string]string{}
		for _, kv := range s.Attributes() {
			m[string(kv.Key)] = kv.Value.Emit()
		}
		byName[s.Name()] = m
	}

	withReq := byName["llm.a"]
	if withReq == nil {
		t.Fatalf("no llm.a span: %v", byName)
	}
	// Both spans are named llm.a; the map kept the last one ended, which is
	// the bare one. Find the one with the request by its attributes.
	var reqSpan, bareSpan map[string]string
	for _, s := range sr.Ended() {
		m := map[string]string{}
		for _, kv := range s.Attributes() {
			m[string(kv.Key)] = kv.Value.Emit()
		}
		if m[attrRound] == "2" {
			reqSpan = m
		}
		if m[attrRound] == "3" {
			bareSpan = m
		}
	}
	checks := map[string]string{
		"llm.input_messages.0.message.role":                                 "system",
		"llm.input_messages.1.message.content":                              "ping please",
		"llm.input_messages.2.message.tool_calls.0.tool_call.function.name": "ping",
		"llm.input_messages.3.message.tool_call_id":                         "c1",
		"input.mime_type": "application/json",
	}
	for k, want := range checks {
		if got := reqSpan[k]; got != want {
			t.Fatalf("%s = %q, want %q", k, got, want)
		}
	}
	if !strings.Contains(reqSpan["llm.tools.0.tool.json_schema"], `"ping"`) {
		t.Fatalf("tool schema missing: %q", reqSpan["llm.tools.0.tool.json_schema"])
	}
	if !strings.Contains(reqSpan[attrInputValue], "ping please") {
		t.Fatalf("input.value should hold the request JSON: %q", reqSpan[attrInputValue])
	}
	if reqSpan[attrOutputValue] != "all done" {
		t.Fatalf("output still expected: %q", reqSpan[attrOutputValue])
	}

	for k := range bareSpan {
		if strings.HasPrefix(k, "llm.input_messages") || strings.HasPrefix(k, "llm.tools") || k == attrInputValue {
			t.Fatalf("bare span grew attribute %s", k)
		}
	}

	prompt := byName["llm.constraints"]
	if prompt[attrInputValue] != "classify this" || prompt["llm.input_messages.0.message.content"] != "classify this" {
		t.Fatalf("structured call's prompt not on span: %v", prompt)
	}
}
