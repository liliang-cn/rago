package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"gopkg.in/yaml.v3"
)

// FileMemoryStore implements domain.MemoryStore using Markdown files with YAML frontmatter
type FileMemoryStore struct {
	baseDir    string
	mu         sync.RWMutex
	indexDirty bool // true when index needs rebuild
	llm        domain.Generator
}

// WithLLM injects an LLM generator used for Reflect() consolidation.
func (s *FileMemoryStore) WithLLM(llm domain.Generator) {
	s.llm = llm
}

// MemoryIndex is the parsed representation of _index.md
type MemoryIndex struct {
	Total     int                  `yaml:"total"`
	UpdatedAt time.Time            `yaml:"updated_at"`
	Entries   []MemoryIndexEntry
}

// MemoryIndexEntry is one line in the index
type MemoryIndexEntry struct {
	ID         string
	Type       domain.MemoryType
	Importance float64
	Summary    string // first 60 chars of content
	IsStale    bool
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

	// Hindsight: temporal, evidence, and provenance fields
	EvidenceIDs     []string                  `yaml:"evidence_ids,omitempty"`
	Confidence      float64                   `yaml:"confidence,omitempty"`
	ValidFrom       time.Time                 `yaml:"valid_from,omitempty"`
	ValidTo         *time.Time                `yaml:"valid_to,omitempty"`
	SupersededBy    string                    `yaml:"superseded_by,omitempty"`
	SourceType      domain.MemorySourceType   `yaml:"source_type,omitempty"`
	Conflicting     bool                      `yaml:"conflicting,omitempty"`
	RevisionHistory []domain.MemoryRevision   `yaml:"revision_history,omitempty"`
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
		ID:              memory.ID,
		Type:            string(memory.Type),
		Importance:      memory.Importance,
		SessionID:       memory.SessionID,
		AccessCount:     memory.AccessCount,
		LastAccessed:    memory.LastAccessed,
		CreatedAt:       memory.CreatedAt,
		UpdatedAt:       time.Now(),
		Metadata:        memory.Metadata,
		EvidenceIDs:     memory.EvidenceIDs,
		Confidence:      memory.Confidence,
		ValidFrom:       memory.ValidFrom,
		ValidTo:         memory.ValidTo,
		SupersededBy:    memory.SupersededBy,
		SourceType:      memory.SourceType,
		Conflicting:     memory.Conflicting,
		RevisionHistory: memory.RevisionHistory,
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

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	s.indexDirty = true
	return nil
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
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, cat := range []string{"streams", "entities"} {
		_ = os.Remove(filepath.Join(s.baseDir, cat, id+".md"))
	}
	s.indexDirty = true
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
	if s.llm == nil {
		return "LLM not configured; skipping reflection.", nil
	}

	// 1. Collect active (non-stale) facts and existing observations for this session
	all, _, err := s.List(ctx, 1000, 0)
	if err != nil {
		return "", err
	}

	var facts []*domain.Memory
	var existingObs []*domain.Memory
	for _, m := range all {
		if sessionID != "" && m.SessionID != sessionID {
			continue
		}
		if IsStale(m) {
			continue
		}
		switch m.Type {
		case domain.MemoryTypeFact:
			facts = append(facts, m)
		case domain.MemoryTypeObservation:
			existingObs = append(existingObs, m)
		}
	}

	if len(facts) < 3 {
		return "Not enough facts to consolidate (need at least 3).", nil
	}

	// 2. Build sets of already-used evidence IDs to avoid double-counting
	usedIDs := make(map[string]bool)
	for _, obs := range existingObs {
		for _, id := range obs.EvidenceIDs {
			usedIDs[id] = true
		}
	}

	// 3. Collect only facts not yet captured in an observation
	var newFacts []*domain.Memory
	for _, f := range facts {
		if !usedIDs[f.ID] {
			newFacts = append(newFacts, f)
		}
	}
	if len(newFacts) < 2 {
		return "All facts are already covered by existing observations.", nil
	}

	// 4. Build prompt with existing observations for recursive merging + conflict detection
	var factLines strings.Builder
	for _, f := range newFacts {
		factLines.WriteString(fmt.Sprintf("- [%s] %s\n", f.ID, f.Content))
	}

	var obsLines strings.Builder
	if len(existingObs) > 0 {
		obsLines.WriteString("\nExisting observations (do not duplicate; update or merge if a new fact fits):\n")
		for _, o := range existingObs {
			obsLines.WriteString(fmt.Sprintf("- [%s] %s\n", o.ID, o.Content))
		}
	}

	promptText := fmt.Sprintf(`You are a memory consolidation engine with strict anti-hallucination rules.

New facts (not yet covered by any observation):
%s
%s
Rules:
1. ONLY use information explicitly present in the facts above. Do NOT invent or infer beyond what is stated.
2. An observation must cite at least 2 fact IDs as evidence.
3. If two facts CONTRADICT each other, set "conflicting": true and include both in evidence_ids.
4. If a new fact extends an existing observation, output an "update_obs_id" field with the existing observation's ID to supersede it.
5. Do not duplicate existing observations unless you are merging/updating them.
6. Confidence: 0.9+ only if facts are highly consistent; lower if partial or ambiguous.

Output valid JSON only:
{
  "observations": [
    {
      "content": "Single sentence synthesizing the facts.",
      "confidence": 0.85,
      "evidence_ids": ["id1", "id2"],
      "conflicting": false,
      "update_obs_id": ""
    }
  ]
}`, factLines.String(), obsLines.String())

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"observations": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"content":       map[string]interface{}{"type": "string"},
						"confidence":    map[string]interface{}{"type": "number"},
						"evidence_ids":  map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
						"conflicting":   map[string]interface{}{"type": "boolean"},
						"update_obs_id": map[string]interface{}{"type": "string"},
					},
					"required": []string{"content", "confidence", "evidence_ids"},
				},
			},
		},
		"required": []string{"observations"},
	}

	result, err := s.llm.GenerateStructured(ctx, promptText, schema, &domain.GenerationOptions{Temperature: 0.2})
	if err != nil {
		return "", fmt.Errorf("LLM reflection failed: %w", err)
	}

	// 5. Parse and store observations
	type obsItem struct {
		Content      string   `json:"content"`
		Confidence   float64  `json:"confidence"`
		EvidenceIDs  []string `json:"evidence_ids"`
		Conflicting  bool     `json:"conflicting"`
		UpdateObsID  string   `json:"update_obs_id"`
	}
	type reflectResult struct {
		Observations []obsItem `json:"observations"`
	}
	var parsed reflectResult
	if err := parseJSON(result.Raw, &parsed); err != nil {
		return "", fmt.Errorf("failed to parse reflection result: %w", err)
	}

	created, updated := 0, 0
	for _, obs := range parsed.Observations {
		if obs.Content == "" || len(obs.EvidenceIDs) < 2 {
			continue
		}

		newID := newUUID()
		obsMemory := &domain.Memory{
			ID:          newID,
			SessionID:   sessionID,
			Type:        domain.MemoryTypeObservation,
			Content:     obs.Content,
			Importance:  obs.Confidence,
			Confidence:  obs.Confidence,
			EvidenceIDs: obs.EvidenceIDs,
			Conflicting: obs.Conflicting,
			SourceType:  domain.MemorySourceConsolidated,
			ValidFrom:   time.Now(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := s.Store(ctx, obsMemory); err != nil {
			continue
		}

		// If this supersedes an existing observation, mark it stale
		if obs.UpdateObsID != "" {
			_ = s.MarkStale(ctx, obs.UpdateObsID, newID)
			updated++
		} else {
			created++
		}
	}

	return fmt.Sprintf("Reflection complete: %d new observations, %d updated from %d facts.", created, updated, len(newFacts)), nil
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

	// Migration compatibility: fill defaults for fields added in the cognitive layer.
	// Old YAML files won't have these fields; zero values are ambiguous, so we back-fill.
	if fm.SourceType == "" {
		fm.SourceType = domain.MemorySourceUserInput // pre-existing memories are treated as user-stated
	}
	if fm.Confidence == 0 && fm.Type != "observation" {
		fm.Confidence = 1.0 // facts/preferences with no recorded confidence are assumed authoritative
	}

	return &domain.Memory{
		ID:              fm.ID,
		SessionID:       fm.SessionID,
		Type:            domain.MemoryType(fm.Type),
		Content:         strings.TrimSpace(parts[2]),
		Importance:      fm.Importance,
		AccessCount:     fm.AccessCount,
		LastAccessed:    fm.LastAccessed,
		Metadata:        fm.Metadata,
		CreatedAt:       fm.CreatedAt,
		UpdatedAt:       fm.UpdatedAt,
		EvidenceIDs:     fm.EvidenceIDs,
		Confidence:      fm.Confidence,
		ValidFrom:       fm.ValidFrom,
		ValidTo:         fm.ValidTo,
		SupersededBy:    fm.SupersededBy,
		SourceType:      fm.SourceType,
		Conflicting:     fm.Conflicting,
		RevisionHistory: fm.RevisionHistory,
	}, nil
}

// MarkStale marks a memory as stale (superseded by a newer memory).
// Sets ValidTo to now, records the superseding ID, and appends a revision entry.
func (s *FileMemoryStore) MarkStale(ctx context.Context, id string, supersededByID string) error {
	m, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	now := time.Now()
	m.ValidTo = &now
	m.SupersededBy = supersededByID
	m.UpdatedAt = now
	m.RevisionHistory = append(m.RevisionHistory, domain.MemoryRevision{
		At:      now,
		By:      "reflect",
		Summary: fmt.Sprintf("superseded by %s", supersededByID),
	})
	return s.Store(ctx, m)
}

// IsStale returns true if the memory has been superseded.
func IsStale(m *domain.Memory) bool {
	return m.ValidTo != nil || m.SupersededBy != ""
}

// indexDir returns the path to the _index/ directory
func (s *FileMemoryStore) indexDir() string {
	return filepath.Join(s.baseDir, "_index")
}

// indexFilePath returns the per-type index file path, e.g. _index/observations.md
func (s *FileMemoryStore) indexFilePath(t domain.MemoryType) string {
	return filepath.Join(s.indexDir(), string(t)+"s.md")
}

// RebuildIndex forces a full rebuild of all per-type index files.
// Useful after manual edits, migrations, or corruption recovery.
func (s *FileMemoryStore) RebuildIndex(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.rebuildIndex(ctx); err != nil {
		return err
	}
	s.indexDirty = false
	return nil
}

// ReadIndex returns the merged memory index across all type files.
// Rebuilds if dirty or missing.
func (s *FileMemoryStore) ReadIndex(ctx context.Context) (*MemoryIndex, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.indexDirty {
		if err := s.rebuildIndex(ctx); err != nil {
			return nil, err
		}
		s.indexDirty = false
	}

	return s.readIndexFiles()
}

// readIndexFiles reads all per-type index files and merges them.
// Caller must hold s.mu (at least RLock).
func (s *FileMemoryStore) readIndexFiles() (*MemoryIndex, error) {
	idx := &MemoryIndex{}
	typeOrder := []domain.MemoryType{
		domain.MemoryTypeObservation,
		domain.MemoryTypeFact,
		domain.MemoryTypePreference,
		domain.MemoryTypeSkill,
		domain.MemoryTypePattern,
		domain.MemoryTypeContext,
	}
	for _, t := range typeOrder {
		data, err := os.ReadFile(s.indexFilePath(t))
		if err != nil {
			continue // file may not exist yet
		}
		partial, err := parseMemoryIndex(data, t)
		if err != nil {
			continue
		}
		idx.Entries = append(idx.Entries, partial.Entries...)
		idx.Total += partial.Total
		if partial.UpdatedAt.After(idx.UpdatedAt) {
			idx.UpdatedAt = partial.UpdatedAt
		}
	}
	return idx, nil
}

// rebuildIndex scans all memory files and rewrites the per-type index files.
// Caller must hold s.mu.Lock().
func (s *FileMemoryStore) rebuildIndex(ctx context.Context) error {
	if err := os.MkdirAll(s.indexDir(), 0755); err != nil {
		return err
	}

	// Collect all entries grouped by type
	groups := map[domain.MemoryType][]MemoryIndexEntry{}
	for _, cat := range []string{"streams", "entities"} {
		files, _ := filepath.Glob(filepath.Join(s.baseDir, cat, "*.md"))
		for _, f := range files {
			m, err := s.readFile(f)
			if err != nil {
				continue
			}
			groups[m.Type] = append(groups[m.Type], MemoryIndexEntry{
				ID:         m.ID,
				Type:       m.Type,
				Importance: m.Importance,
				Summary:    truncate(m.Content, 60),
				IsStale:    IsStale(m),
			})
		}
	}

	typeOrder := []domain.MemoryType{
		domain.MemoryTypeObservation,
		domain.MemoryTypeFact,
		domain.MemoryTypePreference,
		domain.MemoryTypeSkill,
		domain.MemoryTypePattern,
		domain.MemoryTypeContext,
	}

	for _, t := range typeOrder {
		entries := groups[t]
		// Sort: non-stale first, then by importance desc
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].IsStale != entries[j].IsStale {
				return !entries[i].IsStale
			}
			return entries[i].Importance > entries[j].Importance
		})

		if err := writeIndexFile(s.indexFilePath(t), t, entries); err != nil {
			return err
		}
	}
	return nil
}

