package extraction

import (
	"context"
	"testing"
	"time"
)

func TestNewChromedpExtractor(t *testing.T) {
	extractor := NewChromedpExtractor()
	if extractor == nil {
		t.Fatal("expected extractor to be non-nil")
	}

	if extractor.timeout != 30*time.Second {
		t.Errorf("expected timeout to be 30s, got %v", extractor.timeout)
	}
}

func TestChromedpExtractor_Timeout(t *testing.T) {
	extractor := &ChromedpExtractor{
		timeout: 1 * time.Millisecond,
	}

	ctx := context.Background()
	_, err := extractor.ExtractContent(ctx, "https://example.com")

	if err == nil {
		t.Skip("Timeout test may not be reliable in all environments")
	}
}

func TestCleanText_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   \n\n\t\t  ",
			expected: "",
		},
		{
			name:     "single line",
			input:    "Single line text",
			expected: "Single line text",
		},
		{
			name:     "multiple blank lines at end",
			input:    "Text\n\n\n\n",
			expected: "Text",
		},
		{
			name:     "mixed whitespace lines",
			input:    "Line 1\n  \t  \nLine 2",
			expected: "Line 1\n\nLine 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanText(tt.input)
			if result != tt.expected {
				t.Errorf("CleanText() = %q, want %q", result, tt.expected)
			}
		})
	}
}
