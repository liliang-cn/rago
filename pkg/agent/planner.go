package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// Planner generates execution plans from goals using an LLM
type Planner struct {
	llmService domain.Generator
	tools      []domain.ToolDefinition
}

// NewPlanner creates a new planner
func NewPlanner(llmService domain.Generator, tools []domain.ToolDefinition) *Planner {
	return &Planner{
		llmService: llmService,
		tools:      tools,
	}
}

// PlanRequest represents a planning request
type PlanRequest struct {
	Goal       string `json:"goal"`
	Context    string `json:"context,omitempty"`
	SessionID  string `json:"session_id"`
}

// IntentRecognitionResult represents the recognized intent from the goal
type IntentRecognitionResult struct {
	IntentType     string   `json:"intent_type"`     // file_create, file_read, web_search, rag_query, general_qa, etc
	TargetFile     string   `json:"target_file"`     // extracted file path if any
	Topic          string   `json:"topic"`          // main topic/subject
	Requirements    []string `json:"requirements"`    // specific requirements extracted
	Confidence      float64  `json:"confidence"`      // confidence score
}

// PlanResponse represents the LLM's plan response
type PlanResponse struct {
	Reasoning string  `json:"reasoning"`
	Steps     []Step  `json:"steps"`
}

// Plan generates an execution plan for the given goal
func (p *Planner) Plan(ctx context.Context, goal string, session *Session) (*Plan, error) {
	// Step 1: Intent Recognition with context
	intent, err := p.recognizeIntent(ctx, goal, session)
	if err != nil {
		// Fall back to planning without intent context
		intent = &IntentRecognitionResult{
			IntentType:  "general",
			Confidence:  0.0,
		}
	}

	// Step 2: Build system prompt for planning (includes available tools and intent context)
	systemPrompt := p.buildSystemPrompt()

	// Step 3: Build user prompt with goal, context, and intent recognition results
	userPrompt := p.buildUserPromptWithContext(goal, session, intent)

	// Combine system prompt with user prompt for the LLM
	fullPrompt := systemPrompt + "\n\n" + userPrompt

	// Define the expected response schema
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"reasoning": map[string]interface{}{
				"type":        "string",
				"description": "Explanation of the plan and why these steps are necessary",
			},
			"steps": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"description": map[string]interface{}{
							"type":        "string",
							"description": "What this step does",
						},
						"tool": map[string]interface{}{
							"type":        "string",
							"description": "The tool to use for this step. MUST be one of the available tools listed above, or 'llm' for general reasoning.",
							"enum":         p.buildToolEnum(),
						},
						"arguments": map[string]interface{}{
							"type":        "object",
							"description": "Arguments for the tool (use null if not needed)",
						},
					},
					"required": []string{"description", "tool", "arguments"},
				},
			},
		},
		"required": []string{"reasoning", "steps"},
	}

	// Generate structured plan
	opts := &domain.GenerationOptions{
		Temperature: 0.3, // Lower temperature for more consistent planning
		MaxTokens:   2000,
	}

	result, err := p.llmService.GenerateStructured(ctx, fullPrompt, schema, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate plan: %w", err)
	}

	// Parse the structured response
	var planResp PlanResponse
	rawBytes := []byte(result.Raw)
	if err := json.Unmarshal(rawBytes, &planResp); err != nil {
		return nil, fmt.Errorf("failed to parse plan response: %w", err)
	}

	// Validate tool names and apply intent-based corrections
	for i := range planResp.Steps {
		// Check if the tool is valid
		if !p.isValidTool(planResp.Steps[i].Tool) {
			// Use intent-aware tool inference
			planResp.Steps[i].Tool = p.inferToolFromIntent(planResp.Steps[i].Description, intent)
			if planResp.Steps[i].Arguments == nil {
				planResp.Steps[i].Arguments = make(map[string]interface{})
			}
		}

		// Apply intent-based corrections - more aggressive checking
		// If LLM chose 'llm' but description clearly indicates file operation, override it
		if planResp.Steps[i].Tool == "llm" {
			if p.isFileWriteDescription(planResp.Steps[i].Description) {
				// This is a file write step, override the tool
				planResp.Steps[i].Tool = p.findFilesystemTool("write")
				// Ensure arguments include path
				if intent.TargetFile != "" {
					if planResp.Steps[i].Arguments == nil {
						planResp.Steps[i].Arguments = make(map[string]interface{})
					}
					if _, exists := planResp.Steps[i].Arguments["path"]; !exists {
						planResp.Steps[i].Arguments["path"] = intent.TargetFile
					}
				}
			}
		}

		// For file_create intent, ensure content argument exists for write_file
		if intent.IntentType == "file_create" && p.isFilesystemTool(planResp.Steps[i].Tool, "write") {
			if planResp.Steps[i].Arguments == nil {
				planResp.Steps[i].Arguments = make(map[string]interface{})
			}
			// Content will be filled from previous step's output during execution
		}
	}

	// Create plan with steps
	steps := make([]Step, len(planResp.Steps))
	for i, step := range planResp.Steps {
		steps[i] = Step{
			ID:          uuid.New().String(),
			Description: step.Description,
			Tool:        step.Tool,
			Arguments:   step.Arguments,
			Status:      StepStatusPending,
		}
	}

	plan := &Plan{
		ID:        uuid.New().String(),
		Goal:      goal,
		SessionID: session.GetID(),
		Steps:     steps,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    PlanStatusPending,
		Reasoning: planResp.Reasoning,
	}

	return plan, nil
}

