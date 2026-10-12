package agent

import (
	"context"
	"testing"
	"time"

	"github.com/liliang-cn/agent-go/v3/pkg/domain"
)

// stuckMemory answers its lookup after a long wait and ignores cancellation,
// like a remote store waiting on an embedder that was pushed out of memory.
type stuckMemory struct {
	promptTestMemoryService
	wait time.Duration
}

func (m stuckMemory) RetrieveAndInjectWithContextAndLogic(context.Context, string, domain.MemoryQueryContext) (string, []*domain.MemoryWithScore, string, error) {
	time.Sleep(m.wait)
	return "remembered", nil, "", nil
}

// A slow memory lookup does not hold the turn: past its deadline the turn goes
// ahead without it, even when the store ignores the cancellation.
func TestASlowMemoryLookupDoesNotHoldTheTurn(t *testing.T) {
	cfg := testAgentConfig(t.TempDir())
	cfg.Memory.RecallTimeoutMs = 200
	svc, err := New("impatient").
		WithConfig(cfg).
		WithLLM(chunkedAnswerLLM{}).
		WithMemoryService(stuckMemory{wait: 5 * time.Second}).
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	defer svc.Close()

	start := time.Now()
	events, err := svc.RunStream(context.Background(), "Report the disks.")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	final := ""
	for ev := range events {
		if ev.Type == EventTypeComplete {
			final = ev.Content
		}
	}
	if took := time.Since(start); took > 3*time.Second {
		t.Fatalf("the turn waited %s for a memory lookup with a 200ms deadline", took)
	}
	if final != streamedAnswer {
		t.Fatalf("the turn did not answer without memory: %q", final)
	}
}

func TestTheRecallDeadlineDefaultsAndCanBeTurnedOff(t *testing.T) {
	cfg := testAgentConfig(t.TempDir())
	s := &Service{cfg: cfg}
	if got := s.memoryRecallTimeout(); got != DefaultMemoryRecallTimeout {
		t.Fatalf("default = %s", got)
	}
	cfg.Memory.RecallTimeoutMs = -1
	if got := s.memoryRecallTimeout(); got != 0 {
		t.Fatalf("negative should wait as long as it takes, got %s", got)
	}
}
