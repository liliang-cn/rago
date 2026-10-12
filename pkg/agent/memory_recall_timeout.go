package agent

import (
	"context"
	"errors"
	"time"

	"github.com/liliang-cn/agent-go/v3/pkg/domain"
)

// The memory lookup a turn makes before it starts has a deadline.
//
// The turn waits for it: recalled memory goes into the prompt. With memory on
// a remote store that waits in turn on an embedding model somewhere else, the
// lookup took 0.7s on one turn and 24s on the next — the embedder had been
// pushed out of memory on the machine serving it — and the person waited the
// whole 24 for an answer that did not need it. Past the deadline the turn goes
// ahead without recalled memory; the lookup is left to finish or fail on its
// own, and a backend that ignores cancellation cannot hold the turn either.

// DefaultMemoryRecallTimeout is the deadline when none is configured.
const DefaultMemoryRecallTimeout = 2 * time.Second

var errRecallTimeout = errors.New("memory recall timed out")

type recallResult struct {
	context  string
	memories []*domain.MemoryWithScore
	logic    string
}

// memoryRecallTimeout is the configured deadline; zero means none.
func (s *Service) memoryRecallTimeout() time.Duration {
	if s == nil || s.cfg == nil || s.cfg.Memory.RecallTimeoutMs == 0 {
		return DefaultMemoryRecallTimeout
	}
	if s.cfg.Memory.RecallTimeoutMs < 0 {
		return 0
	}
	return time.Duration(s.cfg.Memory.RecallTimeoutMs) * time.Millisecond
}

// recallWithin runs the turn's memory lookup under the deadline.
func (s *Service) recallWithin(ctx context.Context, goal string, qc domain.MemoryQueryContext) (recallResult, error) {
	mem := s.memory()
	limit := s.memoryRecallTimeout()
	if limit <= 0 {
		c, m, l, err := mem.RetrieveAndInjectWithContextAndLogic(ctx, goal, qc)
		return recallResult{c, m, l}, err
	}
	rctx, cancel := context.WithTimeout(ctx, limit)
	type out struct {
		res recallResult
		err error
	}
	done := make(chan out, 1)
	go func() {
		defer cancel()
		c, m, l, err := mem.RetrieveAndInjectWithContextAndLogic(rctx, goal, qc)
		done <- out{recallResult{c, m, l}, err}
	}()
	select {
	case o := <-done:
		if o.err != nil && errors.Is(rctx.Err(), context.DeadlineExceeded) && ctx.Err() == nil {
			return recallResult{}, errRecallTimeout
		}
		return o.res, o.err
	case <-rctx.Done():
		if ctx.Err() != nil {
			return recallResult{}, ctx.Err()
		}
		return recallResult{}, errRecallTimeout
	}
}