// buildToolEnum returns the list of valid tool names for the schema enum
func (p *Planner) buildToolEnum() []string {
	// Start with "llm" as the default
	tools := []string{"llm"}

	// Add available tools
	for _, tool := range p.tools {
		tools = append(tools, tool.Function.Name)
	}

	return tools
}

// isValidTool checks if a tool name is valid
func (p *Planner) isValidTool(toolName string) bool {
	if toolName == "llm" {
		return true
	}

	for _, tool := range p.tools {
		if tool.Function.Name == toolName {
			return true
		}
	}

	return false
}

// buildSystemPrompt creates the system prompt for planning
func (p *Planner) buildSystemPrompt() string {
	toolDescriptions := p.describeAvailableTools()

	return `You are an AI planning agent. Your task is to break down goals into clear, executable steps.

` + toolDescriptions + `

When creating a plan:
1. Think step by step about what needs to be done
2. Break down complex goals into smaller, manageable steps
3. Choose the most appropriate tool for each step
4. Provide clear arguments for each tool call
5. Reason about your plan before presenting the steps

CRITICAL - Match Your Tools to the Task:
- Creating/Saving/Writing files → MUST use mcp_filesystem_write_file (NOT llm!)
- Reading/Opening files → use mcp_filesystem_read_file (NOT llm!)
- Searching the web → use web search tools
- Querying your knowledge base → use rag_query
- Generating content → use llm
- Analyzing data → use llm

EXAMPLE WORKFLOWS:

📝 File Creation Example:
Goal: "Create a markdown document about Go and save to go.md"
Plan:
1. rag_query - Retrieve information about Go language from knowledge base
2. llm - Generate comprehensive markdown content about Go
3. mcp_filesystem_write_file - Save content to "go.md" (path="./go.md", content="{{PREVIOUS_OUTPUT}}")

📄 File Analysis Example:
Goal: "Analyze the main.go file"
Plan:
1. mcp_filesystem_read_file - Read "./main.go" (path="./main.go")
2. llm - Analyze the code structure and provide insights

🔍 Research Example:
Goal: "Find information about quantum computing"
Plan:
1. web_search - Search for quantum computing information
2. llm - Summarize findings

Return your response as JSON with:
- reasoning: Your explanation of the plan
- steps: Array of steps with description, tool, and arguments`
}

