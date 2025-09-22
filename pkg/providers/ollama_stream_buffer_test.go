package providers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStreamBuffer(t *testing.T) {
	tests := []struct {
		name     string
		chunks   []string
		expected []string
	}{
		{
			name: "Complete think tag in single chunk",
			chunks: []string{
				"Before <think>internal</think> After",
			},
			expected: []string{
				"Before ",
				" After",
			},
		},
		{
			name: "Think tag split across chunks",
			chunks: []string{
				"Start <thi",
				"nk>internal thought</thi",
				"nk> End",
			},
			expected: []string{
				"Start ",
				" End",
			},
		},
		{
			name: "Multiple think tags across chunks",
			chunks: []string{
				"First <think>one</think>",
				" Middle <think>two</thi",
				"nk> Last",
			},
			expected: []string{
				"First ",
				" Middle ",
				" Last",
			},
		},
		{
			name: "Partial tag at end of stream",
			chunks: []string{
				"Content <thi",
			},
			expected: []string{
				"Content ",
				// The partial "<thi" is not emitted since it could be start of <think>
			},
		},
		{
			name: "No think tags",
			chunks: []string{
				"Just ",
				"regular ",
				"content",
			},
			expected: []string{
				"Just ",
				"regular ",
				"content",
			},
		},
		{
			name: "Think tag at start",
			chunks: []string{
				"<think>hidden</think>Visible",
			},
			expected: []string{
				"Visible",
			},
		},
		{
			name: "Empty think tag",
			chunks: []string{
				"Before<think></think>After",
			},
			expected: []string{
				"Before",
				"After",
			},
		},
		{
			name: "Very long think tag across many chunks",
			chunks: []string{
				"Start <think>This ",
				"is a very ",
				"long internal ",
				"thought that spans ",
				"multiple chunks",
				"</think> Actual content",
			},
			expected: []string{
				"Start ",
				" Actual content",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output []string
			buffer := NewStreamBuffer(func(s string) {
				output = append(output, s)
			})

			for _, chunk := range tt.chunks {
				buffer.Process(chunk)
			}
			buffer.Flush()

			assert.Equal(t, tt.expected, output)
		})
	}
}

func TestStreamBufferPartialTagHandling(t *testing.T) {
	// Test that partial tags are properly buffered
	chunks := []string{
		"Text before <",
		"thi",
		"nk>content</think> after",
	}
	
	var output strings.Builder
	buffer := NewStreamBuffer(func(s string) {
		output.WriteString(s)
	})
	
	for _, chunk := range chunks {
		buffer.Process(chunk)
	}
	buffer.Flush()
	
	assert.Equal(t, "Text before  after", output.String())
}

func TestStreamBufferFlushBehavior(t *testing.T) {
	// Test that flush doesn't emit content if we're inside a think tag
	chunks := []string{
		"Start <think>unfinished",
	}
	
	var output []string
	buffer := NewStreamBuffer(func(s string) {
		output = append(output, s)
	})
	
	for _, chunk := range chunks {
		buffer.Process(chunk)
	}
	buffer.Flush()
	
	// Should only emit "Start " since we're inside an unclosed think tag
	assert.Equal(t, []string{"Start "}, output)
}

func BenchmarkStreamBuffer(b *testing.B) {
	chunks := []string{
		"Start <think>This is ",
		"some internal thinking ",
		"that should be removed</think>",
		" Middle content <think>More ",
		"thinking here</think> End",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buffer := NewStreamBuffer(func(s string) {})
		for _, chunk := range chunks {
			buffer.Process(chunk)
		}
		buffer.Flush()
	}
}