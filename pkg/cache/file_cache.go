package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// FileCache persists cache entries as JSON files on disk.
// It is intended for local-first flows where cache warmup across restarts matters
// more than raw in-process performance.
type FileCache struct {
	mu         sync.RWMutex
	baseDir    string
	entries    map[string]*CacheEntry
	lru        *lruList
	maxSize    int
	stats      CacheStats
	defaultTTL time.Duration
}

type persistedCacheEntry struct {
	Key        string          `json:"key"`
	Value      json.RawMessage `json:"value"`
	CreatedAt  time.Time       `json:"created_at"`
	ExpiresAt  time.Time       `json:"expires_at"`
	AccessedAt time.Time       `json:"accessed_at"`
	HitCount   int64           `json:"hit_count"`
}

// NewFileCache creates a new file-backed cache and restores unexpired entries from disk.
func NewFileCache(baseDir string, maxSize int, defaultTTL time.Duration) (*FileCache, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	fc := &FileCache{
		baseDir:    baseDir,
		entries:    make(map[string]*CacheEntry),
		lru:        newLRUList(),
		maxSize:    maxSize,
		defaultTTL: defaultTTL,
		stats: CacheStats{
			MaxSize:   maxSize,
			CreatedAt: time.Now(),
		},
	}

	if err := fc.loadFromDisk(); err != nil {
		return nil, err
	}

	return fc, nil
}

// Get retrieves a value from cache.
func (fc *FileCache) Get(ctx context.Context, key string) (interface{}, bool) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	entry, exists := fc.entries[key]
	if !exists {
		fc.stats.Misses++
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		fc.deleteEntryLocked(key)
		fc.stats.Misses++
		return nil, false
	}

	entry.AccessedAt = time.Now()
	entry.HitCount++
	fc.stats.Hits++
	fc.lru.moveToFront(key)
	_ = fc.persistEntryLocked(entry)

	return entry.Value, true
}

// Set stores a value in cache.
func (fc *FileCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	if ttl <= 0 {
		ttl = fc.defaultTTL
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

	if len(fc.entries) >= fc.maxSize && fc.entries[key] == nil {
		evictKey := fc.lru.removeLast()
		if evictKey != "" {
			if err := fc.deleteEntryFileLocked(evictKey); err != nil {
				return err
			}
			delete(fc.entries, evictKey)
			fc.stats.Evictions++
		}
	}

	if err := fc.persistEntryLocked(entry); err != nil {
		return err
	}

	fc.entries[key] = entry
	fc.lru.addToFront(key)
	fc.stats.Size = len(fc.entries)

	return nil
}

// Delete removes a value from cache.
func (fc *FileCache) Delete(ctx context.Context, key string) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	return fc.deleteEntryLocked(key)
}

// Clear removes all entries from cache and disk.
func (fc *FileCache) Clear(ctx context.Context) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	for key := range fc.entries {
		if err := fc.deleteEntryFileLocked(key); err != nil {
			return err
		}
	}

	fc.entries = make(map[string]*CacheEntry)
	fc.lru = newLRUList()
	fc.stats.Size = 0
	fc.stats.LastClear = time.Now()

	return nil
}

// Size returns the number of entries in cache.
func (fc *FileCache) Size() int {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return len(fc.entries)
}

// Stats returns cache statistics.
func (fc *FileCache) Stats() CacheStats {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	fc.stats.Size = len(fc.entries)
	return fc.stats
}

func (fc *FileCache) loadFromDisk() error {
	files, err := filepath.Glob(filepath.Join(fc.baseDir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to list cache files: %w", err)
	}

	type restoreItem struct {
		entry *CacheEntry
	}

	now := time.Now()
	var restored []restoreItem

	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read cache file %s: %w", path, err)
		}

		var persisted persistedCacheEntry
		if err := json.Unmarshal(data, &persisted); err != nil {
			return fmt.Errorf("failed to parse cache file %s: %w", path, err)
		}

		if now.After(persisted.ExpiresAt) {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove expired cache file %s: %w", path, err)
			}
			continue
		}

		value, err := decodePersistedValue(persisted.Value)
		if err != nil {
			return fmt.Errorf("failed to decode cache value %s: %w", path, err)
		}

		restored = append(restored, restoreItem{
			entry: &CacheEntry{
				Key:        persisted.Key,
				Value:      value,
				CreatedAt:  persisted.CreatedAt,
				ExpiresAt:  persisted.ExpiresAt,
				AccessedAt: persisted.AccessedAt,
				HitCount:   persisted.HitCount,
			},
		})
	}

	sort.Slice(restored, func(i, j int) bool {
		return restored[i].entry.AccessedAt.Before(restored[j].entry.AccessedAt)
	})

	for _, item := range restored {
		fc.entries[item.entry.Key] = item.entry
		fc.lru.addToFront(item.entry.Key)
	}

	fc.stats.Size = len(fc.entries)
	return nil
}

func (fc *FileCache) persistEntryLocked(entry *CacheEntry) error {
	valueBytes, err := json.Marshal(entry.Value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache value for key %s: %w", entry.Key, err)
	}

	persisted := persistedCacheEntry{
		Key:        entry.Key,
		Value:      valueBytes,
		CreatedAt:  entry.CreatedAt,
		ExpiresAt:  entry.ExpiresAt,
		AccessedAt: entry.AccessedAt,
		HitCount:   entry.HitCount,
	}

	data, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry for key %s: %w", entry.Key, err)
	}

	if err := os.WriteFile(fc.filePathForKey(entry.Key), data, 0644); err != nil {
		return fmt.Errorf("failed to persist cache key %s: %w", entry.Key, err)
	}

	return nil
}

func (fc *FileCache) deleteEntryLocked(key string) error {
	if _, exists := fc.entries[key]; !exists {
		return nil
	}

	delete(fc.entries, key)
	fc.lru.remove(key)
	fc.stats.Size = len(fc.entries)

	return fc.deleteEntryFileLocked(key)
}

func (fc *FileCache) deleteEntryFileLocked(key string) error {
	if err := os.Remove(fc.filePathForKey(key)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete cache file for key %s: %w", key, err)
	}
	return nil
}

func (fc *FileCache) filePathForKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(fc.baseDir, hex.EncodeToString(sum[:])+".json")
}

func decodePersistedValue(data []byte) (interface{}, error) {
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	return normalizeJSONValue(value), nil
}

func normalizeJSONValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case []interface{}:
		if len(typed) == 0 {
			return []interface{}{}
		}

		allStrings := true
		allNumbers := true
		stringsValue := make([]string, 0, len(typed))
		numbersValue := make([]float64, 0, len(typed))

		for _, item := range typed {
			switch v := item.(type) {
			case string:
				stringsValue = append(stringsValue, v)
				allNumbers = false
			case float64:
				numbersValue = append(numbersValue, v)
				allStrings = false
			default:
				allStrings = false
				allNumbers = false
			}
		}

		switch {
		case allStrings:
			return stringsValue
		case allNumbers:
			return numbersValue
		default:
			normalized := make([]interface{}, 0, len(typed))
			for _, item := range typed {
				normalized = append(normalized, normalizeJSONValue(item))
			}
			return normalized
		}
	case map[string]interface{}:
		normalized := make(map[string]interface{}, len(typed))
		for k, v := range typed {
			normalized[k] = normalizeJSONValue(v)
		}
		return normalized
	default:
		return typed
	}
}
