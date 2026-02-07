package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/store"
)

// Service implements the MemoryService interface
type Service struct {
	store        *store.MemoryStore
	entityMemory *EntityMemory
	llm          domain.Generator
	embedder     domain.Embedder
	minScore     float64
	maxMemories  int

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
		MinScore:    0.01, // Very low threshold - cosine similarity can be low for short texts
		MaxMemories: 5,
	}
}

// NewService creates a new memory service
func NewService(
	memStore *store.MemoryStore,
	llm domain.Generator,
	embedder domain.Embedder,
	config *Config,
) *Service {
	if config == nil {
		config = DefaultConfig()
	}

	return &Service{
		store:        memStore,
		entityMemory: NewEntityMemory(memStore, embedder),
		llm:          llm,
		embedder:     embedder,
		minScore:     config.MinScore,
		maxMemories:  config.MaxMemories,
	}
}

// RetrieveAndInject searches relevant memories and formats them for LLM context
func (s *Service) RetrieveAndInject(ctx context.Context, query string, sessionID string) (string, []*domain.MemoryWithScore, error) {
	// 1. Generate query vector
	vector, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return "", nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// 2. Retrieve: session memories + global memories + entities
	var allMemories []*domain.MemoryWithScore

	// Search entities
	if s.entityMemory != nil {
		entities, err := s.entityMemory.SearchEntities(ctx, query, 3)
		if err == nil {
			for _, ent := range entities {
				content := fmt.Sprintf("Entity: %s (%s) - %s", ent.Name, ent.Type, ent.Description)
				allMemories = append(allMemories, &domain.MemoryWithScore{
					Memory: &domain.Memory{
						Type:       "entity",
						Content:    content,
						Importance: 1.0,
					},
					Score: 1.0, // High priority
				})
			}
		}
	}

	// Search session-specific memories
	if sessionID != "" {
		sessionMems, err := s.store.SearchBySession(ctx, sessionID, vector, s.maxMemories/2)
		if err == nil {
			allMemories = append(allMemories, toDomainMemories(sessionMems)...)
		}
	}

	// Search global memories (in "default" bank)
	globalMems, err := s.store.Search(ctx, vector, s.maxMemories-len(allMemories), s.minScore)
	if err == nil {
		allMemories = append(allMemories, toDomainMemories(globalMems)...)
	}

	// 3. Merge and rank by score
	allMemories = s.mergeAndRank(allMemories)

	// 4. Update access statistics
	for _, m := range allMemories {
		_ = s.store.IncrementAccess(ctx, m.ID)
	}

	// 5. Format for LLM context
	if len(allMemories) == 0 {
		return "", allMemories, nil
	}

	contextStr := s.formatMemories(allMemories)
	return contextStr, allMemories, nil
}

// StoreIfWorthwhile analyzes task completion and decides what to store
func (s *Service) StoreIfWorthwhile(ctx context.Context, req *domain.MemoryStoreRequest) error {
	if s.llm == nil {
		return nil // LLM required for auto-storage
	}

	// 1. Build LLM prompt for memory extraction
	prompt := s.buildSummaryPrompt(req)

	// 2. Use structured generation to extract memories
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"should_store": map[string]interface{}{
				"type": "boolean",
			},
			"reasoning": map[string]interface{}{
				"type": "string",
			},
			"memories": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"type": map[string]interface{}{
							"type": "string",
							"enum": []string{"fact", "skill", "pattern", "context", "preference"},
						},
						"content": map[string]interface{}{
							"type": "string",
						},
						"importance": map[string]interface{}{
							"type": "number",
						},
						"tags": map[string]interface{}{
							"type": "array",
							"items": map[string]string{
								"type": "string",
							},
						},
						"entities": map[string]interface{}{
							"type": "array",
							"items": map[string]string{
								"type": "string",
							},
						},
					},
					"required": []string{"type", "content", "importance"},
				},
			},
		},
		"required": []string{"should_store", "memories"},
	}

	result, err := s.llm.GenerateStructured(ctx, prompt, schema, &domain.GenerationOptions{
		Temperature: 0.3,
		MaxTokens:   1000,
	})
	if err != nil {
		return fmt.Errorf("failed to generate memory summary: %w", err)
	}

	// 3. Parse result
	var summary domain.MemorySummaryResult
	if err := json.Unmarshal([]byte(result.Raw), &summary); err != nil {
		log.Printf("[Memory] Error parsing memory summary: %v", err)
		return fmt.Errorf("failed to parse memory summary: %w", err)
	}

	if !summary.ShouldStore || len(summary.Memories) == 0 {
		return nil // Nothing to store
	}

	// 4. Store each memory
	for _, item := range summary.Memories {
		// Handle entities specifically
		if len(item.Entities) > 0 && s.entityMemory != nil {
			for _, entityName := range item.Entities {
				// Basic entity creation - ideally LLM would provide more detail
				// We create a stub entity here, future retrievals might enrich it
				entity := domain.Entity{
					Name:        entityName,
					Type:        "unknown", // LLM extraction schema needs update to support type
					Description: "Extracted from: " + item.Content,
				}
				// Simple heuristic for type
				if strings.Contains(strings.ToLower(item.Content), "user") {
					entity.Type = "person"
				} else if strings.Contains(strings.ToLower(item.Content), "project") {
					entity.Type = "project"
				}
				
				_ = s.entityMemory.SaveEntity(ctx, entity)
			}
		}

		// Generate embedding for the memory content
		vector, err := s.embedder.Embed(ctx, item.Content)
		if err != nil {
			continue
		}

		memory := &store.Memory{
			ID:         uuid.New().String(),
			SessionID:  req.SessionID,
			Type:       string(item.Type),
			Content:    item.Content,
			Vector:     vector,
			Importance: item.Importance,
			Metadata: map[string]interface{}{
				"tags":     item.Tags.Strings(),
				"entities": item.Entities.Strings(),
				"source":   "task_completion",
			},
		}

		if err := s.store.Store(ctx, memory); err != nil {
			continue
		}
	}

	return nil
}

