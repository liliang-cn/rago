package pool

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/liliang-cn/agent-go/v3/pkg/domain"
)

// Failover.
//
// StrategyFailover used to mean "the first healthy client", but the first of
// a map is whichever one Go's iteration hands out, and a call that failed on
// it simply failed: nothing moved on to the next provider. A server that
// keeps a second model configured for when its primary is down got neither
// half of that.
//
// Providers are now tried in the order they were configured. One whose call
// fails goes to the back of the order for FailoverCooldown, so the calls
// after it stop paying for the failure, and is tried first again once the
// cooldown passes. Every error moves on except the caller's own context
// ending: an upstream that rejects the key today is as gone, for the caller,
// as one that times out. Streams are not failed over; half an answer cannot
// be taken back.

const defaultFailoverCooldown = 30 * time.Second

// usesFailover reports whether a call with opts goes through failover. A call
// that names a model or provider is routed instead.
func (p *Pool) usesFailover(opts *domain.GenerationOptions) bool {
	return p.strategy == StrategyFailover && (opts == nil || (opts.Model == "" && opts.Provider == ""))
}

// failoverCandidates is the healthy clients in configured order, those still
// cooling down after a failure moved to the end.
func (p *Pool) failoverCandidates() []*clientWrapper {
	p.mu.RLock()
	defer p.mu.RUnlock()

	now := time.Now().UnixNano()
	healthy := p.healthyClients()
	ready := make([]*clientWrapper, 0, len(healthy))
	var cooling []*clientWrapper
	for _, w := range healthy {
		if w.failedUntil.Load() > now {
			cooling = append(cooling, w)
		} else {
			ready = append(ready, w)
		}
	}
	return append(ready, cooling...)
}

// failover runs call against each candidate in turn until one succeeds.
func (p *Pool) failover(ctx context.Context, call func(context.Context, *Client) error) error {
	candidates := p.failoverCandidates()
	if len(candidates) == 0 {
		// Every provider is at its concurrency limit (or there are none):
		// wait for a slot the usual way.
		c, err := p.acquire(ctx, p.Get)
		if err != nil {
			return err
		}
		defer p.Release(c)
		return call(ctx, c)
	}

	cooldown := p.config.FailoverCooldown
	if cooldown <= 0 {
		cooldown = defaultFailoverCooldown
	}
	var errs []error
	for _, w := range candidates {
		atomic.AddInt32(&w.activeRequests, 1)
		err := p.attempt(ctx, w.client, call)
		p.Release(w.client)
		if err == nil {
			w.failedUntil.Store(0)
			return nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", w.provider.Name, err))
		if ctx.Err() != nil {
			break
		}
		w.failedUntil.Store(time.Now().Add(cooldown).UnixNano())
	}
	return errors.Join(errs...)
}

// attempt is one provider's turn, bounded by AttemptTimeout when set.
func (p *Pool) attempt(ctx context.Context, c *Client, call func(context.Context, *Client) error) error {
	if p.config.AttemptTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.config.AttemptTimeout)
		defer cancel()
	}
	return call(ctx, c)
}
