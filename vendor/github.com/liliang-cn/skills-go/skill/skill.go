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
	Meta Meta  `yaml:"frontmatter" json:"meta"`
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
	Name                  string   `yaml:"name,omitempty"`
	Description           string   `yaml:"description,omitempty"`
	ArgumentHint          string   `yaml:"argument-hint,omitempty"`
	DisableModelInvocation bool   `yaml:"disable-model-invocation,omitempty"`
	UserInvocable         *bool    `yaml:"user-invocable,omitempty"`
	AllowedTools          []string `yaml:"allowed-tools,omitempty"`
	Model                 string   `yaml:"model,omitempty"`
	Context               string   `yaml:"context,omitempty"`
	Agent                 string   `yaml:"agent,omitempty"`
	Hooks                 *Hooks   `yaml:"hooks,omitempty"`
}

// Hooks defines skill lifecycle hooks
type Hooks struct {
	BeforeInvoke []string `yaml:"before_invoke,omitempty"`
	AfterInvoke  []string `yaml:"after_invoke,omitempty"`
	OnError      []string `yaml:"on_error,omitempty"`
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