// Add directly adds a memory
func (s *Service) Add(ctx context.Context, memory *domain.Memory) error {
	if memory.ID == "" {
		memory.ID = uuid.New().String()
	}
	now := time.Now()
	if memory.CreatedAt.IsZero() {
		memory.CreatedAt = now
	}
	if memory.UpdatedAt.IsZero() {
		memory.UpdatedAt = now
	}

	// Generate embedding if not provided
	if len(memory.Vector) == 0 && s.embedder != nil {
		vector, err := s.embedder.Embed(ctx, memory.Content)
		if err != nil {
			return fmt.Errorf("failed to embed memory: %w", err)
		}
		memory.Vector = vector
	}

	// Convert to store memory
	storeMem := &store.Memory{
		ID:           memory.ID,
		SessionID:    memory.SessionID,
		Type:         string(memory.Type),
		Content:      memory.Content,
		Vector:       memory.Vector,
		Importance:   memory.Importance,
		AccessCount:  memory.AccessCount,
		LastAccessed: memory.LastAccessed,
		Metadata:     memory.Metadata,
		CreatedAt:    memory.CreatedAt,
		UpdatedAt:    memory.UpdatedAt,
	}

	return s.store.Store(ctx, storeMem)
}

// Update updates a memory's content (LLM-driven)
func (s *Service) Update(ctx context.Context, id string, content string) error {
	if s.llm == nil {
		return fmt.Errorf("LLM service required for memory updates")
	}

	// 1. Get existing memory
	storeMem, err := s.store.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get memory: %w", err)
	}

	// 2. Let LLM update the memory based on instruction
	prompt := fmt.Sprintf(`Update the following memory based on the instruction.

Current Memory:
Type: %s
Content: %s
Importance: %.2f

Update Instruction: %s

Return JSON with: content (string), importance (number if changed).
`, storeMem.Type, storeMem.Content, storeMem.Importance, content)

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"content": map[string]interface{}{
				"type": "string",
			},
			"importance": map[string]interface{}{
				"type": "number",
			},
		},
		"required": []string{"content"},
	}

	result, err := s.llm.GenerateStructured(ctx, prompt, schema, &domain.GenerationOptions{
		Temperature: 0.2,
		MaxTokens:   500,
	})
	if err != nil {
		return fmt.Errorf("failed to generate update: %w", err)
	}

	var update struct {
		Content    string  `json:"content"`
		Importance float64 `json:"importance"`
	}
	if err := json.Unmarshal([]byte(result.Raw), &update); err != nil {
		return fmt.Errorf("failed to parse update: %w", err)
	}

	// 3. Update memory
	oldContent := storeMem.Content
	storeMem.Content = update.Content
	if update.Importance > 0 {
		storeMem.Importance = update.Importance
	}

	// Re-embed if content changed
	if update.Content != oldContent && s.embedder != nil {
		if vector, err := s.embedder.Embed(ctx, update.Content); err == nil {
			storeMem.Vector = vector
		}
	}

	return s.store.Update(ctx, storeMem)
}

// Search searches memories by query
func (s *Service) Search(ctx context.Context, query string, topK int) ([]*domain.MemoryWithScore, error) {
	if topK <= 0 {
		topK = 10
	}

	if s.embedder == nil {
		// Fallback to text search
		return s.searchByText(ctx, query, topK)
	}

	vector, err := s.embedder.Embed(ctx, query)
	if err != nil {
		// Fallback to text search
		return s.searchByText(ctx, query, topK)
	}

	results, err := s.store.Search(ctx, vector, topK, s.minScore)
	if err != nil {
		return nil, err
	}

	return toDomainMemories(results), nil
}

