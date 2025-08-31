package builtin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/liliang-cn/rago/pkg/tools"
)

// WebSearchTool provides web search functionality using DuckDuckGo HTML search + chromedp
type WebSearchTool struct {
	client     *http.Client
	maxResults int
	timeout    time.Duration
}

// WebSearchConfig contains configuration for the web search tool
type WebSearchConfig struct {
	MaxResults int           `json:"max_results"`
	Timeout    time.Duration `json:"timeout"`
}

// SearchResult represents a single search result
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Content string `json:"content,omitempty"` // Full page content from chromedp
}

// NewWebSearchTool creates a new web search tool instance
func NewWebSearchTool(config WebSearchConfig) *WebSearchTool {
	if config.MaxResults == 0 {
		config.MaxResults = 3 // Reduced since we're fetching full content
	}
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second // Increased for chromedp
	}

	return &WebSearchTool{
		client: &http.Client{
			Timeout: config.Timeout,
		},
		maxResults: config.MaxResults,
		timeout:    config.Timeout,
	}
}

// Name returns the tool name
func (w *WebSearchTool) Name() string {
	return "web_search"
}

// Description returns the tool description
func (w *WebSearchTool) Description() string {
	return "Search web information and get complete page content, supports queries for prices, news, technical questions, etc."
}

// Parameters returns the tool parameters schema
func (w *WebSearchTool) Parameters() tools.ToolParameters {
	return tools.ToolParameters{
		Type: "object",
		Properties: map[string]tools.ToolParameter{
			"query": {
				Type:        "string",
				Description: "Search keywords or question",
			},
			"max_pages": {
				Type:        "integer",
				Description: "Maximum number of page contents to fetch (1-5)",
				Default:     2,
			},
		},
		Required: []string{"query"},
	}
}

// Validate validates the tool arguments
func (w *WebSearchTool) Validate(args map[string]interface{}) error {
	query, ok := args["query"].(string)
	if !ok || strings.TrimSpace(query) == "" {
		return fmt.Errorf("query is required and must be a non-empty string")
	}
	return nil
}

// Execute performs the web search and fetches page content
func (w *WebSearchTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	query := strings.TrimSpace(args["query"].(string))
	maxPages := 2
	if mp, ok := args["max_pages"].(float64); ok && mp >= 1 && mp <= 5 {
		maxPages = int(mp)
	}

	// Step 1: Search DuckDuckGo HTML to get URLs
	searchResults, err := w.searchDuckDuckGoHTML(ctx, query)
	if err != nil {
		return &tools.ToolResult{
			Data: map[string]interface{}{
				"error":   err.Error(),
				"query":   query,
				"success": false,
			},
		}, fmt.Errorf("search failed: %w", err)
	}

	if len(searchResults) == 0 {
		return &tools.ToolResult{
			Data: map[string]interface{}{
				"query":   query,
				"message": "No relevant search results found",
				"success": false,
			},
		}, nil
	}

	// Step 2: Use chromedp to fetch content from top results
	if len(searchResults) > maxPages {
		searchResults = searchResults[:maxPages]
	}

	for i := range searchResults {
		content, err := w.fetchPageContent(ctx, searchResults[i].URL)
		if err != nil {
			searchResults[i].Content = fmt.Sprintf("Unable to fetch page content: %v", err)
		} else {
			searchResults[i].Content = content
		}
	}

	// Format results
	results := w.formatResults(searchResults, query)

	return &tools.ToolResult{
		Data: map[string]interface{}{
			"query":     query,
			"results":   results,
			"message":   fmt.Sprintf("Searched '%s' and fetched complete content from %d pages", query, len(searchResults)),
			"success":   true,
		},
	}, nil
}

// fetchPageContent uses chromedp to get page content
func (w *WebSearchTool) fetchPageContent(ctx context.Context, urlStr string) (string, error) {
	// Create chromedp context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Create allocator context
	allocCtx, allocCancel := chromedp.NewExecAllocator(timeoutCtx,
		chromedp.NoSandbox,
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)
	defer allocCancel()

	// Create browser context
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	var content string
	err := chromedp.Run(browserCtx,
		chromedp.Navigate(urlStr),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.OuterHTML("html", &content),
	)

	if err != nil {
		return "", fmt.Errorf("chromedp failed: %w", err)
	}

	// Limit content size to avoid too much data, but increase limit
	maxSize := 200000 // 200KB limit - increased for better content extraction
	if len(content) > maxSize {
		// Try to keep more useful content by truncating from the end
		content = content[:maxSize] + "\n\n...[Content too long and truncated, kept first " + fmt.Sprintf("%d", maxSize/1000) + "KB]"
	}

	return content, nil
}

// searchDuckDuckGoHTML searches using DuckDuckGo HTML interface
func (w *WebSearchTool) searchDuckDuckGoHTML(ctx context.Context, query string) ([]SearchResult, error) {
	// Build search URL
	baseURL := "https://html.duckduckgo.com/html/"
	params := url.Values{}
	params.Set("q", query)
	
	fullURL := baseURL + "?" + params.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent to mimic a real browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	// Make request
	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
		defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Warning: failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP returned status code: %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse HTML results
	results := w.parseHTMLResults(string(body))
	
	// Limit results
	if len(results) > w.maxResults {
		results = results[:w.maxResults]
	}

	return results, nil
}

