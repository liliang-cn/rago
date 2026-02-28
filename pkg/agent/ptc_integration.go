package agent

import (
	"context"
	"fmt"
	"os"
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
	router  *ptc.RAGORouter // used to enumerate callTool()-accessible tools for prompts
}

// PTCConfig configures PTC integration
type PTCConfig struct {
	// Enabled enables PTC mode
	Enabled bool `json:"enabled" mapstructure:"enabled"`

	// MaxToolCalls limits the number of tool calls in a single execution
	MaxToolCalls int `json:"max_tool_calls" mapstructure:"max_tool_calls"`

	// Timeout is the maximum execution time
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`

	// Debug enables verbose logging for PTC execution
	Debug bool `json:"debug" mapstructure:"debug"`

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
		Timeout:      600 * time.Second,
		Runtime:      "goja",
		AllowedTools: []string{}, // Empty means all tools allowed
	}
}

// NewPTCIntegration creates a new PTC integration instance
func NewPTCIntegration(config PTCConfig, router *ptc.RAGORouter) (*PTCIntegration, error) {
	if !config.Enabled {
		return &PTCIntegration{config: &config, router: router}, nil
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
		router:  router,
	}, nil
}

// IsCodeResponse checks if the LLM response contains executable code
func (p *PTCIntegration) IsCodeResponse(content string) bool {
	// Primary: <code>...</code> tags
	if strings.Contains(content, "<code>") {
		return true
	}

	// Fallback: markdown fences
	if strings.Contains(content, "```javascript") ||
		strings.Contains(content, "```js") ||
		strings.Contains(content, "```") {
		return true
	}

	// Heuristic: callTool() calls
	if strings.Contains(content, "callTool(") {
		return true
	}

	return false
}

