package marketplace

import (
	"context"
	"testing"
)

func TestSearchTemplates(t *testing.T) {
	// Helper function to create a populated marketplace
	createPopulatedMarketplace := func() *Marketplace {
		storage := NewMockMarketplaceStorage()
		m := NewMarketplace(nil, storage)
		ctx := context.Background()
		
		// Add various templates
		templates := []*AgentTemplate{
			createSampleTemplate("t1", "Data Analysis Tool", "data-analysis"),
			createSampleTemplate("t2", "Code Generator", "coding"),
			createSampleTemplate("t3", "Test Automation", "testing"),
			createSampleTemplate("t4", "Research Assistant", "research"),
			createSampleTemplate("t5", "Content Writer", "content"),
		}
		
		// Customize templates
		templates[0].Tags = []string{"python", "pandas", "analysis"}
		templates[0].Stars = 50
		templates[0].Author.Username = "alice"
		
		templates[1].Tags = []string{"golang", "generator", "code"}
		templates[1].Stars = 30
		templates[1].Author.Username = "bob"
		
		templates[2].Tags = []string{"testing", "automation", "ci"}
		templates[2].Stars = 20
		templates[2].Author.Username = "alice"
		
		templates[3].Tags = []string{"research", "ai", "analysis"}
		templates[3].Stars = 40
		templates[3].Author.Username = "charlie"
		
		templates[4].Tags = []string{"writing", "content", "ai"}
		templates[4].Stars = 10
		templates[4].Author.Username = "bob"
		
		// Publish all templates
		for _, tmpl := range templates {
			if err := m.PublishTemplate(ctx, tmpl); err != nil {
				t.Fatalf("failed to publish template %s: %v", tmpl.ID, err)
			}
		}
		
		return m
	}
	
	t.Run("no filter returns all", func(t *testing.T) {
		m := createPopulatedMarketplace()
		ctx := context.Background()
		
		results, err := m.SearchTemplates(ctx, &TemplateFilter{
			Limit: 100,
		})
		
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		
		if len(results) != 5 {
			t.Errorf("expected 5 templates, got %d", len(results))
		}
	})
	
	t.Run("filter by category", func(t *testing.T) {
		m := createPopulatedMarketplace()
		ctx := context.Background()
		
		results, err := m.SearchTemplates(ctx, &TemplateFilter{
			Category: "coding",
			Limit:    100,
		})
		
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		
		if len(results) != 1 {
			t.Errorf("expected 1 template in coding category, got %d", len(results))
		}
		
		if results[0].ID != "t2" {
			t.Errorf("expected template t2, got %s", results[0].ID)
		}
	})
	
	t.Run("filter by author", func(t *testing.T) {
		m := createPopulatedMarketplace()
		ctx := context.Background()
		
		results, err := m.SearchTemplates(ctx, &TemplateFilter{
			Author: "alice",
			Limit:  100,
		})
		
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		
		if len(results) != 2 {
			t.Errorf("expected 2 templates by alice, got %d", len(results))
		}
		
		// Check both are by alice
		for _, tmpl := range results {
			if tmpl.Author.Username != "alice" {
				t.Errorf("expected author alice, got %s", tmpl.Author.Username)
			}
		}
	})
	
	t.Run("filter by single tag", func(t *testing.T) {
		m := createPopulatedMarketplace()
		ctx := context.Background()
		
		results, err := m.SearchTemplates(ctx, &TemplateFilter{
			Tags:  []string{"analysis"},
			Limit: 100,
		})
		
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		
		if len(results) != 2 {
			t.Errorf("expected 2 templates with 'analysis' tag, got %d", len(results))
		}
	})
	
	t.Run("filter by multiple tags", func(t *testing.T) {
		m := createPopulatedMarketplace()
		ctx := context.Background()
		
		results, err := m.SearchTemplates(ctx, &TemplateFilter{
			Tags:  []string{"ai", "analysis"},
			Limit: 100,
		})
		
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		
		// Should find templates that have at least one of the tags
		if len(results) < 2 {
			t.Errorf("expected at least 2 templates with ai or analysis tags, got %d", len(results))
		}
	})
	
	t.Run("filter by search term", func(t *testing.T) {
		m := createPopulatedMarketplace()
		ctx := context.Background()
		
		results, err := m.SearchTemplates(ctx, &TemplateFilter{
			SearchTerm: "Analysis",
			Limit:      100,
		})
		
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		
		// Should find "Data Analysis Tool"
		found := false
		for _, tmpl := range results {
			if tmpl.ID == "t1" {
				found = true
				break
			}
		}
		
		if !found {
			t.Error("expected to find Data Analysis Tool")
		}
	})
	
	t.Run("filter by minimum stars", func(t *testing.T) {
		m := createPopulatedMarketplace()
		ctx := context.Background()
		
		results, err := m.SearchTemplates(ctx, &TemplateFilter{
			MinStars: 30,
			Limit:    100,
		})
		
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		
		// Should find templates with >= 30 stars (t1=50, t2=30, t4=40)
		if len(results) != 3 {
			t.Errorf("expected 3 templates with >= 30 stars, got %d", len(results))
		}
		
		// Verify all have enough stars
		for _, tmpl := range results {
			if tmpl.Stars < 30 {
				t.Errorf("template %s has %d stars, expected >= 30", tmpl.ID, tmpl.Stars)
			}
		}
	})
	
	t.Run("combined filters", func(t *testing.T) {
		m := createPopulatedMarketplace()
		ctx := context.Background()
		
		results, err := m.SearchTemplates(ctx, &TemplateFilter{
			Author:   "alice",
			MinStars: 25,
			Limit:    100,
		})
		
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		
		// Only t1 should match (by alice with 50 stars)
		if len(results) != 1 {
			t.Errorf("expected 1 template matching combined filter, got %d", len(results))
		}
		
		if len(results) > 0 && results[0].ID != "t1" {
			t.Errorf("expected template t1, got %s", results[0].ID)
		}
	})
	
	t.Run("pagination with offset and limit", func(t *testing.T) {
		m := createPopulatedMarketplace()
		ctx := context.Background()
		
		// Get first page
		page1, err := m.SearchTemplates(ctx, &TemplateFilter{
			Limit:  2,
			Offset: 0,
		})
		
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		
		if len(page1) != 2 {
			t.Errorf("expected 2 templates in first page, got %d", len(page1))
		}
		
		// Get second page
		page2, err := m.SearchTemplates(ctx, &TemplateFilter{
			Limit:  2,
			Offset: 2,
		})
		
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		
		if len(page2) != 2 {
			t.Errorf("expected 2 templates in second page, got %d", len(page2))
		}
		
		// Get third page (should have 1 item)
		page3, err := m.SearchTemplates(ctx, &TemplateFilter{
			Limit:  2,
			Offset: 4,
		})
		
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		
		if len(page3) != 1 {
			t.Errorf("expected 1 template in third page, got %d", len(page3))
		}
		
		// Verify no overlap
		ids := make(map[string]bool)
		for _, tmpl := range append(append(page1, page2...), page3...) {
			if ids[tmpl.ID] {
				t.Errorf("duplicate template ID found: %s", tmpl.ID)
			}
			ids[tmpl.ID] = true
		}
	})
	
	t.Run("offset beyond results", func(t *testing.T) {
		m := createPopulatedMarketplace()
		ctx := context.Background()
		
		results, err := m.SearchTemplates(ctx, &TemplateFilter{
			Limit:  10,
			Offset: 100,
		})
		
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		
		if len(results) != 0 {
			t.Errorf("expected 0 templates with offset beyond range, got %d", len(results))
		}
	})
}

