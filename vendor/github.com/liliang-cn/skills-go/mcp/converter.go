package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/liliang-cn/skills-go/skill"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
)

// ServerConfig defines how to connect to an MCP server
type ServerConfig struct {
	// Name is the skill name to generate (defaults to server name)
	Name string
	// Command is the stdio command to run (e.g., "python server.py")
	Command []string
	// URL is the HTTP endpoint for streamable HTTP transport
	URL string
	// Description is a custom description for the generated skill
	Description string
	// Include specifies what to include in the skill
	Include IncludeConfig
}

// IncludeConfig controls what MCP capabilities to convert
type IncludeConfig struct {
	Tools    bool
	Resources bool
	Prompts  bool
}

// DefaultInclude returns the default inclusion config
func DefaultInclude() IncludeConfig {
	return IncludeConfig{
		Tools:     true,
		Resources: true,
		Prompts:   true,
	}
}

// Converter converts MCP servers to Skills
type Converter struct {
	client    *mcpsdk.Client
	llmClient *openai.Client
	model     string
}

// ConverterOption configures a Converter
type ConverterOption func(*Converter)

// WithLLMClient sets an LLM client for enhanced skill generation
func WithLLMClient(client *openai.Client) ConverterOption {
	return func(c *Converter) {
		c.llmClient = client
	}
}

// WithLLMModel sets the model to use for LLM-based conversion
func WithLLMModel(model string) ConverterOption {
	return func(c *Converter) {
		c.model = model
	}
}

// WithLLM creates a new OpenAI client with the given API key and sets it
func WithLLM(apiKey string, baseURL ...string) ConverterOption {
	return func(c *Converter) {
		opts := []option.RequestOption{option.WithAPIKey(apiKey)}
		if len(baseURL) > 0 && baseURL[0] != "" {
			opts = append(opts, option.WithBaseURL(baseURL[0]))
		}
		client := openai.NewClient(opts...)
		c.llmClient = &client
		if c.model == "" {
			c.model = "gpt-4o"
		}
	}
}

