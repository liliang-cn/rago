package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/liliang-cn/agent-go/pkg/domain"
	"github.com/liliang-cn/agent-go/pkg/prompt"
	"github.com/liliang-cn/agent-go/pkg/store"
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

	// Enhanced memory components
	scorer       *MemoryScorer
	classifier   *QueryClassifier
	noiseFilter  *NoiseFilter
	scopeWeights *ScopeWeightConfig

	// Hybrid search
	enableHybrid bool
	rrfK         float64

	// PageIndex-style navigator for file-based stores (no embedder)
	navigator        *IndexNavigator
	reflectThreshold int // auto-reflect after this many new facts

	mu sync.RWMutex
}

// Config holds configuration for the memory service
type Config struct {
	MinScore    float64 // Minimum relevance score for memory retrieval (default 0.7)
	MaxMemories int     // Maximum memories to inject (default 5)

	// Enhanced features
	ScoringConfig     *ScoringConfig
	ClassifierConfig  *ClassifierConfig
	NoiseFilterConfig *NoiseFilterConfig
	ScopeWeights      *ScopeWeightConfig

	// Hybrid search
	EnableHybrid bool
	RRFK         float64

	// Hindsight: auto-reflect after this many new facts (0 = disabled)
	ReflectThreshold int
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		MinScore:          0.01,
		MaxMemories:       5,
		ScoringConfig:     DefaultScoringConfig(),
		ClassifierConfig:  DefaultClassifierConfig(),
		NoiseFilterConfig: DefaultNoiseFilterConfig(),
		ScopeWeights:      DefaultScopeWeightConfig(),
		EnableHybrid:      false,
		RRFK:              60.0,
		ReflectThreshold:  5,
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

	svc := &Service{
		store:            memStore,
		entityMemory:     NewEntityMemory(memStore, embedder),
		llm:              llm,
		embedder:         embedder,
		promptManager:    prompt.NewManager(),
		minScore:         config.MinScore,
		maxMemories:      config.MaxMemories,
		scorer:           NewMemoryScorer(config.ScoringConfig),
		classifier:       NewQueryClassifier(config.ClassifierConfig),
		scopeWeights:     config.ScopeWeights,
		enableHybrid:     config.EnableHybrid,
		rrfK:             config.RRFK,
		reflectThreshold: config.ReflectThreshold,
	}

	if config.NoiseFilterConfig != nil {
		svc.noiseFilter = NewNoiseFilter(config.NoiseFilterConfig)
	}

	// Wire up IndexNavigator for file-based stores when no embedder is available
	if fileStore, ok := memStore.(*store.FileMemoryStore); ok && llm != nil {
		fileStore.WithLLM(llm)
		svc.navigator = NewIndexNavigator(fileStore, llm)
	}

	return svc
}

// SetPromptManager sets a custom prompt manager
func (s *Service) SetPromptManager(m *prompt.Manager) {
	s.promptManager = m
}

// SetShadowIndex sets the optional vector index for accelerating file-based stores
func (s *Service) SetShadowIndex(idx domain.MemoryStore) {
	s.shadowIndex = idx
}

// RetrieveAndInject searches relevant memories and formats them for LLM context.
// T6a: Routing logic —
//   - embedder available → vector search (+ optional hybrid BM25)
//   - navigator available → IndexNavigator (LLM reads _index/) — used alone or fused via RRF
//   - both → results are RRF-fused for highest recall
//
// Returns: formatted context string, scored memories, navigator reasoning (MemoryLogic), error
func (s *Service) RetrieveAndInject(ctx context.Context, query string, sessionID string) (string, []*domain.MemoryWithScore, error) {
	_, mems, _, err := s.RetrieveAndInjectWithLogic(ctx, query, sessionID)
	return s.formatMemories(mems), mems, err
}

