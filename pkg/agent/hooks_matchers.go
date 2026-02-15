package agent

import (
	"regexp"
	"strings"
)

// ToolNameMatcher matches hooks by tool name(s)
type ToolNameMatcher struct {
	names map[string]bool
}

// NewToolNameMatcher creates a matcher that matches specific tool names
func NewToolNameMatcher(names ...string) *ToolNameMatcher {
	m := &ToolNameMatcher{
		names: make(map[string]bool),
	}
	for _, name := range names {
		m.names[strings.ToLower(name)] = true
	}
	return m
}

// Match returns true if the tool name matches
func (m *ToolNameMatcher) Match(event HookEvent, data HookData) bool {
	return m.names[strings.ToLower(data.ToolName)]
}

// AgentNameMatcher matches hooks by agent name(s)
type AgentNameMatcher struct {
	names map[string]bool
}

// NewAgentNameMatcher creates a matcher that matches specific agent names
func NewAgentNameMatcher(names ...string) *AgentNameMatcher {
	m := &AgentNameMatcher{
		names: make(map[string]bool),
	}
	for _, name := range names {
		m.names[strings.ToLower(name)] = true
	}
	return m
}

// Match returns true if the agent name matches
func (m *AgentNameMatcher) Match(event HookEvent, data HookData) bool {
	// Check SubagentName or AgentID
	return m.names[strings.ToLower(data.SubagentName)] ||
		m.names[strings.ToLower(data.AgentID)]
}

// SessionMatcher matches hooks by session ID(s)
type SessionMatcher struct {
	ids map[string]bool
}

// NewSessionMatcher creates a matcher that matches specific session IDs
func NewSessionMatcher(ids ...string) *SessionMatcher {
	m := &SessionMatcher{
		ids: make(map[string]bool),
	}
	for _, id := range ids {
		m.ids[id] = true
	}
	return m
}

// Match returns true if the session ID matches
func (m *SessionMatcher) Match(event HookEvent, data HookData) bool {
	return m.ids[data.SessionID]
}

// EventMatcher matches hooks by event type(s)
type EventMatcher struct {
	events map[HookEvent]bool
}

// NewEventMatcher creates a matcher that matches specific event types
func NewEventMatcher(events ...HookEvent) *EventMatcher {
	m := &EventMatcher{
		events: make(map[HookEvent]bool),
	}
	for _, event := range events {
		m.events[event] = true
	}
	return m
}

// Match returns true if the event type matches
func (m *EventMatcher) Match(event HookEvent, data HookData) bool {
	return m.events[event]
}

// RegexMatcher matches using regular expressions
type RegexMatcher struct {
	pattern *regexp.Regexp
	field   string // "tool_name", "agent_name", "session_id", etc.
}

// NewRegexMatcher creates a matcher using regex pattern
func NewRegexMatcher(pattern string, field string) (*RegexMatcher, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &RegexMatcher{
		pattern: re,
		field:   field,
	}, nil
}

// Match returns true if the pattern matches
func (m *RegexMatcher) Match(event HookEvent, data HookData) bool {
	var value string
	switch m.field {
	case "tool_name":
		value = data.ToolName
	case "agent_name", "agent_id":
		value = data.AgentID
	case "session_id":
		value = data.SessionID
	case "subagent_name":
		value = data.SubagentName
	case "goal":
		value = data.Goal
	default:
		return false
	}
	return m.pattern.MatchString(value)
}

// CompositeMode defines how composite matchers combine results
type CompositeMode string

const (
	CompositeModeAll CompositeMode = "all" // All matchers must match (AND)
	CompositeModeAny CompositeMode = "any" // Any matcher must match (OR)
)

// CompositeMatcher combines multiple matchers
type CompositeMatcher struct {
	matchers []HookMatcher
	mode     CompositeMode
}

// NewCompositeMatcher creates a composite matcher
func NewCompositeMatcher(mode CompositeMode, matchers ...HookMatcher) *CompositeMatcher {
	return &CompositeMatcher{
		matchers: matchers,
		mode:     mode,
	}
}

// AllMatchers creates a composite matcher that requires all matchers to match (AND)
func AllMatchers(matchers ...HookMatcher) *CompositeMatcher {
	return NewCompositeMatcher(CompositeModeAll, matchers...)
}

// AnyMatcher creates a composite matcher that requires any matcher to match (OR)
func AnyMatcher(matchers ...HookMatcher) *CompositeMatcher {
	return NewCompositeMatcher(CompositeModeAny, matchers...)
}

// Match returns true based on the composite mode
func (m *CompositeMatcher) Match(event HookEvent, data HookData) bool {
	if len(m.matchers) == 0 {
		return true
	}

	switch m.mode {
	case CompositeModeAll:
		for _, matcher := range m.matchers {
			if !matcher.Match(event, data) {
				return false
			}
		}
		return true
	case CompositeModeAny:
		for _, matcher := range m.matchers {
			if matcher.Match(event, data) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// NotMatcher inverts the result of another matcher
type NotMatcher struct {
	matcher HookMatcher
}

// NewNotMatcher creates a matcher that inverts another matcher
func NewNotMatcher(matcher HookMatcher) *NotMatcher {
	return &NotMatcher{
		matcher: matcher,
	}
}

// Match returns the inverse of the wrapped matcher
func (m *NotMatcher) Match(event HookEvent, data HookData) bool {
	return !m.matcher.Match(event, data)
}

// Not creates a not matcher (convenience function)
func Not(matcher HookMatcher) *NotMatcher {
	return NewNotMatcher(matcher)
}

// AlwaysMatcher always matches (useful for global hooks)
type AlwaysMatcher struct{}

// NewAlwaysMatcher creates a matcher that always returns true
func NewAlwaysMatcher() *AlwaysMatcher {
	return &AlwaysMatcher{}
}

// Match always returns true
func (m *AlwaysMatcher) Match(event HookEvent, data HookData) bool {
	return true
}

// NeverMatcher never matches
type NeverMatcher struct{}

// NewNeverMatcher creates a matcher that always returns false
func NewNeverMatcher() *NeverMatcher {
	return &NeverMatcher{}
}

// Match always returns false
func (m *NeverMatcher) Match(event HookEvent, data HookData) bool {
	return false
}

// MetadataMatcher matches based on metadata fields
type MetadataMatcher struct {
	key   string
	value interface{}
}

// NewMetadataMatcher creates a matcher that checks metadata fields
func NewMetadataMatcher(key string, value interface{}) *MetadataMatcher {
	return &MetadataMatcher{
		key:   key,
		value: value,
	}
}

// Match returns true if metadata key matches value
func (m *MetadataMatcher) Match(event HookEvent, data HookData) bool {
	if data.Metadata == nil {
		return false
	}
	val, ok := data.Metadata[m.key]
	if !ok {
		return false
	}
	return val == m.value
}

// ErrorMatcher matches if there was an error
type ErrorMatcher struct {
	matchOnError bool
}

// NewErrorMatcher creates a matcher that matches based on error presence
func NewErrorMatcher(matchOnError bool) *ErrorMatcher {
	return &ErrorMatcher{
		matchOnError: matchOnError,
	}
}

// Match returns true based on error presence
func (m *ErrorMatcher) Match(event HookEvent, data HookData) bool {
	hasError := data.ToolError != nil || data.Error != nil
	return hasError == m.matchOnError
}

// OnError creates a matcher that matches when there's an error
func OnError() *ErrorMatcher {
	return NewErrorMatcher(true)
}

// OnSuccess creates a matcher that matches when there's no error
func OnSuccess() *ErrorMatcher {
	return NewErrorMatcher(false)
}
