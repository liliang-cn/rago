package providers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/liliang-cn/agent-go/v3/pkg/domain"
	"github.com/stretchr/testify/require"
)

type effortServer struct {
	mu       sync.Mutex
	bodies   []map[string]any
	paths    []string
	refuseIt bool
}

func (s *effortServer) handler(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	var body map[string]any
	_ = json.Unmarshal(raw, &body)
	s.mu.Lock()
	s.bodies = append(s.bodies, body)
	s.paths = append(s.paths, r.URL.Path)
	refuse := s.refuseIt
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if _, has := body["reasoning_effort"]; has && refuse {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"Unrecognized request argument supplied: reasoning_effort"}}`))
		return
	}
	_, _ = w.Write([]byte(`{"id":"c","choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"ok"}}]}`))
}

func (s *effortServer) effort(i int) (any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.bodies[i]["reasoning_effort"]
	return v, ok
}

func newEffortProvider(t *testing.T, s *effortServer, effort string) *OpenAILLMProvider {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(s.handler))
	t.Cleanup(srv.Close)
	p, err := NewOpenAILLMProvider(&domain.OpenAIProviderConfig{
		APIKey: "k", BaseURL: srv.URL, LLMModel: "m", EmbeddingModel: "e", ReasoningEffort: effort,
	})
	require.NoError(t, err)
	return p.(*OpenAILLMProvider)
}

func TestConfiguredReasoningEffortIsSentOnChatCompletions(t *testing.T) {
	s := &effortServer{}
	p := newEffortProvider(t, s, "none")
	ctx := context.Background()

	_, err := p.Generate(ctx, "hi", nil)
	require.NoError(t, err)
	_, err = p.GenerateWithTools(ctx, []domain.Message{{Role: "user", Content: "hi"}}, nil, nil)
	require.NoError(t, err)

	for i := 0; i < 2; i++ {
		v, ok := s.effort(i)
		require.True(t, ok, "chat completion %d carried no reasoning_effort", i)
		require.Equal(t, "none", v)
	}
}

func TestACallThatSaysHowToReasonKeepsItsOwnSetting(t *testing.T) {
	s := &effortServer{}
	p := newEffortProvider(t, s, "none")
	_, err := p.Generate(context.Background(), "hi", &domain.GenerationOptions{Thinking: &domain.ThinkingOptions{Type: "enabled"}})
	require.NoError(t, err)
	_, ok := s.effort(0)
	require.False(t, ok, "a call with its own thinking switch must not also get reasoning_effort")
}

func TestAnUpstreamThatRefusesReasoningEffortIsAskedOnce(t *testing.T) {
	s := &effortServer{refuseIt: true}
	p := newEffortProvider(t, s, "low")
	ctx := context.Background()

	out, err := p.Generate(ctx, "hi", nil)
	require.NoError(t, err)
	require.Equal(t, "ok", out)
	_, err = p.Generate(ctx, "again", nil)
	require.NoError(t, err)

	// refused with the field, answered without it, then never sent again
	require.Len(t, s.bodies, 3)
	_, first := s.effort(0)
	_, retry := s.effort(1)
	_, later := s.effort(2)
	require.True(t, first)
	require.False(t, retry)
	require.False(t, later)
}

func TestNoReasoningEffortConfiguredSendsNone(t *testing.T) {
	s := &effortServer{}
	p := newEffortProvider(t, s, "")
	_, err := p.Generate(context.Background(), "hi", nil)
	require.NoError(t, err)
	_, ok := s.effort(0)
	require.False(t, ok)
}

// Only chat completions get the field: some servers refuse an embeddings
// request that carries it.
func TestReasoningEffortIsAddedToChatCompletionsOnly(t *testing.T) {
	body, ok := withReasoningEffort([]byte(`{"model":"m","messages":[]}`), "none")
	require.True(t, ok)
	require.Contains(t, string(body), `"reasoning_effort":"none"`)

	var seen []string
	e := &reasoningEffort{level: "none"}
	next := func(r *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(r.Body)
		seen = append(seen, string(b))
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	}
	req := httptest.NewRequest(http.MethodPost, "http://x/v1/embeddings", strings.NewReader(`{"model":"e","input":"hi"}`))
	_, err := e.add(req, next)
	require.NoError(t, err)
	require.NotContains(t, seen[0], "reasoning_effort")
}
