package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// ExportFormat defines the export format
type ExportFormat string

const (
	ExportFormatJSON  ExportFormat = "json"
	ExportFormatPretty ExportFormat = "pretty"
	ExportFormatConsole ExportFormat = "console"
)

// TraceExporter exports traces to various formats
type TraceExporter struct {
	tracer *Tracer
}

// NewTraceExporter creates a new trace exporter
func NewTraceExporter(tracer *Tracer) *TraceExporter {
	return &TraceExporter{tracer: tracer}
}

// Export exports a trace to the specified format
func (e *TraceExporter) Export(traceID string, format ExportFormat) (string, error) {
	trace, ok := e.tracer.GetTrace(traceID)
	if !ok {
		return "", fmt.Errorf("trace not found: %s", traceID)
	}

	switch format {
	case ExportFormatJSON:
		return e.exportJSON(trace)
	case ExportFormatPretty:
		return e.exportPretty(trace)
	case ExportFormatConsole:
		e.exportConsole(trace)
		return "", nil
	default:
		return "", fmt.Errorf("unknown export format: %s", format)
	}
}

// ExportAll exports all traces
func (e *TraceExporter) ExportAll(format ExportFormat) (string, error) {
	traces := e.tracer.ListTraces()

	switch format {
	case ExportFormatJSON:
		data, err := json.MarshalIndent(traces, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal traces: %w", err)
		}
		return string(data), nil
	case ExportFormatPretty:
		var sb strings.Builder
		for i, trace := range traces {
			if i > 0 {
				sb.WriteString("\n\n")
			}
			pretty, err := e.exportPretty(trace)
			if err != nil {
				return "", err
			}
			sb.WriteString(pretty)
		}
		return sb.String(), nil
	case ExportFormatConsole:
		for _, trace := range traces {
			e.exportConsole(trace)
			fmt.Println()
		}
		return "", nil
	default:
		return "", fmt.Errorf("unknown export format: %s", format)
	}
}

// exportJSON exports a trace as JSON
func (e *TraceExporter) exportJSON(trace *Trace) (string, error) {
	data, err := json.MarshalIndent(trace, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal trace: %w", err)
	}
	return string(data), nil
}

// exportPretty exports a trace in a human-readable format
func (e *TraceExporter) exportPretty(trace *Trace) (string, error) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Trace: %s\n", trace.ID))
	sb.WriteString(fmt.Sprintf("Duration: %s\n", trace.Duration))
	sb.WriteString(fmt.Sprintf("Spans: %d\n", len(trace.Spans)))
	sb.WriteString("\n")

	for _, span := range trace.Spans {
		sb.WriteString(e.formatSpan(span, 0))
	}

	return sb.String(), nil
}

// exportConsole exports a trace to console
func (e *TraceExporter) exportConsole(trace *Trace) {
	fmt.Printf("╔════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  Trace: %s\n", trace.ID)
	if len(trace.ID) < 50 {
		fmt.Printf("%s║", strings.Repeat(" ", 50-len(trace.ID)))
	}
	fmt.Printf("║\n")
	fmt.Printf("║  Duration: %s\n", formatDuration(trace.Duration))
	fmt.Printf("║  Spans: %d\n", len(trace.Spans))
	fmt.Printf("╚════════════════════════════════════════════════════════════╝\n")
	fmt.Println()

	for _, span := range trace.Spans {
		e.printSpan(span, 0)
	}
}

// formatSpan formats a span for pretty output
func (e *TraceExporter) formatSpan(span *Span, indent int) string {
	indentStr := strings.Repeat("  ", indent)

	var sb strings.Builder

	// Status icon
	statusIcon := "✓"
	if span.Status == SpanStatusError {
		statusIcon = "✗"
	} else if span.Status == SpanStatusCanceled {
		statusIcon = "⊘"
	}

	sb.WriteString(fmt.Sprintf("%s[%s] %s %s (%s)\n",
		indentStr, span.Kind, statusIcon, span.Name, formatDuration(span.Duration)))

	// Add attributes
	if len(span.Attributes) > 0 {
		for k, v := range span.Attributes {
			sb.WriteString(fmt.Sprintf("%s  %s: %s\n", indentStr, k, v))
		}
	}

	// Add events
	if len(span.Events) > 0 {
		for _, event := range span.Events {
			sb.WriteString(fmt.Sprintf("%s  @ %s: %s\n", indentStr, formatTime(event.Time), event.Name))
		}
	}

	return sb.String()
}

