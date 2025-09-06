package marketplace

import (
	"context"
	"fmt"
	"testing"
)

func TestStarTemplate(t *testing.T) {
	t.Run("star a template", func(t *testing.T) {
		storage := NewMockMarketplaceStorage()
		m := NewMarketplace(nil, storage)
		ctx := context.Background()
		
		// Add a template
		template := createSampleTemplate("star-1", "Starrable", "automation")
		template.Stars = 10
		err := m.PublishTemplate(ctx, template)
		if err != nil {
			t.Fatalf("failed to publish template: %v", err)
		}
		
		// Star the template
		err = m.StarTemplate(ctx, "star-1", true)
		if err != nil {
			t.Fatalf("expected no error starring template, got %v", err)
		}
		
		// Check star count increased
		if m.templates["star-1"].Stars != 11 {
			t.Errorf("expected stars to be 11, got %d", m.templates["star-1"].Stars)
		}
		
		// Check storage was called
		if storage.GetStarCount("star-1") != 1 {
			t.Error("expected storage star count to be incremented")
		}
	})
	
	t.Run("unstar a template", func(t *testing.T) {
		storage := NewMockMarketplaceStorage()
		m := NewMarketplace(nil, storage)
		ctx := context.Background()
		
		// Add a template with stars
		template := createSampleTemplate("unstar-1", "Unstarrable", "automation")
		template.Stars = 10
		err := m.PublishTemplate(ctx, template)
		if err != nil {
			t.Fatalf("failed to publish template: %v", err)
		}
		
		// Unstar the template
		err = m.StarTemplate(ctx, "unstar-1", false)
		if err != nil {
			t.Fatalf("expected no error unstarring template, got %v", err)
		}
		
		// Check star count decreased
		if m.templates["unstar-1"].Stars != 9 {
			t.Errorf("expected stars to be 9, got %d", m.templates["unstar-1"].Stars)
		}
	})
	
	t.Run("unstar template with zero stars", func(t *testing.T) {
		storage := NewMockMarketplaceStorage()
		m := NewMarketplace(nil, storage)
		ctx := context.Background()
		
		// Add a template with no stars
		template := createSampleTemplate("zero-1", "Zero Stars", "automation")
		template.Stars = 0
		err := m.PublishTemplate(ctx, template)
		if err != nil {
			t.Fatalf("failed to publish template: %v", err)
		}
		
		// Unstar the template
		err = m.StarTemplate(ctx, "zero-1", false)
		if err != nil {
			t.Fatalf("expected no error unstarring template, got %v", err)
		}
		
		// Check star count remains zero
		if m.templates["zero-1"].Stars != 0 {
			t.Errorf("expected stars to remain 0, got %d", m.templates["zero-1"].Stars)
		}
	})
	
	t.Run("star non-existent template", func(t *testing.T) {
		storage := NewMockMarketplaceStorage()
		m := NewMarketplace(nil, storage)
		ctx := context.Background()
		
		err := m.StarTemplate(ctx, "nonexistent", true)
		if err == nil {
			t.Fatal("expected error for non-existent template")
		}
	})
}

func TestInstallTemplate(t *testing.T) {
	t.Run("successful installation", func(t *testing.T) {
		storage := NewMockMarketplaceStorage()
		m := NewMarketplace(nil, storage)
		ctx := context.Background()
		
		// Add a template
		template := createSampleTemplate("install-1", "Installable", "automation")
		template.Config.SystemPrompt = "Test system prompt"
		template.Tools = []string{"tool1", "tool2"}
		err := m.PublishTemplate(ctx, template)
		if err != nil {
			t.Fatalf("failed to publish template: %v", err)
		}
		
		// Install the template
		instance, err := m.InstallTemplate(ctx, "install-1")
		if err != nil {
			t.Fatalf("expected no error installing template, got %v", err)
		}
		
		// Verify instance
		if instance == nil {
			t.Fatal("expected instance to be created")
		}
		
		if instance.TemplateID != "install-1" {
			t.Errorf("expected template ID install-1, got %s", instance.TemplateID)
		}
		
		if instance.Name != "Installable" {
			t.Errorf("expected name Installable, got %s", instance.Name)
		}
		
		if instance.Config.SystemPrompt != "Test system prompt" {
			t.Errorf("expected config to be copied")
		}
		
		if len(instance.Tools) != 2 {
			t.Errorf("expected 2 tools, got %d", len(instance.Tools))
		}
		
		if instance.Status != "installed" {
			t.Errorf("expected status installed, got %s", instance.Status)
		}
		
		if instance.ID == "" {
			t.Error("expected instance ID to be generated")
		}
		
		if instance.CreatedAt.IsZero() {
			t.Error("expected CreatedAt to be set")
		}
	})
	
	t.Run("install non-existent template", func(t *testing.T) {
		storage := NewMockMarketplaceStorage()
		m := NewMarketplace(nil, storage)
		ctx := context.Background()
		
		instance, err := m.InstallTemplate(ctx, "nonexistent")
		if err == nil {
			t.Fatal("expected error for non-existent template")
		}
		
		if instance != nil {
			t.Error("expected nil instance for non-existent template")
		}
	})
}