// describeAvailableTools creates a description of available tools
func (p *Planner) describeAvailableTools() string {
	if len(p.tools) == 0 {
		return "No specific tools available. Use 'llm' for general reasoning."
	}

	// Categorize tools for better understanding
	categories := map[string][]string{
		"Information Retrieval": {},
		"File Operations":       {},
		"Database":              {},
		"Web Search":            {},
		"LLM & Generation":      {},
	}

	// Always include llm
	categories["LLM & Generation"] = append(categories["LLM & Generation"], "llm: General text generation, reasoning, and analysis")

	for _, tool := range p.tools {
		name := tool.Function.Name
		desc := tool.Function.Description

		// Categorize based on tool name patterns
		if strings.Contains(name, "filesystem") || strings.Contains(name, "file_") {
			if strings.Contains(name, "write") || strings.Contains(name, "create") {
				desc = fmt.Sprintf("%s (USE THIS when goal asks to create/save/write file)", desc)
			} else if strings.Contains(name, "read") {
				desc = fmt.Sprintf("%s (USE THIS when goal asks to read/open file)", desc)
			}
			categories["File Operations"] = append(categories["File Operations"], fmt.Sprintf("%s: %s", name, desc))
		} else if strings.Contains(name, "sqlite") {
			categories["Database"] = append(categories["Database"], fmt.Sprintf("%s: %s", name, desc))
		} else if strings.Contains(name, "search") || strings.Contains(name, "web") {
			categories["Web Search"] = append(categories["Web Search"], fmt.Sprintf("%s: %s", name, desc))
		} else if strings.Contains(name, "rag") || strings.Contains(name, "query") {
			categories["Information Retrieval"] = append(categories["Information Retrieval"], fmt.Sprintf("%s: %s", name, desc))
		} else {
			categories["LLM & Generation"] = append(categories["LLM & Generation"], fmt.Sprintf("%s: %s", name, desc))
		}
	}

	// Build categorized description
	var desc strings.Builder
	desc.WriteString("Available tools (categorized):\n\n")

	for category, tools := range categories {
		if len(tools) == 0 {
			continue
		}
		desc.WriteString(fmt.Sprintf("**%s**:\n", category))
		for _, tool := range tools {
			desc.WriteString(fmt.Sprintf("  - %s\n", tool))
		}
		desc.WriteString("\n")
	}

	return desc.String()
}

// containsAny checks if the text contains any of the substrings
func containsAny(text string, substrings []string) bool {
	for _, s := range substrings {
		if strings.Contains(text, s) {
			return true
		}
	}
	return false
}

// PlanWithFallback generates a plan with a simple fallback if LLM fails
func (p *Planner) PlanWithFallback(ctx context.Context, goal string, session *Session) (*Plan, error) {
	plan, err := p.Plan(ctx, goal, session)
	if err != nil {
		// Fallback: create a simple single-step plan using LLM
		return p.createFallbackPlan(goal, session)
	}
	return plan, err
}

