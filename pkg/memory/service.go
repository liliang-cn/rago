package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/prompt"
)

// Service implements the MemoryService interface
type Service struct {
	store         domain.MemoryStore
	shadowIndex   domain.MemoryStore // Optional vector-based shadow index
	entityMemory  *EntityMemory
	llm           domain.Generator
	embedder      domain.Embedder
	promptManager *prompt.Manager
	minScore      float64
	maxMemories   int

	mu sync.RWMutex
}

// Config holds configuration for the memory service
type Config struct {
	MinScore    float64 // Minimum relevance score for memory retrieval (default 0.7)
	MaxMemories int     // Maximum memories to inject (default 5)
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		MinScore:    0.01,
		MaxMemories: 5,
	}
}

// NewService creates a new memory service
func NewService(
	memStore domain.MemoryStore,
	llm domain.Generator,
	embedder domain.Embedder,
	config *Config,
) *Service {
	if config == nil {
		config = DefaultConfig()
	}

	return &Service{
		store:         memStore,
		entityMemory:  NewEntityMemory(memStore, embedder),
		llm:           llm,
		embedder:      embedder,
		promptManager: prompt.NewManager(),
		minScore:      config.MinScore,
		maxMemories:   config.MaxMemories,
	}
}

// SetPromptManager sets a custom prompt manager
func (s *Service) SetPromptManager(m *prompt.Manager) {
	s.promptManager = m
}

// SetShadowIndex sets the optional vector index for accelerating file-based stores
func (s *Service) SetShadowIndex(idx domain.MemoryStore) {
	s.shadowIndex = idx
}

// RetrieveAndInject searches relevant memories and formats them for LLM context
func (s *Service) RetrieveAndInject(ctx context.Context, query string, sessionID string) (string, []*domain.MemoryWithScore, error) {
	var allMemories []*domain.MemoryWithScore

	// 1. Entity Search (if query is not empty)
	if s.entityMemory != nil && query != "" {
		entities, err := s.entityMemory.SearchEntities(ctx, query, 3)
		if err == nil {
			for _, ent := range entities {
				content := fmt.Sprintf("Entity: %s (%s) - %s", ent.Name, ent.Type, ent.Description)
				allMemories = append(allMemories, &domain.MemoryWithScore{
					Memory: &domain.Memory{
						ID:         "ent_" + ent.Name,
						Type:       domain.MemoryTypeFact,
						Content:    content,
						Importance: 1.0,
					},
					Score: 1.0,
				})
			}
		}
	}

	// 2. Vector Search (if embedder available)
	if s.embedder != nil {
		vector, err := s.embedder.Embed(ctx, query)
		if err == nil {
			// Session memories
			if sessionID != "" {
				mems, _ := s.store.SearchBySession(ctx, sessionID, vector, s.maxMemories/2)
				allMemories = append(allMemories, mems...)
			}
			// Global memories
			mems, _ := s.store.Search(ctx, vector, s.maxMemories, s.minScore)
			allMemories = append(allMemories, mems...)
		}
	} else {
		// Fallback to List (Memory Sitemap mode)
		mems, _, _ := s.store.List(ctx, s.maxMemories, 0)
		for _, m := range mems {
			allMemories = append(allMemories, &domain.MemoryWithScore{Memory: m, Score: 0.5})
		}
	}

	// 3. Merge and rank
	allMemories = s.mergeAndRank(allMemories)

	// Update access count
	for _, m := range allMemories {
		if m.ID != "" {
			_ = s.store.IncrementAccess(ctx, m.ID)
		}
	}

	if len(allMemories) == 0 {
		return "", nil, nil
	}

	return s.formatMemories(allMemories), allMemories, nil
}

// StoreIfWorthwhile decides what to store based on task completion
func (s *Service) StoreIfWorthwhile(ctx context.Context, req *domain.MemoryStoreRequest) error {
	if s.llm == nil {
		return nil
	}

	prompt := s.buildSummaryPrompt(req)
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"should_store": map[string]interface{}{"type": "boolean"},
			"memories": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"type":       map[string]interface{}{"type": "string", "enum": []string{"fact", "skill", "pattern", "context", "preference"}},
						"content":    map[string]interface{}{"type": "string"},
						"importance": map[string]interface{}{"type": "number"},
					},
					"required": []string{"type", "content", "importance"},
				},
			},
		},
		"required": []string{"should_store", "memories"},
	}

	result, err := s.llm.GenerateStructured(ctx, prompt, schema, &domain.GenerationOptions{Temperature: 0.3})
	if err != nil {
		return err
	}

	var summary domain.MemorySummaryResult
	if err := json.Unmarshal([]byte(result.Raw), &summary); err != nil {
		return err
	}

	if !summary.ShouldStore {
		return nil
	}

	for _, item := range summary.Memories {
		_ = s.Add(ctx, &domain.Memory{
			ID:         uuid.New().String(),
			SessionID:  req.SessionID,
			Type:       item.Type,
			Content:    item.Content,
			Importance: item.Importance,
			CreatedAt:  time.Now(),
		})
	}

	return nil
}