// printSpan prints a span to console
func (e *TraceExporter) printSpan(span *Span, indent int) {
	indentStr := strings.Repeat("  ", indent)

	// Status icon
	statusIcon := "✓"
	statusColor := "\033[32m" // Green
	if span.Status == SpanStatusError {
		statusIcon = "✗"
		statusColor = "\033[31m" // Red
	} else if span.Status == SpanStatusCanceled {
		statusIcon = "⊘"
		statusColor = "\033[33m" // Yellow
	}

	// Kind icon
	kindIcon := "•"
	switch span.Kind {
	case SpanKindAgent:
		kindIcon = "🤖"
	case SpanKindTool:
		kindIcon = "🔧"
	case SpanKindGuardrail:
		kindIcon = "🛡️"
	case SpanKindHandoff:
		kindIcon = "↪️"
	case SpanKindLLM:
		kindIcon = "💬"
	}

	fmt.Printf("%s%s %s [%s]%s %s %s (%s)\n",
		indentStr, kindIcon, statusIcon, statusColor, span.Kind, "\033[0m", span.Name, formatDuration(span.Duration))

	// Add attributes
	if len(span.Attributes) > 0 {
		for k, v := range span.Attributes {
			fmt.Printf("%s  • %s: %s\n", indentStr, k, v)
		}
	}

	// Add events
	if len(span.Events) > 0 {
		for _, event := range span.Events {
			fmt.Printf("%s  @ %s: %s\n", indentStr, formatTime(event.Time), event.Name)
		}
	}
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%d ns", d.Nanoseconds())
	} else if d < time.Millisecond {
		return fmt.Sprintf("%.1f µs", float64(d.Nanoseconds())/1000)
	} else if d < time.Second {
		return fmt.Sprintf("%.2f ms", float64(d.Microseconds())/1000)
	} else {
		return fmt.Sprintf("%.2f s", d.Seconds())
	}
}

// formatTime formats a time for display
func formatTime(t time.Time) string {
	return t.Format("15:04:05.000")
}

// ExportToFile exports a trace to a file
func (e *TraceExporter) ExportToFile(traceID string, filename string, format ExportFormat) error {
	data, err := e.Export(traceID, format)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, []byte(data), 0644)
}

// ExportAllToFile exports all traces to a file
func (e *TraceExporter) ExportAllToFile(filename string, format ExportFormat) error {
	data, err := e.ExportAll(format)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, []byte(data), 0644)
}

// TraceSummary provides a summary of a trace
type TraceSummary struct {
	TraceID        string            `json:"trace_id"`
	Duration       time.Duration     `json:"duration"`
	SpanCount      int               `json:"span_count"`
	ErrorCount     int               `json:"error_count"`
	AgentID        string            `json:"agent_id,omitempty"`
	SessionID      string            `json:"session_id,omitempty"`
	SpanKinds      map[SpanKind]int  `json:"span_kinds"`
	LongestSpan    *Span             `json:"longest_span,omitempty"`
	SlowestSpan    *Span             `json:"slowest_span,omitempty"`
}

// Summarize creates a summary of a trace
func (e *TraceExporter) Summarize(traceID string) (*TraceSummary, error) {
	trace, ok := e.tracer.GetTrace(traceID)
	if !ok {
		return nil, fmt.Errorf("trace not found: %s", traceID)
	}

	summary := &TraceSummary{
		TraceID:   trace.ID,
		Duration:  trace.Duration,
		SpanCount: len(trace.Spans),
		SpanKinds: make(map[SpanKind]int),
	}

	var longest *Span
	var slowest *Span
	var maxDuration time.Duration

	for _, span := range trace.Spans {
		summary.SpanKinds[span.Kind]++

		if span.Status == SpanStatusError {
			summary.ErrorCount++
		}

		if span.Duration > maxDuration {
			maxDuration = span.Duration
			longest = span
		}

		// Find slowest by kind (LLM calls typically)
		if span.Kind == SpanKindLLM {
			if slowest == nil || span.Duration > slowest.Duration {
				slowest = span
			}
		}

		// Set agent/session IDs
		if span.AgentID != "" {
			summary.AgentID = span.AgentID
		}
		if span.SessionID != "" {
			summary.SessionID = span.SessionID
		}
	}

	summary.LongestSpan = longest
	summary.SlowestSpan = slowest

	return summary, nil
}
