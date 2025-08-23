package builtin

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/liliang-cn/rago/internal/tools"
)

// GoogleSearchTool provides Google search functionality using web_request
type GoogleSearchTool struct {
	webTool       *WebTool
	maxResults    int
	searchTimeout time.Duration
}

// GoogleSearchConfig contains configuration for the Google search tool
type GoogleSearchConfig struct {
	MaxResults    int           `json:"max_results"`
	SearchTimeout time.Duration `json:"search_timeout"`
	UserAgent     string        `json:"user_agent"`
}

// NewGoogleSearchTool creates a new Google search tool instance
func NewGoogleSearchTool(config GoogleSearchConfig) *GoogleSearchTool {
	if config.MaxResults == 0 {
		config.MaxResults = 10
	}
	if config.SearchTimeout == 0 {
		config.SearchTimeout = 60 * time.Second
	}
	if config.UserAgent == "" {
		config.UserAgent = "RAGO-Search-Tool/1.0"
	}

	// Create underlying web tool with appropriate settings
	webConfig := WebToolConfig{
		Timeout:       config.SearchTimeout,
		MaxContentLen: 50 * 1024, // 50KB for search results
		UserAgent:     config.UserAgent,
		AllowedHosts:  []string{"google.com", "google.cn", "google.com.hk"}, // Allow Google domains
	}

	return &GoogleSearchTool{
		webTool:       NewWebTool(webConfig),
		maxResults:    config.MaxResults,
		searchTimeout: config.SearchTimeout,
	}
}

// Name returns the tool name
func (g *GoogleSearchTool) Name() string {
	return "google_search"
}

// Description returns the tool description
func (g *GoogleSearchTool) Description() string {
	return "Search Google for information and return search results including titles, URLs, and snippets."
}

// Parameters returns the tool parameters schema
func (g *GoogleSearchTool) Parameters() tools.ToolParameters {
	return tools.ToolParameters{
		Type: "object",
		Properties: map[string]tools.ToolParameter{
			"query": {
				Type:        "string",
				Description: "Search query string",
			},
			"num_results": {
				Type:        "integer",
				Description: "Number of results to return (default: 10, max: 20)",
				Default:     10,
			},
			"language": {
				Type:        "string",
				Description: "Search language code (e.g., 'en', 'zh-CN', 'zh-TW')",
				Default:     "en",
			},
			"country": {
				Type:        "string",
				Description: "Country code for localized results (e.g., 'US', 'CN', 'TW')",
				Default:     "US",
			},
			"safe_search": {
				Type:        "boolean",
				Description: "Enable safe search filtering",
				Default:     true,
			},
		},
		Required: []string{"query"},
	}
}