func (s *Service) Add(ctx context.Context, memory *domain.Memory) error {
	if memory.ID == "" {
		memory.ID = uuid.New().String()
	}
	if memory.CreatedAt.IsZero() {
		memory.CreatedAt = time.Now()
	}
	
	// Always generate embedding if possible
	if len(memory.Vector) == 0 && s.embedder != nil {
		vec, _ := s.embedder.Embed(ctx, memory.Content)
		memory.Vector = vec
	}

	// 1. Write to Primary Store (The Truth)
	err := s.store.Store(ctx, memory)
	if err != nil {
		return err
	}

	// 2. Write to Shadow Index (The Accelerator)
	if s.shadowIndex != nil {
		// Ensure it has vector before indexing
		if len(memory.Vector) > 0 {
			_ = s.shadowIndex.Store(ctx, memory)
		}
	}

	return nil
}

func (s *Service) Update(ctx context.Context, id string, newInfo string) error {
	if s.llm == nil {
		return fmt.Errorf("LLM required")
	}

	oldMem, err := s.store.Get(ctx, id)
	if err != nil {
		return err
	}

	prompt := fmt.Sprintf("Existing Memory:\n%s\n\nNew Info: %s\n\nMerge and provide updated content.", oldMem.Content, newInfo)
	updated, err := s.llm.Generate(ctx, prompt, nil)
	if err != nil {
		return err
	}

	oldMem.Content = strings.TrimSpace(updated)
	oldMem.UpdatedAt = time.Now()

	return s.store.Update(ctx, oldMem)
}

func (s *Service) Search(ctx context.Context, query string, topK int) ([]*domain.MemoryWithScore, error) {
	if topK <= 0 {
		topK = 10
	}

	// 1. Choose searching backend
	searchStore := s.store
	if s.shadowIndex != nil {
		searchStore = s.shadowIndex // Use vector-capable store for searching
	}

	// 2. Perform search
	if s.embedder == nil {
		mems, _, _ := s.store.List(ctx, topK, 0)
		var res []*domain.MemoryWithScore
		for _, m := range mems {
			res = append(res, &domain.MemoryWithScore{Memory: m, Score: 0.5})
		}
		return res, nil
	}

	vec, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, err
	}

	results, err := searchStore.Search(ctx, vec, topK, s.minScore)
	if err != nil {
		return nil, err
	}

	// 3. Truth Retrieval: For each found ID, fetch fresh content from Primary Store
	// This ensures that even if vector index is slightly stale, we get the human-edited content.
	for i, res := range results {
		if fresh, err := s.store.Get(ctx, res.ID); err == nil {
			results[i].Memory = fresh
		}
	}

	return results, nil
}

func (s *Service) Get(ctx context.Context, id string) (*domain.Memory, error) {
	return s.store.Get(ctx, id)
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]*domain.Memory, int, error) {
	return s.store.List(ctx, limit, offset)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

func (s *Service) ConfigureBank(ctx context.Context, sessionID string, config *domain.MemoryBankConfig) error {
	return s.store.ConfigureBank(ctx, sessionID, config)
}

func (s *Service) Reflect(ctx context.Context, sessionID string) (string, error) {
	return s.store.Reflect(ctx, sessionID)
}

func (s *Service) AddMentalModel(ctx context.Context, model *domain.MentalModel) error {
	return s.store.AddMentalModel(ctx, model)
}

// Helpers

func (s *Service) mergeAndRank(memories []*domain.MemoryWithScore) []*domain.MemoryWithScore {
	seen := make(map[string]bool)
	unique := make([]*domain.MemoryWithScore, 0)
	for _, m := range memories {
		if !seen[m.ID] {
			seen[m.ID] = true
			unique = append(unique, m)
		}
	}
	for i := 0; i < len(unique)-1; i++ {
		for j := i + 1; j < len(unique); j++ {
			if unique[i].Score < unique[j].Score {
				unique[i], unique[j] = unique[j], unique[i]
			}
		}
	}
	return unique
}

func (s *Service) formatMemories(memories []*domain.MemoryWithScore) string {
	var sb strings.Builder
	sb.WriteString("## Relevant Memory\n\n")
	for i, m := range memories {
		sb.WriteString(fmt.Sprintf("[%d] [%s]: %s\n\n", i+1, m.Type, m.Content))
	}
	return sb.String()
}

func (s *Service) buildSummaryPrompt(req *domain.MemoryStoreRequest) string {
	return fmt.Sprintf("Goal: %s\nResult: %s\nExtract memory.", req.TaskGoal, req.TaskResult)
}
