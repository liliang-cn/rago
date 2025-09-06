package tools

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
)

// Cache provides caching for tool execution results
type Cache struct {
	mu            sync.RWMutex
	entries       map[string]*CacheEntry
	maxSize       int
	maxAge        time.Duration
	evictionPolicy EvictionPolicy
	stats         *CacheStats
	cleanupTicker *time.Ticker
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// CacheEntry represents a cached tool result
type CacheEntry struct {
	Key        string
	Response   *core.ToolCallResponse
	CreatedAt  time.Time
	LastAccess time.Time
	AccessCount int64
	Size       int
	TTL        time.Duration
}

// EvictionPolicy defines cache eviction strategy
type EvictionPolicy int

const (
	LRU EvictionPolicy = iota // Least Recently Used
	LFU                       // Least Frequently Used
	FIFO                      // First In First Out
	TTL                       // Time To Live based
)

// CacheStats tracks cache statistics
type CacheStats struct {
	mu          sync.RWMutex
	Hits        int64
	Misses      int64
	Evictions   int64
	Sets        int64
	Gets        int64
	CurrentSize int
	MaxSize     int
}

// NewCache creates a new cache
func NewCache(maxSize int, maxAge time.Duration, policy EvictionPolicy) *Cache {
	c := &Cache{
		entries:        make(map[string]*CacheEntry),
		maxSize:        maxSize,
		maxAge:         maxAge,
		evictionPolicy: policy,
		stats:          &CacheStats{MaxSize: maxSize},
		stopCh:         make(chan struct{}),
	}
	
	// Start cleanup routine
	c.startCleanup()
	
	return c
}

// Get retrieves a cached response
func (c *Cache) Get(request *core.ToolCallRequest) *core.ToolCallResponse {
	key := c.generateKey(request)
	
	c.mu.RLock()
	entry, exists := c.entries[key]
	c.mu.RUnlock()
	
	c.stats.recordGet()
	
	if !exists {
		c.stats.recordMiss()
		return nil
	}
	
	// Check if entry is expired
	if c.isExpired(entry) {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		c.stats.recordMiss()
		c.stats.recordEviction()
		return nil
	}
	
	// Update access information
	c.mu.Lock()
	entry.LastAccess = time.Now()
	entry.AccessCount++
	c.mu.Unlock()
	
	c.stats.recordHit()
	return entry.Response
}

// Set stores a response in the cache
func (c *Cache) Set(request *core.ToolCallRequest, response *core.ToolCallResponse, ttl time.Duration) {
	key := c.generateKey(request)
	
	entry := &CacheEntry{
		Key:         key,
		Response:    response,
		CreatedAt:   time.Now(),
		LastAccess:  time.Now(),
		AccessCount: 0,
		Size:        c.estimateSize(response),
		TTL:         ttl,
	}
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Check if we need to evict entries
	if len(c.entries) >= c.maxSize {
		c.evict()
	}
	
	c.entries[key] = entry
	c.stats.recordSet()
	c.stats.updateSize(len(c.entries))
}

// Delete removes an entry from the cache
func (c *Cache) Delete(request *core.ToolCallRequest) {
	key := c.generateKey(request)
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.entries, key)
	c.stats.updateSize(len(c.entries))
}

// Clear removes all entries from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.entries = make(map[string]*CacheEntry)
	c.stats.updateSize(0)
}

// Invalidate invalidates cache entries matching a pattern
func (c *Cache) Invalidate(pattern string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	invalidated := 0
	for key := range c.entries {
		if matchesPattern(key, pattern) {
			delete(c.entries, key)
			invalidated++
			c.stats.recordEviction()
		}
	}
	
	c.stats.updateSize(len(c.entries))
	return invalidated
}

// InvalidateByServer invalidates all cache entries for a specific server
func (c *Cache) InvalidateByServer(serverName string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	invalidated := 0
	for key, entry := range c.entries {
		if entry.Response != nil && contains(entry.Response.ToolName, serverName) {
			delete(c.entries, key)
			invalidated++
			c.stats.recordEviction()
		}
	}
	
	c.stats.updateSize(len(c.entries))
	return invalidated
}

// GetStats returns cache statistics
func (c *Cache) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return c.stats.getStats()
}

// Stop stops the cache cleanup routine
func (c *Cache) Stop() {
	close(c.stopCh)
	c.wg.Wait()
	if c.cleanupTicker != nil {
		c.cleanupTicker.Stop()
	}
}

// generateKey generates a cache key for a request
func (c *Cache) generateKey(request *core.ToolCallRequest) string {
	// Create a deterministic key from the request
	data := map[string]interface{}{
		"tool":      request.ToolName,
		"arguments": request.Arguments,
	}
	
	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return fmt.Sprintf("%s_%x", request.ToolName, hash[:8])
}

// isExpired checks if a cache entry is expired
func (c *Cache) isExpired(entry *CacheEntry) bool {
	if entry.TTL > 0 {
		return time.Since(entry.CreatedAt) > entry.TTL
	}
	if c.maxAge > 0 {
		return time.Since(entry.CreatedAt) > c.maxAge
	}
	return false
}

// estimateSize estimates the size of a response in bytes
func (c *Cache) estimateSize(response *core.ToolCallResponse) int {
	// Simple estimation based on JSON serialization
	data, _ := json.Marshal(response)
	return len(data)
}

