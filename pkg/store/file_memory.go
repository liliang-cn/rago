package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"gopkg.in/yaml.v3"
)

// FileMemoryStore implements domain.MemoryStore using Markdown files with YAML frontmatter
type FileMemoryStore struct {
	baseDir string
	mu      sync.RWMutex
}

// MemoryFrontmatter represents the YAML header in the markdown file
type MemoryFrontmatter struct {
	ID           string                 `yaml:"id"`
	Type         string                 `yaml:"type"`
	Importance   float64                `yaml:"importance"`
	SessionID    string                 `yaml:"session_id,omitempty"`
	Tags         []string               `yaml:"tags,omitempty"`
	AccessCount  int                    `yaml:"access_count,omitempty"`
	LastAccessed time.Time              `yaml:"last_accessed,omitempty"`
	CreatedAt    time.Time              `yaml:"created_at"`
	UpdatedAt    time.Time              `yaml:"updated_at"`
	Metadata     map[string]interface{} `yaml:"metadata,omitempty"`
}

// NewFileMemoryStore creates a new markdown-based memory store
func NewFileMemoryStore(baseDir string) (*FileMemoryStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create memory directory: %w", err)
	}

	// Create subdirectories for OpenClaw/Mem0 style
	for _, dir := range []string{"streams", "entities"} {
		if err := os.MkdirAll(filepath.Join(baseDir, dir), 0755); err != nil {
			return nil, err
		}
	}

	return &FileMemoryStore{baseDir: baseDir}, nil
}

// Store saves a memory as a markdown file
func (s *FileMemoryStore) Store(ctx context.Context, memory *domain.Memory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Determine category (stream or entity)
	category := "entities"
	if memory.Type == domain.MemoryTypeContext {
		category = "streams"
	}

	// Use ID as filename, ensure it's safe
	fileName := fmt.Sprintf("%s.md", memory.ID)
	path := filepath.Join(s.baseDir, category, fileName)

	fm := MemoryFrontmatter{
		ID:           memory.ID,
		Type:         string(memory.Type),
		Importance:   memory.Importance,
		SessionID:    memory.SessionID,
		AccessCount:  memory.AccessCount,
		LastAccessed: memory.LastAccessed,
		CreatedAt:    memory.CreatedAt,
		UpdatedAt:    time.Now(),
		Metadata:     memory.Metadata,
	}

	// Extract tags from metadata if they exist
	if memory.Metadata != nil {
		if t, ok := memory.Metadata["tags"].([]string); ok {
			fm.Tags = t
		}
	}

	frontmatter, err := yaml.Marshal(fm)
	if err != nil {
		return err
	}

	// Double check content isn't empty
	content := fmt.Sprintf("---\n%s---\n\n%s", string(frontmatter), memory.Content)

	return os.WriteFile(path, []byte(content), 0644)
}

