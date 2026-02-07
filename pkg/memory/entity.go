package memory

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/store"
)

// EntityMemory handles storage and retrieval of entity information
type EntityMemory struct {
	store    *store.MemoryStore
	embedder domain.Embedder
}

// NewEntityMemory creates a new entity memory manager
func NewEntityMemory(store *store.MemoryStore, embedder domain.Embedder) *EntityMemory {
	return &EntityMemory{
		store:    store,
		embedder: embedder,
	}
}

// SaveEntity saves or updates information about an entity
func (em *EntityMemory) SaveEntity(ctx context.Context, entity domain.Entity) error {
	// Create a memory item for the entity
	content := fmt.Sprintf("%s (%s): %s", entity.Name, entity.Type, entity.Description)
	
	vector, err := em.embedder.Embed(ctx, content)
	if err != nil {
		return fmt.Errorf("failed to embed entity content: %w", err)
	}

	memory := &store.Memory{
		ID:         uuid.New().String(),
		SessionID:  "entities", // Use a dedicated bank for entities
		Type:       "entity",
		Content:    content,
		Vector:     vector,
		Importance: 1.0, // Entities are important
		Metadata: map[string]interface{}{
			"entity_name": entity.Name,
			"entity_type": entity.Type,
			"source":      "entity_memory",
		},
	}

	return em.store.Store(ctx, memory)
}

// SearchEntities searches for entities relevant to a query
func (em *EntityMemory) SearchEntities(ctx context.Context, query string, limit int) ([]domain.Entity, error) {
	vector, err := em.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Search in the "entities" bank
	// Note: We need to use SearchBySession directly if possible, or filter results
	// Since SearchBySession is available in MemoryStore, we use it.
	results, err := em.store.SearchBySession(ctx, "entities", vector, limit)
	if err != nil {
		return nil, err
	}

	var entities []domain.Entity
	for _, res := range results {
		name, _ := res.Metadata["entity_name"].(string)
		typ, _ := res.Metadata["entity_type"].(string)
		
		// Extract description from content (format: "Name (Type): Description")
		parts := strings.SplitN(res.Content, ": ", 2)
		description := ""
		if len(parts) > 1 {
			description = parts[1]
		} else {
			description = res.Content
		}

		entities = append(entities, domain.Entity{
			Name:        name,
			Type:        typ,
			Description: description,
		})
	}

	return entities, nil
}
