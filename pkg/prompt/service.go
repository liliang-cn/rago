package prompt

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
)

// Manager handles prompt registration, overrides, and rendering
type Manager struct {
	mu       sync.RWMutex
	prompts  map[string]string
	defaults map[string]string
	funcMap  template.FuncMap
}

// NewManager creates a new prompt manager
func NewManager() *Manager {
	m := &Manager{
		prompts:  make(map[string]string),
		defaults: make(map[string]string),
		funcMap: template.FuncMap{
			"add": func(a, b int) int { return a + b },
			"sub": func(a, b int) int { return a - b },
			"mul": func(a, b int) int { return a * b },
			"div": func(a, b int) int { return a / b },
		},
	}
	m.loadDefaults()
	return m
}

// RegisterDefault registers a default prompt that can be overridden
func (m *Manager) RegisterDefault(key, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaults[key] = content
}

// SetPrompt overrides a prompt (either default or new)
func (m *Manager) SetPrompt(key, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.prompts[key] = content
}

// Get returns the effective prompt content for a key
func (m *Manager) Get(key string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check overrides first
	if p, ok := m.prompts[key]; ok {
		return p
	}
	// Fallback to default
	return m.defaults[key]
}

// Render renders a prompt template with provided data
func (m *Manager) Render(key string, data interface{}) (string, error) {
	content := m.Get(key)
	if content == "" {
		return "", fmt.Errorf("prompt template not found for key: %s", key)
	}

	tmpl, err := template.New(key).Funcs(m.funcMap).Parse(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse prompt template %s: %w", key, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render prompt template %s: %w", key, err)
	}

	return buf.String(), nil
}

// LoadFromDir scans a directory for markdown files and loads them as prompt overrides
// Filename format: namespace.key.md (e.g., planner.intent.md)
func (m *Manager) LoadFromDir(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return nil // Directory doesn't exist, skip
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read prompt directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		path := filepath.Join(dirPath, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		// Use filename (minus .md) as the key
		key := strings.TrimSuffix(entry.Name(), ".md")
		m.SetPrompt(key, string(content))
	}

	return nil
}