// NewConverter creates a new MCP to Skill converter
func NewConverter(opts ...ConverterOption) *Converter {
	c := &Converter{
		client: mcpsdk.NewClient(&mcpsdk.Implementation{
			Name:    "skills-go-converter",
			Version: "1.0.0",
		}, nil),
		model: "gpt-4o",
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Convert converts an MCP server to a Skill and writes it to outputDir
func (c *Converter) Convert(ctx context.Context, cfg *ServerConfig, outputDir string) (*skill.Skill, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create transport and connect
	transport, err := c.createTransport(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}

	session, err := c.client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
	}
	defer session.Close()

	// Discover server capabilities
	info, err := c.getServerInfo(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to get server info: %w", err)
	}

	// Generate skill content
	content, err := c.generateSkillContent(ctx, session, cfg, info)
	if err != nil {
		return nil, fmt.Errorf("failed to generate skill content: %w", err)
	}

	// Determine skill name
	skillName := cfg.Name
	if skillName == "" {
		skillName = sanitizeName(info.Name)
	}
	if skillName == "" {
		skillName = "mcp-server"
	}

	// Generate description
	description := cfg.Description
	if description == "" {
		description = fmt.Sprintf("Skills converted from MCP server: %s", info.Name)
	}

	// Create SKILL.md
	skillPath := filepath.Join(outputDir, skillName)
	if err := os.MkdirAll(skillPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create skill directory: %w", err)
	}

	skillFile := filepath.Join(skillPath, "SKILL.md")
	skillMD := c.formatSkillMD(skillName, description, content)

	if err := os.WriteFile(skillFile, []byte(skillMD), 0644); err != nil {
		return nil, fmt.Errorf("failed to write SKILL.md: %w", err)
	}

	// Create references directory with MCP capability details
	if err := c.createReferences(ctx, session, skillPath, cfg); err != nil {
		return nil, fmt.Errorf("failed to create references: %w", err)
	}

	// Return the skill as a skill.Skill for immediate use
	s := &skill.Skill{
		Meta: skill.Meta{
			Name:        skillName,
			Description: description,
		},
		Name:    skillName,
		Path:    skillPath,
		Content: content,
		LoadedAt: info.Timestamp,
	}

	return s, nil
}

// ConvertWithLLM converts an MCP server to a Skill using LLM for enhanced content generation
func (c *Converter) ConvertWithLLM(ctx context.Context, cfg *ServerConfig, outputDir string) (*skill.Skill, error) {
	if c.llmClient == nil {
		// Fallback to regular conversion if no LLM client
		return c.Convert(ctx, cfg, outputDir)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create transport and connect
	transport, err := c.createTransport(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}

	session, err := c.client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
	}
	defer session.Close()

	// Discover server capabilities
	info, err := c.getServerInfo(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to get server info: %w", err)
	}

	// Collect raw data for LLM
	rawData, err := c.collectRawData(ctx, session, cfg, info)
	if err != nil {
		return nil, fmt.Errorf("failed to collect raw data: %w", err)
	}

	// Generate skill content using LLM
	content, enhancedDesc, err := c.generateSkillContentWithLLM(ctx, rawData, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to generate skill content with LLM: %w", err)
	}

	// Determine skill name
	skillName := cfg.Name
	if skillName == "" {
		skillName = sanitizeName(info.Name)
	}
	if skillName == "" {
		skillName = "mcp-server"
	}

	// Generate description (use LLM enhanced or fallback)
	description := cfg.Description
	if description == "" {
		description = enhancedDesc
	}
	if description == "" {
		description = fmt.Sprintf("Skills converted from MCP server: %s", info.Name)
	}

	// Create SKILL.md
	skillPath := filepath.Join(outputDir, skillName)
	if err := os.MkdirAll(skillPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create skill directory: %w", err)
	}

	skillFile := filepath.Join(skillPath, "SKILL.md")
	skillMD := c.formatSkillMD(skillName, description, content)

	if err := os.WriteFile(skillFile, []byte(skillMD), 0644); err != nil {
		return nil, fmt.Errorf("failed to write SKILL.md: %w", err)
	}

	// Create references directory with MCP capability details
	if err := c.createReferences(ctx, session, skillPath, cfg); err != nil {
		return nil, fmt.Errorf("failed to create references: %w", err)
	}

	// Return the skill as a skill.Skill for immediate use
	s := &skill.Skill{
		Meta: skill.Meta{
			Name:        skillName,
			Description: description,
		},
		Name:     skillName,
		Path:     skillPath,
		Content:  content,
		LoadedAt: info.Timestamp,
	}

	return s, nil
}

// mcpRawData holds raw MCP server data for LLM processing
type mcpRawData struct {
	Name        string                   `json:"name"`
	Version     string                   `json:"version"`
	Protocol    string                   `json:"protocol"`
	Tools       []toolRawData            `json:"tools,omitempty"`
	Resources   []resourceRawData        `json:"resources,omitempty"`
	Templates   []templateRawData        `json:"templates,omitempty"`
	Prompts     []promptRawData          `json:"prompts,omitempty"`
	UserContext string                   `json:"user_context,omitempty"`
}

type toolRawData struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema,omitempty"`
}

type resourceRawData struct {
	Name      string `json:"name"`
	URI       string `json:"uri"`
	MIMEType  string `json:"mime_type,omitempty"`
	Description string `json:"description,omitempty"`
}

type templateRawData struct {
	Name        string `json:"name"`
	URITemplate string `json:"uri_template"`
	Description string `json:"description,omitempty"`
}

type promptRawData struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Arguments   []promptArgRawData `json:"arguments,omitempty"`
}

type promptArgRawData struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
}