func TestGetCategories(t *testing.T) {
	t.Run("return all categories", func(t *testing.T) {
		m := NewMarketplace(nil, nil)
		
		categories := m.GetCategories()
		
		if len(categories) == 0 {
			t.Fatal("expected categories to be returned")
		}
		
		// Check for expected default categories
		expectedCategories := map[string]bool{
			"data-analysis": true,
			"automation":    true,
			"research":      true,
			"content":       true,
			"coding":        true,
			"testing":       true,
			"monitoring":    true,
			"integration":   true,
		}
		
		for _, cat := range categories {
			if !expectedCategories[cat.ID] {
				t.Errorf("unexpected category: %s", cat.ID)
			}
			delete(expectedCategories, cat.ID)
		}
		
		if len(expectedCategories) > 0 {
			t.Errorf("missing categories: %v", expectedCategories)
		}
	})
	
	t.Run("category count updates", func(t *testing.T) {
		storage := NewMockMarketplaceStorage()
		m := NewMarketplace(nil, storage)
		ctx := context.Background()
		
		// Get initial automation category count
		var initialCount int
		for _, cat := range m.GetCategories() {
			if cat.ID == "automation" {
				initialCount = cat.Count
				break
			}
		}
		
		// Add templates to automation category
		for i := 0; i < 3; i++ {
			template := createSampleTemplate(fmt.Sprintf("auto-%d", i), "Auto Template", "automation")
			m.PublishTemplate(ctx, template)
		}
		
		// Check count increased
		var newCount int
		for _, cat := range m.GetCategories() {
			if cat.ID == "automation" {
				newCount = cat.Count
				break
			}
		}
		
		if newCount != initialCount+3 {
			t.Errorf("expected count to increase by 3, got increase of %d", newCount-initialCount)
		}
	})
}

func TestGetPopularTags(t *testing.T) {
	t.Run("tag limit enforcement", func(t *testing.T) {
		storage := NewMockMarketplaceStorage()
		m := NewMarketplace(nil, storage)
		ctx := context.Background()
		
		// Add templates with various tags
		templates := []struct {
			id   string
			tags []string
		}{
			{"t1", []string{"python", "data", "analysis"}},
			{"t2", []string{"golang", "api", "web"}},
			{"t3", []string{"python", "testing", "automation"}},
			{"t4", []string{"javascript", "web", "frontend"}},
			{"t5", []string{"python", "machine-learning", "ai"}},
		}
		
		for _, tmpl := range templates {
			template := createSampleTemplate(tmpl.id, "Template", "automation")
			template.Tags = tmpl.tags
			m.PublishTemplate(ctx, template)
		}
		
		// Get top 3 tags
		tags := m.GetPopularTags(3)
		
		if len(tags) != 3 {
			t.Errorf("expected 3 tags, got %d", len(tags))
		}
	})
	
	t.Run("returns available tags when limit exceeds total", func(t *testing.T) {
		storage := NewMockMarketplaceStorage()
		m := NewMarketplace(nil, storage)
		ctx := context.Background()
		
		// Add template with 2 tags
		template := createSampleTemplate("t1", "Template", "automation")
		template.Tags = []string{"tag1", "tag2"}
		m.PublishTemplate(ctx, template)
		
		// Request more tags than available
		tags := m.GetPopularTags(10)
		
		if len(tags) > 2 {
			t.Errorf("expected at most 2 tags, got %d", len(tags))
		}
	})
}