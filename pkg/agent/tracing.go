package agent

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SpanKind represents the type of span
type SpanKind string

const (
	SpanKindInternal  SpanKind = "internal"  // Internal operations
	SpanKindAgent     SpanKind = "agent"     // Agent operations
	SpanKindTool      SpanKind = "tool"      // Tool execution
	SpanKindGuardrail SpanKind = "guardrail" // Guardrail checks
	SpanKindHandoff   SpanKind = "handoff"   // Handoff operations
	SpanKindLLM       SpanKind = "llm"       // LLM calls
)

// SpanStatus represents the status of a span
type SpanStatus string

const (
	SpanStatusOK       SpanStatus = "ok"
	SpanStatusError    SpanStatus = "error"
	SpanStatusCanceled SpanStatus = "canceled"
)

// Span represents a single operation in a trace
type Span struct {
	ID          string            `json:"id"`
	TraceID     string            `json:"trace_id"`
	ParentID    string            `json:"parent_id,omitempty"`
	Name        string            `json:"name"`
	Kind        SpanKind          `json:"kind"`
	Status      SpanStatus        `json:"status"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     time.Time         `json:"end_time,omitempty"`
	Duration    time.Duration     `json:"duration,omitempty"`
	Events      []SpanEvent       `json:"events,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
	AgentID     string            `json:"agent_id,omitempty"`
	SessionID   string            `json:"session_id,omitempty"`
	PlanID      string            `json:"plan_id,omitempty"`
	StepID      string            `json:"step_id,omitempty"`
}

