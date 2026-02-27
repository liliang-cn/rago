package agent

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/ptc"
	"github.com/liliang-cn/rago/v2/pkg/ptc/runtime/goja"
	"github.com/liliang-cn/rago/v2/pkg/ptc/runtime/wazero"
)

// PTCIntegration handles Programmatic Tool Calling integration
// This allows LLMs to generate JavaScript code instead of JSON tool calls
type PTCIntegration struct {
	service *ptc.Service
	config  *PTCConfig
}

// PTCConfig configures PTC integration
type PTCConfig struct {
	// Enabled enables PTC mode
	Enabled bool `json:"enabled" mapstructure:"enabled"`

	// MaxToolCalls limits the number of tool calls in a single execution
	MaxToolCalls int `json:"max_tool_calls" mapstructure:"max_tool_calls"`

	// Timeout is the maximum execution time
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`

	// AllowedTools is a whitelist of tools that can be called from code
	AllowedTools []string `json:"allowed_tools" mapstructure:"allowed_tools"`

	// BlockedTools is a blacklist of tools that cannot be called
	BlockedTools []string `json:"blocked_tools" mapstructure:"blocked_tools"`

	// Runtime to use: "goja" or "wazero"
	Runtime string `json:"runtime" mapstructure:"runtime"`
}

// DefaultPTCConfig returns default PTC configuration
func DefaultPTCConfig() PTCConfig {
	return PTCConfig{
		Enabled:      false,
		MaxToolCalls: 20,
		Timeout:      30 * time.Second,
		Runtime:      "goja",
		AllowedTools: []string{}, // Empty means all tools allowed
	}
}

// NewPTCIntegration creates a new PTC integration instance
func NewPTCIntegration(config PTCConfig, router *ptc.RAGORouter) (*PTCIntegration, error) {
	if !config.Enabled {
		return &PTCIntegration{config: &config}, nil
	}

	// Create PTC service
	ptcConfig := ptc.DefaultConfig()
	ptcConfig.Enabled = true
	ptcConfig.MaxToolCalls = config.MaxToolCalls
	ptcConfig.DefaultTimeout = config.Timeout

	// Create store for execution history
	store := NewPTCMemoryStore(100)

	service, err := ptc.NewService(&ptcConfig, router, store)
	if err != nil {
		return nil, fmt.Errorf("failed to create PTC service: %w", err)
	}

	// Set runtime
	var runtime ptc.SandboxRuntime
	switch config.Runtime {
	case "wazero":
		runtime = wazero.NewRuntimeWithConfig(&ptcConfig)
	default:
		runtime = goja.NewRuntimeWithConfig(&ptcConfig)
	}
	service.SetRuntime(runtime)

	return &PTCIntegration{
		service: service,
		config:  &config,
	}, nil
}

// IsCodeResponse checks if the LLM response contains executable code
func (p *PTCIntegration) IsCodeResponse(content string) bool {
	// Check for code block markers
	if strings.Contains(content, "```javascript") ||
		strings.Contains(content, "```js") ||
		strings.Contains(content, "```ts") ||
		strings.Contains(content, "```typescript") {
		return true
	}

	// Check for PTC-specific markers
	if strings.Contains(content, "<ptc_code>") ||
		strings.Contains(content, "[PTC_CODE]") {
		return true
	}

	// Check for common code patterns
	codePatterns := []string{
		`callTool\s*\(`,
		`async\s+function`,
		`function\s+\w+\s*\(`,
		`const\s+\w+\s*=`,
		`let\s+\w+\s*=`,
		`export\s+default`,
	}

	for _, pattern := range codePatterns {
		if matched, _ := regexp.MatchString(pattern, content); matched {
			return true
		}
	}

	return false
}

// ExtractCode extracts JavaScript code from LLM response
func (p *PTCIntegration) ExtractCode(content string) string {
	// Try to extract from code blocks first
	codeBlockPatterns := []struct {
		start string
		end   string
	}{
		{"```javascript", "```"},
		{"```js", "```"},
		{"```typescript", "```"},
		{"```ts", "```"},
		{"```\n", "```"},
		{"<ptc_code>", "</ptc_code>"},
		{"[PTC_CODE]", "[/PTC_CODE]"},
	}

	for _, pattern := range codeBlockPatterns {
		startIdx := strings.Index(content, pattern.start)
		if startIdx != -1 {
			codeStart := startIdx + len(pattern.start)
			endIdx := strings.Index(content[codeStart:], pattern.end)
			if endIdx != -1 {
				code := content[codeStart : codeStart+endIdx]
				return strings.TrimSpace(code)
			}
		}
	}

	// If no code blocks found, check if entire content looks like code
	// This handles cases where LLM returns code without markers
	if p.looksLikeCode(content) {
		return strings.TrimSpace(content)
	}

	return ""
}

