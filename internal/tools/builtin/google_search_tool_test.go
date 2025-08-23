package builtin

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGoogleSearchTool_Name(t *testing.T) {
	tool := NewGoogleSearchTool(GoogleSearchConfig{})
	assert.Equal(t, "google_search", tool.Name())
}

func TestGoogleSearchTool_Description(t *testing.T) {
	tool := NewGoogleSearchTool(GoogleSearchConfig{})
	assert.Contains(t, tool.Description(), "Google")
	assert.Contains(t, tool.Description(), "search")
}

func TestGoogleSearchTool_Parameters(t *testing.T) {
	tool := NewGoogleSearchTool(GoogleSearchConfig{})
	params := tool.Parameters()
	assert.Equal(t, "object", params.Type)
	assert.Contains(t, params.Properties, "query")
	assert.Contains(t, params.Properties, "num_results")
	assert.Contains(t, params.Properties, "language")
	assert.Contains(t, params.Required, "query")
}

func TestGoogleSearchTool_Validate(t *testing.T) {
	tool := NewGoogleSearchTool(GoogleSearchConfig{})

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
			name: "valid language code",
			args: map[string]interface{}{
				"query":    "test",
				"language": "zh-CN",
			},
			wantErr: false,
		},
		{
			name: "invalid language code",
			args: map[string]interface{}{
				"query":    "test",
				"language": "invalid-lang",
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

func TestGoogleSearchTool_BuildGoogleURL(t *testing.T) {
	tool := NewGoogleSearchTool(GoogleSearchConfig{})

	tests := []struct {
		name       string
		query      string
		numResults int
		language   string
		country    string
		safeSearch bool
		wantContains []string
	}{
		{
			name:       "basic search",
			query:      "test query",
			numResults: 10,
			language:   "en",
			country:    "US",
			safeSearch: true,
			wantContains: []string{
				"google.com/search",
				"q=test+query",
				"num=10",
				"hl=en",
				"gl=US",
				"safe=active",
			},
		},
		{
			name:       "chinese search",
			query:      "测试搜索",
			numResults: 5,
			language:   "zh-CN",
			country:    "CN",
			safeSearch: false,
			wantContains: []string{
				"google.com/search",
				"num=5",
				"hl=zh-CN",
				"gl=CN",
				"safe=off",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := tool.buildGoogleURL(tt.query, tt.numResults, tt.language, tt.country, tt.safeSearch)
			for _, want := range tt.wantContains {
				assert.Contains(t, url, want)
			}
		})
	}
}

func TestGoogleSearchTool_ParseSearchResultsFromHTML(t *testing.T) {
	tool := NewGoogleSearchTool(GoogleSearchConfig{})
	
	// Test with sample HTML (this is a fallback method)
	html := `<div><h3>Sample Title</h3></div>`
	results := tool.parseSearchResultsFromHTML(html)
	
	assert.Len(t, results, 1)
	assert.Contains(t, results[0]["title"], "fallback")
	assert.Equal(t, 1, results[0]["position"])
}

func TestGoogleSearchTool_Execute_MockWebTool(t *testing.T) {
	// Create a custom GoogleSearchTool with a modified web tool for testing
	tool := NewGoogleSearchTool(GoogleSearchConfig{
		MaxResults:    10,
		SearchTimeout: 30 * time.Second,
	})

	args := map[string]interface{}{
		"query":       "test search",
		"num_results": 5,
		"language":    "en",
	}

	// For this test, we'll verify the URL building and validation logic
	// The actual execution would require a real Chrome browser or more complex mocking
	
	// Test URL building
	url := tool.buildGoogleURL("test search", 5, "en", "US", true)
	assert.Contains(t, url, "google.com/search")
	assert.Contains(t, url, "q=test+search")
	assert.Contains(t, url, "num=5")

	// Test validation passes
	err := tool.Validate(args)
	assert.NoError(t, err)

	// Test actual execution (this may fail without Chrome, which is expected)
	result, err := tool.Execute(context.Background(), args)
	
	// If Chrome is not available, the test should skip or handle gracefully
	if err != nil && (strings.Contains(err.Error(), "chrome") || 
		strings.Contains(err.Error(), "context deadline exceeded") ||
		strings.Contains(err.Error(), "connection refused")) {
		t.Skip("Chrome not available or network issue, skipping Google search execution test")
		return
	}

	// If execution succeeded, verify the structure
	if err == nil && result != nil {
		data := result.Data.(map[string]interface{})
		assert.Equal(t, "test search", data["query"])
		assert.Contains(t, data, "results")
		assert.Contains(t, data, "success")
	}
}

func TestNewGoogleSearchTool_DefaultConfig(t *testing.T) {
	tool := NewGoogleSearchTool(GoogleSearchConfig{})
	
	assert.Equal(t, 10, tool.maxResults)
	assert.Equal(t, 60*time.Second, tool.searchTimeout)
	assert.NotNil(t, tool.webTool)
}

func TestNewGoogleSearchTool_CustomConfig(t *testing.T) {
	config := GoogleSearchConfig{
		MaxResults:    5,
		SearchTimeout: 30 * time.Second,
		UserAgent:     "Custom-Search-Agent/1.0",
	}
	
	tool := NewGoogleSearchTool(config)
	
	assert.Equal(t, 5, tool.maxResults)
	assert.Equal(t, 30*time.Second, tool.searchTimeout)
	assert.Equal(t, "Custom-Search-Agent/1.0", tool.webTool.userAgent)
}

func TestGoogleSearchTool_Integration_Basic(t *testing.T) {
	// This test verifies the tool can be created and validates input correctly
	tool := NewGoogleSearchTool(GoogleSearchConfig{
		MaxResults: 3,
	})

	// Test validation
	args := map[string]interface{}{
		"query": "golang programming",
	}
	
	err := tool.Validate(args)
	assert.NoError(t, err)

	// Test URL building
	url := tool.buildGoogleURL("golang programming", 3, "en", "US", true)
	assert.Contains(t, url, "google.com/search")
	assert.Contains(t, url, "q=golang+programming")
	assert.Contains(t, url, "num=3")
}