// SpanEvent represents an event within a span
type SpanEvent struct {
	Time       time.Time         `json:"time"`
	Name       string            `json:"name"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// Trace represents a collection of related spans
type Trace struct {
	ID        string                 `json:"id"`
	RootSpan  *Span                  `json:"root_span"`
	Spans     map[string]*Span       `json:"spans"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time,omitempty"`
	Duration  time.Duration          `json:"duration,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Tracer creates and manages spans
type Tracer struct {
	mu         sync.RWMutex
	traces     map[string]*Trace
	currentSpans map[string]*Span
	enabled    bool
}

// NewTracer creates a new tracer
func NewTracer() *Tracer {
	return &Tracer{
		traces:       make(map[string]*Trace),
		currentSpans: make(map[string]*Span),
		enabled:      true,
	}
}

// IsEnabled returns whether tracing is enabled
func (t *Tracer) IsEnabled() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.enabled
}

// SetEnabled sets whether tracing is enabled
func (t *Tracer) SetEnabled(enabled bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.enabled = enabled
}

// StartSpan starts a new span
func (t *Tracer) StartSpan(ctx context.Context, name string, kind SpanKind, opts ...SpanOption) (*Span, context.Context) {
	if !t.enabled {
		return nil, ctx
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Get trace ID from context or create new
	traceID := getTraceID(ctx)
	if traceID == "" {
		traceID = newID()
		ctx = withTraceID(ctx, traceID)
	}

	// Get parent span ID from context
	parentID := getParentSpanID(ctx)

	// Create span
	span := &Span{
		ID:        newID(),
		TraceID:   traceID,
		ParentID:  parentID,
		Name:      name,
		Kind:      kind,
		Status:    SpanStatusOK,
		StartTime: time.Now(),
		Events:    []SpanEvent{},
		Attributes: make(map[string]string),
	}

	// Apply options
	for _, opt := range opts {
		opt(span)
	}

	// Get or create trace
	trace, exists := t.traces[traceID]
	if !exists {
		trace = &Trace{
			ID:        traceID,
			Spans:     make(map[string]*Span),
			StartTime: time.Now(),
			Metadata:  make(map[string]interface{}),
		}
		t.traces[traceID] = trace
	}

	// Add span to trace
	trace.Spans[span.ID] = span
	if span.ParentID == "" {
		trace.RootSpan = span
	}

	// Store as current span
	t.currentSpans[span.ID] = span

	// Add span context to context
	ctx = withParentSpanID(ctx, span.ID)
	ctx = withCurrentSpan(ctx, span)

	return span, ctx
}

// EndSpan ends a span
func (t *Tracer) EndSpan(span *Span, err error) {
	if span == nil || !t.enabled {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	span.EndTime = time.Now()
	span.Duration = span.EndTime.Sub(span.StartTime)

	if err != nil {
		span.Status = SpanStatusError
		span.Attributes["error"] = err.Error()
	}

	// Update trace if this is the root span
	if trace, ok := t.traces[span.TraceID]; ok {
		if trace.RootSpan != nil && trace.RootSpan.ID == span.ID {
			trace.EndTime = span.EndTime
			trace.Duration = span.Duration
		}
	}

	delete(t.currentSpans, span.ID)
}

// AddEvent adds an event to a span
func (t *Tracer) AddEvent(span *Span, name string, attrs map[string]string) {
	if span == nil || !t.enabled {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	span.Events = append(span.Events, SpanEvent{
		Time:       time.Now(),
		Name:       name,
		Attributes: attrs,
	})
}

// SetAttribute sets an attribute on a span
func (t *Tracer) SetAttribute(span *Span, key, value string) {
	if span == nil || !t.enabled {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	span.Attributes[key] = value
}

// GetTrace retrieves a trace by ID
func (t *Tracer) GetTrace(traceID string) (*Trace, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	trace, ok := t.traces[traceID]
	return trace, ok
}

// ListTraces returns all traces
func (t *Tracer) ListTraces() []*Trace {
	t.mu.RLock()
	defer t.mu.RUnlock()

	traces := make([]*Trace, 0, len(t.traces))
	for _, trace := range t.traces {
		traces = append(traces, trace)
	}
	return traces
}

// Clear removes all traces
func (t *Tracer) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.traces = make(map[string]*Trace)
	t.currentSpans = make(map[string]*Span)
}

// SpanOption configures a span
type SpanOption func(*Span)

// WithSpanAgentID sets the agent ID
func WithSpanAgentID(agentID string) SpanOption {
	return func(s *Span) {
		s.AgentID = agentID
	}
}

// WithSpanSessionID sets the session ID
func WithSpanSessionID(sessionID string) SpanOption {
	return func(s *Span) {
		s.SessionID = sessionID
	}
}

// WithSpanPlanID sets the plan ID
func WithSpanPlanID(planID string) SpanOption {
	return func(s *Span) {
		s.PlanID = planID
	}
}

// WithSpanStepID sets the step ID
func WithSpanStepID(stepID string) SpanOption {
	return func(s *Span) {
		s.StepID = stepID
	}
}

// WithSpanAttributes sets multiple attributes
func WithSpanAttributes(attrs map[string]string) SpanOption {
	return func(s *Span) {
		for k, v := range attrs {
			s.Attributes[k] = v
		}
	}
}

// Context key types for trace context
type contextKey string

const (
	traceIDKey     contextKey = "trace_id"
	parentSpanKey  contextKey = "parent_span_id"
	currentSpanKey contextKey = "current_span"
)

func getTraceID(ctx context.Context) string {
	if id, ok := ctx.Value(traceIDKey).(string); ok {
		return id
	}
	return ""
}

func getParentSpanID(ctx context.Context) string {
	if id, ok := ctx.Value(parentSpanKey).(string); ok {
		return id
	}
	return ""
}

func withTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

func withParentSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, parentSpanKey, spanID)
}

func withCurrentSpan(ctx context.Context, span *Span) context.Context {
	return context.WithValue(ctx, currentSpanKey, span)
}

// GetCurrentSpan retrieves the current span from context
func GetCurrentSpan(ctx context.Context) (*Span, bool) {
	if span, ok := ctx.Value(currentSpanKey).(*Span); ok {
		return span, true
	}
	return nil, false
}

// GetTraceID retrieves the trace ID from context
func GetTraceID(ctx context.Context) string {
	return getTraceID(ctx)
}

// newID generates a unique ID
func newID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
