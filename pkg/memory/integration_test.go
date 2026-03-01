package memory

import (
	"context"
	"os"
	"testing"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryIntegration(t *testing.T) {
	ctx := context.Background()
	dbPath := "test_memory.db"
	defer os.Remove(dbPath)

	memStore, err := store.NewMemoryStore(dbPath)
	require.NoError(t, err)
	defer memStore.Close()

	// Use nil for LLM and Embedder for basic store/list test
	service := NewService(memStore, nil, nil, nil)

	// 1. Add a memory
	mem := &domain.Memory{
		Content:    "The secret ingredient is love.",
		Type:       domain.MemoryTypeFact,
		Importance: 0.9,
	}
	err = service.Add(ctx, mem)
	assert.NoError(t, err)

	// 2. List memories
	mems, total, err := service.List(ctx, 10, 0)
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, mems, 1)
	assert.Equal(t, "The secret ingredient is love.", mems[0].Content)

	// 3. Search (should fallback to List since embedder is nil)
	results, err := service.Search(ctx, "secret", 5)
	assert.NoError(t, err)
	assert.NotEmpty(t, results)
	assert.Equal(t, "The secret ingredient is love.", results[0].Content)

	// 4. Delete
	err = service.Delete(ctx, mems[0].ID)
	assert.NoError(t, err)

	// 5. Verify deletion
	mems, total, err = service.List(ctx, 10, 0)
	assert.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, mems)
}
