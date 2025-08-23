package builtin

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/liliang-cn/rago/internal/tools"
)

// DuckDuckGoSearchTool provides DuckDuckGo search functionality using HTTP requests
type DuckDuckGoSearchTool struct {
	httpTool      *HTTPTool
	maxResults    int
	searchTimeout time.Duration
}

// DuckDuckGoSearchConfig contains configuration for the DuckDuckGo search tool
type DuckDuckGoSearchConfig struct {
	MaxResults    int           `json:"max_results"`
	SearchTimeout time.Duration `json:"search_timeout"`
	UserAgent     string        `json:"user_agent"`
	SafeSearch    string        `json:"safe_search"` // "strict", "moderate", "off"
}

// NewDuckDuckGoSearchTool creates a new DuckDuckGo search tool instance
func NewDuckDuckGoSearchTool(config DuckDuckGoSearchConfig) *DuckDuckGoSearchTool {
	if config.MaxResults == 0 {
		config.MaxResults = 10
	}
	if config.SearchTimeout == 0 {
		config.SearchTimeout = 30 * time.Second
	}
	if config.UserAgent == "" {
		config.UserAgent = "RAGO-DuckDuckGo-Tool/1.0"
	}
	if config.SafeSearch == "" {
		config.SafeSearch = "moderate"
	}

	// Create underlying HTTP tool
	httpConfig := HTTPToolConfig{
		Timeout:        config.SearchTimeout,
		MaxBodySize:    200 * 1024, // 200KB for search results
		UserAgent:      config.UserAgent,
		FollowRedirect: true,
		AllowedHosts:   []string{"duckduckgo.com", "html.duckduckgo.com"}, // Allow DuckDuckGo domains
	}

	return &DuckDuckGoSearchTool{
		httpTool:      NewHTTPTool(httpConfig),
		maxResults:    config.MaxResults,
		searchTimeout: config.SearchTimeout,
	}
}

// Name returns the tool name
func (d *DuckDuckGoSearchTool) Name() string {
	return "duckduckgo_search"
}

// Description returns the tool description
func (d *DuckDuckGoSearchTool) Description() string {
	return "Search DuckDuckGo for information and return search results including titles, URLs, and snippets. Privacy-focused search engine that doesn't track users."
}

// Parameters returns the tool parameters schema
func (d *DuckDuckGoSearchTool) Parameters() tools.ToolParameters {
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
			"region": {
				Type:        "string",
				Description: "Region for localized results (e.g., 'us-en', 'cn-zh', 'tw-tzh')",
				Default:     "us-en",
			},
			"safe_search": {
				Type:        "string",
				Description: "Safe search setting",
				Enum:        []string{"strict", "moderate", "off"},
				Default:     "moderate",
			},
		},
		Required: []string{"query"},
	}
}