func TestMatchesFilter(t *testing.T) {
	m := &Marketplace{}
	
	t.Run("nil filter matches all", func(t *testing.T) {
		template := createSampleTemplate("test", "Test", "automation")
		
		if !m.matchesFilter(template, nil) {
			t.Error("expected nil filter to match")
		}
	})
	
	t.Run("category matching", func(t *testing.T) {
		template := createSampleTemplate("test", "Test", "automation")
		
		filter := &TemplateFilter{Category: "automation"}
		if !m.matchesFilter(template, filter) {
			t.Error("expected matching category to pass")
		}
		
		filter = &TemplateFilter{Category: "research"}
		if m.matchesFilter(template, filter) {
			t.Error("expected non-matching category to fail")
		}
	})
	
	t.Run("author matching", func(t *testing.T) {
		template := createSampleTemplate("test", "Test", "automation")
		template.Author.Username = "alice"
		
		filter := &TemplateFilter{Author: "alice"}
		if !m.matchesFilter(template, filter) {
			t.Error("expected matching author to pass")
		}
		
		filter = &TemplateFilter{Author: "bob"}
		if m.matchesFilter(template, filter) {
			t.Error("expected non-matching author to fail")
		}
	})
	
	t.Run("tag matching", func(t *testing.T) {
		template := createSampleTemplate("test", "Test", "automation")
		template.Tags = []string{"python", "automation", "testing"}
		
		// Single tag match
		filter := &TemplateFilter{Tags: []string{"python"}}
		if !m.matchesFilter(template, filter) {
			t.Error("expected single matching tag to pass")
		}
		
		// Multiple tags, at least one matches
		filter = &TemplateFilter{Tags: []string{"java", "python"}}
		if !m.matchesFilter(template, filter) {
			t.Error("expected at least one matching tag to pass")
		}
		
		// No matching tags
		filter = &TemplateFilter{Tags: []string{"java", "rust"}}
		if m.matchesFilter(template, filter) {
			t.Error("expected no matching tags to fail")
		}
	})
	
	t.Run("search term matching", func(t *testing.T) {
		template := createSampleTemplate("test", "Data Analysis Tool", "automation")
		template.Description = "A powerful tool for analyzing data"
		
		// Match in name
		filter := &TemplateFilter{SearchTerm: "Analysis"}
		if !m.matchesFilter(template, filter) {
			t.Error("expected search term in name to match")
		}
		
		// Match in description
		filter = &TemplateFilter{SearchTerm: "powerful"}
		if !m.matchesFilter(template, filter) {
			t.Error("expected search term in description to match")
		}
		
		// No match
		filter = &TemplateFilter{SearchTerm: "javascript"}
		if m.matchesFilter(template, filter) {
			t.Error("expected non-matching search term to fail")
		}
	})
	
	t.Run("star threshold", func(t *testing.T) {
		template := createSampleTemplate("test", "Test", "automation")
		template.Stars = 25
		
		filter := &TemplateFilter{MinStars: 20}
		if !m.matchesFilter(template, filter) {
			t.Error("expected template with enough stars to pass")
		}
		
		filter = &TemplateFilter{MinStars: 30}
		if m.matchesFilter(template, filter) {
			t.Error("expected template with too few stars to fail")
		}
	})
}