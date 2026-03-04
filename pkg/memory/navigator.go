package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/store"
)

const navigatorCacheTTL = 5 * time.Minute

type cacheEntry struct {
	ids       []string
	expiresAt time.Time
}

// IndexNavigator implements PageIndex-style memory retrieval for FileMemoryStore.
// Instead of vector similarity, it lets the LLM read the memory index (_index/)
// and reason about which memories are relevant to a query.
type IndexNavigator struct {
	fileStore *store.FileMemoryStore
	llm       domain.Generator

	mu    sync.RWMutex
	cache map[string]cacheEntry // query → selected IDs cache
}

// NewIndexNavigator creates a new IndexNavigator.
func NewIndexNavigator(fileStore *store.FileMemoryStore, llm domain.Generator) *IndexNavigator {
	return &IndexNavigator{
		fileStore: fileStore,
		llm:       llm,
		cache:     make(map[string]cacheEntry),
	}
}

// InvalidateCache clears all cached navigation results.
// Should be called when new memories are stored.
func (n *IndexNavigator) InvalidateCache() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.cache = make(map[string]cacheEntry)
}

// Navigate uses the LLM to read the memory index and select the most relevant memories.
// Results are cached by query string for navigatorCacheTTL.
func (n *IndexNavigator) Navigate(ctx context.Context, query string, topK int) ([]*domain.Memory, error) {
	if n.llm == nil || n.fileStore == nil {
		return nil, nil
	}

	// 1. Check cache
	ids, cached := n.cachedIDs(query)
	if !cached {
		// 2. Read the memory index (TOC) — observations first (higher-level)
		idx, err := n.fileStore.ReadIndex(ctx)
		if err != nil || idx == nil || len(idx.Entries) == 0 {
			return nil, err
		}

		// 3. Build a compact listing: observations first, then other types
		var sb strings.Builder
		sb.WriteString("Memory Index (observations first, then facts; format: [id] importance | summary):\n\n")
		// First pass: observations
		for _, e := range idx.Entries {
			if e.IsStale || e.Type != domain.MemoryTypeObservation {
				continue
			}
			sb.WriteString(fmt.Sprintf("- [%s][obs] %.2f | %s\n", e.ID, e.Importance, e.Summary))
		}
		// Second pass: everything else
		for _, e := range idx.Entries {
			if e.IsStale || e.Type == domain.MemoryTypeObservation {
				continue
			}
			sb.WriteString(fmt.Sprintf("- [%s][%s] %.2f | %s\n", e.ID, e.Type, e.Importance, e.Summary))
		}

		promptText := fmt.Sprintf(`You are a memory retrieval assistant. Given a user query and a memory index, select the IDs of the most relevant memories.

User query: %s

%s
Instructions:
- Select at most %d memory IDs that are most relevant to the query.
- Prefer observations over raw facts when observations cover the topic.
- Use logical reasoning, not just keyword matching.
- If no memories are relevant, return an empty list.

Output valid JSON only:
{"ids": ["id1", "id2"]}`, query, sb.String(), topK)

		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"ids": map[string]interface{}{
					"type":  "array",
					"items": map[string]interface{}{"type": "string"},
				},
			},
			"required": []string{"ids"},
		}

		result, err := n.llm.GenerateStructured(ctx, promptText, schema, &domain.GenerationOptions{Temperature: 0.1})
		if err != nil {
			return nil, fmt.Errorf("navigator LLM call failed: %w", err)
		}

		// 4. Parse selected IDs
		var selected struct {
			IDs []string `json:"ids"`
		}
		raw := strings.TrimSpace(result.Raw)
		if strings.HasPrefix(raw, "```") {
			lines := strings.SplitN(raw, "\n", 2)
			if len(lines) == 2 {
				raw = lines[1]
			}
			raw = strings.TrimSuffix(raw, "```")
			raw = strings.TrimSpace(raw)
		}
		if err := json.Unmarshal([]byte(raw), &selected); err != nil {
			return nil, fmt.Errorf("failed to parse navigator result: %w", err)
		}
		ids = selected.IDs
		n.setCachedIDs(query, ids)
	}

	// 5. Fetch full memory content for each selected ID
	var memories []*domain.Memory
	for _, id := range ids {
		m, err := n.fileStore.Get(ctx, id)
		if err != nil {
			continue
		}
		memories = append(memories, m)
	}

	return memories, nil
}

func (n *IndexNavigator) cachedIDs(query string) ([]string, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	entry, ok := n.cache[query]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.ids, true
}

func (n *IndexNavigator) setCachedIDs(query string, ids []string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.cache[query] = cacheEntry{ids: ids, expiresAt: time.Now().Add(navigatorCacheTTL)}
}