// Validate validates the tool arguments
func (d *DuckDuckGoSearchTool) Validate(args map[string]interface{}) error {
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

	// Validate safe_search
	if safeSearch, ok := args["safe_search"].(string); ok {
		validValues := []string{"strict", "moderate", "off"}
		found := false
		for _, valid := range validValues {
			if safeSearch == valid {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("safe_search must be one of: strict, moderate, off")
		}
	}

	return nil
}

// Execute performs the DuckDuckGo search
func (d *DuckDuckGoSearchTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	query := strings.TrimSpace(args["query"].(string))
	numResults := getIntWithDefault(args, "num_results", 10)
	region := getStringWithDefault(args, "region", "us-en")
	safeSearch := getStringWithDefault(args, "safe_search", "moderate")

	// Limit results to maximum allowed
	if numResults > d.maxResults {
		numResults = d.maxResults
	}

	// Build DuckDuckGo search URL
	searchURL := d.buildDuckDuckGoURL(query, region, safeSearch)

	// Use HTTP tool to get the search results page
	searchArgs := map[string]interface{}{
		"url":    searchURL,
		"method": "GET",
		"headers": map[string]interface{}{
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
			"Accept-Language": "en-US,en;q=0.9",
			"Cache-Control":   "no-cache",
		},
	}

	result, err := d.httpTool.Execute(ctx, searchArgs)
	if err != nil {
		return &tools.ToolResult{
			Data: map[string]interface{}{
				"error": fmt.Sprintf("Failed to fetch search results: %v", err),
				"query": query,
			},
		}, fmt.Errorf("DuckDuckGo search failed: %w", err)
	}

	// Extract search results from HTTP response
	data := result.Data.(map[string]interface{})
	if !data["success"].(bool) {
		return &tools.ToolResult{
			Data: map[string]interface{}{
				"error": "Search request was not successful",
				"query": query,
			},
		}, fmt.Errorf("search request failed")
	}

	htmlContent := data["body"].(string)

	// Parse search results from HTML
	searchResults := d.parseSearchResults(htmlContent, numResults)

	return &tools.ToolResult{
		Data: map[string]interface{}{
			"query":       query,
			"num_results": len(searchResults),
			"region":      region,
			"safe_search": safeSearch,
			"search_url":  searchURL,
			"results":     searchResults,
			"success":     true,
		},
	}, nil
}

// buildDuckDuckGoURL constructs the DuckDuckGo search URL with parameters
func (d *DuckDuckGoSearchTool) buildDuckDuckGoURL(query, region, safeSearch string) string {
	baseURL := "https://html.duckduckgo.com/html"
	
	params := url.Values{}
	params.Add("q", query)
	params.Add("kl", region) // Region/language
	
	// Safe search mapping
	switch safeSearch {
	case "strict":
		params.Add("safe", "strict")
	case "off":
		params.Add("safe", "-1")
	default: // moderate
		params.Add("safe", "moderate")
	}

	return baseURL + "?" + params.Encode()
}

// parseSearchResults extracts search results from DuckDuckGo HTML
func (d *DuckDuckGoSearchTool) parseSearchResults(html string, maxResults int) []map[string]interface{} {
	var results []map[string]interface{}

	// DuckDuckGo HTML structure patterns (these are simplified and may need updates)
	// Look for result links with specific patterns
	
	// Pattern for result titles and links
	titlePattern := regexp.MustCompile(`<a class="result__a" href="([^"]+)">([^<]+)</a>`)
	titleMatches := titlePattern.FindAllStringSubmatch(html, maxResults)
	
	// Pattern for result snippets
	snippetPattern := regexp.MustCompile(`<a class="result__snippet">([^<]+)</a>`)
	snippetMatches := snippetPattern.FindAllStringSubmatch(html, maxResults)

	// Alternative patterns for different DuckDuckGo layouts
	if len(titleMatches) == 0 {
		// Try alternative pattern
		titlePattern = regexp.MustCompile(`href="([^"]+)"[^>]*>([^<]+)</a>`)
		titleMatches = titlePattern.FindAllStringSubmatch(html, maxResults*2) // Get more to filter
	}

	// Combine results
	for i, match := range titleMatches {
		if len(match) >= 3 && i < maxResults {
			resultURL := match[1]
			title := strings.TrimSpace(match[2])
			
			// Filter out DuckDuckGo internal links
			if strings.Contains(resultURL, "duckduckgo.com") {
				continue
			}
			
			// Clean up URL (remove DuckDuckGo redirect if present)
			if strings.HasPrefix(resultURL, "/l/?uddg=") {
				// This is a DuckDuckGo redirect, try to extract real URL
				if decodedURL, err := url.QueryUnescape(resultURL); err == nil {
					resultURL = decodedURL
				}
			}
			
			snippet := ""
			if i < len(snippetMatches) && len(snippetMatches[i]) >= 2 {
				snippet = strings.TrimSpace(snippetMatches[i][1])
			}
			
			// Skip if title is too short or looks like navigation
			if len(title) < 10 || strings.Contains(strings.ToLower(title), "duckduckgo") {
				continue
			}
			
			results = append(results, map[string]interface{}{
				"title":    title,
				"url":      resultURL,
				"snippet":  snippet,
				"position": len(results) + 1,
			})
		}
	}

	// If regex parsing didn't work well, provide a fallback result
	if len(results) == 0 {
		results = append(results, map[string]interface{}{
			"title":    "Search completed",
			"url":      "",
			"snippet":  "DuckDuckGo search was performed but results parsing may need refinement for current HTML structure",
			"position": 1,
		})
	}

	return results
}