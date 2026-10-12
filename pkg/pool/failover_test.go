package pool

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/liliang-cn/agent-go/v3/pkg/domain"
)

// upstream is a fake chat-completions server that answers with status, or
// with a reply naming itself, and records each request body.
type upstream struct {
	srv    *httptest.Server
	status atomic.Int32
	delay  time.Duration
	calls  atomic.Int32

	mu     sync.Mutex
	bodies []map[string]any
}

func newUpstream(t *testing.T, name string) *upstream {
	t.Helper()
	u := &upstream{}
	u.status.Store(http.StatusOK)
	u.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u.calls.Add(1)
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		u.mu.Lock()
		u.bodies = append(u.bodies, body)
		u.mu.Unlock()
		if u.delay > 0 {
			select {
			case <-time.After(u.delay):
			case <-r.Context().Done():
				return
			}
		}
		if s := int(u.status.Load()); s != http.StatusOK {
			http.Error(w, `{"error":"down"}`, s)
			return
		}
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"`+name+`"}}]}`)
	}))
	t.Cleanup(u.srv.Close)
	return u
}

func (u *upstream) lastBody() map[string]any {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.bodies[len(u.bodies)-1]
}

func failoverPool(t *testing.T, cfg PoolConfig, ups ...*upstream) *Pool {
	t.Helper()
	cfg.Enabled = true
	cfg.Strategy = StrategyFailover
	names := []string{"a", "b", "c", "d", "e", "f"}
	for i, u := range ups {
		cfg.Providers = append(cfg.Providers, Provider{Name: names[i], BaseURL: u.srv.URL, Key: "k", ModelName: "m-" + names[i]})
	}
	p, err := NewPool(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

var hello = []domain.Message{{Role: "user", Content: "hi"}}

func TestFailoverTriesProvidersInConfiguredOrder(t *testing.T) {
	ups := []*upstream{newUpstream(t, "a"), newUpstream(t, "b"), newUpstream(t, "c"), newUpstream(t, "d"), newUpstream(t, "e"), newUpstream(t, "f")}
	p := failoverPool(t, PoolConfig{}, ups...)

	// Map iteration would pick a different "first" across runs; the order
	// has to come out the same every time.
	for range 20 {
		res, err := p.GenerateWithTools(context.Background(), hello, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if res.Content != "a" || res.Provider != "a" {
			t.Fatalf("answered by %q (%s), want the first configured provider", res.Content, res.Provider)
		}
	}
	for i, u := range ups[1:] {
		if n := u.calls.Load(); n != 0 {
			t.Fatalf("provider %d was called %d times while the first one was up", i+1, n)
		}
	}
}

func TestFailoverMovesOnWhenThePrimaryFails(t *testing.T) {
	primary, backup := newUpstream(t, "primary"), newUpstream(t, "backup")
	primary.status.Store(http.StatusUnauthorized)
	p := failoverPool(t, PoolConfig{FailoverCooldown: time.Hour}, primary, backup)

	res, err := p.GenerateWithTools(context.Background(), hello, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Content != "backup" {
		t.Fatalf("content = %q, want the backup's answer", res.Content)
	}

	// While it cools down the failed primary goes last, so the next call
	// does not pay for it again.
	if _, err := p.GenerateWithTools(context.Background(), hello, nil, nil); err != nil {
		t.Fatal(err)
	}
	if n := primary.calls.Load(); n != 1 {
		t.Fatalf("primary called %d times, want 1 (skipped while cooling down)", n)
	}
}

func TestFailoverReturnsToThePrimaryAfterTheCooldown(t *testing.T) {
	primary, backup := newUpstream(t, "primary"), newUpstream(t, "backup")
	primary.status.Store(http.StatusBadGateway)
	p := failoverPool(t, PoolConfig{FailoverCooldown: 20 * time.Millisecond}, primary, backup)

	if _, err := p.GenerateWithTools(context.Background(), hello, nil, nil); err != nil {
		t.Fatal(err)
	}
	primary.status.Store(http.StatusOK)
	time.Sleep(40 * time.Millisecond)

	res, err := p.GenerateWithTools(context.Background(), hello, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Content != "primary" {
		t.Fatalf("content = %q, want the recovered primary", res.Content)
	}
}

func TestFailoverReportsEveryProviderWhenAllFail(t *testing.T) {
	a, b := newUpstream(t, "a"), newUpstream(t, "b")
	a.status.Store(http.StatusInternalServerError)
	b.status.Store(http.StatusServiceUnavailable)
	p := failoverPool(t, PoolConfig{}, a, b)

	_, err := p.GenerateWithTools(context.Background(), hello, nil, nil)
	if err == nil {
		t.Fatal("want an error when every provider fails")
	}
	for _, want := range []string{"a: ", "b: ", "status 500", "status 503"} {
		if !contains(err.Error(), want) {
			t.Fatalf("error %q does not mention %q", err, want)
		}
	}
}

func TestFailoverStopsWhenTheCallerGivesUp(t *testing.T) {
	slow, backup := newUpstream(t, "slow"), newUpstream(t, "backup")
	slow.delay = time.Second
	p := failoverPool(t, PoolConfig{}, slow, backup)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err := p.GenerateWithTools(ctx, hello, nil, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want the caller's deadline", err)
	}
	if n := backup.calls.Load(); n != 0 {
		t.Fatalf("backup called %d times after the caller's context ended", n)
	}
}

func TestFailoverAttemptTimeoutLeavesTimeForTheBackup(t *testing.T) {
	slow, backup := newUpstream(t, "slow"), newUpstream(t, "backup")
	slow.delay = time.Second
	p := failoverPool(t, PoolConfig{AttemptTimeout: 30 * time.Millisecond}, slow, backup)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	res, err := p.GenerateWithTools(ctx, hello, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Content != "backup" {
		t.Fatalf("content = %q, want the backup after the primary's turn timed out", res.Content)
	}
}

func TestFailoverRoutedCallsSkipFailover(t *testing.T) {
	a, b := newUpstream(t, "a"), newUpstream(t, "b")
	a.status.Store(http.StatusInternalServerError)
	p := failoverPool(t, PoolConfig{}, a, b)

	// A call that names its provider asked for that one, not for whichever
	// answers.
	_, err := p.GenerateWithTools(context.Background(), hello, nil, &domain.GenerationOptions{Provider: "a"})
	if err == nil {
		t.Fatal("routed call to a failing provider should fail, not fail over")
	}
	if n := b.calls.Load(); n != 0 {
		t.Fatalf("b called %d times for a call routed to a", n)
	}
}

func TestProviderThinkingGoesOnlyToItsOwnProvider(t *testing.T) {
	deepseek, gemma := newUpstream(t, "deepseek"), newUpstream(t, "gemma")
	deepseek.status.Store(http.StatusInternalServerError)
	p, err := NewPool(PoolConfig{
		Enabled:  true,
		Strategy: StrategyFailover,
		Providers: []Provider{
			{Name: "deepseek", BaseURL: deepseek.srv.URL, Key: "k", ModelName: "deepseek-v4-flash", Thinking: "disabled"},
			{Name: "gemma", BaseURL: gemma.srv.URL, Key: "k", ModelName: "gemma"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.GenerateWithTools(context.Background(), hello, nil, nil); err != nil {
		t.Fatal(err)
	}

	th, _ := deepseek.lastBody()["thinking"].(map[string]any)
	if th["type"] != "disabled" {
		t.Fatalf("deepseek body thinking = %v, want disabled", deepseek.lastBody()["thinking"])
	}
	if _, set := gemma.lastBody()["thinking"]; set {
		t.Fatalf("gemma got a thinking field: %v", gemma.lastBody())
	}
	if _, set := gemma.lastBody()["tools"]; set {
		t.Fatalf("a call without tools sent a tools field: %v", gemma.lastBody()["tools"])
	}
}

func TestProviderThinkingYieldsToTheCallsOwn(t *testing.T) {
	u := newUpstream(t, "u")
	c, err := NewClient("u", u.srv.URL, "k", "m")
	if err != nil {
		t.Fatal(err)
	}
	c.SetThinking("disabled")
	opts := &domain.GenerationOptions{Thinking: &domain.ThinkingOptions{Type: "enabled"}}
	if _, err := c.GenerateWithTools(context.Background(), hello, nil, opts); err != nil {
		t.Fatal(err)
	}
	th, _ := u.lastBody()["thinking"].(map[string]any)
	if th["type"] != "enabled" {
		t.Fatalf("thinking = %v, want the call's own enabled", u.lastBody()["thinking"])
	}
}

func TestProviderThinkingStopsAfterTheUpstreamRefusesIt(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if _, set := body["thinking"]; set {
			http.Error(w, `{"error":"unknown field: thinking"}`, http.StatusBadRequest)
			return
		}
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"ok"}}]}`)
	}))
	defer srv.Close()

	c, _ := NewClient("u", srv.URL, "k", "m")
	c.SetThinking("disabled")
	for range 2 {
		if _, err := c.GenerateWithTools(context.Background(), hello, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	if n := calls.Load(); n != 3 {
		t.Fatalf("calls = %d, want 3 (one refusal, then never sent again)", n)
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }
