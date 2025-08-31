package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/liliang-cn/rago/pkg/tools"
)

// HTTPTool handles HTTP requests including GET, POST, PUT, DELETE
type HTTPTool struct {
	client        *http.Client
	maxBodySize   int64
	allowedHosts  []string // Empty means all hosts allowed
	blockedHosts  []string
	userAgent     string
}

// HTTPToolConfig contains configuration for the HTTP tool
type HTTPToolConfig struct {
	Timeout       time.Duration `json:"timeout"`
	MaxBodySize   int64         `json:"max_body_size"`
	AllowedHosts  []string      `json:"allowed_hosts"`
	BlockedHosts  []string      `json:"blocked_hosts"`
	UserAgent     string        `json:"user_agent"`
	FollowRedirect bool         `json:"follow_redirect"`
}

// NewHTTPTool creates a new HTTP tool instance
func NewHTTPTool(config HTTPToolConfig) *HTTPTool {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxBodySize == 0 {
		config.MaxBodySize = 10 * 1024 * 1024 // 10MB default
	}
	if config.UserAgent == "" {
		config.UserAgent = "RAGO-HTTP-Tool/1.0"
	}

	client := &http.Client{
		Timeout: config.Timeout,
	}

	if !config.FollowRedirect {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return &HTTPTool{
		client:       client,
		maxBodySize:  config.MaxBodySize,
		allowedHosts: config.AllowedHosts,
		blockedHosts: config.BlockedHosts,
		userAgent:    config.UserAgent,
	}
}

// Name returns the tool name
func (h *HTTPTool) Name() string {
	return "http_request"
}

// Description returns the tool description
func (h *HTTPTool) Description() string {
	return "Make HTTP requests (GET, POST, PUT, DELETE) to web endpoints. Supports headers, body data, and various response formats."
}

// Parameters returns the tool parameters schema
func (h *HTTPTool) Parameters() tools.ToolParameters {
	return tools.ToolParameters{
		Type: "object",
		Properties: map[string]tools.ToolParameter{
			"method": {
				Type:        "string",
				Description: "HTTP method to use",
				Enum:        []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS"},
				Default:     "GET",
			},
			"url": {
				Type:        "string",
				Description: "Target URL to request",
			},
			"headers": {
				Type:        "object",
				Description: "HTTP headers to include in request",
			},
			"body": {
				Type:        "string",
				Description: "Request body data (for POST, PUT methods)",
			},
			"content_type": {
				Type:        "string",
				Description: "Content-Type header value",
				Default:     "application/json",
			},
			"timeout": {
				Type:        "integer",
				Description: "Request timeout in seconds",
				Default:     30,
			},
		},
		Required: []string{"url"},
	}
}

// Validate validates the tool arguments
func (h *HTTPTool) Validate(args map[string]interface{}) error {
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
	for _, blocked := range h.blockedHosts {
		if strings.Contains(parsedURL.Host, blocked) {
			return fmt.Errorf("host %s is blocked", parsedURL.Host)
		}
	}

	// Check allowed hosts (if specified)
	if len(h.allowedHosts) > 0 {
		allowed := false
		for _, allowedHost := range h.allowedHosts {
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

	// Validate method
	if method, ok := args["method"].(string); ok {
		validMethods := []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS"}
		methodUpper := strings.ToUpper(method)
		found := false
		for _, vm := range validMethods {
			if vm == methodUpper {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unsupported HTTP method: %s", method)
		}
	}

	return nil
}

// Execute performs the HTTP request
func (h *HTTPTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	// Get parameters
	urlStr := args["url"].(string)
	method := strings.ToUpper(getStringWithDefault(args, "method", "GET"))
	body := getStringWithDefault(args, "body", "")
	contentType := getStringWithDefault(args, "content_type", "application/json")

	// Prepare request body
	var bodyReader io.Reader
	if body != "" && (method == "POST" || method == "PUT") {
		bodyReader = strings.NewReader(body)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, urlStr, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent
	req.Header.Set("User-Agent", h.userAgent)

	// Set Content-Type for requests with body
	if bodyReader != nil {
		req.Header.Set("Content-Type", contentType)
	}

	// Add custom headers
	if headersInterface, ok := args["headers"]; ok {
		if headersMap, ok := headersInterface.(map[string]interface{}); ok {
			for key, value := range headersMap {
				if valueStr, ok := value.(string); ok {
					req.Header.Set(key, valueStr)
				}
			}
		}
	}

	// Make request
	startTime := time.Now()
	resp, err := h.client.Do(req)
	elapsed := time.Since(startTime)

	if err != nil {
		return &tools.ToolResult{
			Data: map[string]interface{}{
				"error":    err.Error(),
				"url":      urlStr,
				"method":   method,
				"elapsed":  elapsed.String(),
			},
		}, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body with size limit
	limitReader := io.LimitReader(resp.Body, h.maxBodySize)
	responseBody, err := io.ReadAll(limitReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response headers
	responseHeaders := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			responseHeaders[key] = values[0]
		}
	}

	// Try to parse JSON response
	var parsedBody interface{}
	contentTypeHeader := resp.Header.Get("Content-Type")
	if strings.Contains(contentTypeHeader, "application/json") {
		var jsonBody interface{}
		if err := json.Unmarshal(responseBody, &jsonBody); err == nil {
			parsedBody = jsonBody
		}
	}

	// If JSON parsing failed or not JSON, keep as string
	if parsedBody == nil {
		parsedBody = string(responseBody)
	}

	result := map[string]interface{}{
		"status_code":      resp.StatusCode,
		"status":           resp.Status,
		"url":              urlStr,
		"method":           method,
		"headers":          responseHeaders,
		"body":             parsedBody,
		"content_length":   len(responseBody),
		"content_type":     contentTypeHeader,
		"elapsed":          elapsed.String(),
		"success":          resp.StatusCode >= 200 && resp.StatusCode < 300,
	}

	return &tools.ToolResult{
		Data: result,
	}, nil
}