// RetrieveAndInjectWithLogic is the full retrieval pipeline returning the navigator's
// reasoning string alongside the context and memories.
func (s *Service) RetrieveAndInjectWithLogic(ctx context.Context, query string, sessionID string) (string, []*domain.MemoryWithScore, string, error) {
	// 0. Adaptive retrieval - skip if query doesn't need memory
	if s.classifier != nil && !s.classifier.NeedsMemory(query) {
		return "", nil, "", nil
	}

	var allMemories []*domain.MemoryWithScore
	var memoryLogic string

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
	var vectorResults []*domain.MemoryWithScore
	if s.embedder != nil {
		vector, err := s.embedder.Embed(ctx, query)
		if err == nil {
			scopes := DefaultScopeChain(sessionID, "", "", "")
			vectorResults, _ = s.store.SearchByScope(ctx, vector, scopes.ToSlice(), s.maxMemories*2)

			if s.enableHybrid {
				textMems, _ := s.store.SearchByText(ctx, query, s.maxMemories)
				vectorResults = s.rrfFusion(vectorResults, textMems)
			}
		}
	}

	// 3. Navigator Search (PageIndex-style, for file stores) — captures reasoning
	var navResults []*domain.MemoryWithScore
	if s.navigator != nil {
		navResult, err := s.navigator.NavigateWithReason(ctx, query, s.maxMemories)
		if err == nil && navResult != nil {
			memoryLogic = navResult.Reasoning
			for i, m := range navResult.Memories {
				score := 1.0 - float64(i)*0.05
				navResults = append(navResults, &domain.MemoryWithScore{Memory: m, Score: score})
			}
		}
	}

	// 4. Fuse results
	switch {
	case len(vectorResults) > 0 && len(navResults) > 0:
		// Both available: RRF fusion
		allMemories = append(allMemories, s.rrfFusion(vectorResults, navResults)...)
	case len(vectorResults) > 0:
		allMemories = append(allMemories, vectorResults...)
	case len(navResults) > 0:
		allMemories = append(allMemories, navResults...)
	default:
		// Final fallback: plain list
		mems, _, _ := s.store.List(ctx, s.maxMemories, 0)
		for _, m := range mems {
			allMemories = append(allMemories, &domain.MemoryWithScore{Memory: m, Score: 0.5})
		}
	}

	// 5. Noise filtering
	if s.noiseFilter != nil {
		allMemories = s.noiseFilter.Filter(allMemories)
	}

	// 6. Scoring and ranking
	if s.scorer != nil {
		allMemories = s.scorer.ScoreAll(allMemories)
	} else {
		allMemories = s.mergeAndRank(allMemories)
	}

	// 7. Limit results
	if len(allMemories) > s.maxMemories {
		allMemories = allMemories[:s.maxMemories]
	}

	// Update access count
	for _, m := range allMemories {
		if m.ID != "" && !strings.HasPrefix(m.ID, "ent_") {
			_ = s.store.IncrementAccess(ctx, m.ID)
		}
	}

	if len(allMemories) == 0 {
		return "", nil, memoryLogic, nil
	}

	return s.formatMemories(allMemories), allMemories, memoryLogic, nil
}

