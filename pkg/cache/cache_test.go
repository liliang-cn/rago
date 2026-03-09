package cache

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestMemoryCacheBasicLifecycle(t *testing.T) {
	ctx := context.Background()
	c := NewMemoryCache(2, time.Minute)

	if got, ok := c.Get(ctx, "missing"); ok || got != nil {
		t.Fatalf("expected missing key")
	}

	if err := c.Set(ctx, "a", "value-a", 0); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	got, ok := c.Get(ctx, "a")
	if !ok || got != "value-a" {
		t.Fatalf("unexpected cache hit: %v %v", got, ok)
	}

	stats := c.Stats()
	if stats.Hits != 1 || stats.Misses != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	if err := c.Delete(ctx, "a"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, ok := c.Get(ctx, "a"); ok {
		t.Fatal("expected deleted key to miss")
	}
}

func TestMemoryCacheExpirationAndEviction(t *testing.T) {
	ctx := context.Background()
	c := NewMemoryCache(2, 10*time.Millisecond)

	if err := c.Set(ctx, "a", "value-a", 5*time.Millisecond); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	time.Sleep(15 * time.Millisecond)
	if _, ok := c.Get(ctx, "a"); ok {
		t.Fatal("expected expired key to miss")
	}

	_ = c.Set(ctx, "a", "value-a", time.Minute)
	_ = c.Set(ctx, "b", "value-b", time.Minute)
	if _, ok := c.Get(ctx, "a"); !ok {
		t.Fatal("expected a to exist")
	}
	_ = c.Set(ctx, "c", "value-c", time.Minute)

	if _, ok := c.Get(ctx, "b"); ok {
		t.Fatal("expected least recently used key b to be evicted")
	}
	if stats := c.Stats(); stats.Evictions != 1 {
		t.Fatalf("expected one eviction, got %+v", stats)
	}
}

func TestQueryVectorLLMAndChunkCaches(t *testing.T) {
	ctx := context.Background()

	qc := NewQueryCache(10, time.Minute)
	if err := qc.SetQueryResult(ctx, "q", map[string]interface{}{"a": 1}, "result", 0); err != nil {
		t.Fatalf("query cache set failed: %v", err)
	}
	if got, ok := qc.GetQueryResult(ctx, "q", map[string]interface{}{"a": 1}); !ok || got != "result" {
		t.Fatalf("unexpected query cache hit: %v %v", got, ok)
	}
	keyA := qc.generateQueryKey("q", map[string]interface{}{"a": 1, "b": 2})
	keyB := qc.generateQueryKey("q", map[string]interface{}{"b": 2, "a": 1})
	if keyA != keyB {
		t.Fatalf("expected stable query key across map order")
	}

	vc := NewVectorCache(10, time.Minute)
	vector := []float64{1.0, 2.0}
	if err := vc.SetVector(ctx, "text", vector, 0); err != nil {
		t.Fatalf("vector cache set failed: %v", err)
	}
	if got, ok := vc.GetVector(ctx, "text"); !ok || len(got) != 2 || got[1] != 2.0 {
		t.Fatalf("unexpected vector cache hit: %v %v", got, ok)
	}
	_ = vc.cache.Set(ctx, vc.generateVectorKey("bad"), "not-a-vector", time.Minute)
	if got, ok := vc.GetVector(ctx, "bad"); ok || got != nil {
		t.Fatalf("expected typed vector lookup to reject bad payload")
	}

	lc := NewLLMCache(10, time.Minute)
	if err := lc.SetResponse(ctx, "prompt", "model", 0.2, "answer", 0); err != nil {
		t.Fatalf("llm cache set failed: %v", err)
	}
	if got, ok := lc.GetResponse(ctx, "prompt", "model", 0.2); !ok || got != "answer" {
		t.Fatalf("unexpected llm cache hit: %s %v", got, ok)
	}
	_ = lc.cache.Set(ctx, lc.generateResponseKey("bad", "model", 0.2), 123, time.Minute)
	if got, ok := lc.GetResponse(ctx, "bad", "model", 0.2); ok || got != "" {
		t.Fatalf("expected typed llm lookup to reject bad payload")
	}

	cc := NewChunkCache(10, time.Minute)
	chunks := []string{"a", "b"}
	if err := cc.SetChunks(ctx, "doc", chunks, 0); err != nil {
		t.Fatalf("chunk cache set failed: %v", err)
	}
	if got, ok := cc.GetChunks(ctx, "doc"); !ok || len(got) != 2 {
		t.Fatalf("unexpected chunk cache hit: %v %v", got, ok)
	}
	_ = cc.cache.Set(ctx, "chunks:bad", "oops", time.Minute)
	if got, ok := cc.GetChunks(ctx, "bad"); ok || got != nil {
		t.Fatalf("expected typed chunk lookup to reject bad payload")
	}
}

func TestCacheManager(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultCacheConfig()
	cfg.EnableVectorCache = false

	manager := NewCacheManager(cfg)
	if manager.QueryCache() == nil || manager.LLMCache() == nil || manager.ChunkCache() == nil {
		t.Fatal("expected enabled caches to be initialized")
	}
	if manager.VectorCache() != nil {
		t.Fatal("expected disabled vector cache to be nil")
	}

	_ = manager.QueryCache().SetQueryResult(ctx, "query", nil, "result", 0)
	_ = manager.LLMCache().SetResponse(ctx, "prompt", "model", 0.1, "answer", 0)
	_ = manager.ChunkCache().SetChunks(ctx, "doc", []string{"chunk"}, 0)

	stats := manager.GetStats()
	if len(stats) != 3 {
		t.Fatalf("expected stats for enabled caches only, got %v", stats)
	}

	if err := manager.ClearAll(ctx); err != nil {
		t.Fatalf("clear all failed: %v", err)
	}
	if _, ok := manager.QueryCache().GetQueryResult(ctx, "query", nil); ok {
		t.Fatal("expected query cache to be cleared")
	}
}

func TestFileCachePersistsAcrossRestart(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	fc, err := NewFileCache(dir, 10, time.Minute)
	if err != nil {
		t.Fatalf("new file cache failed: %v", err)
	}

	if err := fc.Set(ctx, "string", "value", 0); err != nil {
		t.Fatalf("set string failed: %v", err)
	}
	if err := fc.Set(ctx, "vector", []float64{1.5, 2.5}, 0); err != nil {
		t.Fatalf("set vector failed: %v", err)
	}
	if err := fc.Set(ctx, "chunks", []string{"a", "b"}, 0); err != nil {
		t.Fatalf("set chunks failed: %v", err)
	}

	reloaded, err := NewFileCache(dir, 10, time.Minute)
	if err != nil {
		t.Fatalf("reload file cache failed: %v", err)
	}

	if got, ok := reloaded.Get(ctx, "string"); !ok || got != "value" {
		t.Fatalf("unexpected reloaded string: %v %v", got, ok)
	}
	if got, ok := reloaded.Get(ctx, "vector"); !ok {
		t.Fatal("expected reloaded vector")
	} else if vector, ok := got.([]float64); !ok || len(vector) != 2 || vector[1] != 2.5 {
		t.Fatalf("unexpected reloaded vector: %#v", got)
	}
	if got, ok := reloaded.Get(ctx, "chunks"); !ok {
		t.Fatal("expected reloaded chunks")
	} else if chunks, ok := got.([]string); !ok || len(chunks) != 2 || chunks[0] != "a" {
		t.Fatalf("unexpected reloaded chunks: %#v", got)
	}
}

func TestFileCacheExpirationEvictionAndClear(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	fc, err := NewFileCache(dir, 2, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("new file cache failed: %v", err)
	}

	if err := fc.Set(ctx, "expire", "soon", 5*time.Millisecond); err != nil {
		t.Fatalf("set expire failed: %v", err)
	}
	time.Sleep(15 * time.Millisecond)
	if _, ok := fc.Get(ctx, "expire"); ok {
		t.Fatal("expected expired key to miss")
	}

	if err := fc.Set(ctx, "a", "value-a", time.Minute); err != nil {
		t.Fatalf("set a failed: %v", err)
	}
	if err := fc.Set(ctx, "b", "value-b", time.Minute); err != nil {
		t.Fatalf("set b failed: %v", err)
	}
	if _, ok := fc.Get(ctx, "a"); !ok {
		t.Fatal("expected a to exist")
	}
	if err := fc.Set(ctx, "c", "value-c", time.Minute); err != nil {
		t.Fatalf("set c failed: %v", err)
	}
	if _, ok := fc.Get(ctx, "b"); ok {
		t.Fatal("expected b to be evicted")
	}

	if err := fc.Clear(ctx); err != nil {
		t.Fatalf("clear failed: %v", err)
	}
	if fc.Size() != 0 {
		t.Fatalf("expected empty file cache, got size %d", fc.Size())
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatalf("glob failed: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected cache directory to be empty, got %v", files)
	}
}

func TestFileCacheManager(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultCacheConfig()
	dir := t.TempDir()

	manager, err := NewFileCacheManager(dir, cfg)
	if err != nil {
		t.Fatalf("new file cache manager failed: %v", err)
	}

	if err := manager.QueryCache().SetQueryResult(ctx, "query", map[string]interface{}{"lang": "go"}, "result", 0); err != nil {
		t.Fatalf("query set failed: %v", err)
	}
	if err := manager.VectorCache().SetVector(ctx, "text", []float64{3, 4}, 0); err != nil {
		t.Fatalf("vector set failed: %v", err)
	}
	if err := manager.LLMCache().SetResponse(ctx, "prompt", "model", 0.1, "answer", 0); err != nil {
		t.Fatalf("llm set failed: %v", err)
	}
	if err := manager.ChunkCache().SetChunks(ctx, "doc", []string{"c1", "c2"}, 0); err != nil {
		t.Fatalf("chunk set failed: %v", err)
	}

	reloaded, err := NewFileCacheManager(dir, cfg)
	if err != nil {
		t.Fatalf("reload file cache manager failed: %v", err)
	}

	if got, ok := reloaded.QueryCache().GetQueryResult(ctx, "query", map[string]interface{}{"lang": "go"}); !ok || got != "result" {
		t.Fatalf("unexpected query result after reload: %v %v", got, ok)
	}
	if got, ok := reloaded.VectorCache().GetVector(ctx, "text"); !ok || len(got) != 2 || got[0] != 3 {
		t.Fatalf("unexpected vector result after reload: %v %v", got, ok)
	}
	if got, ok := reloaded.LLMCache().GetResponse(ctx, "prompt", "model", 0.1); !ok || got != "answer" {
		t.Fatalf("unexpected llm result after reload: %q %v", got, ok)
	}
	if got, ok := reloaded.ChunkCache().GetChunks(ctx, "doc"); !ok || len(got) != 2 || got[1] != "c2" {
		t.Fatalf("unexpected chunk result after reload: %v %v", got, ok)
	}
}

func TestNewCacheManagerWithStore(t *testing.T) {
	cfg := DefaultCacheConfig()

	memManager, err := NewCacheManagerWithStore(StoreTypeMemory, t.TempDir(), cfg)
	if err != nil {
		t.Fatalf("memory manager failed: %v", err)
	}
	if memManager.QueryCache() == nil {
		t.Fatal("expected memory-backed query cache")
	}

	fileDir := t.TempDir()
	fileManager, err := NewCacheManagerWithStore(StoreTypeFile, fileDir, cfg)
	if err != nil {
		t.Fatalf("file manager failed: %v", err)
	}
	if fileManager.NamespaceCache("query") == nil {
		t.Fatal("expected file-backed query namespace")
	}

	if _, err := NewCacheManagerWithStore("bad", fileDir, cfg); err == nil {
		t.Fatal("expected bad store type to fail")
	}
}

func TestLRUList(t *testing.T) {
	lru := newLRUList()
	lru.addToFront("a")
	lru.addToFront("b")
	lru.moveToFront("a")

	if removed := lru.removeLast(); removed != "b" {
		t.Fatalf("expected b to be removed last, got %s", removed)
	}

	lru.remove("a")
	if lru.size() != 0 {
		t.Fatalf("expected empty list, got size %d", lru.size())
	}
	if removed := lru.removeLast(); removed != "" {
		t.Fatalf("expected empty removeLast result, got %s", removed)
	}
}
