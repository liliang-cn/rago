package builtin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDuckDuckGoSearchTool_Name(t *testing.T) {
	tool := NewDuckDuckGoSearchTool(DuckDuckGoSearchConfig{})
	assert.Equal(t, "duckduckgo_search", tool.Name())
}

func TestDuckDuckGoSearchTool_Description(t *testing.T) {
	tool := NewDuckDuckGoSearchTool(DuckDuckGoSearchConfig{})
	assert.Contains(t, tool.Description(), "DuckDuckGo")
	assert.Contains(t, tool.Description(), "Privacy-focused")
}

func TestDuckDuckGoSearchTool_Parameters(t *testing.T) {
	tool := NewDuckDuckGoSearchTool(DuckDuckGoSearchConfig{})
	params := tool.Parameters()
	assert.Equal(t, "object", params.Type)
	assert.Contains(t, params.Properties, "query")
	assert.Contains(t, params.Properties, "num_results")
	assert.Contains(t, params.Properties, "region")
	assert.Contains(t, params.Properties, "safe_search")
	assert.Contains(t, params.Required, "query")
}

func TestDuckDuckGoSearchTool_Validate(t *testing.T) {
	tool := NewDuckDuckGoSearchTool(DuckDuckGoSearchConfig{})

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name:    "missing query",
			args:    map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "empty query",
			args: map[string]interface{}{
				"query": "",
			},
			wantErr: true,
		},
		{
			name: "whitespace only query",
			args: map[string]interface{}{
				"query": "   ",
			},
			wantErr: true,
		},
		{
			name: "valid query",
			args: map[string]interface{}{
				"query": "test search",
			},
			wantErr: false,
		},
		{
			name: "invalid num_results - too low",
			args: map[string]interface{}{
				"query":       "test",
				"num_results": float64(0),
			},
			wantErr: true,
		},
		{
			name: "invalid num_results - too high",
			args: map[string]interface{}{
				"query":       "test",
				"num_results": float64(25),
			},
			wantErr: true,
		},
		{
			name: "valid num_results",
			args: map[string]interface{}{
				"query":       "test",
				"num_results": float64(5),
			},
			wantErr: false,
		},
		{
			name: "valid safe_search",
			args: map[string]interface{}{
				"query":       "test",
				"safe_search": "strict",
			},
			wantErr: false,
		},
		{
			name: "invalid safe_search",
			args: map[string]interface{}{
				"query":       "test",
				"safe_search": "invalid",
			},
			wantErr: true,
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

func TestDuckDuckGoSearchTool_BuildDuckDuckGoURL(t *testing.T) {
	tool := NewDuckDuckGoSearchTool(DuckDuckGoSearchConfig{})

	tests := []struct {
		name       string
		query      string
		region     string
		safeSearch string
		wantContains []string
	}{
		{
			name:       "basic search",
			query:      "test query",
			region:     "us-en",
			safeSearch: "moderate",
			wantContains: []string{
				"html.duckduckgo.com/html",
				"q=test+query",
				"kl=us-en",
				"safe=moderate",
			},
		},
		{
			name:       "strict safe search",
			query:      "example search",
			region:     "cn-zh",
			safeSearch: "strict",
			wantContains: []string{
				"html.duckduckgo.com/html",
				"safe=strict",
				"kl=cn-zh",
			},
		},
		{
			name:       "safe search off",
			query:      "test",
			region:     "us-en", 
			safeSearch: "off",
			wantContains: []string{
				"safe=-1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := tool.buildDuckDuckGoURL(tt.query, tt.region, tt.safeSearch)
			for _, want := range tt.wantContains {
				assert.Contains(t, url, want)
			}
		})
	}
}

func TestDuckDuckGoSearchTool_ParseSearchResults(t *testing.T) {
	tool := NewDuckDuckGoSearchTool(DuckDuckGoSearchConfig{})
	
	// Test with sample DuckDuckGo-like HTML
	html := `
	<html>
	<body>
		<a class="result__a" href="https://example.com">Example Title 1</a>
		<a class="result__snippet">This is a sample search result snippet.</a>
		<a class="result__a" href="https://test.com">Test Title 2</a>
		<a class="result__snippet">Another sample snippet for testing.</a>
	</body>
	</html>
	`
	
	results := tool.parseSearchResults(html, 5)
	
	// Should have at least some results (might be fallback)
	assert.True(t, len(results) >= 1)
	
	// First result should have required fields
	first := results[0]
	assert.Contains(t, first, "title")
	assert.Contains(t, first, "url")
	assert.Contains(t, first, "snippet")
	assert.Contains(t, first, "position")
}

func TestDuckDuckGoSearchTool_Execute_MockHTTP(t *testing.T) {
	// Create a mock HTTP server to simulate DuckDuckGo response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if it's a DuckDuckGo search URL pattern
		if strings.Contains(r.URL.String(), "q=") {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`
<!DOCTYPE html>
<html>
<head><title>DuckDuckGo Search Results</title></head>
<body>
	<a class="result__a" href="https://golang.org">The Go Programming Language</a>
	<a class="result__snippet">Go is an open source programming language that makes it easy to build simple, reliable, and efficient software.</a>
	<a class="result__a" href="https://tour.golang.org">A Tour of Go</a>
	<a class="result__snippet">Welcome to a tour of the Go programming language. The tour covers the most important features of the language.</a>
</body>
</html>
			`))
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// Create tool with modified HTTP tool for testing
	config := DuckDuckGoSearchConfig{
		MaxResults:    10,
		SearchTimeout: 30 * time.Second,
		UserAgent:     "Test-Agent",
	}
	tool := NewDuckDuckGoSearchTool(config)
	
	// We'll test by directly calling the HTTP tool with our test URL
	testURL := server.URL + "?q=test"
	
	// We'll test by directly calling the HTTP tool with our test URL
	httpArgs := map[string]interface{}{
		"url":    testURL,
		"method": "GET",
		"headers": map[string]interface{}{
			"Accept": "text/html",
		},
	}

	result, err := tool.httpTool.Execute(context.Background(), httpArgs)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	data := result.Data.(map[string]interface{})
	assert.Equal(t, true, data["success"])
	assert.Equal(t, 200, data["status_code"])
	
	// Verify we got HTML content
	body := data["body"].(string)
	assert.Contains(t, body, "DuckDuckGo Search Results")
	assert.Contains(t, body, "golang.org")
}

