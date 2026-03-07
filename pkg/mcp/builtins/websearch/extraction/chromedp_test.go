package extraction

import (
	"context"
	"strings"
	"testing"
)

func TestCleanText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes extra newlines",
			input:    "Line 1\n\n\n\nLine 2",
			expected: "Line 1\n\nLine 2",
		},
		{
			name:     "trims whitespace",
			input:    "  \n  Line 1  \n  Line 2  \n  ",
			expected: "Line 1\nLine 2",
		},
		{
			name:     "removes empty lines",
			input:    "Line 1\n\n\n  \n\nLine 2",
			expected: "Line 1\n\nLine 2",
		},
		{
			name:     "handles normal text",
			input:    "Line 1\nLine 2\nLine 3",
			expected: "Line 1\nLine 2\nLine 3",
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

func TestChromedpExtractor_ExtractContent(t *testing.T) {
	t.Skip("Skipping browser-based test in unit tests")

	extractor := NewChromedpExtractor()
	ctx := context.Background()

	content, err := extractor.ExtractContent(ctx, "https://example.com")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if content == "" {
		t.Error("expected non-empty content")
	}

	if !strings.Contains(content, "Example Domain") {
		t.Error("expected content to contain page title")
	}
}
