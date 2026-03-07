package extraction

import (
	"testing"
)

func TestDeepReader_FilterLinks(t *testing.T) {
	reader := NewDeepReader()

	tests := []struct {
		name      string
		baseURL   string
		links     []LinkInfo
		wantCount int // expected count after filtering (approximate)
	}{
		{
			name:    "filters login and auth links",
			baseURL: "https://example.com",
			links: []LinkInfo{
				{URL: "https://example.com/login", Text: "Sign In"},
				{URL: "https://example.com/register", Text: "Create Account"},
				{URL: "https://example.com/article/1", Text: "Interesting Article About Technology"},
			},
			wantCount: 1,
		},
		{
			name:    "filters social media links",
			baseURL: "https://example.com",
			links: []LinkInfo{
				{URL: "https://facebook.com/example", Text: "Follow us on Facebook"},
				{URL: "https://twitter.com/example", Text: "Twitter"},
				{URL: "https://example.com/page", Text: "About Our Company and Services"},
			},
			wantCount: 1,
		},
		{
			name:    "filters external domains when sameDomain is true",
			baseURL: "https://example.com",
			links: []LinkInfo{
				{URL: "https://other.com/page", Text: "External Link"},
				{URL: "https://example.com/page", Text: "Internal Link"},
			},
			wantCount: 1,
		},
		{
			name:    "filters short generic link texts",
			baseURL: "https://example.com",
			links: []LinkInfo{
				{URL: "https://example.com/page1", Text: "here"},
				{URL: "https://example.com/page2", Text: "click here"},
				{URL: "https://example.com/page3", Text: "read more"},
				{URL: "https://example.com/page4", Text: "Detailed Article Title"},
			},
			wantCount: 1,
		},
		{
			name:    "filters file downloads",
			baseURL: "https://example.com",
			links: []LinkInfo{
				{URL: "https://example.com/doc.pdf", Text: "Download PDF"},
				{URL: "https://example.com/file.zip", Text: "Download ZIP"},
				{URL: "https://example.com/article", Text: "Article Title"},
			},
			wantCount: 1,
		},
		{
			name:    "filters duplicates",
			baseURL: "https://example.com",
			links: []LinkInfo{
				{URL: "https://example.com/page", Text: "First Link"},
				{URL: "https://example.com/page", Text: "Second Link Same URL"},
			},
			wantCount: 1,
		},
		{
			name:    "filters javascript and mailto links",
			baseURL: "https://example.com",
			links: []LinkInfo{
				{URL: "javascript:void(0)", Text: "Click"},
				{URL: "mailto:test@example.com", Text: "Email Us"},
				{URL: "https://example.com/page", Text: "Valid Page Link"},
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := reader.filterLinks(tt.baseURL, tt.links)
			if len(filtered) != tt.wantCount {
				t.Errorf("filterLinks() returned %d links, want %d", len(filtered), tt.wantCount)
				for _, l := range filtered {
					t.Logf("  - %s: %s", l.URL, l.Text)
				}
			}
		})
	}
}

func TestDeepReader_FilterLinks_SameDomain(t *testing.T) {
	reader := NewDeepReader(WithSameDomain(true))

	links := []LinkInfo{
		{URL: "https://example.com/page1", Text: "Internal Page One"},
		{URL: "https://example.com/page2", Text: "Internal Page Two"},
		{URL: "https://other.com/page", Text: "External Page"},
	}

	filtered := reader.filterLinks("https://example.com", links)

	for _, l := range filtered {
		if l.URL == "https://other.com/page" {
			t.Error("filterLinks should exclude external domains when sameDomain is true")
		}
	}
}

func TestDeepReader_FilterLinks_CrossDomain(t *testing.T) {
	reader := NewDeepReader(WithSameDomain(false))

	links := []LinkInfo{
		{URL: "https://example.com/page1", Text: "Internal Page One"},
		{URL: "https://other.com/page", Text: "External Page With Long Title"},
	}

	filtered := reader.filterLinks("https://example.com", links)

	hasExternal := false
	for _, l := range filtered {
		if l.URL == "https://other.com/page" {
			hasExternal = true
		}
	}

	if !hasExternal {
		t.Error("filterLinks should include external domains when sameDomain is false")
	}
}

