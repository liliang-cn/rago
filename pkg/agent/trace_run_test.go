package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/liliang-cn/agent-go/v3/pkg/trace"
)

// With tracing on, a run says where its time went: a line per model call on
// the trace output, and the whole timeline as one trace event at the end, which
// is what an app reads to show a person why an answer was slow.
func TestARunTracesItsModelCallsAndEndsWithATimeline(t *testing.T) {
	var lines bytes.Buffer
	trace.SetEnabled(true)
	trace.SetOutput(&lines)
	t.Cleanup(func() { trace.SetEnabled(false); trace.SetLines(false) })

	svc, err := New("traced").
		WithConfig(testAgentConfig(t.TempDir())).
		WithLLM(chunkedAnswerLLM{}).
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	defer svc.Close()

	events, err := svc.RunStream(context.Background(), "Report the disks.")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	var timeline string
	for ev := range events {
		if ev.Type == EventTypeTrace {
			timeline = ev.Content
		}
	}
	if timeline == "" {
		t.Fatal("no trace event at the end of the run")
	}
	var got struct {
		Summary trace.Summary  `json:"summary"`
		Steps   []trace.Record `json:"steps"`
	}
	if err := json.Unmarshal([]byte(timeline), &got); err != nil {
		t.Fatalf("trace event is not the timeline: %v\n%s", err, timeline)
	}
	var turn *trace.Record
	for i := range got.Steps {
		if got.Steps[i].Kind == "model" && got.Steps[i].Name == "turn" {
			turn = &got.Steps[i]
		}
	}
	if turn == nil || got.Summary.ModelCalls == 0 {
		t.Fatalf("the answering model call is missing from the timeline: %+v", got)
	}
	if got.Summary.FirstAnswerTokenMs <= 0 && turn.TTFTMs <= 0 {
		t.Fatalf("no first-token time for a streamed answer: %+v", got.Summary)
	}
	if out := lines.String(); !strings.Contains(out, "model purpose=turn") || !strings.Contains(out, "run end total=") {
		t.Fatalf("trace lines:\n%s", out)
	}
}

type purposeObserver struct {
	BaseObserver
	mu     sync.Mutex
	starts []string
	ends   map[string]int
}

func (o *purposeObserver) OnModelStart(_ context.Context, info ModelInfo) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.starts = append(o.starts, info.Purpose)
}

func (o *purposeObserver) OnModelEnd(_ context.Context, info ModelInfo, _ *ModelResult, _ error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.ends == nil {
		o.ends = map[string]int{}
	}
	o.ends[info.Purpose]++
}

// Observers hear the calls a run makes on its own behalf, named for what they
// were for, with tracing off as much as on — that is where a slow start hid
// from every exporter. Each one that starts also ends, and the answering
// turn is reported once.
func TestObserversHearEveryModelCallByPurpose(t *testing.T) {
	trace.SetEnabled(false)
	obs := &purposeObserver{}
	svc, err := New("observed").
		WithConfig(testAgentConfig(t.TempDir())).
		WithLLM(chunkedAnswerLLM{}).
		WithObserver(obs).
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	defer svc.Close()
	events, err := svc.RunStream(context.Background(), "Report the disks.")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	for range events {
	}

	obs.mu.Lock()
	defer obs.mu.Unlock()
	count := map[string]int{}
	for _, p := range obs.starts {
		if p == "" {
			t.Fatalf("a model call reached the observer with no purpose: %v", obs.starts)
		}
		count[p]++
	}
	if count["turn"] != 1 {
		t.Fatalf("the answering turn was reported %d times: %v", count["turn"], obs.starts)
	}
	if count["constraints"] == 0 {
		t.Fatalf("constraint extraction never reached the observer: %v", obs.starts)
	}
	for p, n := range count {
		if obs.ends[p] != n {
			t.Fatalf("%s: %d starts, %d ends", p, n, obs.ends[p])
		}
	}
}