// collectRawData collects raw MCP server data for LLM processing
func (c *Converter) collectRawData(ctx context.Context, session *mcpsdk.ClientSession, cfg *ServerConfig, info *serverInfo) (*mcpRawData, error) {
	data := &mcpRawData{
		Name:     info.Name,
		Version:  info.Version,
		Protocol: info.Protocol,
	}

	if cfg.Include.Tools {
		tools, err := session.ListTools(ctx, nil)
		if err == nil && tools != nil {
			for _, t := range tools.Tools {
				toolData := toolRawData{
					Name:        t.Name,
					Description: t.Description,
				}
				if t.InputSchema != nil {
					if schemaBytes, ok := t.InputSchema.([]byte); ok {
						var schema map[string]any
						if err := json.Unmarshal(schemaBytes, &schema); err == nil {
							toolData.InputSchema = schema
						}
					}
				}
				data.Tools = append(data.Tools, toolData)
			}
		}
	}

	if cfg.Include.Resources {
		resources, err := session.ListResources(ctx, nil)
		if err == nil && resources != nil {
			for _, r := range resources.Resources {
				data.Resources = append(data.Resources, resourceRawData{
					Name:        r.Name,
					URI:         r.URI,
					MIMEType:    r.MIMEType,
					Description: r.Description,
				})
			}
		}

		templates, err := session.ListResourceTemplates(ctx, nil)
		if err == nil && templates != nil {
			for _, t := range templates.ResourceTemplates {
				data.Templates = append(data.Templates, templateRawData{
					Name:        t.Name,
					URITemplate: t.URITemplate,
					Description: t.Description,
				})
			}
		}
	}

	if cfg.Include.Prompts {
		prompts, err := session.ListPrompts(ctx, nil)
		if err == nil && prompts != nil {
			for _, p := range prompts.Prompts {
				promptData := promptRawData{
					Name:        p.Name,
					Description: p.Description,
				}
				for _, arg := range p.Arguments {
					promptData.Arguments = append(promptData.Arguments, promptArgRawData{
						Name:        arg.Name,
						Description: arg.Description,
						Required:    arg.Required,
					})
				}
				data.Prompts = append(data.Prompts, promptData)
			}
		}
	}

	data.UserContext = cfg.Description

	return data, nil
}

// generateSkillContentWithLLM uses LLM to generate enhanced skill content
func (c *Converter) generateSkillContentWithLLM(ctx context.Context, data *mcpRawData, cfg *ServerConfig) (content string, description string, err error) {
	// Convert data to JSON for the prompt
	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal data: %w", err)
	}

	systemPrompt := `You are an expert at creating AI Skills from MCP (Model Context Protocol) server definitions.
Your task is to convert MCP server capabilities into a well-structured, user-friendly Skill markdown file.

The Skill should:
1. Start with a clear, concise description of what the MCP server does
2. Organize tools/resources/prompts into logical groups with helpful headings
3. Provide clear descriptions for each capability
4. Include practical usage examples where helpful
5. Be written in a professional yet accessible tone
6. Use proper markdown formatting

Output format:
1. First line: A one-line description (for the skill metadata)
2. A blank line
3. The main skill content in markdown

DO NOT include code fences or any other wrapper around the output.`

	userPrompt := fmt.Sprintf("Convert this MCP server definition to a Skill:\n\n```json\n%s\n```\n\nGenerate a comprehensive Skill.md file. The first line should be a concise description, followed by a blank line, then the main content.", string(dataJSON))

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
		openai.UserMessage(userPrompt),
	}

	model := c.model
	if model == "" {
		model = "gpt-4o"
	}

	resp, err := c.llmClient.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    shared.ChatModel(model),
	})
	if err != nil {
		return "", "", fmt.Errorf("LLM call failed: %w", err)
	}

	output := resp.Choices[0].Message.Content

	// Split into description and content
	parts := strings.SplitN(strings.TrimSpace(output), "\n", 2)
	if len(parts) == 0 {
		return "", "", fmt.Errorf("empty LLM response")
	}

	description = strings.TrimSpace(parts[0])
	if len(parts) > 1 {
		content = strings.TrimSpace(parts[1])
	} else {
		content = description
		description = ""
	}

	return content, description, nil
}

// createTransport creates the appropriate transport for the server config
func (c *Converter) createTransport(cfg *ServerConfig) (mcpsdk.Transport, error) {
	if len(cfg.Command) > 0 {
		// Parse command for stdio transport
		var cmd string
		var args []string

		if len(cfg.Command) == 1 {
			parts := strings.Fields(cfg.Command[0])
			if len(parts) > 0 {
				cmd = parts[0]
				args = parts[1:]
			}
		} else {
			cmd = cfg.Command[0]
			args = cfg.Command[1:]
		}

		if cmd == "" {
			return nil, fmt.Errorf("invalid command: %v", cfg.Command)
		}

		return &mcpsdk.CommandTransport{
			Command: exec.Command(cmd, args...),
		}, nil
	}

	if cfg.URL != "" {
		return &mcpsdk.StreamableClientTransport{
			Endpoint: cfg.URL,
		}, nil
	}

	return nil, fmt.Errorf("either Command or URL must be specified")
}

// serverInfo holds information about the MCP server
type serverInfo struct {
	Name      string
	Version   string
	Protocol  string
	Timestamp time.Time
}