// evict removes entries based on the eviction policy
func (c *Cache) evict() {
	if len(c.entries) == 0 {
		return
	}
	
	switch c.evictionPolicy {
	case LRU:
		c.evictLRU()
	case LFU:
		c.evictLFU()
	case FIFO:
		c.evictFIFO()
	case TTL:
		c.evictExpired()
	default:
		c.evictLRU() // Default to LRU
	}
}

// evictLRU evicts the least recently used entry
func (c *Cache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time
	
	for key, entry := range c.entries {
		if oldestKey == "" || entry.LastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.LastAccess
		}
	}
	
	if oldestKey != "" {
		delete(c.entries, oldestKey)
		c.stats.recordEviction()
	}
}

// evictLFU evicts the least frequently used entry
func (c *Cache) evictLFU() {
	var leastKey string
	var leastCount int64 = -1
	
	for key, entry := range c.entries {
		if leastCount == -1 || entry.AccessCount < leastCount {
			leastKey = key
			leastCount = entry.AccessCount
		}
	}
	
	if leastKey != "" {
		delete(c.entries, leastKey)
		c.stats.recordEviction()
	}
}

// evictFIFO evicts the oldest entry
func (c *Cache) evictFIFO() {
	var oldestKey string
	var oldestTime time.Time
	
	for key, entry := range c.entries {
		if oldestKey == "" || entry.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CreatedAt
		}
	}
	
	if oldestKey != "" {
		delete(c.entries, oldestKey)
		c.stats.recordEviction()
	}
}

// evictExpired evicts all expired entries
func (c *Cache) evictExpired() {
	evicted := 0
	for key, entry := range c.entries {
		if c.isExpired(entry) {
			delete(c.entries, key)
			evicted++
			c.stats.recordEviction()
		}
	}
	
	// If no expired entries, fall back to LRU
	if evicted == 0 && len(c.entries) >= c.maxSize {
		c.evictLRU()
	}
}

// startCleanup starts the periodic cleanup routine
func (c *Cache) startCleanup() {
	c.cleanupTicker = time.NewTicker(1 * time.Minute)
	c.wg.Add(1)
	
	go func() {
		defer c.wg.Done()
		
		for {
			select {
			case <-c.stopCh:
				return
			case <-c.cleanupTicker.C:
				c.cleanup()
			}
		}
	}()
}

// cleanup removes expired entries
func (c *Cache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	for key, entry := range c.entries {
		if c.isExpired(entry) {
			delete(c.entries, key)
			c.stats.recordEviction()
		}
	}
	
	c.stats.updateSize(len(c.entries))
}

// CacheStats methods

func (s *CacheStats) recordHit() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Hits++
}

func (s *CacheStats) recordMiss() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Misses++
}

func (s *CacheStats) recordEviction() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Evictions++
}

func (s *CacheStats) recordSet() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Sets++
}

func (s *CacheStats) recordGet() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Gets++
}

func (s *CacheStats) updateSize(size int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CurrentSize = size
}

func (s *CacheStats) getStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	hitRate := float64(0)
	if s.Gets > 0 {
		hitRate = float64(s.Hits) / float64(s.Gets)
	}
	
	return map[string]interface{}{
		"hits":         s.Hits,
		"misses":       s.Misses,
		"hit_rate":     hitRate,
		"evictions":    s.Evictions,
		"sets":         s.Sets,
		"gets":         s.Gets,
		"current_size": s.CurrentSize,
		"max_size":     s.MaxSize,
		"utilization":  float64(s.CurrentSize) / float64(s.MaxSize),
	}
}

// matchesPattern checks if a key matches a pattern (simplified implementation)
func matchesPattern(key, pattern string) bool {
	// Simplified pattern matching - in production use proper pattern matching
	return contains(key, pattern)
}

// CacheWarmer pre-warms the cache with common tool calls
type CacheWarmer struct {
	cache    *Cache
	executor ToolExecutor
	patterns []WarmupPattern
}

// WarmupPattern defines a pattern for cache warming
type WarmupPattern struct {
	ToolName  string
	Arguments map[string]interface{}
	TTL       time.Duration
}

// ToolExecutor interface for executing tools (used by cache warmer)
type ToolExecutor interface {
	Execute(ctx context.Context, request *core.ToolCallRequest) (*core.ToolCallResponse, error)
}

// NewCacheWarmer creates a new cache warmer
func NewCacheWarmer(cache *Cache, executor ToolExecutor) *CacheWarmer {
	return &CacheWarmer{
		cache:    cache,
		executor: executor,
		patterns: []WarmupPattern{},
	}
}

// AddPattern adds a warmup pattern
func (w *CacheWarmer) AddPattern(pattern WarmupPattern) {
	w.patterns = append(w.patterns, pattern)
}

// Warmup executes warmup patterns to pre-populate the cache
func (w *CacheWarmer) Warmup(ctx context.Context) error {
	for _, pattern := range w.patterns {
		request := &core.ToolCallRequest{
			ToolName:  pattern.ToolName,
			Arguments: pattern.Arguments,
		}
		
		// Execute the tool
		response, err := w.executor.Execute(ctx, request)
		if err != nil {
			// Log error but continue with other patterns
			fmt.Printf("Failed to warm cache for %s: %v\n", pattern.ToolName, err)
			continue
		}
		
		// Cache the response
		w.cache.Set(request, response, pattern.TTL)
	}
	
	return nil
}