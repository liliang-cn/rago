package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
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
