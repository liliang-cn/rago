package trace

import (
	"context"
	"testing"
)

type wantingSink struct {
	want bool
	rec  Record
}

func (s *wantingSink) ModelStart(context.Context, string, string) {}
func (s *wantingSink) ModelEnd(_ context.Context, _ string, r Record) {
	s.rec = r
}
func (s *wantingSink) Step(context.Context, Record) {}
func (s *wantingSink) WantBodies() bool             { return s.want }

// A BodySink that wants bodies gets them with AGENTGO_TRACE_BODIES unset;
// a plain sink, or one that declines, does not.
func TestBodySinkReceivesRequestWithoutEnv(t *testing.T) {
	bodies.Store(false)
	for _, want := range []bool{true, false} {
		sink := &wantingSink{want: want}
		ctx := WithSink(context.Background(), sink)
		call := Model(ctx, "constraints")
		call.Request("the prompt")
		call.Reply("the reply")
		call.End(nil, nil)
		got := sink.rec.Request != nil
		if got != want {
			t.Fatalf("WantBodies=%v: sink saw request=%v", want, got)
		}
		if want && (sink.rec.Request != "the prompt" || sink.rec.Reply != "the reply") {
			t.Fatalf("sink got %v / %v", sink.rec.Request, sink.rec.Reply)
		}
	}
}
