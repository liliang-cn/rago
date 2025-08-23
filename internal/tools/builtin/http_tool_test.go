package builtin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPTool_Name(t *testing.T) {
	tool := NewHTTPTool(HTTPToolConfig{})
	assert.Equal(t, "http_request", tool.Name())
}

func TestHTTPTool_Description(t *testing.T) {
	tool := NewHTTPTool(HTTPToolConfig{})
	assert.Contains(t, tool.Description(), "HTTP requests")
}

func TestHTTPTool_Parameters(t *testing.T) {
	tool := NewHTTPTool(HTTPToolConfig{})
	params := tool.Parameters()
	assert.Equal(t, "object", params.Type)
	assert.Contains(t, params.Properties, "method")
	assert.Contains(t, params.Properties, "url")
	assert.Contains(t, params.Required, "url")
}

func TestHTTPTool_Validate(t *testing.T) {
	tool := NewHTTPTool(HTTPToolConfig{})

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name:    "missing URL",
			args:    map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "invalid URL",
			args: map[string]interface{}{
				"url": "not-a-url",
			},
			wantErr: true,
		},
		{
			name: "unsupported scheme",
			args: map[string]interface{}{
				"url": "ftp://example.com",
			},
			wantErr: true,
		},
		{
			name: "valid HTTP URL",
			args: map[string]interface{}{
				"url": "http://example.com",
			},
			wantErr: false,
		},
		{
			name: "valid HTTPS URL",
			args: map[string]interface{}{
				"url": "https://example.com",
			},
			wantErr: false,
		},
		{
			name: "unsupported method",
			args: map[string]interface{}{
				"url":    "https://example.com",
				"method": "TRACE",
			},
			wantErr: true,
		},
		{
			name: "valid method",
			args: map[string]interface{}{
				"url":    "https://example.com",
				"method": "POST",
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

func TestHTTPTool_ValidateBlockedHosts(t *testing.T) {
	tool := NewHTTPTool(HTTPToolConfig{
		BlockedHosts: []string{"blocked.com"},
	})

	args := map[string]interface{}{
		"url": "https://blocked.com/path",
	}
	err := tool.Validate(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blocked")
}

func TestHTTPTool_ValidateAllowedHosts(t *testing.T) {
	tool := NewHTTPTool(HTTPToolConfig{
		AllowedHosts: []string{"allowed.com"},
	})

	// Should fail for non-allowed host
	args := map[string]interface{}{
		"url": "https://notallowed.com/path",
	}
	err := tool.Validate(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in allowed hosts")

	// Should succeed for allowed host
	args = map[string]interface{}{
		"url": "https://allowed.com/path",
	}
	err = tool.Validate(args)
	assert.NoError(t, err)
}

func TestHTTPTool_Execute_GET(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.Header.Get("User-Agent"), "RAGO-HTTP-Tool")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success", "method": "GET"}`))
	}))
	defer server.Close()

	tool := NewHTTPTool(HTTPToolConfig{
		Timeout: 10 * time.Second,
	})

	args := map[string]interface{}{
		"url":    server.URL,
		"method": "GET",
	}

	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.NotNil(t, result)

	data := result.Data.(map[string]interface{})
	assert.Equal(t, 200, data["status_code"])
	assert.Equal(t, true, data["success"])
	assert.Equal(t, "GET", data["method"])
	assert.Contains(t, data, "body")

	// Check if JSON was parsed
	if body, ok := data["body"].(map[string]interface{}); ok {
		assert.Equal(t, "success", body["message"])
		assert.Equal(t, "GET", body["method"])
	}
}

func TestHTTPTool_Execute_POST(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"created": true}`))
	}))
	defer server.Close()

	tool := NewHTTPTool(HTTPToolConfig{})

	args := map[string]interface{}{
		"url":          server.URL,
		"method":       "POST",
		"body":         `{"name": "test"}`,
		"content_type": "application/json",
	}

	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.NotNil(t, result)

	data := result.Data.(map[string]interface{})
	assert.Equal(t, 201, data["status_code"])
	assert.Equal(t, true, data["success"])
	assert.Equal(t, "POST", data["method"])
}

func TestHTTPTool_Execute_WithHeaders(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
		assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	tool := NewHTTPTool(HTTPToolConfig{})

	args := map[string]interface{}{
		"url": server.URL,
		"headers": map[string]interface{}{
			"Authorization":   "Bearer token123",
			"X-Custom-Header": "custom-value",
		},
	}

	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.NotNil(t, result)

	data := result.Data.(map[string]interface{})
	assert.Equal(t, 200, data["status_code"])
	assert.Equal(t, true, data["success"])
}

func TestHTTPTool_Execute_ErrorHandling(t *testing.T) {
	tool := NewHTTPTool(HTTPToolConfig{
		Timeout: 1 * time.Millisecond, // Very short timeout
	})

	args := map[string]interface{}{
		"url": "https://httpstat.us/200?sleep=1000", // This will timeout
	}

	result, err := tool.Execute(context.Background(), args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP request failed")
	
	if result != nil {
		data := result.Data.(map[string]interface{})
		assert.Contains(t, data, "error")
	}
}

func TestHTTPTool_Execute_NonJSONResponse(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Plain text response"))
	}))
	defer server.Close()

	tool := NewHTTPTool(HTTPToolConfig{})

	args := map[string]interface{}{
		"url": server.URL,
	}

	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.NotNil(t, result)

	data := result.Data.(map[string]interface{})
	assert.Equal(t, 200, data["status_code"])
	assert.Equal(t, "Plain text response", data["body"])
	assert.Equal(t, "text/plain", data["content_type"])
}

func TestNewHTTPTool_DefaultConfig(t *testing.T) {
	tool := NewHTTPTool(HTTPToolConfig{})
	
	assert.Equal(t, 30*time.Second, tool.client.Timeout)
	assert.Equal(t, int64(10*1024*1024), tool.maxBodySize)
	assert.Equal(t, "RAGO-HTTP-Tool/1.0", tool.userAgent)
}

func TestNewHTTPTool_CustomConfig(t *testing.T) {
	config := HTTPToolConfig{
		Timeout:     10 * time.Second,
		MaxBodySize: 1024,
		UserAgent:   "Custom-Agent/1.0",
	}
	
	tool := NewHTTPTool(config)
	
	assert.Equal(t, 10*time.Second, tool.client.Timeout)
	assert.Equal(t, int64(1024), tool.maxBodySize)
	assert.Equal(t, "Custom-Agent/1.0", tool.userAgent)
}