// looksLikeCode heuristically determines if content looks like code
func (p *PTCIntegration) looksLikeCode(content string) bool {
	// Count code-like patterns
	patterns := []string{
		`callTool\s*\(`,
		`console\.log`,
		`return\s+`,
		`var\s+\w+`,
		`let\s+\w+`,
		`const\s+\w+`,
		`async\s+`,
		`await\s+`,
		`function\s+`,
		`=>\s*{`,
		`{\s*\n`,
		`}\s*;`,
	}

	count := 0
	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, content); matched {
			count++
		}
	}

	// If 3 or more patterns match, it's likely code
	return count >= 3
}

// ExecuteCode executes JavaScript code in the PTC sandbox
func (p *PTCIntegration) ExecuteCode(ctx context.Context, code string, contextVars map[string]interface{}) (*ptc.ExecutionResult, error) {
	if !p.config.Enabled || p.service == nil {
		return nil, fmt.Errorf("PTC is not enabled")
	}

	// Build execution request
	req := &ptc.ExecutionRequest{
		Code:        code,
		Language:    ptc.LanguageJavaScript,
		Context:     contextVars,
		Tools:       p.config.AllowedTools,
		Timeout:     p.config.Timeout,
		MaxMemoryMB: 64,
	}

	// Execute
	return p.service.Execute(ctx, req)
}

// ShouldUsePTC determines if PTC should be used for a given request
func (p *PTCIntegration) ShouldUsePTC(userMessage string, systemPrompt string) bool {
	if !p.config.Enabled {
		return false
	}

	// Check for PTC-specific keywords in the request
	ptcKeywords := []string{
		"write code",
		"generate code",
		"script",
		"program",
		"automate",
		"execute",
		"run code",
		"javascript",
	}

	lowerMsg := strings.ToLower(userMessage)
	for _, keyword := range ptcKeywords {
		if strings.Contains(lowerMsg, keyword) {
			return true
		}
	}

	return false
}

// GetPTCTools returns PTC-specific tool definitions for LLM
func (p *PTCIntegration) GetPTCTools() []domain.ToolDefinition {
	if !p.config.Enabled {
		return nil
	}

	return []domain.ToolDefinition{
		{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "execute_javascript",
				Description: "Execute JavaScript code in a secure sandbox. Use this when you need to perform complex operations, data transformations, or call multiple tools in sequence. Available functions: callTool(toolName, args) - call any registered tool.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"code": map[string]interface{}{
							"type":        "string",
							"description": "JavaScript code to execute. Use callTool(name, args) to call tools. Return values with 'return' statement.",
						},
						"context": map[string]interface{}{
							"type":        "object",
							"description": "Optional context variables to inject into the sandbox",
						},
					},
					"required": []string{"code"},
				},
			},
		},
	}
}

// GetPTCSystemPrompt returns system prompt additions for PTC mode
func (p *PTCIntegration) GetPTCSystemPrompt() string {
	if !p.config.Enabled {
		return ""
	}

	return `
## Programmatic Tool Calling (PTC) Mode

You can execute JavaScript code to perform complex operations. When appropriate, use the execute_javascript tool.

### Available in Sandbox:
- callTool(name, args) - Call any registered tool by name
- console.log(...args) - Log messages for debugging

### Example:
` + "```javascript" + `
// Call a single tool
const result = callTool('rag_query', { query: 'search term' });
console.log('Result:', result);

// Chain multiple tools
const docs = callTool('rag_list', {});
const queryResult = callTool('rag_query', { query: docs[0].title });

// Return final result
return queryResult;
` + "```" + `

### Guidelines:
1. Use PTC when you need to call multiple tools with complex logic
2. Handle errors gracefully with try-catch
3. Always return a meaningful result
4. Keep code simple and focused
`
}

// ProcessLLMResponse processes an LLM response and executes any code found
func (p *PTCIntegration) ProcessLLMResponse(ctx context.Context, content string, contextVars map[string]interface{}) (*PTCResult, error) {
	result := &PTCResult{
		OriginalContent: content,
	}

	// Check if response contains code
	if !p.IsCodeResponse(content) {
		result.Type = PTCResultTypeText
		return result, nil
	}

	// Extract code
	code := p.ExtractCode(content)
	if code == "" {
		result.Type = PTCResultTypeText
		return result, nil
	}

	result.Code = code
	result.Type = PTCResultTypeCode

	// Execute code if PTC is enabled
	if p.config.Enabled && p.service != nil {
		execResult, err := p.ExecuteCode(ctx, code, contextVars)
		if err != nil {
			result.Error = err.Error()
			result.Type = PTCResultTypeError
			return result, nil
		}

		result.ExecutionResult = execResult
		result.Type = PTCResultTypeExecuted
	}

	return result, nil
}

