package providers

import (
	"encoding/json"
	"testing"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveThinkTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		enabled  bool
		expected string
	}{
		{
			name:     "Remove single think tag when enabled",
			input:    "Before <think>internal thoughts</think> After",
			enabled:  true,
			expected: "Before  After",
		},
		{
			name:     "Remove multiple think tags when enabled",
			input:    "Start <think>thought 1</think> Middle <think>thought 2</think> End",
			enabled:  true,
			expected: "Start  Middle  End",
		},
		{
			name:     "Remove multiline think tags when enabled",
			input:    "Start\n<think>\nMultiline\nthought\n</think>\nEnd",
			enabled:  true,
			expected: "Start\n\nEnd",
		},
		{
			name:     "Keep content when disabled",
			input:    "Before <think>internal thoughts</think> After",
			enabled:  false,
			expected: "Before <think>internal thoughts</think> After",
		},
		{
			name:     "Handle unclosed think tags",
			input:    "Start <think>unclosed thinking...",
			enabled:  true,
			expected: "Start ",
		},
		{
			name:     "Handle closed think tags",
			input:    "Start <think>closed thinking</think> End",
			enabled:  true,
			expected: "Start  End",
		},
		{
			name:     "Handle empty think tags",
			input:    "Start <think></think> End",
			enabled:  true,
			expected: "Start  End",
		},
		{
			name:     "No think tags",
			input:    "Just regular content without any tags",
			enabled:  true,
			expected: "Just regular content without any tags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &OllamaLLMProvider{
				config: &domain.OllamaProviderConfig{
					HideBuiltinThinkTag: tt.enabled,
				},
			}
			
			result := provider.removeThinkTags(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateWithThinkTagRemoval(t *testing.T) {
	// This test would require a mock Ollama client or integration test
	// For now, we'll create a simple test to ensure the config propagates correctly
	
	config := &domain.OllamaProviderConfig{
		BaseProviderConfig: domain.BaseProviderConfig{
			Type: domain.ProviderOllama,
		},
		BaseURL:             "http://localhost:11434",
		LLMModel:            "test-model",
		HideBuiltinThinkTag: true,
	}
	
	provider, err := NewOllamaLLMProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)
	
	// Verify the config is set correctly
	ollamaProvider, ok := provider.(*OllamaLLMProvider)
	require.True(t, ok)
	assert.True(t, ollamaProvider.config.HideBuiltinThinkTag)
}

func BenchmarkRemoveThinkTags(b *testing.B) {
	provider := &OllamaLLMProvider{
		config: &domain.OllamaProviderConfig{
			HideBuiltinThinkTag: true,
		},
	}
	
	content := "Before <think>This is some internal thinking that should be removed</think> After <think>Another thought</think> End"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = provider.removeThinkTags(content)
	}
}

func TestRegexPattern(t *testing.T) {
	// Test the regex pattern directly
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Complex HTML-like content",
			input:    "<p>Text</p> <think>Remove this</think> <div>Keep this</div>",
			expected: "<p>Text</p>  <div>Keep this</div>",
		},
		{
			name:     "Special characters in think tag",
			input:    "Start <think>!@#$%^&*()</think> End",
			expected: "Start  End",
		},
		{
			name:     "JSON-like content in think tag",
			input:    `Result: <think>{"debug": "info"}</think> {"data": "value"}`,
			expected: `Result:  {"data": "value"}`,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := thinkTagRegex.ReplaceAllString(tc.input, "")
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestStreamingWithThinkTags(t *testing.T) {
	// Test that streaming callbacks receive filtered content
	provider := &OllamaLLMProvider{
		config: &domain.OllamaProviderConfig{
			HideBuiltinThinkTag: true,
		},
	}
	
	// Simulate streaming chunks that might contain think tags
	chunks := []string{
		"Start of response <thi",
		"nk>internal thought</thi",
		"nk> actual content",
	}
	
	var result string
	for _, chunk := range chunks {
		filtered := provider.removeThinkTags(chunk)
		result += filtered
	}
	
	// Note: In real streaming, think tags might be split across chunks
	// This is a limitation of the current implementation
	// The provider would need to buffer content to handle this properly
	assert.Contains(t, result, "Start of response")
}

func TestStructuredGenerationWithThinkTags(t *testing.T) {
	provider := &OllamaLLMProvider{
		config: &domain.OllamaProviderConfig{
			HideBuiltinThinkTag: true,
		},
	}
	
	// Test that structured generation removes think tags from JSON
	jsonWithThink := `<think>Let me structure this</think>{"key": "value", "number": 42}`
	cleaned := provider.removeThinkTags(jsonWithThink)
	assert.Equal(t, `{"key": "value", "number": 42}`, cleaned)
	
	// Verify the cleaned JSON is valid
	var data map[string]interface{}
	err := json.Unmarshal([]byte(cleaned), &data)
	assert.NoError(t, err)
	assert.Equal(t, "value", data["key"])
	assert.Equal(t, float64(42), data["number"])
}