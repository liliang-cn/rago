package agent

import (
	"context"
	"strings"
	"sync"
	"testing"
)

// payloadObserver keeps every ModelInfo it is handed, start and end, so a
// test can ask what the model was sent.
type payloadObserver struct {
	BaseObserver
	mu     sync.Mutex
	starts []ModelInfo
	ends   []ModelInfo
}

func (p *payloadObserver) OnModelStart(_ context.Context, info ModelInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.starts = append(p.starts, info)
}

func (p *payloadObserver) OnModelEnd(_ context.Context, info ModelInfo, _ *ModelResult, _ error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ends = append(p.ends, info)
}

func runPayloadService(t *testing.T, payloads bool) *payloadObserver {
	t.Helper()
	obs := &payloadObserver{}
	b := New("payload-agent").
		WithConfig(testAgentConfig(t.TempDir())).
		WithLLM(&observerToolLLM{}).
		WithTool(NewTool("ping", "Ping back", func(_ context.Context, p *pingParams) (any, error) {
			return map[string]any{"ok": true, "echo": p.Msg}, nil
		})).
		WithObserver(obs)
	if payloads {
		b = b.WithObserverPayloads()
	}
	svc, err := b.Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	defer svc.Close()
	events, err := svc.RunStream(context.Background(), "please ping")
	if err != nil {
		t.Fatalf("RunStream: %v", err)
	}
	if final := Concat(events); final != "all done" {
		t.Fatalf("final text %q", final)
	}
	return obs
}

// Default: counts and output only. A host that did not ask for the request
// must not start receiving every prompt because the type grew a field.
func TestObserverPayloadsOffByDefault(t *testing.T) {
	obs := runPayloadService(t, false)
	obs.mu.Lock()
	defer obs.mu.Unlock()
	if len(obs.starts) == 0 {
		t.Fatal("no model starts recorded")
	}
	for _, info := range append(obs.starts, obs.ends...) {
		if info.Request != nil {
			t.Fatalf("Request set without WithObserverPayloads on %s round %d", info.Purpose, info.Round)
		}
		if info.Purpose == "turn" && info.Messages == 0 {
			t.Fatal("the message count must still be reported when payloads are off")
		}
	}
}

// With the option: the turn's messages and tools are on OnModelStart, the
// tool exchange of round one is visible in round two's request, and a call
// made on the run's behalf carries its prompt by OnModelEnd.
func TestObserverPayloadsCarryTheRequest(t *testing.T) {
	obs := runPayloadService(t, true)
	obs.mu.Lock()
	defer obs.mu.Unlock()

	var turns []ModelInfo
	for _, info := range obs.starts {
		if info.Purpose == "turn" {
			turns = append(turns, info)
		}
	}
	if len(turns) < 2 {
		t.Fatalf("expected two answering turns, got %d", len(turns))
	}
	first := turns[0]
	if first.Request == nil {
		t.Fatal("turn has no Request")
	}
	if len(first.Request.Messages) != first.Messages {
		t.Fatalf("Request has %d messages, count says %d", len(first.Request.Messages), first.Messages)
	}
	last := first.Request.Messages[len(first.Request.Messages)-1]
	if last.Role != "user" || last.Content != "please ping" {
		t.Fatalf("last message of the first turn should be the goal, got %+v", last)
	}
	var hasPing bool
	for _, tool := range first.Request.Tools {
		if tool.Function.Name == "ping" {
			hasPing = true
		}
	}
	if !hasPing || len(first.Request.Tools) != first.Tools {
		t.Fatalf("Request.Tools (%d) should list the offered tools including ping; count says %d", len(first.Request.Tools), first.Tools)
	}

	// Round two's request must show round one's tool call and its result:
	// that is the whole point — reading what the model saw, not a count.
	second := turns[1].Request
	if second == nil {
		t.Fatal("second turn has no Request")
	}
	var sawCall, sawResult bool
	for _, m := range second.Messages {
		for _, tc := range m.ToolCalls {
			if tc.Function.Name == "ping" {
				sawCall = true
			}
		}
		// The runtime normalises call ids (fc_ prefix on this provider
		// shape), so match the tail, and the tool message must carry the
		// tool's actual output.
		if m.Role == "tool" && strings.HasSuffix(m.ToolCallID, "call_ping_1") && strings.Contains(m.Content, `"echo":"hi"`) {
			sawResult = true
		}
	}
	if !sawCall || !sawResult {
		t.Fatalf("second turn's request should carry the ping call and its result (call=%v result=%v)", sawCall, sawResult)
	}

	// Calls the runtime makes on its own behalf arrive through the trace
	// sink; their request is known at the end of the call.
	for _, info := range obs.ends {
		if info.Purpose == "turn" || info.Purpose == "final-synthesis" {
			continue
		}
		if info.Request == nil || info.Request.Prompt == "" {
			t.Fatalf("%s call reached OnModelEnd without its prompt", info.Purpose)
		}
	}
}