// PTCResultType indicates the type of PTC result
type PTCResultType string

const (
	PTCResultTypeText     PTCResultType = "text"     // No code found
	PTCResultTypeCode     PTCResultType = "code"     // Code found but not executed
	PTCResultTypeExecuted PTCResultType = "executed" // Code was executed
	PTCResultTypeError    PTCResultType = "error"    // Execution error
)

// PTCResult contains the result of PTC processing
type PTCResult struct {
	Type            PTCResultType         `json:"type"`
	OriginalContent string                `json:"original_content"`
	Code            string                `json:"code,omitempty"`
	ExecutionResult *ptc.ExecutionResult  `json:"execution_result,omitempty"`
	Error           string                `json:"error,omitempty"`
}

// FormatForLLM formats the PTC result for sending back to LLM
func (r *PTCResult) FormatForLLM() string {
	switch r.Type {
	case PTCResultTypeText:
		return r.OriginalContent

	case PTCResultTypeExecuted:
		if r.ExecutionResult == nil {
			return r.OriginalContent
		}

		var sb strings.Builder
		sb.WriteString("Code execution completed.\n")

		if r.ExecutionResult.Success {
			sb.WriteString("**Status:** Success\n")
		} else {
			sb.WriteString("**Status:** Failed\n")
			sb.WriteString(fmt.Sprintf("**Error:** %s\n", r.ExecutionResult.Error))
		}

		if r.ExecutionResult.ReturnValue != nil {
			sb.WriteString(fmt.Sprintf("**Return Value:** %v\n", r.ExecutionResult.ReturnValue))
		}

		if len(r.ExecutionResult.ToolCalls) > 0 {
			sb.WriteString(fmt.Sprintf("\n**Tool Calls (%d):**\n", len(r.ExecutionResult.ToolCalls)))
			for _, tc := range r.ExecutionResult.ToolCalls {
				sb.WriteString(fmt.Sprintf("- %s", tc.ToolName))
				if tc.Error != "" {
					sb.WriteString(fmt.Sprintf(" (Error: %s)", tc.Error))
				} else {
					sb.WriteString(" ✓")
				}
				sb.WriteString("\n")
			}
		}

		if len(r.ExecutionResult.Logs) > 0 {
			sb.WriteString("\n**Logs:**\n")
			for _, log := range r.ExecutionResult.Logs {
				sb.WriteString(fmt.Sprintf("  %s\n", log))
			}
		}

		return sb.String()

	case PTCResultTypeError:
		return fmt.Sprintf("Code execution failed: %s\n\nOriginal response:\n%s", r.Error, r.OriginalContent)

	default:
		return r.OriginalContent
	}
}

// PTCMemoryStore is a simple in-memory store for PTC execution history
type PTCMemoryStore struct {
	records map[string]*ptc.ExecutionHistory
	maxSize int
	mu      sync.Mutex
}

// NewPTCMemoryStore creates a new memory store
func NewPTCMemoryStore(maxSize int) *PTCMemoryStore {
	return &PTCMemoryStore{
		records: make(map[string]*ptc.ExecutionHistory),
		maxSize: maxSize,
	}
}

// Save saves an execution history
func (s *PTCMemoryStore) Save(ctx context.Context, history *ptc.ExecutionHistory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Enforce max size
	if len(s.records) >= s.maxSize {
		// Remove oldest (simple approach)
		var oldestKey string
		var oldestTime time.Time
		for k, h := range s.records {
			if oldestKey == "" || h.ExecutedAt.Before(oldestTime) {
				oldestKey = k
				oldestTime = h.ExecutedAt
			}
		}
		if oldestKey != "" {
			delete(s.records, oldestKey)
		}
	}

	s.records[history.ID] = history
	return nil
}

// Get retrieves an execution history
func (s *PTCMemoryStore) Get(ctx context.Context, id string) (*ptc.ExecutionHistory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	history, ok := s.records[id]
	if !ok {
		return nil, fmt.Errorf("execution not found: %s", id)
	}
	return history, nil
}

// List lists execution histories
func (s *PTCMemoryStore) List(ctx context.Context, limit int) ([]*ptc.ExecutionHistory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	results := make([]*ptc.ExecutionHistory, 0, len(s.records))
	for _, h := range s.records {
		results = append(results, h)
		if limit > 0 && len(results) >= limit {
			break
		}
	}
	return results, nil
}

// Delete removes executions older than the given time
func (s *PTCMemoryStore) Delete(ctx context.Context, before time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for k, h := range s.records {
		if h.ExecutedAt.Before(before) {
			delete(s.records, k)
		}
	}
	return nil
}
