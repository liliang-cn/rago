package otelobserver

import (
	"context"
	"errors"
	"strings"
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/liliang-cn/agent-go/v3/pkg/agent"
	"github.com/liliang-cn/agent-go/v3/pkg/domain"
)

// Every llm, tool and agent span carries the OpenTelemetry GenAI conventions
// beside the OpenInference ones, so a backend built on either spec reads it.
func TestSpansCarryGenAISemconv(t *testing.T) {
	sr, tp := newRecorder()
	obs := New(tp)
	ctx := context.Background()

	req := &agent.ModelRequest{
		Messages: []domain.Message{
			{Role: "user", Content: "ping"},
			{Role: "assistant", ToolCalls: []domain.ToolCall{{ID: "c1", Function: domain.FunctionCall{Name: "ping", Arguments: map[string]any{"msg": "hi"}}}}},
			{Role: "tool", ToolCallID: "c1", Content: `{"ok":true}`},
		},
		Tools: []domain.ToolDefinition{{Type: "function", Function: domain.ToolFunction{Name: "ping"}}},
	}
	mi := agent.ModelInfo{TaskID: "t", SessionID: "sess", AgentName: "bot", Model: "qwen-plus", Round: 1, SpanID: "m1", Purpose: "turn", Request: req}
	obs.OnModelStart(ctx, mi)
	obs.OnModelEnd(ctx, mi, &agent.ModelResult{Content: "done", PromptTokens: 120, CompletionTokens: 7, CachedTokens: 100, Model: "qwen-plus-2025", Provider: "DashScope",
		Response: &agent.ModelResponse{Reasoning: "check again", FinishReason: "tool_calls",
			ToolCalls: []domain.ToolCall{{ID: "c2", Function: domain.FunctionCall{Name: "ping", Arguments: map[string]any{"msg": "again"}}}}}}, nil)

	failed := agent.ModelInfo{TaskID: "t", SessionID: "sess", AgentName: "bot", Model: "qwen-plus", SpanID: "m2", Purpose: "constraints"}
	obs.OnModelStart(ctx, failed)
	obs.OnModelEnd(ctx, failed, nil, errors.New("upstream 503 unavailable"))

	ti := agent.ToolInfo{TaskID: "t", SessionID: "sess", AgentName: "bot", Tool: "ping", CallID: "c1", Args: map[string]any{"msg": "hi"}}
	obs.OnToolStart(ctx, ti)
	obs.OnToolEnd(ctx, ti, map[string]any{"ok": true}, nil)

	si := agent.SubAgentInfo{ParentTaskID: "t", SessionID: "sess", Name: "researcher", SubAgentID: "s1", Goal: "look"}
	obs.OnSubAgentStart(ctx, si)
	obs.OnSubAgentEnd(ctx, si, "found", nil)

	got := map[string]map[string]string{}
	for _, s := range sr.Ended() {
		m := map[string]string{}
		for _, kv := range s.Attributes() {
			m[string(kv.Key)] = kv.Value.Emit()
		}
		switch {
		case m[attrSpanKind] == spanKindLLM && m["agentgo.llm.purpose"] == "turn":
			got["llm"] = m
		case m[attrSpanKind] == spanKindLLM:
			got["llm-failed"] = m
		case m[attrSpanKind] == spanKindTool:
			got["tool"] = m
		case m[attrSpanKind] == spanKindAgent:
			got["agent"] = m
		}
	}

	llm := got["llm"]
	for k, want := range map[string]string{
		genAIOperationName:  "chat",
		genAIRequestModel:   "qwen-plus",
		genAIResponseModel:  "qwen-plus-2025",
		genAIProviderName:   "dashscope",
		genAIUsageInput:     "120",
		genAIUsageOutput:    "7",
		genAIUsageCacheRead: "100",
		genAIConversationID: "sess",
		genAIAgentName:      "bot",
	} {
		if llm[k] != want {
			t.Errorf("llm %s = %q, want %q", k, llm[k], want)
		}
	}
	// OpenInference is still there: both specs on one span.
	if llm[attrLLMModelName] != "qwen-plus" || llm[attrOutputValue] != "done" {
		t.Errorf("OpenInference attributes lost: %v", llm)
	}
	in := llm[genAIInputMessages]
	for _, want := range []string{`"role":"user"`, `"type":"tool_call"`, `"name":"ping"`, `"type":"tool_call_response"`, `"id":"c1"`} {
		if !strings.Contains(in, want) {
			t.Errorf("gen_ai.input.messages missing %s: %s", want, in)
		}
	}
	out := llm[genAIOutputMessages]
	for _, want := range []string{`"type":"reasoning","content":"check again"`, `"content":"done"`, `"type":"tool_call","id":"c2","name":"ping"`, `"finish_reason":"tool_calls"`} {
		if !strings.Contains(out, want) {
			t.Errorf("gen_ai.output.messages missing %s: %s", want, out)
		}
	}
	if !strings.Contains(llm[genAIToolDefinitions], `"ping"`) {
		t.Errorf("gen_ai.tool.definitions: %q", llm[genAIToolDefinitions])
	}

	if f := got["llm-failed"]; f[genAIErrorType] != "server_error" || f[genAIUsageInput] != "" {
		t.Errorf("failed call: error.type=%q input=%q", f[genAIErrorType], f[genAIUsageInput])
	}

	tool := got["tool"]
	for k, want := range map[string]string{
		genAIOperationName:  "execute_tool",
		genAIToolName:       "ping",
		genAIToolType:       "function",
		genAIToolCallID:     "c1",
		genAIToolCallArgs:   `{"msg":"hi"}`,
		genAIToolCallResult: `{"ok":true}`,
	} {
		if tool[k] != want {
			t.Errorf("tool %s = %q, want %q", k, tool[k], want)
		}
	}

	ag := got["agent"]
	if ag[genAIOperationName] != "invoke_agent" || ag[genAIAgentName] != "researcher" {
		t.Errorf("agent span: %v", ag)
	}
}

// The two semconv metrics are recorded beside the agentgo.* ones, with
// gen_ai.token.type splitting input from output.
func TestGenAISemconvMetrics(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	_, tp := newRecorder()
	obs := New(tp, WithMeterProvider(mp))
	ctx := context.Background()

	mi := agent.ModelInfo{TaskID: "t", SessionID: "s", AgentName: "bot", Model: "m", SpanID: "x", Purpose: "turn"}
	obs.OnModelStart(ctx, mi)
	obs.OnModelEnd(ctx, mi, &agent.ModelResult{PromptTokens: 30, CompletionTokens: 5, DurationMs: 1200}, nil)

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			found[m.Name] = true
			if m.Name == genAIMetricTokenUsage {
				h, ok := m.Data.(metricdata.Histogram[int64])
				if !ok {
					t.Fatalf("%s is not an int64 histogram: %T", m.Name, m.Data)
				}
				sums := map[string]int64{}
				for _, dp := range h.DataPoints {
					tt, _ := dp.Attributes.Value(genAITokenType)
					sums[tt.AsString()] = dp.Sum
				}
				if sums["input"] != 30 || sums["output"] != 5 {
					t.Fatalf("token usage by type: %v", sums)
				}
			}
		}
	}
	for _, name := range []string{genAIMetricTokenUsage, genAIMetricOperationDuration, "agentgo.tokens.prompt"} {
		if !found[name] {
			t.Errorf("metric %s not recorded (have %v)", name, found)
		}
	}
}