// writeIndexFile writes one per-type index file atomically (tmp + rename).
func writeIndexFile(path string, t domain.MemoryType, entries []MemoryIndexEntry) error {
	type indexFM struct {
		Type      string    `yaml:"type"`
		Total     int       `yaml:"total"`
		UpdatedAt time.Time `yaml:"updated_at"`
	}
	fm, _ := yaml.Marshal(indexFM{
		Type:      string(t),
		Total:     len(entries),
		UpdatedAt: time.Now(),
	})

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(string(fm))
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("# %ss (%d)\n\n", strings.Title(string(t)), len(entries)))

	for _, e := range entries {
		staleTag := ""
		if e.IsStale {
			staleTag = " ~~[stale]~~"
		}
		sb.WriteString(fmt.Sprintf("- [%s] %.2f | %s%s\n", e.ID, e.Importance, e.Summary, staleTag))
	}

	// Atomic write: write to a temp file in the same directory, then rename.
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".index-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.WriteString(sb.String()); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

// parseMemoryIndex parses a per-type index file.
func parseMemoryIndex(data []byte, t domain.MemoryType) (*MemoryIndex, error) {
	content := string(data)
	if !strings.HasPrefix(content, "---") {
		return &MemoryIndex{}, nil
	}
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return &MemoryIndex{}, nil
	}

	type indexFM struct {
		Total     int       `yaml:"total"`
		UpdatedAt time.Time `yaml:"updated_at"`
	}
	var fm indexFM
	_ = yaml.Unmarshal([]byte(parts[1]), &fm)

	idx := &MemoryIndex{
		Total:     fm.Total,
		UpdatedAt: fm.UpdatedAt,
	}

	for _, line := range strings.Split(parts[2], "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- [") {
			continue
		}
		idEnd := strings.Index(line, "]")
		if idEnd < 0 {
			continue
		}
		id := line[3:idEnd]
		rest := strings.TrimSpace(line[idEnd+1:])

		var importance float64
		var summary string
		if parts2 := strings.SplitN(rest, "|", 2); len(parts2) == 2 {
			fmt.Sscanf(strings.TrimSpace(parts2[0]), "%f", &importance)
			summary = strings.TrimSpace(parts2[1])
			summary = strings.ReplaceAll(summary, " ~~[stale]~~", "")
		}

		idx.Entries = append(idx.Entries, MemoryIndexEntry{
			ID:         id,
			Type:       t,
			Importance: importance,
			Summary:    summary,
			IsStale:    strings.Contains(line, "~~[stale]~~"),
		})
	}

	return idx, nil
}

// truncate shortens s to at most n runes, appending "…" if truncated.
func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	runes := []rune(s)
	return string(runes[:n]) + "…"
}

// newUUID generates a new UUID string.
func newUUID() string {
return uuid.New().String()
}

// parseJSON unmarshals raw JSON into v, stripping markdown code fences if present.
func parseJSON(raw string, v interface{}) error {
raw = strings.TrimSpace(raw)
if strings.HasPrefix(raw, "```") {
lines := strings.SplitN(raw, "\n", 2)
if len(lines) == 2 {
raw = lines[1]
}
raw = strings.TrimSuffix(raw, "```")
raw = strings.TrimSpace(raw)
}
return json.Unmarshal([]byte(raw), v)
}
