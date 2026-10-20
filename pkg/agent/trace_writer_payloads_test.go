package agent

import (
	"context"
	"strings"
	"testing"
)

func TestTraceWriterWritesRequestOnlyWithPayloads(t *testing.T) {
	run := func(payloads bool, maxPayload int) []map[string]any {
		buf := &syncBuffer{}
		tr := NewTraceWriter(buf)
		tr.MaxPayloadLen = maxPayload
		b := New("trace-payload-agent").
			WithConfig(testAgentConfig(t.TempDir())).
			WithLLM(&observerToolLLM{}).
			WithTool(NewTool("ping", "Ping back", func(_ context.Context, p *pingParams) (any, error) {
				return map[string]any{"ok": true}, nil
			})).
			WithObserver(tr)
		if payloads {
			b = b.WithObserverPayloads()
		}
		svc, err := b.Build()
		if err != nil {
			t.Fatal(err)
		}
		defer svc.Close()
		events, err := svc.RunStream(context.Background(), "please ping")
		if err != nil {
			t.Fatal(err)
		}
		Concat(events)
		return parseTrace(t, buf.String())
	}

	modelEnds := func(lines []map[string]any) []map[string]any {
		var out []map[string]any
		for _, l := range lines {
			if l["event"] == "model_end" && l["purpose"] == "turn" {
				out = append(out, l)
			}
		}
		return out
	}

	off := modelEnds(run(false, 0))
	if len(off) == 0 {
		t.Fatal("no model_end lines")
	}
	for _, l := range off {
		if _, ok := l["request"]; ok {
			t.Fatal("request written without WithObserverPayloads")
		}
		if _, ok := l["response"]; ok {
			t.Fatal("response written without WithObserverPayloads")
		}
	}

	on := modelEnds(run(true, 0))
	if len(on) < 2 {
		t.Fatalf("expected two turns, got %d", len(on))
	}
	req, ok := on[0]["request"].(map[string]any)
	if !ok {
		t.Fatalf("first model_end has no request object: %v", on[0])
	}
	msgs, _ := req["messages"].([]any)
	if len(msgs) == 0 {
		t.Fatal("request has no messages")
	}
	lastMsg := msgs[len(msgs)-1].(map[string]any)
	if lastMsg["content"] != "please ping" {
		t.Fatalf("last message should be the goal: %v", lastMsg)
	}
	tools, _ := req["tools"].([]any)
	var hasPing bool
	for _, name := range tools {
		if name == "ping" {
			hasPing = true
		}
	}
	if !hasPing {
		t.Fatalf("tools should be names including ping: %v", tools)
	}
	// Round two carries round one's call.
	req2 := on[1]["request"].(map[string]any)
	raw := strings.Builder{}
	for _, m := range req2["messages"].([]any) {
		mm := m.(map[string]any)
		if calls, ok := mm["tool_calls"].([]any); ok {
			for _, c := range calls {
				raw.WriteString(c.(map[string]any)["name"].(string))
			}
		}
	}
	if !strings.Contains(raw.String(), "ping") {
		t.Fatal("second turn's request should show the ping tool call")
	}

	// Each line has the model's side too: the call it asked for in round
	// one, with its arguments, and the answer in round two.
	resp, ok := on[0]["response"].(map[string]any)
	if !ok {
		t.Fatalf("first model_end has no response: %v", on[0])
	}
	calls, _ := resp["tool_calls"].([]any)
	if len(calls) != 1 || calls[0].(map[string]any)["name"] != "ping" {
		t.Fatalf("first response should ask for ping: %v", resp)
	}
	if final, _ := on[len(on)-1]["response"].(map[string]any); final["content"] != "all done" {
		t.Fatalf("last response should be the answer: %v", on[len(on)-1])
	}

	// MaxPayloadLen clips message content, nothing else.
	clipped := modelEnds(run(true, 4))
	creq := clipped[0]["request"].(map[string]any)
	cmsgs := creq["messages"].([]any)
	last := cmsgs[len(cmsgs)-1].(map[string]any)["content"].(string)
	if last != "plea…" {
		t.Fatalf("clipped content %q", last)
	}
	if got := clipped[len(clipped)-1]["response"].(map[string]any)["content"]; got != "all …" {
		t.Fatalf("clipped answer %q", got)
	}
}