// ExtractCode extracts JavaScript code from LLM response.
// Priority: <code> tags > markdown fences > legacy markers > bare code heuristic.
func (p *PTCIntegration) ExtractCode(content string) string {
	// Priority 1: <code>...</code> tags — the format we explicitly request
	if code := extractBetweenTags(content, "<code>", "</code>"); code != "" {
		return code
	}

	// Priority 2: markdown fences and legacy markers
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

// ExecuteJavascriptTool is the tool-dispatch entry point for the "execute_javascript" tool.
// It extracts the "code" and optional "context" arguments from the LLM tool call, runs
// the code in the sandbox, and returns a human-readable string result.
func (p *PTCIntegration) ExecuteJavascriptTool(ctx context.Context, args map[string]interface{}) (string, error) {
	if !p.config.Enabled || p.service == nil {
		return "", fmt.Errorf("PTC is not enabled")
	}

	code, _ := args["code"].(string)
	if code == "" {
		return "", fmt.Errorf("execute_javascript: 'code' argument is required")
	}

	// Sanitise: some models append free-form text or JSON after the JS code.
	// Try to extract just the code portion.
	code = sanitiseJSCode(code)

	// Extract optional context variables
	var contextVars map[string]interface{}
	if ctxRaw, ok := args["context"]; ok {
		if ctxMap, ok := ctxRaw.(map[string]interface{}); ok {
			contextVars = ctxMap
		}
	}

	execResult, err := p.ExecuteCode(ctx, code, contextVars)
	if err != nil {
		return fmt.Sprintf("execute_javascript failed: %v", err), nil //nolint:nilerr
	}

	result := &PTCResult{
		Type:            PTCResultTypeExecuted,
		OriginalContent: code,
		Code:            code,
		ExecutionResult: execResult,
	}
	if !execResult.Success {
		result.Type = PTCResultTypeError
		result.Error = execResult.Error
	}

	return result.FormatForLLM(), nil
}

// ExecuteCode executes JavaScript code in the sandbox.
func (p *PTCIntegration) ExecuteCode(ctx context.Context, code string, contextVars map[string]interface{}) (*ptc.ExecutionResult, error) {
	if !p.config.Enabled || p.service == nil {
		return nil, fmt.Errorf("PTC is not enabled")
	}

	// WRAPPER: Goja returns the value of the last statement.
	// If the code defines a main function and calls it at the end, 
	// we wrap it to ensure that value is returned.
	wrappedCode := fmt.Sprintf("return (function(){\n%s\n})()", code)

	// Build execution request
	req := &ptc.ExecutionRequest{
		Code:        wrappedCode,
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

// GetAvailableCallTools returns all tools accessible via callTool() inside the JS sandbox.
// This is the dynamic equivalent of Anthropic's allowed_callers field.
// Tools in BlockedTools or with blocked categories are excluded.
func (p *PTCIntegration) GetAvailableCallTools(ctx context.Context) []ptc.ToolInfo {
	if p.router == nil {
		return nil
	}
	all, err := p.router.ListAvailableTools(ctx)
	if err != nil {
		return nil
	}

	// Build blocked set from config
	blocked := make(map[string]bool, len(p.config.BlockedTools))
	for _, name := range p.config.BlockedTools {
		blocked[name] = true
	}

	// If AllowedTools is set, only include those
	if len(p.config.AllowedTools) > 0 {
		allowed := make(map[string]bool, len(p.config.AllowedTools))
		for _, name := range p.config.AllowedTools {
			allowed[name] = true
		}
		var filtered []ptc.ToolInfo
		for _, t := range all {
			if allowed[t.Name] && !blocked[t.Name] {
				filtered = append(filtered, t)
			}
		}
		return filtered
	}

	// Otherwise return all except blocked
	var filtered []ptc.ToolInfo
	for _, t := range all {
		if !blocked[t.Name] {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// GetPTCTools returns PTC-specific tool definitions for LLM.
// availableTools is the list of tools the LLM can call via callTool() inside the sandbox;
// it is embedded in the description so the model knows what is callable.
func (p *PTCIntegration) GetPTCTools(availableTools []ptc.ToolInfo) []domain.ToolDefinition {
	if !p.config.Enabled {
		return nil
	}

	var toolsDesc string
	if len(availableTools) > 0 {
		var sb strings.Builder
		sb.WriteString("\n\nTools available via callTool(name, args):\n")
		for _, t := range availableTools {
			sb.WriteString(fmt.Sprintf("  - %s: %s\n", t.Name, t.Description))
		}
		toolsDesc = sb.String()
	}

	return []domain.ToolDefinition{
		{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "execute_javascript",
				Description: "Execute JavaScript code in a secure sandbox. Use this to call multiple tools in one shot, process large results before they reach your context, or orchestrate complex multi-step logic." + toolsDesc,
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"code": map[string]interface{}{
							"type":        "string",
							"description": "Valid JavaScript code. Use callTool(name, args) to invoke tools. Return a value with the 'return' statement.",
						},
						"context": map[string]interface{}{
							"type":        "object",
							"description": "Optional variables to inject into the sandbox scope",
						},
					},
					"required": []string{"code"},
				},
			},
		},
	}
}

// GetPTCSystemPrompt returns system prompt additions for PTC mode.
func (p *PTCIntegration) GetPTCSystemPrompt(availableTools []ptc.ToolInfo) string {
	if !p.config.Enabled {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(`## Programmatic Tool Calling (PTC)

You have access to a Goja (JavaScript ES5.1+) sandbox. Use it to orchestrate tools and process data.

### ⚠️ PERFORMANCE & TIMEOUT (CRITICAL)
- **NO REASONING**: DO NOT use chain-of-thought, reasoning content, or long explanations.
- **NO PREAMBLE**: Start your response directly with the <code> block.
- **IMMEDIATE CODE**: Output the <code> block immediately to avoid 504 Gateway Timeouts.

### ⚠️ SANDBOX CONSTRAINTS
1. **NO ASYNC/AWAIT**: The environment is synchronous. Do NOT use "async" or "await".
2. **NO PROMISES**: Do NOT use Promises, .then(), or .catch().
3. **NO MODULES**: No "require()", "import", or Node.js built-ins (fs, path, etc.).
4. **NO MARKDOWN FENCES**: Do NOT wrap your response in ` + "```" + `javascript or ` + "```" + `.
5. **ONLY USE <code>**: Your entire response MUST be wrapped in <code>...</code> tags.
6. **NO FUNCTION WRAPPER**: NEVER write ` + "`function main(){...}main()`" + `. Write top-level code directly.
7. **TOP-LEVEL RETURN**: Your code MUST end with a ` + "`return`" + ` statement at the top level.

### API REFERENCE
- callTool(name, args): Executes a tool synchronously and returns the result object.
- console.log(msg): Logs to the debug console.

### EXAMPLE 1: Single Tool Call
<code>
const result = callTool('echo', { message: 'hello' });
return { echoed: result };
</code>

### EXAMPLE 2: Multiple Parallel Tool Calls
<code>
const r1 = callTool('mcp_everything_echo', { message: 'Hello' });
const r2 = callTool('mcp_everything_echo', { message: 'World' });
const combined = r1 + ' ' + r2;
return { combined: combined, parts: [r1, r2] };
</code>

### EXAMPLE 3: File Processing
<code>
const file = callTool('mcp_filesystem_read_text_file', { path: 'go.mod' });
const lines = file.split('\n');
const moduleName = lines[0].split(' ')[1];
return { module: moduleName, totalLines: lines.length };
</code>

### EXAMPLE 4: Filtering with Loop
<code>
const files = callTool('mcp_filesystem_search_files', { pattern: '**/*.go' });
const testFiles = [];
for (var i = 0; i < files.length; i++) {
  if (files[i].indexOf('_test.go') !== -1) {
    testFiles.push(files[i]);
  }
}
return { total: files.length, testCount: testFiles.length };
</code>

### EXAMPLE 5: Tool Chaining
<code>
const content = callTool('mcp_filesystem_read_text_file', { path: 'main.go' });
const review = callTool('code_review', { code: content });
return { filename: 'main.go', review: review };
</code>

### EXAMPLE 6: Skill Invocation
<code>
const code = 'func add(a, b int) int { return a + b }';
const review = callTool('code-reviewer', { code: code });
return { code: code, review: review };
</code>

### EXAMPLE 7: Conditional Logic
<code>
const status = callTool('check_status', {});
var result;
if (status.active) {
  result = callTool('start_process', { id: status.id });
} else {
  result = 'inactive';
}
return { status: status, result: result };
</code>

### MANDATORY RULE
Respond ONLY with the <code> block. No preamble. No postamble. No markdown. No function wrappers.
`)

	if len(availableTools) > 0 {
		sb.WriteString("\n### Available Tools for callTool()\n")
		for _, t := range availableTools {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", t.Name, t.Description))
		}
	}

	return sb.String()
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

	// CLEANUP: Automatically fix LLM "bad habits"
	code = sanitiseJSCode(code)

	result.Code = code
	result.Type = PTCResultTypeCode

	// Execute code if PTC is enabled
	if p.config.Enabled && p.service != nil {
		if os.Getenv("DEBUG") != "" {
			fmt.Printf("\n--- [DEBUG] Executing PTC JavaScript (Sanitized) ---\n%s\n---------------------------------------\n\n", code)
		}
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
	Type            PTCResultType        `json:"type"`
	OriginalContent string               `json:"original_content"`
	Code            string               `json:"code,omitempty"`
	ExecutionResult *ptc.ExecutionResult `json:"execution_result,omitempty"`
	Error           string               `json:"error,omitempty"`
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
			sb.WriteString("**Status:** Success ✅\n")
		} else {
			sb.WriteString("**Status:** Failed ❌\n")
			sb.WriteString(fmt.Sprintf("**Error:** %s\n", r.ExecutionResult.Error))
		}

		// Always show Return Value section
		if r.ExecutionResult.ReturnValue != nil {
			sb.WriteString(fmt.Sprintf("**Return Value:** %+v\n", r.ExecutionResult.ReturnValue))
		} else {
			sb.WriteString("**Return Value:** (none - did you forget to 'return' in JS?)\n")
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
