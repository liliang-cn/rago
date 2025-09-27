package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// CacheEntry represents a cached item
type CacheEntry struct {
	Key        string      `json:"key"`
	Value      interface{} `json:"value"`
	CreatedAt  time.Time   `json:"created_at"`
	ExpiresAt  time.Time   `json:"expires_at"`
	AccessedAt time.Time   `json:"accessed_at"`
	HitCount   int64       `json:"hit_count"`
}

// Cache interface for different cache implementations
type Cache interface {
	Get(ctx context.Context, key string) (interface{}, bool)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
	Size() int
	Stats() CacheStats
}

// CacheStats provides cache statistics
type CacheStats struct {
	Hits       int64     `json:"hits"`
	Misses     int64     `json:"misses"`
	Evictions  int64     `json:"evictions"`
	Size       int       `json:"size"`
	MaxSize    int       `json:"max_size"`
	CreatedAt  time.Time `json:"created_at"`
	LastClear  time.Time `json:"last_clear"`
}

// MemoryCache implements an in-memory cache with LRU eviction
type MemoryCache struct {
	mu         sync.RWMutex
	entries    map[string]*CacheEntry
	lru        *lruList
	maxSize    int
	stats      CacheStats
	defaultTTL time.Duration
}

// NewMemoryCache creates a new memory cache
func NewMemoryCache(maxSize int, defaultTTL time.Duration) *MemoryCache {
	return &MemoryCache{
		entries:    make(map[string]*CacheEntry),
		lru:        newLRUList(),
		maxSize:    maxSize,
		defaultTTL: defaultTTL,
		stats: CacheStats{
			MaxSize:   maxSize,
			CreatedAt: time.Now(),
		},
	}
}

// Get retrieves a value from cache
func (mc *MemoryCache) Get(ctx context.Context, key string) (interface{}, bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	entry, exists := mc.entries[key]
	if !exists {
		mc.stats.Misses++
		return nil, false
	}
	
	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		mc.removeEntry(key)
		mc.stats.Misses++
		return nil, false
	}
	
	// Update access info
	entry.AccessedAt = time.Now()
	entry.HitCount++
	mc.stats.Hits++
	
	// Move to front in LRU
	mc.lru.moveToFront(key)
	
	return entry.Value, true
}

// Set stores a value in cache
func (mc *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	if ttl <= 0 {
		ttl = mc.defaultTTL
	}
	
	now := time.Now()
	entry := &CacheEntry{
		Key:        key,
		Value:      value,
		CreatedAt:  now,
		ExpiresAt:  now.Add(ttl),
		AccessedAt: now,
		HitCount:   0,
	}
	
	// Check if we need to evict
	if len(mc.entries) >= mc.maxSize && mc.entries[key] == nil {
		// Evict least recently used
		evictKey := mc.lru.removeLast()
		if evictKey != "" {
			delete(mc.entries, evictKey)
			mc.stats.Evictions++
		}
	}
	
	mc.entries[key] = entry
	mc.lru.addToFront(key)
	mc.stats.Size = len(mc.entries)
	
	return nil
}

// Delete removes a value from cache
func (mc *MemoryCache) Delete(ctx context.Context, key string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.removeEntry(key)
	return nil
}

// Clear removes all entries from cache
func (mc *MemoryCache) Clear(ctx context.Context) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.entries = make(map[string]*CacheEntry)
	mc.lru = newLRUList()
	mc.stats.Size = 0
	mc.stats.LastClear = time.Now()
	
	return nil
}

// Size returns the number of entries in cache
func (mc *MemoryCache) Size() int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return len(mc.entries)
}

// Stats returns cache statistics
func (mc *MemoryCache) Stats() CacheStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	mc.stats.Size = len(mc.entries)
	return mc.stats
}

// removeEntry removes an entry without locking (must be called with lock held)
func (mc *MemoryCache) removeEntry(key string) {
	if _, exists := mc.entries[key]; exists {
		delete(mc.entries, key)
		mc.lru.remove(key)
		mc.stats.Size = len(mc.entries)
	}
}

// QueryCache specialized cache for RAG queries
type QueryCache struct {
	cache Cache
}

