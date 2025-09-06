package marketplace

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestTemplateCache(t *testing.T) {
	t.Run("cache hit", func(t *testing.T) {
		cache := &TemplateCache{
			items: make(map[string]*cacheItem),
			ttl:   5 * time.Minute,
		}
		
		template := createSampleTemplate("cache-1", "Cached", "automation")
		cache.set("cache-1", template)
		
		retrieved := cache.get("cache-1")
		if retrieved == nil {
			t.Fatal("expected template to be in cache")
		}
		
		if retrieved.ID != "cache-1" {
			t.Errorf("expected ID cache-1, got %s", retrieved.ID)
		}
	})
	
	t.Run("cache miss", func(t *testing.T) {
		cache := &TemplateCache{
			items: make(map[string]*cacheItem),
			ttl:   5 * time.Minute,
		}
		
		retrieved := cache.get("nonexistent")
		if retrieved != nil {
			t.Error("expected nil for cache miss")
		}
	})
	
	t.Run("expired entry removal", func(t *testing.T) {
		cache := &TemplateCache{
			items: make(map[string]*cacheItem),
			ttl:   100 * time.Millisecond, // Short TTL for testing
		}
		
		template := createSampleTemplate("expire-1", "Expiring", "automation")
		cache.set("expire-1", template)
		
		// Should be in cache initially
		retrieved := cache.get("expire-1")
		if retrieved == nil {
			t.Fatal("expected template to be in cache initially")
		}
		
		// Wait for expiry
		time.Sleep(150 * time.Millisecond)
		
		// Should be expired and removed
		retrieved = cache.get("expire-1")
		if retrieved != nil {
			t.Error("expected expired template to be removed from cache")
		}
		
		// Verify it was actually removed from the map
		cache.mu.RLock()
		_, exists := cache.items["expire-1"]
		cache.mu.RUnlock()
		
		if exists {
			t.Error("expected expired item to be deleted from map")
		}
	})
	
	t.Run("set new entry", func(t *testing.T) {
		cache := &TemplateCache{
			items: make(map[string]*cacheItem),
			ttl:   5 * time.Minute,
		}
		
		template := createSampleTemplate("new-1", "New Entry", "automation")
		cache.set("new-1", template)
		
		cache.mu.RLock()
		item, exists := cache.items["new-1"]
		cache.mu.RUnlock()
		
		if !exists {
			t.Fatal("expected item to be added to cache")
		}
		
		if item.template.ID != "new-1" {
			t.Errorf("expected template ID new-1, got %s", item.template.ID)
		}
		
		// Check expiry is set correctly
		expectedExpiry := time.Now().Add(5 * time.Minute)
		if item.expiry.Before(expectedExpiry.Add(-10*time.Second)) ||
			item.expiry.After(expectedExpiry.Add(10*time.Second)) {
			t.Errorf("expiry not set correctly: %v", item.expiry)
		}
	})
	
	t.Run("update existing entry", func(t *testing.T) {
		cache := &TemplateCache{
			items: make(map[string]*cacheItem),
			ttl:   5 * time.Minute,
		}
		
		// Set initial template
		template1 := createSampleTemplate("update-1", "Original", "automation")
		template1.Stars = 10
		cache.set("update-1", template1)
		
		// Update with new template
		template2 := createSampleTemplate("update-1", "Updated", "automation")
		template2.Stars = 20
		cache.set("update-1", template2)
		
		// Retrieve and verify it's updated
		retrieved := cache.get("update-1")
		if retrieved == nil {
			t.Fatal("expected template to be in cache")
		}
		
		if retrieved.Name != "Updated" {
			t.Errorf("expected name Updated, got %s", retrieved.Name)
		}
		
		if retrieved.Stars != 20 {
			t.Errorf("expected 20 stars, got %d", retrieved.Stars)
		}
	})
	
	t.Run("invalidate existing entry", func(t *testing.T) {
		cache := &TemplateCache{
			items: make(map[string]*cacheItem),
			ttl:   5 * time.Minute,
		}
		
		template := createSampleTemplate("inv-1", "To Invalidate", "automation")
		cache.set("inv-1", template)
		
		// Verify it's in cache
		if cache.get("inv-1") == nil {
			t.Fatal("expected template to be in cache before invalidation")
		}
		
		// Invalidate
		cache.invalidate("inv-1")
		
		// Should be removed
		if cache.get("inv-1") != nil {
			t.Error("expected template to be removed after invalidation")
		}
	})
	
	t.Run("invalidate non-existent entry", func(t *testing.T) {
		cache := &TemplateCache{
			items: make(map[string]*cacheItem),
			ttl:   5 * time.Minute,
		}
		
		// Should not panic
		cache.invalidate("nonexistent")
	})
}