func TestDeepReader_MaxLinks(t *testing.T) {
	maxLinks := 3
	reader := NewDeepReader(WithMaxLinks(maxLinks))

	links := []LinkInfo{
		{URL: "https://example.com/page1", Text: "Page One Title"},
		{URL: "https://example.com/page2", Text: "Page Two Title"},
		{URL: "https://example.com/page3", Text: "Page Three Title"},
		{URL: "https://example.com/page4", Text: "Page Four Title"},
		{URL: "https://example.com/page5", Text: "Page Five Title"},
	}

	filtered := reader.filterLinks("https://example.com", links)

	if len(filtered) > maxLinks {
		t.Errorf("filterLinks returned %d links, should be at most %d", len(filtered), maxLinks)
	}
}

func TestDeepReader_Options(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		reader := NewDeepReader()
		if reader.maxLinks != 10 {
			t.Errorf("default maxLinks should be 10, got %d", reader.maxLinks)
		}
		if !reader.sameDomain {
			t.Error("default sameDomain should be true")
		}
		if reader.contentLimit != 2000 {
			t.Errorf("default contentLimit should be 2000, got %d", reader.contentLimit)
		}
	})

	t.Run("custom options", func(t *testing.T) {
		reader := NewDeepReader(
			WithMaxLinks(5),
			WithSameDomain(false),
			WithContentLimit(1000),
		)
		if reader.maxLinks != 5 {
			t.Errorf("maxLinks should be 5, got %d", reader.maxLinks)
		}
		if reader.sameDomain {
			t.Error("sameDomain should be false")
		}
		if reader.contentLimit != 1000 {
			t.Errorf("contentLimit should be 1000, got %d", reader.contentLimit)
		}
	})

	t.Run("maxLinks boundary", func(t *testing.T) {
		reader := NewDeepReader(WithMaxLinks(100))
		if reader.maxLinks > 20 {
			t.Errorf("maxLinks should be capped at 20, got %d", reader.maxLinks)
		}
	})
}

func TestDeepReadResult_ToMarkdown(t *testing.T) {
	result := &DeepReadResult{
		MainURL:     "https://example.com",
		MainTitle:   "Example Page",
		MainContent: "This is the main content.",
		SubPages: []SubPageResult{
			{
				URL:      "https://example.com/page1",
				Title:    "Sub Page One",
				Content:  "Content of sub page one.",
				LinkText: "Link to Page One",
			},
			{
				URL:      "https://example.com/page2",
				Title:    "",
				Content:  "Content of sub page two.",
				LinkText: "Link to Page Two",
				Error:    "",
			},
			{
				URL:      "https://example.com/page3",
				LinkText: "Failed Link",
				Error:    "connection refused",
			},
		},
		TotalLinks:   10,
		CrawledLinks: 3,
	}

	markdown := result.ToMarkdown()

	// Check main content
	if markdown == "" {
		t.Fatal("ToMarkdown returned empty string")
	}

	// Check for key elements
	checks := []string{
		"[Example Page](https://example.com)",
		"This is the main content",
		"## Related Pages",
		"[Link to Page One](https://example.com/page1)",
		"Content of sub page one",
		"[Link to Page Two](https://example.com/page2)",
		"Error: connection refused",
		"Crawled 3 of 10 total links",
	}

	for _, check := range checks {
		if !contains(markdown, check) {
			t.Errorf("Markdown missing expected content: %q", check)
		}
	}
}

func TestDeepReader_ParseLinksFromJSON(t *testing.T) {
	reader := NewDeepReader()

	jsonStr := `{"content":"Some content","links":[{"url":"https://example.com/page1","text":"Page One","type":"link"},{"url":"https://example.com/page2","text":"Page Two","type":"link"}]}`

	links := reader.parseLinksFromJSON(jsonStr)

	if len(links) != 2 {
		t.Errorf("expected 2 links, got %d", len(links))
	}

	if links[0].URL != "https://example.com/page1" {
		t.Errorf("expected URL 'https://example.com/page1', got %q", links[0].URL)
	}
	if links[0].Text != "Page One" {
		t.Errorf("expected Text 'Page One', got %q", links[0].Text)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