// parseHTMLResults extracts search results from HTML
func (w *WebSearchTool) parseHTMLResults(html string) []SearchResult {
	var results []SearchResult

	// More flexible regex patterns for DuckDuckGo HTML results
	// Look for result titles and URLs
	titleURLPattern := regexp.MustCompile(`<a[^>]*class="result__a"[^>]*href="([^"]+)"[^>]*>([^<]+)</a>`)
	snippetPattern := regexp.MustCompile(`<a[^>]*class="result__snippet"[^>]*[^>]*>([^<]+)</a>`)

	// Find all title/URL matches
	titleMatches := titleURLPattern.FindAllStringSubmatch(html, -1)
	snippetMatches := snippetPattern.FindAllStringSubmatch(html, -1)

	// Create a map of snippets for easier lookup
	snippets := make(map[int]string)
	for i, match := range snippetMatches {
		if len(match) >= 2 {
			snippets[i] = w.cleanText(match[1])
		}
	}

	for i, match := range titleMatches {
		if len(match) < 3 {
			continue
		}

		url := w.cleanURL(match[1])
		title := w.cleanText(match[2])

		snippet := ""
		if s, exists := snippets[i]; exists {
			snippet = s
		}

		if title != "" && url != "" && !strings.Contains(url, "duckduckgo.com") {
			results = append(results, SearchResult{
				Title:   title,
				URL:     url,
				Snippet: snippet,
			})
		}
	}

	return results
}

// cleanURL cleans and validates URLs
func (w *WebSearchTool) cleanURL(rawURL string) string {
	// DuckDuckGo uses redirect URLs starting with //duckduckgo.com/l/?uddg=
	if strings.HasPrefix(rawURL, "//duckduckgo.com/l/?uddg=") {
		// Extract the actual URL from the uddg parameter
		if idx := strings.Index(rawURL, "uddg="); idx != -1 {
			encoded := rawURL[idx+5:]
			if idx2 := strings.Index(encoded, "&"); idx2 != -1 {
				encoded = encoded[:idx2]
			}
			if decoded, err := url.QueryUnescape(encoded); err == nil {
				return decoded
			}
		}
	}
	
	// Clean up the URL
	cleanURL := strings.TrimSpace(rawURL)
	if !strings.HasPrefix(cleanURL, "http") && !strings.HasPrefix(cleanURL, "//") {
		return "https://" + cleanURL
	}
	
	if strings.HasPrefix(cleanURL, "//") {
		return "https:" + cleanURL
	}
	
	return cleanURL
}

// cleanText removes HTML tags and cleans text
func (w *WebSearchTool) cleanText(text string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	cleaned := re.ReplaceAllString(text, "")
	
	// Decode HTML entities
	cleaned = strings.ReplaceAll(cleaned, "&amp;", "&")
	cleaned = strings.ReplaceAll(cleaned, "&lt;", "<")
	cleaned = strings.ReplaceAll(cleaned, "&gt;", ">")
	cleaned = strings.ReplaceAll(cleaned, "&quot;", "\"")
	cleaned = strings.ReplaceAll(cleaned, "&#39;", "'")
	cleaned = strings.ReplaceAll(cleaned, "&nbsp;", " ")
	
	// Clean up whitespace
	cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")
	cleaned = strings.TrimSpace(cleaned)
	
	return cleaned
}

// formatResults formats the search results for presentation
func (w *WebSearchTool) formatResults(results []SearchResult, query string) map[string]interface{} {
	if len(results) == 0 {
		return map[string]interface{}{
			"query":   query,
			"count":   0,
			"summary": "No relevant search results found",
			"results": []SearchResult{},
		}
	}

	// Create summary with page contents
	summary := fmt.Sprintf("Searched '%s' and found %d results, fetched complete page content for analysis:\n\n", query, len(results))
	
	pageContents := make([]map[string]interface{}, 0, len(results))
	var fullContent string // Combine all HTML content for LLM analysis
	
	for i, result := range results {
		summary += fmt.Sprintf("%d. Title: %s\n   Link: %s\n   Summary: %s\n\n", i+1, result.Title, result.URL, result.Snippet)
		
		// Add HTML content to summary for direct LLM access
		if result.Content != "" {
			summary += fmt.Sprintf("   Page HTML Content:\n%s\n\n", result.Content)
			fullContent += fmt.Sprintf("\n=== Page %d: %s ===\n%s\n", i+1, result.Title, result.Content)
		}
		
		pageContents = append(pageContents, map[string]interface{}{
			"title":   result.Title,
			"url":     result.URL,
			"snippet": result.Snippet,
			"content": result.Content,
		})
	}

	return map[string]interface{}{
		"query":         query,
		"count":         len(results),
		"summary":       summary,
		"results":       results,
		"page_contents": pageContents, // Full content for LLM analysis
		"full_html":     fullContent,  // All HTML content combined for easy LLM access
	}
}