// getServerInfo retrieves server information from the session
func (c *Converter) getServerInfo(ctx context.Context, session *mcpsdk.ClientSession) (*serverInfo, error) {
	// The server info is available after initialization
	// For now, return basic info
	info := &serverInfo{
		Name:      "mcp-server",
		Version:   "1.0.0",
		Protocol:  "2025-03-26",
		Timestamp: time.Now(),
	}

	// Try to get more info from capabilities
	// Note: The actual implementation depends on the SDK version
	return info, nil
}

// generateSkillContent generates the main skill content from MCP capabilities
func (c *Converter) generateSkillContent(ctx context.Context, session *mcpsdk.ClientSession, cfg *ServerConfig, info *serverInfo) (string, error) {
	var sb strings.Builder

	sb.WriteString("# MCP Server Skill\n\n")
	sb.WriteString(fmt.Sprintf("This skill provides access to tools from the **%s** MCP server.\n\n", info.Name))

	if cfg.Include.Tools {
		tools, err := session.ListTools(ctx, nil)
		if err == nil && tools != nil && len(tools.Tools) > 0 {
			sb.WriteString("## Available Tools\n\n")
			for _, tool := range tools.Tools {
				sb.WriteString(fmt.Sprintf("### %s\n\n", tool.Name))
				if tool.Description != "" {
					sb.WriteString(fmt.Sprintf("%s\n\n", tool.Description))
				}
				// Add input schema info if available
				if tool.InputSchema != nil {
					sb.WriteString("**Input Schema:**\n")
					sb.WriteString("```json\n")
					// Try to serialize the schema
					if schemaBytes, ok := tool.InputSchema.([]byte); ok {
						sb.WriteString(string(schemaBytes))
					} else {
						sb.WriteString(fmt.Sprintf("%#v", tool.InputSchema))
					}
					sb.WriteString("\n```\n\n")
				}
			}
		}
	}

	if cfg.Include.Resources {
		resources, err := session.ListResources(ctx, nil)
		if err == nil && resources != nil && len(resources.Resources) > 0 {
			sb.WriteString("## Available Resources\n\n")
			for _, res := range resources.Resources {
				sb.WriteString(fmt.Sprintf("- **%s** (%s)", res.Name, res.URI))
				if res.Description != "" {
					sb.WriteString(fmt.Sprintf(": %s", res.Description))
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}

		// Also list resource templates
		templates, err := session.ListResourceTemplates(ctx, nil)
		if err == nil && templates != nil && len(templates.ResourceTemplates) > 0 {
			sb.WriteString("### Resource Templates\n\n")
			for _, tmpl := range templates.ResourceTemplates {
				sb.WriteString(fmt.Sprintf("- **%s** (`%s`)", tmpl.Name, tmpl.URITemplate))
				if tmpl.Description != "" {
					sb.WriteString(fmt.Sprintf(": %s", tmpl.Description))
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}
	}

	if cfg.Include.Prompts {
		prompts, err := session.ListPrompts(ctx, nil)
		if err == nil && prompts != nil && len(prompts.Prompts) > 0 {
			sb.WriteString("## Available Prompts\n\n")
			for _, prompt := range prompts.Prompts {
				sb.WriteString(fmt.Sprintf("### %s\n\n", prompt.Name))
				if prompt.Description != "" {
					sb.WriteString(fmt.Sprintf("%s\n\n", prompt.Description))
				}
				if len(prompt.Arguments) > 0 {
					sb.WriteString("**Arguments:**\n")
					for _, arg := range prompt.Arguments {
						sb.WriteString(fmt.Sprintf("- `%s`", arg.Name))
						if arg.Description != "" {
							sb.WriteString(fmt.Sprintf(": %s", arg.Description))
						}
						if arg.Required {
							sb.WriteString(" (required)")
						}
						sb.WriteString("\n")
					}
					sb.WriteString("\n")
				}
			}
		}
	}

	sb.WriteString("## Usage\n\n")
	sb.WriteString("This skill is automatically invoked when relevant to the conversation. ")
	sb.WriteString("The AI agent will call the appropriate tools based on your request.\n")

	return sb.String(), nil
}

// formatSkillMD formats the complete SKILL.md file
func (c *Converter) formatSkillMD(name, description, content string) string {
	var sb strings.Builder

	// YAML frontmatter
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("name: %s\n", name))
	sb.WriteString(fmt.Sprintf("description: %s\n", description))
	sb.WriteString("user-invocable: true\n")
	sb.WriteString("---\n\n")

	// Content
	sb.WriteString(content)

	return sb.String()
}

// createReferences creates reference files with detailed MCP capability information
func (c *Converter) createReferences(ctx context.Context, session *mcpsdk.ClientSession, skillPath string, cfg *ServerConfig) error {
	refDir := filepath.Join(skillPath, "references")
	if err := os.MkdirAll(refDir, 0755); err != nil {
		return err
	}

	if cfg.Include.Tools {
		tools, err := session.ListTools(ctx, nil)
		if err == nil && tools != nil && len(tools.Tools) > 0 {
			// Create tools reference
			var toolsDoc strings.Builder
			toolsDoc.WriteString("# MCP Tools Reference\n\n")
			for _, tool := range tools.Tools {
				toolsDoc.WriteString(fmt.Sprintf("## %s\n\n", tool.Name))
				if tool.Description != "" {
					toolsDoc.WriteString(fmt.Sprintf("**Description:** %s\n\n", tool.Description))
				}
				if tool.InputSchema != nil {
					toolsDoc.WriteString("**Input Schema:**\n```json\n")
					if schemaBytes, ok := tool.InputSchema.([]byte); ok {
						toolsDoc.WriteString(string(schemaBytes))
					} else {
						toolsDoc.WriteString(fmt.Sprintf("%#v", tool.InputSchema))
					}
					toolsDoc.WriteString("\n```\n\n")
				}
			}
			if err := os.WriteFile(filepath.Join(refDir, "tools.md"), []byte(toolsDoc.String()), 0644); err != nil {
				return err
			}
		}
	}

	if cfg.Include.Resources {
		resources, err := session.ListResources(ctx, nil)
		if err == nil && resources != nil && len(resources.Resources) > 0 {
			// Create resources reference
			var resourcesDoc strings.Builder
			resourcesDoc.WriteString("# MCP Resources Reference\n\n")
			for _, res := range resources.Resources {
				resourcesDoc.WriteString(fmt.Sprintf("## %s\n\n", res.Name))
				resourcesDoc.WriteString(fmt.Sprintf("**URI:** `%s`\n\n", res.URI))
				if res.Description != "" {
					resourcesDoc.WriteString(fmt.Sprintf("**Description:** %s\n\n", res.Description))
				}
				if res.MIMEType != "" {
					resourcesDoc.WriteString(fmt.Sprintf("**MIME Type:** %s\n\n", res.MIMEType))
				}
			}
			if err := os.WriteFile(filepath.Join(refDir, "resources.md"), []byte(resourcesDoc.String()), 0644); err != nil {
				return err
			}
		}
	}

	if cfg.Include.Prompts {
		prompts, err := session.ListPrompts(ctx, nil)
		if err == nil && prompts != nil && len(prompts.Prompts) > 0 {
			// Create prompts reference
			var promptsDoc strings.Builder
			promptsDoc.WriteString("# MCP Prompts Reference\n\n")
			for _, prompt := range prompts.Prompts {
				promptsDoc.WriteString(fmt.Sprintf("## %s\n\n", prompt.Name))
				if prompt.Description != "" {
					promptsDoc.WriteString(fmt.Sprintf("**Description:** %s\n\n", prompt.Description))
				}
				if len(prompt.Arguments) > 0 {
					promptsDoc.WriteString("**Arguments:**\n\n")
					for _, arg := range prompt.Arguments {
						promptsDoc.WriteString(fmt.Sprintf("- `%s`", arg.Name))
						if arg.Description != "" {
							promptsDoc.WriteString(fmt.Sprintf(": %s", arg.Description))
						}
						if arg.Required {
							promptsDoc.WriteString(" (required)")
						}
						promptsDoc.WriteString("\n")
					}
					promptsDoc.WriteString("\n")
				}
			}
			if err := os.WriteFile(filepath.Join(refDir, "prompts.md"), []byte(promptsDoc.String()), 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

// sanitizeName converts a string to a valid skill name
func sanitizeName(name string) string {
	// Convert to lowercase, replace spaces with hyphens
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	// Remove any character that's not alphanumeric or hyphen
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	resultStr := result.String()
	// Remove leading/trailing hyphens
	resultStr = strings.Trim(resultStr, "-")
	if resultStr == "" {
		return "mcp-skill"
	}
	return resultStr
}

// ConvertAll converts multiple MCP servers to skills
func (c *Converter) ConvertAll(ctx context.Context, configs []*ServerConfig, outputDir string) ([]*skill.Skill, error) {
	var skills []*skill.Skill
	var errs []error

	for _, cfg := range configs {
		s, err := c.Convert(ctx, cfg, outputDir)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to convert server %s: %w", cfg.Name, err))
			continue
		}
		skills = append(skills, s)
	}

	if len(errs) > 0 {
		return skills, fmt.Errorf("encountered %d errors: %v", len(errs), errs)
	}

	return skills, nil
}

// Discover connects to an MCP server and returns its capabilities without converting
func (c *Converter) Discover(ctx context.Context, cfg *ServerConfig) (*ServerCapabilities, error) {
	transport, err := c.createTransport(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}

	session, err := c.client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer session.Close()

	caps := &ServerCapabilities{}

	if cfg.Include.Tools {
		tools, err := session.ListTools(ctx, nil)
		if err == nil && tools != nil {
			caps.Tools = tools.Tools
		}
	}

	if cfg.Include.Resources {
		resources, err := session.ListResources(ctx, nil)
		if err == nil && resources != nil {
			caps.Resources = resources.Resources
		}
		templates, err := session.ListResourceTemplates(ctx, nil)
		if err == nil && templates != nil {
			caps.ResourceTemplates = templates.ResourceTemplates
		}
	}

	if cfg.Include.Prompts {
		prompts, err := session.ListPrompts(ctx, nil)
		if err == nil && prompts != nil {
			caps.Prompts = prompts.Prompts
		}
	}

	return caps, nil
}

// ServerCapabilities holds discovered MCP server capabilities
type ServerCapabilities struct {
	Tools            []*mcpsdk.Tool
	Resources        []*mcpsdk.Resource
	ResourceTemplates []*mcpsdk.ResourceTemplate
	Prompts          []*mcpsdk.Prompt
}

// ListTools returns the tool names
func (c *ServerCapabilities) ListTools() []string {
	var names []string
	for _, t := range c.Tools {
		names = append(names, t.Name)
	}
	slices.Sort(names)
	return names
}

// ListResources returns the resource URIs
func (c *ServerCapabilities) ListResources() []string {
	var uris []string
	for _, r := range c.Resources {
		uris = append(uris, r.URI)
	}
	slices.Sort(uris)
	return uris
}

// ListPrompts returns the prompt names
func (c *ServerCapabilities) ListPrompts() []string {
	var names []string
	for _, p := range c.Prompts {
		names = append(names, p.Name)
	}
	slices.Sort(names)
	return names
}

// QuickConvertWithLLM converts a command-based MCP server to a skill using LLM in one call
func QuickConvertWithLLM(ctx context.Context, llmClient *openai.Client, command string, outputDir string, args ...string) (*skill.Skill, error) {
	cfg := &ServerConfig{
		Command: append([]string{command}, args...),
		Include: DefaultInclude(),
	}
	c := NewConverter(WithLLMClient(llmClient))
	return c.ConvertWithLLM(ctx, cfg, outputDir)
}

// QuickConvertHTTPWithLLM converts an HTTP-based MCP server to a skill using LLM in one call
func QuickConvertHTTPWithLLM(ctx context.Context, llmClient *openai.Client, url string, outputDir string) (*skill.Skill, error) {
	cfg := &ServerConfig{
		URL:     url,
		Include: DefaultInclude(),
	}
	c := NewConverter(WithLLMClient(llmClient))
	return c.ConvertWithLLM(ctx, cfg, outputDir)
}

// QuickConvertWithLLMAPIKey converts using an API key string (creates client internally)
func QuickConvertWithLLMAPIKey(ctx context.Context, apiKey string, command string, outputDir string, args ...string) (*skill.Skill, error) {
	cfg := &ServerConfig{
		Command: append([]string{command}, args...),
		Include: DefaultInclude(),
	}
	c := NewConverter(WithLLM(apiKey))
	return c.ConvertWithLLM(ctx, cfg, outputDir)
}

// QuickConvertHTTPWithLLMAPIKey converts HTTP server using API key string
func QuickConvertHTTPWithLLMAPIKey(ctx context.Context, apiKey string, url string, outputDir string) (*skill.Skill, error) {
	cfg := &ServerConfig{
		URL:     url,
		Include: DefaultInclude(),
	}
	c := NewConverter(WithLLM(apiKey))
	return c.ConvertWithLLM(ctx, cfg, outputDir)
}
