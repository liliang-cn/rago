package providers

import (
	"strings"
)

// StreamBuffer handles buffering for think tag removal in streaming responses
type StreamBuffer struct {
	buffer   strings.Builder
	inThink  bool
	callback func(string)
}

// NewStreamBuffer creates a new streaming buffer for think tag filtering
func NewStreamBuffer(callback func(string)) *StreamBuffer {
	return &StreamBuffer{
		callback: callback,
	}
}

// Process handles incoming chunks and filters think tags
func (sb *StreamBuffer) Process(chunk string) {
	sb.buffer.WriteString(chunk)
	content := sb.buffer.String()
	
	for {
		if sb.inThink {
			// Look for closing tag
			endIdx := strings.Index(content, "</think>")
			if endIdx != -1 {
				// Found closing tag, remove everything up to and including it
				content = content[endIdx+8:]
				sb.inThink = false
				sb.buffer.Reset()
				sb.buffer.WriteString(content)
			} else {
				// No closing tag yet, keep buffering
				return
			}
		} else {
			// Look for opening tag
			startIdx := strings.Index(content, "<think>")
			if startIdx != -1 {
				// Found opening tag, emit everything before it
				if startIdx > 0 {
					sb.callback(content[:startIdx])
				}
				content = content[startIdx+7:]
				sb.inThink = true
				sb.buffer.Reset()
				sb.buffer.WriteString(content)
			} else {
				// No complete tag, but check if we have a partial opening tag at the end
				// We need to keep potential partial tags in buffer
				possiblePartial := false
				for i := len(content) - 1; i >= 0 && i >= len(content)-7; i-- {
					if content[i] == '<' {
						// Might be start of "<think>"
						if strings.HasPrefix("<think>", content[i:]) {
							possiblePartial = true
							if i > 0 {
								sb.callback(content[:i])
							}
							sb.buffer.Reset()
							sb.buffer.WriteString(content[i:])
							break
						}
					}
				}
				
				if !possiblePartial {
					// No partial tags, emit everything
					if len(content) > 0 {
						sb.callback(content)
					}
					sb.buffer.Reset()
				}
				return
			}
		}
	}
}

// Flush emits any remaining buffered content
func (sb *StreamBuffer) Flush() {
	remaining := sb.buffer.String()
	if len(remaining) > 0 && !sb.inThink {
		// Check if remaining content might be a partial think tag
		// Don't emit if it starts with "<" and could be a partial "<think>"
		if !strings.HasPrefix(remaining, "<") || !strings.HasPrefix("<think>", remaining) {
			sb.callback(remaining)
		}
	}
	sb.buffer.Reset()
}