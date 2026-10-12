package agent

import (
	"context"
	"errors"

	"github.com/liliang-cn/agent-go/v3/pkg/trace"
)

// runtimeSink hands the observers the model calls a run makes on its own
// behalf — constraint extraction, memory picks, compaction, summaries — and
// its memory steps. The answering turn and the final synthesis are reported
// where they are made, with their message and tool counts, so they are not
// reported twice.
type runtimeSink struct{ r *Runtime }

func (s runtimeSink) info(id, purpose string) ModelInfo {
	r := s.r
	return ModelInfo{
		TaskID:    currentTaskID(r.session),
		RunID:     r.runID(),
		SessionID: r.sessionID(),
		AgentName: r.currentAgentName(),
		Round:     r.currentRound,
		SpanID:    id,
		Model:     r.svc.Info().Model,
		Purpose:   purpose,
	}
}

func reportedElsewhere(purpose string) bool {
	return purpose == "turn" || purpose == "final-synthesis"
}

func (s runtimeSink) ModelStart(ctx context.Context, id, purpose string) {
	if reportedElsewhere(purpose) {
		return
	}
	info := s.info(id, purpose)
	s.r.svc.emitObserver(func(o Observer) { o.OnModelStart(ctx, info) })
}

func (s runtimeSink) ModelEnd(ctx context.Context, id string, rec trace.Record) {
	if reportedElsewhere(rec.Name) {
		return
	}
	info := s.info(id, rec.Name)
	res := &ModelResult{
		DurationMs: rec.DurMs, TTFTMs: rec.TTFTMs,
		PromptTokens: rec.In, CompletionTokens: rec.Out, TokensUsed: rec.In + rec.Out, CachedTokens: rec.Cached,
	}
	var err error
	if rec.Err != "" {
		err = errors.New(rec.Err)
	}
	s.r.svc.emitObserver(func(o Observer) { o.OnModelEnd(ctx, info, res, err) })
}

func (s runtimeSink) Step(ctx context.Context, rec trace.Record) {
	if rec.Kind == "tool" {
		return // OnToolStart/OnToolEnd already carry tool calls
	}
	r := s.r
	info := StepInfo{TaskID: currentTaskID(r.session), RunID: r.runID(), SessionID: r.sessionID(),
		Kind: rec.Kind, Name: rec.Name, DurationMs: rec.DurMs, Err: rec.Err}
	r.svc.emitObserver(func(o Observer) {
		if so, ok := o.(StepObserver); ok {
			so.OnStep(ctx, info)
		}
	})
}
