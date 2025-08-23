package builtin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebTool_Name(t *testing.T) {
	tool := NewWebTool(WebToolConfig{})
	assert.Equal(t, "web_request", tool.Name())
}

func TestWebTool_Description(t *testing.T) {
	tool := NewWebTool(WebToolConfig{})
	assert.Contains(t, tool.Description(), "web pages")
	assert.Contains(t, tool.Description(), "headless browser")
}

func TestWebTool_Parameters(t *testing.T) {
	tool := NewWebTool(WebToolConfig{})
	params := tool.Parameters()
	assert.Equal(t, "object", params.Type)
	assert.Contains(t, params.Properties, "action")
	assert.Contains(t, params.Properties, "url")
	assert.Contains(t, params.Required, "url")
	assert.Contains(t, params.Required, "action")
}

func TestWebTool_Validate(t *testing.T) {
	tool := NewWebTool(WebToolConfig{})

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name:    "missing URL",
			args:    map[string]interface{}{"action": "get_text"},
			wantErr: true,
		},
		{
			name:    "missing action",
			args:    map[string]interface{}{"url": "https://example.com"},
			wantErr: true,
		},
		{
			name: "invalid URL",
			args: map[string]interface{}{
				"url":    "not-a-url",
				"action": "get_text",
			},
			wantErr: true,
		},
		{
			name: "unsupported scheme",
			args: map[string]interface{}{
				"url":    "ftp://example.com",
				"action": "get_text",
			},
			wantErr: true,
		},
		{
			name: "valid args",
			args: map[string]interface{}{
				"url":    "https://example.com",
				"action": "get_text",
			},
			wantErr: false,
		},
		{
			name: "unsupported action",
			args: map[string]interface{}{
				"url":    "https://example.com",
				"action": "invalid_action",
			},
			wantErr: true,
		},
		{
			name: "valid action - get_title",
			args: map[string]interface{}{
				"url":    "https://example.com",
				"action": "get_title",
			},
			wantErr: false,
		},
		{
			name: "valid action - screenshot",
			args: map[string]interface{}{
				"url":    "https://example.com",
				"action": "screenshot",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tool.Validate(tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWebTool_ValidateBlockedHosts(t *testing.T) {
	tool := NewWebTool(WebToolConfig{
		BlockedHosts: []string{"blocked.com"},
	})

	args := map[string]interface{}{
		"url":    "https://blocked.com/path",
		"action": "get_text",
	}
	err := tool.Validate(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blocked")
}

func TestWebTool_ValidateAllowedHosts(t *testing.T) {
	tool := NewWebTool(WebToolConfig{
		AllowedHosts: []string{"allowed.com"},
	})

	// Should fail for non-allowed host
	args := map[string]interface{}{
		"url":    "https://notallowed.com/path",
		"action": "get_text",
	}
	err := tool.Validate(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in allowed hosts")

	// Should succeed for allowed host
	args = map[string]interface{}{
		"url":    "https://allowed.com/path",
		"action": "get_text",
	}
	err = tool.Validate(args)
	assert.NoError(t, err)
}

// Note: The following tests require a running Chrome/Chromium browser
// They might be skipped in CI environments without proper setup

func TestWebTool_Execute_GetTitle_MockServer(t *testing.T) {
	// Create test server with HTML content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Test Page Title</title>
</head>
<body>
    <h1>Hello World</h1>
    <p>This is a test page.</p>
</body>
</html>
		`))
	}))
	defer server.Close()

	tool := NewWebTool(WebToolConfig{
		Timeout: 30 * time.Second,
	})

	args := map[string]interface{}{
		"url":       server.URL,
		"action":    "get_title",
		"wait_time": 1,
	}

	// Note: This test will be skipped if Chrome is not available
	result, err := tool.Execute(context.Background(), args)
	if err != nil && strings.Contains(err.Error(), "chrome") {
		t.Skip("Chrome not available, skipping web tool test")
		return
	}

	require.NoError(t, err)
	assert.NotNil(t, result)

	data := result.Data.(map[string]interface{})
	assert.Equal(t, server.URL, data["url"])
	assert.Equal(t, "get_title", data["action"])
	assert.Equal(t, true, data["success"])
	assert.Equal(t, "Test Page Title", data["title"])
}

func TestWebTool_Execute_GetText_MockServer(t *testing.T) {
	// Create test server with HTML content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Test Page</title>
</head>
<body>
    <h1>Main Heading</h1>
    <p>This is paragraph text.</p>
    <div class="content">Special content here</div>
</body>
</html>
		`))
	}))
	defer server.Close()

	tool := NewWebTool(WebToolConfig{
		Timeout: 30 * time.Second,
	})

	args := map[string]interface{}{
		"url":       server.URL,
		"action":    "get_text",
		"wait_time": 1,
	}

	result, err := tool.Execute(context.Background(), args)
	if err != nil && strings.Contains(err.Error(), "chrome") {
		t.Skip("Chrome not available, skipping web tool test")
		return
	}

	require.NoError(t, err)
	assert.NotNil(t, result)

	data := result.Data.(map[string]interface{})
	assert.Equal(t, server.URL, data["url"])
	assert.Equal(t, "get_text", data["action"])
	assert.Equal(t, true, data["success"])
	
	text := data["text"].(string)
	assert.Contains(t, text, "Main Heading")
	assert.Contains(t, text, "This is paragraph text")
}

