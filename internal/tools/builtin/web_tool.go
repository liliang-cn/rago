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

// WebTool handles web page requests using headless Chrome via chromedp
type WebTool struct {
	timeout       time.Duration
	allowedHosts  []string
	blockedHosts  []string
	maxContentLen int
	userAgent     string
}

// WebToolConfig contains configuration for the web tool
type WebToolConfig struct {
	Timeout       time.Duration `json:"timeout"`
	AllowedHosts  []string      `json:"allowed_hosts"`
	BlockedHosts  []string      `json:"blocked_hosts"`
	MaxContentLen int           `json:"max_content_length"`
	UserAgent     string        `json:"user_agent"`
}

// NewWebTool creates a new web tool instance
func NewWebTool(config WebToolConfig) *WebTool {
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}
	if config.MaxContentLen == 0 {
		config.MaxContentLen = 100 * 1024 // 100KB default
	}
	if config.UserAgent == "" {
		config.UserAgent = "RAGO-Web-Tool/1.0"
	}

	return &WebTool{
		timeout:       config.Timeout,
		allowedHosts:  config.AllowedHosts,
		blockedHosts:  config.BlockedHosts,
		maxContentLen: config.MaxContentLen,
		userAgent:     config.UserAgent,
	}
}

// Name returns the tool name
func (w *WebTool) Name() string {
	return "web_request"
}

// Description returns the tool description
func (w *WebTool) Description() string {
	return "Fetch and extract content from web pages using a headless browser. Supports JavaScript rendering, text extraction, screenshots, and page interactions."
}

// Parameters returns the tool parameters schema
func (w *WebTool) Parameters() tools.ToolParameters {
	return tools.ToolParameters{
		Type: "object",
		Properties: map[string]tools.ToolParameter{
			"action": {
				Type:        "string",
				Description: "Action to perform on the web page",
				Enum:        []string{"get_text", "get_title", "screenshot", "click", "get_html", "get_links"},
				Default:     "get_text",
			},
			"url": {
				Type:        "string",
				Description: "Target URL to visit",
			},
			"selector": {
				Type:        "string",
				Description: "CSS selector for element-specific actions (optional)",
			},
			"wait_for": {
				Type:        "string",
				Description: "CSS selector to wait for before performing action",
			},
			"wait_time": {
				Type:        "integer",
				Description: "Time to wait in seconds before performing action",
				Default:     3,
			},
			"full_page": {
				Type:        "boolean",
				Description: "For screenshots, capture full page instead of viewport",
				Default:     false,
			},
		},
		Required: []string{"url", "action"},
	}
}

// Validate validates the tool arguments
func (w *WebTool) Validate(args map[string]interface{}) error {
	// Check required URL
	urlStr, ok := args["url"].(string)
	if !ok || urlStr == "" {
		return fmt.Errorf("url is required and must be a string")
	}

	// Validate URL format
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Check if URL scheme is allowed
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("only http and https URLs are supported")
	}

	// Check blocked hosts
	for _, blocked := range w.blockedHosts {
		if strings.Contains(parsedURL.Host, blocked) {
			return fmt.Errorf("host %s is blocked", parsedURL.Host)
		}
	}

	// Check allowed hosts (if specified)
	if len(w.allowedHosts) > 0 {
		allowed := false
		for _, allowedHost := range w.allowedHosts {
			// Use exact match or suffix match with dot
			if parsedURL.Host == allowedHost || strings.HasSuffix(parsedURL.Host, "."+allowedHost) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("host %s is not in allowed hosts list", parsedURL.Host)
		}
	}

	// Check required action
	action, ok := args["action"].(string)
	if !ok || action == "" {
		return fmt.Errorf("action is required and must be a string")
	}

	validActions := []string{"get_text", "get_title", "screenshot", "click", "get_html", "get_links"}
	found := false
	for _, va := range validActions {
		if va == action {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("unsupported action: %s", action)
	}

	return nil
}