// Get retrieves a memory by ID
func (s *Service) Get(ctx context.Context, id string) (*domain.Memory, error) {
	storeMem, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return toDomainMemory(storeMem), nil
}

// List lists memories
func (s *Service) List(ctx context.Context, limit, offset int) ([]*domain.Memory, int, error) {
	storeMems, total, err := s.store.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	memories := make([]*domain.Memory, len(storeMems))
	for i, sm := range storeMems {
		memories[i] = toDomainMemory(sm)
	}

	return memories, total, nil
}

// Delete removes a memory
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// ConfigureBank sets mission and disposition for a memory bank
func (s *Service) ConfigureBank(ctx context.Context, sessionID string, config *domain.MemoryBankConfig) error {
	return s.store.ConfigureBank(ctx, sessionID, config)
}

// Reflect triggers knowledge consolidation for a bank
func (s *Service) Reflect(ctx context.Context, sessionID string) (string, error) {
	return s.store.Reflect(ctx, sessionID)
}

// AddMentalModel adds a curated mental model
func (s *Service) AddMentalModel(ctx context.Context, model *domain.MentalModel) error {
	return s.store.AddMentalModel(ctx, model)
}

// Helper methods

func (s *Service) mergeAndRank(memories []*domain.MemoryWithScore) []*domain.MemoryWithScore {
	// Remove duplicates by ID
	seen := make(map[string]bool)
	unique := make([]*domain.MemoryWithScore, 0)

	for _, m := range memories {
		if !seen[m.ID] {
			seen[m.ID] = true
			unique = append(unique, m)
		}
	}

	// Sort by score (highest first)
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
		sb.WriteString(fmt.Sprintf("[%d] [%s] (score: %.2f, importance: %.2f)\n%s\n\n",
			i+1, m.Type, m.Score, m.Importance, m.Content))
	}
	return sb.String()
}

func (s *Service) buildSummaryPrompt(req *domain.MemoryStoreRequest) string {
	execLog := ""
	if req.ExecutionLog != "" {
		execLog = fmt.Sprintf("\nExecution Log:\n%s\n", req.ExecutionLog)
	}

	return fmt.Sprintf(`Analyze the completed task and extract any information worth storing in long-term memory.

Task Goal: %s

Task Result: %s
%s
Guidelines:
- Extract facts, skills, patterns, or preferences that could be useful for future tasks
- Only store information that is likely to be referenced again
- Importance score (0-1): >0.8 for critical info, >0.5 for useful info, <0.5 for trivial
- Tags: short keywords for categorization
- Entities: named entities (people, projects, concepts)

Return JSON with: should_store (boolean), reasoning (string), and memories array.
`, req.TaskGoal, req.TaskResult, execLog)
}

func (s *Service) searchByText(ctx context.Context, query string, topK int) ([]*domain.MemoryWithScore, error) {
	// Simple text-based search fallback
	allMems, _, err := s.store.List(ctx, 100, 0)
	if err != nil {
		return nil, err
	}

	var results []*domain.MemoryWithScore
	queryLower := strings.ToLower(query)

	for _, mem := range allMems {
		contentLower := strings.ToLower(mem.Content)
		if strings.Contains(contentLower, queryLower) {
			// Simple relevance score based on occurrence
			score := 0.5
			if strings.HasPrefix(contentLower, queryLower) {
				score = 0.8
			}

			domainMem := toDomainMemory(mem)
			results = append(results, &domain.MemoryWithScore{
				Memory: domainMem,
				Score:  score,
			})

			if len(results) >= topK {
				break
			}
		}
	}

	return results, nil
}

// Conversion functions

func toDomainMemory(sm *store.Memory) *domain.Memory {
	if sm == nil {
		return nil
	}
	return &domain.Memory{
		ID:           sm.ID,
		SessionID:    sm.SessionID,
		Type:         domain.MemoryType(sm.Type),
		Content:      sm.Content,
		Vector:       sm.Vector,
		Importance:   sm.Importance,
		AccessCount:  sm.AccessCount,
		LastAccessed: sm.LastAccessed,
		Metadata:     sm.Metadata,
		CreatedAt:    sm.CreatedAt,
		UpdatedAt:    sm.UpdatedAt,
	}
}

func toDomainMemories(sm []*store.MemoryWithScore) []*domain.MemoryWithScore {
	result := make([]*domain.MemoryWithScore, len(sm))
	for i, m := range sm {
		result[i] = &domain.MemoryWithScore{
			Memory: toDomainMemory(m.Memory),
			Score:  m.Score,
		}
	}
	return result
}
