package pool

import (
	"strings"
	"testing"
)

// A pool with nothing configured must say so, and say what to do, rather than
// reporting a health problem that does not exist.
func TestEmptyPoolSaysNoProviderConfigured(t *testing.T) {
	p, err := NewPool(PoolConfig{Enabled: true, Strategy: StrategyRoundRobin})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Get()
	if err == nil {
		t.Fatal("expected an error from an empty pool")
	}
	msg := err.Error()
	for _, want := range []string{"no healthy clients available", "no LLM provider is configured", "WithLLM", "getting-started.md"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q does not mention %q", msg, want)
		}
	}
}