// Execute performs the web request action
func (w *WebTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	urlStr := args["url"].(string)
	action := args["action"].(string)
	selector := getStringWithDefault(args, "selector", "")
	waitFor := getStringWithDefault(args, "wait_for", "")
	waitTime := getIntWithDefault(args, "wait_time", 3)
	fullPage := getBoolWithDefault(args, "full_page", false)

	// Create a timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	// Create Chrome context with options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent(w.userAgent),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(timeoutCtx, opts...)
	defer allocCancel()

	chromeCtx, chromeCancel := chromedp.NewContext(allocCtx)
	defer chromeCancel()

	var result interface{}
	var err error

	// Build task list
	var tasks []chromedp.Action

	// Navigate to URL
	tasks = append(tasks, chromedp.Navigate(urlStr))

	// Wait for selector if specified
	if waitFor != "" {
		tasks = append(tasks, chromedp.WaitVisible(waitFor, chromedp.ByQuery))
	} else {
		// Default wait for page load
		tasks = append(tasks, chromedp.Sleep(time.Duration(waitTime)*time.Second))
	}

	// Perform the requested action
	switch action {
	case "get_text":
		var text string
		if selector != "" {
			tasks = append(tasks, chromedp.Text(selector, &text, chromedp.ByQuery))
		} else {
			tasks = append(tasks, chromedp.Text("body", &text, chromedp.ByQuery))
		}
		err = chromedp.Run(chromeCtx, tasks...)
		if err == nil {
			// Truncate if too long
			if len(text) > w.maxContentLen {
				text = text[:w.maxContentLen] + "... (truncated)"
			}
			result = map[string]interface{}{
				"text":   text,
				"length": len(text),
			}
		}

	case "get_title":
		var title string
		tasks = append(tasks, chromedp.Title(&title))
		err = chromedp.Run(chromeCtx, tasks...)
		if err == nil {
			result = map[string]interface{}{
				"title": title,
			}
		}

	case "get_html":
		var html string
		if selector != "" {
			tasks = append(tasks, chromedp.OuterHTML(selector, &html, chromedp.ByQuery))
		} else {
			tasks = append(tasks, chromedp.OuterHTML("html", &html, chromedp.ByQuery))
		}
		err = chromedp.Run(chromeCtx, tasks...)
		if err == nil {
			// Truncate if too long
			if len(html) > w.maxContentLen {
				html = html[:w.maxContentLen] + "... (truncated)"
			}
			result = map[string]interface{}{
				"html":   html,
				"length": len(html),
			}
		}

	case "get_links":
		var links []map[string]string
		tasks = append(tasks, chromedp.ActionFunc(func(ctx context.Context) error {
			// Get all link elements and their attributes
			var linkTexts []string
			var linkHrefs []string
			
			// Get all link texts
			if err := chromedp.Evaluate(`
				Array.from(document.querySelectorAll('a[href]')).map(el => el.textContent.trim())
			`, &linkTexts).Do(ctx); err != nil {
				return err
			}
			
			// Get all link hrefs
			if err := chromedp.Evaluate(`
				Array.from(document.querySelectorAll('a[href]')).map(el => el.href)
			`, &linkHrefs).Do(ctx); err != nil {
				return err
			}
			
			// Combine them
			for i, href := range linkHrefs {
				text := ""
				if i < len(linkTexts) {
					text = linkTexts[i]
				}
				if href != "" {
					links = append(links, map[string]string{
						"href": href,
						"text": text,
					})
				}
			}
			return nil
		}))
		err = chromedp.Run(chromeCtx, tasks...)
		if err == nil {
			result = map[string]interface{}{
				"links": links,
				"count": len(links),
			}
		}

	case "screenshot":
		var buf []byte
		if fullPage {
			tasks = append(tasks, chromedp.FullScreenshot(&buf, 90))
		} else {
			tasks = append(tasks, chromedp.CaptureScreenshot(&buf))
		}
		err = chromedp.Run(chromeCtx, tasks...)
		if err == nil {
			result = map[string]interface{}{
				"screenshot_base64": fmt.Sprintf("data:image/png;base64,%s", encodeBase64(buf)),
				"size":              len(buf),
			}
		}

	case "click":
		if selector == "" {
			return nil, fmt.Errorf("selector is required for click action")
		}
		tasks = append(tasks, chromedp.Click(selector, chromedp.ByQuery))
		tasks = append(tasks, chromedp.Sleep(2*time.Second)) // Wait after click
		var title string
		tasks = append(tasks, chromedp.Title(&title))
		err = chromedp.Run(chromeCtx, tasks...)
		if err == nil {
			result = map[string]interface{}{
				"clicked":      true,
				"new_title":    title,
				"selector":     selector,
			}
		}

	default:
		return nil, fmt.Errorf("unsupported action: %s", action)
	}

	if err != nil {
		return &tools.ToolResult{
			Data: map[string]interface{}{
				"error":  err.Error(),
				"url":    urlStr,
				"action": action,
			},
		}, fmt.Errorf("web request failed: %w", err)
	}

	// Add common metadata
	if resultMap, ok := result.(map[string]interface{}); ok {
		resultMap["url"] = urlStr
		resultMap["action"] = action
		resultMap["success"] = true
	}

	return &tools.ToolResult{
		Data: result,
	}, nil
}

// Helper functions
func getIntWithDefault(args map[string]interface{}, key string, defaultValue int) int {
	if value, ok := args[key].(int); ok {
		return value
	}
	if value, ok := args[key].(float64); ok {
		return int(value)
	}
	return defaultValue
}

func getBoolWithDefault(args map[string]interface{}, key string, defaultValue bool) bool {
	if value, ok := args[key].(bool); ok {
		return value
	}
	return defaultValue
}

// Simple base64 encoding function
func encodeBase64(data []byte) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var result strings.Builder
	
	for i := 0; i < len(data); i += 3 {
		var b1, b2, b3 byte
		if i < len(data) {
			b1 = data[i]
		}
		if i+1 < len(data) {
			b2 = data[i+1]
		}
		if i+2 < len(data) {
			b3 = data[i+2]
		}
		
		result.WriteByte(chars[(b1>>2)&0x3F])
		result.WriteByte(chars[((b1<<4)|(b2>>4))&0x3F])
		
		if i+1 < len(data) {
			result.WriteByte(chars[((b2<<2)|(b3>>6))&0x3F])
		} else {
			result.WriteByte('=')
		}
		
		if i+2 < len(data) {
			result.WriteByte(chars[b3&0x3F])
		} else {
			result.WriteByte('=')
		}
	}
	
	return result.String()
}