// createFallbackPlan creates a simple fallback plan
func (p *Planner) createFallbackPlan(goal string, session *Session) (*Plan, error) {
	sessionID := ""
	if session != nil {
		sessionID = session.GetID()
	}

	// Determine best tool based on goal content
	tool := p.selectBestTool(goal)

	plan := &Plan{
		ID:        uuid.New().String(),
		Goal:      goal,
		SessionID: sessionID,
		Steps: []Step{
			{
				ID:          uuid.New().String(),
				Description: fmt.Sprintf("Process goal: %s", goal),
				Tool:        tool,
				Arguments:   map[string]interface{}{"query": goal},
				Status:      StepStatusPending,
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    PlanStatusPending,
		Reasoning: "Fallback plan created due to LLM unavailability",
	}

	return plan, nil
}

// selectBestTool selects the best tool based on goal content
func (p *Planner) selectBestTool(goal string) string {
	// Simple heuristic-based tool selection
	for _, tool := range p.tools {
		// Check if tool name appears in goal
		if contains(goal, tool.Function.Name) {
			return tool.Function.Name
		}
	}

	// Check for specific patterns
	if contains(goal, "search") || contains(goal, "find") || contains(goal, "look up") {
		for _, tool := range p.tools {
			if contains(tool.Function.Name, "search") || contains(tool.Function.Name, "rag") {
				return tool.Function.Name
			}
		}
	}

	if contains(goal, "file") || contains(goal, "read") || contains(goal, "write") {
		for _, tool := range p.tools {
			if contains(tool.Function.Name, "file") {
				return tool.Function.Name
			}
		}
	}

	// Default to first available tool or "llm"
	if len(p.tools) > 0 {
		return p.tools[0].Function.Name
	}
	return "llm"
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ============================================================================
// Intent Recognition - LLM-based intent analysis (Step 1)
// ============================================================================

// recognizeIntent performs LLM-based intent recognition on the goal
func (p *Planner) recognizeIntent(ctx context.Context, goal string, session *Session) (*IntentRecognitionResult, error) {
	// Build intent recognition prompt
	prompt := p.buildIntentRecognitionPrompt(goal, session)

	// Define schema for intent recognition
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"intent_type": map[string]interface{}{
				"type": "string",
				"enum": []string{"file_create", "file_read", "file_edit", "web_search", "rag_query", "analysis", "general_qa"},
				"description": "The primary intent type",
			},
			"target_file": map[string]interface{}{
				"type":        "string",
				"description": "Extracted file path if applicable (e.g., './go.md', '/tmp/output.txt')",
			},
			"topic": map[string]interface{}{
				"type":        "string",
				"description": "Main topic or subject matter being discussed",
			},
			"requirements": map[string]interface{}{
				"type":        "array",
				"items":       map[string]string{"type": "string"},
				"description": "Specific requirements extracted from the goal",
			},
			"confidence": map[string]interface{}{
				"type":        "number",
				"description": "Confidence score from 0.0 to 1.0",
			},
		},
		"required": []string{"intent_type", "confidence"},
	}

	// Call LLM with low temperature for consistent classification
	opts := &domain.GenerationOptions{
		Temperature: 0.1, // Very low temperature for classification
		MaxTokens:   500,
	}

	result, err := p.llmService.GenerateStructured(ctx, prompt, schema, opts)
	if err != nil {
		// Fall back to basic intent recognition
		return p.fallbackIntentRecognition(goal), nil
	}

	// Parse response
	var intent IntentRecognitionResult
	if err := json.Unmarshal([]byte(result.Raw), &intent); err != nil {
		// Fallback on parse error
		return p.fallbackIntentRecognition(goal), nil
	}

	return &intent, nil
}

// buildIntentRecognitionPrompt creates the prompt for LLM-based intent recognition
func (p *Planner) buildIntentRecognitionPrompt(goal string, session *Session) string {
	var prompt strings.Builder

	prompt.WriteString(`You are an intent classifier. Analyze the user's goal and classify it.

Intent Types:
- file_create: User wants to create, generate, or save a file
- file_read: User wants to read, view, or analyze an existing file
- file_edit: User wants to modify or update an existing file
- web_search: User wants to search the web for current information
- rag_query: User wants to query the knowledge base/vector store
- analysis: User wants to analyze, summarize, or compare something
- general_qa: General question that doesn't fit other categories

Examples:
Goal: "Create a markdown doc about Go and save to ./go.md" → file_create, target_file: "./go.md"
Goal: "Read and analyze the main.go file" → file_read, target_file: "main.go"
Goal: "Search for latest AI news" → web_search
Goal: "What's in my knowledge base about Python?" → rag_query
Goal: "Summarize the key points of this article" → analysis
Goal: "What is the capital of France?" → general_qa

`)

	prompt.WriteString(fmt.Sprintf("User Goal: %s\n\n", goal))

	// Add session context if available
	if session != nil && len(session.GetMessages()) > 0 {
		messages := session.GetLastNMessages(3)
		if len(messages) > 0 {
			prompt.WriteString("Recent conversation:\n")
			for _, msg := range messages {
				prompt.WriteString(fmt.Sprintf("- %s: %s\n", msg.Role, msg.Content))
			}
			prompt.WriteString("\n")
		}
	}

	prompt.WriteString("Classify this goal. Return JSON with intent_type, target_file (if applicable), topic, requirements, and confidence.")

	return prompt.String()
}

// fallbackIntentRecognition provides basic regex-based intent recognition as fallback
func (p *Planner) fallbackIntentRecognition(goal string) *IntentRecognitionResult {
	intent := &IntentRecognitionResult{
		IntentType: "general_qa",
		Confidence: 0.5,
	}

	lowerGoal := strings.ToLower(goal)

	// Detect intent type with basic patterns
	if containsAny(lowerGoal, []string{"创建文件", "新建文件", "生成文件", "保存到文件", "写入文件",
		"create file", "new file", "generate file", "save to file", "write file", "create document"}) {
		intent.IntentType = "file_create"
		intent.Confidence = 0.7
		intent.TargetFile = p.extractFilePathFromGoal(goal)
	} else if containsAny(lowerGoal, []string{"读取文件", "查看文件", "打开文件", "分析文件",
		"read file", "check file", "open file", "analyze file"}) {
		intent.IntentType = "file_read"
		intent.Confidence = 0.7
		intent.TargetFile = p.extractFilePathFromGoal(goal)
	} else if containsAny(lowerGoal, []string{"搜索", "查找信息", "web search", "google", "百度"}) {
		intent.IntentType = "web_search"
		intent.Confidence = 0.6
		intent.Topic = p.extractTopic(goal)
	} else if containsAny(lowerGoal, []string{"分析", "总结", "对比", "解释"}) {
		intent.IntentType = "analysis"
		intent.Confidence = 0.6
		intent.Topic = p.extractTopic(goal)
	}

	return intent
}

// extractFilePathFromGoal extracts file path from goal text
func (p *Planner) extractFilePathFromGoal(goal string) string {
	// Look for file patterns: ./path, path.ext, "path/to/file"
	re := regexp.MustCompile(`[./]?[a-zA-Z0-9_\-./]+\.[a-z]{2,4}`)
	matches := re.FindAllString(goal, -1)
	if len(matches) > 0 {
		return matches[0]
	}

	// Look for ./path pattern
	re2 := regexp.MustCompile(`\./[a-zA-Z0-9_\-./]+`)
	matches2 := re2.FindAllString(goal, -1)
	if len(matches2) > 0 {
		path := matches2[0]
		if !strings.Contains(path, ".") {
			return path + ".md"
		}
		return path
	}

	// Look for quoted file names
	re3 := regexp.MustCompile(`["']([a-zA-Z0-9_\-./]+\.[a-z]{2,4})["']`)
	matches3 := re3.FindStringSubmatch(goal)
	if len(matches3) > 1 {
		return matches3[1]
	}

	return ""
}

// extractTopic extracts the main topic/subject from the goal
func (p *Planner) extractTopic(goal string) string {
	// Remove common verbs and prepositions
	topic := goal

	removePatterns := []string{
		"创建", "新建", "生成", "保存", "写入", "读取", "分析", "总结",
		"create", "generate", "save", "write", "read", "analyze", "summarize",
		"一个", "一篇", "一份", "a", "an", "the",
		"的", "关于", "about", "regarding", "concerning",
		"文档", "文件", "document", "file",
		"到", "至", "to", "into", "at",
	}

	for _, pattern := range removePatterns {
		topic = strings.ReplaceAll(topic, pattern, " ")
		topic = strings.ReplaceAll(topic, strings.ToUpper(pattern), " ")
	}

	// Clean up extra spaces
	words := strings.Fields(topic)
	if len(words) > 0 {
		return strings.Join(words, " ")
	}

	return goal
}

// buildUserPromptWithContext creates the user prompt with intent context
func (p *Planner) buildUserPromptWithContext(goal string, session *Session, intent *IntentRecognitionResult) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("Goal: %s\n\n", goal))

	// Add intent recognition context
	prompt.WriteString("Intent Analysis:\n")
	prompt.WriteString(fmt.Sprintf("- Type: %s\n", intent.IntentType))
	if intent.TargetFile != "" {
		prompt.WriteString(fmt.Sprintf("- Target File: %s\n", intent.TargetFile))
	}
	if intent.Topic != "" {
		prompt.WriteString(fmt.Sprintf("- Topic: %s\n", intent.Topic))
	}
	prompt.WriteString(fmt.Sprintf("- Confidence: %.1f\n\n", intent.Confidence))

	// Add specific instructions based on intent
	switch intent.IntentType {
	case "file_create":
		prompt.WriteString("⚠️  IMPORTANT: You MUST include a mcp_filesystem_write_file step to save the file!\n")
		prompt.WriteString("   The workflow should be: 1) Gather information 2) Generate content 3) Write to file\n\n")
	case "file_read":
		prompt.WriteString("⚠️  IMPORTANT: Start with mcp_filesystem_read_file to read the file first!\n\n")
	}

	// Add session context if available
	if session != nil && len(session.GetMessages()) > 0 {
		messages := session.GetLastNMessages(5)
		if len(messages) > 0 {
			prompt.WriteString("Recent conversation context:\n")
			for _, msg := range messages {
				prompt.WriteString(fmt.Sprintf("- [%s]: %s\n", msg.Role, msg.Content))
			}
			prompt.WriteString("\n")
		}
	}

	prompt.WriteString("Create a step-by-step plan to accomplish this goal. Return JSON with reasoning and steps.")

	return prompt.String()
}

