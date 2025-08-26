package builtin

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/liliang-cn/rago/internal/tools"
)

// OpenURLTool provides simple URL opening functionality
type OpenURLTool struct {
	timeout time.Duration
}

// OpenURLConfig contains configuration for the open URL tool
type OpenURLConfig struct {
	Timeout time.Duration `json:"timeout"`
}

// NewOpenURLTool creates a new open URL tool instance
func NewOpenURLTool(config OpenURLConfig) *OpenURLTool {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &OpenURLTool{
		timeout: config.Timeout,
	}
}

// Name returns the tool name
func (o *OpenURLTool) Name() string {
	return "open_url"
}

// Description returns the tool description
func (o *OpenURLTool) Description() string {
	return "Fetch and parse web page content using Chrome browser. Returns extracted text, title, and structured content."
}

// Parameters returns the tool parameters schema
func (o *OpenURLTool) Parameters() tools.ToolParameters {
	return tools.ToolParameters{
		Type: "object",
		Properties: map[string]tools.ToolParameter{
			"url": {
				Type:        "string",
				Description: "The web link URL to open",
			},
			"extract_content": {
				Type:        "boolean",
				Description: "Whether to extract and parse page content",
				Default:     true,
			},
		},
		Required: []string{"url"},
	}
}

// Validate validates the tool arguments
func (o *OpenURLTool) Validate(args map[string]interface{}) error {
	urlStr, ok := args["url"].(string)
	if !ok || strings.TrimSpace(urlStr) == "" {
		return fmt.Errorf("url is required and must be a non-empty string")
	}

	// Validate URL format
	parsedURL, err := url.Parse(strings.TrimSpace(urlStr))
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Check if it's a valid HTTP/HTTPS URL
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("only HTTP and HTTPS URLs are supported")
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("URL must have a valid host")
	}

	return nil
}

// Execute performs the URL fetching and content extraction
func (o *OpenURLTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	urlStr := strings.TrimSpace(args["url"].(string))
	extractContent := true
	if ec, ok := args["extract_content"].(bool); ok {
		extractContent = ec
	}

	// Parse and validate the URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return &tools.ToolResult{
			Data: map[string]interface{}{
				"error":   fmt.Sprintf("URL parsing failed: %v", err),
				"url":     urlStr,
				"success": false,
			},
		}, fmt.Errorf("URL parsing failed: %w", err)
	}

	cleanURL := parsedURL.String()
	
	if !extractContent {
		return &tools.ToolResult{
			Data: map[string]interface{}{
				"url":     cleanURL,
				"host":    parsedURL.Host,
				"scheme":  parsedURL.Scheme,
				"message": fmt.Sprintf("URL prepared: %s", cleanURL),
				"success": true,
			},
		}, nil
	}

	// Use chromedp to fetch and parse content
	content, title, err := o.fetchPageContent(ctx, cleanURL)
	if err != nil {
		return &tools.ToolResult{
			Data: map[string]interface{}{
				"error":   fmt.Sprintf("Failed to fetch content: %v", err),
				"url":     cleanURL,
				"success": false,
			},
		}, fmt.Errorf("content fetch failed: %w", err)
	}

	// Extract and clean text content
	textContent := o.extractTextContent(content)
	
	return &tools.ToolResult{
		Data: map[string]interface{}{
			"url":          cleanURL,
			"title":        title,
			"host":         parsedURL.Host,
			"scheme":       parsedURL.Scheme,
			"content":      textContent,
			"raw_html":     content,
			"content_size": len(textContent),
			"message":      fmt.Sprintf("Successfully fetched content from: %s", cleanURL),
			"success":      true,
		},
	}, nil
}

// fetchPageContent uses chromedp to get page content and title
func (o *OpenURLTool) fetchPageContent(ctx context.Context, urlStr string) (string, string, error) {
	// Create chromedp context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, o.timeout)
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
	var title string
	err := chromedp.Run(browserCtx,
		chromedp.Navigate(urlStr),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.OuterHTML("html", &content),
		chromedp.Title(&title),
	)

	if err != nil {
		return "", "", fmt.Errorf("chromedp failed: %w", err)
	}

	// Limit content size to avoid too much data
	maxSize := 500000 // 500KB limit
	if len(content) > maxSize {
		content = content[:maxSize] + "\n\n...[Content too long and truncated, kept first " + fmt.Sprintf("%d", maxSize/1000) + "KB]"
	}

	return content, title, nil
}

// extractTextContent extracts clean text content from HTML
func (o *OpenURLTool) extractTextContent(html string) string {
	// Remove script and style tags with their content
	scriptRe := regexp.MustCompile(`<script[^>]*>[\s\S]*?</script>`)
	html = scriptRe.ReplaceAllString(html, "")
	
	styleRe := regexp.MustCompile(`<style[^>]*>[\s\S]*?</style>`)
	html = styleRe.ReplaceAllString(html, "")

	// Remove HTML tags
	tagRe := regexp.MustCompile(`<[^>]*>`)
	text := tagRe.ReplaceAllString(html, " ")
	
	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&mdash;", "—")
	text = strings.ReplaceAll(text, "&ndash;", "–")
	
	// Clean up whitespace
	spaceRe := regexp.MustCompile(`\s+`)
	text = spaceRe.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)
	
	// Limit text size
	maxTextSize := 100000 // 100KB text limit
	if len(text) > maxTextSize {
		text = text[:maxTextSize] + "\n\n...[Text content truncated]"
	}
	
	return text
}