// NewQueryCache creates a new query cache
func NewQueryCache(maxSize int, ttl time.Duration) *QueryCache {
	return &QueryCache{
		cache: NewMemoryCache(maxSize, ttl),
	}
}

// GetQueryResult retrieves a cached query result
func (qc *QueryCache) GetQueryResult(ctx context.Context, query string, filters map[string]interface{}) (interface{}, bool) {
	key := qc.generateQueryKey(query, filters)
	return qc.cache.Get(ctx, key)
}

// SetQueryResult stores a query result in cache
func (qc *QueryCache) SetQueryResult(ctx context.Context, query string, filters map[string]interface{}, result interface{}, ttl time.Duration) error {
	key := qc.generateQueryKey(query, filters)
	return qc.cache.Set(ctx, key, result, ttl)
}

// generateQueryKey generates a unique key for a query
func (qc *QueryCache) generateQueryKey(query string, filters map[string]interface{}) string {
	data := map[string]interface{}{
		"query":   query,
		"filters": filters,
	}
	
	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return "query:" + hex.EncodeToString(hash[:])
}

// VectorCache specialized cache for vector embeddings
type VectorCache struct {
	cache Cache
}

// NewVectorCache creates a new vector cache
func NewVectorCache(maxSize int, ttl time.Duration) *VectorCache {
	return &VectorCache{
		cache: NewMemoryCache(maxSize, ttl),
	}
}

// GetVector retrieves a cached vector
func (vc *VectorCache) GetVector(ctx context.Context, text string) ([]float64, bool) {
	key := vc.generateVectorKey(text)
	value, exists := vc.cache.Get(ctx, key)
	if !exists {
		return nil, false
	}
	
	vector, ok := value.([]float64)
	if !ok {
		return nil, false
	}
	
	return vector, true
}

// SetVector stores a vector in cache
func (vc *VectorCache) SetVector(ctx context.Context, text string, vector []float64, ttl time.Duration) error {
	key := vc.generateVectorKey(text)
	return vc.cache.Set(ctx, key, vector, ttl)
}

// generateVectorKey generates a unique key for a text
func (vc *VectorCache) generateVectorKey(text string) string {
	hash := sha256.Sum256([]byte(text))
	return "vector:" + hex.EncodeToString(hash[:])
}

// LLMCache specialized cache for LLM responses
type LLMCache struct {
	cache Cache
}

// NewLLMCache creates a new LLM response cache
func NewLLMCache(maxSize int, ttl time.Duration) *LLMCache {
	return &LLMCache{
		cache: NewMemoryCache(maxSize, ttl),
	}
}

// GetResponse retrieves a cached LLM response
func (lc *LLMCache) GetResponse(ctx context.Context, prompt string, model string, temperature float64) (string, bool) {
	key := lc.generateResponseKey(prompt, model, temperature)
	value, exists := lc.cache.Get(ctx, key)
	if !exists {
		return "", false
	}
	
	response, ok := value.(string)
	if !ok {
		return "", false
	}
	
	return response, true
}

// SetResponse stores an LLM response in cache
func (lc *LLMCache) SetResponse(ctx context.Context, prompt string, model string, temperature float64, response string, ttl time.Duration) error {
	key := lc.generateResponseKey(prompt, model, temperature)
	return lc.cache.Set(ctx, key, response, ttl)
}

// generateResponseKey generates a unique key for an LLM request
func (lc *LLMCache) generateResponseKey(prompt string, model string, temperature float64) string {
	data := fmt.Sprintf("%s:%s:%.2f", prompt, model, temperature)
	hash := sha256.Sum256([]byte(data))
	return "llm:" + hex.EncodeToString(hash[:])
}

// ChunkCache specialized cache for document chunks
type ChunkCache struct {
	cache Cache
}

// NewChunkCache creates a new chunk cache
func NewChunkCache(maxSize int, ttl time.Duration) *ChunkCache {
	return &ChunkCache{
		cache: NewMemoryCache(maxSize, ttl),
	}
}

// GetChunks retrieves cached chunks for a document
func (cc *ChunkCache) GetChunks(ctx context.Context, documentID string) ([]string, bool) {
	key := "chunks:" + documentID
	value, exists := cc.cache.Get(ctx, key)
	if !exists {
		return nil, false
	}
	
	chunks, ok := value.([]string)
	if !ok {
		return nil, false
	}
	
	return chunks, true
}