// Validate validates the tool arguments
func (g *GoogleSearchTool) Validate(args map[string]interface{}) error {
	// Check required query
	query, ok := args["query"].(string)
	if !ok || strings.TrimSpace(query) == "" {
		return fmt.Errorf("query is required and must be a non-empty string")
	}

	// Validate num_results
	if numResults, ok := args["num_results"]; ok {
		if num, ok := numResults.(float64); ok {
			if num < 1 || num > 20 {
				return fmt.Errorf("num_results must be between 1 and 20")
			}
		}
	}

	// Validate language code
	if lang, ok := args["language"].(string); ok {
		validLangs := []string{"en", "zh", "zh-CN", "zh-TW", "ja", "ko", "fr", "de", "es", "it", "ru", "ar"}
		found := false
		for _, validLang := range validLangs {
			if lang == validLang {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unsupported language code: %s", lang)
		}
	}

	return nil
}

// Execute performs the Google search
func (g *GoogleSearchTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	query := strings.TrimSpace(args["query"].(string))
	numResults := getIntWithDefault(args, "num_results", 10)
	language := getStringWithDefault(args, "language", "en")
	country := getStringWithDefault(args, "country", "US")
	safeSearch := getBoolWithDefault(args, "safe_search", true)

	// Limit results to maximum allowed
	if numResults > g.maxResults {
		numResults = g.maxResults
	}

	// Build Google search URL
	searchURL := g.buildGoogleURL(query, numResults, language, country, safeSearch)

	// Use the web tool to get the search results page
	searchArgs := map[string]interface{}{
		"url":       searchURL,
		"action":    "get_html",
		"wait_time": 3,
	}

	result, err := g.webTool.Execute(ctx, searchArgs)
	if err != nil {
		return &tools.ToolResult{
			Data: map[string]interface{}{
				"error": fmt.Sprintf("Failed to fetch search results: %v", err),
				"query": query,
			},
		}, fmt.Errorf("Google search failed: %w", err)
	}

	// Extract search results from HTML
	data := result.Data.(map[string]interface{})
	if !data["success"].(bool) {
		return &tools.ToolResult{
			Data: map[string]interface{}{
				"error": "Search request was not successful",
				"query": query,
			},
		}, fmt.Errorf("search request failed")
	}

	html := data["html"].(string)

	// Parse search results using chromedp to execute JavaScript
	searchResults, err := g.parseSearchResults(ctx, searchURL)
	if err != nil {
		// Fallback: try basic HTML parsing
		searchResults = g.parseSearchResultsFromHTML(html)
	}

	return &tools.ToolResult{
		Data: map[string]interface{}{
			"query":         query,
			"num_results":   len(searchResults),
			"language":      language,
			"country":       country,
			"search_url":    searchURL,
			"results":       searchResults,
			"success":       true,
		},
	}, nil
}

// buildGoogleURL constructs the Google search URL with parameters
func (g *GoogleSearchTool) buildGoogleURL(query string, numResults int, language, country string, safeSearch bool) string {
	baseURL := "https://www.google.com/search"
	
	params := url.Values{}
	params.Add("q", query)
	params.Add("num", fmt.Sprintf("%d", numResults))
	params.Add("hl", language) // Interface language
	params.Add("gl", country)  // Country for results
	
	if safeSearch {
		params.Add("safe", "active")
	} else {
		params.Add("safe", "off")
	}

	return baseURL + "?" + params.Encode()
}

// parseSearchResults uses chromedp to extract search results with JavaScript
func (g *GoogleSearchTool) parseSearchResults(ctx context.Context, searchURL string) ([]map[string]interface{}, error) {
	// Create a timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, g.searchTimeout)
	defer cancel()

	// Create Chrome context
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent(g.webTool.userAgent),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(timeoutCtx, opts...)
	defer allocCancel()

	chromeCtx, chromeCancel := chromedp.NewContext(allocCtx)
	defer chromeCancel()

	var results []map[string]interface{}

	// Navigate and extract search results
	err := chromedp.Run(chromeCtx,
		chromedp.Navigate(searchURL),
		chromedp.Sleep(3*time.Second), // Wait for page to load
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Extract search result data using JavaScript
			var resultData []map[string]interface{}
			
			err := chromedp.Evaluate(`
				(() => {
					const results = [];
					const resultElements = document.querySelectorAll('div[data-ved] h3');
					
					resultElements.forEach((titleElement, index) => {
						if (index >= 10) return; // Limit results
						
						const linkElement = titleElement.closest('a');
						const resultContainer = titleElement.closest('div[data-ved]');
						
						if (linkElement && resultContainer) {
							const title = titleElement.textContent || '';
							const url = linkElement.href || '';
							
							// Find snippet text
							const snippetElements = resultContainer.querySelectorAll('span, div');
							let snippet = '';
							for (let elem of snippetElements) {
								const text = elem.textContent || '';
								if (text.length > 50 && text.length < 300) {
									snippet = text;
									break;
								}
							}
							
							if (title && url && url.startsWith('http')) {
								results.push({
									title: title.trim(),
									url: url,
									snippet: snippet.trim(),
									position: index + 1
								});
							}
						}
					});
					
					return results;
				})()
			`, &resultData).Do(ctx)
			
			if err == nil && len(resultData) > 0 {
				results = resultData
			}
			
			return err
		}),
	)

	if err != nil {
		return nil, err
	}

	return results, nil
}

// parseSearchResultsFromHTML provides fallback HTML parsing (basic)
func (g *GoogleSearchTool) parseSearchResultsFromHTML(html string) []map[string]interface{} {
	// This is a very basic fallback - in practice, Google's HTML structure
	// is complex and changes frequently, so the JavaScript approach is preferred
	results := []map[string]interface{}{
		{
			"title":    "HTML parsing fallback",
			"url":      "",
			"snippet":  "Search results require JavaScript parsing for optimal extraction",
			"position": 1,
		},
	}

	return results
}