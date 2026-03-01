package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ============================================================
// Memory Files (inspired by OpenClaw)
// ============================================================
//
// OpenClaw uses several Markdown files for persistent context:
// - MEMORY.md: Long-term memory
// - AGENTS.md: Agent configuration and behavior
// - SOUL.md: Agent personality
// - TOOLS.md: Available tools documentation
// - HEARTBEAT.md: Checklist for autonomous operation

// MemoryManager manages memory files for LongRun
type MemoryManager struct {
	workDir    string
	files      map[string]*MemoryFile
	mu         sync.RWMutex
	autoSave   bool
	saveInterval time.Duration
}

// MemoryFile represents a memory file
type MemoryFile struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Content     string    `json:"content"`
	LastUpdated time.Time `json:"last_updated"`
}

// NewMemoryManager creates a new memory manager
func NewMemoryManager(workDir string) (*MemoryManager, error) {
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}

	mm := &MemoryManager{
		workDir:      workDir,
		files:        make(map[string]*MemoryFile),
		autoSave:     true,
		saveInterval: 30 * time.Second,
	}

	// Initialize default files
	defaultFiles := map[string]string{
		"MEMORY.md":    memoryDefault,
		"AGENTS.md":    agentsDefault,
		"SOUL.md":      soulDefault,
		"TOOLS.md":     toolsDefault,
		"HEARTBEAT.md": heartbeatDefault,
	}

	for name, content := range defaultFiles {
		path := filepath.Join(workDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return nil, fmt.Errorf("failed to create %s: %w", name, err)
			}
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", name, err)
		}

		mm.files[name] = &MemoryFile{
			Name:        name,
			Path:        path,
			Content:     string(content),
			LastUpdated: time.Now(),
		}
	}

	return mm, nil
}

// Get retrieves a memory file by name
func (mm *MemoryManager) Get(name string) (*MemoryFile, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	file, ok := mm.files[name]
	if !ok {
		return nil, fmt.Errorf("memory file not found: %s", name)
	}

	// Read latest content
	content, err := os.ReadFile(file.Path)
	if err != nil {
		return nil, err
	}

	file.Content = string(content)
	return file, nil
}

// Set updates a memory file
func (mm *MemoryManager) Set(name, content string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	file, ok := mm.files[name]
	if !ok {
		path := filepath.Join(mm.workDir, name)
		file = &MemoryFile{
			Name: name,
			Path: path,
		}
		mm.files[name] = file
	}

	if err := os.WriteFile(file.Path, []byte(content), 0644); err != nil {
		return err
	}

	file.Content = content
	file.LastUpdated = time.Now()
	return nil
}

// Append appends content to a memory file
func (mm *MemoryManager) Append(name, content string) error {
	file, err := mm.Get(name)
	if err != nil {
		return err
	}

	newContent := file.Content + "\n" + content
	return mm.Set(name, newContent)
}

// BuildContext builds the full context string for the agent
func (mm *MemoryManager) BuildContext() (string, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	var sb strings.Builder

	// Order matters for context priority
	order := []string{"SOUL.md", "AGENTS.md", "TOOLS.md", "MEMORY.md", "HEARTBEAT.md"}

	for _, name := range order {
		file, ok := mm.files[name]
		if !ok {
			continue
		}

		content, err := os.ReadFile(file.Path)
		if err != nil {
			continue
		}

		sb.WriteString(fmt.Sprintf("\n\n# %s\n\n", name))
		sb.WriteString(string(content))
	}

	return sb.String(), nil
}

// AppendMemory appends a memory entry
func (mm *MemoryManager) AppendMemory(entry string) error {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	formatted := fmt.Sprintf("\n## %s\n\n%s\n", timestamp, entry)
	return mm.Append("MEMORY.md", formatted)
}

// List returns all memory files
func (mm *MemoryManager) List() []*MemoryFile {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	files := make([]*MemoryFile, 0, len(mm.files))
	for _, file := range mm.files {
		files = append(files, file)
	}
	return files
}

// Default file contents

const memoryDefault = `# Memory

Long-term memory for the agent. Entries are timestamped and persist across sessions.

## Usage

- Agent reads this file at each heartbeat
- Agent appends important learnings and decisions
- Manual edits are preserved

## Entries

`

const agentsDefault = `# Agent Configuration

## Identity

You are an autonomous AI assistant running in LongRun mode.
You execute tasks from HEARTBEAT.md and the task queue on a schedule.

## Behavior

1. Check HEARTBEAT.md at each heartbeat
2. Process pending checklist items
3. Execute tasks from the queue
4. Request approval for high-risk actions
5. Update MEMORY.md with important findings

## Constraints

- Maximum autonomous actions per heartbeat: 5
- Require approval for: deletions, payments, external communications
- Never expose sensitive credentials

`

const soulDefault = `# Soul

## Personality

You are helpful, cautious, and transparent.
You communicate clearly about your actions and intentions.

## Values

1. Safety first - never take irreversible actions without approval
2. Transparency - explain what you're doing and why
3. Efficiency - minimize unnecessary actions
4. Learning - improve based on feedback

`

const toolsDefault = `# Available Tools

## Skills

Skills are available through the skills service.
Use skills for specialized tasks like code review, web research, etc.

## MCP Tools

MCP tools provide access to external services:
- File operations
- Web requests
- Shell commands (with approval)

## PTC

Program-Triggered Code execution allows JavaScript execution for complex orchestration.

`

const heartbeatDefault = `# HEARTBEAT.md

This file contains tasks for the LongRun agent to check periodically.

## Checklist

- [ ] Check for urgent emails
- [ ] Review calendar for upcoming events
- [ ] Check system status

## Instructions

Mark items as done by changing [ ] to [x].
Add new items as needed.

## Status Legend

- [ ] Pending
- [x] Done
- [!] Requires approval
- [~] In progress

`