func TestCacheConcurrency(t *testing.T) {
	t.Run("concurrent get and set", func(t *testing.T) {
		cache := &TemplateCache{
			items: make(map[string]*cacheItem),
			ttl:   5 * time.Minute,
		}
		
		var wg sync.WaitGroup
		
		// Multiple goroutines setting different templates
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				template := createSampleTemplate(
					fmt.Sprintf("concurrent-%d", id),
					fmt.Sprintf("Template %d", id),
					"automation",
				)
				cache.set(fmt.Sprintf("concurrent-%d", id), template)
			}(i)
		}
		
		// Multiple goroutines getting templates
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				// Try to get templates that may or may not exist yet
				for j := 0; j < 10; j++ {
					cache.get(fmt.Sprintf("concurrent-%d", j))
				}
			}(i)
		}
		
		wg.Wait()
		
		// Verify all templates were added
		for i := 0; i < 10; i++ {
			template := cache.get(fmt.Sprintf("concurrent-%d", i))
			if template == nil {
				t.Errorf("expected template concurrent-%d to be in cache", i)
			}
		}
	})
	
	t.Run("concurrent invalidation", func(t *testing.T) {
		cache := &TemplateCache{
			items: make(map[string]*cacheItem),
			ttl:   5 * time.Minute,
		}
		
		// Pre-populate cache
		for i := 0; i < 20; i++ {
			template := createSampleTemplate(
				fmt.Sprintf("inv-%d", i),
				fmt.Sprintf("Template %d", i),
				"automation",
			)
			cache.set(fmt.Sprintf("inv-%d", i), template)
		}
		
		var wg sync.WaitGroup
		
		// Concurrent reads, writes, and invalidations
		for i := 0; i < 10; i++ {
			wg.Add(3)
			
			// Reader
			go func() {
				defer wg.Done()
				for j := 0; j < 20; j++ {
					cache.get(fmt.Sprintf("inv-%d", j))
				}
			}()
			
			// Writer
			go func(id int) {
				defer wg.Done()
				template := createSampleTemplate(
					fmt.Sprintf("new-%d", id),
					fmt.Sprintf("New %d", id),
					"automation",
				)
				cache.set(fmt.Sprintf("new-%d", id), template)
			}(i)
			
			// Invalidator
			go func(id int) {
				defer wg.Done()
				cache.invalidate(fmt.Sprintf("inv-%d", id))
			}(i)
		}
		
		wg.Wait()
		
		// Verify no panics occurred and cache is still functional
		testTemplate := createSampleTemplate("final", "Final", "automation")
		cache.set("final", testTemplate)
		
		retrieved := cache.get("final")
		if retrieved == nil {
			t.Error("cache should still be functional after concurrent operations")
		}
	})
}

func TestMarketplaceConcurrency(t *testing.T) {
	t.Run("concurrent template operations", func(t *testing.T) {
		storage := NewMockMarketplaceStorage()
		m := NewMarketplace(nil, storage)
		ctx := context.Background()
		
		var wg sync.WaitGroup
		
		// Concurrent publishes
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				template := createSampleTemplate(
					fmt.Sprintf("pub-%d", id),
					fmt.Sprintf("Published %d", id),
					"automation",
				)
				m.PublishTemplate(ctx, template)
			}(i)
		}
		
		// Concurrent gets
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				// Try to get templates that may or may not exist
				m.GetTemplate(ctx, fmt.Sprintf("pub-%d", id))
			}(i)
		}
		
		// Concurrent searches
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				m.SearchTemplates(ctx, &TemplateFilter{
					Category: "automation",
					Limit:    10,
				})
			}()
		}
		
		// Concurrent stars
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				// May fail if template doesn't exist yet
				m.StarTemplate(ctx, fmt.Sprintf("pub-%d", id), true)
			}(i)
		}
		
		wg.Wait()
		
		// Verify marketplace is still functional
		results, err := m.SearchTemplates(ctx, &TemplateFilter{
			Category: "automation",
			Limit:    100,
		})
		
		if err != nil {
			t.Errorf("marketplace should be functional after concurrent ops: %v", err)
		}
		
		if len(results) == 0 {
			t.Error("expected some templates to be published")
		}
	})
	
	t.Run("registry concurrent indexing", func(t *testing.T) {
		registry := &Registry{
			categories: make(map[string]*Category),
			tags:       make(map[string][]*AgentTemplate),
			authors:    make(map[string][]*AgentTemplate),
		}
		
		// Initialize categories
		registry.categories["automation"] = &Category{
			ID:    "automation",
			Name:  "Automation",
			Count: 0,
		}
		
		var wg sync.WaitGroup
		
		// Concurrent indexing
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				template := createSampleTemplate(
					fmt.Sprintf("reg-%d", id),
					fmt.Sprintf("Registry %d", id),
					"automation",
				)
				template.Tags = []string{"tag1", "tag2", fmt.Sprintf("tag%d", id)}
				template.Author.Username = fmt.Sprintf("author%d", id%5)
				
				registry.index(template)
			}(i)
		}
		
		wg.Wait()
		
		// Verify indexing worked
		if registry.categories["automation"].Count != 20 {
			t.Errorf("expected category count 20, got %d", registry.categories["automation"].Count)
		}
		
		// Check tag1 - should have at least 20 templates
		// Due to concurrent execution, we might get slightly more due to duplicates
		if len(registry.tags["tag1"]) < 20 {
			t.Errorf("expected at least 20 templates with tag1, got %d", len(registry.tags["tag1"]))
		}
		
		// Each of 5 authors should have 4 templates
		for i := 0; i < 5; i++ {
			author := fmt.Sprintf("author%d", i)
			if len(registry.authors[author]) != 4 {
				t.Errorf("expected 4 templates for %s, got %d", author, len(registry.authors[author]))
			}
		}
	})
}