// ============================================================================
// Tool Inference - Intent-aware tool selection helper
// ============================================================================

// inferToolFromIntent infers the appropriate tool based on description and recognized intent
func (p *Planner) inferToolFromIntent(description string, intent *IntentRecognitionResult) string {
	lowerDesc := strings.ToLower(description)

	// Use recognized intent to guide tool selection
	switch intent.IntentType {
	case "file_create":
		// Look for write/create keywords in description
		if containsAny(lowerDesc, []string{"保存", "写入", "创建", "save", "write", "create", "输出", "output"}) {
			return p.findFilesystemTool("write")
		}
	case "file_read":
		if containsAny(lowerDesc, []string{"读取", "打开", "查看", "read", "open", "check"}) {
			return p.findFilesystemTool("read")
		}
	case "web_search":
		for _, tool := range p.tools {
			if strings.Contains(strings.ToLower(tool.Function.Name), "search") {
				return tool.Function.Name
			}
		}
	case "rag_query":
		for _, tool := range p.tools {
			if strings.Contains(strings.ToLower(tool.Function.Name), "query") || strings.Contains(strings.ToLower(tool.Function.Name), "rag") {
				return tool.Function.Name
			}
		}
	}

	// Fallback to keyword matching
	if containsAny(lowerDesc, []string{"保存", "写入", "创建文件", "save to file", "write file", "create file"}) {
		return p.findFilesystemTool("write")
	}
	if containsAny(lowerDesc, []string{"读取", "打开", "read file", "open file"}) {
		return p.findFilesystemTool("read")
	}
	if containsAny(lowerDesc, []string{"搜索", "查找", "search", "web search"}) {
		for _, tool := range p.tools {
			if strings.Contains(strings.ToLower(tool.Function.Name), "search") {
				return tool.Function.Name
			}
		}
	}

	// Default to llm
	return "llm"
}

