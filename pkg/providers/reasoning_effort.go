package providers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/openai/openai-go/v3/option"
)

// Reasoning effort on the OpenAI-compatible provider.
//
// pool.Client sends a configured reasoning_effort (pool/reasoning_effort.go);
// the provider NewOpenAILLMProvider builds had no way to, so a caller on a
// reasoning model could not turn its thinking down. qwen3.8:27b behind ollama
// thinks before every call by default: one steward question with a single
// knowledge_search took 110 s.
//
// The field is added on the wire, to chat completions only — an embeddings
// request carrying it is refused by some servers — and only when the call has
// not said how to reason itself (its own reasoning_effort, or a "thinking"
// switch). An upstream that refuses the field gets the request again without
// it, and is not sent it again.
type reasoningEffort struct {
	level    string
	rejected atomic.Bool
}

func (e *reasoningEffort) add(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
	if e.rejected.Load() || req.Method != http.MethodPost || req.Body == nil ||
		!strings.HasSuffix(req.URL.Path, "/chat/completions") {
		return next(req)
	}
	body, err := io.ReadAll(req.Body)
	_ = req.Body.Close()
	if err != nil {
		return nil, err
	}
	sent, ok := withReasoningEffort(body, e.level)
	if !ok {
		return next(withBody(req, body))
	}
	resp, err := next(withBody(req, sent))
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		return resp, err
	}
	refusal, rerr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if rerr == nil && bytes.Contains(refusal, []byte("reasoning_effort")) {
		e.rejected.Store(true)
		return next(withBody(req, body))
	}
	resp.Body = io.NopCloser(bytes.NewReader(refusal))
	return resp, rerr
}

// withReasoningEffort is body with reasoning_effort set, unless the request
// already says how to reason or is not a JSON object.
func withReasoningEffort(body []byte, level string) ([]byte, bool) {
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil || fields == nil {
		return body, false
	}
	if _, set := fields["reasoning_effort"]; set {
		return body, false
	}
	if _, set := fields["thinking"]; set {
		return body, false
	}
	fields["reasoning_effort"], _ = json.Marshal(level)
	out, err := json.Marshal(fields)
	if err != nil {
		return body, false
	}
	return out, true
}

func withBody(req *http.Request, body []byte) *http.Request {
	out := req.Clone(req.Context())
	out.Body = io.NopCloser(bytes.NewReader(body))
	out.ContentLength = int64(len(body))
	out.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }
	return out
}
