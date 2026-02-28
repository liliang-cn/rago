package skill

import (
	"time"
)

// LoadLevel indicates how much of the skill is loaded
type LoadLevel int

const (
	// LoadLevelMetadata only loads name, description and other frontmatter
	LoadLevelMetadata LoadLevel = iota
	// LoadLevelContent loads frontmatter and markdown content, but no external resources
	LoadLevelContent
	// LoadLevelFull loads everything including scripts and references
	LoadLevelFull
)

// Skill represents a complete skill with metadata, content, and resources
type Skill struct {
	// Metadata
	Meta Meta   `yaml:"frontmatter" json:"meta"`
	Path string `json:"path"`
	Name string `json:"name"`

	// Content
	Content string `json:"content"` // SKILL.md content without frontmatter
	Raw     string `json:"raw"`     // Raw file content including frontmatter

	// Resources
	Resources *Resources `json:"resources,omitempty"`

	// State
	LoadedAt  time.Time `json:"loaded_at"`
	Version   string    `json:"version,omitempty"`
	LoadLevel LoadLevel `json:"load_level"`
}

// Meta is the YAML frontmatter of SKILL.md
type Meta struct {
	Name                   string            `yaml:"name,omitempty"`
	Description            string            `yaml:"description,omitempty"`
	ArgumentHint           string            `yaml:"argument-hint,omitempty"`
	DisableModelInvocation bool              `yaml:"disable-model-invocation,omitempty"`
	UserInvocable          *bool             `yaml:"user-invocable,omitempty"`
	AllowedTools           StringOrSlice     `yaml:"allowed-tools,omitempty"`
	Model                  string            `yaml:"model,omitempty"`
	Context                string            `yaml:"context,omitempty"`
	Agent                  string            `yaml:"agent,omitempty"`
	Hooks                  *Hooks            `yaml:"hooks,omitempty"`
	Metadata               map[string]string `yaml:"metadata,omitempty"`
	License                string            `yaml:"license,omitempty"`
	Compatibility          string            `yaml:"compatibility,omitempty"`
}

// Hooks defines skill lifecycle hooks
type Hooks struct {
	BeforeInvoke []string     `yaml:"before_invoke,omitempty"`
	AfterInvoke  []string     `yaml:"after_invoke,omitempty"`
	OnError      []string     `yaml:"on_error,omitempty"`
	PreToolUse   []HookAction `yaml:"PreToolUse,omitempty"`
	PostToolUse  []HookAction `yaml:"PostToolUse,omitempty"`
	Stop         []HookAction `yaml:"Stop,omitempty"`
}

// HookAction defines a single hook action
type HookAction struct {
	Matcher string       `yaml:"matcher,omitempty"`
	Hooks   []HookConfig `yaml:"hooks,omitempty"`
}

// HookConfig defines hook configuration
type HookConfig struct {
	Type    string `yaml:"type,omitempty"`
	Command string `yaml:"command,omitempty"`
}

// StringOrSlice allows a field to be either a string or a slice of strings
type StringOrSlice []string

func (s *StringOrSlice) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var single string
	if err := unmarshal(&single); err == nil {
		*s = parseDelimitedString(single)
		return nil
	}

	var multi []string
	if err := unmarshal(&multi); err != nil {
		return err
	}
	*s = multi
	return nil
}

func parseDelimitedString(s string) []string {
	s = trimSpace(s)
	if s == "" {
		return nil
	}

	if stringsContains(s, ",") {
		return splitAndTrim(s, ",")
	}
	return splitAndTrim(s, " ")
}

func splitAndTrim(s, sep string) []string {
	parts := stringsSplit(s, sep)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = trimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func stringsContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func stringsSplit(s, sep string) []string {
	if sep == "" {
		return []string{s}
	}
	var result []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// Resources contains additional skill resources
type Resources struct {
	Scripts    []Script    `json:"scripts,omitempty"`
	References []Reference `json:"references,omitempty"`
	Assets     []Asset     `json:"assets,omitempty"`
	Templates  []Template  `json:"templates,omitempty"`
}

// Script represents an executable script
type Script struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Language string `json:"language"`
}

// Reference represents a reference document
type Reference struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`
}

// Asset represents an asset file
type Asset struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type,omitempty"`
}

// Template represents a template file
type Template struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Content string `json:"content,omitempty"`
}

// InvocationContext is the context when invoking a skill
type InvocationContext struct {
	Arguments   []string          `json:"arguments"`
	SessionID   string            `json:"session_id"`
	Environment map[string]string `json:"environment"`
	Variables   map[string]string `json:"variables"`
	UserID      string            `json:"user_id,omitempty"`
	IsUser      bool              `json:"is_user"` // true if invoked by user
}

// InvocationResult is the result of skill invocation
type InvocationResult struct {
	Content    string            `json:"content"`
	Rendered   string            `json:"rendered"`
	Referenced []string          `json:"referenced,omitempty"`
	ScriptsRun []string          `json:"scripts_run,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Duration   time.Duration     `json:"duration"`
	Error      error             `json:"error,omitempty"`
}

// IsUserInvocable returns whether the skill can be invoked by users
func (s *Skill) IsUserInvocable() bool {
	if s.Meta.UserInvocable == nil {
		return true // default is true
	}
	return *s.Meta.UserInvocable
}

// IsModelInvocable returns whether the skill can be invoked by the model
func (s *Skill) IsModelInvocable() bool {
	return !s.Meta.DisableModelInvocation
}

// ShouldFork returns whether the skill should run in a forked context
func (s *Skill) ShouldFork() bool {
	return s.Meta.Context == "fork"
}

// GetModel returns the model to use, or empty string for default
func (s *Skill) GetModel() string {
	return s.Meta.Model
}

// GetAgent returns the agent type to use for forked context
func (s *Skill) GetAgent() string {
	if s.Meta.Agent != "" {
		return s.Meta.Agent
	}
	return "general-purpose"
}