// StoreIfWorthwhile decides what to store based on task completion.
// T6b: After storing new facts, checks if ReflectThreshold is reached and triggers async Reflect.
func (s *Service) StoreIfWorthwhile(ctx context.Context, req *domain.MemoryStoreRequest) error {
	if s.llm == nil {
		return nil
	}

	// 1. Triviality check: Don't bother LLM for very short/common greetings
	cleanGoal := strings.ToLower(strings.TrimSpace(req.TaskGoal))
	if len(cleanGoal) < 5 || cleanGoal == "hi" || cleanGoal == "hello" || cleanGoal == "hey" {
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

	result, err := s.llm.GenerateStructured(ctx, prompt, schema, &domain.GenerationOptions{Temperature: 0.1})
	if err != nil {
		return nil // Silent ignore for extraction errors to keep the chat clean
	}

	if result.Raw == "" || !result.Valid {
		return nil // No valid JSON found or schema mismatch - nothing worth storing
	}

	var summary domain.MemorySummaryResult
	if err := json.Unmarshal([]byte(result.Raw), &summary); err != nil {
		// Only log if there's actual content that failed to parse, 
		// but since we checked result.Valid, this is unlikely.
		return nil
	}

	if !summary.ShouldStore {
		return nil
	}

	newFactCount := 0
	for _, item := range summary.Memories {
		mem := &domain.Memory{
			ID:         uuid.New().String(),
			SessionID:  req.SessionID,
			Type:       item.Type,
			Content:    item.Content,
			Importance: item.Importance,
			SourceType: domain.MemorySourceInferred, // stored by agent inference
			CreatedAt:  time.Now(),
		}
		_ = s.Add(ctx, mem)
		if item.Type == domain.MemoryTypeFact {
			newFactCount++
		}
	}

	// Invalidate navigator cache since new memories were stored
	if s.navigator != nil {
		s.navigator.InvalidateCache()
	}

	// T6b: Async Reflect trigger — count facts for this session, fire if threshold reached
	if newFactCount > 0 && s.reflectThreshold > 0 {
		go s.maybeReflect(req.SessionID)
	}

	return nil
}

// maybeReflect counts active facts for sessionID and triggers Reflect if threshold is met.
// Only runs for FileMemoryStore (which has LLM-driven Reflect); other stores handle this internally.
// Runs in a goroutine; errors are silently swallowed.
func (s *Service) maybeReflect(sessionID string) {
	fileStore, ok := s.store.(*store.FileMemoryStore)
	if !ok {
		return
	}

	ctx := context.Background()
	facts, _, err := fileStore.List(ctx, 1000, 0)
	if err != nil {
		return
	}

	count := 0
	for _, m := range facts {
		if m.Type == domain.MemoryTypeFact &&
			(sessionID == "" || m.SessionID == sessionID) &&
			!store.IsStale(m) {
			count++
		}
	}

	if count >= s.reflectThreshold {
		_, _ = fileStore.Reflect(ctx, sessionID)
		// Invalidate cache again after reflection produced new observations
		if s.navigator != nil {
			s.navigator.InvalidateCache()
		}
	}
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
	var finalResults []*domain.MemoryWithScore
	for _, res := range results {
		if fresh, err := s.store.Get(ctx, res.ID); err == nil {
			res.Memory = fresh
			finalResults = append(finalResults, res)
		} else if s.shadowIndex == nil {
			// If we are NOT in hybrid mode, keep the result even if Get fails (unlikely)
			finalResults = append(finalResults, res)
		}
		// In hybrid mode, if Get fails, it means the file is missing, so we skip it.
	}

	return finalResults, nil
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

// rrfFusion performs Reciprocal Rank Fusion for hybrid search
// RRF score = sum(1 / (k + rank)) for each list
func (s *Service) rrfFusion(vector, text []*domain.MemoryWithScore) []*domain.MemoryWithScore {
	k := s.rrfK
	if k <= 0 {
		k = 60.0
	}

	// Calculate RRF scores
	scores := make(map[string]float64)
	memories := make(map[string]*domain.MemoryWithScore)

	// Vector results
	for rank, m := range vector {
		if m == nil || m.Memory == nil {
			continue
		}
		id := m.ID
		if id == "" {
			id = m.Content // Fallback to content as ID
		}
		scores[id] += 1.0 / (k + float64(rank+1))
		if _, exists := memories[id]; !exists {
			memories[id] = m
		}
	}

	// Text results
	for rank, m := range text {
		if m == nil || m.Memory == nil {
			continue
		}
		id := m.ID
		if id == "" {
			id = m.Content
		}
		scores[id] += 1.0 / (k + float64(rank+1))
		if _, exists := memories[id]; !exists {
			memories[id] = m
		}
	}

	// Build result list
	var results []*domain.MemoryWithScore
	for id, score := range scores {
		if m, exists := memories[id]; exists {
			// Create copy to avoid modifying original
			result := &domain.MemoryWithScore{
				Memory: m.Memory,
				Score:  score,
			}
			results = append(results, result)
		}
	}

	// Sort by RRF score
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Score < results[j].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results
}

func (s *Service) formatMemories(memories []*domain.MemoryWithScore) string {
	if len(memories) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("## Relevant Memory\n\n")
	for i, m := range memories {
		sb.WriteString(fmt.Sprintf("[%d] [%s]: %s\n\n", i+1, m.Type, m.Content))
	}
	return sb.String()
}

func (s *Service) buildSummaryPrompt(req *domain.MemoryStoreRequest) string {
	return fmt.Sprintf("SYSTEM: You are a background background memory process. ANALYZE the following interaction and EXTRACT new facts or user preferences. RETURN VALID JSON ONLY.\n\nInteraction:\nGoal: %s\nResult: %s\n\nTask: Extract potential memories according to the schema.", req.TaskGoal, req.TaskResult)
}

// MemoryEvolutionNode represents one step in a memory's evolution path.
type MemoryEvolutionNode struct {
	Memory     *domain.Memory         `json:"memory"`
	Children   []*MemoryEvolutionNode `json:"children,omitempty"`  // memories that supersede this one
	EvidenceOf *domain.Memory         `json:"evidence_of,omitempty"` // observation this is evidence for
}

// GetEvolution returns the full evolution graph rooted at the given memoryID.
// It traces both forward (SupersededBy chain) and backward (EvidenceIDs) to show
// how a raw Fact evolved through Reflect() into an Observation.
func (s *Service) GetEvolution(ctx context.Context, memoryID string) (*MemoryEvolutionNode, error) {
	root, err := s.store.Get(ctx, memoryID)
	if err != nil {
		return nil, fmt.Errorf("memory %s not found: %w", memoryID, err)
	}

	node := &MemoryEvolutionNode{Memory: root}

	// Forward: find any memory that supersedes this one
	if root.SupersededBy != "" {
		child, err := s.GetEvolution(ctx, root.SupersededBy)
		if err == nil {
			node.Children = []*MemoryEvolutionNode{child}
		}
	}

	// Backward: if this is an observation, find which observation (if any) it is evidence for
	// by scanning all observations for EvidenceIDs containing this ID
	if root.Type != domain.MemoryTypeObservation {
		all, _, err := s.store.List(ctx, 1000, 0)
		if err == nil {
			for _, m := range all {
				if m.Type == domain.MemoryTypeObservation {
					for _, eid := range m.EvidenceIDs {
						if eid == memoryID {
							node.EvidenceOf = m
							break
						}
					}
				}
				if node.EvidenceOf != nil {
					break
				}
			}
		}
	}

	return node, nil
}