// Search performs a simplified keyword search (since we're embedding-free)
func (s *FileMemoryStore) Search(ctx context.Context, vector []float64, topK int, minScore float64) ([]*domain.MemoryWithScore, error) {
	// In the file-based store, we prioritize "Intent-based reading".
	// We'll return the most important/recent memories.
	all, _, err := s.List(ctx, 100, 0)
	if err != nil {
		return nil, err
	}

	var results []*domain.MemoryWithScore
	for _, m := range all {
		results = append(results, &domain.MemoryWithScore{
			Memory: m,
			Score:  1.0,
		})
	}

	if len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

func (s *FileMemoryStore) SearchBySession(ctx context.Context, sessionID string, vector []float64, topK int) ([]*domain.MemoryWithScore, error) {
	all, _, err := s.List(ctx, 1000, 0)
	if err != nil {
		return nil, err
	}

	var results []*domain.MemoryWithScore
	for _, m := range all {
		if m.SessionID == sessionID {
			results = append(results, &domain.MemoryWithScore{
				Memory: m,
				Score:  1.0,
			})
		}
	}
	return results, nil
}

// SearchByScope searches memories within specific scopes
func (s *FileMemoryStore) SearchByScope(ctx context.Context, vector []float64, scopes []domain.MemoryScope, topK int) ([]*domain.MemoryWithScore, error) {
	all, _, err := s.List(ctx, 1000, 0)
	if err != nil {
		return nil, err
	}

	// Build a map of scope bank IDs for quick lookup
	scopeMap := make(map[string]bool)
	for _, scope := range scopes {
		bankID := scopeToBankIDFile(scope)
		scopeMap[bankID] = true
	}

	var results []*domain.MemoryWithScore
	for _, m := range all {
		// Check if memory's session/bank matches any scope
		bankID := m.SessionID
		if bankID == "" {
			bankID = "global"
		}
		if scopeMap[bankID] {
			results = append(results, &domain.MemoryWithScore{
				Memory: m,
				Score:  1.0,
			})
		}
	}

	if len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

// StoreWithScope stores a memory with a specific scope
func (s *FileMemoryStore) StoreWithScope(ctx context.Context, memory *domain.Memory, scope domain.MemoryScope) error {
	// Set the session ID based on scope
	memory.SessionID = scopeToBankIDFile(scope)
	return s.Store(ctx, memory)
}

// SearchByText performs full-text search on file-based memories
func (s *FileMemoryStore) SearchByText(ctx context.Context, query string, topK int) ([]*domain.MemoryWithScore, error) {
	all, _, err := s.List(ctx, 1000, 0)
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)
	var results []*domain.MemoryWithScore

	for _, m := range all {
		contentLower := strings.ToLower(m.Content)
		if strings.Contains(contentLower, queryLower) {
			// Simple scoring based on substring match
			score := 0.5
			if strings.Contains(contentLower, queryLower) {
				score = 0.8
			}
			results = append(results, &domain.MemoryWithScore{
				Memory: m,
				Score:  score,
			})
		}
	}

	if len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

// scopeToBankIDFile converts MemoryScope to bank ID for file store
func scopeToBankIDFile(scope domain.MemoryScope) string {
	if scope.Type == domain.MemoryScopeGlobal {
		return "global"
	}
	if scope.ID == "" {
		return string(scope.Type)
	}
	return fmt.Sprintf("%s:%s", scope.Type, scope.ID)
}

func (s *FileMemoryStore) Get(ctx context.Context, id string) (*domain.Memory, error) {
	for _, cat := range []string{"streams", "entities"} {
		path := filepath.Join(s.baseDir, cat, id+".md")
		if _, err := os.Stat(path); err == nil {
			return s.readFile(path)
		}
	}
	return nil, fmt.Errorf("memory %s not found", id)
}

func (s *FileMemoryStore) Update(ctx context.Context, memory *domain.Memory) error {
	return s.Store(ctx, memory)
}

func (s *FileMemoryStore) IncrementAccess(ctx context.Context, id string) error {
	m, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	m.AccessCount++
	m.LastAccessed = time.Now()
	return s.Store(ctx, m)
}

func (s *FileMemoryStore) GetByType(ctx context.Context, memoryType domain.MemoryType, limit int) ([]*domain.Memory, error) {
	all, _, _ := s.List(ctx, 1000, 0)
	var filtered []*domain.Memory
	for _, m := range all {
		if m.Type == memoryType {
			filtered = append(filtered, m)
		}
		if len(filtered) >= limit {
			break
		}
	}
	return filtered, nil
}

func (s *FileMemoryStore) List(ctx context.Context, limit, offset int) ([]*domain.Memory, int, error) {
	var all []*domain.Memory

	for _, cat := range []string{"streams", "entities"} {
		files, _ := filepath.Glob(filepath.Join(s.baseDir, cat, "*.md"))
		for _, f := range files {
			m, err := s.readFile(f)
			if err == nil {
				all = append(all, m)
			}
		}
	}

	total := len(all)
	if offset >= total {
		return nil, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return all[offset:end], total, nil
}

func (s *FileMemoryStore) Delete(ctx context.Context, id string) error {
	for _, cat := range []string{"streams", "entities"} {
		_ = os.Remove(filepath.Join(s.baseDir, cat, id+".md"))
	}
	return nil
}

func (s *FileMemoryStore) DeleteBySession(ctx context.Context, sessionID string) error {
	all, _, _ := s.List(ctx, 1000, 0)
	for _, m := range all {
		if m.SessionID == sessionID {
			_ = s.Delete(ctx, m.ID)
		}
	}
	return nil
}

func (s *FileMemoryStore) InitSchema(ctx context.Context) error {
	return nil
}

func (s *FileMemoryStore) ConfigureBank(ctx context.Context, sessionID string, config *domain.MemoryBankConfig) error {
	return nil
}

func (s *FileMemoryStore) Reflect(ctx context.Context, sessionID string) (string, error) {
	return "Manual reflection is required for file-based memories.", nil
}

func (s *FileMemoryStore) AddMentalModel(ctx context.Context, model *domain.MentalModel) error {
	m := &domain.Memory{
		ID:         model.ID,
		Type:       domain.MemoryTypePattern,
		Content:    fmt.Sprintf("Mental Model: %s\n%s", model.Name, model.Content),
		Importance: 1.0,
		CreatedAt:  time.Now(),
	}
	return s.Store(ctx, m)
}

func (s *FileMemoryStore) readFile(path string) (*domain.Memory, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	contentStr := string(data)
	if !strings.HasPrefix(contentStr, "---") {
		return nil, fmt.Errorf("invalid markdown format in %s", path)
	}

	parts := strings.SplitN(contentStr, "---", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("failed to split frontmatter from content in %s", path)
	}

	var fm MemoryFrontmatter
	if err := yaml.Unmarshal([]byte(parts[1]), &fm); err != nil {
		return nil, err
	}

	return &domain.Memory{
		ID:           fm.ID,
		SessionID:    fm.SessionID,
		Type:         domain.MemoryType(fm.Type),
		Content:      strings.TrimSpace(parts[2]),
		Importance:   fm.Importance,
		AccessCount:  fm.AccessCount,
		LastAccessed: fm.LastAccessed,
		Metadata:     fm.Metadata,
		CreatedAt:    fm.CreatedAt,
		UpdatedAt:    fm.UpdatedAt,
	}, nil
}
