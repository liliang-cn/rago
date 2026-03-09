package memory

import (
	"context"
	"testing"

	"github.com/liliang-cn/agent-go/pkg/domain"
	"github.com/liliang-cn/agent-go/pkg/store"
)

func TestFileMemoryIntegration(t *testing.T) {
	ctx := context.Background()

	memStore, err := store.NewFileMemoryStore(t.TempDir())
	if err != nil {
		t.Fatalf("new file memory store failed: %v", err)
	}

	service := NewService(memStore, nil, nil, nil)

	mem := &domain.Memory{
		ID:         "pref-1",
		Content:    "Alice likes tea in the morning.",
		Type:       domain.MemoryTypePreference,
		Importance: 0.9,
		SessionID:  "session-file-1",
	}
	if err := service.Add(ctx, mem); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	mems, total, err := service.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if total != 1 || len(mems) != 1 {
		t.Fatalf("unexpected list result: total=%d len=%d", total, len(mems))
	}

	results, err := service.Search(ctx, "tea", 5)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) != 1 || results[0].Content != mem.Content {
		t.Fatalf("unexpected search results: %+v", results)
	}

	formatted, recalled, err := service.RetrieveAndInject(ctx, "what does Alice like to drink?", "session-file-1")
	if err != nil {
		t.Fatalf("retrieve and inject failed: %v", err)
	}
	if formatted == "" || len(recalled) == 0 {
		t.Fatalf("expected retrieved memory context, got formatted=%q recalled=%d", formatted, len(recalled))
	}
	if recalled[0].Content != mem.Content {
		t.Fatalf("unexpected recalled memory: %+v", recalled[0])
	}

	if err := service.Delete(ctx, mem.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	mems, total, err = service.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("list after delete failed: %v", err)
	}
	if total != 0 || len(mems) != 0 {
		t.Fatalf("expected empty list after delete, got total=%d len=%d", total, len(mems))
	}
}