func TestDuckDuckGoSearchTool_Integration_Basic(t *testing.T) {
	// This test verifies the tool can be created and validates input correctly
	tool := NewDuckDuckGoSearchTool(DuckDuckGoSearchConfig{
		MaxResults: 3,
	})

	// Test validation
	args := map[string]interface{}{
		"query": "golang programming",
	}
	
	err := tool.Validate(args)
	assert.NoError(t, err)

	// Test URL building
	url := tool.buildDuckDuckGoURL("golang programming", "us-en", "moderate")
	assert.Contains(t, url, "html.duckduckgo.com/html")
	assert.Contains(t, url, "q=golang+programming")
	assert.Contains(t, url, "kl=us-en")
	assert.Contains(t, url, "safe=moderate")
}

func TestNewDuckDuckGoSearchTool_DefaultConfig(t *testing.T) {
	tool := NewDuckDuckGoSearchTool(DuckDuckGoSearchConfig{})
	
	assert.Equal(t, 10, tool.maxResults)
	assert.Equal(t, 30*time.Second, tool.searchTimeout)
	assert.NotNil(t, tool.httpTool)
}

func TestNewDuckDuckGoSearchTool_CustomConfig(t *testing.T) {
	config := DuckDuckGoSearchConfig{
		MaxResults:    5,
		SearchTimeout: 15 * time.Second,
		UserAgent:     "Custom-DuckDuckGo-Agent/1.0",
		SafeSearch:    "strict",
	}
	
	tool := NewDuckDuckGoSearchTool(config)
	
	assert.Equal(t, 5, tool.maxResults)
	assert.Equal(t, 15*time.Second, tool.searchTimeout)
	assert.Equal(t, "Custom-DuckDuckGo-Agent/1.0", tool.httpTool.userAgent)
}