// SetChunks stores chunks for a document
func (cc *ChunkCache) SetChunks(ctx context.Context, documentID string, chunks []string, ttl time.Duration) error {
	key := "chunks:" + documentID
	return cc.cache.Set(ctx, key, chunks, ttl)
}

// CacheManager manages multiple cache types
type CacheManager struct {
	queryCache  *QueryCache
	vectorCache *VectorCache
	llmCache    *LLMCache
	chunkCache  *ChunkCache
	config      CacheConfig
}

// CacheConfig configuration for cache manager
type CacheConfig struct {
	EnableQueryCache  bool          `json:"enable_query_cache"`
	EnableVectorCache bool          `json:"enable_vector_cache"`
	EnableLLMCache    bool          `json:"enable_llm_cache"`
	EnableChunkCache  bool          `json:"enable_chunk_cache"`
	MaxSize           int           `json:"max_size"`
	DefaultTTL        time.Duration `json:"default_ttl"`
	QueryCacheTTL     time.Duration `json:"query_cache_ttl"`
	VectorCacheTTL    time.Duration `json:"vector_cache_ttl"`
	LLMCacheTTL       time.Duration `json:"llm_cache_ttl"`
	ChunkCacheTTL     time.Duration `json:"chunk_cache_ttl"`
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		EnableQueryCache:  true,
		EnableVectorCache: true,
		EnableLLMCache:    true,
		EnableChunkCache:  true,
		MaxSize:           1000,
		DefaultTTL:        1 * time.Hour,
		QueryCacheTTL:     30 * time.Minute,
		VectorCacheTTL:    2 * time.Hour,
		LLMCacheTTL:       1 * time.Hour,
		ChunkCacheTTL:     2 * time.Hour,
	}
}

// NewCacheManager creates a new cache manager
func NewCacheManager(config CacheConfig) *CacheManager {
	cm := &CacheManager{
		config: config,
	}
	
	if config.EnableQueryCache {
		cm.queryCache = NewQueryCache(config.MaxSize, config.QueryCacheTTL)
	}
	
	if config.EnableVectorCache {
		cm.vectorCache = NewVectorCache(config.MaxSize, config.VectorCacheTTL)
	}
	
	if config.EnableLLMCache {
		cm.llmCache = NewLLMCache(config.MaxSize, config.LLMCacheTTL)
	}
	
	if config.EnableChunkCache {
		cm.chunkCache = NewChunkCache(config.MaxSize, config.ChunkCacheTTL)
	}
	
	return cm
}

// QueryCache returns the query cache
func (cm *CacheManager) QueryCache() *QueryCache {
	return cm.queryCache
}

// VectorCache returns the vector cache
func (cm *CacheManager) VectorCache() *VectorCache {
	return cm.vectorCache
}

// LLMCache returns the LLM cache
func (cm *CacheManager) LLMCache() *LLMCache {
	return cm.llmCache
}

// ChunkCache returns the chunk cache
func (cm *CacheManager) ChunkCache() *ChunkCache {
	return cm.chunkCache
}

// ClearAll clears all caches
func (cm *CacheManager) ClearAll(ctx context.Context) error {
	if cm.queryCache != nil {
		cm.queryCache.cache.Clear(ctx)
	}
	if cm.vectorCache != nil {
		cm.vectorCache.cache.Clear(ctx)
	}
	if cm.llmCache != nil {
		cm.llmCache.cache.Clear(ctx)
	}
	if cm.chunkCache != nil {
		cm.chunkCache.cache.Clear(ctx)
	}
	return nil
}

// GetStats returns statistics for all caches
func (cm *CacheManager) GetStats() map[string]CacheStats {
	stats := make(map[string]CacheStats)
	
	if cm.queryCache != nil {
		stats["query"] = cm.queryCache.cache.Stats()
	}
	if cm.vectorCache != nil {
		stats["vector"] = cm.vectorCache.cache.Stats()
	}
	if cm.llmCache != nil {
		stats["llm"] = cm.llmCache.cache.Stats()
	}
	if cm.chunkCache != nil {
		stats["chunk"] = cm.chunkCache.cache.Stats()
	}
	
	return stats
}