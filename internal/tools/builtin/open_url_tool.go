package builtin

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

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
	return "打开指定的网页链接，生成可点击的URL链接。支持HTTP和HTTPS协议。"
}

// Parameters returns the tool parameters schema
func (o *OpenURLTool) Parameters() tools.ToolParameters {
	return tools.ToolParameters{
		Type: "object",
		Properties: map[string]tools.ToolParameter{
			"url": {
				Type:        "string",
				Description: "要打开的网页链接地址",
			},
			"description": {
				Type:        "string", 
				Description: "链接的描述或标题",
				Default:     "",
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

// Execute performs the URL opening operation
func (o *OpenURLTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	urlStr := strings.TrimSpace(args["url"].(string))
	description := getStringWithDefault(args, "description", "")

	// Parse and validate the URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return &tools.ToolResult{
			Data: map[string]interface{}{
				"error":   fmt.Sprintf("URL解析失败: %v", err),
				"url":     urlStr,
				"success": false,
			},
		}, fmt.Errorf("URL parsing failed: %w", err)
	}

	// Generate a clean URL
	cleanURL := parsedURL.String()
	
	// If no description provided, extract from URL
	if description == "" {
		description = o.generateDescription(parsedURL)
	}

	return &tools.ToolResult{
		Data: map[string]interface{}{
			"url":         cleanURL,
			"description": description,
			"host":        parsedURL.Host,
			"scheme":      parsedURL.Scheme,
			"message":     fmt.Sprintf("已为您准备打开链接: %s", cleanURL),
			"clickable_link": fmt.Sprintf("[%s](%s)", description, cleanURL),
			"success":     true,
		},
	}, nil
}

// generateDescription creates a description from the URL
func (o *OpenURLTool) generateDescription(parsedURL *url.URL) string {
	host := parsedURL.Host
	
	// Remove www. prefix if present
	if strings.HasPrefix(host, "www.") {
		host = host[4:]
	}
	
	// Generate description based on common sites
	switch {
	case strings.Contains(host, "github.com"):
		return fmt.Sprintf("GitHub - %s", host)
	case strings.Contains(host, "stackoverflow.com"):
		return fmt.Sprintf("Stack Overflow - %s", host) 
	case strings.Contains(host, "baidu.com"):
		return fmt.Sprintf("百度 - %s", host)
	case strings.Contains(host, "google.com"):
		return fmt.Sprintf("Google - %s", host)
	case strings.Contains(host, "zhihu.com"):
		return fmt.Sprintf("知乎 - %s", host)
	case strings.Contains(host, "bilibili.com"):
		return fmt.Sprintf("哔哩哔哩 - %s", host)
	default:
		return fmt.Sprintf("网页链接 - %s", host)
	}
}