func TestWebTool_Execute_GetHTML_MockServer(t *testing.T) {
	// Create test server with HTML content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Test Page</title>
</head>
<body>
    <h1>Main Heading</h1>
    <p>This is paragraph text.</p>
</body>
</html>
		`))
	}))
	defer server.Close()

	tool := NewWebTool(WebToolConfig{
		Timeout: 30 * time.Second,
	})

	args := map[string]interface{}{
		"url":       server.URL,
		"action":    "get_html",
		"selector":  "body",
		"wait_time": 1,
	}

	result, err := tool.Execute(context.Background(), args)
	if err != nil && strings.Contains(err.Error(), "chrome") {
		t.Skip("Chrome not available, skipping web tool test")
		return
	}

	require.NoError(t, err)
	assert.NotNil(t, result)

	data := result.Data.(map[string]interface{})
	assert.Equal(t, server.URL, data["url"])
	assert.Equal(t, "get_html", data["action"])
	assert.Equal(t, true, data["success"])
	
	html := data["html"].(string)
	assert.Contains(t, html, "<h1>Main Heading</h1>")
	assert.Contains(t, html, "<p>This is paragraph text.</p>")
}

func TestWebTool_Execute_GetLinks_MockServer(t *testing.T) {
	// Create test server with HTML content containing links
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Test Page</title>
</head>
<body>
    <h1>Test Links</h1>
    <a href="https://example.com">Example Link</a>
    <a href="/local-path">Local Link</a>
    <a href="mailto:test@example.com">Email Link</a>
</body>
</html>
		`))
	}))
	defer server.Close()

	tool := NewWebTool(WebToolConfig{
		Timeout: 30 * time.Second,
	})

	args := map[string]interface{}{
		"url":       server.URL,
		"action":    "get_links",
		"wait_time": 1,
	}

	result, err := tool.Execute(context.Background(), args)
	if err != nil && strings.Contains(err.Error(), "chrome") {
		t.Skip("Chrome not available, skipping web tool test")
		return
	}

	require.NoError(t, err)
	assert.NotNil(t, result)

	data := result.Data.(map[string]interface{})
	assert.Equal(t, server.URL, data["url"])
	assert.Equal(t, "get_links", data["action"])
	assert.Equal(t, true, data["success"])
	
	links := data["links"].([]map[string]string)
	assert.True(t, len(links) >= 2) // At least example.com and local-path links
	
	// Debug: print actual links found
	t.Logf("Found %d links:", len(links))
	for i, link := range links {
		t.Logf("  %d: href=%s, text=%s", i+1, link["href"], link["text"])
	}
	
	// Check if we found the expected links (allowing for URL normalization)
	foundExampleLink := false
	foundLocalLink := false
	for _, link := range links {
		// Example.com link (Chrome may add trailing slash)
		if (link["href"] == "https://example.com" || link["href"] == "https://example.com/") && 
		   link["text"] == "Example Link" {
			foundExampleLink = true
		}
		// Local path (Chrome converts relative to absolute)
		if (link["href"] == "/local-path" || strings.Contains(link["href"], "/local-path")) && 
		   link["text"] == "Local Link" {
			foundLocalLink = true
		}
	}
	assert.True(t, foundExampleLink, "Should find example.com link")
	assert.True(t, foundLocalLink, "Should find local path link")
}

func TestNewWebTool_DefaultConfig(t *testing.T) {
	tool := NewWebTool(WebToolConfig{})
	
	assert.Equal(t, 60*time.Second, tool.timeout)
	assert.Equal(t, 100*1024, tool.maxContentLen)
	assert.Equal(t, "RAGO-Web-Tool/1.0", tool.userAgent)
}

func TestNewWebTool_CustomConfig(t *testing.T) {
	config := WebToolConfig{
		Timeout:       30 * time.Second,
		MaxContentLen: 1024,
		UserAgent:     "Custom-Web-Agent/1.0",
	}
	
	tool := NewWebTool(config)
	
	assert.Equal(t, 30*time.Second, tool.timeout)
	assert.Equal(t, 1024, tool.maxContentLen)
	assert.Equal(t, "Custom-Web-Agent/1.0", tool.userAgent)
}

func TestGetHelperFunctions(t *testing.T) {
	args := map[string]interface{}{
		"string_val":  "test",
		"int_val":     42,
		"float_val":   3.14,
		"bool_val":    true,
	}

	// Test getStringWithDefault
	assert.Equal(t, "test", getStringWithDefault(args, "string_val", "default"))
	assert.Equal(t, "default", getStringWithDefault(args, "missing", "default"))

	// Test getIntWithDefault  
	assert.Equal(t, 42, getIntWithDefault(args, "int_val", 0))
	assert.Equal(t, 3, getIntWithDefault(args, "float_val", 0)) // float64 to int
	assert.Equal(t, 0, getIntWithDefault(args, "missing", 0))

	// Test getBoolWithDefault
	assert.Equal(t, true, getBoolWithDefault(args, "bool_val", false))
	assert.Equal(t, false, getBoolWithDefault(args, "missing", false))
}

func TestEncodeBase64(t *testing.T) {
	tests := []struct {
		input    []byte
		expected string
	}{
		{[]byte(""), ""},
		{[]byte("f"), "Zg=="},
		{[]byte("fo"), "Zm8="},
		{[]byte("foo"), "Zm9v"},
		{[]byte("foob"), "Zm9vYg=="},
		{[]byte("fooba"), "Zm9vYmE="},
		{[]byte("foobar"), "Zm9vYmFy"},
		{[]byte("hello world"), "aGVsbG8gd29ybGQ="},
	}

	for _, test := range tests {
		result := encodeBase64(test.input)
		assert.Equal(t, test.expected, result, "Failed for input: %s", string(test.input))
	}
}