// isFinalContentGenerationStep checks if this step is the final content generation step
// (which should be followed by or converted to a file write step)
func (p *Planner) isFinalContentGenerationStep(description string, stepIndex, totalSteps int) bool {
	lowerDesc := strings.ToLower(description)

	// Check if description indicates writing/saving to a file
	if containsAny(lowerDesc, []string{
		"写入文件", "保存到文件", "输出到文件", "导出到", "保存为",
		"write file", "save to file", "save as", "export to",
		"写入", "保存", "write", "save",
	}) {
		return true
	}

	// If it's the last step and description mentions generating/creating content
	if stepIndex == totalSteps-1 {
		return containsAny(lowerDesc, []string{
			"生成", "创建", "generate", "create", "compose", "produce",
			"内容", "content", "文档", "document", "markdown",
		})
	}
	return false
}

// findFilesystemTool finds a filesystem tool with the given operation type
func (p *Planner) findFilesystemTool(operation string) string {
	opKeywords := map[string][]string{
		"write": {"write_file", "create_file", "write", "save"},
		"read":  {"read_file", "get_file", "open", "read"},
		"list":  {"list_directory", "list_files", "ls"},
	}

	keywords, exists := opKeywords[operation]
	if !exists {
		return "llm"
	}

	for _, tool := range p.tools {
		toolName := strings.ToLower(tool.Function.Name)
		for _, keyword := range keywords {
			if strings.Contains(toolName, keyword) {
				return tool.Function.Name
			}
		}
	}

	// If no specific filesystem tool found, try to find any filesystem tool
	for _, tool := range p.tools {
		if strings.Contains(strings.ToLower(tool.Function.Name), "filesystem") {
			return tool.Function.Name
		}
	}

	return "llm"
}

// isFilesystemTool checks if a tool name is a filesystem tool of the given operation type
func (p *Planner) isFilesystemTool(toolName, operation string) bool {
	opKeywords := map[string][]string{
		"write": {"write_file", "create_file", "write", "save"},
		"read":  {"read_file", "get_file", "open", "read"},
		"list":  {"list_directory", "list_files", "ls"},
	}

	keywords, exists := opKeywords[operation]
	if !exists {
		return false
	}

	lowerToolName := strings.ToLower(toolName)
	for _, keyword := range keywords {
		if strings.Contains(lowerToolName, keyword) {
			return true
		}
	}
	return false
}

// isFileWriteDescription checks if a step description indicates a file write operation
func (p *Planner) isFileWriteDescription(description string) bool {
	lowerDesc := strings.ToLower(description)
	return containsAny(lowerDesc, []string{
		"写入文件", "保存到文件", "输出到文件", "导出到", "保存为", "创建文件",
		"write file", "save to file", "save as", "export to", "write to",
		"写入", "保存", "write", "save",
	})
}
